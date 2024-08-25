package domain_test

import (
	"hsbot/internal/core/domain"
	"testing"

	"github.com/stretchr/testify/assert"
)

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
			got := domain.ParseCommandArgs(testCase.args)

			assert.Equal(t, testCase.want, got)
		})
	}
}
