package model

type AppConfig struct {
	DeleteInterval      int
	DeleteOlderThanDays int
	KeepWithReactions   bool
	KeepWithRenotes     bool
}

func NewAppConfig(interval int, deleteOlderThanDays int, keepReactions bool, keepRenotes bool) *AppConfig {
	return &AppConfig{
		DeleteInterval:      interval,
		DeleteOlderThanDays: deleteOlderThanDays,
		KeepWithReactions:   keepReactions,
		KeepWithRenotes:     keepRenotes,
	}
}

func (c *AppConfig) IsSafeInterval() bool {
	return c.DeleteInterval >= 10
}
