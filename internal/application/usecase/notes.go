package usecase

import (
	"fmt"
	"misskeyNotedel/internal/domain/model"
	"misskeyNotedel/internal/domain/repository"
	"time"
)

func (i *deleteNotesInteractor) scanNotes(user *model.User) ([]model.Note, error) {
	var targets []model.Note
	var untilID model.NoteID
	cutoff := i.computeCutoff(i.config.NoteOlderThan)

	for {
		batch, err := i.repo.FetchNotes(user.ID, untilID, repository.FetchNotesOptions{
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
			if !cutoff.IsZero() && note.CreatedAt.After(cutoff) {
				continue
			}
			if !note.ShouldKeep(i.config) {
				targets = append(targets, note)
			}
		}

		untilID = batch[len(batch)-1].ID
		if !i.config.Quiet {
			i.log.Info(fmt.Sprintf("Scanning notes... Found %d targets so far", len(targets)))
		}
	}

	return targets, nil
}

func (i *deleteNotesInteractor) executeNoteDeletions(user *model.User, targets []model.Note) {
	i.unpinDeletableNotes(user, targets)

	total := len(targets)
	i.log.Info(fmt.Sprintf("Starting note deletion: %d targets", total))

	deleted := 0
	for idx, note := range targets {
		if i.config.MaxDelete > 0 && deleted >= i.config.MaxDelete {
			i.log.Warn(fmt.Sprintf("Reached max-delete limit (%d). Stopping note deletion.", i.config.MaxDelete))
			break
		}

		num := idx + 1
		if err := i.repo.DeleteNote(note.ID); err != nil {
			if isNotFoundError(err) {
				i.log.Warn(fmt.Sprintf("[%d/%d] Note already deleted: %s", num, total, note.ID))
				i.sleepBetweenDeletions()
				continue
			}
			if isAuthError(err) {
				i.log.Error(fmt.Sprintf("[%d/%d] Auth error, stopping: %s", num, total, note.ID), err)
				return
			}
			if isRateLimitError(err) {
				i.log.Warn(fmt.Sprintf("[%d/%d] Rate limited. Backing off for 60s before retry.", num, total))
				time.Sleep(60 * time.Second)
			}
			if note.IsRenote() && isNonPublicRenoteError(err) {
				i.log.Warn(fmt.Sprintf("[%d/%d] Skipped renote %s (non-public origin)", num, total, note.ID))
				i.sleepBetweenDeletions()
				continue
			}
			i.log.Error(fmt.Sprintf("[%d/%d] Failed to delete note %s (%s)", num, total, note.ID, note.KindLabel()), err)
			i.sleepOnError()
			continue
		}

		deleted++
		i.log.Info(fmt.Sprintf("[%d/%d] Deleted: %s", num, total, note.GetSummary()))
		i.sleepBetweenDeletions()
	}

	i.log.Info(fmt.Sprintf("Note deletion done. Deleted: %d / Scanned: %d", deleted, total))
}

func (i *deleteNotesInteractor) unpinDeletableNotes(user *model.User, targets []model.Note) {
	if len(user.PinnedNotes) == 0 {
		return
	}

	targetIDs := make(map[model.NoteID]bool, len(targets))
	for _, n := range targets {
		targetIDs[n.ID] = true
	}

	for _, pinned := range user.PinnedNotes {
		if !targetIDs[pinned.ID] {
			continue
		}
		if err := i.repo.UnpinNote(pinned.ID); err != nil {
			i.log.Error(fmt.Sprintf("Failed to unpin note %s", pinned.ID), err)
		} else {
			i.log.Info(fmt.Sprintf("Unpinned note: %s", pinned.ID))
		}
	}
}

func (i *deleteNotesInteractor) computeCutoff(olderThan time.Duration) time.Time {
	if olderThan <= 0 {
		return time.Time{}
	}
	return time.Now().Add(-olderThan)
}
