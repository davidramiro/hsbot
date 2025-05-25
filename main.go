package main

import (
	"context"
	"hsbot/internal/adapters/converter"
	"hsbot/internal/adapters/generator"
	"hsbot/internal/adapters/handler"
	"hsbot/internal/adapters/sender"
	"hsbot/internal/core/domain/command"
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

	t := sender.NewTelegram(b)

	or := generator.NewOpenRouter(viper.GetString("openrouter.api_key"),
		viper.GetString("chat.system_prompt"))

	magick, err := converter.NewMagick()
	if err != nil {
		log.Panic().Err(err).Msg("failed initializing magick converter")
	}

	fal := generator.NewFAL(
		viper.GetString("fal.image_gen_url"),
		viper.GetString("fal.image_edit_url"),
		viper.GetString("fal.whisper_url"),
		viper.GetString("fal.api_key"))

	convoTimeout, err := time.ParseDuration(viper.GetString("chat.context_timeout"))
	if err != nil {
		log.Panic().Err(err).Msg("invalid timeout for chat context in config")
	}

	registry := &command.Registry{}

	chat, err := command.NewChat(or, t, fal, "/chat", convoTimeout)
	if err != nil {
		log.Panic().Err(err).Msg("failed initializing chat handler")
	}

	registry.Register(chat)
	registry.Register(command.NewModels(chat, t, "/models"))
	registry.Register(command.NewImage(fal, t, t, "/image"))
	registry.Register(command.NewEdit(fal, t, t, "/edit"))
	registry.Register(command.NewScale(magick, t, t, "/scale"))
	registry.Register(command.NewTranscribe(fal, t, "/transcribe"))

	handlerTimeout, err := time.ParseDuration(viper.GetString("handler.timeout"))
	if err != nil {
		log.Panic().Err(err).Msg("invalid timeout for handler in config")
	}

	commandHandler := handler.NewCommand(registry, handlerTimeout)

	b.RegisterHandler(bot.HandlerTypeMessageText, "/", bot.MatchTypePrefix, commandHandler.Handle)
	b.RegisterHandler(bot.HandlerTypePhotoCaption, "/", bot.MatchTypePrefix, commandHandler.Handle)

	log.Info().Msg("bot listening")
	b.Start(ctx)
}

func noOpHandler(_ context.Context, _ *bot.Bot, _ *models.Update) {}
