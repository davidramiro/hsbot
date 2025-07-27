package generator

import (
	"context"
	"errors"
	"hsbot/internal/core/domain"
	"testing"

	"github.com/spf13/viper"

	"github.com/stretchr/testify/require"

	"github.com/revrost/go-openrouter"
	"github.com/stretchr/testify/assert"
)

// mockClient is a test double for the OpenRouterClient interface.
type mockClient struct {
	createChatCompletionFunc func(ctx context.Context,
		ccr openrouter.ChatCompletionRequest) (openrouter.ChatCompletionResponse, error)
}

func (m *mockClient) CreateChatCompletion(ctx context.Context,
	ccr openrouter.ChatCompletionRequest) (openrouter.ChatCompletionResponse, error) {
	return m.createChatCompletionFunc(ctx, ccr)
}

func TestNewOpenRouter(t *testing.T) {
	orig := viper.Get("openrouter.models")
	defer viper.Set("openrouter.models", orig)

	mockedModels := []map[string]interface{}{
		{"Keyword": "gpt", "Identifier": "openai/gpt-4.1", "Default": 1},
		{"Keyword": "claude", "Identifier": "anthropic/claude-sonnet-4", "Default": 2},
	}

	var expectedModels []domain.Model
	for _, m := range mockedModels {
		expectedModels = append(expectedModels, domain.Model{
			Keyword:    m["Keyword"].(string),
			Identifier: m["Identifier"].(string),
			Default:    m["Default"].(int),
		})
	}
	viper.Set("openrouter.models", expectedModels)

	apiKey := "fakeApiKey"
	systemPrompt := "system test"
	or, err := NewOpenRouter(apiKey, systemPrompt)

	require.NoError(t, err)
	assert.NotNil(t, or)
	assert.Len(t, or.Models, 2)
	assert.Equal(t, expectedModels, or.Models)
	assert.Equal(t, systemPrompt, or.systemPrompt)
}

func TestOpenRouterGenerator_GenerateFromPrompt(t *testing.T) {
	testCases := []struct {
		name         string
		systemPrompt string
		prompts      []domain.Prompt
		mockResp     openrouter.ChatCompletionResponse
		mockErr      error
		expectedResp domain.ModelResponse
		expectErr    bool
	}{
		{
			name:         "success, single user prompt",
			systemPrompt: "system",
			prompts: []domain.Prompt{
				{
					Prompt: "hi",
					Author: domain.User,
				},
			},
			mockResp: openrouter.ChatCompletionResponse{
				Choices: []openrouter.ChatCompletionChoice{{
					Message: openrouter.ChatCompletionMessage{
						Content: openrouter.Content{Text: "hello!"},
					},
				}},
				Model: "openai/gpt-4.1",
				Usage: &openrouter.Usage{
					CompletionTokens: 7,
					TotalTokens:      9,
				},
			},
			expectedResp: domain.ModelResponse{
				Response: "hello!",
				Metadata: domain.ResponseMetadata{
					Model:            "openai/gpt-4.1",
					CompletionTokens: 7,
					TotalTokens:      9,
				},
			},
			expectErr: false,
		},
		{
			name:         "success, user and system prompt",
			systemPrompt: "system",
			prompts: []domain.Prompt{
				{
					Prompt: "i'm an assistant.",
					Author: domain.System,
				},
				{
					Prompt: "hi",
					Author: domain.User,
				},
			},
			mockResp: openrouter.ChatCompletionResponse{
				Choices: []openrouter.ChatCompletionChoice{{
					Message: openrouter.ChatCompletionMessage{
						Content: openrouter.Content{Text: "hello!"},
					},
				}},
				Model: "openai/gpt-4.1",
				Usage: &openrouter.Usage{
					CompletionTokens: 7,
					TotalTokens:      9,
				},
			},
			expectedResp: domain.ModelResponse{
				Response: "hello!",
				Metadata: domain.ResponseMetadata{
					Model:            "openai/gpt-4.1",
					CompletionTokens: 7,
					TotalTokens:      9,
				},
			},
			expectErr: false,
		},
		{
			name:         "API error returned",
			systemPrompt: "system",
			prompts: []domain.Prompt{
				{
					Prompt: "fail",
					Author: domain.User,
				},
			},
			mockErr:   errors.New("api failure"),
			expectErr: true,
		},
		{
			name:         "prompt with image",
			systemPrompt: "system",
			prompts: []domain.Prompt{
				{
					Prompt:   "describe this",
					Author:   domain.User,
					ImageURL: "http://image",
				},
			},
			mockResp: openrouter.ChatCompletionResponse{
				Choices: []openrouter.ChatCompletionChoice{{
					Message: openrouter.ChatCompletionMessage{
						Content: openrouter.Content{Text: "It's a cat."},
					},
				}},
				Model: "openai/gpt-4.1",
				Usage: &openrouter.Usage{
					CompletionTokens: 4,
					TotalTokens:      10,
				},
			},
			expectedResp: domain.ModelResponse{
				Response: "It's a cat.",
				Metadata: domain.ResponseMetadata{
					Model:            "openai/gpt-4.1",
					CompletionTokens: 4,
					TotalTokens:      10,
				},
			},
			expectErr: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mock := &mockClient{
				createChatCompletionFunc: func(_ context.Context,
					_ openrouter.ChatCompletionRequest) (openrouter.ChatCompletionResponse, error) {
					return tc.mockResp, tc.mockErr
				},
			}
			gen := &OpenRouter{
				client:        mock,
				systemPrompt:  tc.systemPrompt,
				Models:        []domain.Model{{Keyword: "gpt", Identifier: "gpt", Default: 1}},
				defaultModels: []domain.Model{{Keyword: "gpt", Identifier: "gpt", Default: 1}},
			}
			resp, err := gen.GenerateFromPrompt(t.Context(), tc.prompts)
			if tc.expectErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.expectedResp, resp)
			}
		})
	}
}

func TestOpenRouter_RetryCompletion(t *testing.T) {
	defaultModels := []domain.Model{
		{Identifier: "model1", Default: 1},
		{Identifier: "model2", Default: 2},
	}
	systemPrompt := "system"

	successResp := openrouter.ChatCompletionResponse{
		Choices: []openrouter.ChatCompletionChoice{{
			Message: openrouter.ChatCompletionMessage{
				Content: openrouter.Content{Text: "retry success!"},
			},
		}},
		Model: "model1",
		Usage: &openrouter.Usage{
			CompletionTokens: 3,
			TotalTokens:      5,
		},
	}

	tests := []struct {
		name        string
		failures    int
		errorType   string
		wantErr     string
		wantRetries int
	}{
		{
			name:        "Succeeds after provider error retry",
			failures:    1,
			errorType:   "provider",
			wantErr:     "",
			wantRetries: 1,
		},
		{
			name:        "Abort on non-provider error immediately",
			failures:    1,
			errorType:   "other",
			wantErr:     "some other error",
			wantRetries: 0,
		},
		{
			name:        "Exhaust retries on persistent provider error",
			failures:    len(defaultModels) + 1,
			errorType:   "provider",
			wantErr:     "failed to get a response from openrouter, retry count: 1",
			wantRetries: len(defaultModels),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			attempt := 0
			mock := &mockClient{
				createChatCompletionFunc: func(_ context.Context, _ openrouter.ChatCompletionRequest) (openrouter.ChatCompletionResponse, error) {
					attempt++
					if attempt <= tt.failures {
						if tt.errorType == "provider" {
							return openrouter.ChatCompletionResponse{}, errors.New("Provider returned error")
						}
						return openrouter.ChatCompletionResponse{}, errors.New("some other error")
					}
					return successResp, nil
				},
			}

			gen := &OpenRouter{
				client:        mock,
				systemPrompt:  systemPrompt,
				Models:        defaultModels,
				defaultModels: defaultModels,
			}
			req := openrouter.ChatCompletionRequest{
				Model:    defaultModels[0].Identifier,
				Messages: []openrouter.ChatCompletionMessage{},
				Usage:    &openrouter.IncludeUsage{Include: true},
			}
			resp, err := gen.retryCompletion(t.Context(), req)
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, "retry success!", resp.Response)
				assert.Equal(t, "model1", resp.Metadata.Model)
			}
			assert.Equal(t, tt.wantRetries, attempt-1)
		})
	}
}

func TestFindModelByMessage(t *testing.T) {
	models := []domain.Model{
		{Keyword: "gpt"},
		{Keyword: "claude"},
		{Keyword: "default", Default: 1},
	}

	handler := &OpenRouter{
		Models: models,
	}

	tests := []struct {
		name        string
		message     string
		wantModel   domain.Model
		wantMessage string
	}{
		{
			name:        "Match GPT model keyword (case-insensitive)",
			message:     "Hello #GPT",
			wantModel:   models[0],
			wantMessage: "Hello ",
		},
		{
			name:        "Match Claude model keyword",
			message:     "Please use #claude for this",
			wantModel:   models[1],
			wantMessage: "Please use  for this",
		},
		{
			name:        "No keyword, fallback to empty",
			message:     "Just a normal message",
			wantModel:   domain.Model{},
			wantMessage: "Just a normal message",
		},
		{
			name:        "Multiple keywords, first match is returned",
			message:     "#gpt and #claude in text",
			wantModel:   models[0],
			wantMessage: " and #claude in text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := tt.message
			gotModel := handler.findModelByMessage(&msg)
			assert.Equal(t, tt.wantModel, gotModel)
			assert.Equal(t, tt.wantMessage, msg)
		})
	}
}
