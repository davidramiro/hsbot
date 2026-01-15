package generator

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"hsbot/internal/core/domain"
	"io"
	"net/http"
	"sort"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"

	"github.com/revrost/go-openrouter"
)

// OpenRouter wraps the OpenRouter API.
type OpenRouter struct {
	client        OpenRouterClient
	Models        []domain.Model
	defaultModels []domain.Model
	systemPrompt  string
}

// OpenRouterClient wraps all used methods from *openrouter.Client. Used for mocking in tests.
type OpenRouterClient interface {
	CreateChatCompletion(ctx context.Context,
		ccr openrouter.ChatCompletionRequest) (openrouter.ChatCompletionResponse, error)
}

func NewOpenRouter(apiKey, systemPrompt string) (*OpenRouter, error) {
	or := &OpenRouter{
		systemPrompt: systemPrompt,
		client: openrouter.NewClient(
			apiKey,
			openrouter.WithXTitle("hsbot"),
		),
	}

	var models []domain.Model
	err := viper.UnmarshalKey("openrouter.models", &models)
	if err != nil {
		log.Error().Err(err).Msg("failed to unmarshal openrouter models from config")
		return nil, err
	}

	sort.Slice(models, func(i, j int) bool {
		return models[i].Default < models[j].Default
	})

	var defaultModels []domain.Model
	for _, model := range models {
		if model.Default != 0 {
			defaultModels = append(defaultModels, model)
		}
	}

	if len(defaultModels) == 0 {
		return nil, errors.New("no default model found")
	}

	or.Models = models
	or.defaultModels = defaultModels

	return or, nil
}

func (o *OpenRouter) GenerateFromPrompt(
	ctx context.Context, prompts []domain.Prompt) (domain.ModelResponse, error) {
	messages := make([]openrouter.ChatCompletionMessage, len(prompts)+1)

	messages[0] = openrouter.ChatCompletionMessage{
		Role: openrouter.ChatMessageRoleSystem,
		Content: openrouter.Content{
			Text: o.systemPrompt,
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
			msg, err := createUserMessage(ctx, prompt)
			if err != nil {
				return domain.ModelResponse{}, fmt.Errorf("could not create openrouter response: %w", err)
			}
			messages[i+1] = msg
		}
	}

	latestPrompt := prompts[len(prompts)-1].Prompt
	model := o.findModelByMessage(&latestPrompt)
	prompts[len(prompts)-1].Prompt = latestPrompt

	ccr := openrouter.ChatCompletionRequest{
		Messages: messages,
		Usage: &openrouter.IncludeUsage{
			Include: true,
		},
		Model: model.Identifier,
	}

	return o.retryCompletion(ctx, ccr)
}

const ORProviderError = "Provider returned error"

func (o *OpenRouter) retryCompletion(ctx context.Context,
	ccr openrouter.ChatCompletionRequest) (domain.ModelResponse, error) {
	for i := -1; i < len(o.defaultModels); i++ {
		if ccr.Model == "" {
			// no specific model requested, start with first index from default models
			i = 0
		}

		// we're either on a retry or default model iteration
		if i != -1 {
			ccr.Model = o.defaultModels[i].Identifier
		}

		resp, err := o.client.CreateChatCompletion(ctx, ccr)
		if err != nil {
			if strings.Contains(err.Error(), ORProviderError) {
				continue
			}
			return domain.ModelResponse{}, fmt.Errorf("openrouter API error: %w", err)
		}

		return domain.ModelResponse{
			Response: resp.Choices[0].Message.Content.Text,
			Metadata: domain.ResponseMetadata{
				Model:            resp.Model,
				CompletionTokens: resp.Usage.CompletionTokens,
				TotalTokens:      resp.Usage.TotalTokens,
				Cost:             resp.Usage.Cost,
				Retries:          i,
			},
		}, nil
	}

	return domain.ModelResponse{},
		fmt.Errorf("failed to get a response from openrouter, retry count: %d", len(o.defaultModels)-1)
}

func createUserMessage(ctx context.Context, prompt domain.Prompt) (openrouter.ChatCompletionMessage, error) {
	if prompt.ImageURL != "" {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, prompt.ImageURL, nil)
		if err != nil {
			return openrouter.ChatCompletionMessage{}, fmt.Errorf("could not create image dl request: %w", err)
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return openrouter.ChatCompletionMessage{}, fmt.Errorf("could not download image: %w", err)
		}

		defer resp.Body.Close()

		data, err := io.ReadAll(resp.Body)
		if err != nil {
			return openrouter.ChatCompletionMessage{}, fmt.Errorf("could not read image bytes: %w", err)
		}

		// Detect the actual type (image/jpeg, image/png, etc.)
		mimeType := http.DetectContentType(data)

		// Encode to Base64
		encoded := base64.StdEncoding.EncodeToString(data)

		return openrouter.ChatCompletionMessage{
			Role: openrouter.ChatMessageRoleUser,
			Content: openrouter.Content{Multi: []openrouter.ChatMessagePart{
				{
					Type: openrouter.ChatMessagePartTypeImageURL,
					ImageURL: &openrouter.ChatMessageImageURL{URL: fmt.Sprintf("data:%s;base64,%s",
						mimeType, encoded)},
				},
				{
					Type: openrouter.ChatMessagePartTypeText,
					Text: prompt.Prompt,
				},
			},
			},
		}, nil
	}

	return openrouter.ChatCompletionMessage{
		Role: openrouter.ChatMessageRoleUser,
		Content: openrouter.Content{
			Text: prompt.Prompt,
		},
	}, nil
}

func (o *OpenRouter) findModelByMessage(message *string) domain.Model {
	for _, model := range o.Models {
		lowercaseMessage := strings.ToLower(*message)
		lowerCaseModel := strings.ToLower("#" + model.Keyword)
		if strings.Contains(lowercaseMessage, lowerCaseModel) {
			i := strings.Index(lowercaseMessage, lowerCaseModel)
			*message = (*message)[:i] + (*message)[i+len(lowerCaseModel):]
			return model
		}
	}

	return domain.Model{}
}
