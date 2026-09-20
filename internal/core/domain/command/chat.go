package command

import (
	"context"
	"errors"
	"fmt"
	"hsbot/internal/core/domain"
	"hsbot/internal/core/port"
	"time"

	"github.com/rs/zerolog"

	"github.com/rs/zerolog/log"
)

type Chat struct {
	textGenerator  port.TextGenerator
	textSender     port.TextSender
	transcriber    port.Transcriber
	audioGenerator port.AudioGenerator
	audioSender    port.AudioSender
	command        string
	speakCommand   string
	conversations  *conversationStore
	debugReplies   bool

	tracker port.Tracker
	l       *zerolog.Logger
}

type ChatParams struct {
	TextGenerator  port.TextGenerator
	TextSender     port.TextSender
	Transcriber    port.Transcriber
	AudioGenerator port.AudioGenerator
	AudioSender    port.AudioSender
	Command        string
	SpeakCommand   string
	CacheDuration  time.Duration
	DebugReplies   bool
	Tracker        port.Tracker
}

func NewChat(p ChatParams) (*Chat, error) {
	if p.SpeakCommand != "" && (p.AudioGenerator == nil || p.AudioSender == nil) {
		return nil, errors.New("spoken replies require audio generator and sender")
	}

	logger := log.With().
		Str("command", p.Command).
		Str("handler", "chat").
		Logger()

	h := &Chat{
		textGenerator:  p.TextGenerator,
		textSender:     p.TextSender,
		transcriber:    p.Transcriber,
		audioGenerator: p.AudioGenerator,
		audioSender:    p.AudioSender,
		command:        p.Command,
		speakCommand:   p.SpeakCommand,
		conversations:  newConversationStore(p.CacheDuration),
		debugReplies:   p.DebugReplies,
		tracker:        p.Tracker,
		l:              &logger,
	}

	return h, nil
}

func (c *Chat) GetCommand() string {
	return c.command
}

func (c *Chat) Clear(chatID int64) (*Conversation, bool) {
	return c.conversations.delete(chatID)
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

	if !c.tracker.CheckLimit(ctx, message.ChatID) {
		l.Debug().Msg("spending limit reached")
		return nil
	}

	action := domain.Typing
	if c.shallSpeak(message) {
		action = domain.RecordingVoiceAction
	}
	go c.textSender.SendChatAction(ctx, message.ChatID, action)

	promptText, err := c.extractPrompt(ctx, message)
	if err != nil {
		return c.textSender.NotifyAndReturnError(ctx, fmt.Errorf("failed to extract prompt: %w", err),
			message)
	}

	conversation := c.conversations.getOrCreate(message.ChatID)

	if message.QuotedText != "" && message.ImageURL == "" {
		if !message.IsReplyToBot {
			conversation.messages = append(conversation.messages, domain.Prompt{
				Author: domain.User,
				Prompt: message.ReplyToUsername + ": " + message.QuotedText})
		}

		conversation.messages = append(conversation.messages, domain.Prompt{
			Author: domain.User,
			Prompt: promptText})
	} else {
		conversation.messages = append(conversation.messages, domain.Prompt{
			Author:   domain.User,
			Prompt:   promptText,
			ImageURL: message.ImageURL})
	}

	response, err := c.textGenerator.GenerateFromPrompt(ctx, conversation.messages)
	if err != nil {
		err := fmt.Errorf("failed to generate response: %w", err)
		conversation.messages = append(conversation.messages, domain.Prompt{Author: domain.System, Prompt: err.Error()})
		return c.textSender.NotifyAndReturnError(ctx, err, message)
	}

	c.tracker.AddCost(message.ChatID, response.Metadata.Cost)

	conversation.messages = append(conversation.messages,
		domain.Prompt{Author: domain.System, Prompt: response.Response})

	if err := c.sendResponse(ctx, message, response.Response); err != nil {
		return err
	}

	if c.debugReplies {
		c.sendDebugInfo(ctx, message, response.Metadata, len(conversation.messages))
	}

	return nil
}

func (c *Chat) shallSpeak(message *domain.Message) bool {
	return c.speakCommand != "" && ParseCommand(message.Text) == c.speakCommand
}

func (c *Chat) sendResponse(ctx context.Context, message *domain.Message, text string) error {
	if !c.shallSpeak(message) {
		_, err := c.textSender.SendMessageReply(ctx, message, text)
		return err
	}

	if !c.tracker.CheckLimit(ctx, message.ChatID) {
		log.Debug().Msg("spending limit reached")
		return nil
	}

	audioResponse, err := c.audioGenerator.Speak(ctx, text)
	if err != nil {
		return c.textSender.NotifyAndReturnError(ctx, fmt.Errorf("failed to generate speech: %w", err), message)
	}

	c.tracker.AddCost(message.ChatID, audioResponse.Cost)

	if err := c.audioSender.SendAudioReply(ctx, message, audioResponse.Data); err != nil {
		return c.textSender.NotifyAndReturnError(ctx, fmt.Errorf("failed to send voice: %w", err), message)
	}

	return nil
}

func (c *Chat) sendDebugInfo(ctx context.Context, message *domain.Message,
	metadata domain.ResponseMetadata, length int) {
	debug := fmt.Sprintf(`debug:
model: %s | retries: %d
c tokens: %d | total tokens: %d
convo size: %d | cost: %f`,
		metadata.Model,
		metadata.Retries,
		metadata.CompletionTokens,
		metadata.TotalTokens,
		length,
		metadata.Cost)

	_, err := c.textSender.SendMessageReply(ctx, message, debug)
	if err != nil {
		log.Warn().Int64("chatID", message.ChatID).Err(err).Msg("failed to send debug info")
	}
}

func (c *Chat) extractPrompt(ctx context.Context, message *domain.Message) (string, error) {
	promptText := ParseCommandArgs(message.Text)

	if message.AudioURL != "" {
		if !c.tracker.CheckLimit(ctx, message.ChatID) {
			log.Debug().Msg("spending limit reached")
			return "", domain.ErrMissingPrompt
		}

		transcript, err := c.transcriber.Transcribe(ctx, message.AudioURL)
		if err != nil {
			return "", fmt.Errorf("failed to generate transcript: %w", err)
		}

		c.tracker.AddCost(message.ChatID, transcript.Metadata.Cost)

		if promptText == "" {
			promptText = transcript.Response
		} else {
			promptText += ": " + transcript.Response
		}
	}

	if promptText == "" {
		return "", domain.ErrEmptyPrompt
	}

	promptText = message.Username + ": " + promptText
	return promptText, nil
}
