package commands

import (
	"context"
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
	cache         map[int64]*Conversation
	mutex         sync.Mutex
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
		cache:         make(map[int64]*Conversation),
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

	promptText := domain.ParseCommandArgs(message.Text)
	if promptText == "" {
		err := h.textSender.SendMessageReply(ctx, message.ChatID, message.ID, "please input a prompt")
		if err != nil {
			l.Error().Err(err).Msg(domain.ErrSendingReplyFailed)
			return err
		}
		return nil
	}

	if message.AudioURL != "" {
		transcript, err := h.transcriber.GenerateFromAudio(ctx, message.AudioURL)
		if err != nil {
			l.Error().Err(err).Msg(domain.ErrSendingReplyFailed)
			return err
		}

		promptText += ": " + transcript
	}

	promptText = message.Username + ": " + promptText

	h.mutex.Lock()
	defer h.mutex.Unlock()

	conversation, ok := h.cache[message.ChatID]
	if !ok {
		l.Debug().Msg("new conversation")

		h.cache[message.ChatID] = &Conversation{
			chatID:     message.ChatID,
			exitSignal: make(chan struct{}),
		}
		conversation = h.cache[message.ChatID]
	} else {
		conversation.exitSignal <- struct{}{}
	}

	conversation.timestamp = time.Now()

	if message.QuotedText != "" && message.ImageURL == "" {
		// if there's a user message being replied to, add the previous message to the context
		if !message.IsReplyToBot {
			conversation.messages = append(conversation.messages, domain.Prompt{Author: domain.User,
				Prompt: message.QuotedText})
		}

		conversation.messages = append(conversation.messages, domain.Prompt{Author: domain.User,
			Prompt: promptText})
	} else {
		conversation.messages = append(conversation.messages, domain.Prompt{Author: domain.User,
			Prompt: promptText, ImageURL: message.ImageURL})
	}

	response, err := h.textGenerator.GenerateFromPrompt(ctx, conversation.messages)
	if err != nil {
		conversation.messages = append(conversation.messages, domain.Prompt{Author: domain.System, Prompt: err.Error()})

		err = h.textSender.SendMessageReply(ctx,
			message.ChatID,
			message.ID,
			fmt.Sprintf("failed to generate reply: %s", err))
		if err != nil {
			l.Error().Err(err).Msg(domain.ErrSendingReplyFailed)
			return err
		}

		return nil
	}

	conversation.messages = append(conversation.messages, domain.Prompt{Author: domain.System, Prompt: response})

	go h.startConversationTimer(conversation)

	err = h.textSender.SendMessageReply(ctx,
		message.ChatID,
		message.ID,
		response)
	if err != nil {
		l.Error().Err(err).Msg(domain.ErrSendingReplyFailed)
		return err
	}

	return nil
}

func (h *ChatHandler) startConversationTimer(convo *Conversation) {
	t := time.NewTimer(h.cacheDuration)

	for {
		select {
		case <-t.C:
			log.Debug().Int64("chatID", convo.chatID).Msg("clearing conversation")

			h.mutex.Lock()
			delete(h.cache, convo.chatID)
			h.mutex.Unlock()
			return
		case <-convo.exitSignal:
			t.Stop()
			return
		}
	}
}
