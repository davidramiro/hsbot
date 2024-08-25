package commands

import (
	"context"
	"errors"
	"github.com/stretchr/testify/assert"
	"hsbot/internal/core/domain"
	"testing"
	"time"
)

type MockTextGenerator struct {
	response string
	err      error
	Message  string
}

func (m *MockTextGenerator) GenerateFromPrompt(ctx context.Context, prompts []domain.Prompt) (string, error) {
	return m.response, m.err
}

type MockTextSender struct {
	err     error
	Message string
}

func (m *MockTextSender) SendMessageReply(ctx context.Context, chatID int64, messageID int, message string) error {
	m.Message = message
	return m.err
}

func (m *MockTextSender) SendChatAction(ctx context.Context, chatID int64, action domain.Action) {}

func TestChatHandlerClearingCache(t *testing.T) {
	mg := &MockTextGenerator{response: "mock response"}
	ms := &MockTextSender{}

	chatHandler := NewChatHandler(mg, ms,
		"/chat", time.Second*3, time.Second)

	assert.NotNil(t, chatHandler)

	err := chatHandler.Respond(context.Background(), &domain.Message{ChatID: 1, ID: 1, Text: "/chat prompt"})

	assert.NoError(t, err)
	assert.Equal(t, "mock response", ms.Message)
	assert.Equal(t, len(chatHandler.cache), 1)

	time.Sleep(time.Second * 4)

	assert.Equal(t, len(chatHandler.cache), 0)
}

func TestChatHandlerCache(t *testing.T) {
	mg := &MockTextGenerator{response: "mock response"}
	ms := &MockTextSender{}

	chatHandler := NewChatHandler(mg, ms,
		"/chat", time.Second*3, time.Second)

	assert.NotNil(t, chatHandler)

	err := chatHandler.Respond(context.Background(), &domain.Message{ChatID: 1, ID: 1, Text: "/chat prompt"})

	assert.NoError(t, err)
	assert.Equal(t, "mock response", ms.Message)
	assert.Equal(t, len(chatHandler.cache), 1)

	err = chatHandler.Respond(context.Background(), &domain.Message{ChatID: 1, ID: 2, Text: "/chat prompt2"})
	assert.NoError(t, err)
	assert.Equal(t, len(chatHandler.cache), 1)

	assert.Equal(t, len(chatHandler.cache[1].messages), 4)

	assert.Equal(t, chatHandler.cache[1].messages[0].Prompt, "prompt")
	assert.Equal(t, chatHandler.cache[1].messages[1].Prompt, "mock response")
	assert.Equal(t, chatHandler.cache[1].messages[2].Prompt, "prompt2")
	assert.Equal(t, chatHandler.cache[1].messages[3].Prompt, "mock response")
}

func TestGeneratorError(t *testing.T) {
	mg := &MockTextGenerator{err: errors.New("mock error")}
	ms := &MockTextSender{}

	chatHandler := NewChatHandler(mg, ms,
		"/chat", time.Second*3, time.Second)

	assert.NotNil(t, chatHandler)

	chatHandler.Respond(context.Background(), &domain.Message{ChatID: 1, ID: 1, Text: "/chat prompt"})

	assert.Equal(t, "failed to generate reply: mock error", ms.Message)
}

func TestEmptyPromptError(t *testing.T) {
	mg := &MockTextGenerator{err: errors.New("mock error")}
	ms := &MockTextSender{}

	chatHandler := NewChatHandler(mg, ms,
		"/chat", time.Second*3, time.Second)

	assert.NotNil(t, chatHandler)

	chatHandler.Respond(context.Background(), &domain.Message{ChatID: 1, ID: 1, Text: "/chat"})

	assert.Equal(t, "please input a prompt", ms.Message)
}

func TestSendMessageError(t *testing.T) {
	mg := &MockTextGenerator{response: "mock response"}
	ms := &MockTextSender{err: errors.New("mock error")}

	chatHandler := NewChatHandler(mg, ms,
		"/chat", time.Second*3, time.Second)

	assert.NotNil(t, chatHandler)

	err := chatHandler.Respond(context.Background(), &domain.Message{ChatID: 1, ID: 1, Text: "/chat prompt"})

	assert.Equal(t, "mock response", ms.Message)
	assert.Errorf(t, err, "failed to send reply")
}

func TestSendGenerateErrorAndMessageError(t *testing.T) {
	mg := &MockTextGenerator{err: errors.New("mock error")}
	ms := &MockTextSender{err: errors.New("mock error")}

	chatHandler := NewChatHandler(mg, ms,
		"/chat", time.Second*3, time.Second)

	assert.NotNil(t, chatHandler)

	err := chatHandler.Respond(context.Background(), &domain.Message{ChatID: 1, ID: 1, Text: "/chat prompt"})

	assert.Equal(t, "failed to generate reply: mock error", ms.Message)
	assert.Errorf(t, err, "failed to send reply")
}
