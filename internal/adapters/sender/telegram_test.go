package sender

import (
	"context"
	"errors"
	"hsbot/internal/core/domain"
	"testing"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type MockBot struct {
	mock.Mock
}

func (m *MockBot) SendMessage(ctx context.Context, params *bot.SendMessageParams) (*models.Message, error) {
	args := m.Called(ctx, params)
	msg, _ := args.Get(0).(*models.Message)
	return msg, args.Error(1)
}
func (m *MockBot) SendPhoto(ctx context.Context, params *bot.SendPhotoParams) (*models.Message, error) {
	args := m.Called(ctx, params)
	msg, _ := args.Get(0).(*models.Message)
	return msg, args.Error(1)
}
func (m *MockBot) SendChatAction(ctx context.Context, params *bot.SendChatActionParams) (bool, error) {
	args := m.Called(ctx, params)
	return args.Bool(0), args.Error(1)
}

func TestTelegramSender_SendMessageReply(t *testing.T) {
	longText := ""
	for range TelegramMessageLimit + 10 {
		longText += "x"
	}

	tests := []struct {
		name      string
		text      string
		wantCalls int
		setupMock func(mb *MockBot)
		wantErr   bool
	}{
		{
			name:      "single message",
			text:      "hello",
			wantCalls: 1,
			setupMock: func(mb *MockBot) {
				mb.On("SendMessage", mock.Anything, mock.MatchedBy(func(params *bot.SendMessageParams) bool {
					return params.Text == "hello"
				})).
					Return(&models.Message{ID: 123}, nil).
					Once()
			},
			wantErr: false,
		},
		{
			name:      "message chunked in two",
			text:      longText,
			wantCalls: 2,
			setupMock: func(mb *MockBot) {
				mb.On("SendMessage", mock.Anything, mock.MatchedBy(func(params *bot.SendMessageParams) bool {
					return len(params.Text) <= TelegramMessageLimit
				})).
					Return(&models.Message{ID: 456}, nil).
					Twice()
			},
			wantErr: false,
		},
		{
			name:      "send fails on first",
			text:      "fail",
			wantCalls: 1,
			setupMock: func(mb *MockBot) {
				mb.On("SendMessage", mock.Anything, mock.Anything).Return(nil, errors.New("fail")).Once()
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mb := new(MockBot)
			sender := NewTelegram(mb)

			msg := &domain.Message{
				ID:     42,
				ChatID: 1001,
			}

			tc.setupMock(mb)
			_, err := sender.SendMessageReply(t.Context(), msg, tc.text)

			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			mb.AssertNumberOfCalls(t, "SendMessage", tc.wantCalls)
			mb.AssertExpectations(t)
		})
	}
}

func TestTelegramSender_SendImageURLReply(t *testing.T) {
	tests := []struct {
		name    string
		retErr  error
		wantErr bool
	}{
		{
			name:    "success",
			retErr:  nil,
			wantErr: false,
		},
		{
			name:    "send fails",
			retErr:  errors.New("fail"),
			wantErr: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mb := new(MockBot)
			sender := NewTelegram(mb)

			msg := &domain.Message{ID: 10, ChatID: 20}
			mb.On("SendPhoto", mock.Anything, mock.Anything).
				Return(&models.Message{}, tc.retErr).Once()

			err := sender.SendImageURLReply(t.Context(), msg, "http://image.url/a.png")

			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			mb.AssertExpectations(t)
		})
	}
}

func TestTelegramSender_SendImageFileReply(t *testing.T) {
	tests := []struct {
		name    string
		file    []byte
		retErr  error
		wantErr bool
	}{
		{
			name:    "success",
			file:    []byte("pngdata"),
			retErr:  nil,
			wantErr: false,
		},
		{
			name:    "fail send",
			file:    []byte("fake"),
			retErr:  errors.New("fail"),
			wantErr: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mb := new(MockBot)
			sender := NewTelegram(mb)

			msg := &domain.Message{ID: 33, ChatID: 44}
			mb.On("SendPhoto", mock.Anything, mock.Anything).
				Return(&models.Message{}, tc.retErr).Once()

			err := sender.SendImageFileReply(t.Context(), msg, tc.file)

			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			mb.AssertExpectations(t)
		})
	}
}

func TestTelegramSender_NotifyAndReturnError(t *testing.T) {
	tests := []struct {
		name            string
		sendMsgRetErr   error
		originalErr     error
		wantOriginalErr bool
	}{
		{
			name:            "send ok",
			sendMsgRetErr:   nil,
			originalErr:     errors.New("original"),
			wantOriginalErr: true,
		},
		{
			name:            "send fails",
			sendMsgRetErr:   errors.New("sendfail"),
			originalErr:     errors.New("original"),
			wantOriginalErr: false,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mb := new(MockBot)
			sender := NewTelegram(mb)

			msg := &domain.Message{ID: 55, ChatID: 88}
			mb.On("SendMessage", mock.Anything, mock.Anything).
				Return(&models.Message{ID: 101}, tc.sendMsgRetErr)

			err := sender.NotifyAndReturnError(t.Context(), tc.originalErr, msg)

			if tc.wantOriginalErr {
				require.Error(t, tc.originalErr)
			} else {
				require.Error(t, err)
			}
			mb.AssertExpectations(t)
		})
	}
}

func TestSendChatAction_RepeatsAndStopsOnContextCancel(t *testing.T) {
	mb := new(MockBot)
	sender := NewTelegram(mb)

	ctx, cancel := context.WithCancel(t.Context())
	chatID := int64(12345)
	action := domain.Typing

	mb.On("SendChatAction", mock.Anything, &bot.SendChatActionParams{
		ChatID: chatID,
		Action: models.ChatAction(domain.Typing),
	}).Return(true, nil).Times(2)

	go func() {
		sender.SendChatAction(ctx, chatID, action)
	}()

	// Wait to let it tick at least 2 times
	time.Sleep(2 * ChatActionRepeatSeconds * time.Second)
	cancel() // stop goroutine

	// Give time for goroutine to exit
	time.Sleep(20 * time.Millisecond)

	count := len(mb.Calls)
	if count < 2 {
		t.Errorf("expected at least 3 chat actions sent, got %d", count)
	}
}
