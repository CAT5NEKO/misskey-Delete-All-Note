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
	APIToken     string
	InstanceHost string
	Scheme       string

	DeleteIntervalSeconds int

	NoteOlderThan     time.Duration
	KeepWithReactions bool
	KeepWithRenotes   bool
	KeepConditionMode string

	DriveOlderThan  time.Duration
	DriveDeleteMode string

	SkipNoteDeletion bool

	DryRun         bool
	AssumeYes      bool
	MaxDeleteCount int
	ForceUnlock    bool

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
		deleteIntervalSeconds: envIntDefault("DELETE_INTERVAL_SECONDS", 30),
		maxDeleteCount:        envIntDefault("MAX_DELETE_COUNT", 0),
		lockFile:              envStrDefault("LOCK_FILE", defaultLockFilePath()),
	}

	readEnvString(&raw.apiToken, "API_TOKEN", "")
	readEnvString(&raw.instanceHost, "INSTANCE_HOST", "")
	readEnvString(&raw.noteOlderThan, "NOTE_OLDER_THAN", "")
	readEnvString(&raw.keepReactions, "KEEP_WITH_REACTIONS", "false")
	readEnvString(&raw.keepRenotes, "KEEP_WITH_RENOTES", "false")
	readEnvString(&raw.keepMode, "KEEP_CONDITION_MODE", "or")
	readEnvString(&raw.driveOlderThan, "DRIVE_OLDER_THAN", "")
	readEnvString(&raw.driveDeleteMode, "DRIVE_DELETE_MODE", "none")
	readEnvString(&raw.skipNoteDeletion, "SKIP_NOTE_DELETION", "false")

	raw.dryRun = envBool("DRY_RUN")
	raw.assumeYes = envBool("ASSUME_YES")
	raw.forceUnlock = envBool("FORCE_UNLOCK")
	raw.verbose = envBool("VERBOSE")
	raw.quiet = envBool("QUIET")

	fs := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	registerFlags(fs, &raw)
	if err := fs.Parse(os.Args[1:]); err != nil {
		return nil, err
	}

	return raw.toConfig()
}

type rawConfig struct {
	apiToken, instanceHost                              string
	noteOlderThan, keepReactions, keepRenotes, keepMode string
	driveOlderThan, driveDeleteMode, skipNoteDeletion   string
	lockFile                                            string
	deleteIntervalSeconds, maxDeleteCount               int
	dryRun, assumeYes, forceUnlock, verbose, quiet      bool
}

func registerFlags(fs *flag.FlagSet, r *rawConfig) {
	fs.StringVar(&r.apiToken, "api-token", r.apiToken, "Misskey API token")
	fs.StringVar(&r.instanceHost, "instance-host", r.instanceHost, "Misskey instance host")
	fs.IntVar(&r.deleteIntervalSeconds, "delete-interval-seconds", r.deleteIntervalSeconds, "Seconds between each deletion (min 5)")
	fs.StringVar(&r.noteOlderThan, "note-older-than", r.noteOlderThan, "Only delete notes older than this duration (e.g. 7d, 12h, 30m)")
	fs.StringVar(&r.keepReactions, "keep-with-reactions", r.keepReactions, "Keep notes with reactions (true/false)")
	fs.StringVar(&r.keepRenotes, "keep-with-renotes", r.keepRenotes, "Keep notes that were renoted (true/false)")
	fs.StringVar(&r.keepMode, "keep-condition-mode", r.keepMode, "Condition mode: 'or' or 'and'")
	fs.StringVar(&r.driveOlderThan, "drive-older-than", r.driveOlderThan, "Only delete drive files older than this (e.g. 30d, 7d12h)")
	fs.StringVar(&r.driveDeleteMode, "drive-delete-mode", r.driveDeleteMode, "Drive deletion mode: 'none', 'all', 'unused'")
	fs.StringVar(&r.skipNoteDeletion, "skip-note-deletion", r.skipNoteDeletion, "Skip note deletion, only process drive files (true/false)")
	fs.BoolVar(&r.dryRun, "dry-run", r.dryRun, "Show targets without deleting")
	fs.BoolVar(&r.assumeYes, "assume-yes", r.assumeYes, "Skip confirmation prompt")
	fs.IntVar(&r.maxDeleteCount, "max-delete-count", r.maxDeleteCount, "Stop after N deletions (0=unlimited)")
	fs.BoolVar(&r.forceUnlock, "force-unlock", r.forceUnlock, "Ignore existing lock file")
	fs.BoolVar(&r.verbose, "verbose", r.verbose, "Verbose output (show skip reasons)")
	fs.BoolVar(&r.verbose, "v", r.verbose, "Short for --verbose")
	fs.BoolVar(&r.quiet, "quiet", r.quiet, "Quiet mode (errors only)")
	fs.BoolVar(&r.quiet, "q", r.quiet, "Short for --quiet")
	fs.StringVar(&r.lockFile, "lock-file", r.lockFile, "Lock file path (default: OS temp dir)")
}

func (r *rawConfig) toConfig() (*Config, error) {
	if r.apiToken == "" || r.instanceHost == "" {
		return nil, errors.New("API_TOKEN and INSTANCE_HOST are required (set via --api-token/--instance-host or environment variables)")
	}

	scheme := "https"
	if strings.HasPrefix(r.instanceHost, "http://") {
		scheme = "http"
		r.instanceHost = strings.TrimPrefix(r.instanceHost, "http://")
	} else {
		r.instanceHost = strings.TrimPrefix(r.instanceHost, "https://")
	}
	if r.instanceHost == "" {
		return nil, errors.New("invalid INSTANCE_HOST")
	}

	noteOlderThan, err := ParseDuration(r.noteOlderThan)
	if err != nil {
		return nil, fmt.Errorf("invalid NOTE_OLDER_THAN %q: %w", r.noteOlderThan, err)
	}

	driveOlderThan, err := ParseDuration(r.driveOlderThan)
	if err != nil {
		return nil, fmt.Errorf("invalid DRIVE_OLDER_THAN %q: %w", r.driveOlderThan, err)
	}

	interval := r.deleteIntervalSeconds
	if interval < 5 {
		interval = 30
	}

	keepMode := strings.ToLower(strings.TrimSpace(r.keepMode))
	if keepMode != "or" && keepMode != "and" {
		keepMode = "or"
	}

	driveMode := strings.ToLower(strings.TrimSpace(r.driveDeleteMode))
	if driveMode != "none" && driveMode != "all" && driveMode != "unused" {
		driveMode = "none"
	}

	maxDeleteCount := r.maxDeleteCount
	if maxDeleteCount < 0 {
		maxDeleteCount = 0
	}

	lockFile := r.lockFile
	if lockFile == "" {
		lockFile = defaultLockFilePath()
	}

	return &Config{
		APIToken:              r.apiToken,
		InstanceHost:          r.instanceHost,
		Scheme:                scheme,
		DeleteIntervalSeconds: interval,
		NoteOlderThan:         noteOlderThan,
		KeepWithReactions:     parseBoolOr(r.keepReactions, false),
		KeepWithRenotes:       parseBoolOr(r.keepRenotes, false),
		KeepConditionMode:     keepMode,
		DriveOlderThan:        driveOlderThan,
		DriveDeleteMode:       driveMode,
		SkipNoteDeletion:      parseBoolOr(r.skipNoteDeletion, false),
		DryRun:                r.dryRun,
		AssumeYes:             r.assumeYes,
		MaxDeleteCount:        maxDeleteCount,
		ForceUnlock:           r.forceUnlock,
		Verbose:               r.verbose,
		Quiet:                 r.quiet,
		LockFile:              lockFile,
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
