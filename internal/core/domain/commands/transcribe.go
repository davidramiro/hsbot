package commands

import (
	"context"
	"fmt"
	"hsbot/internal/core/domain"
	"hsbot/internal/core/port"

	"github.com/rs/zerolog/log"
)

type TranscribeHandler struct {
	transcriber port.Transcriber
	textSender  port.TextSender
	command     string
}

func NewTranscribeHandler(transcriber port.Transcriber, textSender port.TextSender, command string) *TranscribeHandler {
	return &TranscribeHandler{transcriber: transcriber, textSender: textSender, command: command}
}

func (h *TranscribeHandler) GetCommand() string {
	return h.command
}

func (h *TranscribeHandler) Respond(ctx context.Context, message *domain.Message) {
	l := log.With().
		Int("messageId", message.ID).
		Int64("chatId", message.ChatID).
		Str("audioURL", message.AudioURL).
		Str("command", h.GetCommand()).
		Logger()

	l.Info().Msg("handling request")

	ctx, cancel := context.WithCancel(ctx)
	go h.textSender.SendChatAction(ctx, message.ChatID, domain.Typing)

	if message.AudioURL == "" {
		err := h.textSender.SendMessageReply(ctx, message.ChatID, message.ID, "no audio found for transcription")
		if err != nil {
			l.Error().Err(err).Msg(domain.ErrSendingReplyFailed)
			cancel()
			return
		}
		cancel()
		return
	}

	resp, err := h.transcriber.GenerateFromAudio(ctx, message.AudioURL)
	if err != nil {
		err := h.textSender.SendMessageReply(ctx, message.ChatID, message.ID, fmt.Sprintf("transcription failed: %s", err))
		if err != nil {
			l.Error().Err(err).Msg(domain.ErrSendingReplyFailed)
			cancel()
			return
		}
		cancel()
		return
	}

	err = h.textSender.SendMessageReply(ctx, message.ChatID, *message.ReplyToMessageID, resp)
	if err != nil {
		l.Error().Err(err).Msg(domain.ErrSendingReplyFailed)
		cancel()
		return
	}
	cancel()
}
