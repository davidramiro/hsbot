package converter

import (
	"context"
	"errors"
	"fmt"
	"hsbot/internal/adapters/file"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog/log"
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

	outFile := fmt.Sprintf("%sliq%s", strings.TrimSuffix(path, extension), extension)
	command := createCommand(m.magickBinary, power, path, outFile)

	defer file.RemoveTempFile(path)
	defer file.RemoveTempFile(outFile)

	log.Debug().
		Strs("command", command).
		Str("outFile", outFile).
		Str("path", path).
		Msg("scaling image")

	cmd := exec.Command(command[0], command[1:]...)
	out, err := cmd.Output()
	if err != nil {
		log.Error().Bytes("magickStderr", out).Msg("magick commands failed")
		return nil, err
	}

	log.Debug().Msg("magick commands finished")

	return file.GetTempFile(outFile)
}

func createCommand(baseCommands []string, power float32, inFile, outFile string) []string {
	size := MaxPower - (power / PowerFactor)
	dimensions := fmt.Sprintf("%d%%x%d%%", int(size), int(size))
	return append(baseCommands, inFile, "-liquid-rescale", dimensions, outFile)
}
