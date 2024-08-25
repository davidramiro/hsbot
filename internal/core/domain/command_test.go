package domain

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

type MockResponder struct {
	command string
}

func (m *MockResponder) Respond(ctx context.Context, message *Message) error {
	return nil
}

func (m *MockResponder) GetCommand() string {
	return m.command
}

func TestRegister(t *testing.T) {
	cr := &CommandRegistry{}
	mr := &MockResponder{command: "/test"}

	cr.Register(mr)
	assert.Equal(t, 1, len(cr.commands))
}

func TestGetNotRegistered(t *testing.T) {
	cr := &CommandRegistry{}

	_, err := cr.Get("test")
	assert.Errorf(t, err, "can't fetch commands, registry not initialized")
}

func TestGetCommandNotFound(t *testing.T) {
	cr := &CommandRegistry{}
	mr := &MockResponder{command: "/test"}

	cr.Register(mr)
	assert.Equal(t, 1, len(cr.commands))

	_, err := cr.Get("/foo")
	assert.Errorf(t, err, "command not found")
}

func TestGetCommandFound(t *testing.T) {
	cr := &CommandRegistry{}
	mr := &MockResponder{command: "/test"}

	cr.Register(mr)
	assert.Equal(t, 1, len(cr.commands))

	cmd, err := cr.Get("/test")
	assert.NoError(t, err)
	assert.NotNil(t, cmd)

	assert.Equal(t, "/test", cmd.GetCommand())
}

func TestListServices(t *testing.T) {
	cr := &CommandRegistry{}
	mr1 := &MockResponder{command: "/foo"}
	mr2 := &MockResponder{command: "/bar"}

	cr.Register(mr1)
	cr.Register(mr2)
	assert.Equal(t, 2, len(cr.commands))

	list := cr.ListCommands()

	assert.Equal(t, 2, len(list))
	assert.Equal(t, "/foo", list[0])
	assert.Equal(t, "/bar", list[1])
}

func TestParseCommandArgs(t *testing.T) {
	type TestCase struct {
		description string
		args        string
		want        string
	}

	testCases := []TestCase{
		{
			description: "should discard first word",
			args:        "/scale 12",
			want:        "12",
		},
		{
			description: "should only discard first word",
			args:        "/scale 12 13",
			want:        "12 13",
		},
		{
			description: "empty on no args",
			args:        "/scale",
			want:        "",
		},
		{
			description: "empty on no input",
			args:        "",
			want:        "",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.description, func(t *testing.T) {
			got := ParseCommandArgs(testCase.args)

			assert.Equal(t, testCase.want, got)
		})
	}
}

func TestParseCommand(t *testing.T) {
	type TestCase struct {
		description string
		args        string
		want        string
	}

	testCases := []TestCase{
		{
			description: "should return first word",
			args:        "/chat",
			want:        "/chat",
		},
		{
			description: "should discard following word",
			args:        "/chat prompt",
			want:        "/chat",
		},
		{
			description: "should discard following words",
			args:        "/chat prompt something",
			want:        "/chat",
		},
		{
			description: "empty on no input",
			args:        "",
			want:        "",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.description, func(t *testing.T) {
			got := ParseCommand(testCase.args)

			assert.Equal(t, testCase.want, got)
		})
	}
}
