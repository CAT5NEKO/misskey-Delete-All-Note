package model

type AppConfig struct {
	DeleteInterval        int
	DeleteOlderThanDays   int
	KeepWithReactions     bool
	KeepWithRenotes       bool
	DeleteDriveFiles      bool
	DeleteDriveUnusedOnly bool
	DeleteDriveOnly       bool
}

func NewAppConfig(interval int, deleteOlderThanDays int, keepReactions bool, keepRenotes bool, deleteDriveFiles bool, deleteDriveUnusedOnly bool, deleteDriveOnly bool) *AppConfig {
	return &AppConfig{
		DeleteInterval:        interval,
		DeleteOlderThanDays:   deleteOlderThanDays,
		KeepWithReactions:     keepReactions,
		KeepWithRenotes:       keepRenotes,
		DeleteDriveFiles:      deleteDriveFiles,
		DeleteDriveUnusedOnly: deleteDriveUnusedOnly,
		DeleteDriveOnly:       deleteDriveOnly,
	}
}

func (c *AppConfig) IsSafeInterval() bool {
	return c.DeleteInterval >= 10
}
