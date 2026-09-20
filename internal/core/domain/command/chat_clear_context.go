package command

import (
	"context"
	"fmt"
	"hsbot/internal/core/domain"
	"hsbot/internal/core/port"
	"time"
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

func (c *ChatClearContext) Respond(ctx context.Context, timeout time.Duration, message *domain.Message) error {
	ctx, cancel, l := beginRespond(ctx, timeout, message, c.GetCommand())
	defer cancel()

	conversation, ok := c.chat.Clear(message.ChatID)
	if !ok {
		l.Debug().Msg("no conversation in cache")

		_, err := c.textSender.SendMessageReply(ctx, message, "no conversation context")
		if err != nil {
			err = fmt.Errorf("error sending cache clearing response: %w", err)
			return c.textSender.NotifyAndReturnError(ctx, err, message)
		}

		return nil
	}

	size := len(conversation.messages)

	var plural string
	if size != 1 {
		plural = "s"
	}

	l.Debug().Msg("cleared conversation cache")

	_, err := c.textSender.SendMessageReply(ctx, message,
		fmt.Sprintf("cleared conversation context with %d message%s", size, plural))
	if err != nil {
		err = fmt.Errorf("error sending cache clearing response: %w", err)
		return c.textSender.NotifyAndReturnError(ctx, err, message)
	}

	return nil
}
