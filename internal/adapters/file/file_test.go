package file

import (
	"context"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestDownloadFile(t *testing.T) {
	res, err := DownloadFile(context.Background(), "https://kore.cc/test.txt")
	assert.NoError(t, err)

	assert.Equal(t, []byte("test\n"), res)
}

func TestSaveTemp(t *testing.T) {
	res, err := DownloadFile(context.Background(), "https://kore.cc/test.txt")
	assert.NoError(t, err)

	path, err := SaveTempFile(res, "txt")
	assert.NoError(t, err)

	defer RemoveTempFile(path)

	stat, err := os.Stat(path)
	assert.NoError(t, err)

	assert.Equal(t, int64(5), stat.Size())
}

func TestGetTemp(t *testing.T) {
	res, err := DownloadFile(context.Background(), "https://kore.cc/test.txt")
	assert.NoError(t, err)

	path, err := SaveTempFile(res, "txt")
	assert.NoError(t, err)
	defer RemoveTempFile(path)

	stat, err := os.Stat(path)

	assert.NoError(t, err)
	assert.Equal(t, int64(5), stat.Size())

	file, err := GetTempFile(path)
	assert.NoError(t, err)
	assert.Equal(t, []byte("test\n"), file)
}
