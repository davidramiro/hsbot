package port

import (
	"context"
	"hsbot/internal/core/domain"
)

type TextSender interface {
	// SendMessageReply sends a reply to a specified message with the given text and returns the sent message ID and
	// an error if any.
	SendMessageReply(ctx context.Context, message *domain.Message, text string) (int, error)
	// SendChatAction sends a specified chat action (e.g., typing, sending photo) to indicate activity in a given chat.
	SendChatAction(ctx context.Context, chatID int64, action domain.Action)
	// NotifyAndReturnError sends an error notification based on the provided message context and returns the error.
	NotifyAndReturnError(ctx context.Context, err error, message *domain.Message) error
}

type ImageSender interface {
	// SendImageURLReply sends an image to the chat as a reply using a URL in response to the provided message.
	SendImageURLReply(ctx context.Context, message *domain.Message, url string) error
	// SendImageFileReply sends an image as a file in response to the provided message within the specified context.
	SendImageFileReply(ctx context.Context, message *domain.Message, file []byte) error
}
