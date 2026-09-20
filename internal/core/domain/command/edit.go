package command

import (
	"context"
	"fmt"
	"hsbot/internal/core/domain"
	"hsbot/internal/core/port"
	"time"
)

type Edit struct {
	imageGenerator port.ImageGenerator
	imageSender    port.ImageSender
	textSender     port.TextSender
	tracker        port.Tracker
	command        string
}

func NewEdit(imageGenerator port.ImageGenerator,
	imageSender port.ImageSender,
	textSender port.TextSender,
	tracker port.Tracker,
	command string) *Edit {
	return &Edit{imageGenerator: imageGenerator,
		imageSender: imageSender,
		textSender:  textSender,
		tracker:     tracker,
		command:     command}
}

func (e *Edit) GetCommand() string {
	return e.command
}

func (e *Edit) Respond(ctx context.Context, timeout time.Duration, message *domain.Message) error {
	ctx, cancel, l := beginRespond(ctx, timeout, message, e.GetCommand())
	defer cancel()

	if !e.tracker.CheckLimit(ctx, message.ChatID) {
		l.Debug().Msg("spending limit reached")
		return nil
	}

	go e.textSender.SendChatAction(ctx, message.ChatID, domain.SendingPhoto)

	prompt := ParseCommandArgs(message.Text)
	if prompt == "" {
		_ = e.textSender.NotifyAndReturnError(ctx, domain.ErrMissingPrompt, message)
		return nil
	}

	if message.ImageURL == "" {
		_ = e.textSender.NotifyAndReturnError(ctx, domain.ErrMissingImage, message)
		return nil
	}

	image, err := e.imageGenerator.EditImage(ctx, domain.Prompt{Prompt: prompt, ImageURL: message.ImageURL})
	if err != nil {
		err = fmt.Errorf("error creating edited image: %w", err)
		return e.textSender.NotifyAndReturnError(ctx, err, message)
	}

	e.tracker.AddCost(message.ChatID, image.Cost)

	err = e.imageSender.SendImageFileReply(ctx, message, image.Data)
	if err != nil {
		err = fmt.Errorf("error sending edited image: %w", err)
		return e.textSender.NotifyAndReturnError(ctx, err, message)
	}

	return nil
}
