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
	b, err := bot.New(token)
	if err != nil {
		log.Panic().Err(err).Msg("failed initializing telegram bot")
	}

	s := sender.NewTelegramSender(b)

	claudeGenerator := generator.NewClaudeGenerator(viper.GetString("claude.api_key"),
		viper.GetString("claude.system_prompt"))
	fluxGenerator := generator.NewFALGenerator(viper.GetString("fal.flux_url"), viper.GetString("fal.api_key"))
	magickConverter, err := converter.NewMagickConverter()
	if err != nil {
		log.Panic().Err(err).Msg("failed initializing magick converter")
	}
	transcriber := generator.NewFALGenerator(viper.GetString("fal.whisper_url"), viper.GetString("fal.api_key"))

	convoTimeout, err := time.ParseDuration(viper.GetString("chat.context_timeout"))
	if err != nil {
		log.Panic().Err(err).Msg("invalid timeout for chat context in config")
	}

	commandRegistry := &domain.CommandRegistry{}
	commandRegistry.Register(commands.NewChatHandler(claudeGenerator, s, transcriber, "/chat", convoTimeout))
	commandRegistry.Register(commands.NewImageHandler(fluxGenerator, s, s, "/image"))
	commandRegistry.Register(commands.NewScaleHandler(magickConverter, s, s, "/scale"))
	commandRegistry.Register(commands.NewTranscribeHandler(transcriber, s, "/transcribe"))

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
