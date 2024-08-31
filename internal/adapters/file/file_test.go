package file

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/stretchr/testify/assert"
)

func TestDownloadFile(t *testing.T) {
	res, err := DownloadFile(context.Background(), "https://kore.cc/test.txt")
	require.NoError(t, err)

	assert.Equal(t, []byte("test\n"), res)
}

func TestSaveTemp(t *testing.T) {
	res, err := DownloadFile(context.Background(), "https://kore.cc/test.txt")
	require.NoError(t, err)

	path, err := SaveTempFile(res, "txt")
	require.NoError(t, err)

	defer RemoveTempFile(path)

	stat, err := os.Stat(path)
	require.NoError(t, err)

	assert.Equal(t, int64(5), stat.Size())
}

func TestGetTemp(t *testing.T) {
	res, err := DownloadFile(context.Background(), "https://kore.cc/test.txt")
	require.NoError(t, err)

	path, err := SaveTempFile(res, "txt")
	require.NoError(t, err)
	defer RemoveTempFile(path)

	stat, err := os.Stat(path)

	require.NoError(t, err)
	assert.Equal(t, int64(5), stat.Size())

	file, err := GetTempFile(path)
	require.NoError(t, err)
	assert.Equal(t, []byte("test\n"), file)
}
