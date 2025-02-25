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
	response        string
	thoughtResponse string
	err             error
	Message         string
}

func (m *MockTextGenerator) GenerateFromPrompt(_ context.Context, _ []domain.Prompt) (string, error) {
	return m.response, m.err
}

func (m *MockTextGenerator) ThinkFromPrompt(_ context.Context, _ domain.Prompt) (string, string, error) {
	return m.response, m.thoughtResponse, m.err
}

type MockTextSender struct {
	err     error
	Message string
}

func (m *MockTextSender) SendMessageReply(_ context.Context, _ int64, _ int, message string) (int, error) {
	m.Message = message
	return 0, m.err
}

func (m *MockTextSender) SendChatAction(_ context.Context, _ int64, _ domain.Action) {}

func TestChatHandlerClearingCache(t *testing.T) {
	mg := &MockTextGenerator{response: "mock response"}
	ms := &MockTextSender{}
	mt := &MockTranscriber{}

	chatHandler := NewChatHandler(mg, ms, mt,
		"/chat", time.Second*3)

	assert.NotNil(t, chatHandler)

	err := chatHandler.Respond(context.Background(), time.Minute, &domain.Message{ChatID: 1, ID: 1, Text: "/chat prompt"})

	require.NoError(t, err)
	assert.Equal(t, "mock response", ms.Message)

	c, ok := chatHandler.cache.Load(int64(1))
	require.True(t, ok)

	conversation, ok := c.(*Conversation)
	require.True(t, ok)
	assert.Len(t, conversation.messages, 2)

	time.Sleep(time.Second * 4)

	_, ok = chatHandler.cache.Load(int64(1))
	assert.False(t, ok)
}

func TestChatHandlerCache(t *testing.T) {
	mg := &MockTextGenerator{response: "mock response"}
	ms := &MockTextSender{}
	mt := &MockTranscriber{}

	chatHandler := NewChatHandler(mg, ms, mt,
		"/chat", time.Second*3)

	assert.NotNil(t, chatHandler)

	err := chatHandler.Respond(context.Background(), time.Minute, &domain.Message{
		ChatID: 1, ID: 1, Username: "@unit", Text: "/chat prompt"})

	require.NoError(t, err)
	assert.Equal(t, "mock response", ms.Message)

	size := 0
	chatHandler.cache.Range(func(_, _ interface{}) bool {
		size++
		return true
	})
	assert.Equal(t, 1, size)

	err = chatHandler.Respond(context.Background(), time.Minute, &domain.Message{
		ChatID: 1, ID: 2, Username: "@unit", Text: "/chat prompt2"})
	require.NoError(t, err)

	c, ok := chatHandler.cache.Load(int64(1))
	require.True(t, ok)

	conversation, ok := c.(*Conversation)
	require.True(t, ok)
	assert.Len(t, conversation.messages, 4)

	assert.Equal(t, "@unit: prompt", conversation.messages[0].Prompt)
	assert.Equal(t, "mock response", conversation.messages[1].Prompt)
	assert.Equal(t, "@unit: prompt2", conversation.messages[2].Prompt)
	assert.Equal(t, "mock response", conversation.messages[3].Prompt)
}

func TestChatHandlerCacheMultipleConversations(t *testing.T) {
	mg := &MockTextGenerator{response: "mock response"}
	ms := &MockTextSender{}
	mt := &MockTranscriber{}

	chatHandler := NewChatHandler(mg, ms, mt,
		"/chat", time.Second*3)

	assert.NotNil(t, chatHandler)

	err := chatHandler.Respond(context.Background(), time.Minute, &domain.Message{
		ChatID: 1, ID: 1, Username: "@unit", Text: "/chat prompt chat id 1"})

	require.NoError(t, err)
	assert.Equal(t, "mock response", ms.Message)

	err = chatHandler.Respond(context.Background(), time.Minute, &domain.Message{
		ChatID: 2, ID: 2, Username: "@unit", Text: "/chat prompt chat id 2"})
	require.NoError(t, err)

	size := 0
	chatHandler.cache.Range(func(_, _ interface{}) bool {
		size++
		return true
	})
	assert.Equal(t, 2, size)

	c1, ok := chatHandler.cache.Load(int64(1))
	require.True(t, ok)

	conversation1, ok := c1.(*Conversation)
	require.True(t, ok)
	assert.Len(t, conversation1.messages, 2)

	c2, ok := chatHandler.cache.Load(int64(2))
	require.True(t, ok)

	conversation2, ok := c2.(*Conversation)
	require.True(t, ok)
	assert.Len(t, conversation2.messages, 2)

	assert.Equal(t, "@unit: prompt chat id 1", conversation1.messages[0].Prompt)
	assert.Equal(t, "mock response", conversation1.messages[1].Prompt)
	assert.Equal(t, "@unit: prompt chat id 2", conversation2.messages[0].Prompt)
	assert.Equal(t, "mock response", conversation2.messages[1].Prompt)
}

func TestChatHandlerCacheResetTimeout(t *testing.T) {
	mg := &MockTextGenerator{response: "mock response"}
	ms := &MockTextSender{}
	mt := &MockTranscriber{}

	chatHandler := NewChatHandler(mg, ms, mt,
		"/chat", time.Second*3)

	assert.NotNil(t, chatHandler)

	err := chatHandler.Respond(context.Background(), time.Minute, &domain.Message{
		ChatID: 1, ID: 1, Username: "@unit", Text: "/chat prompt"})

	require.NoError(t, err)
	assert.Equal(t, "mock response", ms.Message)

	size := 0
	chatHandler.cache.Range(func(_, _ interface{}) bool {
		size++
		return true
	})
	assert.Equal(t, 1, size)

	time.Sleep(time.Second * 2)

	err = chatHandler.Respond(context.Background(), time.Minute, &domain.Message{
		ChatID: 1, ID: 2, Username: "@unit", Text: "/chat prompt2"})
	require.NoError(t, err)

	size = 0
	chatHandler.cache.Range(func(_, _ interface{}) bool {
		size++
		return true
	})
	assert.Equal(t, 1, size)

	time.Sleep(time.Second * 2)

	err = chatHandler.Respond(context.Background(), time.Minute, &domain.Message{
		ChatID: 1, ID: 2, Username: "@unit", Text: "/chat prompt3"})
	require.NoError(t, err)

	c, ok := chatHandler.cache.Load(int64(1))
	require.True(t, ok)

	conversation, ok := c.(*Conversation)
	require.True(t, ok)
	assert.Len(t, conversation.messages, 6)

	assert.Equal(t, "@unit: prompt", conversation.messages[0].Prompt)
	assert.Equal(t, "mock response", conversation.messages[1].Prompt)
	assert.Equal(t, "@unit: prompt2", conversation.messages[2].Prompt)
	assert.Equal(t, "mock response", conversation.messages[3].Prompt)
	assert.Equal(t, "@unit: prompt3", conversation.messages[4].Prompt)
	assert.Equal(t, "mock response", conversation.messages[5].Prompt)
}

func TestGeneratorError(t *testing.T) {
	mg := &MockTextGenerator{err: errors.New("mock error")}
	ms := &MockTextSender{}
	mt := &MockTranscriber{}

	chatHandler := NewChatHandler(mg, ms, mt,
		"/chat", time.Second*3)

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
		"/chat", time.Second*3)

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
		"/chat", time.Second*3)

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
		"/chat", time.Second*3)

	assert.NotNil(t, chatHandler)

	err := chatHandler.Respond(context.Background(), time.Minute, &domain.Message{ChatID: 1, ID: 1, Text: "/chat prompt"})

	assert.Equal(t, "failed to generate reply: mock error", ms.Message)
	require.Errorf(t, err, "failed to send reply")
}
