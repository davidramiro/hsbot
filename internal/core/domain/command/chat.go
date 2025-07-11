package command

import (
	"context"
	"errors"
	"fmt"
	"hsbot/internal/core/domain"
	"hsbot/internal/core/port"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"

	"github.com/spf13/viper"

	"github.com/rs/zerolog/log"
)

type Chat struct {
	textGenerator port.TextGenerator
	textSender    port.TextSender
	transcriber   port.Transcriber
	cacheDuration time.Duration
	command       string
	cache         *sync.Map
	models        []domain.Model
	defaultModel  domain.Model
	l             *zerolog.Logger
}

type Conversation struct {
	timestamp  time.Time
	messages   []domain.Prompt
	exitSignal chan struct{}
	chatID     int64
}

func NewChat(textGenerator port.TextGenerator, textSender port.TextSender, transcriber port.Transcriber,
	command string, cacheDuration time.Duration) (*Chat, error) {
	var models []domain.Model
	err := viper.UnmarshalKey("openrouter.models", &models)
	if err != nil {
		log.Error().Err(err).Msg("failed to unmarshal openrouter models from config")
		return nil, err
	}

	var model domain.Model
	err = viper.UnmarshalKey("openrouter.default_model", &model)
	if err != nil {
		log.Error().Err(err).Msg("failed to unmarshal openrouter default model from config")
		return nil, err
	}

	logger := log.With().
		Str("command", command).
		Str("handler", "chat").
		Logger()

	h := &Chat{
		textGenerator: textGenerator,
		textSender:    textSender,
		transcriber:   transcriber,
		cacheDuration: cacheDuration,
		command:       command,
		cache:         &sync.Map{},
		models:        models,
		defaultModel:  model,
		l:             &logger,
	}

	return h, nil
}

func (c *Chat) GetCommand() string {
	return c.command
}

func (c *Chat) Respond(ctx context.Context, timeout time.Duration, message *domain.Message) error {
	l := c.l.With().
		Int("messageId", message.ID).
		Int64("chatId", message.ChatID).
		Str("func", "Respond").
		Logger()

	l.Debug().Str("prompt", message.Text).
		Str("quoted", message.QuotedText).
		Str("image", message.ImageURL).
		Str("audio", message.AudioURL).
		Str("username", message.Username).
		Msg("handling request")

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	go c.textSender.SendChatAction(ctx, message.ChatID, domain.Typing)

	promptText, model, err := c.extractPrompt(ctx, message)
	if err != nil {
		err := c.textSender.NotifyAndReturnError(ctx, fmt.Errorf("failed to extract prompt: %w", err),
			message)
		return err
	}

	conversation, err := c.getConversationForMessage(message)
	if err != nil {
		if err := c.textSender.NotifyAndReturnError(ctx, fmt.Errorf("failed to get conversation: %w", err),
			message); err != nil {
			return err
		}
	}

	conversation.timestamp = time.Now()
	go c.startConversationTimer(conversation)

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
			Model:  model,
			Prompt: promptText})
	} else {
		conversation.messages = append(conversation.messages, domain.Prompt{
			Author:   domain.User,
			Model:    model,
			Prompt:   promptText,
			ImageURL: message.ImageURL})
	}

	response, err := c.textGenerator.GenerateFromPrompt(ctx, conversation.messages)
	if err != nil {
		err := fmt.Errorf("failed to generate response: %w", err)
		conversation.messages = append(conversation.messages, domain.Prompt{Author: domain.System, Prompt: err.Error()})
		return c.textSender.NotifyAndReturnError(ctx, err, message)
	}

	conversation.messages = append(conversation.messages,
		domain.Prompt{Author: domain.System, Prompt: response.Response})

	_, err = c.textSender.SendMessageReply(ctx,
		message,
		response.Response)
	if err != nil {
		return err
	}

	if viper.GetBool("bot.debug_replies") {
		go c.sendDebugInfo(message, response.Metadata, len(conversation.messages))
	}

	return nil
}

func (c *Chat) getConversationForMessage(message *domain.Message) (*Conversation, error) {
	l := c.l.With().
		Int("messageId", message.ID).
		Int64("chatId", message.ChatID).
		Str("func", "getConversationForMessage").
		Logger()

	var conversation *Conversation
	conv, ok := c.cache.Load(message.ChatID)
	if !ok {
		l.Trace().Msg("new conversation")

		c.cache.Store(message.ChatID, &Conversation{
			chatID:     message.ChatID,
			exitSignal: make(chan struct{}),
		})
		conv, _ = c.cache.Load(message.ChatID)
		conversation, ok = conv.(*Conversation)
		if !ok {
			return nil, errors.New("conversation type error")
		}
	} else {
		conversation, ok = conv.(*Conversation)
		if !ok {
			return nil, errors.New("conversation type error")
		}
		l.Trace().Msg("existing conversation, stopping timer")
		conversation.exitSignal <- struct{}{}
	}
	return conversation, nil
}

func (c *Chat) sendDebugInfo(message *domain.Message, metadata domain.ResponseMetadata, length int) {
	debug := fmt.Sprintf(`debug: model: %s
c tokens: %d | total tokens: %d
convo size: %d`,
		metadata.Model,
		metadata.CompletionTokens,
		metadata.TotalTokens,
		length)

	ctx, cancel := context.WithTimeout(context.Background(), viper.GetDuration("chat.context_timeout"))
	defer cancel()

	_, err := c.textSender.SendMessageReply(ctx,
		message,
		debug)
	if err != nil {
		log.Warn().Int64("chatID", message.ChatID).Err(err).Msg("failed to send debug info")
	}
}

func (c *Chat) extractPrompt(ctx context.Context, message *domain.Message) (string, domain.Model, error) {
	promptText := ParseCommandArgs(message.Text)
	if promptText == "" {
		return "", domain.Model{}, domain.ErrEmptyPrompt
	}

	model := c.findModelByMessage(&promptText)

	if message.AudioURL != "" {
		transcript, err := c.transcriber.GenerateFromAudio(ctx, message.AudioURL)
		if err != nil {
			return "", domain.Model{}, fmt.Errorf("failed to generate transcript: %w", err)
		}

		promptText += ": " + transcript
	}

	promptText = message.Username + ": " + promptText
	return promptText, model, nil
}

func (c *Chat) startConversationTimer(convo *Conversation) {
	t := time.NewTimer(c.cacheDuration)

	for {
		select {
		case <-t.C:
			c.l.Debug().Int64("chatID", convo.chatID).Msg("clearing conversation")
			c.cache.Delete(convo.chatID)
			return
		case <-convo.exitSignal:
			t.Stop()
			return
		}
	}
}

func (c *Chat) findModelByMessage(message *string) domain.Model {
	for _, model := range c.models {
		lowercaseMessage := strings.ToLower(*message)
		lowerCaseModel := strings.ToLower("#" + model.Keyword)
		if strings.Contains(lowercaseMessage, lowerCaseModel) {
			i := strings.Index(lowercaseMessage, lowerCaseModel)
			*message = (*message)[:i] + (*message)[i+len(lowerCaseModel):]
			return model
		}
	}

	return c.defaultModel
}
