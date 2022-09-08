package containers

import (
	"github.com/rs/zerolog/log"

	"github.com/ntk148v/koker/pkg/images"
)

func InitContainer(img string) error {
	imgSHA, err := images.DownloadImage(img)
	if err != nil {
		return err
	}
	log.Info().Msgf("Download image %s successfully", imgSHA)
	return nil
}
