package command

import (
	"context"
	"errors"
	"hsbot/internal/core/domain"
	"testing"
	"time"

	"github.com/spf13/viper"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type MockTextGenerator struct {
	response        string
	thoughtResponse string
	err             error
	Message         string
}

func (m *MockTextGenerator) GenerateFromPrompt(_ context.Context, _ []domain.Prompt) (domain.ModelResponse, error) {
	return domain.ModelResponse{
		Response: m.response,
		Metadata: domain.ResponseMetadata{
			Model:            "unit-test",
			CompletionTokens: 24,
			TotalTokens:      42,
		},
	}, m.err
}

func (m *MockTextGenerator) ThinkFromPrompt(_ context.Context, _ domain.Prompt) (string, string, error) {
	return m.response, m.thoughtResponse, m.err
}

type MockTextSender struct {
	err     error
	Message string
}

func (m *MockTextSender) SendMessageReply(_ context.Context, _ *domain.Message, message string) (int, error) {
	m.Message = message
	return 0, m.err
}

func (m *MockTextSender) NotifyAndReturnError(_ context.Context, err error, _ *domain.Message) error {
	m.Message = err.Error()
	if m.err != nil {
		return m.err
	}
	return err
}

func (m *MockTextSender) SendChatAction(_ context.Context, _ int64, _ domain.Action) {}

func TestChatHandlerSimpleSuccess(t *testing.T) {
	mg := &MockTextGenerator{response: "mock response"}
	ms := &MockTextSender{}
	mt := &MockTranscriber{}

	chatHandler, _ := NewChat(mg, ms, mt,
		"/chat", time.Second*3)

	assert.NotNil(t, chatHandler)

	err := chatHandler.Respond(t.Context(), time.Minute, &domain.Message{ChatID: 1, ID: 1, Text: "/chat prompt"})

	require.NoError(t, err)
	assert.Equal(t, "mock response", ms.Message)
}

func TestChatHandlerTranscribeSuccess(t *testing.T) {
	mg := &MockTextGenerator{response: "mock response"}
	ms := &MockTextSender{}
	mt := &MockTranscriber{}

	chatHandler, _ := NewChat(mg, ms, mt,
		"/chat", time.Second*3)

	assert.NotNil(t, chatHandler)

	err := chatHandler.Respond(t.Context(), time.Minute, &domain.Message{
		ChatID: 1, ID: 1, Username: "@unit", Text: "/chat transcribe", AudioURL: "foo"})

	c, ok := chatHandler.cache.Load(int64(1))
	require.True(t, ok)

	conversation, ok := c.(*Conversation)
	require.True(t, ok)
	assert.Len(t, conversation.messages, 2)

	assert.Equal(t, "@unit: transcribe: foo", conversation.messages[0].Prompt)
	assert.Equal(t, "mock response", conversation.messages[1].Prompt)

	require.NoError(t, err)
	assert.Equal(t, "mock response", ms.Message)
}

func TestChatHandlerTranscribeError(t *testing.T) {
	mg := &MockTextGenerator{response: "mock response"}
	ms := &MockTextSender{}
	mt := &MockTranscriber{err: errors.New("foo")}

	chatHandler, _ := NewChat(mg, ms, mt,
		"/chat", time.Second*3)

	assert.NotNil(t, chatHandler)

	err := chatHandler.Respond(t.Context(), time.Minute, &domain.Message{
		ChatID: 1, ID: 1, Username: "@unit", Text: "/chat transcribe", AudioURL: "bar"})

	require.Errorf(t, err, "foo")
	assert.Equal(t, "failed to extract prompt: failed to generate transcript: foo", ms.Message)
}

func TestChatHandlerErrorEmptyPrompt(t *testing.T) {
	mg := &MockTextGenerator{response: "mock response"}
	ms := &MockTextSender{}
	mt := &MockTranscriber{}

	chatHandler, _ := NewChat(mg, ms, mt,
		"/chat", time.Second*3)

	assert.NotNil(t, chatHandler)

	err := chatHandler.Respond(t.Context(), time.Minute, &domain.Message{
		ChatID: 1, ID: 1, Username: "@unit", Text: "/chat"})

	require.Errorf(t, err, "foo")
	assert.Equal(t, "failed to extract prompt: empty prompt", ms.Message)
}

func TestChatHandlerDebugMessage(t *testing.T) {
	mg := &MockTextGenerator{response: "mock response"}
	ms := &MockTextSender{}
	mt := &MockTranscriber{}

	viper.SetDefault("bot.debug_replies", true)

	chatHandler, _ := NewChat(mg, ms, mt,
		"/chat", time.Second*3)

	assert.NotNil(t, chatHandler)

	err := chatHandler.Respond(t.Context(), time.Minute, &domain.Message{ChatID: 1, ID: 1, Text: "/chat prompt"})

	require.NoError(t, err)
	assert.Equal(t, "mock response\n\n--\ndebug: model: unit-test\nc tokens: 24 | total tokens: 42\n"+
		"convo size: 2", ms.Message)
}

func TestChatHandlerClearingCache(t *testing.T) {
	mg := &MockTextGenerator{response: "mock response"}
	ms := &MockTextSender{}
	mt := &MockTranscriber{}

	viper.SetDefault("bot.debug_replies", false)

	chatHandler, _ := NewChat(mg, ms, mt,
		"/chat", time.Second*3)

	assert.NotNil(t, chatHandler)

	err := chatHandler.Respond(t.Context(), time.Minute, &domain.Message{ChatID: 1, ID: 1, Text: "/chat prompt"})

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

	viper.SetDefault("bot.debug_replies", false)

	chatHandler, _ := NewChat(mg, ms, mt,
		"/chat", time.Second*3)

	assert.NotNil(t, chatHandler)

	err := chatHandler.Respond(t.Context(), time.Minute, &domain.Message{
		ChatID: 1, ID: 1, Username: "@unit", Text: "/chat prompt"})

	require.NoError(t, err)
	assert.Equal(t, "mock response", ms.Message)

	size := 0
	chatHandler.cache.Range(func(_, _ interface{}) bool {
		size++
		return true
	})
	assert.Equal(t, 1, size)

	err = chatHandler.Respond(t.Context(), time.Minute, &domain.Message{
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

	viper.SetDefault("bot.debug_replies", false)

	chatHandler, _ := NewChat(mg, ms, mt,
		"/chat", time.Second*3)

	assert.NotNil(t, chatHandler)

	err := chatHandler.Respond(t.Context(), time.Minute, &domain.Message{
		ChatID: 1, ID: 1, Username: "@unit", Text: "/chat prompt chat id 1"})

	require.NoError(t, err)
	assert.Equal(t, "mock response", ms.Message)

	err = chatHandler.Respond(t.Context(), time.Minute, &domain.Message{
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

	viper.SetDefault("bot.debug_replies", false)

	chatHandler, _ := NewChat(mg, ms, mt,
		"/chat", time.Second*3)

	assert.NotNil(t, chatHandler)

	err := chatHandler.Respond(t.Context(), time.Minute, &domain.Message{
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

	err = chatHandler.Respond(t.Context(), time.Minute, &domain.Message{
		ChatID: 1, ID: 2, Username: "@unit", Text: "/chat prompt2"})
	require.NoError(t, err)

	size = 0
	chatHandler.cache.Range(func(_, _ interface{}) bool {
		size++
		return true
	})
	assert.Equal(t, 1, size)

	time.Sleep(time.Second * 2)

	err = chatHandler.Respond(t.Context(), time.Minute, &domain.Message{
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

	chatHandler, _ := NewChat(mg, ms, mt,
		"/chat", time.Second*3)

	assert.NotNil(t, chatHandler)

	err := chatHandler.Respond(t.Context(), time.Minute, &domain.Message{ChatID: 1, ID: 1, Text: "/chat prompt"})
	require.Error(t, err)

	assert.Equal(t, "failed to generate response: mock error", ms.Message)
}

func TestEmptyPromptError(t *testing.T) {
	mg := &MockTextGenerator{err: errors.New("mock error")}
	ms := &MockTextSender{}
	mt := &MockTranscriber{}

	chatHandler, _ := NewChat(mg, ms, mt,
		"/chat", time.Second*3)

	assert.NotNil(t, chatHandler)

	err := chatHandler.Respond(t.Context(), time.Minute, &domain.Message{ChatID: 1, ID: 1, Text: "/chat"})
	require.Error(t, err)

	assert.Equal(t, "failed to extract prompt: empty prompt", ms.Message)
}

func TestSendMessageError(t *testing.T) {
	mg := &MockTextGenerator{response: "mock response"}
	ms := &MockTextSender{err: errors.New("mock error")}
	mt := &MockTranscriber{}

	chatHandler, _ := NewChat(mg, ms, mt,
		"/chat", time.Second*3)

	assert.NotNil(t, chatHandler)

	err := chatHandler.Respond(t.Context(), time.Minute, &domain.Message{ChatID: 1, ID: 1, Text: "/chat prompt"})

	assert.Equal(t, "mock response", ms.Message)
	require.Errorf(t, err, "failed to send reply")
}

func TestSendGenerateErrorAndMessageError(t *testing.T) {
	mg := &MockTextGenerator{err: errors.New("mock error")}
	ms := &MockTextSender{err: errors.New("mock error")}
	mt := &MockTranscriber{}

	chatHandler, _ := NewChat(mg, ms, mt,
		"/chat", time.Second*3)

	assert.NotNil(t, chatHandler)

	err := chatHandler.Respond(t.Context(), time.Minute, &domain.Message{ChatID: 1, ID: 1, Text: "/chat prompt"})

	assert.Equal(t, "failed to generate response: mock error", ms.Message)
	require.Errorf(t, err, "failed to send reply")
}

func TestFindModelByMessage(t *testing.T) {
	models := []domain.Model{
		{Keyword: "gpt"},
		{Keyword: "claude"},
	}
	defaultModel := domain.Model{Keyword: "default"}

	handler := &Chat{
		models:       models,
		defaultModel: defaultModel,
	}

	tests := []struct {
		name        string
		message     string
		wantModel   domain.Model
		wantMessage string
	}{
		{
			name:        "Match GPT model keyword (case-insensitive)",
			message:     "Hello #GPT",
			wantModel:   models[0],
			wantMessage: "Hello ",
		},
		{
			name:        "Match Claude model keyword",
			message:     "Please use #claude for this",
			wantModel:   models[1],
			wantMessage: "Please use  for this",
		},
		{
			name:        "No keyword, fallback to default",
			message:     "Just a normal message",
			wantModel:   defaultModel,
			wantMessage: "Just a normal message",
		},
		{
			name:        "Multiple keywords, first match is returned",
			message:     "#gpt and #claude in text",
			wantModel:   models[0],
			wantMessage: " and #claude in text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := tt.message
			gotModel := handler.findModelByMessage(&msg)
			assert.Equal(t, tt.wantModel, gotModel)
			assert.Equal(t, tt.wantMessage, msg)
		})
	}
}
