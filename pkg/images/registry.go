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

var registryPath = filepath.Join(constants.KokerImagesPath, "registry.json")

var lock = &sync.Mutex{}

var imgReg registry

// LoadRegistry creats image registry singleton instance from file
func LoadRegistry() error {
	if imgReg == nil {
		lock.Lock()
		defer lock.Unlock()
		if _, err := os.Stat(registryPath); os.IsNotExist(err) {
			ioutil.WriteFile(registryPath, []byte("{}"), 0644)
			imgReg = registry{}
			return nil
		}

		log.Debug().Msg("Load image registry")
		data, err := ioutil.ReadFile(registryPath)
		if err != nil {
			return errors.Wrap(err, "unable to load image registry")
		}

		if err := json.Unmarshal(data, &imgReg); err != nil {
			return errors.Wrap(err, "unable to marshal image registry")
		}
	} else {
		log.Debug().Msg("Image registry already loaded")
	}
	return nil
}

// SaveRegistry writes image registry to file
func SaveRegistry() error {
	b, err := json.Marshal(&imgReg)
	if err != nil {
		return errors.Wrap(err, "unable to marshal image regsitry")
	}
	if err = ioutil.WriteFile(registryPath, b, 0644); err != nil {
		return errors.Wrap(err, "unable to save registry to file")
	}
	return nil
}

func SetImage(k string, v Image) {
	imgReg.set(k, v)
}

func GetImage(k string) (Image, bool) {
	return imgReg.get(k)
}

func DelImage(k string) {
	imgReg.del(k)
}

type registry map[string]Image

func (r registry) set(k string, v Image) {
	r[k] = v
}

func (r registry) del(k string) {
	delete(r, k)
}

func (r registry) get(k string) (Image, bool) {
	v, ok := r[k]
	return v, ok
}
