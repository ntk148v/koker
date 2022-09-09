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
	log.Info().Str("image", img).
		Str("imgSHA", imgSHA).
		Msg("Download image successfully")
	return nil
}
