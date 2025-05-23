package main

import (
	"context"
	"hsbot/internal/adapters/converter"
	"hsbot/internal/adapters/generator"
	"hsbot/internal/adapters/handler"
	"hsbot/internal/adapters/sender"
	"hsbot/internal/core/domain"
	"hsbot/internal/core/domain/commands"
	"os"
	"os/signal"
	"time"

	"github.com/go-telegram/bot/models"

	"github.com/rs/zerolog"

	"github.com/go-telegram/bot"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

func main() {
	log.Info().Msg("starting hsbot...")

	viper.AddConfigPath(".")
	viper.SetConfigType("toml")

	log.Info().Msg("reading config file...")
	err := viper.ReadInConfig()
	if err != nil {
		log.Fatal().Err(err).Msg("could not read config file")
	}

	var logLevel zerolog.Level

	switch viper.GetString("bot.log_level") {
	case "info":
		logLevel = zerolog.InfoLevel
	case "debug":
		logLevel = zerolog.DebugLevel
	default:
		logLevel = zerolog.InfoLevel
	}

	zerolog.SetGlobalLevel(logLevel)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	token := viper.GetString("telegram.bot_token")
	opts := []bot.Option{
		bot.WithDefaultHandler(noOpHandler),
	}

	b, err := bot.New(token, opts...)
	if err != nil {
		log.Panic().Err(err).Msg("failed initializing telegram bot")
	}

	s := sender.NewTelegramSender(b)

	orGenerator := generator.NewOpenRouterGenerator(viper.GetString("openrouter.api_key"),
		viper.GetString("chat.system_prompt"))

	magickConverter, err := converter.NewMagickConverter()
	if err != nil {
		log.Panic().Err(err).Msg("failed initializing magick converter")
	}
	falGenerator := generator.NewFALGenerator(
		viper.GetString("fal.flux_url"),
		viper.GetString("fal.edit_url"),
		viper.GetString("fal.whisper_url"),
		viper.GetString("fal.api_key"))

	convoTimeout, err := time.ParseDuration(viper.GetString("chat.context_timeout"))
	if err != nil {
		log.Panic().Err(err).Msg("invalid timeout for chat context in config")
	}

	commandRegistry := &domain.CommandRegistry{}

	chatHandler, err := commands.NewChatHandler(orGenerator, s, falGenerator, "/chat", convoTimeout)
	if err != nil {
		log.Panic().Err(err).Msg("failed initializing chat handler")
	}

	commandRegistry.Register(chatHandler)
	commandRegistry.Register(commands.NewModelHandler(chatHandler, s, "/models"))
	commandRegistry.Register(commands.NewImageHandler(falGenerator, s, s, "/image"))
	commandRegistry.Register(commands.NewEditHandler(falGenerator, s, s, "/edit"))
	commandRegistry.Register(commands.NewScaleHandler(magickConverter, s, s, "/scale"))
	commandRegistry.Register(commands.NewTranscribeHandler(falGenerator, s, "/transcribe"))

	handlerTimeout, err := time.ParseDuration(viper.GetString("handler.timeout"))
	if err != nil {
		log.Panic().Err(err).Msg("invalid timeout for handler in config")
	}

	commandHandler := handler.NewCommandHandler(commandRegistry, handlerTimeout)

	b.RegisterHandler(bot.HandlerTypeMessageText, "/", bot.MatchTypePrefix, commandHandler.Handle)
	b.RegisterHandler(bot.HandlerTypePhotoCaption, "/", bot.MatchTypePrefix, commandHandler.Handle)

	log.Info().Msg("bot listening")
	b.Start(ctx)
}

func noOpHandler(_ context.Context, _ *bot.Bot, _ *models.Update) {}
