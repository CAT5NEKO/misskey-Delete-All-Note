package model

import (
	"testing"
	"time"
)

func TestNote_ShouldKeep(t *testing.T) {
	strPtr := func(s string) *string { return &s }

	tests := []struct {
		name     string
		note     Note
		config   AppConfig
		expected bool
	}{
		{
			name:     "NoReactionsNoRenotes_ShouldNotKeep",
			note:     Note{Reactions: map[string]int{}, RenoteCount: 0},
			config:   AppConfig{KeepWithReactions: true, KeepWithRenotes: true},
			expected: false,
		},
		{
			name:     "WithReactions_FlagTrue_ShouldKeep",
			note:     Note{Reactions: map[string]int{"like": 1}, RenoteCount: 0},
			config:   AppConfig{KeepWithReactions: true, KeepWithRenotes: false},
			expected: true,
		},
		{
			name:     "WithReactions_FlagFalse_ShouldNotKeep",
			note:     Note{Reactions: map[string]int{"like": 1}, RenoteCount: 0},
			config:   AppConfig{KeepWithReactions: false, KeepWithRenotes: false},
			expected: false,
		},
		{
			name:     "WithRenotes_FlagTrue_ShouldKeep",
			note:     Note{Reactions: map[string]int{}, RenoteCount: 5},
			config:   AppConfig{KeepWithReactions: false, KeepWithRenotes: true},
			expected: true,
		},
		{
			name:     "WithRenotes_FlagFalse_ShouldNotKeep",
			note:     Note{Reactions: map[string]int{}, RenoteCount: 5},
			config:   AppConfig{KeepWithReactions: false, KeepWithRenotes: false},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.note.ShouldKeep(&tt.config); got != tt.expected {
				t.Errorf("Note.ShouldKeep() = %v, want %v", got, tt.expected)
			}
		})
	}

	t.Run("GetSummary", func(t *testing.T) {
		n1 := Note{Text: strPtr("Hello World"), CW: nil}
		if n1.GetSummary() != "Hello World" {
			t.Errorf("GetSummary failed: %s", n1.GetSummary())
		}

		n2 := Note{Text: strPtr("Body"), CW: strPtr("Warning")}
		if n2.GetSummary() != "Warning" {
			t.Errorf("GetSummary should prioritize CW: %s", n2.GetSummary())
		}

		n3 := Note{Text: strPtr("This is a very long text that should be truncated at some point")}
		expected := "This is a very long ..."
		if n3.GetSummary() != expected {
			t.Errorf("GetSummary failed truncation: got %s, want %s", n3.GetSummary(), expected)
		}
	})

	t.Run("KindLabel", func(t *testing.T) {
		renoteID := NoteID("r1")

		n1 := Note{ID: "n1"}
		if got := n1.KindLabel(); got != "note" {
			t.Errorf("KindLabel note failed: got %s", got)
		}

		n2 := Note{ID: "n2", RenoteID: &renoteID}
		if got := n2.KindLabel(); got != "renote" {
			t.Errorf("KindLabel renote failed: got %s", got)
		}

		n3 := Note{ID: "n3", RenoteID: &renoteID, Text: strPtr("comment")}
		if got := n3.KindLabel(); got != "quote-renote" {
			t.Errorf("KindLabel quote-renote failed: got %s", got)
		}
	})
}

func TestAppConfig_IsSafeInterval(t *testing.T) {
	c1 := AppConfig{DeleteInterval: 10}
	if !c1.IsSafeInterval() {
		t.Error("10 should be safe")
	}

	c2 := AppConfig{DeleteInterval: 9}
	if c2.IsSafeInterval() {
		t.Error("9 should not be safe")
	}
}

func TestNote_ShouldKeep_WithAgeFilter(t *testing.T) {
	now := time.Now()
	oldNote := Note{CreatedAt: now.Add(-72 * time.Hour)}
	newNote := Note{CreatedAt: now.Add(-12 * time.Hour)}
	config := AppConfig{DeleteOlderThanDays: 2}

	if got := oldNote.ShouldKeep(&config); got {
		t.Error("old note should not be kept by age filter")
	}

	if got := newNote.ShouldKeep(&config); !got {
		t.Error("new note should be kept by age filter")
	}
}
