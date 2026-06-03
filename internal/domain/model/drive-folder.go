package model

type DriveFolderID string

type DriveFolder struct {
	ID       DriveFolderID  `json:"id"`
	ParentID *DriveFolderID `json:"parentId"`
	Name     string         `json:"name"`
}
