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

func initLog(sourceDir, targetDir string) {
	exePath, err := os.Executable()
	if err != nil {
		log.Fatalf("Failed to get executable path: %v", err)
	}
	dir := filepath.Dir(exePath)
	logFilePath := filepath.Join(dir, "f2d3.log")

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

func logDone() {
	log.Printf("Done. Finished at %s", time.Now().Format(time.RFC3339))
}
