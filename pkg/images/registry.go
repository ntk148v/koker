package images

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/ntk148v/koker/pkg/utils"
)

var registryPath = filepath.Join(utils.KokerImagesPath, "registry.json")

type registry map[string]string

func loadRegistry() (registry, error) {
	var r registry
	if _, err := os.Stat(registryPath); os.IsNotExist(err) {
		ioutil.WriteFile(registryPath, []byte("{}"), 0644)
		return r, nil
	}

	data, err := ioutil.ReadFile(registryPath)
	if err != nil {
		return r, fmt.Errorf("Unable to read image registry due to %v", err)
	}

	if err := json.Unmarshal(data, &r); err != nil {
		return r, fmt.Errorf("Unable to load image registry due to %v", err)
	}
	return r, nil
}

func saveRegistry(r registry) error {
	b, err := json.Marshal(r)
	if err != nil {
		return fmt.Errorf("Unable to marshal image registry due to %v", err)
	}
	if err = ioutil.WriteFile(registryPath, b, 0644); err != nil {
		return fmt.Errorf("Unable to save image registry due to %v", err)
	}
	return nil
}

func (r registry) set(k, v string) {
	r[k] = v
}

func (r registry) delete(k string) {
	delete(r, k)
}

func (r registry) get(k string) (string, bool) {
	v, ok := r[k]
	return v, ok
}
