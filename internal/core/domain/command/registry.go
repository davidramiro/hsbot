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
	if r.commands == nil {
		return []string{}
	}

	keys := make([]string, len(r.commands))

	i := 0
	for k := range r.commands {
		keys[i] = k
		i++
	}

	return keys
}

// ParseCommandArgs extracts the portion of the string after the first space, or returns an empty string if none exists.
func ParseCommandArgs(args string) string {
	idx := strings.IndexByte(args, ' ')
	if idx == -1 {
		return ""
	}
	return args[idx+1:]
}

// ParseCommand extracts the primary command from the input string, discarding arguments and handle, returning it in lowercase.
func ParseCommand(args string) string {
	command := strings.Split(args, " ")[0]
	if strings.Contains(command, "@") {
		command = strings.Split(command, "@")[0]
	}
	return strings.ToLower(command)
}
