package model

import "time"

type AppConfig struct {
	DeleteIntervalSeconds int
	NoteOlderThan         time.Duration
	KeepWithReactions     bool
	KeepWithRenotes       bool
	KeepConditionMode     string
	DriveOlderThan        time.Duration
	DriveDeleteMode       string
	SkipNoteDeletion      bool
	DryRun                bool
	AssumeYes             bool
	MaxDeleteCount        int
	ForceUnlock           bool
	Verbose               bool
	Quiet                 bool
	LockFile              string
}

func (c *AppConfig) IsSafeInterval() bool {
	return c.DeleteIntervalSeconds >= 10
}
