package domain

import "errors"

var (
	ErrEmptyPrompt     = errors.New("empty prompt")
	ErrMissingImage    = errors.New("missing image")
	ErrMissingAudio    = errors.New("reply to an audio")
	ErrMissingPrompt   = errors.New("missing prompt")
	ErrCommandNotFound = errors.New("command not found")
)
