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
	FetchDriveFiles(folderID *model.DriveFolderID, untilID model.DriveFileID) ([]model.DriveFile, error)
	FetchDriveFolders(parentID *model.DriveFolderID, untilID model.DriveFolderID) ([]model.DriveFolder, error)
	DeleteDriveFile(fileID model.DriveFileID) error
	DriveFileHasAttachedNotes(fileID model.DriveFileID) (bool, error)
}
