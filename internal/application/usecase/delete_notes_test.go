package usecase

import (
	"errors"
	"misskeyNotedel/internal/domain/model"
	"testing"
	"time"
)

type mockRepository struct {
	fetchUserFunc  func() (*model.User, error)
	fetchNotesFunc func(userID model.UserID, untilID model.NoteID) ([]model.Note, error)
	deleteNoteFunc func(noteID model.NoteID) error
	unpinNoteFunc  func(noteID model.NoteID) error
}

func (m *mockRepository) FetchUser() (*model.User, error) { return m.fetchUserFunc() }
func (m *mockRepository) FetchNotes(u model.UserID, i model.NoteID) ([]model.Note, error) {
	return m.fetchNotesFunc(u, i)
}
func (m *mockRepository) DeleteNote(i model.NoteID) error { return m.deleteNoteFunc(i) }
func (m *mockRepository) UnpinNote(i model.NoteID) error  { return m.unpinNoteFunc(i) }

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
			fetchNotesFunc: func(_ model.UserID, _ model.NoteID) ([]model.Note, error) {
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
			fetchNotesFunc: func(_ model.UserID, until model.NoteID) ([]model.Note, error) {
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
			fetchNotesFunc: func(_ model.UserID, until model.NoteID) ([]model.Note, error) {
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
			fetchNotesFunc: func(_ model.UserID, until model.NoteID) ([]model.Note, error) {
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
}
