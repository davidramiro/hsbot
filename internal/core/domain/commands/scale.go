package commands

import (
	"context"
	"errors"
	"fmt"
	"hsbot/internal/core/domain"
	"hsbot/internal/core/port"
	"strconv"
	"time"

	"github.com/rs/zerolog/log"
)

type ScaleHandler struct {
	imageConverter port.ImageConverter
	textSender     port.TextSender
	imageSender    port.ImageSender
	command        string
}

func NewScaleHandler(imageConverter port.ImageConverter, textSender port.TextSender, imageSender port.ImageSender,
	command string) *ScaleHandler {
	return &ScaleHandler{imageConverter: imageConverter, textSender: textSender, imageSender: imageSender,
		command: command}
}

func (h *ScaleHandler) GetCommand() string {
	return h.command
}

func (h *ScaleHandler) Respond(ctx context.Context, timeout time.Duration, message *domain.Message) error {
	l := log.With().
		Int("messageId", message.ID).
		Int64("chatId", message.ChatID).
		Str("command", h.GetCommand()).
		Logger()

	l.Info().Msg("handling request")

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	go h.textSender.SendChatAction(ctx, message.ChatID, domain.SendingPhoto)

	if message.ImageURL == "" {
		_ = h.textSender.NotifyAndReturnError(ctx, errors.New("missing image"), message)
		return nil
	}

	args := domain.ParseCommandArgs(message.Text)

	var power float64
	var err error
	if args == "" {
		power = 50
	} else {
		power, err = strconv.ParseFloat(args, 32)
		if err != nil {
			_ = h.textSender.NotifyAndReturnError(ctx, errors.New("usage: /scale or /scale <power>, 1-100"), message)
			return nil
		}
	}

	rescaled, err := h.imageConverter.Scale(ctx, message.ImageURL, float32(power))
	if err != nil {
		return h.textSender.NotifyAndReturnError(ctx, fmt.Errorf("failed to scale image: %w", err), message)
	}

	err = h.imageSender.SendImageFileReply(ctx, message, rescaled)
	if err != nil {
		return h.textSender.NotifyAndReturnError(ctx, fmt.Errorf("failed send scaled image: %w", err), message)
	}

	return nil
}
