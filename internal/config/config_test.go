package config

import (
	"os"
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const validTOML = `
[bot]
log_level = "debug"
debug_replies = true

[chat]
context_timeout = "2m"
system_prompt = "hi"

[handler]
timeout = "10s"

[telegram]
bot_token = "token"
admin_username = "admin"
allowed_chat_ids = [1, 2]
daily_spend_limit = 1.5
api_url = "https://example.com"

[openrouter]
api_key = "key"
tts_model = "tts"
stt_model = "stt"
models = [{ keyword = "gpt", identifier = "openai/gpt", Default = 1}]
image_models = [{ keyword = "flash", identifier = "img", Default = 1}]
voices = [{ keyword = "default", identifier = "v", Default = 1}]
`

func writeConfig(t *testing.T, contents string) {
	t.Helper()
	t.Chdir(t.TempDir())
	viper.Reset()
	t.Cleanup(viper.Reset)
	require.NoError(t, os.WriteFile("config.toml", []byte(contents), 0o600))
}

func TestLoad(t *testing.T) {
	writeConfig(t, validTOML)

	cfg, err := Load()
	require.NoError(t, err)
	assert.Equal(t, "debug", cfg.LogLevel)
	assert.True(t, cfg.DebugReplies)
	assert.Equal(t, 10*time.Second, cfg.HandlerTimeout)
	assert.Equal(t, 2*time.Minute, cfg.Chat.ContextTimeout)
	assert.Equal(t, "hi", cfg.Chat.SystemPrompt)
	assert.Equal(t, "token", cfg.Telegram.BotToken)
	assert.Equal(t, "admin", cfg.Telegram.AdminUsername)
	assert.Equal(t, []int64{1, 2}, cfg.Telegram.AllowedChatIDs)
	assert.InDelta(t, 1.5, cfg.Telegram.DailySpendLimit, 1e-9)
	assert.Equal(t, "https://example.com", cfg.Telegram.APIURL)
	assert.Equal(t, "key", cfg.OpenRouter.APIKey)
	assert.Equal(t, "tts", cfg.OpenRouter.TTSModel)
	assert.Equal(t, "stt", cfg.OpenRouter.STTModel)
	require.Len(t, cfg.OpenRouter.Models, 1)
	assert.Equal(t, "gpt", cfg.OpenRouter.Models[0].Keyword)
	require.Len(t, cfg.OpenRouter.ImageModels, 1)
	require.Len(t, cfg.OpenRouter.Voices, 1)
}

func TestLoadMissingFile(t *testing.T) {
	t.Chdir(t.TempDir())
	viper.Reset()
	t.Cleanup(viper.Reset)

	_, err := Load()
	require.Error(t, err)
	assert.ErrorContains(t, err, "read config")
}

func TestLoadInvalidTimeout(t *testing.T) {
	writeConfig(t, `
[handler]
timeout = "nope"
[telegram]
allowed_chat_ids = [1]
[openrouter]
models = []
image_models = []
voices = []
`)

	_, err := Load()
	require.Error(t, err)
	assert.ErrorContains(t, err, "invalid handler.timeout")
}

func TestLoadInvalidAllowedChatIDs(t *testing.T) {
	writeConfig(t, `
[handler]
timeout = "1s"
[telegram]
allowed_chat_ids = "nope"
`)

	_, err := Load()
	require.Error(t, err)
	assert.ErrorContains(t, err, "telegram.allowed_chat_ids")
}

func TestLoadInvalidModels(t *testing.T) {
	writeConfig(t, `
[handler]
timeout = "1s"
[telegram]
allowed_chat_ids = [1]
[openrouter]
models = "nope"
`)

	_, err := Load()
	require.Error(t, err)
	assert.ErrorContains(t, err, "openrouter.models")
}

func TestLoadInvalidImageModels(t *testing.T) {
	writeConfig(t, `
[handler]
timeout = "1s"
[telegram]
allowed_chat_ids = [1]
[openrouter]
models = []
image_models = "nope"
`)

	_, err := Load()
	require.Error(t, err)
	assert.ErrorContains(t, err, "openrouter.image_models")
}

func TestLoadInvalidVoices(t *testing.T) {
	writeConfig(t, `
[handler]
timeout = "1s"
[telegram]
allowed_chat_ids = [1]
[openrouter]
models = []
image_models = []
voices = "nope"
`)

	_, err := Load()
	require.Error(t, err)
	assert.ErrorContains(t, err, "openrouter.voices")
}
