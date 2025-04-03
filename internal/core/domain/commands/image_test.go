package commands

import (
	"context"
	"errors"
	"hsbot/internal/core/domain"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type MockImageGenerator struct {
	response string
	err      error
	Message  string
}

func (m *MockImageGenerator) GenerateFromPrompt(_ context.Context, prompt string) (string, error) {
	m.Message = prompt
	return m.response, m.err
}

func (m *MockImageGenerator) EditFromPrompt(_ context.Context, prompt domain.Prompt) (string, error) {
	return "", nil
}

type MockImageSender struct {
	err     error
	Message string
}

func (m *MockImageSender) SendImageURLReply(_ context.Context, _ int64, _ int, url string) error {
	m.Message = url
	return m.err
}

func (m *MockImageSender) SendImageFileReply(_ context.Context, _ int64, _ int, file []byte) error {
	m.Message = string(file)
	return m.err
}

func TestNewImageHandler(t *testing.T) {
	mg := &MockImageGenerator{}
	ms := &MockImageSender{}
	ts := &MockTextSender{}

	imageHandler := NewImageHandler(mg, ms, ts, "/image")

	assert.NotNil(t, imageHandler)
	assert.Equal(t, "/image", imageHandler.GetCommand())
}

func TestImageRepondSuccessful(t *testing.T) {
	mg := &MockImageGenerator{response: "https://example.org/image.png"}
	mi := &MockImageSender{}
	mt := &MockTextSender{}

	imageHandler := NewImageHandler(mg, mi, mt, "/image")

	err := imageHandler.Respond(context.Background(), time.Minute,
		&domain.Message{ChatID: 1, ID: 1, Text: "/image prompt"})
	require.NoError(t, err)

	assert.Equal(t, "https://example.org/image.png", mi.Message)
}

func TestImageRepondSendFailed(t *testing.T) {
	mg := &MockImageGenerator{response: "https://example.org/image.png"}
	mi := &MockImageSender{err: errors.New("mock error")}
	mt := &MockTextSender{}

	imageHandler := NewImageHandler(mg, mi, mt, "/image")

	err := imageHandler.Respond(context.Background(), time.Minute,
		&domain.Message{ChatID: 1, ID: 1, Text: "/image prompt"})
	require.Errorf(t, err, "mock error")
}

func TestImageRepondErrorEmptyPrompt(t *testing.T) {
	mg := &MockImageGenerator{}
	mi := &MockImageSender{}
	mt := &MockTextSender{}

	imageHandler := NewImageHandler(mg, mi, mt, "/image")

	err := imageHandler.Respond(context.Background(), time.Minute,
		&domain.Message{ChatID: 1, ID: 1, Text: "/image"})
	require.NoError(t, err)

	assert.Equal(t, "missing image prompt", mt.Message)
}

func TestImageRepondErrorGenerating(t *testing.T) {
	mg := &MockImageGenerator{err: errors.New("mock error")}
	mi := &MockImageSender{}
	mt := &MockTextSender{}

	imageHandler := NewImageHandler(mg, mi, mt, "/image")

	err := imageHandler.Respond(context.Background(), time.Minute,
		&domain.Message{ChatID: 1, ID: 1, Text: "/image prompt"})
	require.NoError(t, err)

	assert.Equal(t, "error getting FAL response: mock error", mt.Message)
}

func TestImageRepondErrorGeneratingAndSending(t *testing.T) {
	mg := &MockImageGenerator{err: errors.New("mock error")}
	mi := &MockImageSender{}
	mt := &MockTextSender{err: errors.New("mock error")}

	imageHandler := NewImageHandler(mg, mi, mt, "/image")

	err := imageHandler.Respond(context.Background(), time.Minute,
		&domain.Message{ChatID: 1, ID: 1, Text: "/image prompt"})
	require.Errorf(t, err, "mock error")

	assert.Equal(t, "error getting FAL response: mock error", mt.Message)
}

func TestImageRepondErrorEmptyPromptAndErrorSending(t *testing.T) {
	mg := &MockImageGenerator{}
	mi := &MockImageSender{}
	mt := &MockTextSender{err: errors.New("mock error")}

	imageHandler := NewImageHandler(mg, mi, mt, "/image")

	err := imageHandler.Respond(context.Background(), time.Minute,
		&domain.Message{ChatID: 1, ID: 1, Text: "/image"})
	require.Errorf(t, err, "mock error")

	assert.Equal(t, "missing image prompt", mt.Message)
}
