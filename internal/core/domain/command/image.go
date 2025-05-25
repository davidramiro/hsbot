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

type Image struct {
	imageGenerator port.ImageGenerator
	imageSender    port.ImageSender
	textSender     port.TextSender
	command        string
}

func NewImage(imageGenerator port.ImageGenerator,
	imageSender port.ImageSender,
	textSender port.TextSender,
	command string) *Image {
	return &Image{imageGenerator: imageGenerator,
		imageSender: imageSender,
		textSender:  textSender,
		command:     command}
}

func (i *Image) GetCommand() string {
	return i.command
}

func (i *Image) Respond(ctx context.Context, timeout time.Duration, message *domain.Message) error {
	l := log.With().
		Int("messageId", message.ID).
		Int64("chatId", message.ChatID).
		Str("imageURL", message.ImageURL).
		Str("command", i.GetCommand()).
		Logger()

	l.Info().Msg("handling request")

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	go i.textSender.SendChatAction(ctx, message.ChatID, domain.SendingPhoto)

	prompt := ParseCommandArgs(message.Text)
	if prompt == "" {
		_ = i.textSender.NotifyAndReturnError(ctx, errors.New("missing image prompt"), message)
		return nil
	}

	imageURL, err := i.imageGenerator.GenerateFromPrompt(ctx, prompt)
	if err != nil {
		err = fmt.Errorf("error generating image: %w", err)
		return i.textSender.NotifyAndReturnError(ctx, err, message)
	}

	err = i.imageSender.SendImageURLReply(ctx, message, imageURL)
	if err != nil {
		err = fmt.Errorf("error sending edited image: %w", err)
		return i.textSender.NotifyAndReturnError(ctx, err, message)
	}

	return nil
}
