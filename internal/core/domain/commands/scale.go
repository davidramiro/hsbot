package commands

import (
	"context"
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

	if message.ImageURL == "" || message.ReplyToMessageID == nil {
		_, err := h.textSender.SendMessageReply(ctx, message.ChatID, message.ID,
			"reply to an image")
		if err != nil {
			l.Error().Err(err).Msg(domain.ErrSendingReplyFailed)
			return err
		}
		return nil
	}

	args := domain.ParseCommandArgs(message.Text)

	var power float64
	var err error
	if args == "" {
		power = 80
	} else {
		power, err = strconv.ParseFloat(args, 32)
		if err != nil {
			_, err := h.textSender.SendMessageReply(ctx, message.ChatID, message.ID,
				"usage: /scale or /scale <power>, 1-100")
			if err != nil {
				l.Error().Err(err).Msg(domain.ErrSendingReplyFailed)
				return err
			}
			return nil
		}
	}

	rescaled, err := h.imageConverter.Scale(ctx, message.ImageURL, float32(power))
	if err != nil {
		_, err := h.textSender.SendMessageReply(ctx, message.ChatID, message.ID,
			fmt.Sprintf("failed to scale image: %s", err))
		if err != nil {
			l.Error().Err(err).Msg(domain.ErrSendingReplyFailed)
			return err
		}
		return nil
	}

	err = h.imageSender.SendImageFileReply(ctx, message.ChatID, *message.ReplyToMessageID, rescaled)
	if err != nil {
		l.Error().Err(err).Msg(domain.ErrSendingReplyFailed)
		return err
	}

	return nil
}
