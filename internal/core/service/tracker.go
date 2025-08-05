package service

import (
	"context"
	"fmt"
	"hsbot/internal/core/domain"
	"hsbot/internal/core/port"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

type Tracker interface {
	AddCost(chatID int64, cost float64)
	CheckLimit(ctx context.Context, chatID int64) bool
	GetSpent(chatID int64) float64
}

type UsageTracker struct {
	chats      map[int64]float64
	dailyLimit float64
	mutex      sync.Mutex
	sender     port.TextSender
}

func NewUsageTracker(ctx context.Context, sender port.TextSender) *UsageTracker {
	ut := &UsageTracker{
		chats:      make(map[int64]float64),
		sender:     sender,
		dailyLimit: viper.GetFloat64("telegram.daily_spend_limit"),
	}

	go ut.ResetDailyLimit(ctx)

	return ut
}

func (t *UsageTracker) AddCost(chatID int64, cost float64) {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	t.chats[chatID] += cost
}

const overLimit = "You have exceeded your daily spending limit: $%.2f. Limit will reset in %s."

func (t *UsageTracker) CheckLimit(ctx context.Context, chatID int64) bool {
	if t.chats[chatID] > t.dailyLimit {
		_, err := t.sender.SendMessageReply(ctx,
			&domain.Message{ChatID: chatID},
			fmt.Sprintf(overLimit, t.dailyLimit, time.Until(getNextResetTime()).Truncate(time.Second)))
		if err != nil {
			log.Warn().Err(err).Msg("failed to send daily limit exceeded warning")
		}
		return false
	}

	return true
}

func (t *UsageTracker) ResetDailyLimit(ctx context.Context) {
	reset := getNextResetTime()

	for {
		log.Debug().Time("reset", reset).Msg("running reset timer")
		select {
		case <-time.After(time.Until(reset)):
			log.Debug().Msg("resetting daily limit")
			t.mutex.Lock()
			t.chats = make(map[int64]float64)
			t.mutex.Unlock()
			time.Sleep(time.Second)
			reset = getNextResetTime()
		case <-ctx.Done():
			log.Debug().Msg("stopping daily limit reset")
			return
		}
	}
}

const hoursPerDay = 24

func getNextResetTime() time.Time {
	return time.Now().UTC().AddDate(0, 0, 1).Truncate(time.Hour * hoursPerDay)
}

func (t *UsageTracker) GetSpent(chatID int64) float64 {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	return t.chats[chatID]
}
