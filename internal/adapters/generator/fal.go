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

type FALGenerator struct {
	falAPIKey     string
	fluxAPIURL    string
	whisperAPIURL string
	omnigenAPIURL string
}

func NewFALGenerator(fluxAPIURL, omnigenAPIURL, whisperAPIURL, apiKey string) *FALGenerator {
	return &FALGenerator{
		falAPIKey:     apiKey,
		fluxAPIURL:    fluxAPIURL,
		whisperAPIURL: whisperAPIURL,
		omnigenAPIURL: omnigenAPIURL,
	}
}

type fluxImageRequest struct {
	Prompt              string `json:"prompt"`
	EnableSafetyChecker bool   `json:"enable_safety_checker"`
	SafetyTolerance     string `json:"safety_tolerance"`
	ImageSize           string `json:"image_size"`
}

type imageEditRequest struct {
	Prompt              string   `json:"prompt"`
	EnableSafetyChecker bool     `json:"enable_safety_checker"`
	ImageSize           string   `json:"image_size"`
	InputImageUrls      []string `json:"input_image_urls"`
}

const imgIdentifierPrompt = ": <img><|image_1|></img>"

type imageResponse struct {
	Images []struct {
		URL         string `json:"url"`
		ContentType string `json:"content_type"`
	} `json:"images"`
	Prompt string `json:"prompt"`
}

func (f *FALGenerator) GenerateFromPrompt(ctx context.Context, prompt string) (string, error) {
	falRequest := fluxImageRequest{
		Prompt:              prompt,
		EnableSafetyChecker: false,
		ImageSize:           "square",
		SafetyTolerance:     "5",
	}

	payloadBuf := new(bytes.Buffer)
	err := json.NewEncoder(payloadBuf).Encode(falRequest)
	if err != nil {
		return "", err
	}

	body, err := f.postFALRequest(ctx, f.fluxAPIURL, payloadBuf)
	if err != nil {
		return "", err
	}

	log.Info().Interface("body", body).Msg("FAL imageResponse")

	var result imageResponse
	if err := json.Unmarshal(body, &result); err != nil {
		log.Error().Err(err).Msg("error unmarshalling FAL imageResponse")
		return "", err
	}

	log.Info().Interface("result", result).Msg("FAL imageResponse")

	return result.Images[0].URL, nil
}

func (f *FALGenerator) EditFromPrompt(ctx context.Context, prompt domain.Prompt) (string, error) {
	if len(prompt.Prompt) == 0 {
		return "", errors.New("missing prompt")
	}

	if len(prompt.ImageURL) == 0 {
		return "", errors.New("missing image")
	}

	falRequest := imageEditRequest{
		Prompt:              prompt.Prompt + imgIdentifierPrompt,
		EnableSafetyChecker: false,
		ImageSize:           "square",
		InputImageUrls:      []string{prompt.ImageURL},
	}

	payloadBuf := new(bytes.Buffer)
	err := json.NewEncoder(payloadBuf).Encode(falRequest)
	if err != nil {
		return "", err
	}

	body, err := f.postFALRequest(ctx, f.omnigenAPIURL, payloadBuf)
	if err != nil {
		return "", err
	}

	log.Info().Interface("body", body).Msg("FAL imageResponse")

	var result imageResponse
	if err := json.Unmarshal(body, &result); err != nil {
		log.Error().Err(err).Msg("error unmarshalling FAL imageResponse")
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

func (f *FALGenerator) GenerateFromAudio(ctx context.Context, url string) (string, error) {
	falRequest := audioRequest{
		AudioURL: url,
	}

	payloadBuf := new(bytes.Buffer)
	err := json.NewEncoder(payloadBuf).Encode(falRequest)
	if err != nil {
		return "", err
	}

	body, err := f.postFALRequest(ctx, f.whisperAPIURL, payloadBuf)
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

func (f *FALGenerator) postFALRequest(ctx context.Context, url string, payloadBuf *bytes.Buffer) ([]byte, error) {
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
