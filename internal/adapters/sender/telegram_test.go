package sender

import (
	"context"
	"errors"
	"github.com/PaulSonOfLars/gotgbot/v2"
	"hsbot/internal/core/domain"
	"testing"
	"time"

	"github.com/go-telegram/bot/models"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type MockBot struct {
	mock.Mock
}

func (m *MockBot) SendMessageWithContext(ctx context.Context, chatId int64, text string, opts *gotgbot.SendMessageOpts) (*gotgbot.Message, error) {
	args := m.Called(ctx, chatId, text, opts)
	msg, _ := args.Get(0).(*gotgbot.Message)
	return msg, args.Error(1)
}
func (m *MockBot) SendPhotoWithContext(ctx context.Context, chatId int64, photo gotgbot.InputFileOrString, opts *gotgbot.SendPhotoOpts) (*gotgbot.Message, error) {
	args := m.Called(ctx, chatId, photo, opts)
	msg, _ := args.Get(0).(*gotgbot.Message)
	return msg, args.Error(1)
}
func (m *MockBot) SendChatActionWithContext(ctx context.Context, chatId int64, action string, opts *gotgbot.SendChatActionOpts) (bool, error) {
	args := m.Called(ctx, chatId, action, opts)
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
				mb.On("SendMessageWithContext", mock.Anything, mock.MatchedBy(func(param string) bool {
					return param == "hello"
				})).
					Return(&gotgbot.Message{MessageId: 123}, nil).
					Once()
			},
			wantErr: false,
		},
		{
			name:      "message chunked in two",
			text:      longText,
			wantCalls: 2,
			setupMock: func(mb *MockBot) {
				mb.On("SendMessageWithContext", mock.Anything, mock.MatchedBy(func(param string) bool {
					return len(param) <= TelegramMessageLimit
				})).
					Return(&gotgbot.Message{MessageId: 456}, nil).
					Twice()
			},
			wantErr: false,
		},
		{
			name:      "send fails on first",
			text:      "fail",
			wantCalls: 1,
			setupMock: func(mb *MockBot) {
				mb.On("SendMessageWithContext", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(nil, errors.New("fail")).Once()
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
			mb.On("SendPhotoWithContext", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
				Return(&gotgbot.Message{}, tc.retErr).Once()

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
			mb.On("SendMessageWithContext", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
				Return(&gotgbot.Message{MessageId: 101}, tc.sendMsgRetErr)

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

	// Use a channel to track calls deterministically
	callCh := make(chan struct{}, 10)
	mb.On("SendChatAction", mock.Anything, mock.Anything).Twice().Return(true, nil).
		Run(func(_ mock.Arguments) {
			callCh <- struct{}{}
		})

	go sender.SendChatAction(ctx, chatID, action)

	// Wait for a few calls, then cancel
	for range 2 {
		select {
		case <-callCh:
			// expected
		case <-time.After(time.Second*ChatActionRepeatSeconds + time.Millisecond*200):
			t.Fatal("timed out waiting for SendChatAction call")
		}
	}
	cancel()

	time.Sleep(20 * time.Millisecond)
	remaining := len(callCh)
	if remaining != 0 {
		t.Errorf("SendChatAction called after cancel: %d extra calls", remaining)
	}
	mb.AssertCalled(t, "SendChatAction", mock.Anything, mock.Anything)
}
