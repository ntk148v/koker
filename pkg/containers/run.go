package containers

import (
	"github.com/ntk148v/koker/pkg/images"
	"github.com/rs/zerolog/log"
)

func InitContainer(img string) error {
	imgSHA, err := images.DownloadImage(img)
	if err != nil {
		return err
	}
	log.Info().Msgf("Download image %s successfully", imgSHA)
	return nil
}
