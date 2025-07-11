package sender

import (
	"bytes"
	"context"
	"fmt"
	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/go-telegram/bot/models"
	"hsbot/internal/core/domain"
	"hsbot/internal/core/port"
	"time"

	"github.com/rs/zerolog/log"
)

const TelegramMessageLimit = 4096

// TelegramBotAPI is a wrapper interface for all the used methods of the *bot.Bot struct. Used for mocking in tests.
type TelegramBotAPI interface {
	SendMessageWithContext(ctx context.Context, chatId int64, text string, opts *gotgbot.SendMessageOpts) (*gotgbot.Message, error)
	SendPhotoWithContext(ctx context.Context, chatId int64, photo gotgbot.InputFileOrString, opts *gotgbot.SendPhotoOpts) (*gotgbot.Message, error)
	SendChatActionWithContext(ctx context.Context, chatId int64, action string, opts *gotgbot.SendChatActionOpts) (bool, error)
}

type Telegram struct {
	b TelegramBotAPI
}

func NewTelegram(bot TelegramBotAPI) *Telegram {
	return &Telegram{b: bot}
}

var _ port.TextSender = (*Telegram)(nil)

func (s *Telegram) SendMessageReply(
	ctx context.Context,
	message *domain.Message,
	text string) (int64, error) {
	replies := (len(text) + TelegramMessageLimit - 1) / TelegramMessageLimit
	var lastSentID int64

	for i := range replies {
		substr := text[i*TelegramMessageLimit : min(len(text), (i+1)*TelegramMessageLimit)]
		sent, err := s.b.SendMessageWithContext(ctx, message.ChatID, substr, &gotgbot.SendMessageOpts{
			ReplyParameters: &gotgbot.ReplyParameters{
				MessageId: message.ID,
				ChatId:    message.ChatID,
			},
		})
		if err != nil {
			return -1, fmt.Errorf("failed to send message: %w", err)
		}

		lastSentID = sent.MessageId
	}

	log.Debug().Int64("chatID", message.ChatID).Str("text", text).Msg("sent reply")

	return lastSentID, nil
}

func (s *Telegram) SendImageURLReply(ctx context.Context, message *domain.Message, url string) error {
	_, err := s.b.SendPhotoWithContext(ctx, message.ChatID, gotgbot.InputFileByURL(url), &gotgbot.SendPhotoOpts{
		ReplyParameters: &gotgbot.ReplyParameters{
			MessageId:                message.ID,
			ChatId:                   message.ChatID,
			AllowSendingWithoutReply: true,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to send image: %w", err)
	}

	log.Debug().Int64("chatID", message.ChatID).Str("url", url).Msg("sent photo reply")

	return nil
}

func (s *Telegram) SendImageFileReply(ctx context.Context, message *domain.Message, file []byte) error {

	f := gotgbot.InputFileByReader(fmt.Sprintf("%d", message.ID), bytes.NewReader(file))

	_, err := s.b.SendPhotoWithContext(ctx, message.ChatID, f, &gotgbot.SendPhotoOpts{
		ReplyParameters: &gotgbot.ReplyParameters{
			MessageId:                message.ID,
			ChatId:                   message.ChatID,
			AllowSendingWithoutReply: true,
		},
	})

	if err != nil {
		return fmt.Errorf("failed to send image: %w", err)
	}

	log.Debug().Int64("chatID", message.ChatID).Int("size", len(file)).Msg("sent photo reply")

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

		_, err := s.b.SendChatActionWithContext(ctx, chatID, string(action), nil)
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
