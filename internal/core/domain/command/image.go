package command

import (
	"context"
	"errors"
	"fmt"
	"hsbot/internal/core/domain"
	"hsbot/internal/core/port"
	"hsbot/internal/core/service"
	"time"

	"github.com/rs/zerolog/log"
)

type Image struct {
	imageGenerator port.ImageGenerator
	imageSender    port.ImageSender
	textSender     port.TextSender
	track          service.Tracker
	command        string
}

func NewImage(imageGenerator port.ImageGenerator,
	imageSender port.ImageSender,
	textSender port.TextSender,
	track service.Tracker,
	command string) *Image {
	return &Image{imageGenerator: imageGenerator,
		imageSender: imageSender,
		textSender:  textSender,
		track:       track,
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

	if !i.track.CheckLimit(ctx, message.ChatID) {
		l.Debug().Msg("spending limit reached")
		return nil
	}

	go i.textSender.SendChatAction(ctx, message.ChatID, domain.SendingPhoto)

	prompt := ParseCommandArgs(message.Text)
	if prompt == "" {
		_ = i.textSender.NotifyAndReturnError(ctx, errors.New("missing image prompt"), message)
		return nil
	}

	image, err := i.imageGenerator.GenerateImage(ctx, prompt)
	if err != nil {
		err = fmt.Errorf("error generating image: %w", err)
		return i.textSender.NotifyAndReturnError(ctx, err, message)
	}

	i.track.AddCost(message.ChatID, image.Cost)

	err = i.imageSender.SendImageFileReply(ctx, message, image.Data)
	if err != nil {
		err = fmt.Errorf("error sending image: %w", err)
		return i.textSender.NotifyAndReturnError(ctx, err, message)
	}

	return nil
}
