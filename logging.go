// logging.go — логирование и настройка лог-файла
package main

import (
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const version = "v0.1.0"

// initLog инициализирует логирование в файл f2d3.log в текущем каталоге
func initLog(sourceDir, targetDir string) {
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatalf("Failed to get current working directory: %v", err)
	}
	logFilePath := filepath.Join(cwd, "f2d3.log")

	logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("Failed to open log file: %v", err)
	}
	log.SetOutput(logFile)

	log.Printf("f2d3 version: %s", version)
	log.Printf("Start time: %s", time.Now().Format(time.RFC3339))
	log.Printf("Command: %s", strings.Join(os.Args, " "))
	log.Printf("Source: %s", sourceDir)
	log.Printf("Target: %s", targetDir)
}

// logDone записывает время завершения работы
func logDone() {
	log.Printf("Done. Finished at %s", time.Now().Format(time.RFC3339))
}
