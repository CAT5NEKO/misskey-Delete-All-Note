package usecase

import (
	"fmt"
	"misskeyNotedel/internal/domain/model"
	"time"
)

const maxFolderDepth = 50

func (i *deleteNotesInteractor) scanDrive(user *model.User) ([]model.DriveFile, error) {
	var targets []model.DriveFile
	cutoff := i.computeCutoff(i.config.DriveOlderThan)

	if err := i.collectDriveFiles(nil, 0, cutoff, &targets); err != nil {
		return nil, err
	}

	if len(user.PinnedNotes) > 0 {
		i.log.Warn(fmt.Sprintf("Pinned notes detected. Be aware that unpinning/note deletion does not touch drive files attached to those notes."))
	}

	return targets, nil
}

func (i *deleteNotesInteractor) collectDriveFiles(folderID *model.DriveFolderID, depth int, cutoff time.Time, targets *[]model.DriveFile) error {
	if depth > maxFolderDepth {
		i.log.Warn("Drive folder depth limit reached. Skipping deeper folders.")
		return nil
	}

	var fileUntilID model.DriveFileID
	for {
		batch, err := i.repo.FetchDriveFiles(folderID, fileUntilID)
		if err != nil {
			return err
		}
		if len(batch) == 0 {
			break
		}

		for _, file := range batch {
			if !cutoff.IsZero() && file.CreatedAt.After(cutoff) {
				continue
			}
			*targets = append(*targets, file)
		}

		fileUntilID = batch[len(batch)-1].ID
		if !i.config.Quiet {
			i.log.Info(fmt.Sprintf("Scanning drive... Found %d targets so far", len(*targets)))
		}
	}

	var folderUntilID model.DriveFolderID
	for {
		folders, err := i.repo.FetchDriveFolders(folderID, folderUntilID)
		if err != nil {
			return err
		}
		if len(folders) == 0 {
			break
		}
		for _, folder := range folders {
			folderIDCopy := folder.ID
			if err := i.collectDriveFiles(&folderIDCopy, depth+1, cutoff, targets); err != nil {
				return err
			}
		}
		folderUntilID = folders[len(folders)-1].ID
	}

	return nil
}

func (i *deleteNotesInteractor) executeDriveDeletions(user *model.User, targets []model.DriveFile) {
	protected := i.protectedDriveFileIDs(user)

	total := len(targets)
	i.logInfo(fmt.Sprintf("Starting drive deletion: %d targets", total))

	deleted := 0
	for idx, file := range targets {
		if i.config.MaxDelete > 0 && deleted >= i.config.MaxDelete {
			i.log.Warn(fmt.Sprintf("Reached max-delete limit (%d). Stopping drive deletion.", i.config.MaxDelete))
			break
		}

		num := idx + 1

		if _, ok := protected[file.ID]; ok {
			i.logInfo(fmt.Sprintf("[%d/%d] Skipped profile file: %s", num, total, file.ID))
			i.sleepBetweenDeletions()
			continue
		}

		if i.config.DriveMode == "unused" {
			attached, err := i.repo.DriveFileHasAttachedNotes(file.ID)
			if err != nil {
				if isAuthError(err) {
					i.log.Error(fmt.Sprintf("[%d/%d] Auth error checking file %s, stopping", num, total, file.ID), err)
					return
				}
				i.log.Error(fmt.Sprintf("[%d/%d] Error checking attachments for drive file %s", num, total, file.ID), err)
				i.sleepOnError()
				continue
			}
			if attached {
				if i.config.Verbose {
					i.log.Info(fmt.Sprintf("[%d/%d] Skipped attached drive file: %s", num, total, file.ID))
				}
				i.sleepBetweenDeletions()
				continue
			}
		}

		if err := i.repo.DeleteDriveFile(file.ID); err != nil {
			if isNotFoundError(err) {
				i.log.Warn(fmt.Sprintf("[%d/%d] Drive file already deleted: %s", num, total, file.ID))
				i.sleepBetweenDeletions()
				continue
			}
			if isAuthError(err) {
				i.log.Error(fmt.Sprintf("[%d/%d] Auth error deleting file %s, stopping", num, total, file.ID), err)
				return
			}
			if isRateLimitError(err) {
				i.log.Warn(fmt.Sprintf("[%d/%d] Rate limited. Backing off for 60s before retry.", num, total))
				time.Sleep(60 * time.Second)
				continue
			}
			i.log.Error(fmt.Sprintf("[%d/%d] Error deleting drive file %s (%s)", num, total, file.ID, file.Name), err)
			i.sleepOnError()
			continue
		}

		deleted++
		i.logInfo(fmt.Sprintf("[%d/%d] Deleted drive file: %s", num, total, file.Name))
		i.sleepBetweenDeletions()
	}

	i.logInfo(fmt.Sprintf("Drive deletion done. Deleted: %d / Scanned: %d", deleted, total))
}

func (i *deleteNotesInteractor) protectedDriveFileIDs(user *model.User) map[model.DriveFileID]struct{} {
	protected := make(map[model.DriveFileID]struct{})
	if user == nil {
		return protected
	}
	if user.AvatarID != nil {
		protected[*user.AvatarID] = struct{}{}
	}
	if user.BannerID != nil {
		protected[*user.BannerID] = struct{}{}
	}
	return protected
}
