package command

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
	imageURL string
	err      error
	Message  string
}

func (m *MockImageGenerator) GenerateFromPrompt(_ context.Context, prompt string) (string, error) {
	m.Message = prompt
	return m.response, m.err
}

func (m *MockImageGenerator) EditFromPrompt(_ context.Context, _ domain.Prompt) (string, error) {
	return m.imageURL, m.err
}

type MockImageSender struct {
	calledURL string
	called    bool
	err       error
}

func (m *MockImageSender) SendImageURLReply(_ context.Context, _ *domain.Message, imageURL string) error {
	m.calledURL = imageURL
	m.called = true
	return m.err
}

func (m *MockImageSender) SendImageFileReply(_ context.Context, _ *domain.Message, file []byte) error {
	m.calledURL = string(file)
	return m.err
}

func TestNewImageHandler(t *testing.T) {
	mg := &MockImageGenerator{}
	ms := &MockImageSender{}
	ts := &MockTextSender{}

	imageHandler := NewImage(mg, ms, ts, "/image")

	assert.NotNil(t, imageHandler)
	assert.Equal(t, "/image", imageHandler.GetCommand())
}

func TestImageRepondSuccessful(t *testing.T) {
	mg := &MockImageGenerator{response: "https://example.org/image.png"}
	mi := &MockImageSender{}
	mt := &MockTextSender{}

	imageHandler := NewImage(mg, mi, mt, "/image")

	err := imageHandler.Respond(t.Context(), time.Minute,
		&domain.Message{ChatID: 1, ID: 1, Text: "/image prompt"})
	require.NoError(t, err)

	assert.Equal(t, "https://example.org/image.png", mi.calledURL)
}

func TestImageRepondSendFailed(t *testing.T) {
	mg := &MockImageGenerator{response: "https://example.org/image.png"}
	mi := &MockImageSender{err: errors.New("mock error")}
	mt := &MockTextSender{}

	imageHandler := NewImage(mg, mi, mt, "/image")

	_ = imageHandler.Respond(t.Context(), time.Minute,
		&domain.Message{ChatID: 1, ID: 1, Text: "/image prompt"})
	require.Equal(t, "error sending edited image: mock error", mt.Message)
}

func TestImageRepondErrorEmptyPrompt(t *testing.T) {
	mg := &MockImageGenerator{}
	mi := &MockImageSender{}
	mt := &MockTextSender{}

	imageHandler := NewImage(mg, mi, mt, "/image")

	err := imageHandler.Respond(t.Context(), time.Minute,
		&domain.Message{ChatID: 1, ID: 1, Text: "/image"})
	require.NoError(t, err)

	assert.Equal(t, "missing image prompt", mt.Message)
}

func TestImageRepondErrorGenerating(t *testing.T) {
	mg := &MockImageGenerator{err: errors.New("mock error")}
	mi := &MockImageSender{}
	mt := &MockTextSender{}

	imageHandler := NewImage(mg, mi, mt, "/image")

	_ = imageHandler.Respond(t.Context(), time.Minute,
		&domain.Message{ChatID: 1, ID: 1, Text: "/image prompt"})

	require.Equal(t, "error generating image: mock error", mt.Message)
}

func TestImageRepondErrorGeneratingAndSending(t *testing.T) {
	mg := &MockImageGenerator{err: errors.New("mock error")}
	mi := &MockImageSender{}
	mt := &MockTextSender{err: errors.New("mock error")}

	imageHandler := NewImage(mg, mi, mt, "/image")

	_ = imageHandler.Respond(t.Context(), time.Minute,
		&domain.Message{ChatID: 1, ID: 1, Text: "/image prompt"})

	require.EqualError(t, mt.err, "mock error")
}

func TestImageRepondErrorEmptyPromptAndErrorSending(t *testing.T) {
	mg := &MockImageGenerator{}
	mi := &MockImageSender{}
	mt := &MockTextSender{err: errors.New("mock error")}

	imageHandler := NewImage(mg, mi, mt, "/image")

	_ = imageHandler.Respond(t.Context(), time.Minute,
		&domain.Message{ChatID: 1, ID: 1, Text: "/image"})

	require.EqualError(t, mt.err, "mock error")
}
