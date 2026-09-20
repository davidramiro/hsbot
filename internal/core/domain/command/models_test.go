package command

import (
	"errors"
	"hsbot/internal/core/domain"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockCatalog struct {
	models []domain.Model
}

func (m mockCatalog) ListTextModels() []domain.Model {
	return m.models
}

func TestNewModels(t *testing.T) {
	h := NewModels(mockCatalog{}, &MockTextSender{}, "/models")
	require.NotNil(t, h)
	assert.Equal(t, "/models", h.GetCommand())
}

func TestModelsRespond(t *testing.T) {
	sender := &MockTextSender{}
	h := NewModels(mockCatalog{models: []domain.Model{
		{Keyword: "gpt", Identifier: "openai/gpt"},
		{Keyword: "claude", Identifier: "anthropic/claude"},
	}}, sender, "/models")

	err := h.Respond(t.Context(), time.Second, &domain.Message{ChatID: 1, ID: 1})
	require.NoError(t, err)
	assert.Contains(t, sender.Message, "openai/gpt")
	assert.Contains(t, sender.Message, "Keyword: gpt")
	assert.Contains(t, sender.Message, "anthropic/claude")
	assert.Contains(t, sender.Message, "image recognition")
}

func TestModelsRespondSendError(t *testing.T) {
	sender := &MockTextSender{err: errors.New("send fail")}
	h := NewModels(mockCatalog{}, sender, "/models")

	err := h.Respond(t.Context(), time.Second, &domain.Message{ChatID: 1, ID: 1})
	require.Error(t, err)
	assert.ErrorContains(t, err, "failed to send message")
}
