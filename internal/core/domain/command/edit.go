package command

import (
	"context"
	"errors"
	"fmt"
	"hsbot/internal/core/domain"
	"hsbot/internal/core/port"
	"hsbot/internal/core/service"
	"time"

	"github.com/spf13/viper"

	"github.com/rs/zerolog/log"
)

type Edit struct {
	imageGenerator port.ImageGenerator
	imageSender    port.ImageSender
	textSender     port.TextSender
	track          service.Tracker
	cost           float64
	command        string
}

func NewEdit(imageGenerator port.ImageGenerator,
	imageSender port.ImageSender,
	textSender port.TextSender,
	track service.Tracker,
	command string) *Edit {
	return &Edit{imageGenerator: imageGenerator,
		imageSender: imageSender,
		textSender:  textSender,
		cost:        viper.GetFloat64("fal.image_edit_cost"),
		track:       track,
		command:     command}
}

func (e *Edit) GetCommand() string {
	return e.command
}

func (e *Edit) Respond(ctx context.Context, timeout time.Duration, message *domain.Message) error {
	l := log.With().
		Int("messageId", message.ID).
		Int64("chatId", message.ChatID).
		Str("imageURL", message.ImageURL).
		Str("command", e.GetCommand()).
		Logger()

	l.Info().Msg("handling request")

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	if !e.track.CheckLimit(ctx, message.ChatID) {
		l.Debug().Msg("spending limit reached")
		return nil
	}

	go e.textSender.SendChatAction(ctx, message.ChatID, domain.SendingPhoto)

	prompt := ParseCommandArgs(message.Text)
	if prompt == "" {
		_ = e.textSender.NotifyAndReturnError(ctx, errors.New("empty prompt"), message)
		return nil
	}

	if message.ImageURL == "" {
		_ = e.textSender.NotifyAndReturnError(ctx, errors.New("missing image"), message)
		return nil
	}

	imageURL, err := e.imageGenerator.EditFromPrompt(ctx, domain.Prompt{Prompt: prompt, ImageURL: message.ImageURL})
	if err != nil {
		err = fmt.Errorf("error creating edited image: %w", err)
		return e.textSender.NotifyAndReturnError(ctx, err, message)
	}

	e.track.AddCost(message.ChatID, e.cost)

	err = e.imageSender.SendImageURLReply(ctx, message, imageURL)
	if err != nil {
		err = fmt.Errorf("error sending edited image: %w", err)
		return e.textSender.NotifyAndReturnError(ctx, err, message)
	}

	return nil
}
