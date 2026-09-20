package handler

import (
	"context"
	"hsbot/internal/core/domain"
	"hsbot/internal/core/domain/command"
	"hsbot/internal/core/port"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/rs/zerolog/log"
)

type Command struct {
	commandRegistry port.CommandRegistry
	timeout         time.Duration
	auth            port.Authorizer
}

func NewCommand(commandRegistry port.CommandRegistry, timeout time.Duration, authorizer port.Authorizer) *Command {
	return &Command{commandRegistry: commandRegistry, timeout: timeout, auth: authorizer}
}

func (c *Command) Handle(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}

	if update.Message.Photo != nil {
		update.Message.Text = update.Message.Caption
	}

	log.Debug().Str("message", update.Message.Text).Msg("received command")

	if !c.auth.IsAuthorized(ctx, update.Message.Chat.ID) {
		log.Debug().Msg("not authorized")
		return
	}

	cmd := command.ParseCommand(update.Message.Text)
	commandHandler, err := c.commandRegistry.Get(cmd)
	if err != nil {
		log.Debug().Str("command", cmd).Msg("no handler for command")
		return
	}

	var replyToMessageID *int
	var quotedText string
	var isReplyToBot bool
	var replyToUsername string

	if update.Message.ReplyToMessage != nil {
		botUser, err := b.GetMe(ctx)
		if err != nil {
			log.Err(err).Str("command", cmd).Msg("failed to get bot user")
			return
		}
		if update.Message.ReplyToMessage.From.ID == botUser.ID {
			isReplyToBot = true
			quotedText = update.Message.ReplyToMessage.Text
		} else {
			replyToUsername = update.Message.ReplyToMessage.From.Username
			quotedText = update.Message.ReplyToMessage.Text
		}

		id := update.Message.ReplyToMessage.ID
		replyToMessageID = &id
	}

	go func() {
		err := commandHandler.Respond(ctx, c.timeout, &domain.Message{
			ID:               update.Message.ID,
			ChatID:           update.Message.Chat.ID,
			Text:             update.Message.Text,
			Username:         getUserNameFromMessage(update.Message.From),
			ReplyToMessageID: replyToMessageID,
			ReplyToUsername:  replyToUsername,
			IsReplyToBot:     isReplyToBot,
			QuotedText:       quotedText,
			ImageURL:         getOptionalImage(ctx, b, update),
			AudioURL:         getOptionalAudio(ctx, b, update),
		})
		if err != nil {
			log.Err(err).Str("command", cmd).Msg("failed to respond to command")
		}
	}()
}

func getOptionalImage(ctx context.Context, b *bot.Bot, update *models.Update) string {
	var photos []models.PhotoSize

	if update.Message.Photo != nil {
		photos = update.Message.Photo
	}

	if update.Message.ReplyToMessage != nil {
		if update.Message.ReplyToMessage.Photo != nil {
			photos = update.Message.ReplyToMessage.Photo
		}
	}

	if len(photos) == 0 {
		return ""
	}

	f, err := b.GetFile(ctx, &bot.GetFileParams{FileID: findMediumSizedImage(photos)})
	if err != nil {
		log.Error().Msg("error getting file from telegram api")
		return ""
	}

	return b.FileDownloadLink(f)
}

func getOptionalAudio(ctx context.Context, b *bot.Bot, update *models.Update) string {
	var fileID string
	if update.Message.Audio != nil {
		fileID = update.Message.Audio.FileID
	}

	if update.Message.ReplyToMessage != nil {
		if update.Message.ReplyToMessage.Voice != nil {
			fileID = update.Message.ReplyToMessage.Voice.FileID
		}

		if update.Message.ReplyToMessage.Audio != nil {
			fileID = update.Message.ReplyToMessage.Audio.FileID
		}
	}

	if fileID == "" {
		return ""
	}

	f, err := b.GetFile(ctx, &bot.GetFileParams{FileID: fileID})
	if err != nil {
		log.Error().Msg("error getting file from telegram api")
		return ""
	}

	return b.FileDownloadLink(f)
}

const minSize = 80000
const maxSize = 130000

func findMediumSizedImage(photos []models.PhotoSize) string {
	for _, photo := range photos {
		if photo.FileSize > minSize && photo.FileSize < maxSize {
			return photo.FileID
		}
	}

	return photos[len(photos)-1].FileID
}

func getUserNameFromMessage(user *models.User) string {
	if user.Username == "" {
		return user.FirstName
	}

	return "@" + user.Username
}
