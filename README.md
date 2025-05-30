# hsbot

A telegram bot for generating LLM responses, manipulating images and transcribing audio.

## Prerequisites

- [Telegram bot](https://core.telegram.org/bots) API token
- [OpenRouter](https://openrouter.ai/) API key
- [fal.ai](https://fal.ai/docs) API key
- [ImageMagick](https://imagemagick.org/index.php) binary installed

Copy `config.sample.toml` to `config.toml` and set your keys/options.

## Handlers

- `/chat`: Keeping conversation context for a duration defined in the config, this handler uses OpenRouter to generate
chat responses. Use `#keyword` in a message to target a specific model. Also works with replying to images, 
when using a model that supports vision.
- `/models`: Show a list of currently active models for `/chat`.
- `/image`: Generating images from a prompt, set to use Flux as default.
- `/edit`: Edit images via prompt
- `/scale`: Liquid rescale images with a power factor
- `/transcribe`: Transcribe audio files and voice messages

## Development

The base architecture is hexagonal. For business logic and its interfaces, extend the ports side on `internal/core`.
Implementations that talks to something else than the business logic should be created as an adapter in 
`internal/adapters`.

Commands are stored and fetched dynamically, use the `CommandRegistry` to register new commands. After that, you can
create the handler in `main.go`.