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

// Magick uses local execution with ImageMagick to transform images.
type Magick struct {
	magickBinary []string
}

func NewMagick() (*Magick, error) {
	eh := &Magick{}
	commands := [][]string{{"magick", "convert", "-version"}, {"convert", "-version"}}

	for _, command := range commands {
		// #nosec G204: no user input
		_, err := exec.Command(command[0], command[1:]...).Output()
		if err != nil {
			log.Debug().Strs("command", command).Msg("binary not found")
			continue
		}

		log.Debug().Strs("command", command).Msg("binary found")
		eh.magickBinary = command[:len(command)-1]
		break
	}

	if len(eh.magickBinary) == 0 {
		return nil, errors.New("magick binary not available")
	}

	return eh, nil
}

func (m *Magick) Scale(ctx context.Context, imageURL string, power float32) ([]byte, error) {
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

	// #nosec G204: only a float as user input
	cmd := exec.Command(command[0], command[1:]...)
	err = cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("error executing magick command: %w", err)
	}

	log.Debug().Msg("magick command finished")

	return file.GetTempFile(outFile)
}

// createCommand builds an ImageMagick command with args for the wanted result.
func createCommand(baseCommands []string, power float32, inFile, outFile string) []string {
	size := MaxPower - (power / PowerFactor)
	dimensions := fmt.Sprintf("%d%%x%d%%", int(size), int(size))
	return append(baseCommands, inFile, "-liquid-rescale", dimensions, outFile)
}
