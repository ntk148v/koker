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
	log.Info().Str("repository", repositoryPath).Msg("Load image repository from file")
	if imgRepo == nil {
		lock.Lock()
		defer lock.Unlock()
		if _, err := os.Stat(repositoryPath); os.IsNotExist(err) {
			os.WriteFile(repositoryPath, []byte("{}"), 0644)
			imgRepo = make(repository)
			return nil
		}

		log.Debug().Msg("Load image repository")
		data, err := ioutil.ReadFile(repositoryPath)
		if err != nil {
			return errors.Wrap(err, "unable to load image repository")
		}

		if err := json.Unmarshal(data, &imgRepo); err != nil {
			return errors.Wrap(err, "unable to marshal image repository")
		}
	} else {
		log.Debug().Msg("Image repository already loaded")
	}
	return nil
}

// SaveRepository writes image repository to file
func SaveRepository() error {
	log.Info().Str("repository", repositoryPath).Msg("Save image repository to file")
	lock.Lock()
	defer lock.Unlock()
	if imgRepo == nil {
		imgRepo = make(repository)
	}
	b, err := json.Marshal(imgRepo)
	if err != nil {
		return errors.Wrap(err, "unable to marshal image regsitry")
	}
	if err = os.WriteFile(repositoryPath, b, 0644); err != nil {
		return errors.Wrap(err, "unable to save repository to file")
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
	// TODO(kiennt26): Check there is any container running from image
	log.Info().Msg("Remove image")
	img, exist := imgRepo.get(k)
	if !exist {
		log.Warn().Msg("Image doesn't exist, or maybe you're using image's id which is not supported yet")
		return
	}
	// Delete directories
	os.RemoveAll(filepath.Join(constants.KokerImagesPath, img.Manifest.Config.Digest.Hex))
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
