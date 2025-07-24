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

type Image struct {
	imageGenerator port.ImageGenerator
	imageSender    port.ImageSender
	textSender     port.TextSender
	track          service.Tracker
	auth           service.Authorizer
	cost           float64
	command        string
}

func NewImage(imageGenerator port.ImageGenerator,
	imageSender port.ImageSender,
	textSender port.TextSender,
	auth service.Authorizer,
	track service.Tracker,
	command string) *Image {
	return &Image{imageGenerator: imageGenerator,
		imageSender: imageSender,
		textSender:  textSender,
		cost:        viper.GetFloat64("fal.image_edit_cost"),
		track:       track,
		auth:        auth,
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

	if !i.auth.IsAuthorized(ctx, message.ChatID) {
		l.Debug().Msg("not authorized")
		return nil
	}

	if !i.track.CheckLimit(ctx, message.ChatID) {
		l.Debug().Msg("spend limit reached")
		return nil
	}

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

	i.track.AddCost(message.ChatID, i.cost)

	err = i.imageSender.SendImageURLReply(ctx, message, imageURL)
	if err != nil {
		err = fmt.Errorf("error sending edited image: %w", err)
		return i.textSender.NotifyAndReturnError(ctx, err, message)
	}

	return nil
}
