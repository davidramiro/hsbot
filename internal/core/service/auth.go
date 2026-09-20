package service

import (
	"context"
	"fmt"
	"hsbot/internal/core/domain"
	"hsbot/internal/core/port"
	"slices"

	"github.com/rs/zerolog/log"
)

type ChatAuthorizer struct {
	allowlist     []int64
	adminUsername string
	sender        port.TextSender
}

func NewAuthorizer(sender port.TextSender, allowlist []int64, adminUsername string) *ChatAuthorizer {
	return &ChatAuthorizer{
		allowlist:     allowlist,
		adminUsername: adminUsername,
		sender:        sender,
	}
}

const forbidden = "You are not authorized to use this bot. Please contact @%s with this ID to get access: %d"

func (a *ChatAuthorizer) IsAuthorized(ctx context.Context, chatID int64) bool {
	if slices.Contains(a.allowlist, chatID) {
		return true
	}

	_, err := a.sender.SendMessageReply(ctx,
		&domain.Message{ChatID: chatID},
		fmt.Sprintf(forbidden, a.adminUsername, chatID))
	if err != nil {
		log.Err(err).Msg("failed to send unauthorized warning")
	}

	return false
}
