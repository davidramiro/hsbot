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

type Transcribe struct {
	transcriber port.Transcriber
	textSender  port.TextSender
	command     string
}

func NewTranscribe(transcriber port.Transcriber, textSender port.TextSender, command string) *Transcribe {
	return &Transcribe{transcriber: transcriber, textSender: textSender, command: command}
}

func (h *Transcribe) GetCommand() string {
	return h.command
}

func (h *Transcribe) Respond(ctx context.Context, timeout time.Duration, message *domain.Message) error {
	l := log.With().
		Int("messageId", message.ID).
		Int64("chatId", message.ChatID).
		Str("audioURL", message.AudioURL).
		Str("command", h.GetCommand()).
		Logger()

	l.Info().Msg("handling request")

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	go h.textSender.SendChatAction(ctx, message.ChatID, domain.Typing)

	if message.AudioURL == "" {
		_ = h.textSender.NotifyAndReturnError(ctx, errors.New("reply to an audio"), message)
		return nil
	}

	resp, err := h.transcriber.GenerateFromAudio(ctx, message.AudioURL)
	if err != nil {
		return h.textSender.NotifyAndReturnError(ctx, fmt.Errorf("failed to generate audio: %w", err), message)
	}

	_, err = h.textSender.SendMessageReply(ctx, message, resp)
	if err != nil {
		err = fmt.Errorf("error sending transcript: %w", err)
		return h.textSender.NotifyAndReturnError(ctx, err, message)
	}

	return nil
}
