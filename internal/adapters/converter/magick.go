package converter

import (
	"context"
	"errors"
	"fmt"
	"github.com/rs/zerolog/log"
	"hsbot/internal/adapters/file"
	"os/exec"
	"path/filepath"
)

const MaxPower = 100
const PowerFactor = 1.3

type MagickConverter struct {
	magickBinary []string
}

func NewMagickConverter() (*MagickConverter, error) {
	eh := &MagickConverter{}
	commands := [][]string{{"magick", "convert", "-version"}, {"convert", "-version"}}

	for _, command := range commands {
		_, err := exec.Command(command[0], command[1:]...).Output()
		if err != nil {
			log.Debug().Strs("commands", command).Msg("binary not found")
			continue
		}

		log.Debug().Strs("commands", command).Msg("binary found")
		eh.magickBinary = command[:len(command)-1]
		break
	}

	if len(eh.magickBinary) == 0 {
		return nil, errors.New("magick binary not available")
	}

	return eh, nil
}

func (m *MagickConverter) Scale(ctx context.Context, imageURL string, power float32) ([]byte, error) {
	f, err := file.Download(ctx, imageURL)
	if err != nil {
		return nil, err
	}

	path, err := file.SaveTemp(f, filepath.Ext(imageURL))
	if err != nil {
		return nil, err
	}

	size := MaxPower - (power / PowerFactor)
	dimensions := fmt.Sprintf("%d%%x%d%%", int(size), int(size))

	args := append(m.magickBinary, path, "-liquid-rescale", dimensions, path)

	cmd := exec.Command(args[0], args[1:]...)
	out, err := cmd.Output()
	if err != nil {
		log.Error().Bytes("magickStderr", out).Msg("magick commands failed")
		return nil, err
	}

	log.Debug().Msg("magick commands finished")

	return file.GetTemp(path)
}
