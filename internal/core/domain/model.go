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
	Model    Model
}

type Message struct {
	ID               int
	ChatID           int64
	Username         string
	ReplyToMessageID *int
	ReplyToUsername  string
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

type ModelResponse struct {
	Response string
	Metadata ResponseMetadata
}

type Model struct {
	Keyword    string `json:"keyword"`
	Identifier string `json:"identifier"`
}

type ResponseMetadata struct {
	Model            string
	CompletionTokens int
	TotalTokens      int
}
