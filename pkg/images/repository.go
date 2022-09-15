package images

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"

	"github.com/ntk148v/koker/pkg/constants"
)

var (
	repositoryPath = filepath.Join(constants.KokerImagesPath, "repositories.json")
	lock           = &sync.Mutex{}
	imgRepo        repository
)

func ListAllImages() ([]map[string]string, error) {
	all := make([]map[string]string, 0)
	for _, v := range imgRepo {
		all = append(all, map[string]string{
			"repository": v.Repository,
			"tag":        v.Tag,
			"id":         v.ID,
		})
	}

	return all, nil
}

// LoadRepository creates image repository instance from file
func LoadRepository() error {
	log.Info().Str("registry", repositoryPath).Msg("Load image registry from file")
	if imgRepo == nil {
		lock.Lock()
		defer lock.Unlock()
		if _, err := os.Stat(repositoryPath); os.IsNotExist(err) {
			ioutil.WriteFile(repositoryPath, []byte("{}"), 0644)
			imgRepo = make(repository)
			return nil
		}

		log.Debug().Msg("Load image registry")
		data, err := ioutil.ReadFile(repositoryPath)
		if err != nil {
			return errors.Wrap(err, "unable to load image registry")
		}

		if err := json.Unmarshal(data, &imgRepo); err != nil {
			return errors.Wrap(err, "unable to marshal image registry")
		}
	} else {
		log.Debug().Msg("Image registry already loaded")
	}
	return nil
}

// SaveRepository writes image registry to file
func SaveRepository() error {
	log.Info().Str("repository", repositoryPath).Msg("Save image registry to file")
	b, err := json.Marshal(imgRepo)
	if err != nil {
		return errors.Wrap(err, "unable to marshal image regsitry")
	}
	if err = ioutil.WriteFile(repositoryPath, b, 0644); err != nil {
		return errors.Wrap(err, "unable to save registry to file")
	}
	return nil
}

func SetImage(k string, v Metadata) {
	imgRepo.set(k, v)
}

func GetImage(k string) (Metadata, bool) {
	return imgRepo.get(k)
}

func DelImage(k string) {
	imgRepo.del(k)
}

type repository map[string]Metadata

func (r repository) set(k string, v Metadata) {
	r[k] = v
}

func (r repository) del(k string) {
	delete(r, k)
}

func (r repository) get(k string) (Metadata, bool) {
	v, ok := r[k]
	return v, ok
}
