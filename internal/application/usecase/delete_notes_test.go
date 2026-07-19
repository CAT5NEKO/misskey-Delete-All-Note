package usecase

import (
	"errors"
	"misskeyNotedel/internal/domain/model"
	"misskeyNotedel/internal/domain/repository"
	"os"
	"testing"
	"time"
)

var testConfig = func(overrides ...func(*model.AppConfig)) *model.AppConfig {
	lockFile, _ := os.CreateTemp("", "misskeyNotedel-test-lock-*")
	name := lockFile.Name()
	lockFile.Close()
	os.Remove(name)

	cfg := &model.AppConfig{
		DeleteInterval: 0,
		LockFile:       name,
		Force:          true,
		Yes:            true,
		DriveMode:      "none",
	}
	for _, o := range overrides {
		o(cfg)
	}
	return cfg
}

func withDriveMode(mode string) func(*model.AppConfig) {
	return func(c *model.AppConfig) { c.DriveMode = mode }
}

func withSkipNotes() func(*model.AppConfig) {
	return func(c *model.AppConfig) { c.SkipNotes = true }
}

func withKeepRenotes() func(*model.AppConfig) {
	return func(c *model.AppConfig) { c.KeepWithRenotes = true }
}

func withDryRun() func(*model.AppConfig) {
	return func(c *model.AppConfig) { c.DryRun = true }
}

func withMaxDelete(n int) func(*model.AppConfig) {
	return func(c *model.AppConfig) { c.MaxDelete = n }
}

func withNoteOlderThan(d time.Duration) func(*model.AppConfig) {
	return func(c *model.AppConfig) { c.NoteOlderThan = d }
}

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
	return m.fetchDriveFilesFunc(folderID, i)
}
func (m *mockRepository) FetchDriveFolders(parentID *model.DriveFolderID, i model.DriveFolderID) ([]model.DriveFolder, error) {
	if m.fetchDriveFoldersFunc == nil {
		return []model.DriveFolder{}, nil
	}
	return m.fetchDriveFoldersFunc(parentID, i)
}
func (m *mockRepository) DeleteDriveFile(i model.DriveFileID) error { return m.deleteDriveFileFunc(i) }
func (m *mockRepository) DriveFileHasAttachedNotes(i model.DriveFileID) (bool, error) {
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

func containsMsg(msgs []string, substr string) bool {
	for _, m := range msgs {
		if len(m) >= len(substr) {
			for i := 0; i <= len(m)-len(substr); i++ {
				if m[i:i+len(substr)] == substr {
					return true
				}
			}
		}
	}
	return false
}

func TestExecute_NoteDeletion_NoTargets(t *testing.T) {
	repo := &mockRepository{
		fetchUserFunc: func() (*model.User, error) {
			return &model.User{ID: "u1", NotesCount: 0}, nil
		},
		fetchNotesFunc: func(_ model.UserID, _ model.NoteID, _ repository.FetchNotesOptions) ([]model.Note, error) {
			return []model.Note{}, nil
		},
	}
	logger := &mockLogger{}
	uc := NewDeleteNotesUseCase(repo, testConfig(), logger)

	if err := uc.Execute(); err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if !containsMsg(logger.infoMsgs, "No deletion targets found") {
		t.Error("should have logged 'No deletion targets found'")
	}
}

func TestExecute_NoteDeletion_WithDeletions(t *testing.T) {
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
		fetchNotesFunc: func(_ model.UserID, until model.NoteID, _ repository.FetchNotesOptions) ([]model.Note, error) {
			if until == "" {
				return []model.Note{
					{ID: "n1", RenoteCount: 0},
					{ID: "p1", RenoteCount: 10},
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
	uc := NewDeleteNotesUseCase(repo, testConfig(withKeepRenotes()), logger)

	if err := uc.Execute(); err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if unpinCount != 0 {
		t.Errorf("Expected 0 unpins (p1 is pinned but not a deletion target), got %d", unpinCount)
	}
	if deleteCount != 1 {
		t.Errorf("Expected 1 delete (n1 only, p1 is kept), got %d", deleteCount)
	}
}

func TestExecute_UnpinOnlyDeletableNotes(t *testing.T) {
	unpinCount := 0
	repo := &mockRepository{
		fetchUserFunc: func() (*model.User, error) {
			return &model.User{
				ID:          "u1",
				NotesCount:  1,
				PinnedNotes: []model.Note{{ID: "keep"}},
			}, nil
		},
		fetchNotesFunc: func(_ model.UserID, until model.NoteID, _ repository.FetchNotesOptions) ([]model.Note, error) {
			if until == "" {
				return []model.Note{
					{ID: "keep", Reactions: map[string]int{"like": 5}},
				}, nil
			}
			return []model.Note{}, nil
		},
		unpinNoteFunc: func(id model.NoteID) error {
			unpinCount++
			return nil
		},
		deleteNoteFunc: func(id model.NoteID) error {
			return nil
		},
	}
	logger := &mockLogger{}
	cfg := testConfig(func(c *model.AppConfig) {
		c.KeepWithReactions = true
	})
	uc := NewDeleteNotesUseCase(repo, cfg, logger)

	if err := uc.Execute(); err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if unpinCount != 0 {
		t.Errorf("Expected 0 unpins (kept note is pinned but should not be unpinned), got %d", unpinCount)
	}
}

func TestExecute_FetchUserAuthError(t *testing.T) {
	repo := &mockRepository{
		fetchUserFunc: func() (*model.User, error) {
			return nil, errors.New("API error [AUTHENTICATION_FAILED] (HTTP 401): invalid token")
		},
	}
	uc := NewDeleteNotesUseCase(repo, testConfig(), &mockLogger{})

	err := uc.Execute()
	if err == nil {
		t.Fatal("Expected error for auth failure")
	}
	if !containsMsg([]string{err.Error()}, "authentication failed") {
		t.Errorf("expected auth error message, got: %v", err)
	}
}

func TestExecute_FetchUserError(t *testing.T) {
	repo := &mockRepository{
		fetchUserFunc: func() (*model.User, error) {
			return nil, errors.New("network error")
		},
	}
	uc := NewDeleteNotesUseCase(repo, testConfig(), &mockLogger{})

	err := uc.Execute()
	if err == nil {
		t.Fatal("Expected error, got nil")
	}
}

func TestExecute_NonPublicRenoteDeleteError(t *testing.T) {
	renoteID := model.NoteID("src1")
	deleteAttempts := 0
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
			deleteAttempts++
			return errors.New("API error [] (HTTP 500): renderAnnounce: cannot render non-public note")
		},
	}
	logger := &mockLogger{}
	uc := NewDeleteNotesUseCase(repo, testConfig(), logger)

	if err := uc.Execute(); err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if deleteAttempts != 1 {
		t.Errorf("Expected 1 delete attempt, got %d", deleteAttempts)
	}
	if !containsMsg(logger.warnMsgs, "Skipped renote") {
		t.Error("Expected a warning log for skipped renote")
	}
}

func TestExecute_NoteAlreadyDeleted(t *testing.T) {
	deleteAttempts := 0
	repo := &mockRepository{
		fetchUserFunc: func() (*model.User, error) {
			return &model.User{ID: "u1", NotesCount: 1}, nil
		},
		fetchNotesFunc: func(_ model.UserID, until model.NoteID, _ repository.FetchNotesOptions) ([]model.Note, error) {
			if until == "" {
				return []model.Note{{ID: "n1"}}, nil
			}
			return []model.Note{}, nil
		},
		deleteNoteFunc: func(id model.NoteID) error {
			deleteAttempts++
			return errors.New("API error [NO_SUCH_NOTE] (HTTP 400): no such note")
		},
	}
	logger := &mockLogger{}
	uc := NewDeleteNotesUseCase(repo, testConfig(), logger)

	if err := uc.Execute(); err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if deleteAttempts != 1 {
		t.Errorf("Expected 1 delete attempt, got %d", deleteAttempts)
	}
	if !containsMsg(logger.warnMsgs, "already deleted") {
		t.Error("Expected warning for already deleted note")
	}
}

func TestExecute_NoteAgeFilter(t *testing.T) {
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
	uc := NewDeleteNotesUseCase(repo, testConfig(withNoteOlderThan(48*time.Hour)), logger)

	if err := uc.Execute(); err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if deleteCount != 1 {
		t.Errorf("Expected 1 delete (old note), got %d", deleteCount)
	}
}

func TestExecute_IncludesRepliesInScan(t *testing.T) {
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
	uc := NewDeleteNotesUseCase(repo, testConfig(), &mockLogger{})

	if err := uc.Execute(); err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if !repliesRequested {
		t.Fatal("Expected reply notes to be requested")
	}
	if deleteCount != 1 {
		t.Fatalf("Expected 1 delete for reply note, got %d", deleteCount)
	}
}

func TestExecute_DryRun(t *testing.T) {
	deleteCalled := false
	repo := &mockRepository{
		fetchUserFunc: func() (*model.User, error) {
			return &model.User{ID: "u1", NotesCount: 1}, nil
		},
		fetchNotesFunc: func(_ model.UserID, until model.NoteID, _ repository.FetchNotesOptions) ([]model.Note, error) {
			if until == "" {
				return []model.Note{{ID: "n1"}}, nil
			}
			return []model.Note{}, nil
		},
		deleteNoteFunc: func(id model.NoteID) error {
			deleteCalled = true
			return nil
		},
	}
	logger := &mockLogger{}
	uc := NewDeleteNotesUseCase(repo, testConfig(withDryRun()), logger)

	if err := uc.Execute(); err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if deleteCalled {
		t.Error("Delete should NOT have been called in dry-run mode")
	}
	if !containsMsg(logger.infoMsgs, "DRY RUN") {
		t.Error("Expected DRY RUN log message")
	}
}

func TestExecute_MaxDelete_Notes(t *testing.T) {
	deleteCount := 0
	repo := &mockRepository{
		fetchUserFunc: func() (*model.User, error) {
			return &model.User{ID: "u1", NotesCount: 5}, nil
		},
		fetchNotesFunc: func(_ model.UserID, until model.NoteID, _ repository.FetchNotesOptions) ([]model.Note, error) {
			if until == "" {
				return []model.Note{
					{ID: "n1"}, {ID: "n2"}, {ID: "n3"}, {ID: "n4"}, {ID: "n5"},
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
	uc := NewDeleteNotesUseCase(repo, testConfig(withMaxDelete(2)), logger)

	if err := uc.Execute(); err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if deleteCount != 2 {
		t.Errorf("Expected 2 deletes (max limit), got %d", deleteCount)
	}
	if !containsMsg(logger.warnMsgs, "max-delete limit") {
		t.Error("Expected max-delete warning")
	}
}

func TestExecute_DriveDeletion_All(t *testing.T) {
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
	uc := NewDeleteNotesUseCase(repo, testConfig(withDriveMode("all")), logger)

	if err := uc.Execute(); err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if deleteCount != 1 {
		t.Errorf("Expected 1 drive delete, got %d", deleteCount)
	}
}

func TestExecute_DriveDeletion_TraversesFolders(t *testing.T) {
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
	uc := NewDeleteNotesUseCase(repo, testConfig(withDriveMode("all")), &mockLogger{})

	if err := uc.Execute(); err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if deleteCount != 2 {
		t.Errorf("Expected 2 drive deletes (root + folder), got %d", deleteCount)
	}
}

func TestExecute_DriveDeletion_UnusedOnly(t *testing.T) {
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
	uc := NewDeleteNotesUseCase(repo, testConfig(withDriveMode("unused")), &mockLogger{})

	if err := uc.Execute(); err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if deleteCount != 1 {
		t.Errorf("Expected 1 drive delete for unused file, got %d", deleteCount)
	}
}

func TestExecute_SkipNotesWithDrive(t *testing.T) {
	deleteCount := 0
	notesFetched := false
	now := time.Now()
	repo := &mockRepository{
		fetchUserFunc: func() (*model.User, error) {
			return &model.User{ID: "u1"}, nil
		},
		fetchNotesFunc: func(_ model.UserID, _ model.NoteID, _ repository.FetchNotesOptions) ([]model.Note, error) {
			notesFetched = true
			return []model.Note{}, nil
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
	cfg := testConfig(withSkipNotes(), withDriveMode("all"))
	uc := NewDeleteNotesUseCase(repo, cfg, logger)

	if err := uc.Execute(); err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if notesFetched {
		t.Error("FetchNotes should NOT have been called in skip-notes mode")
	}
	if deleteCount != 1 {
		t.Errorf("Expected 1 drive delete in skip-notes mode, got %d", deleteCount)
	}
}

func TestExecute_DriveDeletion_SkipsProfileFiles(t *testing.T) {
	deleteCount := 0
	now := time.Now()
	avatarID := model.DriveFileID("avatar")
	bannerID := model.DriveFileID("banner")
	repo := &mockRepository{
		fetchUserFunc: func() (*model.User, error) {
			return &model.User{ID: "u1", AvatarID: &avatarID, BannerID: &bannerID}, nil
		},
		fetchNotesFunc: func(_ model.UserID, _ model.NoteID, _ repository.FetchNotesOptions) ([]model.Note, error) {
			return []model.Note{}, nil
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
	cfg := testConfig(withDriveMode("all"), withSkipNotes())
	uc := NewDeleteNotesUseCase(repo, cfg, &mockLogger{})

	if err := uc.Execute(); err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if deleteCount != 1 {
		t.Errorf("Expected 1 drive delete after skipping profile files, got %d", deleteCount)
	}
}

func TestExecute_DriveDeletion_SkipsMissingFile(t *testing.T) {
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
			return errors.New("API error [NO_SUCH_FILE] (HTTP 400): no such file")
		},
	}
	logger := &mockLogger{}
	uc := NewDeleteNotesUseCase(repo, testConfig(withDriveMode("all")), logger)

	if err := uc.Execute(); err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if deleteCount != 1 {
		t.Errorf("Expected 1 delete attempt for missing file, got %d", deleteCount)
	}
	if !containsMsg(logger.warnMsgs, "already deleted") {
		t.Error("Expected warning for already deleted drive file")
	}
}

func TestExecute_DriveDeletion_SkipProfileBeforeAttachmentCheck(t *testing.T) {
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
			return nil
		},
	}
	cfg := testConfig(withDriveMode("unused"))
	uc := NewDeleteNotesUseCase(repo, cfg, &mockLogger{})

	if err := uc.Execute(); err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if attachmentChecks != 2 {
		t.Errorf("Expected 2 attachment checks (non-profile files only), got %d", attachmentChecks)
	}
}
