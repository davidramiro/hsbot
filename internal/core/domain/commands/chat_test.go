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

type MockTextGenerator struct {
	response string
	err      error
	Message  string
}

func (m *MockTextGenerator) GenerateFromPrompt(_ context.Context, _ []domain.Prompt) (string, error) {
	return m.response, m.err
}

type MockTextSender struct {
	err     error
	Message string
}

func (m *MockTextSender) SendMessageReply(_ context.Context, _ int64, _ int, message string) error {
	m.Message = message
	return m.err
}

func (m *MockTextSender) SendChatAction(_ context.Context, _ int64, _ domain.Action) {}

func TestChatHandlerClearingCache(t *testing.T) {
	mg := &MockTextGenerator{response: "mock response"}
	ms := &MockTextSender{}
	mt := &MockTranscriber{}

	chatHandler := NewChatHandler(mg, ms, mt,
		"/chat", time.Second*3, time.Second)

	assert.NotNil(t, chatHandler)

	err := chatHandler.Respond(context.Background(), time.Minute, &domain.Message{ChatID: 1, ID: 1, Text: "/chat prompt"})

	require.NoError(t, err)
	assert.Equal(t, "mock response", ms.Message)
	assert.Len(t, chatHandler.cache, 1)

	time.Sleep(time.Second * 4)

	assert.Empty(t, chatHandler.cache)
}

func TestChatHandlerCache(t *testing.T) {
	mg := &MockTextGenerator{response: "mock response"}
	ms := &MockTextSender{}
	mt := &MockTranscriber{}

	chatHandler := NewChatHandler(mg, ms, mt,
		"/chat", time.Second*3, time.Second)

	assert.NotNil(t, chatHandler)

	err := chatHandler.Respond(context.Background(), time.Minute, &domain.Message{ChatID: 1, ID: 1, Text: "/chat prompt"})

	require.NoError(t, err)
	assert.Equal(t, "mock response", ms.Message)
	assert.Len(t, chatHandler.cache, 1)

	err = chatHandler.Respond(context.Background(), time.Minute, &domain.Message{ChatID: 1, ID: 2, Text: "/chat prompt2"})
	require.NoError(t, err)
	assert.Len(t, chatHandler.cache, 1)

	assert.Len(t, chatHandler.cache[1].messages, 4)

	assert.Equal(t, "prompt", chatHandler.cache[1].messages[0].Prompt)
	assert.Equal(t, "mock response", chatHandler.cache[1].messages[1].Prompt)
	assert.Equal(t, "prompt2", chatHandler.cache[1].messages[2].Prompt)
	assert.Equal(t, "mock response", chatHandler.cache[1].messages[3].Prompt)
}

func TestGeneratorError(t *testing.T) {
	mg := &MockTextGenerator{err: errors.New("mock error")}
	ms := &MockTextSender{}
	mt := &MockTranscriber{}

	chatHandler := NewChatHandler(mg, ms, mt,
		"/chat", time.Second*3, time.Second)

	assert.NotNil(t, chatHandler)

	err := chatHandler.Respond(context.Background(), time.Minute, &domain.Message{ChatID: 1, ID: 1, Text: "/chat prompt"})
	require.NoError(t, err)

	assert.Equal(t, "failed to generate reply: mock error", ms.Message)
}

func TestEmptyPromptError(t *testing.T) {
	mg := &MockTextGenerator{err: errors.New("mock error")}
	ms := &MockTextSender{}
	mt := &MockTranscriber{}

	chatHandler := NewChatHandler(mg, ms, mt,
		"/chat", time.Second*3, time.Second)

	assert.NotNil(t, chatHandler)

	err := chatHandler.Respond(context.Background(), time.Minute, &domain.Message{ChatID: 1, ID: 1, Text: "/chat"})
	require.NoError(t, err)

	assert.Equal(t, "please input a prompt", ms.Message)
}

func TestSendMessageError(t *testing.T) {
	mg := &MockTextGenerator{response: "mock response"}
	ms := &MockTextSender{err: errors.New("mock error")}
	mt := &MockTranscriber{}

	chatHandler := NewChatHandler(mg, ms, mt,
		"/chat", time.Second*3, time.Second)

	assert.NotNil(t, chatHandler)

	err := chatHandler.Respond(context.Background(), time.Minute, &domain.Message{ChatID: 1, ID: 1, Text: "/chat prompt"})

	assert.Equal(t, "mock response", ms.Message)
	require.Errorf(t, err, "failed to send reply")
}

func TestSendGenerateErrorAndMessageError(t *testing.T) {
	mg := &MockTextGenerator{err: errors.New("mock error")}
	ms := &MockTextSender{err: errors.New("mock error")}
	mt := &MockTranscriber{}

	chatHandler := NewChatHandler(mg, ms, mt,
		"/chat", time.Second*3, time.Second)

	assert.NotNil(t, chatHandler)

	err := chatHandler.Respond(context.Background(), time.Minute, &domain.Message{ChatID: 1, ID: 1, Text: "/chat prompt"})

	assert.Equal(t, "failed to generate reply: mock error", ms.Message)
	require.Errorf(t, err, "failed to send reply")
}
