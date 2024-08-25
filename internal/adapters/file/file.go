package file

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gofrs/uuid/v5"
	"github.com/rs/zerolog/log"
)

func Download(ctx context.Context, path string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, path, nil)
	if err != nil {
		log.Error().Err(err).Str("path", path).Msg("")
		return nil, errors.New("could build get request")
	}

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		log.Error().Err(err).Str("path", path).Msg("")
		return nil, errors.New("could not download file")
	}
	defer res.Body.Close()

	buf, err := io.ReadAll(res.Body)

	if err != nil {
		log.Error().Err(err).Str("path", path).Msg("")
		return nil, errors.New("could not write file content")
	}

	return buf, nil
}

func SaveTemp(data []byte, extension string) (string, error) {
	id, err := uuid.NewV4()
	if err != nil {
		return "", err
	}

	log.Debug().Int("bytes", len(data)).Str("extension", extension).Msg("creating temp file")

	path := filepath.Join(os.TempDir(), fmt.Sprintf("%s%s", id.String(), extension))

	f, err := os.Create(path)
	if err != nil {
		log.Error().Err(err).Msg("could not create temp file")
		return "", err
	}

	defer f.Close()

	if _, err := f.Write(data); err != nil {
		return "", err
	}

	log.Debug().Str("path", f.Name()).Msg("created file")

	return f.Name(), nil
}

func GetTemp(path string) ([]byte, error) {
	buf, err := os.ReadFile(path)
	if err != nil {
		log.Error().Err(err).Msg("")
		return nil, err
	}

	defer removeTempFile(path)

	return buf, nil
}

func removeTempFile(path string) {
	err := os.Remove(path)
	if err != nil {
		log.Warn().Str("path", path).Err(err).Msg("could not clean up temp file")
	}
	log.Debug().Str("path", path).Msg("cleaned up temp file")
}
