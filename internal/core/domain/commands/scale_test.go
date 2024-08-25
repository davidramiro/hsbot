package commands

import (
	"context"
	"errors"
	"github.com/stretchr/testify/assert"
	"hsbot/internal/core/domain"
	"testing"
)

type MockImageConverter struct {
	err      error
	response []byte
}

func (m *MockImageConverter) Scale(ctx context.Context, imageURL string, power float32) ([]byte, error) {
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
	err := scaleHandler.Respond(context.Background(), &domain.Message{ImageURL: "foo", ReplyToMessageID: id})
	assert.NoError(t, err)

	assert.Equal(t, "success", ms.Message)
}

func TestScaleRespondErrorNoReply(t *testing.T) {
	mg := &MockImageConverter{response: []byte("success")}
	ms := &MockImageSender{}
	ts := &MockTextSender{}

	scaleHandler := NewScaleHandler(mg, ts, ms, "/scale")

	id := new(int)
	*id = 1
	err := scaleHandler.Respond(context.Background(), &domain.Message{ImageURL: "foo", Text: "/scale 80"})
	assert.NoError(t, err)

	assert.Equal(t, "reply to an image", ts.Message)
}

func TestScaleRespondErrorNoReplyAndErrorSending(t *testing.T) {
	mg := &MockImageConverter{response: []byte("success")}
	ms := &MockImageSender{}
	ts := &MockTextSender{err: errors.New("mock error")}

	scaleHandler := NewScaleHandler(mg, ts, ms, "/scale")

	id := new(int)
	*id = 1
	err := scaleHandler.Respond(context.Background(), &domain.Message{ImageURL: "foo", Text: "/scale 80"})
	assert.Errorf(t, err, "mock error")

	assert.Equal(t, "reply to an image", ts.Message)
}

func TestScaleRespondInvalidParam(t *testing.T) {
	mg := &MockImageConverter{response: []byte("success")}
	ms := &MockImageSender{}
	ts := &MockTextSender{}

	scaleHandler := NewScaleHandler(mg, ts, ms, "/scale")

	id := new(int)
	*id = 1
	err := scaleHandler.Respond(context.Background(), &domain.Message{ImageURL: "foo", ReplyToMessageID: id,
		Text: "/scale foo"})
	assert.NoError(t, err)

	assert.Equal(t, "usage: /scale or /scale <power>, 1-100", ts.Message)
}

func TestScaleRespondInvalidParamAndErrorSending(t *testing.T) {
	mg := &MockImageConverter{response: []byte("success")}
	ms := &MockImageSender{}
	ts := &MockTextSender{err: errors.New("mock error")}

	scaleHandler := NewScaleHandler(mg, ts, ms, "/scale")

	id := new(int)
	*id = 1
	err := scaleHandler.Respond(context.Background(), &domain.Message{ImageURL: "foo", ReplyToMessageID: id,
		Text: "/scale foo"})
	assert.Errorf(t, err, "mock error")

	assert.Equal(t, "usage: /scale or /scale <power>, 1-100", ts.Message)
}

func TestScaleRespondErrorScaleFailed(t *testing.T) {
	mg := &MockImageConverter{err: errors.New("mock error")}
	ms := &MockImageSender{}
	ts := &MockTextSender{}

	scaleHandler := NewScaleHandler(mg, ts, ms, "/scale")

	id := new(int)
	*id = 1
	err := scaleHandler.Respond(context.Background(), &domain.Message{ImageURL: "foo", ReplyToMessageID: id,
		Text: "/scale 80"})
	assert.NoError(t, err)

	assert.Equal(t, "failed to scale image: mock error", ts.Message)
}

func TestScaleRespondErrorScaleFailedAndErrorSending(t *testing.T) {
	mg := &MockImageConverter{err: errors.New("mock error")}
	ms := &MockImageSender{}
	ts := &MockTextSender{err: errors.New("mock error")}

	scaleHandler := NewScaleHandler(mg, ts, ms, "/scale")

	id := new(int)
	*id = 1
	err := scaleHandler.Respond(context.Background(), &domain.Message{ImageURL: "foo", ReplyToMessageID: id,
		Text: "/scale 80"})
	assert.Errorf(t, err, "mock error")

	assert.Equal(t, "failed to scale image: mock error", ts.Message)
}

func TestScaleRespondSendImageFailed(t *testing.T) {
	mg := &MockImageConverter{response: []byte("success")}
	ms := &MockImageSender{err: errors.New("mock error")}
	ts := &MockTextSender{}

	scaleHandler := NewScaleHandler(mg, ts, ms, "/scale")

	id := new(int)
	*id = 1
	err := scaleHandler.Respond(context.Background(), &domain.Message{ImageURL: "foo", ReplyToMessageID: id})
	assert.Errorf(t, err, "mock error")
}
