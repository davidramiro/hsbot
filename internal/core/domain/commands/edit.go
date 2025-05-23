package commands

import (
	"context"
	"errors"
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
		_ = h.textSender.NotifyAndReturnError(ctx, errors.New("empty prompt"), message)
		return nil
	}

	if message.ImageURL == "" {
		_ = h.textSender.NotifyAndReturnError(ctx, errors.New("missing image"), message)
		return nil
	}

	imageURL, err := h.imageGenerator.EditFromPrompt(ctx, domain.Prompt{Prompt: prompt, ImageURL: message.ImageURL})
	if err != nil {
		err = fmt.Errorf("error creating edited image: %w", err)
		return h.textSender.NotifyAndReturnError(ctx, err, message)
	}

	err = h.imageSender.SendImageURLReply(ctx, message, imageURL)
	if err != nil {
		err = fmt.Errorf("error sending edited image: %w", err)
		return h.textSender.NotifyAndReturnError(ctx, err, message)
	}

	return nil
}
