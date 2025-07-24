package service

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestAddCost(t *testing.T) {
	tracker := &UsageTracker{
		chats: make(map[int64]float64),
		mutex: &sync.Mutex{},
	}
	tests := []struct {
		name        string
		chatID      int64
		initialCost float64
		addCost     float64
		wantTotal   float64
	}{
		{
			name:        "Add first cost",
			chatID:      1,
			initialCost: 0,
			addCost:     2.50,
			wantTotal:   2.50,
		},
		{
			name:        "Add to existing cost",
			chatID:      2,
			initialCost: 1.00,
			addCost:     3.00,
			wantTotal:   4.00,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tracker.chats[tt.chatID] = tt.initialCost
			tracker.AddCost(tt.chatID, tt.addCost)
			assert.Equal(t, tt.wantTotal, tracker.chats[tt.chatID])
		})
	}
}

func TestCheckLimit(t *testing.T) {
	dailyLimit := 5.00
	viper.Set("telegram.daily_spend_limit", dailyLimit)
	tests := []struct {
		name          string
		chatID        int64
		spent         float64
		expectAllowed bool
		expectMessage bool
		simulateErr   error
	}{
		{
			name:          "Below limit",
			chatID:        1,
			spent:         4.99,
			expectAllowed: true,
		},
		{
			name:          "At limit",
			chatID:        2,
			spent:         5.00,
			expectAllowed: true,
		},
		{
			name:          "Above limit and message sent",
			chatID:        3,
			spent:         5.01,
			expectAllowed: false,
			expectMessage: true,
		},
		{
			name:          "Above limit with send error",
			chatID:        4,
			spent:         7.00,
			expectAllowed: false,
			expectMessage: true,
			simulateErr:   assert.AnError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSender := &mockTextSender{sendError: tt.simulateErr}
			tracker := &UsageTracker{
				chats:      map[int64]float64{tt.chatID: tt.spent},
				mutex:      &sync.Mutex{},
				dailyLimit: dailyLimit,
				sender:     mockSender,
			}
			ctx := context.Background()
			result := tracker.CheckLimit(ctx, tt.chatID)
			assert.Equal(t, tt.expectAllowed, result)
			if tt.expectMessage {
				assert.Equal(t, 1, mockSender.callCount)
				assert.NotEmpty(t, mockSender.sendReplies[0])
				expectedText := fmt.Sprintf(overLimit,
					tracker.dailyLimit, time.Until(getNextResetTime()).Truncate(time.Second))

				assert.Equal(t, expectedText, mockSender.sendReplies[0])
			} else {
				assert.Equal(t, 0, mockSender.callCount)
			}
		})
	}
}

func TestNewUsageTracker(t *testing.T) {
	dailyLimit := 10.00
	viper.Set("telegram.daily_spend_limit", dailyLimit)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mockSender := &mockTextSender{}
	tracker := NewUsageTracker(ctx, mockSender)

	assert.NotNil(t, tracker.chats)
	assert.Equal(t, dailyLimit, tracker.dailyLimit)
	assert.Equal(t, mockSender, tracker.sender)
}

func TestGetNextResetTime(t *testing.T) {
	now := time.Now()
	reset := getNextResetTime()
	assert.Equal(t, 0, reset.Hour())
	assert.Equal(t, 0, reset.Minute())
	assert.Equal(t, 0, reset.Second())
	assert.Equal(t, now.Day()+1, reset.Day())
}
