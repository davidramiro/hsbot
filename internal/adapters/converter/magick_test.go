package converter

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreateCommand(t *testing.T) {
	got := createCommand([]string{"magick", "convert"}, 50, "in.png", "out.png")

	assert.Equal(t, []string{"magick", "convert", "in.png", "-liquid-rescale", "61%x61%", "out.png"}, got)
}

func TestCreateCommandMaxPower(t *testing.T) {
	got := createCommand([]string{"convert"}, MaxPower, "a.jpg", "b.jpg")

	assert.Equal(t, "23%x23%", got[3])
}
