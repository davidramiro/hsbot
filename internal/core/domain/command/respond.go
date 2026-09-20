package command

import (
	"context"
	"hsbot/internal/core/domain"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func beginRespond(
	ctx context.Context,
	timeout time.Duration,
	message *domain.Message,
	command string,
) (context.Context, context.CancelFunc, zerolog.Logger) {
	l := log.With().
		Int("messageId", message.ID).
		Int64("chatId", message.ChatID).
		Str("command", command).
		Logger()
	l.Info().Msg("handling request")

	ctx, cancel := context.WithTimeout(ctx, timeout)
	return ctx, cancel, l
}
