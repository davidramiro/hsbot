package generator

import (
	"context"
	"errors"
	"hsbot/internal/core/domain"
	"testing"

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
					Model:  domain.Model{Identifier: "openai/gpt-4.1"},
				},
			},
			mockResp: openrouter.ChatCompletionResponse{
				Choices: []openrouter.ChatCompletionChoice{{
					Message: openrouter.ChatCompletionMessage{
						Content: openrouter.Content{Text: "hello!"},
					},
				}},
				Model: "openai/gpt-4.1",
				Usage: openrouter.Usage{
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
					Model:  domain.Model{Identifier: "openai/gpt-4.1"},
				},
			},
			mockResp: openrouter.ChatCompletionResponse{
				Choices: []openrouter.ChatCompletionChoice{{
					Message: openrouter.ChatCompletionMessage{
						Content: openrouter.Content{Text: "hello!"},
					},
				}},
				Model: "openai/gpt-4.1",
				Usage: openrouter.Usage{
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
					Model:  domain.Model{Identifier: "openai/gpt-4.1"},
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
					Model:    domain.Model{Identifier: "openai/gpt-4.1"},
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
				Usage: openrouter.Usage{
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
				client:       mock,
				systemPrompt: tc.systemPrompt,
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
