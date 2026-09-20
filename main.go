package main

import (
	"context"
	"hsbot/internal/adapters/converter"
	"hsbot/internal/adapters/generator"
	"hsbot/internal/adapters/handler"
	"hsbot/internal/adapters/sender"
	"hsbot/internal/config"
	"hsbot/internal/core/domain/command"
	"hsbot/internal/core/service"
	"os"
	"os/signal"

	"github.com/go-telegram/bot/models"

	"github.com/rs/zerolog"

	"github.com/go-telegram/bot"
	"github.com/rs/zerolog/log"
)

func main() {
	log.Info().Msg("starting hsbot...")

	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("could not read config file")
	}

	initLogger(cfg.LogLevel)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	b, err := initBot(cfg.Telegram)
	if err != nil {
		log.Panic().Err(err).Msg("failed initializing telegram bot")
	}

	t := sender.NewTelegram(b)

	registry := initHandlers(ctx, t, cfg)

	auth := service.NewAuthorizer(t, cfg.Telegram.AllowedChatIDs, cfg.Telegram.AdminUsername)

	commandHandler := handler.NewCommand(registry, cfg.HandlerTimeout, auth)

	b.RegisterHandler(bot.HandlerTypeMessageText, "/", bot.MatchTypePrefix, commandHandler.Handle)
	b.RegisterHandler(bot.HandlerTypePhotoCaption, "/", bot.MatchTypePrefix, commandHandler.Handle)

	log.Info().Msg("bot listening")
	b.Start(ctx)
}

func initHandlers(ctx context.Context, t *sender.Telegram, cfg *config.Config) *command.Registry {
	magick, err := converter.NewMagick(ctx)
	if err != nil {
		log.Panic().Err(err).Msg("failed initializing magick converter")
	}

	or, err := generator.NewOpenRouter(generator.Config{
		APIKey:       cfg.OpenRouter.APIKey,
		SystemPrompt: cfg.Chat.SystemPrompt,
		TextModels:   cfg.OpenRouter.Models,
		ImageModels:  cfg.OpenRouter.ImageModels,
		Voices:       cfg.OpenRouter.Voices,
		TTSModel:     cfg.OpenRouter.TTSModel,
		STTModel:     cfg.OpenRouter.STTModel,
	})
	if err != nil {
		log.Panic().Err(err).Msg("failed initializing openrouter generator")
	}

	registry := command.NewRegistry()

	tracker := service.NewUsageTracker(ctx, t, cfg.Telegram.DailySpendLimit)

	chat, err := command.NewChat(command.ChatParams{
		TextGenerator:  or,
		TextSender:     t,
		Transcriber:    or,
		AudioGenerator: or,
		AudioSender:    t,
		Command:        "/chat",
		SpeakCommand:   "/speak",
		CacheDuration:  cfg.Chat.ContextTimeout,
		DebugReplies:   cfg.DebugReplies,
		Tracker:        tracker,
	})

	if err != nil {
		log.Panic().Err(err).Msg("failed initializing chat handler")
	}

	registry.Register(chat)
	registry.RegisterAlias("/speak", chat)
	registry.Register(command.NewModels(or, t, "/models"))
	registry.Register(command.NewImage(or, t, t, tracker, "/image"))
	registry.Register(command.NewEdit(or, t, t, tracker, "/edit"))
	registry.Register(command.NewScale(magick, t, t, "/scale"))
	registry.Register(command.NewTranscribe(or, t, tracker, "/transcribe"))
	registry.Register(command.NewChatClearContext(chat, t, "/clear"))
	registry.Register(command.NewDebug(t, "/debug"))
	registry.Register(command.NewSpent(tracker, t, "/spent"))
	return registry
}

func initBot(cfg config.Telegram) (*bot.Bot, error) {
	opts := []bot.Option{
		bot.WithDefaultHandler(noOpHandler),
		bot.WithServerURL(cfg.APIURL),
	}

	return bot.New(cfg.BotToken, opts...)
}

func initLogger(level string) {
	var logLevel zerolog.Level

	switch level {
	case "trace":
		logLevel = zerolog.TraceLevel
	case "debug":
		logLevel = zerolog.DebugLevel
	default:
		logLevel = zerolog.InfoLevel
	}

	zerolog.SetGlobalLevel(logLevel)
}

func noOpHandler(_ context.Context, _ *bot.Bot, _ *models.Update) {}
