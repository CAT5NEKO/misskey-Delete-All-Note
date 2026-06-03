package usecase

import (
	"errors"
	"misskeyNotedel/internal/domain/model"
	"misskeyNotedel/internal/domain/repository"
	"testing"
	"time"
)

type mockRepository struct {
	fetchUserFunc                 func() (*model.User, error)
	fetchNotesFunc                func(userID model.UserID, untilID model.NoteID, opts repository.FetchNotesOptions) ([]model.Note, error)
	deleteNoteFunc                func(noteID model.NoteID) error
	unpinNoteFunc                 func(noteID model.NoteID) error
	fetchDriveFilesFunc           func(folderID *model.DriveFolderID, untilID model.DriveFileID) ([]model.DriveFile, error)
	fetchDriveFoldersFunc         func(parentID *model.DriveFolderID, untilID model.DriveFolderID) ([]model.DriveFolder, error)
	deleteDriveFileFunc           func(fileID model.DriveFileID) error
	driveFileHasAttachedNotesFunc func(fileID model.DriveFileID) (bool, error)
}

func (m *mockRepository) FetchUser() (*model.User, error) { return m.fetchUserFunc() }
func (m *mockRepository) FetchNotes(u model.UserID, i model.NoteID, opts repository.FetchNotesOptions) ([]model.Note, error) {
	return m.fetchNotesFunc(u, i, opts)
}
func (m *mockRepository) DeleteNote(i model.NoteID) error { return m.deleteNoteFunc(i) }
func (m *mockRepository) UnpinNote(i model.NoteID) error  { return m.unpinNoteFunc(i) }
func (m *mockRepository) FetchDriveFiles(folderID *model.DriveFolderID, i model.DriveFileID) ([]model.DriveFile, error) {
	if m.fetchDriveFilesFunc == nil {
		return nil, errors.New("FetchDriveFiles not implemented")
	}
	return m.fetchDriveFilesFunc(folderID, i)
}
func (m *mockRepository) FetchDriveFolders(parentID *model.DriveFolderID, i model.DriveFolderID) ([]model.DriveFolder, error) {
	if m.fetchDriveFoldersFunc == nil {
		return []model.DriveFolder{}, nil
	}
	return m.fetchDriveFoldersFunc(parentID, i)
}
func (m *mockRepository) DeleteDriveFile(i model.DriveFileID) error {
	if m.deleteDriveFileFunc == nil {
		return errors.New("DeleteDriveFile not implemented")
	}
	return m.deleteDriveFileFunc(i)
}
func (m *mockRepository) DriveFileHasAttachedNotes(i model.DriveFileID) (bool, error) {
	if m.driveFileHasAttachedNotesFunc == nil {
		return false, errors.New("DriveFileHasAttachedNotes not implemented")
	}
	return m.driveFileHasAttachedNotesFunc(i)
}

type mockLogger struct {
	infoMsgs  []string
	warnMsgs  []string
	errorMsgs []string
}

func (m *mockLogger) Info(msg string)           { m.infoMsgs = append(m.infoMsgs, msg) }
func (m *mockLogger) Warn(msg string)           { m.warnMsgs = append(m.warnMsgs, msg) }
func (m *mockLogger) Error(msg string, _ error) { m.errorMsgs = append(m.errorMsgs, msg) }

func TestDeleteNotesUseCase_Execute(t *testing.T) {
	t.Run("Success_NoTargets", func(t *testing.T) {
		repo := &mockRepository{
			fetchUserFunc: func() (*model.User, error) {
				return &model.User{ID: "u1", NotesCount: 0}, nil
			},
			fetchNotesFunc: func(_ model.UserID, _ model.NoteID, _ repository.FetchNotesOptions) ([]model.Note, error) {
				return []model.Note{}, nil
			},
		}
		logger := &mockLogger{}
		uc := NewDeleteNotesUseCase(repo, &model.AppConfig{DeleteInterval: 10}, logger)

		err := uc.Execute()
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		foundNoTargets := false
		for _, msg := range logger.infoMsgs {
			if msg == "No deletion targets found." {
				foundNoTargets = true
				break
			}
		}
		if !foundNoTargets {
			t.Error("Should have logged 'No deletion targets found.'")
		}
	})

	t.Run("Success_WithDeletions", func(t *testing.T) {
		deleteCount := 0
		unpinCount := 0
		repo := &mockRepository{
			fetchUserFunc: func() (*model.User, error) {
				return &model.User{
					ID:          "u1",
					NotesCount:  2,
					PinnedNotes: []model.Note{{ID: "p1"}},
				}, nil
			},
			unpinNoteFunc: func(id model.NoteID) error {
				unpinCount++
				return nil
			},
			fetchNotesFunc: func(_ model.UserID, until model.NoteID, opts repository.FetchNotesOptions) ([]model.Note, error) {
				if !opts.WithReplies {
					t.Fatal("expected replies to be included in the scan")
				}
				if !opts.WithChannelNotes {
					t.Fatal("expected channel notes to be included in the scan")
				}
				if until == "" {
					return []model.Note{
						{ID: "n1", RenoteCount: 0},
						{ID: "n2", RenoteCount: 10},
					}, nil
				}
				return []model.Note{}, nil
			},
			deleteNoteFunc: func(id model.NoteID) error {
				deleteCount++
				return nil
			},
		}
		logger := &mockLogger{}
		// Keep with renotes = true
		config := &model.AppConfig{DeleteInterval: 0, KeepWithRenotes: true}
		uc := NewDeleteNotesUseCase(repo, config, logger)

		err := uc.Execute()
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if unpinCount != 1 {
			t.Errorf("Expected 1 unpin, got %d", unpinCount)
		}
		if deleteCount != 1 {
			t.Errorf("Expected 1 delete (n1), got %d. n2 should have been kept.", deleteCount)
		}
	})

	t.Run("FetchUserError", func(t *testing.T) {
		repo := &mockRepository{
			fetchUserFunc: func() (*model.User, error) {
				return nil, errors.New("api error")
			},
		}
		logger := &mockLogger{}
		uc := NewDeleteNotesUseCase(repo, &model.AppConfig{}, logger)

		err := uc.Execute()
		if err == nil {
			t.Error("Expected error, got nil")
		}
	})

	t.Run("SkipKnownNonPublicRenoteDeleteError", func(t *testing.T) {
		renoteID := model.NoteID("src1")
		deleteCount := 0
		repo := &mockRepository{
			fetchUserFunc: func() (*model.User, error) {
				return &model.User{ID: "u1", NotesCount: 1}, nil
			},
			fetchNotesFunc: func(_ model.UserID, until model.NoteID, _ repository.FetchNotesOptions) ([]model.Note, error) {
				if until == "" {
					return []model.Note{{ID: "n1", RenoteID: &renoteID}}, nil
				}
				return []model.Note{}, nil
			},
			deleteNoteFunc: func(id model.NoteID) error {
				deleteCount++
				return errors.New("HTTP 500 returned from notes/delete: {\"error\":{\"info\":{\"e\":{\"message\":\"renderAnnounce: cannot render non-public note\"}}}}")
			},
			unpinNoteFunc: func(id model.NoteID) error {
				return nil
			},
		}

		logger := &mockLogger{}
		uc := NewDeleteNotesUseCase(repo, &model.AppConfig{DeleteInterval: 0}, logger)

		err := uc.Execute()
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if deleteCount != 1 {
			t.Fatalf("Expected 1 delete attempt, got %d", deleteCount)
		}
		if len(logger.warnMsgs) == 0 {
			t.Fatal("Expected a warning log for skipped renote")
		}
	})

	t.Run("Success_OnlyDeleteOlderThanDays", func(t *testing.T) {
		deleteCount := 0
		now := time.Now()
		repo := &mockRepository{
			fetchUserFunc: func() (*model.User, error) {
				return &model.User{ID: "u1", NotesCount: 2}, nil
			},
			fetchNotesFunc: func(_ model.UserID, until model.NoteID, _ repository.FetchNotesOptions) ([]model.Note, error) {
				if until == "" {
					return []model.Note{
						{ID: "new", CreatedAt: now.Add(-24 * time.Hour)},
						{ID: "old", CreatedAt: now.Add(-72 * time.Hour)},
					}, nil
				}
				return []model.Note{}, nil
			},
			deleteNoteFunc: func(id model.NoteID) error {
				deleteCount++
				return nil
			},
		}
		logger := &mockLogger{}
		config := &model.AppConfig{DeleteInterval: 10, DeleteOlderThanDays: 2}
		uc := NewDeleteNotesUseCase(repo, config, logger)

		err := uc.Execute()
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if deleteCount != 1 {
			t.Fatalf("Expected 1 delete for old note, got %d", deleteCount)
		}

		foundLog := false
		for _, msg := range logger.infoMsgs {
			if msg == "Deleting only notes older than 2 days." {
				foundLog = true
				break
			}
		}
		if !foundLog {
			t.Error("Expected age filter log message")
		}
	})

	t.Run("Success_IncludesRepliesInScan", func(t *testing.T) {
		deleteCount := 0
		repliesRequested := false
		repo := &mockRepository{
			fetchUserFunc: func() (*model.User, error) {
				return &model.User{ID: "u1", NotesCount: 1}, nil
			},
			fetchNotesFunc: func(_ model.UserID, until model.NoteID, opts repository.FetchNotesOptions) ([]model.Note, error) {
				if opts.WithReplies {
					repliesRequested = true
				}
				if !opts.WithChannelNotes {
					t.Fatal("expected channel notes to be included in the scan")
				}
				if until == "" {
					return []model.Note{{ID: "reply-1"}}, nil
				}
				return []model.Note{}, nil
			},
			deleteNoteFunc: func(id model.NoteID) error {
				deleteCount++
				return nil
			},
		}
		logger := &mockLogger{}
		uc := NewDeleteNotesUseCase(repo, &model.AppConfig{DeleteInterval: 0}, logger)

		err := uc.Execute()
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if !repliesRequested {
			t.Fatal("Expected reply notes to be requested")
		}
		if deleteCount != 1 {
			t.Fatalf("Expected 1 delete for reply note, got %d", deleteCount)
		}
	})

	t.Run("Success_DriveDeletion_All", func(t *testing.T) {
		deleteCount := 0
		now := time.Now()
		repo := &mockRepository{
			fetchUserFunc: func() (*model.User, error) {
				return &model.User{ID: "u1", NotesCount: 0}, nil
			},
			fetchNotesFunc: func(_ model.UserID, _ model.NoteID, _ repository.FetchNotesOptions) ([]model.Note, error) {
				return []model.Note{}, nil
			},
			fetchDriveFilesFunc: func(_ *model.DriveFolderID, until model.DriveFileID) ([]model.DriveFile, error) {
				if until == "" {
					return []model.DriveFile{{ID: "f1", Name: "a.jpg", CreatedAt: now.Add(-48 * time.Hour)}}, nil
				}
				return []model.DriveFile{}, nil
			},
			deleteDriveFileFunc: func(id model.DriveFileID) error {
				deleteCount++
				return nil
			},
		}
		logger := &mockLogger{}
		config := &model.AppConfig{DeleteInterval: 0, DeleteDriveFiles: true}
		uc := NewDeleteNotesUseCase(repo, config, logger)

		err := uc.Execute()
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if deleteCount != 1 {
			t.Fatalf("Expected 1 drive delete, got %d", deleteCount)
		}
	})

	t.Run("Success_DriveDeletion_TraversesFolders", func(t *testing.T) {
		deleteCount := 0
		now := time.Now()
		folderID := model.DriveFolderID("folder-1")
		repo := &mockRepository{
			fetchUserFunc: func() (*model.User, error) {
				return &model.User{ID: "u1", NotesCount: 0}, nil
			},
			fetchNotesFunc: func(_ model.UserID, _ model.NoteID, _ repository.FetchNotesOptions) ([]model.Note, error) {
				return []model.Note{}, nil
			},
			fetchDriveFilesFunc: func(parent *model.DriveFolderID, until model.DriveFileID) ([]model.DriveFile, error) {
				if until != "" {
					return []model.DriveFile{}, nil
				}
				if parent == nil {
					return []model.DriveFile{{ID: "root", Name: "root.png", CreatedAt: now.Add(-48 * time.Hour)}}, nil
				}
				if *parent == folderID {
					return []model.DriveFile{{ID: "child", Name: "child.png", CreatedAt: now.Add(-48 * time.Hour)}}, nil
				}
				return []model.DriveFile{}, nil
			},
			fetchDriveFoldersFunc: func(parent *model.DriveFolderID, until model.DriveFolderID) ([]model.DriveFolder, error) {
				if until != "" {
					return []model.DriveFolder{}, nil
				}
				if parent == nil {
					return []model.DriveFolder{{ID: folderID}}, nil
				}
				return []model.DriveFolder{}, nil
			},
			deleteDriveFileFunc: func(id model.DriveFileID) error {
				deleteCount++
				return nil
			},
		}
		logger := &mockLogger{}
		config := &model.AppConfig{DeleteInterval: 0, DeleteDriveFiles: true}
		uc := NewDeleteNotesUseCase(repo, config, logger)

		err := uc.Execute()
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if deleteCount != 2 {
			t.Fatalf("Expected 2 drive deletes (root + folder), got %d", deleteCount)
		}
	})

	t.Run("Success_DriveDeletion_UnusedOnly", func(t *testing.T) {
		deleteCount := 0
		now := time.Now()
		repo := &mockRepository{
			fetchUserFunc: func() (*model.User, error) {
				return &model.User{ID: "u1", NotesCount: 0}, nil
			},
			fetchNotesFunc: func(_ model.UserID, _ model.NoteID, _ repository.FetchNotesOptions) ([]model.Note, error) {
				return []model.Note{}, nil
			},
			fetchDriveFilesFunc: func(_ *model.DriveFolderID, until model.DriveFileID) ([]model.DriveFile, error) {
				if until == "" {
					return []model.DriveFile{
						{ID: "f1", Name: "attached.png", CreatedAt: now.Add(-48 * time.Hour)},
						{ID: "f2", Name: "unused.png", CreatedAt: now.Add(-48 * time.Hour)},
					}, nil
				}
				return []model.DriveFile{}, nil
			},
			driveFileHasAttachedNotesFunc: func(id model.DriveFileID) (bool, error) {
				return id == "f1", nil
			},
			deleteDriveFileFunc: func(id model.DriveFileID) error {
				deleteCount++
				return nil
			},
		}
		logger := &mockLogger{}
		config := &model.AppConfig{DeleteInterval: 0, DeleteDriveFiles: true, DeleteDriveUnusedOnly: true}
		uc := NewDeleteNotesUseCase(repo, config, logger)

		err := uc.Execute()
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if deleteCount != 1 {
			t.Fatalf("Expected 1 drive delete for unused file, got %d", deleteCount)
		}
	})

	t.Run("Success_DriveOnlyMode", func(t *testing.T) {
		deleteCount := 0
		now := time.Now()
		repo := &mockRepository{
			fetchUserFunc: func() (*model.User, error) {
				return &model.User{ID: "u1"}, nil
			},
			fetchNotesFunc: func(_ model.UserID, _ model.NoteID, _ repository.FetchNotesOptions) ([]model.Note, error) {
				t.Fatal("FetchNotes should not be called in drive-only mode")
				return nil, nil
			},
			fetchDriveFilesFunc: func(_ *model.DriveFolderID, until model.DriveFileID) ([]model.DriveFile, error) {
				if until == "" {
					return []model.DriveFile{{ID: "f1", Name: "drive-only.png", CreatedAt: now.Add(-48 * time.Hour)}}, nil
				}
				return []model.DriveFile{}, nil
			},
			deleteDriveFileFunc: func(id model.DriveFileID) error {
				deleteCount++
				return nil
			},
		}
		logger := &mockLogger{}
		config := &model.AppConfig{DeleteInterval: 0, DeleteDriveOnly: true}
		uc := NewDeleteNotesUseCase(repo, config, logger)

		err := uc.Execute()
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if deleteCount != 1 {
			t.Fatalf("Expected 1 drive delete in drive-only mode, got %d", deleteCount)
		}
	})

	t.Run("Success_DriveDeletion_SkipsProfileFiles", func(t *testing.T) {
		deleteCount := 0
		now := time.Now()
		avatarID := model.DriveFileID("avatar")
		bannerID := model.DriveFileID("banner")
		repo := &mockRepository{
			fetchUserFunc: func() (*model.User, error) {
				return &model.User{ID: "u1", AvatarID: &avatarID, BannerID: &bannerID}, nil
			},
			fetchNotesFunc: func(_ model.UserID, _ model.NoteID, _ repository.FetchNotesOptions) ([]model.Note, error) {
				t.Fatal("FetchNotes should not be called in drive-only mode")
				return nil, nil
			},
			fetchDriveFilesFunc: func(_ *model.DriveFolderID, until model.DriveFileID) ([]model.DriveFile, error) {
				if until == "" {
					return []model.DriveFile{
						{ID: avatarID, Name: "avatar.png", CreatedAt: now.Add(-48 * time.Hour)},
						{ID: bannerID, Name: "banner.png", CreatedAt: now.Add(-48 * time.Hour)},
						{ID: "f3", Name: "other.png", CreatedAt: now.Add(-48 * time.Hour)},
					}, nil
				}
				return []model.DriveFile{}, nil
			},
			deleteDriveFileFunc: func(id model.DriveFileID) error {
				deleteCount++
				return nil
			},
		}
		logger := &mockLogger{}
		config := &model.AppConfig{DeleteInterval: 0, DeleteDriveOnly: true}
		uc := NewDeleteNotesUseCase(repo, config, logger)

		err := uc.Execute()
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if deleteCount != 1 {
			t.Fatalf("Expected 1 drive delete after skipping profile files, got %d", deleteCount)
		}
	})

	t.Run("Success_DriveDeletion_SkipsMissingFile", func(t *testing.T) {
		deleteCount := 0
		now := time.Now()
		repo := &mockRepository{
			fetchUserFunc: func() (*model.User, error) {
				return &model.User{ID: "u1"}, nil
			},
			fetchNotesFunc: func(_ model.UserID, _ model.NoteID, _ repository.FetchNotesOptions) ([]model.Note, error) {
				return []model.Note{}, nil
			},
			fetchDriveFilesFunc: func(_ *model.DriveFolderID, until model.DriveFileID) ([]model.DriveFile, error) {
				if until == "" {
					return []model.DriveFile{{ID: "missing", Name: "missing.png", CreatedAt: now.Add(-48 * time.Hour)}}, nil
				}
				return []model.DriveFile{}, nil
			},
			deleteDriveFileFunc: func(id model.DriveFileID) error {
				deleteCount++
				return errors.New("HTTP 400 returned from drive/files/delete: {\"error\":{\"code\":\"NO_SUCH_FILE\"}}")
			},
		}
		logger := &mockLogger{}
		config := &model.AppConfig{DeleteInterval: 0, DeleteDriveFiles: true}
		uc := NewDeleteNotesUseCase(repo, config, logger)

		err := uc.Execute()
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if deleteCount != 1 {
			t.Fatalf("Expected 1 delete attempt for missing file, got %d", deleteCount)
		}
		if len(logger.warnMsgs) == 0 {
			t.Fatal("Expected a warning log for missing drive file")
		}
	})

	t.Run("Success_DriveDeletion_SkipProfileBeforeAttachmentCheck", func(t *testing.T) {
		deleteCount := 0
		attachmentChecks := 0
		now := time.Now()
		avatarID := model.DriveFileID("avatar")
		repo := &mockRepository{
			fetchUserFunc: func() (*model.User, error) {
				return &model.User{ID: "u1", AvatarID: &avatarID}, nil
			},
			fetchNotesFunc: func(_ model.UserID, _ model.NoteID, _ repository.FetchNotesOptions) ([]model.Note, error) {
				return []model.Note{}, nil
			},
			fetchDriveFilesFunc: func(_ *model.DriveFolderID, until model.DriveFileID) ([]model.DriveFile, error) {
				if until == "" {
					return []model.DriveFile{
						{ID: avatarID, Name: "avatar.png", CreatedAt: now.Add(-48 * time.Hour)},
						{ID: "f1", Name: "attached.png", CreatedAt: now.Add(-48 * time.Hour)},
						{ID: "f2", Name: "free.png", CreatedAt: now.Add(-48 * time.Hour)},
					}, nil
				}
				return []model.DriveFile{}, nil
			},
			driveFileHasAttachedNotesFunc: func(id model.DriveFileID) (bool, error) {
				attachmentChecks++
				return id == "f1", nil
			},
			deleteDriveFileFunc: func(id model.DriveFileID) error {
				deleteCount++
				return nil
			},
		}
		logger := &mockLogger{}
		config := &model.AppConfig{DeleteInterval: 0, DeleteDriveFiles: true, DeleteDriveUnusedOnly: true}
		uc := NewDeleteNotesUseCase(repo, config, logger)

		err := uc.Execute()
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if deleteCount != 1 {
			t.Fatalf("Expected 1 drive delete after skipping profile/attached files, got %d", deleteCount)
		}
		if attachmentChecks != 2 {
			t.Fatalf("Expected 2 attachment checks (non-profile files only), got %d", attachmentChecks)
		}
	})
}
