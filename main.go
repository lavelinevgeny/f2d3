package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rwcarlsen/goexif/exif"
	"github.com/schollz/progressbar/v3"
)

var videoExtensions = map[string]bool{
	".mp4": true,
	".avi": true,
	".mov": true,
	".mkv": true,
	".mts": true,
	".3gp": true,
}

var bar *progressbar.ProgressBar

const version = "v0.1.0"

var useLog bool

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: f2d3 <sourceDir> <targetDir> [--log]")
		os.Exit(1)
	}

	sourceDir := os.Args[1]
	targetDir := os.Args[2]
	useLog = len(os.Args) > 3 && os.Args[3] == "--log"

	if useLog {
		setupLogFile()
		log.Printf("f2d3 version: %s", version)
		log.Printf("Start time: %s", time.Now().Format(time.RFC3339))
		log.Printf("Command: %s", strings.Join(os.Args, " "))
		log.Printf("Source: %s", sourceDir)
		log.Printf("Target: %s", targetDir)
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
		log.Printf("Done. Finished at %s", time.Now().Format(time.RFC3339))
	}
}

func setupLogFile() {
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

func isImage(ext string) bool {
	switch ext {
	case ".jpg", ".jpeg", ".png", ".heic":
		return true
	default:
		return false
	}
}

func getFileDate(path string) (time.Time, error) {
	ext := strings.ToLower(filepath.Ext(path))
	if isImage(ext) {
		f, err := os.Open(path)
		if err != nil {
			return time.Time{}, err
		}
		defer f.Close()
		x, err := exif.Decode(f)
		if err == nil {
			return x.DateTime()
		}
	}
	info, err := os.Stat(path)
	if err != nil {
		return time.Time{}, err
	}
	return info.ModTime(), nil
}

func ensureUniqueFilename(path string) (string, bool) {
	dir := filepath.Dir(path)
	name := filepath.Base(path)
	ext := filepath.Ext(name)
	base := strings.TrimSuffix(name, ext)

	newPath := path
	count := 1
	for {
		if _, err := os.Stat(newPath); os.IsNotExist(err) {
			return newPath, newPath != path
		}
		newName := fmt.Sprintf("%s_%d%s", base, count, ext)
		newPath = filepath.Join(dir, newName)
		count++
	}
}

func copyFile(src, dst string) error {
	if useLog {
		log.Printf("Opening source: %s", src)
	}
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer in.Close()

	if useLog {
		log.Printf("Creating target directory: %s", filepath.Dir(dst))
	}
	if err := os.MkdirAll(filepath.Dir(dst), os.ModePerm); err != nil {
		return fmt.Errorf("failed to create target dir: %w", err)
	}

	if useLog {
		log.Printf("Creating destination file: %s", dst)
	}
	out, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer out.Close()

	if useLog {
		log.Printf("Copying data...")
	}
	_, err = io.Copy(out, in)
	if err != nil {
		return fmt.Errorf("copy failed: %w", err)
	}
	return nil
}
