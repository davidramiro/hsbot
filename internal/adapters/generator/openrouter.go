package generator

import (
	"context"
	"fmt"
	"hsbot/internal/core/domain"

	"github.com/revrost/go-openrouter"
)

type OpenRouterGenerator struct {
	client       *openrouter.Client
	systemPrompt string
}

func NewOpenRouterGenerator(apiKey, systemPrompt string) *OpenRouterGenerator {
	return &OpenRouterGenerator{
		systemPrompt: systemPrompt,
		client: openrouter.NewClient(
			apiKey,
			openrouter.WithXTitle("hsbot"),
		),
	}
}

func (c *OpenRouterGenerator) GenerateFromPrompt(
	ctx context.Context, prompts []domain.Prompt) (domain.ModelResponse, error) {
	messages := make([]openrouter.ChatCompletionMessage, len(prompts)+1)

	messages[0] = openrouter.ChatCompletionMessage{
		Role: openrouter.ChatMessageRoleSystem,
		Content: openrouter.Content{
			Text: c.systemPrompt,
		},
	}

	for i, prompt := range prompts {
		switch prompt.Author {
		case domain.System:
			messages[i+1] = openrouter.ChatCompletionMessage{
				Role: openrouter.ChatMessageRoleAssistant,
				Content: openrouter.Content{
					Text: prompt.Prompt,
				},
			}
		case domain.User:
			messages[i+1] = createUserMessage(prompt)
		}
	}

	ccr := openrouter.ChatCompletionRequest{
		Messages: messages,
		Model:    prompts[len(prompts)-1].Model.Identifier,
	}

	resp, err := c.client.CreateChatCompletion(ctx, ccr)
	if err != nil {
		return domain.ModelResponse{}, fmt.Errorf("openrouter API error: %w", err)
	}

	return domain.ModelResponse{
		Response: resp.Choices[0].Message.Content.Text,
		Metadata: domain.ResponseMetadata{
			Model:            resp.Model,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		},
	}, nil
}

func createUserMessage(prompt domain.Prompt) openrouter.ChatCompletionMessage {
	if prompt.ImageURL != "" {
		return openrouter.ChatCompletionMessage{
			Role: openrouter.ChatMessageRoleUser,
			Content: openrouter.Content{Multi: []openrouter.ChatMessagePart{
				{
					Type:     openrouter.ChatMessagePartTypeImageURL,
					ImageURL: &openrouter.ChatMessageImageURL{URL: prompt.ImageURL},
				},
				{
					Type: openrouter.ChatMessagePartTypeText,
					Text: prompt.Prompt,
				},
			},
			},
		}
	}

	return openrouter.ChatCompletionMessage{
		Role: openrouter.ChatMessageRoleUser,
		Content: openrouter.Content{
			Text: prompt.Prompt,
		},
	}
}
