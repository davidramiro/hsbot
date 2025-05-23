package commands

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

type ChatHandler struct {
	textGenerator port.TextGenerator
	textSender    port.TextSender
	transcriber   port.Transcriber
	cacheDuration time.Duration
	command       string
	cache         sync.Map
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

func NewChatHandler(textGenerator port.TextGenerator, textSender port.TextSender, transcriber port.Transcriber,
	command string, cacheDuration time.Duration) (*ChatHandler, error) {
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

	h := &ChatHandler{
		textGenerator: textGenerator,
		textSender:    textSender,
		transcriber:   transcriber,
		cacheDuration: cacheDuration,
		command:       command,
		models:        models,
		defaultModel:  model,
		l:             &logger,
	}

	return h, nil
}

func (c *ChatHandler) GetCommand() string {
	return c.command
}

func (c *ChatHandler) Respond(ctx context.Context, timeout time.Duration, message *domain.Message) error {
	l := c.l.With().
		Int("messageId", message.ID).
		Int64("chatId", message.ChatID).
		Str("func", "Respond").
		Logger()

	l.Info().Msg("handling request")

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

	l.Debug().Msg("reply generated")
	conversation.messages = append(conversation.messages,
		domain.Prompt{Author: domain.System, Prompt: response.Response})

	if viper.GetBool("bot.debug_replies") {
		addDebugInfo(&response.Response, response.Metadata, len(conversation.messages))
	}

	_, err = c.textSender.SendMessageReply(ctx,
		message,
		response.Response)
	if err != nil {
		return err
	}

	return nil
}

func (c *ChatHandler) getConversationForMessage(message *domain.Message) (*Conversation, error) {
	l := c.l.With().
		Int("messageId", message.ID).
		Int64("chatId", message.ChatID).
		Str("func", "getConversationForMessage").
		Logger()

	var conversation *Conversation
	conv, ok := c.cache.Load(message.ChatID)
	if !ok {
		l.Debug().Msg("new conversation")

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
		l.Debug().Msg("existing conversation, stopping timer")
		conversation.exitSignal <- struct{}{}
	}
	return conversation, nil
}

func addDebugInfo(response *string, metadata domain.ResponseMetadata, length int) {
	*response = fmt.Sprintf(`%s

--
debug: model: %s
c tokens: %d | total tokens: %d
convo size: %d`,
		*response,
		metadata.Model,
		metadata.CompletionTokens,
		metadata.TotalTokens,
		length)
}

func (c *ChatHandler) extractPrompt(ctx context.Context, message *domain.Message) (string, domain.Model, error) {
	promptText := domain.ParseCommandArgs(message.Text)
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

func (c *ChatHandler) startConversationTimer(convo *Conversation) {
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

func (c *ChatHandler) findModelByMessage(message *string) domain.Model {
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
