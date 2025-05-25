package command

import (
	"errors"
	"hsbot/internal/core/domain"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEditHandler_Success(t *testing.T) {
	mg := &MockImageGenerator{imageURL: "http://image.url"}
	ms := &MockImageSender{}
	mt := &MockTextSender{}

	eh := NewEdit(mg, ms, mt, "/edit")

	msg := &domain.Message{
		ID:       1,
		ChatID:   1,
		Text:     "/edit enhance the picture",
		ImageURL: "imgurl",
	}

	err := eh.Respond(t.Context(), time.Second, msg)
	require.NoError(t, err)
	assert.True(t, ms.called, "image sender should be called")
	assert.Equal(t, "http://image.url", ms.calledURL)
	assert.Empty(t, mt.Message)
}

func TestEditHandler_EmptyPrompt(t *testing.T) {
	mg := &MockImageGenerator{}
	ms := &MockImageSender{}
	mt := &MockTextSender{}

	eh := NewEdit(mg, ms, mt, "/edit")

	msg := &domain.Message{
		ID:     1,
		ChatID: 1,
		Text:   "/edit",
	}

	err := eh.Respond(t.Context(), time.Second, msg)
	require.NoError(t, err)
	assert.Equal(t, "empty prompt", mt.Message)
	assert.False(t, ms.called)
}

func TestEditHandler_MissingImage(t *testing.T) {
	mg := &MockImageGenerator{}
	ms := &MockImageSender{}
	mt := &MockTextSender{}

	eh := NewEdit(mg, ms, mt, "/edit")

	msg := &domain.Message{
		ID:     1,
		ChatID: 1,
		Text:   "/edit do something",
	}

	err := eh.Respond(t.Context(), time.Second, msg)
	require.NoError(t, err)
	assert.Equal(t, "missing image", mt.Message)
	assert.False(t, ms.called)
}

func TestEditHandler_EditFromPromptError(t *testing.T) {
	mg := &MockImageGenerator{err: errors.New("gen-failed")}
	ms := &MockImageSender{}
	mt := &MockTextSender{}

	eh := NewEdit(mg, ms, mt, "/edit")

	msg := &domain.Message{
		ID:       1,
		ChatID:   1,
		Text:     "/edit change style",
		ImageURL: "image.png",
	}

	err := eh.Respond(t.Context(), time.Second, msg)
	require.Error(t, err)
	assert.Contains(t, mt.Message, "error creating edited image: gen-failed")
	assert.False(t, ms.called)
}

func TestEditHandler_SendImageURLReplyError(t *testing.T) {
	mg := &MockImageGenerator{imageURL: "http://image.url"}
	ms := &MockImageSender{err: errors.New("send-failed")}
	mt := &MockTextSender{}

	eh := NewEdit(mg, ms, mt, "/edit")

	msg := &domain.Message{
		ID:       1,
		ChatID:   1,
		Text:     "/edit something cool",
		ImageURL: "img2",
	}

	err := eh.Respond(t.Context(), time.Second, msg)
	require.Error(t, err)
	assert.Contains(t, mt.Message, "error sending edited image: send-failed")
	assert.True(t, ms.called)
}
