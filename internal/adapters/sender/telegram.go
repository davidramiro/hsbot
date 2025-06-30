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

// TelegramBotAPI is a wrapper interface for all the used methods of the *bot.Bot struct. Used for mocking in tests.
type TelegramBotAPI interface {
	SendMessage(ctx context.Context, params *bot.SendMessageParams) (*models.Message, error)
	SendPhoto(ctx context.Context, params *bot.SendPhotoParams) (*models.Message, error)
	SendChatAction(ctx context.Context, params *bot.SendChatActionParams) (bool, error)
}

type Telegram struct {
	bot TelegramBotAPI
}

func NewTelegram(bot TelegramBotAPI) *Telegram {
	return &Telegram{bot: bot}
}

func (s *Telegram) SendMessageReply(
	ctx context.Context,
	message *domain.Message,
	text string) (int, error) {
	replies := (len(text) + TelegramMessageLimit - 1) / TelegramMessageLimit
	lastSentID := -1

	for i := range replies {
		substr := text[i*TelegramMessageLimit : min(len(text), (i+1)*TelegramMessageLimit)]
		sent, err := s.bot.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: message.ChatID,
			Text:   substr,
			ReplyParameters: &models.ReplyParameters{
				MessageID: message.ID,
				ChatID:    message.ChatID,
			},
		})
		if err != nil {
			return -1, fmt.Errorf("failed to send message: %w", err)
		}

		lastSentID = sent.ID
	}

	log.Debug().Int64("chatID", message.ChatID).Str("text", text).Msg("sent reply")

	return lastSentID, nil
}

func (s *Telegram) SendImageURLReply(ctx context.Context, message *domain.Message, url string) error {
	params := &bot.SendPhotoParams{
		ChatID: message.ChatID,
		ReplyParameters: &models.ReplyParameters{
			MessageID: message.ID,
			ChatID:    message.ChatID,
		},
		Photo: &models.InputFileString{Data: url},
	}

	log.Debug().Int64("chatID", message.ChatID).Str("url", url).Msg("sent photo reply")

	_, err := s.bot.SendPhoto(ctx, params)
	if err != nil {
		return fmt.Errorf("failed to send image: %w", err)
	}

	return nil
}

func (s *Telegram) SendImageFileReply(ctx context.Context, message *domain.Message, file []byte) error {
	params := &bot.SendPhotoParams{
		ChatID: message.ChatID,
		Photo: &models.InputFileUpload{Filename: fmt.Sprintf("%d.png", message.ID),
			Data: bytes.NewReader(file)},
		ReplyParameters: &models.ReplyParameters{
			MessageID: message.ID,
			ChatID:    message.ChatID,
		},
	}

	log.Debug().Int64("chatID", message.ChatID).Int("size", len(file)).Msg("sent photo reply")

	_, err := s.bot.SendPhoto(ctx, params)
	if err != nil {
		return fmt.Errorf("failed to send image: %w", err)
	}

	return nil
}

const ChatActionRepeatSeconds = 5

func (s *Telegram) SendChatAction(ctx context.Context, chatID int64, action domain.Action) {
	log.Debug().Int64("chatID", chatID).Msg("starting action routine")

	for {
		var chatAction models.ChatAction
		switch action {
		case domain.SendingPhoto:
			chatAction = models.ChatActionUploadPhoto
		case domain.Typing:
			chatAction = models.ChatActionTyping
		default:
			chatAction = models.ChatActionTyping
		}

		log.Trace().Int64("chatID", chatID).Str("chatAction", string(chatAction)).
			Msg("transmitting action")
		_, err := s.bot.SendChatAction(ctx, &bot.SendChatActionParams{
			ChatID: chatID,
			Action: chatAction,
		})
		if err != nil {
			log.Err(err).Msg("error sending chat action")
			return
		}

		select {
		case <-ctx.Done():
			log.Trace().Int64("chatID", chatID).Msg("done, stopping action routine")
			return
		case <-time.After(ChatActionRepeatSeconds * time.Second):
		}
	}
}

func (s *Telegram) NotifyAndReturnError(ctx context.Context, err error, message *domain.Message) error {
	_, err2 := s.SendMessageReply(ctx,
		message,
		fmt.Sprintf("error: %s", err))
	if err2 != nil {
		return fmt.Errorf("failed sending error message: %w: %w", err, err2)
	}
	return err
}
