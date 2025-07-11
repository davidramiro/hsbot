package handler

import (
	"context"
	"errors"
	"fmt"
	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"hsbot/internal/core/domain"
	"hsbot/internal/core/domain/command"
	"hsbot/internal/core/port"
	"time"

	"github.com/rs/zerolog/log"
)

type Command struct {
	commandRegistry port.CommandRegistry
	timeout         time.Duration
}

func NewCommand(commandRegistry port.CommandRegistry, timeout time.Duration) *Command {
	return &Command{commandRegistry: commandRegistry, timeout: timeout}
}

func (c *Command) Handle(b *gotgbot.Bot, ctx *ext.Context) error {
	if ctx.EffectiveMessage == nil {
		return errors.New("no message")
	}

	if ctx.EffectiveMessage.Photo != nil {
		ctx.EffectiveMessage.Text = ctx.EffectiveMessage.Caption
	}

	log.Debug().Str("message", ctx.EffectiveMessage.Text).Msg("received command")

	cmd := command.ParseCommand(ctx.EffectiveMessage.Text)
	commandHandler, err := c.commandRegistry.Get(cmd)
	if err != nil {
		log.Debug().Str("command", cmd).Msg("no handler for command")
		return fmt.Errorf("no handler for command: %w", err)
	}

	var quotedText string
	var isReplyToBot bool
	var replyToUsername string
	var replyToMessageID int64

	if ctx.EffectiveMessage.ReplyToMessage != nil {

		if ctx.EffectiveMessage.ReplyToMessage.From.Id == ctx.Bot.Id {
			isReplyToBot = true
			quotedText = ctx.EffectiveMessage.ReplyToMessage.Text
		} else {
			replyToUsername = ctx.EffectiveMessage.ReplyToMessage.From.Username
			quotedText = ctx.EffectiveMessage.ReplyToMessage.Text
		}
	}

	imageURL := make(chan string)
	audioURL := make(chan string)

	go getOptionalImage(ctx, b, imageURL)
	go getOptionalAudio(ctx, b, audioURL)

	go func() {
		err := commandHandler.Respond(context.Background(), c.timeout, &domain.Message{
			ID:               ctx.EffectiveMessage.MessageId,
			ChatID:           ctx.EffectiveChat.Id,
			Text:             ctx.EffectiveMessage.Text,
			Username:         getUserNameOrFirstName(ctx.EffectiveUser),
			ReplyToMessageID: &replyToMessageID,
			ReplyToUsername:  replyToUsername,
			IsReplyToBot:     isReplyToBot,
			QuotedText:       quotedText,
			ImageURL:         <-imageURL,
			AudioURL:         <-audioURL,
		})
		if err != nil {
			log.Err(err).Str("command", cmd).Msg("failed to respond to command")
		}

	}()

	return nil
}

func getOptionalImage(ctx *ext.Context, b *gotgbot.Bot, url chan<- string) {
	var photos []gotgbot.PhotoSize

	if ctx.EffectiveMessage.ReplyToMessage != nil {
		if ctx.EffectiveMessage.ReplyToMessage.Photo != nil {
			photos = ctx.EffectiveMessage.ReplyToMessage.Photo
		}
	}

	if ctx.EffectiveMessage.Photo != nil {
		photos = ctx.EffectiveMessage.Photo
	}

	if len(photos) == 0 {
		url <- ""
		return
	}

	f, err := b.GetFile(findMediumSizedImage(photos), nil)
	if err != nil {
		log.Error().Msg("error getting file from telegram api")
		url <- ""
		return
	}

	url <- f.URL(b, nil)
}

func getOptionalAudio(ctx *ext.Context, b *gotgbot.Bot, url chan<- string) {
	var fileID string

	if ctx.EffectiveMessage.ReplyToMessage != nil {
		if ctx.EffectiveMessage.ReplyToMessage.Voice != nil {
			fileID = ctx.EffectiveMessage.ReplyToMessage.Voice.FileId
		}

		if ctx.EffectiveMessage.ReplyToMessage.Audio != nil {
			fileID = ctx.EffectiveMessage.ReplyToMessage.Audio.FileId
		}
	}

	if ctx.EffectiveMessage.Audio != nil {
		fileID = ctx.EffectiveMessage.Audio.FileId
	}

	if fileID == "" {
		url <- ""
		return
	}

	f, err := b.GetFile(fileID, nil)
	if err != nil {
		log.Error().Msg("error getting file from telegram api")
		url <- ""
		return
	}

	url <- f.URL(b, nil)
}

const minSize = 80000
const maxSize = 130000

func findMediumSizedImage(photos []gotgbot.PhotoSize) string {
	for _, photo := range photos {
		if photo.FileSize > minSize && photo.FileSize < maxSize {
			return photo.FileId
		}
	}

	return photos[len(photos)-1].FileId
}

func getUserNameOrFirstName(user *gotgbot.User) string {
	if user.Username == "" {
		return user.FirstName
	}

	return "@" + user.Username
}
