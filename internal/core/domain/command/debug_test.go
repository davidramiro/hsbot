package command

import (
	"context"
	"hsbot/internal/core/domain"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type MockSender struct {
	mock.Mock
}

func (m *MockSender) SendChatAction(_ context.Context, _ int64, _ domain.Action) {
	// mocked
}

func (m *MockSender) NotifyAndReturnError(_ context.Context, _ error, _ *domain.Message) error {
	// mocked
	return nil
}

func (m *MockSender) SendMessageReply(ctx context.Context, message *domain.Message, text string) (int, error) {
	args := m.Called(ctx, message, text)
	return args.Int(0), args.Error(1)
}

func TestDebug_Respond_SendsDebugInfo(t *testing.T) {
	mockSender := new(MockSender)
	cmd := "debug"
	debugCmd := NewDebug(mockSender, cmd)

	msg := &domain.Message{ID: 123, ChatID: 456}

	mockSender.
		On(
			"SendMessageReply",
			mock.Anything,
			msg,
			mock.MatchedBy(func(text string) bool {
				return strings.Contains(text, "allocated mem:") &&
					strings.Contains(text, "threads running:") &&
					strings.Contains(text, "heap:") &&
					strings.Contains(text, "stack:") &&
					strings.Contains(text, "compiled with")
			}),
		).
		Return(1, nil)

	err := debugCmd.Respond(t.Context(), time.Second, msg)
	require.NoError(t, err)
	mockSender.AssertExpectations(t)
}
