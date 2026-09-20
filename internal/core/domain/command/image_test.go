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
	data    []byte
	err     error
	Message string
}

func (m *MockImageGenerator) GenerateImage(_ context.Context, prompt string) (domain.GeneratedImage, error) {
	m.Message = prompt
	return domain.GeneratedImage{Data: m.data, Cost: 0.01}, m.err
}

func (m *MockImageGenerator) EditImage(_ context.Context, _ domain.Prompt) (domain.GeneratedImage, error) {
	return domain.GeneratedImage{Data: m.data, Cost: 0.01}, m.err
}

type MockImageSender struct {
	calledFile []byte
	calledURL  string
	called     bool
	err        error
}

func (m *MockImageSender) SendImageURLReply(_ context.Context, _ *domain.Message, imageURL string) error {
	m.calledURL = imageURL
	m.called = true
	return m.err
}

func (m *MockImageSender) SendImageFileReply(_ context.Context, _ *domain.Message, file []byte) error {
	m.calledFile = file
	m.calledURL = string(file)
	m.called = true
	return m.err
}

func TestNewImageHandler(t *testing.T) {
	mg := &MockImageGenerator{}
	ms := &MockImageSender{}
	ts := &MockTextSender{}
	mtr := &MockTracker{withinLimit: true}

	imageHandler := NewImage(mg, ms, ts, mtr, "/image")

	assert.NotNil(t, imageHandler)
	assert.Equal(t, "/image", imageHandler.GetCommand())
}

func TestImageRepondSuccessful(t *testing.T) {
	mg := &MockImageGenerator{data: []byte("png")}
	ms := &MockImageSender{}
	ts := &MockTextSender{}
	mtr := &MockTracker{withinLimit: true}

	imageHandler := NewImage(mg, ms, ts, mtr, "/image")

	err := imageHandler.Respond(t.Context(), time.Minute,
		&domain.Message{ChatID: 1, ID: 1, Text: "/image prompt"})
	require.NoError(t, err)

	assert.Equal(t, []byte("png"), ms.calledFile)
}

func TestImageRepondSendFailed(t *testing.T) {
	mg := &MockImageGenerator{data: []byte("png")}
	mi := &MockImageSender{err: errors.New("mock error")}
	mt := &MockTextSender{}
	mtr := &MockTracker{withinLimit: true}

	imageHandler := NewImage(mg, mi, mt, mtr, "/image")

	_ = imageHandler.Respond(t.Context(), time.Minute,
		&domain.Message{ChatID: 1, ID: 1, Text: "/image prompt"})
	require.Equal(t, "error sending image: mock error", mt.Message)
}

func TestImageRepondErrorEmptyPrompt(t *testing.T) {
	mg := &MockImageGenerator{}
	mi := &MockImageSender{}
	mt := &MockTextSender{}
	mtr := &MockTracker{withinLimit: true}

	imageHandler := NewImage(mg, mi, mt, mtr, "/image")

	err := imageHandler.Respond(t.Context(), time.Minute,
		&domain.Message{ChatID: 1, ID: 1, Text: "/image"})
	require.NoError(t, err)

	assert.Equal(t, "missing image prompt", mt.Message)
}

func TestImageRepondErrorGenerating(t *testing.T) {
	mg := &MockImageGenerator{err: errors.New("mock error")}
	mi := &MockImageSender{}
	mt := &MockTextSender{}
	mtr := &MockTracker{withinLimit: true}

	imageHandler := NewImage(mg, mi, mt, mtr, "/image")

	_ = imageHandler.Respond(t.Context(), time.Minute,
		&domain.Message{ChatID: 1, ID: 1, Text: "/image prompt"})

	require.Equal(t, "error generating image: mock error", mt.Message)
}

func TestImageRepondErrorGeneratingAndSending(t *testing.T) {
	mg := &MockImageGenerator{err: errors.New("mock error")}
	mi := &MockImageSender{}
	mt := &MockTextSender{err: errors.New("mock error")}
	mtr := &MockTracker{withinLimit: true}

	imageHandler := NewImage(mg, mi, mt, mtr, "/image")

	_ = imageHandler.Respond(t.Context(), time.Minute,
		&domain.Message{ChatID: 1, ID: 1, Text: "/image prompt"})

	require.EqualError(t, mt.err, "mock error")
}

func TestImageRepondErrorEmptyPromptAndErrorSending(t *testing.T) {
	mg := &MockImageGenerator{}
	mi := &MockImageSender{}
	mt := &MockTextSender{err: errors.New("mock error")}
	mtr := &MockTracker{withinLimit: true}

	imageHandler := NewImage(mg, mi, mt, mtr, "/image")

	_ = imageHandler.Respond(t.Context(), time.Minute,
		&domain.Message{ChatID: 1, ID: 1, Text: "/image"})

	require.EqualError(t, mt.err, "mock error")
}
