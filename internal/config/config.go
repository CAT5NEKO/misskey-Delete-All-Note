package config

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Token string
	Host  string

	DeleteInterval int

	NoteOlderThan     time.Duration
	KeepWithReactions bool
	KeepWithRenotes   bool
	KeepConditionMode string

	DriveOlderThan time.Duration
	DriveMode      string

	SkipNotes bool

	DryRun    bool
	Yes       bool
	MaxDelete int
	Force     bool

	Verbose bool
	Quiet   bool

	LockFile string
}

var dayRe = regexp.MustCompile(`(\d+(?:\.\d+)?)d`)

func ParseDuration(s string) (time.Duration, error) {
	s = strings.TrimSpace(s)
	if s == "" || s == "0" {
		return 0, nil
	}

	var days float64
	if m := dayRe.FindStringSubmatch(s); m != nil {
		days, _ = strconv.ParseFloat(m[1], 64)
		s = dayRe.ReplaceAllString(s, "")
	}

	s = strings.TrimSpace(s)

	var d time.Duration
	if s != "" {
		var err error
		d, err = time.ParseDuration(s)
		if err != nil {
			return 0, fmt.Errorf("invalid duration: %w", err)
		}
	}

	if d < 0 || days < 0 {
		return 0, fmt.Errorf("negative duration not allowed")
	}

	return time.Duration(days*float64(24*time.Hour)) + d, nil
}

func defaultLockFilePath() string {
	return filepath.Join(os.TempDir(), "misskeyNotedel.lock")
}

func Load() (*Config, error) {
	raw := rawConfig{
		deleteInterval: envIntDefault("DELETE_INTERVAL", 30),
		maxDelete:      envIntDefault("MAX_DELETE", 0),
		lockFile:       envStrDefault("LOCK_FILE", defaultLockFilePath()),
	}

	readEnvString(&raw.token, "TOKEN", "")
	readEnvString(&raw.host, "HOST", "")
	readEnvString(&raw.noteOlderThan, "NOTE_OLDER_THAN", "")
	readEnvString(&raw.keepReactions, "KEEP_WITH_REACTIONS", "false")
	readEnvString(&raw.keepRenotes, "KEEP_WITH_RENOTES", "false")
	readEnvString(&raw.keepMode, "KEEP_CONDITION_MODE", "or")
	readEnvString(&raw.driveOlderThan, "DRIVE_OLDER_THAN", "")
	readEnvString(&raw.driveMode, "DRIVE_MODE", "none")
	readEnvString(&raw.skipNotes, "SKIP_NOTES", "false")

	raw.dryRun = envBool("DRY_RUN")
	raw.yes = envBool("YES")
	raw.force = envBool("FORCE")
	raw.verbose = envBool("VERBOSE")
	raw.quiet = envBool("QUIET")

	fs := flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	registerFlags(fs, &raw)
	if err := fs.Parse(os.Args[1:]); err != nil {
		return nil, err
	}

	return raw.toConfig()
}

type rawConfig struct {
	token, host                                       string
	noteOlderThan, keepReactions, keepRenotes, keepMode string
	driveOlderThan, driveMode, skipNotes, lockFile    string
	deleteInterval, maxDelete                         int
	dryRun, yes, force, verbose, quiet                bool
}

func registerFlags(fs *flag.FlagSet, r *rawConfig) {
	fs.StringVar(&r.token, "token", r.token, "Misskey API token")
	fs.StringVar(&r.host, "host", r.host, "Misskey host")
	fs.IntVar(&r.deleteInterval, "delete-interval", r.deleteInterval, "Seconds between each deletion (min 5)")
	fs.StringVar(&r.noteOlderThan, "note-older-than", r.noteOlderThan, "Only delete notes older than this duration (e.g. 7d, 12h, 30m)")
	fs.StringVar(&r.keepReactions, "keep-with-reactions", r.keepReactions, "Keep notes with reactions (true/false)")
	fs.StringVar(&r.keepRenotes, "keep-with-renotes", r.keepRenotes, "Keep notes that were renoted (true/false)")
	fs.StringVar(&r.keepMode, "keep-condition-mode", r.keepMode, "Condition mode: 'or' or 'and'")
	fs.StringVar(&r.driveOlderThan, "drive-older-than", r.driveOlderThan, "Only delete drive files older than this (e.g. 30d, 7d12h)")
	fs.StringVar(&r.driveMode, "drive-mode", r.driveMode, "Drive deletion mode: 'none', 'all', 'unused'")
	fs.StringVar(&r.skipNotes, "skip-notes", r.skipNotes, "Skip note deletion, only process drive files (true/false)")
	fs.BoolVar(&r.dryRun, "dry-run", r.dryRun, "Show targets without deleting")
	fs.BoolVar(&r.yes, "yes", r.yes, "Skip confirmation prompt")
	fs.IntVar(&r.maxDelete, "max-delete", r.maxDelete, "Stop after N deletions (0=unlimited)")
	fs.BoolVar(&r.force, "force", r.force, "Ignore existing lock file")
	fs.BoolVar(&r.verbose, "verbose", r.verbose, "Verbose output (show skip reasons)")
	fs.BoolVar(&r.verbose, "v", r.verbose, "Short for --verbose")
	fs.BoolVar(&r.quiet, "quiet", r.quiet, "Quiet mode (errors only)")
	fs.BoolVar(&r.quiet, "q", r.quiet, "Short for --quiet")
	fs.StringVar(&r.lockFile, "lock-file", r.lockFile, "Lock file path (default: OS temp dir)")
}

func (r *rawConfig) toConfig() (*Config, error) {
	if r.token == "" || r.host == "" {
		return nil, errors.New("TOKEN and HOST are required (set via --token/--host or environment variables)")
	}

	r.host = strings.TrimPrefix(r.host, "https://")
	r.host = strings.TrimPrefix(r.host, "http://")
	if r.host == "" {
		return nil, errors.New("invalid HOST")
	}

	noteOlderThan, err := ParseDuration(r.noteOlderThan)
	if err != nil {
		return nil, fmt.Errorf("invalid NOTE_OLDER_THAN %q: %w", r.noteOlderThan, err)
	}

	driveOlderThan, err := ParseDuration(r.driveOlderThan)
	if err != nil {
		return nil, fmt.Errorf("invalid DRIVE_OLDER_THAN %q: %w", r.driveOlderThan, err)
	}

	interval := r.deleteInterval
	if interval < 5 {
		interval = 30
	}

	keepMode := strings.ToLower(strings.TrimSpace(r.keepMode))
	if keepMode != "or" && keepMode != "and" {
		keepMode = "or"
	}

	driveMode := strings.ToLower(strings.TrimSpace(r.driveMode))
	if driveMode != "none" && driveMode != "all" && driveMode != "unused" {
		driveMode = "none"
	}

	maxDelete := r.maxDelete
	if maxDelete < 0 {
		maxDelete = 0
	}

	lockFile := r.lockFile
	if lockFile == "" {
		lockFile = defaultLockFilePath()
	}

	return &Config{
		Token:              r.token,
		Host:               r.host,
		DeleteInterval:     interval,
		NoteOlderThan:      noteOlderThan,
		KeepWithReactions:  parseBoolOr(r.keepReactions, false),
		KeepWithRenotes:    parseBoolOr(r.keepRenotes, false),
		KeepConditionMode:  keepMode,
		DriveOlderThan:     driveOlderThan,
		DriveMode:          driveMode,
		SkipNotes:          parseBoolOr(r.skipNotes, false),
		DryRun:             r.dryRun,
		Yes:                r.yes,
		MaxDelete:          maxDelete,
		Force:              r.force,
		Verbose:            r.verbose,
		Quiet:              r.quiet,
		LockFile:           lockFile,
	}, nil
}

func readEnvString(dst *string, key, def string) {
	v := os.Getenv(key)
	if v == "" {
		*dst = def
	} else {
		*dst = v
	}
}

func envStrDefault(key, def string) string {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	return v
}

func envIntDefault(key string, def int) int {
	s := os.Getenv(key)
	if s == "" {
		return def
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return v
}

func envBool(key string) bool {
	v, err := strconv.ParseBool(os.Getenv(key))
	return err == nil && v
}

func parseBoolOr(s string, def bool) bool {
	if s == "" {
		return def
	}
	v, err := strconv.ParseBool(s)
	if err != nil {
		return def
	}
	return v
}
