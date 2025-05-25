package file

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gofrs/uuid/v5"
	"github.com/rs/zerolog/log"
)

// DownloadFile returns the byte content of a file on a provided URL.
func DownloadFile(ctx context.Context, path string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, path, nil)
	if err != nil {
		err = fmt.Errorf("error creating request %w", err)
		log.Error().Err(err).Str("path", path).Send()
		return nil, err
	}

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		err = fmt.Errorf("error executing request %w", err)
		log.Error().Err(err).Str("path", path).Send()
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		err = fmt.Errorf("unexpected status code on download: %d", res.StatusCode)
		log.Error().Err(err).Str("path", path).Send()
		return nil, err
	}

	buf, err := io.ReadAll(res.Body)
	if err != nil {
		err = fmt.Errorf("error reading response %w", err)
		log.Error().Err(err).Str("path", path).Send()
		return nil, err
	}

	return buf, nil
}

// SaveTempFile saves bytes to a temp location and returns the path.
func SaveTempFile(data []byte, extension string) (string, error) {
	id, err := uuid.NewV4()
	if err != nil {
		return "", err
	}

	log.Debug().Int("bytes", len(data)).Str("extension", extension).Msg("creating temp file")

	path := filepath.Join(os.TempDir(), fmt.Sprintf("%s%s", id.String(), extension))

	f, err := os.Create(path)
	if err != nil {
		err = fmt.Errorf("error creating temp file %w", err)
		log.Error().Err(err).Send()
		return "", err
	}

	defer f.Close()

	if _, err := f.Write(data); err != nil {
		err = fmt.Errorf("error writing temp file %w", err)
		log.Error().Err(err).Send()
		return "", err
	}

	log.Debug().Str("path", f.Name()).Msg("created file")

	return f.Name(), nil
}

// GetTempFile retrieves a temporarily stored file by its path, as returned from SaveTempFile().
func GetTempFile(path string) ([]byte, error) {
	buf, err := os.ReadFile(path)
	if err != nil {
		err = fmt.Errorf("error reading temp file %w", err)
		log.Error().Err(err).Send()
		return nil, err
	}

	return buf, nil
}

// RemoveTempFile removes a specified temporary file at the given path and logs success or failure.
func RemoveTempFile(path string) {
	err := os.Remove(path)
	if err != nil {
		log.Warn().Str("path", path).Err(err).Msg("could not clean up temp file")
		return
	}
	log.Debug().Str("path", path).Msg("cleaned up temp file")
}
