package generator

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"hsbot/internal/core/domain"
	"io"
	"net/http"

	"github.com/rs/zerolog/log"
)

// FAL provides a wrapper for the FAL API.
type FAL struct {
	falAPIKey               string
	imageGenerationEndpoint string
	whisperEndpoint         string
	imageEditingEndpoint    string
}

func NewFAL(imageGenerationEndpoint, imageEditingEndpoint, whisperEndpoint, apiKey string) *FAL {
	return &FAL{
		falAPIKey:               apiKey,
		imageGenerationEndpoint: imageGenerationEndpoint,
		whisperEndpoint:         whisperEndpoint,
		imageEditingEndpoint:    imageEditingEndpoint,
	}
}

type imageGenerationRequest struct {
	Prompt string `json:"prompt"`
}

type imageEditRequest struct {
	Prompt              string `json:"prompt"`
	EnableSafetyChecker bool   `json:"enable_safety_checker"`
	InputImageURL       string `json:"image_url"`
}

type imageResponse struct {
	Images []struct {
		URL string `json:"url"`
	} `json:"images"`
	Prompt string `json:"prompt"`
}

func (f *FAL) GenerateFromPrompt(ctx context.Context, prompt string) (string, error) {
	falRequest := imageGenerationRequest{
		Prompt: prompt,
	}

	payloadBuf := new(bytes.Buffer)
	err := json.NewEncoder(payloadBuf).Encode(falRequest)
	if err != nil {
		return "", err
	}

	body, err := f.postFALRequest(ctx, f.imageGenerationEndpoint, payloadBuf)
	if err != nil {
		return "", err
	}

	log.Info().Interface("body", body).Msg("FAL imageResponse")

	var result imageResponse
	if err := json.Unmarshal(body, &result); err != nil {
		log.Error().Err(err).Msg("error unmarshalling FAL imageResponse")
		return "", err
	}

	if len(result.Images) == 0 {
		err = errors.New("no images returned")
		log.Error().Err(err).Send()
		return "", err
	}

	log.Info().Interface("result", result).Msg("FAL imageResponse")

	return result.Images[0].URL, nil
}

func (f *FAL) EditFromPrompt(ctx context.Context, prompt domain.Prompt) (string, error) {
	if len(prompt.Prompt) == 0 {
		return "", errors.New("missing prompt")
	}

	if len(prompt.ImageURL) == 0 {
		return "", errors.New("missing image")
	}

	falRequest := imageEditRequest{
		Prompt:              prompt.Prompt,
		InputImageURL:       prompt.ImageURL,
		EnableSafetyChecker: false,
	}

	payloadBuf := new(bytes.Buffer)
	err := json.NewEncoder(payloadBuf).Encode(falRequest)
	if err != nil {
		return "", err
	}

	body, err := f.postFALRequest(ctx, f.imageEditingEndpoint, payloadBuf)
	if err != nil {
		return "", err
	}

	log.Info().Interface("body", body).Msg("FAL imageResponse")

	var result imageResponse
	if err := json.Unmarshal(body, &result); err != nil {
		log.Error().Err(err).Msg("error unmarshalling FAL imageResponse")
		return "", err
	}

	if len(result.Images) == 0 {
		err = errors.New("no images returned")
		log.Error().Err(err).Send()
		return "", err
	}

	log.Info().Interface("result", result).Msg("FAL imageResponse")

	return result.Images[0].URL, nil
}

type audioRequest struct {
	AudioURL string `json:"audio_url"`
}

type audioResponse struct {
	Text string `json:"text"`
}

func (f *FAL) GenerateFromAudio(ctx context.Context, url string) (string, error) {
	falRequest := audioRequest{
		AudioURL: url,
	}

	payloadBuf := new(bytes.Buffer)
	err := json.NewEncoder(payloadBuf).Encode(falRequest)
	if err != nil {
		return "", err
	}

	body, err := f.postFALRequest(ctx, f.whisperEndpoint, payloadBuf)
	if err != nil {
		return "", err
	}
	log.Info().Interface("body", body).Msg("FAL audioResponse")

	var result audioResponse
	if err := json.Unmarshal(body, &result); err != nil {
		log.Error().Err(err).Msg("error unmarshalling FAL audioResponse")
		return "", err
	}

	log.Info().Interface("result", result).Msg("FAL audioResponse")

	return result.Text, nil
}

func (f *FAL) postFALRequest(ctx context.Context, url string, payloadBuf *bytes.Buffer) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, payloadBuf)
	if err != nil {
		log.Error().Err(err).Msg("error creating POST request for FAL")
		return nil, err
	}

	req.Header.Add("Authorization", "Key "+f.falAPIKey)
	req.Header.Add("Content-Type", "application/json")

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		log.Error().Err(err).Msg("error executing request to FAL")
		return nil, err
	}

	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		log.Error().Err(err).Msg("error parsing FAL response")
		return nil, err
	}
	return body, nil
}
