package config

import (
	"fmt"
	"hsbot/internal/core/domain"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	LogLevel       string
	DebugReplies   bool
	HandlerTimeout time.Duration
	Chat           Chat
	Telegram       Telegram
	OpenRouter     OpenRouter
}

type Chat struct {
	ContextTimeout      time.Duration
	SystemPrompt        string
	VoicePromptAddition string
}

type Telegram struct {
	BotToken        string
	APIURL          string
	AdminUsername   string
	AllowedChatIDs  []int64
	DailySpendLimit float64
}

type OpenRouter struct {
	APIKey      string
	TTSModel    string
	STTModel    string
	Models      []domain.Model
	ImageModels []domain.Model
	Voices      []domain.Model
}

func Load() (*Config, error) {
	viper.AddConfigPath(".")
	viper.SetConfigType("toml")

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	timeout, err := time.ParseDuration(viper.GetString("handler.timeout"))
	if err != nil {
		return nil, fmt.Errorf("invalid handler.timeout: %w", err)
	}

	var allowed []int64
	if err := viper.UnmarshalKey("telegram.allowed_chat_ids", &allowed); err != nil {
		return nil, fmt.Errorf("telegram.allowed_chat_ids: %w", err)
	}

	var models []domain.Model
	if err := viper.UnmarshalKey("openrouter.models", &models); err != nil {
		return nil, fmt.Errorf("openrouter.models: %w", err)
	}

	var imageModels []domain.Model
	if err := viper.UnmarshalKey("openrouter.image_models", &imageModels); err != nil {
		return nil, fmt.Errorf("openrouter.image_models: %w", err)
	}

	var voices []domain.Model
	if err := viper.UnmarshalKey("openrouter.voices", &voices); err != nil {
		return nil, fmt.Errorf("openrouter.voices: %w", err)
	}

	return &Config{
		LogLevel:       viper.GetString("bot.log_level"),
		DebugReplies:   viper.GetBool("bot.debug_replies"),
		HandlerTimeout: timeout,
		Chat: Chat{
			ContextTimeout:      viper.GetDuration("chat.context_timeout"),
			SystemPrompt:        viper.GetString("chat.system_prompt"),
			VoicePromptAddition: viper.GetString("chat.voice_prompt_addition"),
		},
		Telegram: Telegram{
			BotToken:        viper.GetString("telegram.bot_token"),
			APIURL:          viper.GetString("telegram.api_url"),
			AdminUsername:   viper.GetString("telegram.admin_username"),
			AllowedChatIDs:  allowed,
			DailySpendLimit: viper.GetFloat64("telegram.daily_spend_limit"),
		},
		OpenRouter: OpenRouter{
			APIKey:      viper.GetString("openrouter.api_key"),
			TTSModel:    viper.GetString("openrouter.tts_model"),
			STTModel:    viper.GetString("openrouter.stt_model"),
			Models:      models,
			ImageModels: imageModels,
			Voices:      voices,
		},
	}, nil
}
