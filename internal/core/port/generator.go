package port

import (
	"context"
	"hsbot/internal/core/domain"
)

type TextGenerator interface {
	GenerateFromPrompt(ctx context.Context, prompts []domain.Prompt) (string, error)
	ThinkFromPrompt(ctx context.Context, prompt domain.Prompt) (string, string, error)
}

type Transcriber interface {
	GenerateFromAudio(ctx context.Context, url string) (string, error)
}

type ImageGenerator interface {
	GenerateFromPrompt(ctx context.Context, prompt string) (string, error)
}
