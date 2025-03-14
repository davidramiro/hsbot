package generator

import (
	"context"
	"fmt"
	"hsbot/internal/adapters/file"
	"hsbot/internal/core/domain"

	"github.com/liushuangls/go-anthropic/v2"
)

const MaxTokens = 8192
const MaxThinkingTokens = 2048

type ClaudeGenerator struct {
	client       *anthropic.Client
	systemPrompt string
}

func NewClaudeGenerator(apiKey, systemPrompt string) *ClaudeGenerator {
	return &ClaudeGenerator{
		systemPrompt: systemPrompt,
		client:       anthropic.NewClient(apiKey),
	}
}

func (c *ClaudeGenerator) GenerateFromPrompt(ctx context.Context, prompts []domain.Prompt) (string, error) {
	messages := make([]anthropic.Message, len(prompts))

	for i, prompt := range prompts {
		if prompt.Author == domain.System {
			messages[i] = anthropic.NewAssistantTextMessage(prompt.Prompt)
		} else if prompt.Author == domain.User {
			message, err := createUserMessage(ctx, prompt)
			if err != nil {
				return "", err
			}
			messages[i] = message
		}
	}

	resp, err := c.client.CreateMessages(ctx, anthropic.MessagesRequest{
		Model:     anthropic.ModelClaude3Dot7SonnetLatest,
		System:    c.systemPrompt,
		Messages:  messages,
		MaxTokens: MaxTokens,
	})
	if err != nil {
		return "", fmt.Errorf("claude API error: %w", err)
	}

	return resp.Content[0].GetText(), nil
}

func (c *ClaudeGenerator) ThinkFromPrompt(ctx context.Context, prompt domain.Prompt) (string, string, error) {
	messages := make([]anthropic.Message, 1)

	message, err := createUserMessage(ctx, prompt)
	if err != nil {
		return "", "", err
	}

	messages[0] = message

	resp, err := c.client.CreateMessages(ctx, anthropic.MessagesRequest{
		Model:     anthropic.ModelClaude3Dot7SonnetLatest,
		Messages:  messages,
		MaxTokens: MaxTokens,
		Thinking: &anthropic.Thinking{
			Type:         anthropic.ThinkingTypeEnabled,
			BudgetTokens: MaxThinkingTokens,
		},
	})
	if err != nil {
		return "", "", fmt.Errorf("claude API error: %w", err)
	}

	var thoughts string
	var answer string

	for _, msg := range resp.Content {
		if msg.Type == anthropic.MessagesContentTypeThinking && msg.MessageContentThinking != nil {
			thoughts = msg.MessageContentThinking.Thinking
		} else if msg.Type == anthropic.MessagesContentTypeText {
			answer = msg.GetText()
		}
	}

	return thoughts, answer, nil
}

func createUserMessage(ctx context.Context, prompt domain.Prompt) (anthropic.Message, error) {
	if prompt.ImageURL != "" {
		f, err := file.DownloadFile(ctx, prompt.ImageURL)
		if err != nil {
			return anthropic.Message{}, fmt.Errorf("error downloading image: %w", err)
		}

		return anthropic.Message{
			Role: anthropic.RoleUser,
			Content: []anthropic.MessageContent{
				anthropic.NewImageMessageContent(anthropic.MessageContentSource{
					Type:      "base64",
					MediaType: "image/jpeg",
					Data:      f,
				}),
				anthropic.NewTextMessageContent(prompt.Prompt),
			},
		}, nil
	}

	return anthropic.NewUserTextMessage(prompt.Prompt), nil
}
