package port

import (
	"context"
	"hsbot/internal/core/domain"
)

type TextGenerator interface {
	// GenerateFromPrompt generates a response based on the provided prompts within the given context. Returns a
	// domain.ModelResponse or an error.
	GenerateFromPrompt(ctx context.Context, prompts []domain.Prompt) (domain.ModelResponse, error)
}

type Transcriber interface {
	// GenerateFromAudio generates a text transcription from the audio file located at the provided URL.
	// It returns the transcribed text or an error if the transcription fails.
	GenerateFromAudio(ctx context.Context, url string) (string, error)
}

type ImageGenerator interface {
	// GenerateFromPrompt generates an image based on the provided textual prompt within the given context.
	GenerateFromPrompt(ctx context.Context, prompt string) (string, error)
	// EditFromPrompt edits an existing image based on the supplied prompt details within the provided execution
	// context.
	EditFromPrompt(ctx context.Context, prompt domain.Prompt) (string, error)
}
