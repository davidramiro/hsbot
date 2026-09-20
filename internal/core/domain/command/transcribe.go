package command

import (
	"context"
	"fmt"
	"hsbot/internal/core/domain"
	"hsbot/internal/core/port"
	"time"
)

type Transcribe struct {
	transcriber port.Transcriber
	textSender  port.TextSender
	tracker     port.Tracker
	command     string
}

func NewTranscribe(
	transcriber port.Transcriber, textSender port.TextSender, tracker port.Tracker, command string) *Transcribe {
	return &Transcribe{transcriber: transcriber, textSender: textSender, tracker: tracker, command: command}
}

func (h *Transcribe) GetCommand() string {
	return h.command
}

func (h *Transcribe) Respond(ctx context.Context, timeout time.Duration, message *domain.Message) error {
	ctx, cancel, l := beginRespond(ctx, timeout, message, h.GetCommand())
	defer cancel()

	go h.textSender.SendChatAction(ctx, message.ChatID, domain.Typing)

	if !h.tracker.CheckLimit(ctx, message.ChatID) {
		l.Debug().Msg("spending limit reached")
		return nil
	}

	if message.AudioURL == "" {
		_ = h.textSender.NotifyAndReturnError(ctx, domain.ErrMissingAudio, message)
		return nil
	}

	resp, err := h.transcriber.Transcribe(ctx, message.AudioURL)
	if err != nil {
		return h.textSender.NotifyAndReturnError(ctx, fmt.Errorf("failed to generate audio: %w", err), message)
	}

	h.tracker.AddCost(message.ChatID, resp.Metadata.Cost)

	_, err = h.textSender.SendMessageReply(ctx, message, resp.Response)
	if err != nil {
		err = fmt.Errorf("error sending transcript: %w", err)
		return h.textSender.NotifyAndReturnError(ctx, err, message)
	}

	return nil
}
