package generator

import (
	"context"
	"fmt"
	"hsbot/internal/adapters/file"
	"hsbot/internal/core/domain"

	"github.com/liushuangls/go-anthropic/v2"
)

const MaxTokens = 5000

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
	var messages []anthropic.Message

	for _, prompt := range prompts {
		if prompt.Author == domain.System {
			messages = append(messages, anthropic.NewAssistantTextMessage(prompt.Prompt))
		} else if prompt.Author == domain.User {
			message, err := createUserMessage(ctx, prompt)
			if err != nil {
				return "", err
			}
			messages = append(messages, message)
		}
	}

	resp, err := c.client.CreateMessages(ctx, anthropic.MessagesRequest{
		Model:     anthropic.ModelClaude3Dot5Sonnet20240620,
		System:    c.systemPrompt,
		Messages:  messages,
		MaxTokens: MaxTokens,
	})
	if err != nil {
		return "", fmt.Errorf("claude API error: %w", err)
	}

	return resp.Content[0].GetText(), nil
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
				anthropic.NewImageMessageContent(anthropic.MessageContentImageSource{
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
