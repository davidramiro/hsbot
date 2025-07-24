package command

import (
	"context"
	"fmt"
	"hsbot/internal/core/domain"
	"hsbot/internal/core/port"
	"hsbot/internal/core/service"
	"time"
)

type Spent struct {
	tracker service.Tracker
	sender  port.TextSender
	command string
}

func NewSpent(tracker service.Tracker, ts port.TextSender, command string) *Spent {
	return &Spent{
		tracker: tracker,
		sender:  ts,
		command: command,
	}
}

func (s *Spent) GetCommand() string {
	return s.command
}

const spentMessage = "Spent today within ChatID %d: $%.2f."

func (s *Spent) Respond(ctx context.Context, _ time.Duration, message *domain.Message) error {
	_, err := s.sender.SendMessageReply(ctx, message, fmt.Sprintf(spentMessage,
		message.ChatID, s.tracker.GetSpent(message.ChatID)))
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	return nil
}
