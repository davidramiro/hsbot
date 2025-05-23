package file

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/stretchr/testify/assert"
)

func TestDownloadFile(t *testing.T) {
	want := []byte("test\n")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write(want)
		assert.NoError(t, err)
	}))
	defer srv.Close()

	res, err := DownloadFile(t.Context(), srv.URL)
	require.NoError(t, err)

	assert.Equal(t, want, res)
}

func TestSaveTemp(t *testing.T) {
	want := []byte("test\n")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write(want)
		assert.NoError(t, err)
	}))
	defer srv.Close()

	res, err := DownloadFile(t.Context(), srv.URL)
	require.NoError(t, err)

	path, err := SaveTempFile(res, "txt")
	require.NoError(t, err)

	defer RemoveTempFile(path)

	stat, err := os.Stat(path)
	require.NoError(t, err)

	assert.Equal(t, int64(5), stat.Size())
}

func TestGetTemp(t *testing.T) {
	want := []byte("test\n")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write(want)
		assert.NoError(t, err)
	}))
	defer srv.Close()

	res, err := DownloadFile(t.Context(), srv.URL)
	require.NoError(t, err)

	path, err := SaveTempFile(res, "txt")
	require.NoError(t, err)
	defer RemoveTempFile(path)

	stat, err := os.Stat(path)

	require.NoError(t, err)
	assert.Equal(t, int64(5), stat.Size())

	file, err := GetTempFile(path)
	require.NoError(t, err)
	assert.Equal(t, want, file)
}
