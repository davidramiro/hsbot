package generator

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"hsbot/internal/adapters/file"
	"hsbot/internal/core/domain"
	"net/http"
	"sort"
	"strings"

	"github.com/rs/zerolog/log"

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

type Config struct {
	APIKey       string
	SystemPrompt string
	TextModels   []domain.Model
	ImageModels  []domain.Model
	Voices       []domain.Model
	TTSModel     string
	STTModel     string
}

func NewOpenRouter(cfg Config) (*OpenRouter, error) {
	textModels, defaultTextModels := splitDefaults(cfg.TextModels)
	if len(defaultTextModels) == 0 {
		return nil, errors.New("no default model found")
	}

	imageModels, defaultImageModels := splitDefaults(cfg.ImageModels)
	if len(defaultImageModels) == 0 {
		return nil, errors.New("no default image model found")
	}

	if len(cfg.Voices) == 0 {
		return nil, errors.New("no voice models found")
	}

	return &OpenRouter{
		systemPrompt: cfg.SystemPrompt,
		client: openrouter.NewClient(
			cfg.APIKey,
			openrouter.WithXTitle("hsbot"),
		),
		TextModels:         textModels,
		defaultTextModels:  defaultTextModels,
		imageModels:        imageModels,
		defaultImageModels: defaultImageModels,
		voiceModels:        cfg.Voices,
		ttsModel:           cfg.TTSModel,
		sttModel:           cfg.STTModel,
	}, nil
}

func splitDefaults(models []domain.Model) ([]domain.Model, []domain.Model) {
	sort.Slice(models, func(i, j int) bool {
		return models[i].Default < models[j].Default
	})

	var defaults []domain.Model
	for _, model := range models {
		if model.Default != 0 {
			defaults = append(defaults, model)
		}
	}

	return models, defaults
}

func (o *OpenRouter) ListTextModels() []domain.Model {
	return o.TextModels
}

func (o *OpenRouter) GenerateFromPrompt(
	ctx context.Context, prompts []domain.Prompt) (domain.GeneratedText, error) {
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
				return domain.GeneratedText{}, fmt.Errorf("could not create openrouter response: %w", err)
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
	ccr openrouter.ChatCompletionRequest) (domain.GeneratedText, error) {
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
			return domain.GeneratedText{}, fmt.Errorf("openrouter API error: %w", err)
		}

		return domain.GeneratedText{
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

	return domain.GeneratedText{},
		fmt.Errorf("failed to get a response from openrouter, retry count: %d", len(o.defaultTextModels)-1)
}

func createUserMessage(ctx context.Context, prompt domain.Prompt) (openrouter.ChatCompletionMessage, error) {
	if prompt.ImageURL == "" {
		return openrouter.ChatCompletionMessage{
			Role: openrouter.ChatMessageRoleUser,
			Content: openrouter.Content{
				Text: prompt.Prompt,
			},
		}, nil
	}

	data, err := file.DownloadFile(ctx, prompt.ImageURL)
	if err != nil {
		return openrouter.ChatCompletionMessage{}, fmt.Errorf("could not download image: %w", err)
	}

	mimeType := http.DetectContentType(data)
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

func findModelByKeyword(models []domain.Model, message *string, fallback domain.Model) domain.Model {
	lowercaseMessage := strings.ToLower(*message)
	for _, model := range models {
		lowerCaseModel := strings.ToLower("#" + model.Keyword)
		if i := strings.Index(lowercaseMessage, lowerCaseModel); i != -1 {
			*message = (*message)[:i] + (*message)[i+len(lowerCaseModel):]
			return model
		}
	}

	return fallback
}

func (o *OpenRouter) findModelByMessage(message *string) domain.Model {
	return findModelByKeyword(o.TextModels, message, domain.Model{})
}

func (o *OpenRouter) findImageModel(message *string) domain.Model {
	fallback := domain.Model{}
	if len(o.defaultImageModels) > 0 {
		fallback = o.defaultImageModels[0]
	}
	return findModelByKeyword(o.imageModels, message, fallback)
}

func (o *OpenRouter) NewImage(ctx context.Context, prompt string) (domain.GeneratedImage, error) {
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

// TODO: openrouter does not return usage on speech api yet. ballpark...
const speechCost = 0.001

func (o *OpenRouter) Speak(ctx context.Context, text string) (domain.GeneratedAudio, error) {
	if len(o.voiceModels) == 0 {
		return domain.GeneratedAudio{}, errors.New("no voice models configured")
	}

	result, err := o.client.CreateSpeech(ctx, openrouter.SpeechRequest{
		Model:          o.ttsModel,
		Input:          text,
		Voice:          o.voiceModels[0].Identifier,
		ResponseFormat: openrouter.SpeechResponseFormatMp3,
	})
	if err != nil {
		return domain.GeneratedAudio{}, fmt.Errorf("openrouter speech API error: %w", err)
	}

	return domain.GeneratedAudio{
		Data: result.Audio,
		Cost: speechCost,
	}, nil
}

func (o *OpenRouter) Transcribe(ctx context.Context, url string) (domain.GeneratedText, error) {
	f, err := file.DownloadFile(ctx, url)
	if err != nil {
		return domain.GeneratedText{}, fmt.Errorf("failed to download audio: %w", err)
	}

	res, err := o.client.CreateTranscription(ctx, openrouter.TranscriptionRequest{
		Model:      o.sttModel,
		InputAudio: openrouter.NewTranscriptionInputAudio(f, openrouter.AudioFormatOgg),
	})
	if err != nil {
		return domain.GeneratedText{}, fmt.Errorf("openrouter transcription API error: %w", err)
	}

	resp := domain.GeneratedText{Response: res.Text}

	if res.Usage != nil {
		resp.Metadata.Cost = res.Usage.Cost
	}

	log.Debug().Interface("result", resp.Response).Msg("openrouter transcript")

	return resp, nil
}
