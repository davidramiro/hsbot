package commands

import (
	"context"
	"fmt"
	"hsbot/internal/core/domain"
	"hsbot/internal/core/port"
	"time"

	"github.com/rs/zerolog/log"
)

type EditHandler struct {
	imageGenerator port.ImageGenerator
	imageSender    port.ImageSender
	textSender     port.TextSender
	command        string
}

func NewEditHandler(imageGenerator port.ImageGenerator,
	imageSender port.ImageSender,
	textSender port.TextSender,
	command string) *EditHandler {
	return &EditHandler{imageGenerator: imageGenerator,
		imageSender: imageSender,
		textSender:  textSender,
		command:     command}
}

func (h *EditHandler) GetCommand() string {
	return h.command
}

func (h *EditHandler) Respond(ctx context.Context, timeout time.Duration, message *domain.Message) error {
	l := log.With().
		Int("messageId", message.ID).
		Int64("chatId", message.ChatID).
		Str("imageURL", message.ImageURL).
		Str("command", h.GetCommand()).
		Logger()

	l.Info().Msg("handling request")

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	go h.textSender.SendChatAction(ctx, message.ChatID, domain.SendingPhoto)

	prompt := domain.ParseCommandArgs(message.Text)
	if prompt == "" {
		_, err := h.textSender.SendMessageReply(ctx, message.ChatID, message.ID, "missing image prompt")
		if err != nil {
			l.Error().Err(err).Msg(domain.ErrSendingReplyFailed)
			return err
		}
		return nil
	}

	if message.ImageURL == "" {
		_, err := h.textSender.SendMessageReply(ctx, message.ChatID, message.ID, "missing image")
		if err != nil {
			l.Error().Err(err).Msg(domain.ErrSendingReplyFailed)
			return err
		}
		return nil
	}

	imageURL, err := h.imageGenerator.EditFromPrompt(ctx, domain.Prompt{Prompt: prompt, ImageURL: message.ImageURL})
	if err != nil {
		errMsg := "error getting FAL response"
		l.Error().Err(err).Str("imageURL", imageURL).Msg(errMsg)
		_, err := h.textSender.SendMessageReply(ctx,
			message.ChatID,
			message.ID,
			fmt.Sprintf("%s: %s", errMsg, err))
		if err != nil {
			l.Error().Err(err).Msg(domain.ErrSendingReplyFailed)
			return err
		}
		return nil
	}

	err = h.imageSender.SendImageURLReply(ctx, message.ChatID, message.ID, imageURL)
	if err != nil {
		l.Error().Err(err).Msg(domain.ErrSendingReplyFailed)
		return err
	}

	return nil
}
