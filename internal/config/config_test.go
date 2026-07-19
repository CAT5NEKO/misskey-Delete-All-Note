package config

import (
	"os"
	"testing"
	"time"
)

func TestParseDuration(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    time.Duration
		wantErr bool
	}{
		{"empty", "", 0, false},
		{"zero string", "0", 0, false},
		{"zero seconds", "0s", 0, false},
		{"30 minutes", "30m", 30 * time.Minute, false},
		{"1 hour", "1h", time.Hour, false},
		{"1.5 hours", "1.5h", 90 * time.Minute, false},
		{"12 hours", "12h", 12 * time.Hour, false},
		{"1 day", "1d", 24 * time.Hour, false},
		{"7 days", "7d", 7 * 24 * time.Hour, false},
		{"half day", "0.5d", 12 * time.Hour, false},
		{"7 days 12 hours 30 minutes", "7d12h30m", 7*24*time.Hour + 12*time.Hour + 30*time.Minute, false},
		{"hours and minutes", "1h30m", 90 * time.Minute, false},
		{"with spaces", " 7d 12h ", 7*24*time.Hour + 12*time.Hour, false},
		{"days only with trailing space", "7d ", 7 * 24 * time.Hour, false},
		{"negative", "-7d", 0, true},
		{"negative hours", "-1h", 0, true},
		{"invalid", "abc", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseDuration(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseDuration(%q) expected error, got nil", tt.input)
				}
				return
			}
			if err != nil {
				t.Errorf("ParseDuration(%q) unexpected error: %v", tt.input, err)
				return
			}
			if got != tt.want {
				t.Errorf("ParseDuration(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseBoolOr(t *testing.T) {
	tests := []struct {
		input string
		def   bool
		want  bool
	}{
		{"", false, false},
		{"", true, true},
		{"true", false, true},
		{"false", true, false},
		{"1", false, true},
		{"0", true, false},
		{"invalid", false, false},
		{"invalid", true, true},
	}
	for _, tt := range tests {
		got := parseBoolOr(tt.input, tt.def)
		if got != tt.want {
			t.Errorf("parseBoolOr(%q, %v) = %v, want %v", tt.input, tt.def, got, tt.want)
		}
	}
}

func TestLoad_TokenHostRequired(t *testing.T) {
	os.Unsetenv("TOKEN")
	os.Unsetenv("HOST")
	os.Args = []string{"test"}

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for missing TOKEN/HOST")
	}
}

func TestLoad_HostStripProtocol(t *testing.T) {
	os.Setenv("TOKEN", "test-token")
	os.Setenv("HOST", "https://misskey.example.com")
	defer os.Unsetenv("TOKEN")
	defer os.Unsetenv("HOST")
	os.Args = []string{"test"}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Host != "misskey.example.com" {
		t.Errorf("expected host without protocol, got %q", cfg.Host)
	}
}

func TestLoad_Defaults(t *testing.T) {
	os.Setenv("TOKEN", "test-token")
	os.Setenv("HOST", "misskey.example.com")
	defer os.Unsetenv("TOKEN")
	defer os.Unsetenv("HOST")
	os.Args = []string{"test"}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.DeleteInterval != 30 {
		t.Errorf("default DeleteInterval = %d, want 30", cfg.DeleteInterval)
	}
	if cfg.KeepConditionMode != "or" {
		t.Errorf("default KeepConditionMode = %q, want 'or'", cfg.KeepConditionMode)
	}
	if cfg.DriveMode != "none" {
		t.Errorf("default DriveMode = %q, want 'none'", cfg.DriveMode)
	}
}

func TestLoad_DeleteIntervalClamped(t *testing.T) {
	os.Setenv("TOKEN", "test-token")
	os.Setenv("HOST", "misskey.example.com")
	os.Setenv("DELETE_INTERVAL", "2")
	defer os.Unsetenv("TOKEN")
	defer os.Unsetenv("HOST")
	defer os.Unsetenv("DELETE_INTERVAL")
	os.Args = []string{"test"}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.DeleteInterval != 30 {
		t.Errorf("expected interval clamped to 30, got %d", cfg.DeleteInterval)
	}
}

func TestLoad_EnvKeepConditionMode(t *testing.T) {
	os.Setenv("TOKEN", "test-token")
	os.Setenv("HOST", "misskey.example.com")
	os.Setenv("KEEP_CONDITION_MODE", "and")
	os.Setenv("KEEP_WITH_REACTIONS", "true")
	os.Setenv("KEEP_WITH_RENOTES", "true")
	defer os.Unsetenv("TOKEN")
	defer os.Unsetenv("HOST")
	defer os.Unsetenv("KEEP_CONDITION_MODE")
	defer os.Unsetenv("KEEP_WITH_REACTIONS")
	defer os.Unsetenv("KEEP_WITH_RENOTES")
	os.Args = []string{"test"}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.KeepConditionMode != "and" {
		t.Errorf("KeepConditionMode = %q, want 'and'", cfg.KeepConditionMode)
	}
	if !cfg.KeepWithReactions {
		t.Error("KeepWithReactions should be true")
	}
	if !cfg.KeepWithRenotes {
		t.Error("KeepWithRenotes should be true")
	}
}

func TestLoad_InvalidKeepModeDefaults(t *testing.T) {
	os.Setenv("TOKEN", "test-token")
	os.Setenv("HOST", "misskey.example.com")
	os.Setenv("KEEP_CONDITION_MODE", "invalid")
	defer os.Unsetenv("TOKEN")
	defer os.Unsetenv("HOST")
	defer os.Unsetenv("KEEP_CONDITION_MODE")
	os.Args = []string{"test"}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.KeepConditionMode != "or" {
		t.Errorf("expected fallback to 'or', got %q", cfg.KeepConditionMode)
	}
}

func TestLoad_InvalidDriveModeDefaults(t *testing.T) {
	os.Setenv("TOKEN", "test-token")
	os.Setenv("HOST", "misskey.example.com")
	os.Setenv("DRIVE_MODE", "everything")
	defer os.Unsetenv("TOKEN")
	defer os.Unsetenv("HOST")
	defer os.Unsetenv("DRIVE_MODE")
	os.Args = []string{"test"}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.DriveMode != "none" {
		t.Errorf("expected fallback to 'none', got %q", cfg.DriveMode)
	}
}

func TestLoad_NegativeMaxDelete(t *testing.T) {
	os.Setenv("TOKEN", "test-token")
	os.Setenv("HOST", "misskey.example.com")
	os.Setenv("MAX_DELETE", "-5")
	defer os.Unsetenv("TOKEN")
	defer os.Unsetenv("HOST")
	defer os.Unsetenv("MAX_DELETE")
	os.Args = []string{"test"}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.MaxDelete != 0 {
		t.Errorf("expected MaxDelete=0, got %d", cfg.MaxDelete)
	}
}

func TestLoad_EnvBoolFlags(t *testing.T) {
	os.Setenv("TOKEN", "test-token")
	os.Setenv("HOST", "misskey.example.com")
	os.Setenv("DRY_RUN", "true")
	os.Setenv("YES", "true")
	os.Setenv("FORCE", "true")
	os.Setenv("VERBOSE", "true")
	os.Setenv("QUIET", "1")
	defer os.Unsetenv("TOKEN")
	defer os.Unsetenv("HOST")
	defer os.Unsetenv("DRY_RUN")
	defer os.Unsetenv("YES")
	defer os.Unsetenv("FORCE")
	defer os.Unsetenv("VERBOSE")
	defer os.Unsetenv("QUIET")
	os.Args = []string{"test"}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cfg.DryRun {
		t.Error("DryRun should be true from env")
	}
	if !cfg.Yes {
		t.Error("Yes should be true from env")
	}
	if !cfg.Force {
		t.Error("Force should be true from env")
	}
	if !cfg.Verbose {
		t.Error("Verbose should be true from env")
	}
	if !cfg.Quiet {
		t.Error("Quiet should be true from env")
	}
}

func TestLoad_NoteOlderThanDuration(t *testing.T) {
	os.Setenv("TOKEN", "test-token")
	os.Setenv("HOST", "misskey.example.com")
	os.Setenv("NOTE_OLDER_THAN", "12h30m")
	defer os.Unsetenv("TOKEN")
	defer os.Unsetenv("HOST")
	defer os.Unsetenv("NOTE_OLDER_THAN")
	os.Args = []string{"test"}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := 12*time.Hour + 30*time.Minute
	if cfg.NoteOlderThan != want {
		t.Errorf("NoteOlderThan = %v, want %v", cfg.NoteOlderThan, want)
	}
}

func TestLoad_InvalidDuration(t *testing.T) {
	os.Setenv("TOKEN", "test-token")
	os.Setenv("HOST", "misskey.example.com")
	os.Setenv("NOTE_OLDER_THAN", "xyzzy")
	defer os.Unsetenv("TOKEN")
	defer os.Unsetenv("HOST")
	defer os.Unsetenv("NOTE_OLDER_THAN")
	os.Args = []string{"test"}

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for invalid duration")
	}
}
