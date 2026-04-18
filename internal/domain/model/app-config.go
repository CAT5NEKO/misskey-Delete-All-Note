package model

type AppConfig struct {
	DeleteInterval    int
	KeepWithReactions bool
	KeepWithRenotes   bool
}

func NewAppConfig(interval int, keepReactions bool, keepRenotes bool) *AppConfig {
	return &AppConfig{
		DeleteInterval:    interval,
		KeepWithReactions: keepReactions,
		KeepWithRenotes:   keepRenotes,
	}
}

func (c *AppConfig) IsSafeInterval() bool {
	return c.DeleteInterval >= 10
}
