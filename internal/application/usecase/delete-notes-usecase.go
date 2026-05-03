package usecase

import (
	"fmt"
	"misskeyNotedel/internal/domain/model"
	"misskeyNotedel/internal/domain/repository"
	"misskeyNotedel/internal/shared/logger"
	"strings"
	"time"
)

type DeleteNotesUseCase interface {
	Execute() error
}

type deleteNotesInteractor struct {
	repository repository.MisskeyRepository
	config     *model.AppConfig
	logger     logger.Logger
}

func NewDeleteNotesUseCase(repo repository.MisskeyRepository, config *model.AppConfig, log logger.Logger) DeleteNotesUseCase {
	return &deleteNotesInteractor{
		repository: repo,
		config:     config,
		logger:     log,
	}
}

func (i *deleteNotesInteractor) Execute() error {
	if !i.config.IsSafeInterval() {
		i.logger.Warn("Delete interval is set to less than 10 seconds. This may cause rate limiting.")
	}
	if i.config.DeleteOlderThanDays > 0 {
		i.logger.Info(fmt.Sprintf("Deleting only notes older than %d days.", i.config.DeleteOlderThanDays))
	}

	user, err := i.repository.FetchUser()
	if err != nil {
		return err
	}

	i.logger.Info(fmt.Sprintf("Target User: %s @%s (%d Total Notes)", user.Name, user.Username, user.NotesCount))

	for _, note := range user.PinnedNotes {
		if err := i.repository.UnpinNote(note.ID); err != nil {
			i.logger.Error(fmt.Sprintf("Failed to unpin note %s", note.ID), err)
		} else {
			i.logger.Info(fmt.Sprintf("Unpinned note: %s", note.ID))
		}
	}

	i.logger.Info("Scanning for deletion targets...")
	targets, err := i.scanDeletionTargets(user.ID)
	if err != nil {
		return err
	}

	targetCount := len(targets)
	if targetCount == 0 {
		i.logger.Info("No deletion targets found.")
		return nil
	}

	i.logger.Info(fmt.Sprintf("Found %d deletion targets. Starting deletion process...", targetCount))

	for index, note := range targets {
		currentNumber := index + 1
		if err := i.repository.DeleteNote(note.ID); err != nil {
			if note.IsRenote() && isNonPublicRenoteDeleteError(err) {
				i.logger.Warn(fmt.Sprintf("[%d/%d] Skipped renote %s due to Misskey internal error for non-public origin", currentNumber, targetCount, note.ID))
				time.Sleep(time.Duration(i.config.DeleteInterval) * time.Second)
				continue
			}

			i.logger.Error(fmt.Sprintf("[%d/%d] Error deleting note %s (kind=%s)", currentNumber, targetCount, note.ID, note.KindLabel()), err)
			time.Sleep(15 * time.Minute)
			continue
		}

		i.logger.Info(fmt.Sprintf("[%d/%d] Deleted: %s", currentNumber, targetCount, note.GetSummary()))
		time.Sleep(time.Duration(i.config.DeleteInterval) * time.Second)
	}

	i.logger.Info(fmt.Sprintf("Process completed. Total deleted: %d", targetCount))
	return nil
}

func isNonPublicRenoteDeleteError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "renderAnnounce: cannot render non-public note")
}

func (i *deleteNotesInteractor) scanDeletionTargets(userID model.UserID) ([]model.Note, error) {
	var targets []model.Note
	var untilID model.NoteID
	var cutoff time.Time
	if i.config.DeleteOlderThanDays > 0 {
		cutoff = time.Now().Add(-time.Duration(i.config.DeleteOlderThanDays) * 24 * time.Hour)
	}

	for {
		batch, err := i.repository.FetchNotes(userID, untilID, repository.FetchNotesOptions{
			WithReplies:      true,
			WithChannelNotes: true,
		})
		if err != nil {
			return nil, err
		}
		if len(batch) == 0 {
			break
		}

		for _, note := range batch {
			if i.config.DeleteOlderThanDays > 0 && note.CreatedAt.After(cutoff) {
				continue
			}
			if !note.ShouldKeep(i.config) {
				targets = append(targets, note)
			}
		}

		untilID = batch[len(batch)-1].ID
		i.logger.Info(fmt.Sprintf("Scanning... Found %d targets so far", len(targets)))
	}

	return targets, nil
}
