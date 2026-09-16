package images

import (
	"fmt"
	"sync"
	"testing"
)

func TestRepositoryConcurrency(t *testing.T) {
	imgRepo = make(repository)

	var wg sync.WaitGroup
	// Run concurrent writers and readers
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			key := fmt.Sprintf("test-image-%d", idx)
			meta := Metadata{
				ID:         fmt.Sprintf("id-%d", idx),
				Repository: fmt.Sprintf("repo-%d", idx),
				Tag:        "latest",
			}
			SetImage(key, meta)
			if _, ok := GetImage(key); !ok {
				t.Errorf("expected to find image %s", key)
			}
			_, _ = ListAllImages()
		}(i)
	}

	wg.Wait()

	list, err := ListAllImages()
	if err != nil {
		t.Fatalf("ListAllImages failed: %v", err)
	}
	if len(list) != 20 {
		t.Errorf("expected 20 images, got %d", len(list))
	}
}
