package command

import (
	"context"
	"hsbot/internal/core/domain"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type MockResponder struct {
	command string
}

func (m *MockResponder) Respond(_ context.Context, _ time.Duration, _ *domain.Message) error {
	return nil
}

func (m *MockResponder) GetCommand() string {
	return m.command
}

func TestRegister(t *testing.T) {
	cr := &Registry{}
	mr := &MockResponder{command: "/test"}

	cr.Register(mr)
	assert.Len(t, cr.commands, 1)
}

func TestGetNotRegistered(t *testing.T) {
	cr := &Registry{}

	_, err := cr.Get("test")
	require.Errorf(t, err, "can't fetch command, registry not initialized")
}

func TestGetCommandNotFound(t *testing.T) {
	cr := &Registry{}
	mr := &MockResponder{command: "/test"}

	cr.Register(mr)
	assert.Len(t, cr.commands, 1)

	_, err := cr.Get("/foo")
	require.Errorf(t, err, "command not found")
}

func TestGetCommandFound(t *testing.T) {
	cr := &Registry{}
	mr := &MockResponder{command: "/test"}

	cr.Register(mr)
	assert.Len(t, cr.commands, 1)

	cmd, err := cr.Get("/test")
	require.NoError(t, err)
	assert.NotNil(t, cmd)

	assert.Equal(t, "/test", cmd.GetCommand())
}

func TestListServices(t *testing.T) {
	cr := &Registry{}
	mr1 := &MockResponder{command: "/foo"}
	mr2 := &MockResponder{command: "/bar"}

	cr.Register(mr1)
	cr.Register(mr2)
	assert.Len(t, cr.commands, 2)

	list := cr.ListCommands()

	assert.Len(t, list, 2)
	assert.Contains(t, list, "/foo")
	assert.Contains(t, list, "/bar")
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
