package generator

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"hsbot/internal/adapters/file"
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
	client             OpenRouterClient
	TextModels         []domain.Model
	defaultTextModels  []domain.Model
	imageModels        []domain.Model
	defaultImageModels []domain.Model
	voiceModels        []domain.Model
	ttsModel           string
	sttModel           string
	systemPrompt       string
}

// OpenRouterClient wraps all used methods from *openrouter.Client. Used for mocking in tests.
type OpenRouterClient interface {
	CreateChatCompletion(ctx context.Context,
		ccr openrouter.ChatCompletionRequest) (openrouter.ChatCompletionResponse, error)
	CreateSpeech(ctx context.Context, request openrouter.SpeechRequest) (openrouter.SpeechResponse, error)
	CreateTranscription(ctx context.Context,
		request openrouter.TranscriptionRequest) (openrouter.TranscriptionResponse, error)
	CreateImages(ctx context.Context,
		request openrouter.ImageGenerationRequest) (openrouter.ImageGenerationResponse, error)
}

func NewOpenRouter(apiKey, systemPrompt string) (*OpenRouter, error) {
	or := &OpenRouter{
		systemPrompt: systemPrompt,
		client: openrouter.NewClient(
			apiKey,
			openrouter.WithXTitle("hsbot"),
		),
	}

	models, defaultModels, err := unmarshalModels("openrouter.models")
	if err != nil {
		log.Error().Err(err).Msg("failed to unmarshal openrouter models from config")
		return nil, err
	}

	if len(defaultModels) == 0 {
		return nil, errors.New("no default model found")
	}

	or.TextModels = models
	or.defaultTextModels = defaultModels

	imageModels, defaultImageModels, err := unmarshalModels("openrouter.image_models")
	if err != nil {
		log.Error().Err(err).Msg("failed to unmarshal openrouter image models from config")
		return nil, err
	}

	if len(defaultImageModels) == 0 {
		return nil, errors.New("no default image model found")
	}

	or.imageModels = imageModels
	or.defaultImageModels = defaultImageModels

	or.ttsModel = viper.GetString("openrouter.tts_model")
	or.sttModel = viper.GetString("openrouter.stt_model")

	var voices []domain.Model
	err = viper.UnmarshalKey("openrouter.voices", &voices)
	if err != nil {
		log.Error().Err(err).Msg("failed to unmarshal openrouter voices from config")
		return nil, err
	}

	or.voiceModels = voices

	if len(or.voiceModels) == 0 {
		return nil, errors.New("no voice models found")
	}

	return or, nil
}

func unmarshalModels(key string) ([]domain.Model, []domain.Model, error) {
	var models []domain.Model
	if err := viper.UnmarshalKey(key, &models); err != nil {
		return nil, nil, err
	}

	sort.Slice(models, func(i, j int) bool {
		return models[i].Default < models[j].Default
	})

	var defaults []domain.Model
	for _, model := range models {
		if model.Default != 0 {
			defaults = append(defaults, model)
		}
	}

	return models, defaults, nil
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
	for i := -1; i < len(o.defaultTextModels); i++ {
		if ccr.Model == "" {
			// no specific model requested, start with first index from default models
			i = 0
		}

		// we're either on a retry or default model iteration
		if i != -1 {
			ccr.Model = o.defaultTextModels[i].Identifier
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
		fmt.Errorf("failed to get a response from openrouter, retry count: %d", len(o.defaultTextModels)-1)
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
	for _, model := range o.TextModels {
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

func (o *OpenRouter) findImageModel(message *string) domain.Model {
	for _, model := range o.imageModels {
		lowercaseMessage := strings.ToLower(*message)
		lowerCaseModel := strings.ToLower("#" + model.Keyword)
		if strings.Contains(lowercaseMessage, lowerCaseModel) {
			i := strings.Index(lowercaseMessage, lowerCaseModel)
			*message = (*message)[:i] + (*message)[i+len(lowerCaseModel):]
			return model
		}
	}

	if len(o.defaultImageModels) > 0 {
		return o.defaultImageModels[0]
	}

	return domain.Model{}
}

func (o *OpenRouter) GenerateImage(ctx context.Context, prompt string) (domain.GeneratedImage, error) {
	return o.createImage(ctx, prompt, "")
}

func (o *OpenRouter) EditImage(ctx context.Context, prompt domain.Prompt) (domain.GeneratedImage, error) {
	if prompt.Prompt == "" {
		return domain.GeneratedImage{}, errors.New("missing prompt")
	}

	if prompt.ImageURL == "" {
		return domain.GeneratedImage{}, errors.New("missing image")
	}

	return o.createImage(ctx, prompt.Prompt, prompt.ImageURL)
}

func (o *OpenRouter) createImage(ctx context.Context, prompt, imageURL string) (domain.GeneratedImage, error) {
	model := o.findImageModel(&prompt)
	if model.Identifier == "" {
		return domain.GeneratedImage{}, errors.New("no image model configured")
	}

	req := openrouter.ImageGenerationRequest{
		Model:        model.Identifier,
		Prompt:       prompt,
		OutputFormat: openrouter.ImageOutputFormatPng,
	}

	if imageURL != "" {
		req.InputReferences = []openrouter.ImageInputReference{{
			Type:     openrouter.ImageInputReferenceTypeImageURL,
			ImageURL: openrouter.ImageURLRef{URL: imageURL},
		}}
	}

	resp, err := o.client.CreateImages(ctx, req)
	if err != nil {
		return domain.GeneratedImage{}, fmt.Errorf("openrouter image API error: %w", err)
	}

	return imageFromResponse(resp)
}

func imageFromResponse(resp openrouter.ImageGenerationResponse) (domain.GeneratedImage, error) {
	if len(resp.Data) == 0 || resp.Data[0].B64JSON == "" {
		return domain.GeneratedImage{}, errors.New("no images returned from openrouter")
	}

	data, err := base64.StdEncoding.DecodeString(resp.Data[0].B64JSON)
	if err != nil {
		return domain.GeneratedImage{}, fmt.Errorf("failed to decode image: %w", err)
	}

	var cost float64
	if resp.Usage != nil && resp.Usage.Cost != nil {
		cost = *resp.Usage.Cost
	}

	return domain.GeneratedImage{Data: data, Cost: cost}, nil
}

func (o *OpenRouter) GenerateSpeech(ctx context.Context, text string) ([]byte, error) {
	if len(o.voiceModels) == 0 {
		return nil, errors.New("no voice models configured")
	}

	result, err := o.client.CreateSpeech(ctx, openrouter.SpeechRequest{
		Model:          o.ttsModel,
		Input:          text,
		Voice:          o.voiceModels[0].Identifier,
		ResponseFormat: openrouter.SpeechResponseFormatMp3,
	})
	if err != nil {
		return nil, fmt.Errorf("openrouter speech API error: %w", err)
	}

	return result.Audio, nil
}

func (o *OpenRouter) GenerateFromAudio(ctx context.Context, url string) (string, error) {
	f, err := file.DownloadFile(ctx, url)
	if err != nil {
		return "", fmt.Errorf("failed to download audio: %w", err)
	}

	resp, err := o.client.CreateTranscription(ctx, openrouter.TranscriptionRequest{
		Model:      o.sttModel,
		InputAudio: openrouter.NewTranscriptionInputAudio(f, openrouter.AudioFormatOgg),
	})
	if err != nil {
		return "", fmt.Errorf("openrouter transcription API error: %w", err)
	}

	log.Debug().Interface("result", resp.Text).Msg("openrouter transcript")

	return resp.Text, nil
}
