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

	exif "github.com/rwcarlsen/goexif/exif"
)

var videoExtensions = map[string]bool{
	".mp4": true,
	".avi": true,
	".mov": true,
	".mkv": true,
	".mts": true,
	".3gp": true,
}

func main() {
	if len(os.Args) != 3 {
		log.Fatalf("Usage: f2d3 <sourceDir> <targetDir>")
	}

	sourceDir := os.Args[1]
	targetDir := os.Args[2]

	fmt.Println("f2d3 - file to date tree organizer")

	checkTargetDirectory(targetDir)

	err := filepath.WalkDir(sourceDir, func(path string, d os.DirEntry, err error) error {
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
		log.Printf("[WARN] Failed to get date for %s: %v", path, err)
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
	if renamed {
		log.Printf("[INFO] Renamed %s -> %s", filename, filepath.Base(newPath))
	}
	return copyFile(path, newPath)
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
	if err := os.MkdirAll(filepath.Dir(dst), os.ModePerm); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}
