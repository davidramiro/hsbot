package command

import (
	"errors"
	"hsbot/internal/core/port"
	"strings"

	"github.com/rs/zerolog/log"
)

type Registry struct {
	commands map[string]port.Command
}

func (r *Registry) Register(handler port.Command) {
	if r.commands == nil {
		r.commands = make(map[string]port.Command)
	}

	log.Info().Str("handler", handler.GetCommand()).Msg("adding command handler to registry")
	r.commands[handler.GetCommand()] = handler
}

func (r *Registry) Get(command string) (port.Command, error) {
	log.Debug().Interface("command", command).Msg("fetching command handler from registry")

	if r.commands == nil {
		err := errors.New("can't fetch command, registry not initialized")
		return nil, err
	}

	handler, ok := r.commands[command]
	if !ok {
		return nil, errors.New("command not found")
	}

	return handler, nil
}

func (r *Registry) ListCommands() []string {
	keys := make([]string, len(r.commands))

	i := 0
	for k := range r.commands {
		keys[i] = k
		i++
	}

	return keys
}

func ParseCommandArgs(args string) string {
	command := strings.Split(args, " ")
	return strings.Join(command[1:], " ")
}

func ParseCommand(args string) string {
	command := strings.Split(args, " ")
	return strings.ToLower(command[0])
}
