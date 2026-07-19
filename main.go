package main

import (
	"fmt"
	"log"
	"misskeyNotedel/internal/application/usecase"
	"misskeyNotedel/internal/config"
	"misskeyNotedel/internal/domain/model"
	"misskeyNotedel/internal/infrastructure/misskey"

	"github.com/joho/godotenv"
)

type consoleLogger struct{}

func (l *consoleLogger) Info(message string) {
	fmt.Printf("[INFO] %s\n", message)
}

func (l *consoleLogger) Warn(message string) {
	fmt.Printf("[WARN] %s\n", message)
}

func (l *consoleLogger) Error(message string, err error) {
	fmt.Printf("[ERROR] %s: %v\n", message, err)
}

func main() {
	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Configuration error: %v", err)
	}

	client, err := misskey.NewMisskeyClient(cfg.Token, cfg.Host, cfg.Scheme)
	if err != nil {
		log.Fatalf("Failed to initialize client: %v", err)
	}

	appCfg := &model.AppConfig{
		DeleteInterval:    cfg.DeleteInterval,
		NoteOlderThan:     cfg.NoteOlderThan,
		KeepWithReactions: cfg.KeepWithReactions,
		KeepWithRenotes:   cfg.KeepWithRenotes,
		KeepConditionMode: cfg.KeepConditionMode,
		DriveOlderThan:    cfg.DriveOlderThan,
		DriveMode:         cfg.DriveMode,
		SkipNotes:         cfg.SkipNotes,
		DryRun:            cfg.DryRun,
		Yes:               cfg.Yes,
		MaxDelete:         cfg.MaxDelete,
		Force:             cfg.Force,
		Verbose:           cfg.Verbose,
		Quiet:             cfg.Quiet,
		LockFile:          cfg.LockFile,
	}

	logger := &consoleLogger{}
	deleteUseCase := usecase.NewDeleteNotesUseCase(client, appCfg, logger)

	if err := deleteUseCase.Execute(); err != nil {
		log.Fatalf("Execution failed: %v", err)
	}

	fmt.Println("Process completed.")
}
