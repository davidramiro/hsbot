package commands

import (
	"context"
	"errors"
	"github.com/stretchr/testify/assert"
	"hsbot/internal/core/domain"
	"testing"
)

type MockTranscriber struct {
	err error
}

func (m *MockTranscriber) GenerateFromAudio(ctx context.Context, url string) (string, error) {
	return url, m.err
}

func TestNewTranscribeHandler(t *testing.T) {
	mt := &MockTranscriber{}
	ts := &MockTextSender{}

	transcribeHandler := NewTranscribeHandler(mt, ts, "/transcribe")

	assert.NotNil(t, transcribeHandler)
	assert.Equal(t, "/transcribe", transcribeHandler.GetCommand())
}

func TestTranscribeRespondSuccessful(t *testing.T) {
	mt := &MockTranscriber{}
	ts := &MockTextSender{}

	transcribeHandler := NewTranscribeHandler(mt, ts, "/transcribe")

	err := transcribeHandler.Respond(context.Background(), &domain.Message{AudioURL: "mock"})
	assert.NoError(t, err)

	assert.Equal(t, "mock", ts.Message)
}

func TestTranscribeRespondErrorGenerating(t *testing.T) {
	mt := &MockTranscriber{err: errors.New("mock error")}
	ts := &MockTextSender{}

	transcribeHandler := NewTranscribeHandler(mt, ts, "/transcribe")

	err := transcribeHandler.Respond(context.Background(), &domain.Message{AudioURL: "mock"})
	assert.NoError(t, err)

	assert.Equal(t, "transcription failed: mock error", ts.Message)
}

func TestTranscribeRespondErrorGeneratingAndSending(t *testing.T) {
	mt := &MockTranscriber{err: errors.New("mock error")}
	ts := &MockTextSender{err: errors.New("mock error")}

	transcribeHandler := NewTranscribeHandler(mt, ts, "/transcribe")

	err := transcribeHandler.Respond(context.Background(), &domain.Message{AudioURL: "mock"})
	assert.Errorf(t, err, "mock error")

	assert.Equal(t, "transcription failed: mock error", ts.Message)
}

func TestTranscribeRespondErrorSending(t *testing.T) {
	mt := &MockTranscriber{}
	ts := &MockTextSender{err: errors.New("mock error")}

	transcribeHandler := NewTranscribeHandler(mt, ts, "/transcribe")

	err := transcribeHandler.Respond(context.Background(), &domain.Message{AudioURL: "mock"})
	assert.Errorf(t, err, "mock error")

	assert.Equal(t, "mock", ts.Message)
}

func TestTranscribeRespondErrorEmptyURLAndSending(t *testing.T) {
	mt := &MockTranscriber{}
	ts := &MockTextSender{err: errors.New("mock error")}

	transcribeHandler := NewTranscribeHandler(mt, ts, "/transcribe")

	err := transcribeHandler.Respond(context.Background(), &domain.Message{})
	assert.Errorf(t, err, "mock error")

	assert.Equal(t, "reply to an audio", ts.Message)
}
