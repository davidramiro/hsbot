package command

import (
	"context"
	"errors"
	"fmt"
	"hsbot/internal/core/domain"
	"hsbot/internal/core/port"
	"strconv"
	"time"
)

type Scale struct {
	imageConverter port.ImageConverter
	textSender     port.TextSender
	imageSender    port.ImageSender
	command        string
}

func NewScale(imageConverter port.ImageConverter, textSender port.TextSender, imageSender port.ImageSender,
	command string) *Scale {
	return &Scale{imageConverter: imageConverter, textSender: textSender, imageSender: imageSender,
		command: command}
}

func (s *Scale) GetCommand() string {
	return s.command
}

func (s *Scale) Respond(ctx context.Context, timeout time.Duration, message *domain.Message) error {
	ctx, cancel, _ := beginRespond(ctx, timeout, message, s.GetCommand())
	defer cancel()

	go s.textSender.SendChatAction(ctx, message.ChatID, domain.SendingPhoto)

	if message.ImageURL == "" {
		_ = s.textSender.NotifyAndReturnError(ctx, domain.ErrMissingImage, message)
		return nil
	}

	args := ParseCommandArgs(message.Text)

	var power float64
	var err error
	if args == "" {
		power = 50
	} else {
		power, err = strconv.ParseFloat(args, 32)
		if err != nil {
			_ = s.textSender.NotifyAndReturnError(ctx, errors.New("usage: /scale or /scale <power>, 1-100"), message)
			return nil
		}
	}

	rescaled, err := s.imageConverter.Scale(ctx, message.ImageURL, float32(power))
	if err != nil {
		return s.textSender.NotifyAndReturnError(ctx, fmt.Errorf("failed to scale image: %w", err), message)
	}

	err = s.imageSender.SendImageFileReply(ctx, message, rescaled)
	if err != nil {
		return s.textSender.NotifyAndReturnError(ctx, fmt.Errorf("failed send scaled image: %w", err), message)
	}

	return nil
}
