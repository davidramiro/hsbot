package generator

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"

	"github.com/rs/zerolog/log"
)

type FALGenerator struct {
	apiKey string
	apiURL string
}

func NewFALGenerator(apiURL, apiKey string) *FALGenerator {
	return &FALGenerator{
		apiKey: apiKey,
		apiURL: apiURL,
	}
}

type imageRequest struct {
	Prompt              string `json:"prompt"`
	EnableSafetyChecker bool   `json:"enable_safety_checker"`
	SafetyTolerance     string `json:"safety_tolerance"`
	ImageSize           string `json:"image_size"`
}

type imageResponse struct {
	Images []struct {
		URL         string `json:"url"`
		ContentType string `json:"content_type"`
	} `json:"images"`
	Prompt string `json:"prompt"`
}

func (f *FALGenerator) GenerateFromPrompt(ctx context.Context, prompt string) (string, error) {
	falRequest := imageRequest{
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

	body, err := f.postFALRequest(ctx, payloadBuf)
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

	body, err := f.postFALRequest(ctx, payloadBuf)
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

func (f *FALGenerator) postFALRequest(ctx context.Context, payloadBuf *bytes.Buffer) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, f.apiURL, payloadBuf)
	if err != nil {
		log.Error().Err(err).Msg("error creating POST request for FAL")
		return nil, err
	}

	req.Header.Add("Authorization", "Key "+f.apiKey)
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
