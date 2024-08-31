package commands

import (
	"context"
	"errors"
	"hsbot/internal/core/domain"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type MockImageConverter struct {
	err      error
	response []byte
}

func (m *MockImageConverter) Scale(_ context.Context, _ string, _ float32) ([]byte, error) {
	return m.response, m.err
}

func TestNewScaleHandler(t *testing.T) {
	mg := &MockImageConverter{}
	ms := &MockImageSender{}
	ts := &MockTextSender{}

	scaleHandler := NewScaleHandler(mg, ts, ms, "/scale")

	assert.NotNil(t, scaleHandler)
	assert.Equal(t, "/scale", scaleHandler.GetCommand())
}

func TestScaleRespondSuccessful(t *testing.T) {
	mg := &MockImageConverter{response: []byte("success")}
	ms := &MockImageSender{}
	ts := &MockTextSender{}

	scaleHandler := NewScaleHandler(mg, ts, ms, "/scale")

	id := new(int)
	*id = 1
	err := scaleHandler.Respond(context.Background(), time.Minute, &domain.Message{ImageURL: "foo", ReplyToMessageID: id})
	require.NoError(t, err)

	assert.Equal(t, "success", ms.Message)
}

func TestScaleRespondErrorNoReply(t *testing.T) {
	mg := &MockImageConverter{response: []byte("success")}
	ms := &MockImageSender{}
	ts := &MockTextSender{}

	scaleHandler := NewScaleHandler(mg, ts, ms, "/scale")

	id := new(int)
	*id = 1
	err := scaleHandler.Respond(context.Background(), time.Minute, &domain.Message{ImageURL: "foo", Text: "/scale 80"})
	require.NoError(t, err)

	assert.Equal(t, "reply to an image", ts.Message)
}

func TestScaleRespondErrorNoReplyAndErrorSending(t *testing.T) {
	mg := &MockImageConverter{response: []byte("success")}
	ms := &MockImageSender{}
	ts := &MockTextSender{err: errors.New("mock error")}

	scaleHandler := NewScaleHandler(mg, ts, ms, "/scale")

	id := new(int)
	*id = 1
	err := scaleHandler.Respond(context.Background(), time.Minute, &domain.Message{ImageURL: "foo", Text: "/scale 80"})
	require.Errorf(t, err, "mock error")

	assert.Equal(t, "reply to an image", ts.Message)
}

func TestScaleRespondInvalidParam(t *testing.T) {
	mg := &MockImageConverter{response: []byte("success")}
	ms := &MockImageSender{}
	ts := &MockTextSender{}

	scaleHandler := NewScaleHandler(mg, ts, ms, "/scale")

	id := new(int)
	*id = 1
	err := scaleHandler.Respond(context.Background(), time.Minute, &domain.Message{ImageURL: "foo", ReplyToMessageID: id,
		Text: "/scale foo"})
	require.NoError(t, err)

	assert.Equal(t, "usage: /scale or /scale <power>, 1-100", ts.Message)
}

func TestScaleRespondInvalidParamAndErrorSending(t *testing.T) {
	mg := &MockImageConverter{response: []byte("success")}
	ms := &MockImageSender{}
	ts := &MockTextSender{err: errors.New("mock error")}

	scaleHandler := NewScaleHandler(mg, ts, ms, "/scale")

	id := new(int)
	*id = 1
	err := scaleHandler.Respond(context.Background(), time.Minute, &domain.Message{ImageURL: "foo", ReplyToMessageID: id,
		Text: "/scale foo"})
	require.Errorf(t, err, "mock error")

	assert.Equal(t, "usage: /scale or /scale <power>, 1-100", ts.Message)
}

func TestScaleRespondErrorScaleFailed(t *testing.T) {
	mg := &MockImageConverter{err: errors.New("mock error")}
	ms := &MockImageSender{}
	ts := &MockTextSender{}

	scaleHandler := NewScaleHandler(mg, ts, ms, "/scale")

	id := new(int)
	*id = 1
	err := scaleHandler.Respond(context.Background(), time.Minute, &domain.Message{ImageURL: "foo", ReplyToMessageID: id,
		Text: "/scale 80"})
	require.NoError(t, err)

	assert.Equal(t, "failed to scale image: mock error", ts.Message)
}

func TestScaleRespondErrorScaleFailedAndErrorSending(t *testing.T) {
	mg := &MockImageConverter{err: errors.New("mock error")}
	ms := &MockImageSender{}
	ts := &MockTextSender{err: errors.New("mock error")}

	scaleHandler := NewScaleHandler(mg, ts, ms, "/scale")

	id := new(int)
	*id = 1
	err := scaleHandler.Respond(context.Background(), time.Minute, &domain.Message{ImageURL: "foo", ReplyToMessageID: id,
		Text: "/scale 80"})
	require.Errorf(t, err, "mock error")

	assert.Equal(t, "failed to scale image: mock error", ts.Message)
}

func TestScaleRespondSendImageFailed(t *testing.T) {
	mg := &MockImageConverter{response: []byte("success")}
	ms := &MockImageSender{err: errors.New("mock error")}
	ts := &MockTextSender{}

	scaleHandler := NewScaleHandler(mg, ts, ms, "/scale")

	id := new(int)
	*id = 1
	err := scaleHandler.Respond(context.Background(), time.Minute, &domain.Message{ImageURL: "foo", ReplyToMessageID: id})
	require.Errorf(t, err, "mock error")
}
