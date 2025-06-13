package command

import (
	"context"
	"errors"
	"fmt"
	"hsbot/internal/core/domain"
	"hsbot/internal/core/port"
	"time"

	"github.com/rs/zerolog/log"
)

type ChatClearContext struct {
	chat       *Chat
	textSender port.TextSender
	command    string
}

func NewChatClearContext(chat *Chat, sender port.TextSender, command string) *ChatClearContext {
	return &ChatClearContext{chat: chat, textSender: sender, command: command}
}

func (c *ChatClearContext) GetCommand() string {
	return c.command
}

func (c *ChatClearContext) Respond(ctx context.Context, _ time.Duration, message *domain.Message) error {
	l := log.With().
		Int("messageId", message.ID).
		Int64("chatId", message.ChatID).
		Str("command", c.GetCommand()).
		Logger()

	l.Info().Msg("handling request")

	conv, ok := c.chat.cache.Load(message.ChatID)
	if !ok {
		l.Debug().Msg("no conversation in cache")

		_, err := c.textSender.SendMessageReply(ctx, message, "no conversation context")
		if err != nil {
			err = fmt.Errorf("error sending cache clearing response: %w", err)
			return c.textSender.NotifyAndReturnError(ctx, err, message)
		}

		return nil
	}

	conversation, ok := conv.(*Conversation)
	if !ok {
		return errors.New("conversation type error")
	}

	size := len(conversation.messages)

	var plural string
	if size != 1 {
		plural = "s"
	}

	c.chat.cache.Delete(message.ChatID)
	conversation.exitSignal <- struct{}{}

	l.Debug().Msg("cleared conversation cache")

	_, err := c.textSender.SendMessageReply(ctx, message,
		fmt.Sprintf("cleared conversation context with %d message%s", size, plural))
	if err != nil {
		err = fmt.Errorf("error sending cache clearing response: %w", err)
		return c.textSender.NotifyAndReturnError(ctx, err, message)
	}

	return nil
}
