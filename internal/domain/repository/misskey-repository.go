package repository

import "misskeyNotedel/internal/domain/model"

type MisskeyRepository interface {
	FetchUser() (*model.User, error)
	FetchNotes(userID model.UserID, untilID model.NoteID) ([]model.Note, error)
	DeleteNote(noteID model.NoteID) error
	UnpinNote(noteID model.NoteID) error
}
