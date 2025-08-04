package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/schollz/progressbar/v3"
)

var (
	// расширения, которые считаем видео
	videoExtensions = map[string]bool{
		".mp4": true, ".avi": true, ".mov": true,
		".mkv": true, ".mts": true, ".3gp": true,
	}
	// прогрессбар
	bar *progressbar.ProgressBar
	// флаги из CLI
	useLog, moveFiles bool
	// списки для итогового отчёта
	skipList    []string
	renamedList []string
)

func main() {
	var src, dst string
	var positionalArgs []string

	for _, arg := range os.Args[1:] {
		switch arg {
		case "--log":
			useLog = true
		case "--move":
			moveFiles = true
		default:
			positionalArgs = append(positionalArgs, arg)
		}
	}

	if len(positionalArgs) < 2 {
		fmt.Println("Usage: f2d3 <sourceDir> <targetDir> [--log] [--move]")
		fmt.Printf("Received %d argument(s):\n", len(os.Args)-1)
		for i, arg := range os.Args[1:] {
			fmt.Printf("%d: %s\n", i, arg)
		}
		os.Exit(1)
	}

	src = positionalArgs[0]
	dst = positionalArgs[1]

	if useLog {
		initLog(src, dst)
	}

	checkTargetDirectory(dst)

	total, err := countFiles(src)
	if err != nil {
		logf(LogErr, "Failed to count files: %v", err)
		os.Exit(1)
	}

	bar = progressbar.NewOptions(total,
		progressbar.OptionSetDescription("Processing"),
		progressbar.OptionShowCount(),
		progressbar.OptionShowIts(),
		progressbar.OptionSetPredictTime(true),
	)

	err = filepath.WalkDir(src, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		return processFile(path, dst)
	})
	if err != nil {
		logf(LogErr, "Error walking directory: %v", err)
		os.Exit(1)
	}

	// Итоговый отчёт
	if len(skipList) > 0 {
		fmt.Println("\nSkipped identical files:")
		for _, p := range skipList {
			fmt.Println("  ", p)
			logf(LogInfo, "Skipped identical: %s", p)
		}
	}
	if len(renamedList) > 0 {
		fmt.Println("\nRenamed files:")
		for _, m := range renamedList {
			fmt.Println("  ", m)
			logf(LogInfo, "Renamed: %s", m)
		}
	}

	if useLog {
		logDone()
	}
}

// checkTargetDirectory проверяет, существует ли целевой каталог, и если он не пуст, запрашивает подтверждение.
func checkTargetDirectory(targetDir string) {
	entries, err := os.ReadDir(targetDir)
	if err != nil {
		if os.IsNotExist(err) {
			if err := os.MkdirAll(targetDir, os.ModePerm); err != nil {
				logf(LogErr, "Failed to create target directory: %v", err)
				os.Exit(1)
			}
			return
		}
		logf(LogErr, "Failed to read target directory: %v", err)
		os.Exit(1)
	}
	if len(entries) > 0 {
		fmt.Print("Target directory is not empty. Continue? (y/N): ")
		scanner := bufio.NewScanner(os.Stdin)
		scanner.Scan()
		if strings.TrimSpace(strings.ToLower(scanner.Text())) != "y" {
			fmt.Println("Operation cancelled.")
			os.Exit(0)
		}
	}
}

// countFiles проходит по всему дереву и возвращает количество файлов (не директорий).
func countFiles(root string) (int, error) {
	count := 0
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			count++
		}
		return nil
	})
	return count, err
}

// processFile обрабатывает один файл: вычисляет дату, категорию и решает, копировать или перемещать.
func processFile(path, targetBase string) error {
	ext := strings.ToLower(filepath.Ext(path))
	isVideo := videoExtensions[ext]

	t, err := getFileDate(path)
	if err != nil {
		logf(LogWarning, "Failed to get date for %s: %v", path, err)
		t = time.Now()
	}

	year := t.Format("2006")
	date := t.Format("20060102")
	category := ""
	if isVideo {
		category = "VIDEO"
	}

	relDir := filepath.Join(year, date, category)
	filename := filepath.Base(path)
	destDir := filepath.Join(targetBase, relDir)
	destPath := filepath.Join(destDir, filename)

	finalDst, skip, renamed := resolveDestination(path, destPath)

	if skip {
		skipList = append(skipList, path)
		logf(LogInfo, "Skipped identical: %s", path)
		bar.Add(1)
		return nil
	}

	if renamed {
		msg := fmt.Sprintf("%s -> %s", path, finalDst)
		renamedList = append(renamedList, msg)
		logf(LogInfo, "Renamed: %s", msg)
	}

	logf(LogInfo, "Copying: %s -> %s", path, finalDst)
	if err := copyFile(path, finalDst); err != nil {
		logf(LogErr, "Copy failed: %s -> %s : %v", path, finalDst, err)
		return err
	}
	logf(LogInfo, "Copied: %s", finalDst)

	if moveFiles {
		if err := os.Remove(path); err != nil {
			logf(LogErr, "Failed to remove original file: %s : %v", path, err)
			return err
		}
		logf(LogInfo, "Removed: %s", path)
	}

	bar.Add(1)
	return nil
}
