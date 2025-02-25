package commands

import (
	"context"
	"fmt"
	"hsbot/internal/core/domain"
	"hsbot/internal/core/port"
	"time"

	"github.com/rs/zerolog/log"
)

type ThinkHandler struct {
	textGenerator port.TextGenerator
	textSender    port.TextSender
	transcriber   port.Transcriber
	command       string
}

func NewThinkHandler(textGenerator port.TextGenerator, textSender port.TextSender, transcriber port.Transcriber,
	command string) *ThinkHandler {
	h := &ThinkHandler{
		textGenerator: textGenerator,
		textSender:    textSender,
		transcriber:   transcriber,
		command:       command,
	}

	return h
}

func (t *ThinkHandler) GetCommand() string {
	return t.command
}

func (t *ThinkHandler) Respond(ctx context.Context, timeout time.Duration, message *domain.Message) error {
	l := log.With().
		Int("messageId", message.ID).
		Int64("chatId", message.ChatID).
		Str("command", t.GetCommand()).
		Logger()

	l.Info().Msg("handling request")

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	go t.textSender.SendChatAction(ctx, message.ChatID, domain.Typing)

	promptText, err := t.extractPrompt(ctx, message)
	if err != nil {
		log.Error().Err(err).Msg("failed to extract prompt text")
	}

	if promptText == "" {
		log.Debug().Msg(domain.ErrEmptyPrompt)
		return nil
	}

	thoughts, answer, err := t.textGenerator.ThinkFromPrompt(ctx, domain.Prompt{
		Prompt:   promptText,
		ImageURL: message.ImageURL,
		Author:   domain.User,
	})
	if err != nil {
		l.Error().Err(err).Msg("failed to generate prompt")

		_, err = t.textSender.SendMessageReply(ctx,
			message.ChatID,
			message.ID,
			fmt.Sprintf("failed to generate reply: %s", err))
		if err != nil {
			l.Error().Err(err).Msg(domain.ErrSendingReplyFailed)
			return err
		}

		return nil
	}

	sentID, err := t.textSender.SendMessageReply(ctx,
		message.ChatID,
		message.ID,
		fmt.Sprintf("Thinking:\n\n%s", thoughts))
	if err != nil {
		l.Error().Err(err).Msg(domain.ErrSendingReplyFailed)
		return err
	}

	_, err = t.textSender.SendMessageReply(ctx,
		message.ChatID,
		sentID,
		answer)
	if err != nil {
		l.Error().Err(err).Msg(domain.ErrSendingReplyFailed)
		return err
	}

	return nil
}

func (t *ThinkHandler) extractPrompt(ctx context.Context, message *domain.Message) (string, error) {
	l := log.With().
		Int("messageId", message.ID).
		Int64("chatId", message.ChatID).
		Str("command", t.GetCommand()).
		Logger()

	promptText := domain.ParseCommandArgs(message.Text)
	if promptText == "" {
		_, err := t.textSender.SendMessageReply(ctx, message.ChatID, message.ID, "please input a prompt")
		if err != nil {
			l.Error().Err(err).Msg(domain.ErrSendingReplyFailed)
			return "", err
		}
		return "", nil
	}

	if message.AudioURL != "" {
		transcript, err := t.transcriber.GenerateFromAudio(ctx, message.AudioURL)
		if err != nil {
			l.Error().Err(err).Msg(domain.ErrSendingReplyFailed)
			return "", err
		}

		promptText += ": " + transcript
	}

	return promptText, nil
}
