package main

import (
	"context"
	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers/filters/message"
	"hsbot/internal/adapters/converter"
	"hsbot/internal/adapters/generator"
	"hsbot/internal/adapters/handler"
	"hsbot/internal/adapters/sender"
	"hsbot/internal/core/domain/command"
	"os"
	"os/signal"
	"time"

	"github.com/rs/zerolog"

	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

func main() {
	log.Info().Msg("starting hsbot...")

	err := initConfig()
	if err != nil {
		log.Fatal().Err(err).Msg("could not read config file")
	}

	initLogger()

	_, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	token := viper.GetString("telegram.bot_token")

	b, err := gotgbot.NewBot(token, nil)
	if err != nil {
		log.Panic().Err(err).Msg("failed initializing telegram bot")
	}

	// Create updater and dispatcher.
	dispatcher := ext.NewDispatcher(&ext.DispatcherOpts{
		// If a handler returns an error, log it and continue going.
		Error: func(b *gotgbot.Bot, ctx *ext.Context, err error) ext.DispatcherAction {
			log.Err(err).Msg("error handling telegram bot update")
			return ext.DispatcherActionNoop
		},
		MaxRoutines: ext.DefaultMaxRoutines,
	})
	updater := ext.NewUpdater(dispatcher, nil)

	t := sender.NewTelegram(b)

	registry := initHandlers(t)

	handlerTimeout, err := time.ParseDuration(viper.GetString("handler.timeout"))
	if err != nil {
		log.Panic().Err(err).Msg("invalid timeout for handler in config")
	}

	commandHandler := handler.NewCommand(registry, handlerTimeout)

	dispatcher.AddHandler(handlers.NewMessage(message.Text, commandHandler.Handle))

	// Start receiving updates.
	err = updater.StartPolling(b, &ext.PollingOpts{
		DropPendingUpdates: true,
		GetUpdatesOpts: &gotgbot.GetUpdatesOpts{
			Timeout: 9,
			RequestOpts: &gotgbot.RequestOpts{
				Timeout: time.Second * 10,
			},
		},
	})
	if err != nil {
		log.Panic().Err(err).Msg("failed to start polling updater")
	}

	updater.Idle()
}

func initHandlers(t *sender.Telegram) *command.Registry {
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

	registry := &command.Registry{}

	chat, err := command.NewChat(or, t, fal, "/chat", viper.GetDuration("chat.context_timeout"))
	if err != nil {
		log.Panic().Err(err).Msg("failed initializing chat handler")
	}

	registry.Register(chat)
	registry.Register(command.NewModels(chat, t, "/models"))
	registry.Register(command.NewImage(fal, t, t, "/image"))
	registry.Register(command.NewEdit(fal, t, t, "/edit"))
	registry.Register(command.NewScale(magick, t, t, "/scale"))
	registry.Register(command.NewTranscribe(fal, t, "/transcribe"))
	registry.Register(command.NewChatClearContext(chat, t, "/clear"))
	registry.Register(command.NewDebug(t, "/debug"))
	return registry
}

func initLogger() {
	var logLevel zerolog.Level

	switch viper.GetString("bot.log_level") {
	case "trace":
		logLevel = zerolog.TraceLevel
	case "debug":
		logLevel = zerolog.DebugLevel
	default:
		logLevel = zerolog.InfoLevel
	}

	zerolog.SetGlobalLevel(logLevel)
}

func initConfig() error {
	viper.AddConfigPath(".")
	viper.SetConfigType("toml")

	log.Info().Msg("reading config file...")
	return viper.ReadInConfig()
}
