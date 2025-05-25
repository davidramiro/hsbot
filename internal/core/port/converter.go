package port

import "context"

type ImageConverter interface {
	// Scale transforms an image with liquid rescaling specified by the imageURL based on the given strength factor and
	// returns the processed image as bytes.
	Scale(ctx context.Context, imageURL string, power float32) ([]byte, error)
}
