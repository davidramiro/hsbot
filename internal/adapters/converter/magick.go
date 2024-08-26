package converter

import (
	"context"
	"errors"
	"fmt"
	"github.com/rs/zerolog/log"
	"hsbot/internal/adapters/file"
	"os/exec"
	"path/filepath"
	"strings"
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
	f, err := file.DownloadFile(ctx, imageURL)
	if err != nil {
		return nil, err
	}

	extension := filepath.Ext(imageURL)
	path, err := file.SaveTempFile(f, extension)
	if err != nil {
		return nil, err
	}

	size := MaxPower - (power / PowerFactor)
	outFile := fmt.Sprintf("%sliq%s", strings.TrimSuffix(path, extension), extension)
	dimensions := fmt.Sprintf("%d%%x%d%%", int(size), int(size))

	defer file.RemoveTempFile(path)
	defer file.RemoveTempFile(outFile)

	args := append(m.magickBinary, path, "-liquid-rescale", dimensions, outFile)

	log.Debug().Strs("args", args).
		Str("dimensions", dimensions).
		Str("outFile", outFile).
		Str("path", path).
		Msg("scaling image")

	cmd := exec.Command(args[0], args[1:]...)
	out, err := cmd.Output()
	if err != nil {
		log.Error().Bytes("magickStderr", out).Msg("magick commands failed")
		return nil, err
	}

	log.Debug().Msg("magick commands finished")

	return file.GetTempFile(outFile)
}
