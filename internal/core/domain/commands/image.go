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

type ImageHandler struct {
	imageGenerator port.ImageGenerator
	imageSender    port.ImageSender
	textSender     port.TextSender
	command        string
}

func NewImageHandler(imageGenerator port.ImageGenerator,
	imageSender port.ImageSender,
	textSender port.TextSender,
	command string) *ImageHandler {
	return &ImageHandler{imageGenerator: imageGenerator,
		imageSender: imageSender,
		textSender:  textSender,
		command:     command}
}

func (h *ImageHandler) GetCommand() string {
	return h.command
}

func (h *ImageHandler) Respond(ctx context.Context, timeout time.Duration, message *domain.Message) error {
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
		_ = h.textSender.NotifyAndReturnError(ctx, errors.New("missing image prompt"), message)
		return nil
	}

	imageURL, err := h.imageGenerator.GenerateFromPrompt(ctx, prompt)
	if err != nil {
		err = fmt.Errorf("error generating image: %w", err)
		return h.textSender.NotifyAndReturnError(ctx, err, message)
	}

	err = h.imageSender.SendImageURLReply(ctx, message, imageURL)
	if err != nil {
		err = fmt.Errorf("error sending edited image: %w", err)
		return h.textSender.NotifyAndReturnError(ctx, err, message)
	}

	return nil
}
