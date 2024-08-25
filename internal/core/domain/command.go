package domain

import (
	"context"
	"errors"
	"strings"

	"github.com/rs/zerolog/log"
)

type CommandResponder interface {
	Respond(ctx context.Context, message *Message)
	GetCommand() string
}

type CommandRegistry struct {
	commands map[string]CommandResponder
}

func (c *CommandRegistry) Register(handler CommandResponder) {
	if c.commands == nil {
		c.commands = make(map[string]CommandResponder)
	}

	log.Info().Str("handler", handler.GetCommand()).Msg("adding command handler to registry")
	c.commands[handler.GetCommand()] = handler
}

func (c *CommandRegistry) Get(command string) (CommandResponder, error) {
	log.Debug().Interface("commands", command).Msg("fetching command handler from registry")

	if c.commands == nil {
		err := errors.New("can't fetch commands, registry not initialized")
		return nil, err
	}

	handler, ok := c.commands[command]
	if !ok {
		return nil, errors.New("commands not found")
	}

	return handler, nil
}
func (c *CommandRegistry) ListServices() []string {
	keys := make([]string, len(c.commands))

	i := 0
	for k := range c.commands {
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
	return command[0]
}
