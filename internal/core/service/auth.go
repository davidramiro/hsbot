package service

import (
	"context"
	"errors"
	"fmt"
	"hsbot/internal/core/domain"
	"hsbot/internal/core/port"

	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

type Authorizer interface {
	IsAuthorized(ctx context.Context, chatID int64) bool
}

type ChatAuthorizer struct {
	allowlist []int64
	sender    port.TextSender
}

func NewAuthorizer(sender port.TextSender) (*ChatAuthorizer, error) {
	var list []int64

	err := viper.UnmarshalKey("telegram.allowed_chat_ids", &list)
	if err != nil {
		return nil, errors.New("failed to load allowed chat IDs")
	}

	return &ChatAuthorizer{
		allowlist: list,
		sender:    sender,
	}, nil
}

const forbidden = "You are not authorized to use this bot. Please contact @%s with this ID to get access: %d"

func (a *ChatAuthorizer) IsAuthorized(ctx context.Context, chatID int64) bool {
	for _, id := range a.allowlist {
		if id == chatID {
			return true
		}
	}

	_, err := a.sender.SendMessageReply(ctx,
		&domain.Message{ChatID: chatID},
		fmt.Sprintf(forbidden, viper.GetString("telegram.admin_username"), chatID))
	if err != nil {
		log.Err(err).Msg("failed to send unauthorized warning")
	}

	return false
}
