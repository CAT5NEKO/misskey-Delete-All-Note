package model

type UserID string

type User struct {
	ID          UserID       `json:"id"`
	Name        string       `json:"name"`
	Username    string       `json:"username"`
	AvatarID    *DriveFileID `json:"avatarId"`
	BannerID    *DriveFileID `json:"bannerId"`
	NotesCount  int          `json:"notesCount"`
	PinnedNotes []Note       `json:"pinnedNotes"`
}
