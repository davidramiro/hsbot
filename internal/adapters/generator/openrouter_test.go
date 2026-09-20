package generator

import (
	"context"
	"errors"
	"hsbot/internal/core/domain"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/revrost/go-openrouter"
	"github.com/stretchr/testify/assert"
)

// mockClient is a test double for the OpenRouterClient interface.
type mockClient struct {
	createChatCompletionFunc func(ctx context.Context,
		ccr openrouter.ChatCompletionRequest) (openrouter.ChatCompletionResponse, error)
	createSpeechFunc func(ctx context.Context,
		request openrouter.SpeechRequest) (openrouter.SpeechResponse, error)
	createTranscriptionFunc func(ctx context.Context,
		request openrouter.TranscriptionRequest) (openrouter.TranscriptionResponse, error)
	createImagesFunc func(ctx context.Context,
		request openrouter.ImageGenerationRequest) (openrouter.ImageGenerationResponse, error)
}

func (m *mockClient) CreateChatCompletion(ctx context.Context,
	ccr openrouter.ChatCompletionRequest) (openrouter.ChatCompletionResponse, error) {
	return m.createChatCompletionFunc(ctx, ccr)
}

func (m *mockClient) CreateSpeech(ctx context.Context,
	request openrouter.SpeechRequest) (openrouter.SpeechResponse, error) {
	if m.createSpeechFunc == nil {
		return openrouter.SpeechResponse{}, nil
	}
	return m.createSpeechFunc(ctx, request)
}

func (m *mockClient) CreateTranscription(ctx context.Context,
	request openrouter.TranscriptionRequest) (openrouter.TranscriptionResponse, error) {
	if m.createTranscriptionFunc == nil {
		return openrouter.TranscriptionResponse{}, nil
	}
	return m.createTranscriptionFunc(ctx, request)
}

func (m *mockClient) CreateImages(ctx context.Context,
	request openrouter.ImageGenerationRequest) (openrouter.ImageGenerationResponse, error) {
	if m.createImagesFunc == nil {
		return openrouter.ImageGenerationResponse{}, nil
	}
	return m.createImagesFunc(ctx, request)
}

func TestNewOpenRouter(t *testing.T) {
	expectedModels := []domain.Model{
		{Keyword: "gpt", Identifier: "openai/gpt-4.1", Default: 1},
		{Keyword: "claude", Identifier: "anthropic/claude-sonnet-4", Default: 2},
	}

	or, err := NewOpenRouter(Config{
		APIKey:       "fakeApiKey",
		SystemPrompt: "system test",
		TextModels:   expectedModels,
		ImageModels: []domain.Model{
			{Keyword: "flash", Identifier: "google/gemini-2.5-flash-image", Default: 1},
		},
		Voices: []domain.Model{
			{Keyword: "default", Identifier: "voice-id", Default: 1},
		},
		TTSModel: "tts-model",
		STTModel: "stt-model",
	})

	require.NoError(t, err)
	assert.NotNil(t, or)
	assert.Len(t, or.TextModels, 2)
	assert.Equal(t, expectedModels, or.TextModels)
	assert.Equal(t, "system test", or.systemPrompt)
}

func TestOpenRouterGenerator_GenerateFromPrompt(t *testing.T) {
	imgSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "image/jpeg")
		_, _ = w.Write([]byte{0xff, 0xd8, 0xff})
	}))
	t.Cleanup(imgSrv.Close)

	testCases := []struct {
		name         string
		systemPrompt string
		prompts      []domain.Prompt
		mockResp     openrouter.ChatCompletionResponse
		mockErr      error
		expectedResp domain.GeneratedText
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
			expectedResp: domain.GeneratedText{
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
			expectedResp: domain.GeneratedText{
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
					ImageURL: imgSrv.URL,
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
			expectedResp: domain.GeneratedText{
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
				client:            mock,
				systemPrompt:      tc.systemPrompt,
				TextModels:        []domain.Model{{Keyword: "gpt", Identifier: "gpt", Default: 1}},
				defaultTextModels: []domain.Model{{Keyword: "gpt", Identifier: "gpt", Default: 1}},
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
				client:            mock,
				systemPrompt:      systemPrompt,
				TextModels:        defaultModels,
				defaultTextModels: defaultModels,
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
		TextModels: models,
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

func TestOpenRouter_GenerateSpeech(t *testing.T) {
	tests := []struct {
		name    string
		audio   []byte
		mockErr error
		wantErr bool
	}{
		{
			name:  "success",
			audio: []byte("mp3-bytes"),
		},
		{
			name:    "api error",
			mockErr: errors.New("speech failed"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockClient{
				createSpeechFunc: func(_ context.Context, req openrouter.SpeechRequest) (openrouter.SpeechResponse, error) {
					assert.Equal(t, "tts-model", req.Model)
					assert.Equal(t, "hello", req.Input)
					assert.Equal(t, "voice-id", req.Voice)
					assert.Equal(t, openrouter.SpeechResponseFormatMp3, req.ResponseFormat)
					return openrouter.SpeechResponse{Audio: tt.audio}, tt.mockErr
				},
			}
			gen := &OpenRouter{
				client:      mock,
				ttsModel:    "tts-model",
				voiceModels: []domain.Model{{Keyword: "default", Identifier: "voice-id"}},
			}

			got, err := gen.Speak(t.Context(), "hello")
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.audio, got.Data)
		})
	}
}

func TestOpenRouter_GenerateImage(t *testing.T) {
	cost := 0.011
	mock := &mockClient{
		createImagesFunc: func(_ context.Context, req openrouter.ImageGenerationRequest) (openrouter.ImageGenerationResponse, error) {
			assert.Equal(t, "google/gemini-2.5-flash-image", req.Model)
			assert.Equal(t, "a cat", req.Prompt)
			assert.Empty(t, req.InputReferences)
			return openrouter.ImageGenerationResponse{
				Data:  []openrouter.ImageGenerationData{{B64JSON: "aW1hZ2U="}},
				Usage: &openrouter.ImageGenerationUsage{Cost: &cost},
			}, nil
		},
	}

	gen := &OpenRouter{
		client: mock,
		defaultImageModels: []domain.Model{
			{Keyword: "flash", Identifier: "google/gemini-2.5-flash-image", Default: 1},
		},
	}

	got, err := gen.NewImage(t.Context(), "a cat")
	require.NoError(t, err)
	assert.Equal(t, []byte("image"), got.Data)
	assert.InDelta(t, 0.011, got.Cost, 1e-9)
}

func TestOpenRouter_GenerateImageKeyword(t *testing.T) {
	mock := &mockClient{
		createImagesFunc: func(_ context.Context, req openrouter.ImageGenerationRequest) (openrouter.ImageGenerationResponse, error) {
			assert.Equal(t, "openai/gpt-image-1", req.Model)
			assert.Equal(t, " a cat", req.Prompt)
			return openrouter.ImageGenerationResponse{
				Data: []openrouter.ImageGenerationData{{B64JSON: "aW1hZ2U="}},
			}, nil
		},
	}

	gen := &OpenRouter{
		client: mock,
		imageModels: []domain.Model{
			{Keyword: "gpt", Identifier: "openai/gpt-image-1"},
		},
		defaultImageModels: []domain.Model{
			{Keyword: "flash", Identifier: "google/gemini-2.5-flash-image", Default: 1},
		},
	}

	got, err := gen.NewImage(t.Context(), "#gpt a cat")
	require.NoError(t, err)
	assert.Equal(t, []byte("image"), got.Data)
}

func TestOpenRouter_EditImage(t *testing.T) {
	mock := &mockClient{
		createImagesFunc: func(_ context.Context, req openrouter.ImageGenerationRequest) (openrouter.ImageGenerationResponse, error) {
			assert.Equal(t, "google/gemini-2.5-flash-image", req.Model)
			assert.Equal(t, "make it night", req.Prompt)
			require.Len(t, req.InputReferences, 1)
			assert.Equal(t, "https://img.example/cat.png", req.InputReferences[0].ImageURL.URL)
			return openrouter.ImageGenerationResponse{
				Data: []openrouter.ImageGenerationData{{B64JSON: "aW1hZ2U="}},
			}, nil
		},
	}

	gen := &OpenRouter{
		client: mock,
		defaultImageModels: []domain.Model{
			{Keyword: "flash", Identifier: "google/gemini-2.5-flash-image", Default: 1},
		},
	}

	got, err := gen.EditImage(t.Context(), domain.Prompt{
		Prompt:   "make it night",
		ImageURL: "https://img.example/cat.png",
	})
	require.NoError(t, err)
	assert.Equal(t, []byte("image"), got.Data)
}

func TestOpenRouter_EditImageMissing(t *testing.T) {
	gen := &OpenRouter{}

	_, err := gen.EditImage(t.Context(), domain.Prompt{ImageURL: "https://img.example/cat.png"})
	require.Error(t, err)

	_, err = gen.EditImage(t.Context(), domain.Prompt{Prompt: "edit"})
	require.Error(t, err)
}

func TestOpenRouter_GenerateImageAPIError(t *testing.T) {
	mock := &mockClient{
		createImagesFunc: func(_ context.Context, _ openrouter.ImageGenerationRequest) (openrouter.ImageGenerationResponse, error) {
			return openrouter.ImageGenerationResponse{}, errors.New("boom")
		},
	}
	gen := &OpenRouter{
		client: mock,
		defaultImageModels: []domain.Model{
			{Keyword: "flash", Identifier: "google/gemini-2.5-flash-image", Default: 1},
		},
	}

	_, err := gen.NewImage(t.Context(), "a cat")
	require.Error(t, err)
}

func TestNewOpenRouterErrors(t *testing.T) {
	valid := Config{
		TextModels:  []domain.Model{{Keyword: "gpt", Identifier: "gpt", Default: 1}},
		ImageModels: []domain.Model{{Keyword: "flash", Identifier: "img", Default: 1}},
		Voices:      []domain.Model{{Keyword: "v", Identifier: "voice"}},
	}

	t.Run("no default text model", func(t *testing.T) {
		cfg := valid
		cfg.TextModels = []domain.Model{{Keyword: "gpt", Identifier: "gpt"}}
		_, err := NewOpenRouter(cfg)
		require.EqualError(t, err, "no default model found")
	})

	t.Run("no default image model", func(t *testing.T) {
		cfg := valid
		cfg.ImageModels = []domain.Model{{Keyword: "flash", Identifier: "img"}}
		_, err := NewOpenRouter(cfg)
		require.EqualError(t, err, "no default image model found")
	})

	t.Run("no voices", func(t *testing.T) {
		cfg := valid
		cfg.Voices = nil
		_, err := NewOpenRouter(cfg)
		require.EqualError(t, err, "no voice models found")
	})
}

func TestOpenRouter_ListTextModels(t *testing.T) {
	models := []domain.Model{{Keyword: "gpt", Identifier: "gpt"}}
	or := &OpenRouter{TextModels: models}
	assert.Equal(t, models, or.ListTextModels())
}

func TestOpenRouter_GenerateFromPromptImageDownloadError(t *testing.T) {
	gen := &OpenRouter{
		client:            &mockClient{},
		TextModels:        []domain.Model{{Keyword: "gpt", Identifier: "gpt", Default: 1}},
		defaultTextModels: []domain.Model{{Keyword: "gpt", Identifier: "gpt", Default: 1}},
	}

	_, err := gen.GenerateFromPrompt(t.Context(), []domain.Prompt{{
		Author:   domain.User,
		Prompt:   "see",
		ImageURL: "://bad",
	}})
	require.Error(t, err)
	assert.ErrorContains(t, err, "could not create openrouter response")
}

func TestImageFromResponseErrors(t *testing.T) {
	_, err := imageFromResponse(openrouter.ImageGenerationResponse{})
	require.EqualError(t, err, "no images returned from openrouter")

	_, err = imageFromResponse(openrouter.ImageGenerationResponse{
		Data: []openrouter.ImageGenerationData{{B64JSON: "not-base64"}},
	})
	require.Error(t, err)
	assert.ErrorContains(t, err, "failed to decode image")
}

func TestOpenRouter_CreateImageNoModel(t *testing.T) {
	gen := &OpenRouter{}
	_, err := gen.NewImage(t.Context(), "a cat")
	require.EqualError(t, err, "no image model configured")
}

func TestOpenRouter_SpeakNoVoices(t *testing.T) {
	gen := &OpenRouter{}
	_, err := gen.Speak(t.Context(), "hi")
	require.EqualError(t, err, "no voice models configured")
}

func TestOpenRouter_Transcribe(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ogg"))
	}))
	t.Cleanup(srv.Close)

	cost := 0.02
	mock := &mockClient{
		createTranscriptionFunc: func(_ context.Context, req openrouter.TranscriptionRequest) (openrouter.TranscriptionResponse, error) {
			assert.Equal(t, "stt-model", req.Model)
			return openrouter.TranscriptionResponse{
				Text:  "hello",
				Usage: &openrouter.TranscriptionUsage{Cost: cost},
			}, nil
		},
	}
	gen := &OpenRouter{client: mock, sttModel: "stt-model"}

	got, err := gen.Transcribe(t.Context(), srv.URL)
	require.NoError(t, err)
	assert.Equal(t, "hello", got.Response)
	assert.InDelta(t, cost, got.Metadata.Cost, 1e-9)
}

func TestOpenRouter_TranscribeDownloadError(t *testing.T) {
	gen := &OpenRouter{}
	_, err := gen.Transcribe(t.Context(), "://bad")
	require.Error(t, err)
	assert.ErrorContains(t, err, "failed to download audio")
}

func TestOpenRouter_TranscribeAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ogg"))
	}))
	t.Cleanup(srv.Close)

	mock := &mockClient{
		createTranscriptionFunc: func(_ context.Context, _ openrouter.TranscriptionRequest) (openrouter.TranscriptionResponse, error) {
			return openrouter.TranscriptionResponse{}, errors.New("stt fail")
		},
	}
	gen := &OpenRouter{client: mock, sttModel: "stt-model"}

	_, err := gen.Transcribe(t.Context(), srv.URL)
	require.Error(t, err)
	assert.ErrorContains(t, err, "openrouter transcription API error")
}
