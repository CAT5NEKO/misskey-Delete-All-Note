package model

import "time"

type AppConfig struct {
	DeleteInterval    int
	NoteOlderThan     time.Duration
	KeepWithReactions bool
	KeepWithRenotes   bool
	KeepConditionMode string
	DriveOlderThan    time.Duration
	DriveMode         string
	SkipNotes         bool
	DryRun            bool
	Yes               bool
	MaxDelete         int
	Force             bool
	Verbose           bool
	Quiet             bool
	LockFile          string
}

func (c *AppConfig) IsSafeInterval() bool {
	return c.DeleteInterval >= 10
}
