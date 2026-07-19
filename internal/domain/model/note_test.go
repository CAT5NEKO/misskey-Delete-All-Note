package model

import (
	"testing"
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
			config:   AppConfig{KeepWithReactions: true, KeepWithRenotes: true, KeepConditionMode: "or"},
			expected: false,
		},
		{
			name:     "WithReactions_FlagTrue_OrMode_ShouldKeep",
			note:     Note{Reactions: map[string]int{"like": 1}, RenoteCount: 0},
			config:   AppConfig{KeepWithReactions: true, KeepWithRenotes: false, KeepConditionMode: "or"},
			expected: true,
		},
		{
			name:     "WithReactions_FlagFalse_ShouldNotKeep",
			note:     Note{Reactions: map[string]int{"like": 1}, RenoteCount: 0},
			config:   AppConfig{KeepWithReactions: false, KeepWithRenotes: false, KeepConditionMode: "or"},
			expected: false,
		},
		{
			name:     "WithRenotes_FlagTrue_OrMode_ShouldKeep",
			note:     Note{Reactions: map[string]int{}, RenoteCount: 5},
			config:   AppConfig{KeepWithReactions: false, KeepWithRenotes: true, KeepConditionMode: "or"},
			expected: true,
		},
		{
			name:     "WithRenotes_FlagFalse_ShouldNotKeep",
			note:     Note{Reactions: map[string]int{}, RenoteCount: 5},
			config:   AppConfig{KeepWithReactions: false, KeepWithRenotes: false, KeepConditionMode: "or"},
			expected: false,
		},
		// AND mode tests
		{
			name:     "AndMode_BothFlags_BothProperties_ShouldKeep",
			note:     Note{Reactions: map[string]int{"like": 1}, RenoteCount: 5},
			config:   AppConfig{KeepWithReactions: true, KeepWithRenotes: true, KeepConditionMode: "and"},
			expected: true,
		},
		{
			name:     "AndMode_BothFlags_OnlyReactions_ShouldNotKeep",
			note:     Note{Reactions: map[string]int{"like": 1}, RenoteCount: 0},
			config:   AppConfig{KeepWithReactions: true, KeepWithRenotes: true, KeepConditionMode: "and"},
			expected: false,
		},
		{
			name:     "AndMode_BothFlags_OnlyRenotes_ShouldNotKeep",
			note:     Note{Reactions: map[string]int{}, RenoteCount: 5},
			config:   AppConfig{KeepWithReactions: true, KeepWithRenotes: true, KeepConditionMode: "and"},
			expected: false,
		},
		{
			name:     "AndMode_BothFlags_Neither_ShouldNotKeep",
			note:     Note{Reactions: map[string]int{}, RenoteCount: 0},
			config:   AppConfig{KeepWithReactions: true, KeepWithRenotes: true, KeepConditionMode: "and"},
			expected: false,
		},
		{
			name:     "AndMode_OnlyReactionsFlag_HasReactions_ShouldKeep",
			note:     Note{Reactions: map[string]int{"like": 1}, RenoteCount: 0},
			config:   AppConfig{KeepWithReactions: true, KeepWithRenotes: false, KeepConditionMode: "and"},
			expected: true,
		},
		{
			name:     "AndMode_OnlyReactionsFlag_NoReactions_ShouldNotKeep",
			note:     Note{Reactions: map[string]int{}, RenoteCount: 0},
			config:   AppConfig{KeepWithReactions: true, KeepWithRenotes: false, KeepConditionMode: "and"},
			expected: false,
		},
		{
			name:     "AndMode_OnlyRenotesFlag_HasRenotes_ShouldKeep",
			note:     Note{Reactions: map[string]int{}, RenoteCount: 5},
			config:   AppConfig{KeepWithReactions: false, KeepWithRenotes: true, KeepConditionMode: "and"},
			expected: true,
		},
		{
			name:     "AndMode_OnlyRenotesFlag_NoRenotes_ShouldNotKeep",
			note:     Note{Reactions: map[string]int{}, RenoteCount: 0},
			config:   AppConfig{KeepWithReactions: false, KeepWithRenotes: true, KeepConditionMode: "and"},
			expected: false,
		},
		{
			name:     "AndMode_NeitherFlag_ShouldNotKeep",
			note:     Note{Reactions: map[string]int{"like": 1}, RenoteCount: 5},
			config:   AppConfig{KeepWithReactions: false, KeepWithRenotes: false, KeepConditionMode: "and"},
			expected: false,
		},
		// Default mode (empty) should behave as "or"
		{
			name:     "DefaultMode_WithReactions_ShouldKeep",
			note:     Note{Reactions: map[string]int{"like": 1}, RenoteCount: 0},
			config:   AppConfig{KeepWithReactions: true, KeepWithRenotes: false},
			expected: true,
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
