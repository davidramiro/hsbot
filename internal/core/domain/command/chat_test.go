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

type MockTextGenerator struct {
	response        string
	thoughtResponse string
	err             error
	Message         string
}

func (m *MockTextGenerator) GenerateFromPrompt(_ context.Context, _ []domain.Prompt) (domain.GeneratedText, error) {
	return domain.GeneratedText{
		Response: m.response,
		Metadata: domain.ResponseMetadata{
			Model:            "unit-test",
			CompletionTokens: 24,
			TotalTokens:      42,
			Cost:             0.42,
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

type MockAudioGenerator struct {
	audio []byte
	err   error
}

func (m *MockAudioGenerator) Speak(_ context.Context, _ string) (domain.GeneratedAudio, error) {
	return domain.GeneratedAudio{Data: m.audio}, m.err
}

type MockAudioSender struct {
	file []byte
	err  error
}

func (m *MockAudioSender) SendAudioReply(_ context.Context, _ *domain.Message, file []byte) error {
	m.file = file
	return m.err
}

type MockTracker struct {
	withinLimit bool
}

func (m MockTracker) GetSpent(_ int64) float64 {
	return 0.0
}

func (m MockTracker) AddCost(_ int64, _ float64) {
}

func (m MockTracker) CheckLimit(_ context.Context, _ int64) bool {
	return m.withinLimit
}

func TestChatHandlerSimpleSuccess(t *testing.T) {
	mg := &MockTextGenerator{response: "mock response"}
	ms := &MockTextSender{}
	mt := &MockTranscriber{}
	mtr := &MockTracker{withinLimit: true}

	chatHandler, _ := NewChat(ChatParams{
		TextGenerator: mg,
		TextSender:    ms,
		Transcriber:   mt,
		Command:       "/chat",
		CacheDuration: time.Second * 3,
		Tracker:       mtr,
	})

	assert.NotNil(t, chatHandler)

	err := chatHandler.Respond(t.Context(), time.Minute, &domain.Message{ChatID: 1, ID: 1, Text: "/chat prompt"})

	require.NoError(t, err)
	assert.Equal(t, "mock response", ms.Message)
}

func TestChatHandlerTranscribeSuccess(t *testing.T) {
	mg := &MockTextGenerator{response: "mock response"}
	ms := &MockTextSender{}
	mt := &MockTranscriber{}
	mtr := &MockTracker{withinLimit: true}

	chatHandler, _ := NewChat(ChatParams{
		TextGenerator: mg,
		TextSender:    ms,
		Transcriber:   mt,
		Command:       "/chat",
		CacheDuration: time.Second * 3,
		Tracker:       mtr,
	})

	assert.NotNil(t, chatHandler)

	err := chatHandler.Respond(t.Context(), time.Minute, &domain.Message{
		ChatID: 1, ID: 1, Username: "@unit", Text: "/chat transcribe", AudioURL: "foo"})

	conversation, ok := chatHandler.conversations.get(1)
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
	mtr := &MockTracker{withinLimit: true}

	chatHandler, _ := NewChat(ChatParams{
		TextGenerator: mg,
		TextSender:    ms,
		Transcriber:   mt,
		Command:       "/chat",
		CacheDuration: time.Second * 3,
		Tracker:       mtr,
	})

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
	mtr := &MockTracker{withinLimit: true}

	chatHandler, _ := NewChat(ChatParams{
		TextGenerator: mg,
		TextSender:    ms,
		Transcriber:   mt,
		Command:       "/chat",
		CacheDuration: time.Second * 3,
		Tracker:       mtr,
	})

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
	mtr := &MockTracker{withinLimit: true}

	chatHandler, _ := NewChat(ChatParams{
		TextGenerator: mg,
		TextSender:    ms,
		Transcriber:   mt,
		Command:       "/chat",
		CacheDuration: time.Second * 3,
		DebugReplies:  true,
		Tracker:       mtr,
	})

	assert.NotNil(t, chatHandler)

	err := chatHandler.Respond(t.Context(), time.Minute, &domain.Message{ChatID: 1, ID: 1, Text: "/chat prompt"})

	require.NoError(t, err)
	assert.Equal(t, "debug:\nmodel: unit-test | retries: 0\nc tokens: 24 | total tokens: 42\n"+
		"convo size: 2 | cost: 0.420000", ms.Message)
}

func TestChatHandlerClearingCache(t *testing.T) {
	mg := &MockTextGenerator{response: "mock response"}
	ms := &MockTextSender{}
	mt := &MockTranscriber{}
	mtr := &MockTracker{withinLimit: true}

	chatHandler, _ := NewChat(ChatParams{
		TextGenerator: mg,
		TextSender:    ms,
		Transcriber:   mt,
		Command:       "/chat",
		CacheDuration: time.Second * 3,
		Tracker:       mtr,
	})

	assert.NotNil(t, chatHandler)

	err := chatHandler.Respond(t.Context(), time.Minute, &domain.Message{ChatID: 1, ID: 1, Text: "/chat prompt"})

	require.NoError(t, err)
	assert.Equal(t, "mock response", ms.Message)

	conversation, ok := chatHandler.conversations.get(1)
	require.True(t, ok)
	assert.Len(t, conversation.messages, 2)

	time.Sleep(time.Second * 4)

	_, ok = chatHandler.conversations.get(1)
	assert.False(t, ok)
}

func TestChatHandlerCache(t *testing.T) {
	mg := &MockTextGenerator{response: "mock response"}
	ms := &MockTextSender{}
	mt := &MockTranscriber{}
	mtr := &MockTracker{withinLimit: true}

	chatHandler, _ := NewChat(ChatParams{
		TextGenerator: mg,
		TextSender:    ms,
		Transcriber:   mt,
		Command:       "/chat",
		CacheDuration: time.Second * 3,
		Tracker:       mtr,
	})

	assert.NotNil(t, chatHandler)

	err := chatHandler.Respond(t.Context(), time.Minute, &domain.Message{
		ChatID: 1, ID: 1, Username: "@unit", Text: "/chat prompt"})

	require.NoError(t, err)
	assert.Equal(t, "mock response", ms.Message)

	assert.Equal(t, 1, chatHandler.conversations.len())

	err = chatHandler.Respond(t.Context(), time.Minute, &domain.Message{
		ChatID: 1, ID: 2, Username: "@unit", Text: "/chat prompt2"})
	require.NoError(t, err)

	conversation, ok := chatHandler.conversations.get(1)
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
	mtr := &MockTracker{withinLimit: true}

	chatHandler, _ := NewChat(ChatParams{
		TextGenerator: mg,
		TextSender:    ms,
		Transcriber:   mt,
		Command:       "/chat",
		CacheDuration: time.Second * 3,
		Tracker:       mtr,
	})

	assert.NotNil(t, chatHandler)

	err := chatHandler.Respond(t.Context(), time.Minute, &domain.Message{
		ChatID: 1, ID: 1, Username: "@unit", Text: "/chat prompt chat id 1"})

	require.NoError(t, err)
	assert.Equal(t, "mock response", ms.Message)

	err = chatHandler.Respond(t.Context(), time.Minute, &domain.Message{
		ChatID: 2, ID: 2, Username: "@unit", Text: "/chat prompt chat id 2"})
	require.NoError(t, err)

	assert.Equal(t, 2, chatHandler.conversations.len())

	conversation1, ok := chatHandler.conversations.get(1)
	require.True(t, ok)
	assert.Len(t, conversation1.messages, 2)

	conversation2, ok := chatHandler.conversations.get(2)
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
	mtr := &MockTracker{withinLimit: true}

	chatHandler, _ := NewChat(ChatParams{
		TextGenerator: mg,
		TextSender:    ms,
		Transcriber:   mt,
		Command:       "/chat",
		CacheDuration: time.Second * 3,
		Tracker:       mtr,
	})

	assert.NotNil(t, chatHandler)

	err := chatHandler.Respond(t.Context(), time.Minute, &domain.Message{
		ChatID: 1, ID: 1, Username: "@unit", Text: "/chat prompt"})

	require.NoError(t, err)
	assert.Equal(t, "mock response", ms.Message)

	assert.Equal(t, 1, chatHandler.conversations.len())

	time.Sleep(time.Second * 2)

	err = chatHandler.Respond(t.Context(), time.Minute, &domain.Message{
		ChatID: 1, ID: 2, Username: "@unit", Text: "/chat prompt2"})
	require.NoError(t, err)

	assert.Equal(t, 1, chatHandler.conversations.len())

	time.Sleep(time.Second * 2)

	err = chatHandler.Respond(t.Context(), time.Minute, &domain.Message{
		ChatID: 1, ID: 2, Username: "@unit", Text: "/chat prompt3"})
	require.NoError(t, err)

	conversation, ok := chatHandler.conversations.get(1)
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
	mtr := &MockTracker{withinLimit: true}

	chatHandler, _ := NewChat(ChatParams{
		TextGenerator: mg,
		TextSender:    ms,
		Transcriber:   mt,
		Command:       "/chat",
		CacheDuration: time.Second * 3,
		Tracker:       mtr,
	})

	assert.NotNil(t, chatHandler)

	err := chatHandler.Respond(t.Context(), time.Minute, &domain.Message{ChatID: 1, ID: 1, Text: "/chat prompt"})
	require.Error(t, err)

	assert.Equal(t, "failed to generate response: mock error", ms.Message)
}

func TestEmptyPromptError(t *testing.T) {
	mg := &MockTextGenerator{err: errors.New("mock error")}
	ms := &MockTextSender{}
	mt := &MockTranscriber{}
	mtr := &MockTracker{withinLimit: true}

	chatHandler, _ := NewChat(ChatParams{
		TextGenerator: mg,
		TextSender:    ms,
		Transcriber:   mt,
		Command:       "/chat",
		CacheDuration: time.Second * 3,
		Tracker:       mtr,
	})

	assert.NotNil(t, chatHandler)

	err := chatHandler.Respond(t.Context(), time.Minute, &domain.Message{ChatID: 1, ID: 1, Text: "/chat"})
	require.Error(t, err)

	assert.Equal(t, "failed to extract prompt: empty prompt", ms.Message)
}

func TestSendMessageError(t *testing.T) {
	mg := &MockTextGenerator{response: "mock response"}
	ms := &MockTextSender{err: errors.New("mock error")}
	mt := &MockTranscriber{}
	mtr := &MockTracker{withinLimit: true}

	chatHandler, _ := NewChat(ChatParams{
		TextGenerator: mg,
		TextSender:    ms,
		Transcriber:   mt,
		Command:       "/chat",
		CacheDuration: time.Second * 3,
		Tracker:       mtr,
	})

	assert.NotNil(t, chatHandler)

	err := chatHandler.Respond(t.Context(), time.Minute, &domain.Message{ChatID: 1, ID: 1, Text: "/chat prompt"})

	assert.Equal(t, "mock response", ms.Message)
	require.Errorf(t, err, "failed to send reply")
}

func TestSendGenerateErrorAndMessageError(t *testing.T) {
	mg := &MockTextGenerator{err: errors.New("mock error")}
	ms := &MockTextSender{err: errors.New("mock error")}
	mt := &MockTranscriber{}
	mtr := &MockTracker{withinLimit: true}

	chatHandler, _ := NewChat(ChatParams{
		TextGenerator: mg,
		TextSender:    ms,
		Transcriber:   mt,
		Command:       "/chat",
		CacheDuration: time.Second * 3,
		Tracker:       mtr,
	})

	assert.NotNil(t, chatHandler)

	err := chatHandler.Respond(t.Context(), time.Minute, &domain.Message{ChatID: 1, ID: 1, Text: "/chat prompt"})

	assert.Equal(t, "failed to generate response: mock error", ms.Message)
	require.Errorf(t, err, "failed to send reply")
}

func TestNewChatSpokenRequiresAudio(t *testing.T) {
	_, err := NewChat(ChatParams{
		Command:      "/chat",
		SpeakCommand: "/speak",
	})
	require.Error(t, err)
}

func TestChatHandlerSpeakSuccess(t *testing.T) {
	mg := &MockTextGenerator{response: "mock response"}
	ms := &MockTextSender{}
	mt := &MockTranscriber{}
	mtr := &MockTracker{withinLimit: true}
	mag := &MockAudioGenerator{audio: []byte("mp3")}
	mas := &MockAudioSender{}

	chatHandler, err := NewChat(ChatParams{
		TextGenerator:  mg,
		TextSender:     ms,
		Transcriber:    mt,
		AudioGenerator: mag,
		AudioSender:    mas,
		Command:        "/chat",
		SpeakCommand:   "/speak",
		CacheDuration:  time.Second * 3,
		Tracker:        mtr,
	})
	require.NoError(t, err)

	err = chatHandler.Respond(t.Context(), time.Minute, &domain.Message{ChatID: 1, ID: 1, Text: "/speak prompt"})
	require.NoError(t, err)
	assert.Equal(t, []byte("mp3"), mas.file)
	assert.Empty(t, ms.Message)
}

func TestChatHandlerSpeakSharesContextWithChat(t *testing.T) {
	mg := &MockTextGenerator{response: "mock response"}
	ms := &MockTextSender{}
	mt := &MockTranscriber{}
	mtr := &MockTracker{withinLimit: true}
	mag := &MockAudioGenerator{audio: []byte("mp3")}
	mas := &MockAudioSender{}

	chatHandler, err := NewChat(ChatParams{
		TextGenerator:  mg,
		TextSender:     ms,
		Transcriber:    mt,
		AudioGenerator: mag,
		AudioSender:    mas,
		Command:        "/chat",
		SpeakCommand:   "/speak",
		CacheDuration:  time.Second * 3,
		Tracker:        mtr,
	})
	require.NoError(t, err)

	err = chatHandler.Respond(t.Context(), time.Minute, &domain.Message{
		ChatID: 1, ID: 1, Username: "@unit", Text: "/chat prompt"})
	require.NoError(t, err)

	err = chatHandler.Respond(t.Context(), time.Minute, &domain.Message{
		ChatID: 1, ID: 2, Username: "@unit", Text: "/speak prompt2"})
	require.NoError(t, err)

	conversation, ok := chatHandler.conversations.get(1)
	require.True(t, ok)
	assert.Len(t, conversation.messages, 4)
	assert.Equal(t, []byte("mp3"), mas.file)
}

func TestChatHandlerSpeakGeneratorError(t *testing.T) {
	mg := &MockTextGenerator{response: "mock response"}
	ms := &MockTextSender{}
	mt := &MockTranscriber{}
	mtr := &MockTracker{withinLimit: true}
	mag := &MockAudioGenerator{err: errors.New("tts fail")}
	mas := &MockAudioSender{}

	chatHandler, err := NewChat(ChatParams{
		TextGenerator:  mg,
		TextSender:     ms,
		Transcriber:    mt,
		AudioGenerator: mag,
		AudioSender:    mas,
		Command:        "/chat",
		SpeakCommand:   "/speak",
		CacheDuration:  time.Second * 3,
		Tracker:        mtr,
	})
	require.NoError(t, err)

	err = chatHandler.Respond(t.Context(), time.Minute, &domain.Message{ChatID: 1, ID: 1, Text: "/speak prompt"})
	require.Error(t, err)
	assert.Equal(t, "failed to generate speech: tts fail", ms.Message)
}

func TestChatHandlerAudioOnlyPrompt(t *testing.T) {
	mg := &MockTextGenerator{response: "mock response"}
	ms := &MockTextSender{}
	mt := &MockTranscriber{}
	mtr := &MockTracker{withinLimit: true}

	chatHandler, err := NewChat(ChatParams{
		TextGenerator: mg,
		TextSender:    ms,
		Transcriber:   mt,
		Command:       "/chat",
		CacheDuration: time.Second * 3,
		Tracker:       mtr,
	})
	require.NoError(t, err)

	err = chatHandler.Respond(t.Context(), time.Minute, &domain.Message{
		ChatID: 1, ID: 1, Username: "@unit", Text: "/chat", AudioURL: "foo"})
	require.NoError(t, err)

	conversation, ok := chatHandler.conversations.get(1)
	require.True(t, ok)
	assert.Equal(t, "@unit: foo", conversation.messages[0].Prompt)
}
