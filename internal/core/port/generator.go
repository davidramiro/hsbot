package port

import (
	"context"
	"hsbot/internal/core/domain"
)

type TextGenerator interface {
	GenerateFromPrompt(ctx context.Context, prompts []domain.Prompt) (string, error)
}

type Transcriber interface {
	GenerateFromAudio(ctx context.Context, text string) (string, error)
}

type ImageGenerator interface {
	GenerateFromPrompt(ctx context.Context, prompt string) (string, error)
}
