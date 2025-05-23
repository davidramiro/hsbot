package port

import (
	"context"
	"hsbot/internal/core/domain"
)

type TextSender interface {
	SendMessageReply(ctx context.Context, message *domain.Message, text string) (int, error)
	SendChatAction(ctx context.Context, chatID int64, action domain.Action)
	NotifyAndReturnError(ctx context.Context, err error, message *domain.Message) error
}

type ImageSender interface {
	SendImageURLReply(ctx context.Context, message *domain.Message, url string) error
	SendImageFileReply(ctx context.Context, message *domain.Message, file []byte) error
}
