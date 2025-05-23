package domain

import "errors"

var (
	ErrSendingReplyFailed = errors.New("failed to send reply")
	ErrEmptyPrompt        = errors.New("empty prompt")
)
