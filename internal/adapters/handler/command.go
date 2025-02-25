package handler

import (
	"context"
	"hsbot/internal/core/domain"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/rs/zerolog/log"
)

type CommandHandler struct {
	commandRegistry *domain.CommandRegistry
	timeout         time.Duration
}

func NewCommandHandler(commandRegistry *domain.CommandRegistry, timeout time.Duration) *CommandHandler {
	return &CommandHandler{commandRegistry: commandRegistry, timeout: timeout}
}

func (h *CommandHandler) Handle(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}

	if update.Message.Photo != nil {
		update.Message.Text = update.Message.Caption
	}

	log.Debug().Str("message", update.Message.Text).Msg("registering chat command handler")

	cmd := domain.ParseCommand(update.Message.Text)
	commandHandler, err := h.commandRegistry.Get(cmd)
	if err != nil {
		log.Debug().Str("command", cmd).Msg("no handler for command")
		return
	}

	replyToMessageID := new(int)
	var quotedText string
	var isReplyToBot bool
	var replyToUsername string

	if update.Message.ReplyToMessage != nil {
		botUser, err := b.GetMe(ctx)
		if err != nil {
			log.Debug().Str("command", cmd).Msg("failed to get bot user")
			return
		}
		if update.Message.ReplyToMessage.From.ID == botUser.ID {
			isReplyToBot = true
			quotedText = update.Message.ReplyToMessage.Text
		} else {
			replyToUsername = update.Message.ReplyToMessage.From.Username
			quotedText = update.Message.ReplyToMessage.Text
		}

		*replyToMessageID = update.Message.ReplyToMessage.ID
	}

	imageURL := make(chan string)
	audioURL := make(chan string)

	go getOptionalImage(ctx, b, update, imageURL)
	go getOptionalAudio(ctx, b, update, audioURL)

	go func() {
		err := commandHandler.Respond(ctx, h.timeout, &domain.Message{
			ID:               update.Message.ID,
			ChatID:           update.Message.Chat.ID,
			Text:             update.Message.Text,
			Username:         getUserNameFromMessage(update.Message.From),
			ReplyToMessageID: replyToMessageID,
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
}

func getOptionalImage(ctx context.Context, b *bot.Bot, update *models.Update, url chan<- string) {
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
		url <- ""
		return
	}

	f, err := b.GetFile(ctx, &bot.GetFileParams{FileID: findLargestImage(photos)})
	if err != nil {
		log.Error().Msg("error getting file from telegram api")
		url <- ""
		return
	}

	url <- b.FileDownloadLink(f)
}

func getOptionalAudio(ctx context.Context, b *bot.Bot, update *models.Update, url chan<- string) {
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
		url <- ""
		return
	}

	f, err := b.GetFile(ctx, &bot.GetFileParams{FileID: fileID})
	if err != nil {
		log.Error().Msg("error getting file from telegram api")
		url <- ""
		return
	}

	url <- b.FileDownloadLink(f)
}

func findLargestImage(photos []models.PhotoSize) string {
	maxSize := -1
	var maxID string
	for _, photo := range photos {
		if photo.FileSize > maxSize {
			maxSize = photo.FileSize
			maxID = photo.FileID
		}
	}

	return maxID
}

func getUserNameFromMessage(user *models.User) string {
	if user.Username == "" {
		return user.FirstName
	}

	return "@" + user.Username
}
