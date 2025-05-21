package port

import (
	"context"
	"hsbot/internal/core/domain"
)

type TextGenerator interface {
	GenerateFromPrompt(ctx context.Context, prompts []domain.Prompt) (domain.ModelResponse, error)
}

type Transcriber interface {
	GenerateFromAudio(ctx context.Context, url string) (string, error)
}

type ImageGenerator interface {
	GenerateFromPrompt(ctx context.Context, prompt string) (string, error)
	EditFromPrompt(ctx context.Context, prompt domain.Prompt) (string, error)
}
