package command

import (
	"errors"
	"hsbot/internal/core/domain"
	"hsbot/internal/core/port"
	"maps"
	"slices"
	"strings"

	"github.com/rs/zerolog/log"
)

var ErrRegistryNotInitialized = errors.New("can't fetch command, registry not initialized")

type Registry struct {
	commands map[string]port.Command
}

func NewRegistry() *Registry {
	return &Registry{commands: make(map[string]port.Command)}
}

func (r *Registry) Register(handler port.Command) {
	r.ensure()
	log.Info().Str("handler", handler.GetCommand()).Msg("adding command handler to registry")
	r.commands[handler.GetCommand()] = handler
}

func (r *Registry) RegisterAlias(alias string, handler port.Command) {
	r.ensure()
	log.Info().Str("handler", alias).Msg("adding command alias to registry")
	r.commands[alias] = handler
}

func (r *Registry) Get(command string) (port.Command, error) {
	log.Trace().Interface("command", command).Msg("fetching command handler from registry")

	if r.commands == nil {
		return nil, ErrRegistryNotInitialized
	}

	handler, ok := r.commands[command]
	if !ok {
		return nil, domain.ErrCommandNotFound
	}

	return handler, nil
}

func (r *Registry) ListCommands() []string {
	if r.commands == nil {
		return []string{}
	}

	return slices.Collect(maps.Keys(r.commands))
}

func (r *Registry) ensure() {
	if r.commands == nil {
		r.commands = make(map[string]port.Command)
	}
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
