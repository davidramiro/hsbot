package port

import (
	"context"
	"hsbot/internal/core/domain"
)

type TextGenerator interface {
	// GenerateFromPrompt generates a response based on the provided prompts within the given context. Returns a
	// domain.GeneratedText or an error.
	GenerateFromPrompt(ctx context.Context, prompts []domain.Prompt) (domain.GeneratedText, error)
}

type Transcriber interface {
	// GenerateFromAudio generates a text transcription from the audio file located at the provided URL.
	// It returns the transcribed text or an error if the transcription fails.
	Transcribe(ctx context.Context, url string) (domain.GeneratedText, error)
}

type ImageGenerator interface {
	NewImage(ctx context.Context, prompt string) (domain.GeneratedImage, error)
	EditImage(ctx context.Context, prompt domain.Prompt) (domain.GeneratedImage, error)
}

type AudioGenerator interface {
	Speak(ctx context.Context, text string) (domain.GeneratedAudio, error)
}

type ModelCatalog interface {
	ListTextModels() []domain.Model
}
