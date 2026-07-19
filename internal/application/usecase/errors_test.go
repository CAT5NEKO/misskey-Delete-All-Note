package usecase

import (
	"errors"
	"testing"
)

func TestIsNotFoundError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"NO_SUCH_NOTE", errors.New("API error [NO_SUCH_NOTE] (HTTP 400): no such note"), true},
		{"NO_SUCH_FILE", errors.New("API error [NO_SUCH_FILE] (HTTP 400): no such file"), true},
		{"HTTP 404", errors.New("API error [] (HTTP 404): not found"), true},
		{"other error", errors.New("some other error"), false},
		{"auth error", errors.New("API error [AUTHENTICATION_FAILED] (HTTP 401): invalid token"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isNotFoundError(tt.err); got != tt.want {
				t.Errorf("isNotFoundError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsNonPublicRenoteError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"match", errors.New("API error [] (HTTP 500): renderAnnounce: cannot render non-public note"), true},
		{"other", errors.New("something else"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isNonPublicRenoteError(tt.err); got != tt.want {
				t.Errorf("isNonPublicRenoteError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsAuthError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"HTTP 401", errors.New("API error [] (HTTP 401): unauthorized"), true},
		{"HTTP 403", errors.New("API error [] (HTTP 403): forbidden"), true},
		{"HTTP 404", errors.New("API error [] (HTTP 404): not found"), false},
		{"other", errors.New("network error"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isAuthError(tt.err); got != tt.want {
				t.Errorf("isAuthError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsRateLimitError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"HTTP 429", errors.New("API error [] (HTTP 429): too many requests"), true},
		{"HTTP 400", errors.New("API error [] (HTTP 400): bad request"), false},
		{"other", errors.New("timeout"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isRateLimitError(tt.err); got != tt.want {
				t.Errorf("isRateLimitError() = %v, want %v", got, tt.want)
			}
		})
	}
}
