package generator

import (
	"encoding/json"
	"hsbot/internal/core/domain"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFALGenerator_GenerateFromPrompt(t *testing.T) {
	tests := []struct {
		name           string
		responseBody   interface{}
		responseStatus int
		wantURL        string
		wantErr        bool
	}{
		{
			name: "success",
			responseBody: map[string]interface{}{
				"images": []interface{}{
					map[string]interface{}{"url": "http://img-url.com/1.png"},
				},
				"prompt": "flowers",
			},
			responseStatus: http.StatusOK,
			wantURL:        "http://img-url.com/1.png",
			wantErr:        false,
		},
		{
			name:           "api error",
			responseBody:   "invalid",
			responseStatus: http.StatusInternalServerError,
			wantURL:        "",
			wantErr:        true,
		},
		{
			name:           "malformed JSON",
			responseBody:   "{not_json}",
			responseStatus: http.StatusOK,
			wantURL:        "",
			wantErr:        true,
		},
		{
			name: "missing images",
			responseBody: map[string]interface{}{
				"images": []interface{}{},
				"prompt": "noimg",
			},
			responseStatus: http.StatusOK,
			wantURL:        "",
			wantErr:        true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tc.responseStatus)
				switch b := tc.responseBody.(type) {
				case string:
					w.Write([]byte(b))
				default:
					json.NewEncoder(w).Encode(b)
				}
			}))
			defer srv.Close()

			g := NewFAL(srv.URL, srv.URL, srv.URL, "test-api-key")

			got, err := g.GenerateFromPrompt(t.Context(), "test prompt")
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.wantURL, got)
			}
		})
	}
}

func TestFALGenerator_EditFromPrompt(t *testing.T) {
	tests := []struct {
		name           string
		input          domain.Prompt
		responseBody   interface{}
		responseStatus int
		wantURL        string
		wantErr        bool
	}{
		{
			name: "success",
			input: domain.Prompt{
				Prompt:   "a cat",
				ImageURL: "http://input-url.com/cat.png",
			},
			responseBody: map[string]interface{}{
				"images": []interface{}{
					map[string]interface{}{"url": "http://img-url.com/edit.png"},
				},
				"prompt": "a cat",
			},
			responseStatus: http.StatusOK,
			wantURL:        "http://img-url.com/edit.png",
			wantErr:        false,
		},
		{
			name: "missing prompt",
			input: domain.Prompt{
				Prompt:   "",
				ImageURL: "http://image.png",
			},
			wantErr: true,
		},
		{
			name: "missing image",
			input: domain.Prompt{
				Prompt:   "edit img",
				ImageURL: "",
			},
			wantErr: true,
		},
		{
			name: "malformed JSON",
			input: domain.Prompt{
				Prompt:   "bad json",
				ImageURL: "http://image.png",
			},
			responseBody:   "{not_json}",
			responseStatus: http.StatusOK,
			wantErr:        true,
		},
		{
			name: "missing images in response",
			input: domain.Prompt{
				Prompt:   "missing img",
				ImageURL: "http://image.png",
			},
			responseBody: map[string]interface{}{
				"images": []interface{}{},
				"prompt": "missing img",
			},
			responseStatus: http.StatusOK,
			wantErr:        true,
		},
		{
			name: "api error",
			input: domain.Prompt{
				Prompt:   "fail",
				ImageURL: "http://fail.png",
			},
			responseBody:   "err",
			responseStatus: http.StatusInternalServerError,
			wantErr:        true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tc.responseStatus)
				switch b := tc.responseBody.(type) {
				case string:
					w.Write([]byte(b))
				case nil:
					// For cases like missing prompt/image, handler should return before sending a request
				default:
					json.NewEncoder(w).Encode(b)
				}
			}))
			defer srv.Close()

			g := NewFAL(srv.URL, srv.URL, srv.URL, "test-api-key")
			ctx := t.Context()

			got, err := g.EditFromPrompt(ctx, tc.input)
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.wantURL, got)
			}
		})
	}
}

func TestFALGenerator_GenerateFromAudio(t *testing.T) {
	tests := []struct {
		name           string
		responseBody   interface{}
		responseStatus int
		wantText       string
		wantErr        bool
	}{
		{
			name: "success",
			responseBody: map[string]interface{}{
				"text": "This is a test transcript.",
			},
			responseStatus: http.StatusOK,
			wantText:       "This is a test transcript.",
			wantErr:        false,
		},
		{
			name:           "api error",
			responseBody:   "error",
			responseStatus: http.StatusInternalServerError,
			wantErr:        true,
		},
		{
			name:           "malformed JSON",
			responseBody:   "{badjson}",
			responseStatus: http.StatusOK,
			wantErr:        true,
		},
		{
			name:           "missing field",
			responseBody:   `{}`,
			responseStatus: http.StatusOK,
			wantText:       "",
			wantErr:        false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tc.responseStatus)
				switch b := tc.responseBody.(type) {
				case string:
					w.Write([]byte(b))
				case nil:
					// Should skip, not used here
				default:
					json.NewEncoder(w).Encode(b)
				}
			}))
			defer srv.Close()

			g := NewFAL(srv.URL, srv.URL, srv.URL, "test-api-key")
			ctx := t.Context()

			got, err := g.GenerateFromAudio(ctx, "http://audio-url.com/audio.wav")
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.wantText, got)
			}
		})
	}
}
