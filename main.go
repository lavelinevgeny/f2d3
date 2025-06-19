// main.go — запуск, аргументы, логика обхода и прогресса

package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/schollz/progressbar/v3"
)

var (
	videoExtensions = map[string]bool{
		".mp4": true,
		".avi": true,
		".mov": true,
		".mkv": true,
		".mts": true,
		".3gp": true,
	}
	bar    *progressbar.ProgressBar
	useLog bool
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: f2d3 <sourceDir> <targetDir> [--log]")
		os.Exit(1)
	}

	sourceDir := os.Args[1]
	targetDir := os.Args[2]
	useLog = len(os.Args) > 3 && os.Args[3] == "--log"

	if useLog {
		initLog(sourceDir, targetDir)
	}

	checkTargetDirectory(targetDir)

	totalFiles, err := countFiles(sourceDir)
	if err != nil {
		log.Fatalf("Failed to count files: %v", err)
	}

	bar = progressbar.NewOptions(totalFiles,
		progressbar.OptionSetDescription("Processing"),
		progressbar.OptionShowCount(),
		progressbar.OptionShowIts(),
		progressbar.OptionSetPredictTime(true),
		progressbar.OptionClearOnFinish(),
	)

	err = filepath.WalkDir(sourceDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		return processFile(path, sourceDir, targetDir)
	})
	if err != nil {
		log.Fatalf("Error walking directory: %v", err)
	}

	if useLog {
		logDone()
	}
}

func checkTargetDirectory(targetDir string) {
	entries, err := os.ReadDir(targetDir)
	if err != nil {
		if os.IsNotExist(err) {
			err := os.MkdirAll(targetDir, os.ModePerm)
			if err != nil {
				log.Fatalf("Failed to create target directory: %v", err)
			}
			return
		} else {
			log.Fatalf("Failed to read target directory: %v", err)
		}
	}

	if len(entries) > 0 {
		fmt.Print("Target directory is not empty. Continue? (y/N): ")
		scanner := bufio.NewScanner(os.Stdin)
		scanner.Scan()
		response := strings.TrimSpace(strings.ToLower(scanner.Text()))
		if response != "y" {
			fmt.Println("Operation cancelled.")
			os.Exit(0)
		}
	}
}

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

func processFile(path, baseDir, targetBase string) error {
	ext := strings.ToLower(filepath.Ext(path))
	isVideo := videoExtensions[ext]
	t, err := getFileDate(path)
	if err != nil {
		if useLog {
			log.Printf("[WARN] Failed to get date for %s: %v", path, err)
		}
		t = time.Now()
	}

	year := t.Format("2006")
	date := t.Format("20060102")
	category := ""
	if isVideo {
		category = "VIDEO"
	}

	relPath := filepath.Join(year, date, category)
	filename := filepath.Base(path)
	targetDir := filepath.Join(targetBase, relPath)
	targetPath := filepath.Join(targetDir, filename)

	newPath, renamed := ensureUniqueFilename(targetPath)
	if useLog && renamed {
		log.Printf("[INFO] Renamed %s -> %s", filename, filepath.Base(newPath))
	}

	if newPath == targetPath {
		if _, err := os.Stat(newPath); err == nil && filesAreEqual(path, newPath) {
			bar.Add(1)
			return nil // пропускаем
		}
	}

	if useLog {
		log.Printf("Copying: %s -> %s", path, newPath)
	}
	err = copyFile(path, newPath)
	if err != nil {
		if useLog {
			log.Printf("[ERROR] Copy failed: %s -> %s : %v", path, newPath, err)
		}
		return err
	}
	if useLog {
		log.Printf("Copied: %s", newPath)
	}
	bar.Add(1)
	return nil
}
