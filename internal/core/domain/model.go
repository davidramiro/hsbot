package domain

import (
	"strings"
)

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

type Model struct {
	Keyword    string
	Identifier string
}

var allModels = []Model{ModelGemini, ModelClaude, ModelGPT, ModelGrok, ModelDeepSeek, ModelUnslop}

var (
	ModelClaude   = Model{Keyword: "claude", Identifier: "anthropic/claude-3.5-sonnet:beta"}
	ModelGPT      = Model{Keyword: "gpt", Identifier: "openai/gpt-4.1"}
	ModelGemini   = Model{Keyword: "gemini", Identifier: "google/gemini-2.5-pro-preview"}
	ModelGrok     = Model{Keyword: "grok", Identifier: "x-ai/grok-3-beta"}
	ModelDeepSeek = Model{Keyword: "deepseek", Identifier: "deepseek/deepseek-chat-v3-0324"}
	ModelUnslop   = Model{Keyword: "unslop", Identifier: "thedrummer/unslopnemo-12b"}
)

func FindModelByMessage(message *string) Model {
	for _, model := range allModels {
		if strings.Contains(strings.ToLower(*message), "#"+strings.ToLower(model.Keyword)) {
			*message = strings.ReplaceAll(*message, "#"+strings.ToLower(model.Keyword), "")
			return model
		}
	}

	return ModelClaude
}

type ModelResponse struct {
	Response string
	Metadata ResponseMetadata
}

type ResponseMetadata struct {
	Model            string
	CompletionTokens int
	TotalTokens      int
}
