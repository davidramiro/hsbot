package commands

import (
	"context"
	"fmt"
	"hsbot/internal/core/domain"
	"hsbot/internal/core/port"
	"time"

	"github.com/rs/zerolog/log"
)

type ChatHandler struct {
	textGenerator port.TextGenerator
	textSender    port.TextSender
	command       string
	cache         map[int64]*Conversation
}

type Conversation struct {
	timestamp time.Time
	messages  []domain.Prompt
}

func NewChatHandler(textGenerator port.TextGenerator, textSender port.TextSender, command string,
	cacheDuration, tickRate time.Duration) *ChatHandler {
	h := &ChatHandler{
		textGenerator: textGenerator,
		textSender:    textSender,
		command:       command,
	}

	go h.clearCache(cacheDuration, tickRate)

	return h
}

func (h *ChatHandler) GetCommand() string {
	return h.command
}

func (h *ChatHandler) Respond(ctx context.Context, message *domain.Message) error {
	l := log.With().
		Int("messageId", message.ID).
		Int64("chatId", message.ChatID).
		Str("command", h.GetCommand()).
		Logger()

	l.Info().Msg("handling request")

	ctx, cancel := context.WithCancel(ctx)
	go h.textSender.SendChatAction(ctx, message.ChatID, domain.Typing)

	promptText := domain.ParseCommandArgs(message.Text)
	if promptText == "" {
		err := h.textSender.SendMessageReply(ctx, message.ChatID, message.ID, "please input a prompt")
		if err != nil {
			l.Error().Err(err).Msg(domain.ErrSendingReplyFailed)
			cancel()
			return err
		}
		cancel()
		return nil
	}

	conversation, ok := h.cache[message.ChatID]
	if !ok {
		l.Debug().Msg("new conversation")
		h.cache = make(map[int64]*Conversation)

		h.cache[message.ChatID] = &Conversation{
			messages: []domain.Prompt{
				{
					Author:   domain.User,
					Prompt:   promptText,
					ImageURL: message.ImageURL,
				},
			},
		}
		conversation = h.cache[message.ChatID]
		l.Debug().Int("message cache size", len(h.cache[message.ChatID].messages)).Msg("")
	} else {
		conversation.messages = append(conversation.messages, domain.Prompt{Author: domain.User,
			Prompt: promptText, ImageURL: message.ImageURL})
	}

	response, err := h.textGenerator.GenerateFromPrompt(ctx, conversation.messages)
	if err != nil {
		err := h.textSender.SendMessageReply(ctx,
			message.ChatID,
			message.ID,
			fmt.Sprintf("failed to generate reply: %s", err))
		if err != nil {
			l.Error().Err(err).Msg(domain.ErrSendingReplyFailed)
			cancel()
			return err
		}
		cancel()
		return nil
	}

	conversation.messages = append(conversation.messages, domain.Prompt{Author: domain.System, Prompt: response})
	conversation.timestamp = time.Now()

	err = h.textSender.SendMessageReply(ctx,
		message.ChatID,
		message.ID,
		response)
	if err != nil {
		l.Error().Err(err).Msg(domain.ErrSendingReplyFailed)
		cancel()
		return err
	}

	cancel()
	return nil
}

func (h *ChatHandler) clearCache(timeout, tick time.Duration) {
	log.Debug().Msg("gpt cache timer started")

	for range time.Tick(tick) {
		for chatID := range h.cache {
			log.Debug().Int64("chatID", chatID).Msg("checking timestamp for id")
			messageTime := h.cache[chatID].timestamp
			if messageTime.Add(timeout).Before(time.Now()) {
				log.Debug().Int64("chatID", chatID).Msg("expired chat, resetting")
				delete(h.cache, chatID)
			}
		}
	}
}
