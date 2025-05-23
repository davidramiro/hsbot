package commands

import (
	"context"
	"fmt"
	"hsbot/internal/core/domain"
	"hsbot/internal/core/port"
	"strings"
	"time"
)

type ModelHandler struct {
	ch      *ChatHandler
	ts      port.TextSender
	command string
}

func NewModelHandler(ch *ChatHandler, ts port.TextSender, command string) *ModelHandler {
	return &ModelHandler{
		ch:      ch,
		ts:      ts,
		command: command,
	}
}

func (m *ModelHandler) GetCommand() string {
	return m.command
}

func (m *ModelHandler) Respond(ctx context.Context, _ time.Duration, message *domain.Message) error {
	models := m.ch.models

	sb := &strings.Builder{}

	_, err := sb.WriteString("hsbot is multimodal. You can choose the LLM you want to interact with by " +
		"adding a #keyword to your prompts in /chat mode. Here's a list of currently active models:\n\n")
	if err != nil {
		return fmt.Errorf("failed to construct response: %w", err)
	}

	for _, model := range models {
		_, err = fmt.Fprintf(sb, "Model: %s, Keyword: %s\n", model.Identifier, model.Keyword)
		if err != nil {
			return fmt.Errorf("failed to construct response: %w", err)
		}
	}

	_, err = sb.WriteString("\nKeep in mind that not every model has image recognition capabilities.")
	if err != nil {
		return fmt.Errorf("failed to construct response: %w", err)
	}

	_, err = m.ts.SendMessageReply(ctx, message, sb.String())
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	return nil
}
