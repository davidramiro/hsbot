package command

import (
	"errors"
	"hsbot/internal/core/domain"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type spentTracker struct {
	MockTracker
	spent float64
}

func (s spentTracker) GetSpent(_ int64) float64 {
	return s.spent
}

func TestNewSpent(t *testing.T) {
	h := NewSpent(spentTracker{spent: 1.25}, &MockTextSender{}, "/spent")
	require.NotNil(t, h)
	assert.Equal(t, "/spent", h.GetCommand())
}

func TestSpentRespond(t *testing.T) {
	sender := &MockTextSender{}
	h := NewSpent(spentTracker{spent: 1.25}, sender, "/spent")

	err := h.Respond(t.Context(), time.Second, &domain.Message{ChatID: 42})
	require.NoError(t, err)
	assert.Equal(t, "Spent today within ChatID 42: $1.25.", sender.Message)
}

func TestSpentRespondSendError(t *testing.T) {
	sender := &MockTextSender{err: errors.New("boom")}
	h := NewSpent(spentTracker{}, sender, "/spent")

	err := h.Respond(t.Context(), time.Second, &domain.Message{ChatID: 1})
	require.Error(t, err)
	assert.ErrorContains(t, err, "failed to send message")
}
