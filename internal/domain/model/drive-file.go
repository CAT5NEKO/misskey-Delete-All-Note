package model

import "time"

type DriveFileID string

type DriveFile struct {
	ID        DriveFileID `json:"id"`
	CreatedAt time.Time   `json:"createdAt"`
	Name      string      `json:"name"`
	Type      string      `json:"type"`
	Size      int64       `json:"size"`
}
