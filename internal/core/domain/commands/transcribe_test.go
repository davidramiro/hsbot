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

type MockTranscriber struct {
	err error
}

func (m *MockTranscriber) GenerateFromAudio(_ context.Context, url string) (string, error) {
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

	err := transcribeHandler.Respond(t.Context(), time.Minute, &domain.Message{AudioURL: "mock"})
	require.NoError(t, err)

	assert.Equal(t, "mock", ts.Message)
}

func TestTranscribeRespondErrorGenerating(t *testing.T) {
	mt := &MockTranscriber{err: errors.New("mock error")}
	ts := &MockTextSender{}

	transcribeHandler := NewTranscribeHandler(mt, ts, "/transcribe")

	err := transcribeHandler.Respond(t.Context(), time.Minute, &domain.Message{AudioURL: "mock"})
	require.Error(t, err)

	assert.Equal(t, "failed to generate audio: mock error", ts.Message)
}

func TestTranscribeRespondErrorGeneratingAndSending(t *testing.T) {
	mt := &MockTranscriber{err: errors.New("mock error")}
	ts := &MockTextSender{err: errors.New("mock error")}

	transcribeHandler := NewTranscribeHandler(mt, ts, "/transcribe")

	err := transcribeHandler.Respond(t.Context(), time.Minute, &domain.Message{AudioURL: "mock"})
	require.Errorf(t, err, "mock error")

	assert.Equal(t, "failed to generate audio: mock error", ts.Message)
}

func TestTranscribeRespondErrorSending(t *testing.T) {
	mt := &MockTranscriber{}
	ts := &MockTextSender{err: errors.New("mock error")}

	transcribeHandler := NewTranscribeHandler(mt, ts, "/transcribe")

	_ = transcribeHandler.Respond(t.Context(), time.Minute, &domain.Message{AudioURL: "mock"})
	assert.Equal(t, "error sending transcript: mock error", ts.Message)
}

func TestTranscribeRespondErrorEmptyURLAndSending(t *testing.T) {
	mt := &MockTranscriber{}
	ts := &MockTextSender{err: errors.New("mock error")}

	transcribeHandler := NewTranscribeHandler(mt, ts, "/transcribe")

	_ = transcribeHandler.Respond(t.Context(), time.Minute, &domain.Message{})
	assert.Equal(t, "reply to an audio", ts.Message)
}
