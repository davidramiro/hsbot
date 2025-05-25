package port

import (
	"context"
	"hsbot/internal/core/domain"
	"time"
)

type Command interface {
	// Respond processes a given message within a specified timeout and responds to the originating context.
	Respond(ctx context.Context, timeout time.Duration, message *domain.Message) error
	// GetCommand retrieves the command identifier associated with a specific command handler.
	GetCommand() string
}

type CommandRegistry interface {
	// Register adds a new command handler to the command registry.
	Register(handler Command)
	// Get retrieves a registered Command based on its string identifier or returns an error if not found.
	Get(command string) (Command, error)
	// ListCommands returns a list of all command identifiers currently registered in the command registry.
	ListCommands() []string
}
