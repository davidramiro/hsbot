package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"hsbot/internal/core/domain"

	"github.com/stretchr/testify/assert"
)

type mockTextSender struct {
	sendCalled  bool
	callCount   int
	sendReplies []string
	sendError   error
}

func (m *mockTextSender) SendChatAction(_ context.Context, _ int64, _ domain.Action) {
	panic("implement me")
}

func (m *mockTextSender) NotifyAndReturnError(_ context.Context, _ error, _ *domain.Message) error {
	panic("implement me")
}

func (m *mockTextSender) SendMessageReply(_ context.Context, _ *domain.Message, text string) (int, error) {
	m.callCount++
	m.sendCalled = true
	m.sendReplies = append(m.sendReplies, text)
	if m.sendError != nil {
		return 1, m.sendError
	}
	return len(text), nil
}

func TestNewAuthorizer(t *testing.T) {
	auth := NewAuthorizer(&mockTextSender{}, []int64{1, 2, 3}, "adminuser")

	require.NotNil(t, auth)
	assert.Equal(t, []int64{1, 2, 3}, auth.allowlist)
	assert.Equal(t, "adminuser", auth.adminUsername)
}

func TestChatAuthorizer_IsAuthorized(t *testing.T) {
	adminUsername := "adminuser"

	tests := []struct {
		name         string
		allowlist    []int64
		chatID       int64
		sendErr      error
		want         bool
		expectSend   bool
		expectedText string
	}{
		{
			name:       "chatID is allowed",
			allowlist:  []int64{123, 456},
			chatID:     123,
			want:       true,
			expectSend: false,
		},
		{
			name:         "chatID not allowed sends message",
			allowlist:    []int64{111, 222},
			chatID:       333,
			expectSend:   true,
			want:         false,
			expectedText: "You are not authorized to use this bot. Please contact @adminuser with this ID to get access: 333",
		},
		{
			name:         "Send message fails for unauthorized chatID",
			allowlist:    []int64{999},
			chatID:       888,
			expectSend:   true,
			want:         false,
			sendErr:      errors.New("send failed"),
			expectedText: "You are not authorized to use this bot. Please contact @adminuser with this ID to get access: 888",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSender := &mockTextSender{sendError: tt.sendErr}
			a := &ChatAuthorizer{
				allowlist:     tt.allowlist,
				adminUsername: adminUsername,
				sender:        mockSender,
			}

			ctx := t.Context()
			got := a.IsAuthorized(ctx, tt.chatID)

			assert.Equal(t, tt.want, got)
			if tt.expectSend {
				assert.True(t, mockSender.sendCalled, "SendMessageReply should have been called")
				assert.NotEmpty(t, mockSender.sendReplies)
				assert.Equal(t, tt.expectedText, mockSender.sendReplies[0])
			} else {
				assert.False(t, mockSender.sendCalled, "SendMessageReply should not have been called")
			}
		})
	}
}
