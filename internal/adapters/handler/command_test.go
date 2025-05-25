package handler

import (
	"context"
	"errors"
	"hsbot/internal/core/port"
	"testing"
	"time"

	"hsbot/internal/core/domain"

	"github.com/go-telegram/bot/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockRegistry struct {
	mock.Mock
	cmd port.Command
}

func (m *MockRegistry) Get(cmd string) (port.Command, error) {
	args := m.Called(cmd)
	return m.cmd, args.Error(1)
}

func (m *MockRegistry) Register(handler port.Command) {
	m.cmd = handler
	m.Called(handler)
}

func (m *MockRegistry) ListCommands() []string {
	m.Called()
	return []string{"foo", "bar"}
}

type MockCmdHandler struct{ mock.Mock }

func (m *MockCmdHandler) Respond(ctx context.Context, timeout time.Duration, msg *domain.Message) error {
	args := m.Called(ctx, timeout, msg)
	return args.Error(0)
}

func (m *MockCmdHandler) GetCommand() string {
	m.Called()
	return ""
}

func makeUpdate(txt string) *models.Update {
	return &models.Update{
		Message: &models.Message{
			ID:   1,
			Text: txt,
			Chat: models.Chat{ID: 100},
			From: &models.User{ID: 200, Username: "bob", FirstName: "bob"},
		},
	}
}

func TestCommandHandler_Handle(t *testing.T) {
	type testcase struct {
		name       string
		update     *models.Update
		mockSetup  func(r *MockRegistry, ch *MockCmdHandler)
		wantCalled bool
		wantMsg    *domain.Message
	}

	tests := []testcase{
		{
			name:   "no message in update",
			update: &models.Update{},
			mockSetup: func(_ *MockRegistry, _ *MockCmdHandler) {
				// No call
			},
			wantCalled: false,
			wantMsg:    nil,
		},
		{
			name:   "unknown command",
			update: makeUpdate("/unknown"),
			mockSetup: func(r *MockRegistry, _ *MockCmdHandler) {
				r.On("Get", "/unknown").Return(nil, errors.New("no handler"))
			},
			wantCalled: false,
			wantMsg:    nil,
		},
		{
			name:   "known command, Respond called successfully",
			update: makeUpdate("/hello"),
			mockSetup: func(r *MockRegistry, ch *MockCmdHandler) {
				r.On("Get", "/hello").Return(ch, nil)
				ch.On("Respond", mock.Anything, mock.Anything,
					mock.AnythingOfType("*domain.Message")).Return(nil)
			},
			wantCalled: true,
			wantMsg: &domain.Message{
				ID:               1,
				ChatID:           100,
				Username:         "@bob",
				ReplyToMessageID: new(int),
				ReplyToUsername:  "",
				IsReplyToBot:     false,
				QuotedText:       "",
				ImageURL:         "",
				AudioURL:         "",
				Text:             "/hello",
			},
		},
		{
			name:   "known command, Respond returns error",
			update: makeUpdate("/fail"),
			mockSetup: func(r *MockRegistry, ch *MockCmdHandler) {
				r.On("Get", "/fail").Return(ch, nil)
				ch.On("Respond", mock.Anything, mock.Anything,
					mock.AnythingOfType("*domain.Message")).Return(errors.New("fail"))
			},
			wantCalled: true,
			wantMsg:    nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			reg := new(MockRegistry)
			handler := new(MockCmdHandler)
			reg.cmd = handler
			// Prepare mocks for this test case
			tc.mockSetup(reg, handler)

			ch := NewCommand(reg, 3*time.Second)
			ch.Handle(t.Context(), nil, tc.update)

			// as the Respond() call is a goroutine, wait for finish
			time.Sleep(100 * time.Millisecond)

			reg.AssertExpectations(t)
			if tc.wantCalled {
				if tc.wantMsg != nil {
					handler.AssertCalled(t, "Respond",
						mock.Anything,
						mock.Anything,
						mock.MatchedBy(func(msg *domain.Message) bool {
							assert.Equal(t, tc.wantMsg, msg)
							return assert.ObjectsAreEqual(tc.wantMsg, msg)
						}),
					)
				} else {
					handler.AssertCalled(t, "Respond",
						mock.Anything,
						mock.Anything,
						mock.AnythingOfType("*domain.Message"),
					)
				}
			} else {
				assert.Empty(t, handler.Calls)
			}
		})
	}
}

func Test_findMediumSizedImage(t *testing.T) {
	tests := []struct {
		name   string
		photos []models.PhotoSize
		want   string
	}{
		{
			name: "returns medium-sized photo if present",
			photos: []models.PhotoSize{
				{FileID: "id1", FileSize: 12000},
				{FileID: "id2", FileSize: 100000}, // medium
				{FileID: "id3", FileSize: 150000},
			},
			want: "id2",
		},
		{
			name: "falls back to last photo if none matched",
			photos: []models.PhotoSize{
				{FileID: "only", FileSize: 10},
				{FileID: "last", FileSize: 20},
			},
			want: "last",
		},
		{
			name: "single photo fallback",
			photos: []models.PhotoSize{
				{FileID: "only", FileSize: 10},
			},
			want: "only",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := findMediumSizedImage(tc.photos)
			assert.Equal(t, tc.want, got)
		})
	}
}

func Test_getUserNameFromMessage(t *testing.T) {
	tests := []struct {
		name     string
		user     *models.User
		expected string
	}{
		{
			name:     "username present",
			user:     &models.User{Username: "alice", FirstName: "Alice"},
			expected: "@alice",
		},
		{
			name:     "empty username, fallback to first name",
			user:     &models.User{Username: "", FirstName: "Bob"},
			expected: "Bob",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, getUserNameFromMessage(tc.user))
		})
	}
}
