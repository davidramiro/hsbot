package port

import "context"

type ImageConverter interface {
	Scale(ctx context.Context, imageURL string, power float32) ([]byte, error)
}
