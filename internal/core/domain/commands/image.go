package commands

import (
	"context"
	"fmt"
	"hsbot/internal/core/domain"
	"hsbot/internal/core/port"

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

func (h *ImageHandler) Respond(ctx context.Context, message *domain.Message) {
	l := log.With().
		Int("messageId", message.ID).
		Int64("chatId", message.ChatID).
		Str("imageURL", message.ImageURL).
		Str("command", h.GetCommand()).
		Logger()

	l.Info().Msg("handling request")

	ctx, cancel := context.WithCancel(ctx)
	go h.textSender.SendChatAction(ctx, message.ChatID, domain.SendingPhoto)

	prompt := domain.ParseCommandArgs(message.Text)
	if prompt == "" {
		err := h.textSender.SendMessageReply(ctx, message.ChatID, message.ID, "missing image prompt")
		if err != nil {
			l.Error().Err(err).Msg(domain.ErrSendingReplyFailed)
			cancel()
			return
		}
		cancel()
		return
	}

	imageURL, err := h.imageGenerator.GenerateFromPrompt(ctx, prompt)
	if err != nil {
		errMsg := "error getting FAL response"
		l.Error().Err(err).Str("imageURL", imageURL).Msg(errMsg)
		err := h.textSender.SendMessageReply(ctx,
			message.ChatID,
			message.ID,
			fmt.Sprintf("error getting FAL response: %s", err))
		if err != nil {
			l.Error().Err(err).Msg(domain.ErrSendingReplyFailed)
			cancel()
			return
		}
		cancel()
		return
	}

	err = h.imageSender.SendImageURLReply(ctx, message.ChatID, message.ID, imageURL)
	if err != nil {
		l.Error().Err(err).Msg(domain.ErrSendingReplyFailed)
		cancel()
		return
	}
	cancel()
}
