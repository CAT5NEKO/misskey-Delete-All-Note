package main

import (
	"fmt"
	"log"
	"misskeyNotedel/internal/application/usecase"
	"misskeyNotedel/internal/domain/model"
	"misskeyNotedel/internal/infrastructure/misskey"
	"os"
	"strconv"

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

	deleteInterval := getEnvInt("DELETE_INTERVAL", 30)
	deleteOlderThanDays := getEnvInt("DELETE_OLDER_THAN_DAYS", 0)
	keepReactions := getEnvBool("KEEP_WITH_REACTIONS", false)
	keepRenotes := getEnvBool("KEEP_WITH_RENOTES", false)

	client, err := misskey.NewMisskeyClient()
	if err != nil {
		log.Fatalf("Failed to initialize client: %v", err)
	}

	config := model.NewAppConfig(deleteInterval, deleteOlderThanDays, keepReactions, keepRenotes)
	logger := &consoleLogger{}
	deleteUseCase := usecase.NewDeleteNotesUseCase(client, config, logger)

	if err := deleteUseCase.Execute(); err != nil {
		log.Fatalf("Execution failed: %v", err)
	}

	fmt.Println("Process completed.")
}

func getEnvInt(key string, defaultValue int) int {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultValue
	}
	return value
}

func getEnvBool(key string, defaultValue bool) bool {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	value, err := strconv.ParseBool(valueStr)
	if err != nil {
		return defaultValue
	}
	return value
}
