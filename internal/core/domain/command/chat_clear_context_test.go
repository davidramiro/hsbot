package command

import (
	"context"
	"errors"
	"hsbot/internal/core/domain"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/stretchr/testify/assert"
)

type mockTextSender struct {
	replyCalls     []string
	notifyErrCalls []error
	replyErr       error
	notifyErr      error
}

func (m *mockTextSender) SendChatAction(_ context.Context, _ int64, _ domain.Action) {
}

func (m *mockTextSender) SendMessageReply(_ context.Context, _ *domain.Message, text string) (int, error) {
	m.replyCalls = append(m.replyCalls, text)
	return 0, m.replyErr
}

func (m *mockTextSender) NotifyAndReturnError(_ context.Context, err error, _ *domain.Message) error {
	m.notifyErrCalls = append(m.notifyErrCalls, err)
	return m.notifyErr
}

func TestChatClearContext_Respond_ClearsCacheAndReplies(t *testing.T) {
	chat := &Chat{conversations: newConversationStore(time.Minute)}

	chatID := int64(101)
	conversation := chat.conversations.getOrCreate(chatID)
	conversation.messages = []domain.Prompt{
		{
			Prompt: "mock message",
			Author: domain.System,
		},
		{
			Prompt: "mock message 2",
			Author: domain.User,
		},
	}

	msg := &domain.Message{ID: 1, ChatID: chatID}
	sender := &mockTextSender{}

	cc := NewChatClearContext(chat, sender, "/clear")

	err := cc.Respond(t.Context(), time.Second, msg)

	require.NoError(t, err)

	_, ok := chat.conversations.get(chatID)
	assert.False(t, ok, "Conversation should be deleted from cache")
	assert.Equal(t, "cleared conversation context with 2 messages", sender.replyCalls[0])
}

func TestChatClearContext_Respond_NoConversationInCache(t *testing.T) {
	chat := &Chat{conversations: newConversationStore(time.Minute)}

	chatID := int64(202)
	msg := &domain.Message{ID: 2, ChatID: chatID}
	sender := &mockTextSender{}

	cc := NewChatClearContext(chat, sender, "/clear")

	err := cc.Respond(t.Context(), time.Second, msg)

	require.NoError(t, err)
	assert.Contains(t, sender.replyCalls[0], "no conversation context")
}

func TestChatClearContext_Respond_SendReplyFails(t *testing.T) {
	chat := &Chat{conversations: newConversationStore(time.Minute)}

	chatID := int64(303)
	conversation := chat.conversations.getOrCreate(chatID)
	conversation.messages = []domain.Prompt{}

	msg := &domain.Message{ID: 3, ChatID: chatID}
	sender := &mockTextSender{
		replyErr:  errors.New("send failure"),
		notifyErr: errors.New("notify failure"),
	}

	cc := NewChatClearContext(chat, sender, "/clear")

	err := cc.Respond(t.Context(), time.Second, msg)

	require.Error(t, err)
	assert.Len(t, sender.notifyErrCalls, 1)
}

func TestChatClearContext_Respond_NoConvoReplyFails(t *testing.T) {
	chat := &Chat{conversations: newConversationStore(time.Minute)}

	msg := &domain.Message{ID: 3, ChatID: 101}
	sender := &mockTextSender{
		replyErr:  errors.New("send failure"),
		notifyErr: errors.New("notify failure"),
	}

	cc := NewChatClearContext(chat, sender, "/clear")

	err := cc.Respond(t.Context(), time.Second, msg)

	require.Error(t, err)
	assert.Len(t, sender.notifyErrCalls, 1)
}
