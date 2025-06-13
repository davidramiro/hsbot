package command

import (
	"context"
	"fmt"
	"hsbot/internal/core/domain"
	"hsbot/internal/core/port"
	"runtime"
	"runtime/debug"
	"runtime/metrics"
	"time"

	"github.com/rs/zerolog/log"
)

type Debug struct {
	textSender port.TextSender
	command    string
}

func NewDebug(sender port.TextSender, command string) *Debug {
	return &Debug{textSender: sender, command: command}
}

func (d *Debug) GetCommand() string {
	return d.command
}

const kb = 1024
const debugTemplate = `allocated mem: %d KB
threads running: %d
heap: %d KB
stack: %d KB
compiled with %s for %s-%s
`
const metricCount = 3

func (d *Debug) Respond(ctx context.Context, _ time.Duration, message *domain.Message) error {
	l := log.With().
		Int("messageId", message.ID).
		Int64("chatId", message.ChatID).
		Str("command", d.GetCommand()).
		Logger()

	data := make([]metrics.Sample, metricCount)
	data[0] = metrics.Sample{Name: "/memory/classes/heap/objects:bytes"}
	data[1] = metrics.Sample{Name: "/memory/classes/heap/stacks:bytes"}
	data[2] = metrics.Sample{Name: "/memory/classes/total:bytes"}

	metrics.Read(data)

	for _, sample := range data {
		log.Info().Str("name", sample.Name).Msgf("%d", sample.Value.Uint64())
	}

	l.Info().Msg("handling request")

	var goos, goarch string
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, setting := range info.Settings {
			switch setting.Key {
			case "GOOS":
				goos = setting.Value
			case "GOARCH":
				goarch = setting.Value
			}
		}
	}

	_, err := d.textSender.SendMessageReply(ctx, message,
		fmt.Sprintf(
			debugTemplate,
			data[2].Value.Uint64()/kb,
			runtime.NumGoroutine(),
			data[0].Value.Uint64()/kb,
			data[1].Value.Uint64()/kb,
			runtime.Version(), goos, goarch,
		))
	if err != nil {
		return err
	}

	return nil
}
