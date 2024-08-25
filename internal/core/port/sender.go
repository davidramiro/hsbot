package port

import (
	"context"
	"hsbot/internal/core/domain"
)

type TextSender interface {
	SendMessageReply(ctx context.Context, chatID int64, messageID int, message string) error
	SendChatAction(ctx context.Context, chatID int64, action domain.Action)
}

type ImageSender interface {
	SendImageURLReply(ctx context.Context, chatID int64, messageID int, url string) error
	SendImageFileReply(ctx context.Context, chatID int64, messageID int, file []byte) error
}
