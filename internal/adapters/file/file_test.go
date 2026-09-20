package file

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDownloadFile(t *testing.T) {
	tests := []struct {
		name       string
		inputBytes []byte
		status     int
		wantErr    bool
	}{
		{
			name:       "success",
			inputBytes: []byte("test\n"),
			status:     http.StatusOK,
			wantErr:    false,
		},
		{
			name:       "not found",
			inputBytes: []byte("not found"),
			status:     http.StatusNotFound,
			wantErr:    true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tc.status)
				_, err := w.Write(tc.inputBytes)
				assert.NoError(t, err)
			}))
			defer srv.Close()

			res, err := DownloadFile(t.Context(), srv.URL)
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.inputBytes, res)
			}
		})
	}
}

func TestSaveTempFile(t *testing.T) {
	tests := []struct {
		name      string
		content   []byte
		extension string
		wantSize  int64
		wantErr   bool
	}{
		{
			name:      "success",
			content:   []byte("test\n"),
			extension: "txt",
			wantSize:  5,
			wantErr:   false,
		},
		{
			name:      "empty file",
			content:   []byte(""),
			extension: "dat",
			wantSize:  0,
			wantErr:   false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			path, err := SaveTempFile(tc.content, tc.extension)
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				defer RemoveTempFile(path)

				stat, err := os.Stat(path)
				require.NoError(t, err)
				assert.Equal(t, tc.wantSize, stat.Size())
			}
		})
	}
}

func TestGetTempFile(t *testing.T) {
	tests := []struct {
		name    string
		content []byte
		ext     string
		want    []byte
	}{
		{
			name:    "success",
			content: []byte("test\n"),
			ext:     "txt",
			want:    []byte("test\n"),
		},
		{
			name:    "empty data",
			content: []byte(""),
			ext:     "dat",
			want:    []byte{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			path, err := SaveTempFile(tc.content, tc.ext)
			require.NoError(t, err)
			defer RemoveTempFile(path)
			stat, err := os.Stat(path)
			require.NoError(t, err)
			assert.Equal(t, int64(len(tc.want)), stat.Size())

			file, err := GetTempFile(path)
			require.NoError(t, err)
			assert.Equal(t, tc.want, file)
		})
	}
}

func TestDownloadFileInvalidURL(t *testing.T) {
	_, err := DownloadFile(t.Context(), "://bad")
	require.Error(t, err)
	assert.ErrorContains(t, err, "error creating request")
}

func TestDownloadFileRequestError(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err := DownloadFile(ctx, "http://127.0.0.1:1")
	require.Error(t, err)
	assert.ErrorContains(t, err, "error executing request")
}

func TestGetTempFileMissing(t *testing.T) {
	_, err := GetTempFile(filepath.Join(t.TempDir(), "missing.bin"))
	require.Error(t, err)
}

func TestRemoveTempFileMissing(t *testing.T) {
	RemoveTempFile(filepath.Join(t.TempDir(), "missing.bin"))
}
