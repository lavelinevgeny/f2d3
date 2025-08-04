// logging.go — логирование в файл и вывод в консоль с уровнями syslog
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

// version приложения
const version = "v0.1.0"

// LogLevel типизированный уровень логирования
type LogLevel string

// Уровни, соответствующие RFC5424/syslog
const (
	LogEmerg   LogLevel = "EMERG"   // system is unusable
	LogAlert   LogLevel = "ALERT"   // action must be taken immediately
	LogCrit    LogLevel = "CRIT"    // critical conditions
	LogErr     LogLevel = "ERR"     // error conditions
	LogWarning LogLevel = "WARNING" // warning conditions
	LogNotice  LogLevel = "NOTICE"  // normal but significant
	LogInfo    LogLevel = "INFO"    // informational
	LogDebug   LogLevel = "DEBUG"   // debug-level messages
)

// capitalize делает первую букву строки заглавной (не трогая остальное)
func capitalize(s string) string {
	if s == "" {
		return s
	}
	r, sz := utf8.DecodeRuneInString(s)
	return string(unicode.ToUpper(r)) + s[sz:]
}

// logf печатает в файл (если useLog=true) и в консоль:
// - INFO, NOTICE → stdout (с заглавной буквы)
// - ERR, CRIT, ALERT, EMERG → stderr
// - WARNING, DEBUG → только в файл
func logf(level LogLevel, format string, args ...interface{}) {
	// 1) формируем сырое сообщение
	raw := fmt.Sprintf(format, args...)

	// 2) для INFO/NOTICE приводим к sentence case
	msg := raw
	if level == LogInfo || level == LogNotice {
		msg = capitalize(raw)
	}

	prefix := fmt.Sprintf("[%s] ", strings.ToUpper(string(level)))

	// 3) пишем в файл
	if cfg.UseLog {
		log.Printf(prefix + msg)
	}

	// 4) дублируем в консоль по уровню
	switch level {
	case LogInfo, LogNotice:
		fmt.Println(prefix + msg)
	case LogErr, LogCrit, LogAlert, LogEmerg:
		fmt.Fprintln(os.Stderr, prefix+msg)
	}
}

// initLog настраивает вывод пакета log в файл f2d3.log
func initLog(sourceDir, targetDir string) {

	if !cfg.UseLog {
		return
	}

	cwd, err := os.Getwd()
	if err != nil {
		log.Fatalf("Failed to get current working directory: %v", err)
	}
	logFilePath := filepath.Join(cwd, "f2d3.log")

	// открываем или создаём файл
	logFile, err := os.OpenFile(logFilePath,
		os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o666)
	if err != nil {
		log.Fatalf("Failed to open log file: %v", err)
	}
	log.SetOutput(logFile)

	// стартовые записи
	logf(LogInfo, "f2d3 version: %s", version)
	logf(LogInfo, "Start time: %s", time.Now().Format(time.RFC3339))
	logf(LogInfo, "Command: %s", strings.Join(os.Args, " "))
	logf(LogInfo, "Source: %s", sourceDir)
	logf(LogInfo, "Target: %s", targetDir)
}
