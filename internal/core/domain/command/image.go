package command

import (
	"context"
	"fmt"
	"hsbot/internal/core/domain"
	"hsbot/internal/core/port"
	"time"
)

type Image struct {
	imageGenerator port.ImageGenerator
	imageSender    port.ImageSender
	textSender     port.TextSender
	tracker        port.Tracker
	command        string
}

func NewImage(imageGenerator port.ImageGenerator,
	imageSender port.ImageSender,
	textSender port.TextSender,
	tracker port.Tracker,
	command string) *Image {
	return &Image{imageGenerator: imageGenerator,
		imageSender: imageSender,
		textSender:  textSender,
		tracker:     tracker,
		command:     command}
}

func (i *Image) GetCommand() string {
	return i.command
}

func (i *Image) Respond(ctx context.Context, timeout time.Duration, message *domain.Message) error {
	ctx, cancel, l := beginRespond(ctx, timeout, message, i.GetCommand())
	defer cancel()

	if !i.tracker.CheckLimit(ctx, message.ChatID) {
		l.Debug().Msg("spending limit reached")
		return nil
	}

	go i.textSender.SendChatAction(ctx, message.ChatID, domain.SendingPhoto)

	prompt := ParseCommandArgs(message.Text)
	if prompt == "" {
		_ = i.textSender.NotifyAndReturnError(ctx, domain.ErrMissingPrompt, message)
		return nil
	}

	image, err := i.imageGenerator.NewImage(ctx, prompt)
	if err != nil {
		err = fmt.Errorf("error generating image: %w", err)
		return i.textSender.NotifyAndReturnError(ctx, err, message)
	}

	i.tracker.AddCost(message.ChatID, image.Cost)

	err = i.imageSender.SendImageFileReply(ctx, message, image.Data)
	if err != nil {
		err = fmt.Errorf("error sending image: %w", err)
		return i.textSender.NotifyAndReturnError(ctx, err, message)
	}

	return nil
}
