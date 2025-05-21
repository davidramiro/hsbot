package commands

import (
	"context"
	"errors"
	"fmt"
	"hsbot/internal/core/domain"
	"hsbot/internal/core/port"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

type ChatHandler struct {
	textGenerator port.TextGenerator
	textSender    port.TextSender
	transcriber   port.Transcriber
	cacheDuration time.Duration
	command       string
	cache         sync.Map
}

type Conversation struct {
	timestamp  time.Time
	messages   []domain.Prompt
	exitSignal chan struct{}
	chatID     int64
}

func NewChatHandler(textGenerator port.TextGenerator, textSender port.TextSender, transcriber port.Transcriber,
	command string, cacheDuration time.Duration) *ChatHandler {
	h := &ChatHandler{
		textGenerator: textGenerator,
		textSender:    textSender,
		transcriber:   transcriber,
		cacheDuration: cacheDuration,
		command:       command,
	}

	return h
}

func (h *ChatHandler) GetCommand() string {
	return h.command
}

func (h *ChatHandler) Respond(ctx context.Context, timeout time.Duration, message *domain.Message) error {
	l := log.With().
		Int("messageId", message.ID).
		Int64("chatId", message.ChatID).
		Str("command", h.GetCommand()).
		Logger()

	l.Info().Msg("handling request")

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	go h.textSender.SendChatAction(ctx, message.ChatID, domain.Typing)

	promptText, model, err := h.extractPrompt(ctx, message)
	if err != nil {
		log.Error().Err(err).Msg("failed to extract prompt text")
	}

	if promptText == "" {
		log.Debug().Msg(domain.ErrEmptyPrompt)
		return nil
	}

	var conversation *Conversation
	c, ok := h.cache.Load(message.ChatID)
	if !ok {
		l.Debug().Msg("new conversation")

		h.cache.Store(message.ChatID, &Conversation{
			chatID:     message.ChatID,
			exitSignal: make(chan struct{}),
		})
		c, _ = h.cache.Load(message.ChatID)
		conversation, ok = c.(*Conversation)
		if !ok {
			err := errors.New("conversation type error")
			l.Error().Err(err).Send()
			return err
		}
	} else {
		conversation, ok = c.(*Conversation)
		if !ok {
			err := errors.New("conversation type error")
			l.Error().Err(err).Send()
			return err
		}
		l.Debug().Msg("existing conversation, stopping timer")
		conversation.exitSignal <- struct{}{}
	}

	conversation.timestamp = time.Now()
	go h.startConversationTimer(conversation)

	if message.QuotedText != "" && message.ImageURL == "" {
		// if there's a user message being replied to, add the previous message to the context
		if !message.IsReplyToBot {
			conversation.messages = append(conversation.messages, domain.Prompt{
				Author: domain.User,
				Model:  model,
				Prompt: message.ReplyToUsername + ": " + message.QuotedText})
		}

		conversation.messages = append(conversation.messages, domain.Prompt{
			Author: domain.User,
			Prompt: promptText})
	} else {
		conversation.messages = append(conversation.messages, domain.Prompt{
			Author:   domain.User,
			Model:    model,
			Prompt:   promptText,
			ImageURL: message.ImageURL})
	}

	response, err := h.textGenerator.GenerateFromPrompt(ctx, conversation.messages)
	if err != nil {
		l.Error().Err(err).Msg("failed to generate prompt")
		conversation.messages = append(conversation.messages, domain.Prompt{Author: domain.System, Prompt: err.Error()})

		_, err = h.textSender.SendMessageReply(ctx,
			message.ChatID,
			message.ID,
			fmt.Sprintf("failed to generate reply: %s", err))
		if err != nil {
			l.Error().Err(err).Msg(domain.ErrSendingReplyFailed)
			return err
		}

		return nil
	}

	l.Debug().Msg("reply generated")
	conversation.messages = append(conversation.messages, domain.Prompt{Author: domain.System, Prompt: response})

	_, err = h.textSender.SendMessageReply(ctx,
		message.ChatID,
		message.ID,
		response)
	if err != nil {
		l.Error().Err(err).Msg(domain.ErrSendingReplyFailed)
		return err
	}

	return nil
}

func (h *ChatHandler) extractPrompt(ctx context.Context, message *domain.Message) (string, domain.Model, error) {
	l := log.With().
		Int("messageId", message.ID).
		Int64("chatId", message.ChatID).
		Str("command", h.GetCommand()).
		Logger()

	promptText := domain.ParseCommandArgs(message.Text)
	if promptText == "" {
		_, err := h.textSender.SendMessageReply(ctx, message.ChatID, message.ID, "please input a prompt")
		if err != nil {
			l.Error().Err(err).Msg(domain.ErrSendingReplyFailed)
			return "", domain.Model{}, err
		}
		return "", domain.Model{}, nil
	}
	model := domain.FindModelByMessage(&promptText)

	if message.AudioURL != "" {
		transcript, err := h.transcriber.GenerateFromAudio(ctx, message.AudioURL)
		if err != nil {
			l.Error().Err(err).Msg(domain.ErrSendingReplyFailed)
			return "", domain.Model{}, err
		}

		promptText += ": " + transcript
	}

	promptText = message.Username + ": " + promptText
	return promptText, model, nil
}

func (h *ChatHandler) startConversationTimer(convo *Conversation) {
	t := time.NewTimer(h.cacheDuration)

	for {
		select {
		case <-t.C:
			log.Debug().Int64("chatID", convo.chatID).Msg("clearing conversation")
			h.cache.Delete(convo.chatID)
			return
		case <-convo.exitSignal:
			t.Stop()
			return
		}
	}
}
