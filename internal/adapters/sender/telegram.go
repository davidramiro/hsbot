package sender

import (
	"bytes"
	"context"
	"fmt"
	"hsbot/internal/core/domain"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/rs/zerolog/log"
)

const TelegramMessageLimit = 4096

type TelegramSender struct {
	bot *bot.Bot
}

func NewTelegramSender(bot *bot.Bot) *TelegramSender {
	return &TelegramSender{bot: bot}
}

func (s *TelegramSender) SendMessageReply(
	ctx context.Context,
	chatID int64,
	messageID int,
	message string) (int, error) {
	replies := (len(message) + TelegramMessageLimit - 1) / TelegramMessageLimit
	lastSentID := -1

	for i := range replies {
		substr := message[i*TelegramMessageLimit : min(len(message), (i+1)*TelegramMessageLimit)]
		sent, err := s.bot.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   substr,
			ReplyParameters: &models.ReplyParameters{
				MessageID: messageID,
				ChatID:    chatID,
			},
		})
		if err != nil {
			log.Error().Err(err).Msg("failed to send text response")
			return -1, err
		}

		lastSentID = sent.ID
	}

	return lastSentID, nil
}

func (s *TelegramSender) SendImageURLReply(ctx context.Context, chatID int64, messageID int, url string) error {
	params := &bot.SendPhotoParams{
		ChatID: chatID,
		ReplyParameters: &models.ReplyParameters{
			MessageID: messageID,
			ChatID:    chatID,
		},
		Photo: &models.InputFileString{Data: url},
	}

	_, err := s.bot.SendPhoto(ctx, params)
	if err != nil {
		log.Error().Err(err).Msg("failed to send photo response")
		return err
	}

	return nil
}

func (s *TelegramSender) SendImageFileReply(ctx context.Context, chatID int64, messageID int, file []byte) error {
	params := &bot.SendPhotoParams{
		ChatID: chatID,
		Photo: &models.InputFileUpload{Filename: fmt.Sprintf("%d.png", messageID),
			Data: bytes.NewReader(file)},
		ReplyParameters: &models.ReplyParameters{
			MessageID: messageID,
			ChatID:    chatID,
		},
	}

	_, err := s.bot.SendPhoto(ctx, params)
	if err != nil {
		log.Error().Err(err).Msg("failed to send file response")
		return err
	}

	return nil
}

const ChatActionRepeatSeconds = 5

func (s *TelegramSender) SendChatAction(ctx context.Context, chatID int64, action domain.Action) {
	log.Debug().Int64("chatID", chatID).Msg("starting action routine")

	for {
		select {
		case <-ctx.Done():
			log.Debug().Int64("chatID", chatID).Msg("done, stopping action routine")
			return
		default:
		}

		var chatAction models.ChatAction
		switch action {
		case domain.SendingPhoto:
			chatAction = models.ChatActionUploadPhoto
		case domain.Typing:
			chatAction = models.ChatActionTyping
		default:
			chatAction = models.ChatActionTyping
		}

		log.Debug().Int64("chatID", chatID).Str("chatAction", string(chatAction)).
			Msg("transmitting action")
		_, err := s.bot.SendChatAction(ctx, &bot.SendChatActionParams{
			ChatID: chatID,
			Action: chatAction,
		})
		if err != nil {
			log.Err(err).Msg("error sending chat action")
			return
		}

		time.Sleep(ChatActionRepeatSeconds * time.Second)
	}
}
