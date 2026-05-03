package repository

import "misskeyNotedel/internal/domain/model"

type FetchNotesOptions struct {
	WithReplies      bool
	WithChannelNotes bool
}

type MisskeyRepository interface {
	FetchUser() (*model.User, error)
	FetchNotes(userID model.UserID, untilID model.NoteID, opts FetchNotesOptions) ([]model.Note, error)
	DeleteNote(noteID model.NoteID) error
	UnpinNote(noteID model.NoteID) error
}
