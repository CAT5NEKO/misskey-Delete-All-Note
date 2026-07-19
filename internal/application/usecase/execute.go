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
	repo   repository.MisskeyRepository
	config *model.AppConfig
	log    logger.Logger
}

func NewDeleteNotesUseCase(repo repository.MisskeyRepository, config *model.AppConfig, log logger.Logger) DeleteNotesUseCase {
	return &deleteNotesInteractor{repo: repo, config: config, log: log}
}

func (i *deleteNotesInteractor) Execute() error {
	cleanup, err := acquireLock(i.config.LockFile, i.config.Force)
	if err != nil {
		return fmt.Errorf("lock: %w", err)
	}
	defer cleanup()

	if !i.config.IsSafeInterval() {
		i.log.Warn("Delete interval is less than 10 seconds. This may cause rate limiting.")
	}

	user, err := i.repo.FetchUser()
	if err != nil {
		if isAuthError(err) {
			return fmt.Errorf("authentication failed: %w", err)
		}
		return fmt.Errorf("failed to fetch user: %w", err)
	}

	i.logInfo(fmt.Sprintf("Target User: %s @%s (%d Total Notes)", user.Name, user.Username, user.NotesCount))

	i.logAgeFilters()

	skipNotes := i.config.SkipNotes
	doDrive := i.config.DriveMode != "" && i.config.DriveMode != "none"

	if skipNotes && !doDrive {
		i.log.Warn("Nothing to do: SkipNotes=true and DriveMode=none.")
		return nil
	}

	var noteTargets []model.Note
	var driveTargets []model.DriveFile

	if !skipNotes {
		noteTargets, err = i.scanNotes(user)
		if err != nil {
			if isAuthError(err) {
				return fmt.Errorf("authentication failed during note scan: %w", err)
			}
			i.log.Error("Failed to scan notes", err)
		}
	} else {
		i.logInfo("SkipNotes enabled: skipping note deletion")
	}

	if doDrive {
		driveTargets, err = i.scanDrive(user)
		if err != nil {
			if isAuthError(err) {
				return fmt.Errorf("authentication failed during drive scan: %w", err)
			}
			i.log.Error("Failed to scan drive files", err)
		}
	}

	if i.config.DryRun {
		i.printDryRunSummary(noteTargets, driveTargets)
		return nil
	}

	if len(noteTargets) == 0 && len(driveTargets) == 0 {
		i.logInfo("No deletion targets found.")
		return nil
	}

	if !i.config.Yes && !i.confirm(len(noteTargets), len(driveTargets)) {
		i.logInfo("Cancelled.")
		return nil
	}

	if len(noteTargets) > 0 {
		i.executeNoteDeletions(user, noteTargets)
	}

	if len(driveTargets) > 0 {
		i.executeDriveDeletions(user, driveTargets)
	}

	i.logInfo("Process completed.")
	return nil
}

func (i *deleteNotesInteractor) logAgeFilters() {
	if i.config.NoteOlderThan > 0 {
		i.logInfo(fmt.Sprintf("Note age filter: older than %s", i.config.NoteOlderThan))
	}
	if i.config.DriveOlderThan > 0 {
		i.logInfo(fmt.Sprintf("Drive age filter: older than %s", i.config.DriveOlderThan))
	}
	if keep := i.buildKeepSummary(); keep != "" {
		i.logInfo(fmt.Sprintf("Keep condition: %s", keep))
	}
}

func (i *deleteNotesInteractor) buildKeepSummary() string {
	var parts []string
	if i.config.KeepWithReactions {
		parts = append(parts, "reactions")
	}
	if i.config.KeepWithRenotes {
		parts = append(parts, "renotes")
	}
	if len(parts) == 0 {
		return ""
	}
	mode := "OR"
	if i.config.KeepConditionMode == "and" {
		mode = "AND"
	}
	return fmt.Sprintf("[%s] %s", mode, strings.Join(parts, ", "))
}

func (i *deleteNotesInteractor) confirm(noteCount, driveCount int) bool {
	fmt.Printf("\nWill delete %d notes and %d drive files.\nContinue? (y/N): ", noteCount, driveCount)
	var answer string
	fmt.Scanln(&answer)
	answer = strings.ToLower(strings.TrimSpace(answer))
	return answer == "y" || answer == "yes"
}

func (i *deleteNotesInteractor) printDryRunSummary(notes []model.Note, files []model.DriveFile) {
	i.log.Info(fmt.Sprintf("--- DRY RUN ---"))
	i.log.Info(fmt.Sprintf("Notes to delete: %d", len(notes)))
	if i.config.Verbose {
		for _, n := range notes {
			i.log.Info(fmt.Sprintf("  [%s] %s (%s)", n.KindLabel(), n.GetSummary(), n.ID))
		}
	}
	i.log.Info(fmt.Sprintf("Drive files to delete: %d", len(files)))
	if i.config.Verbose {
		for _, f := range files {
			i.log.Info(fmt.Sprintf("  [%s] %s (%s)", f.Type, f.Name, f.ID))
		}
	}
}

func (i *deleteNotesInteractor) sleepBetweenDeletions() {
	time.Sleep(time.Duration(i.config.DeleteInterval) * time.Second)
}

func (i *deleteNotesInteractor) sleepOnError() {
	i.log.Warn("Sleeping 15 minutes due to unexpected error...")
	time.Sleep(15 * time.Minute)
}

func (i *deleteNotesInteractor) logInfo(msg string) {
	if i.config.Quiet {
		return
	}
	i.log.Info(msg)
}
