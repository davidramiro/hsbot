package domain

type Author string

const (
	User   Author = "user"
	System Author = "system"
)

type Prompt struct {
	Prompt   string
	ImageURL string
	Author   Author
}

type Message struct {
	ID               int
	ChatID           int64
	ReplyToMessageID *int
	IsReplyToBot     bool
	QuotedText       string
	ImageURL         string
	AudioURL         string
	Text             string
}

type Action string

const (
	Typing       Action = "typing"
	SendingPhoto Action = "sending_photo"
)
