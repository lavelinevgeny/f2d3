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
		".mp4": true, ".avi": true, ".mov": true,
		".mkv": true, ".mts": true, ".3gp": true,
	}
	bar         *progressbar.ProgressBar
	useLog      bool
	skipList    []string
	renamedList []string
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: f2d3 <sourceDir> <targetDir> [--log]")
		os.Exit(1)
	}
	src := os.Args[1]
	dst := os.Args[2]
	useLog = len(os.Args) > 3 && os.Args[3] == "--log"

	if useLog {
		initLog(src, dst)
	}

	checkTargetDirectory(dst)

	total, err := countFiles(src)
	if err != nil {
		log.Fatalf("Failed to count files: %v", err)
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
		return processFile(path, src, dst)
	})
	if err != nil {
		log.Fatalf("Error walking directory: %v", err)
	}

	// Вывод пропущенных идентичных файлов
	if len(skipList) > 0 {
		fmt.Println()
		fmt.Println("Skipped identical files:")
		for _, p := range skipList {
			fmt.Println("  ", p)
			if useLog {
				log.Printf("[INFO] Skipped identical: %s", p)
			}
		}
	}

	// Вывод переименованных файлов
	if len(renamedList) > 0 {
		fmt.Println()
		fmt.Println("Renamed files:")
		for _, m := range renamedList {
			fmt.Println("  ", m)
			if useLog {
				log.Printf("[INFO] Renamed: %s", m)
			}
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
				log.Fatalf("Failed to create target directory: %v", err)
			}
			return
		}
		log.Fatalf("Failed to read target directory: %v", err)
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

// countFiles проходит по всему tree и возвращает количество файлов (не директорий).
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

// processFile обрабатывает один файл: вычисляет дату, категорию, определяет, копировать/переименовывать/пропускать.
func processFile(path, baseDir, targetBase string) error {
	ext := strings.ToLower(filepath.Ext(path))
	isVideo := videoExtensions[ext]

	t, err := getFileDate(path)
	if err != nil && useLog {
		log.Printf("[WARN] Failed to get date for %s: %v", path, err)
	}
	if err != nil {
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
		msg := fmt.Sprintf("%s", path)
		fmt.Println("[INFO] Skipped identical:", msg)
		skipList = append(skipList, msg)
		if useLog {
			log.Printf("[INFO] Skipped identical: %s", msg)
		}
		bar.Add(1)
		return nil
	}

	if renamed {
		msg := fmt.Sprintf("%s -> %s", path, finalDst)
		fmt.Println("[INFO] Renamed:", msg)
		renamedList = append(renamedList, msg)
		if useLog {
			log.Printf("[INFO] Renamed: %s", msg)
		}
	}

	if useLog {
		log.Printf("Copying: %s -> %s", path, finalDst)
	}
	if err := copyFile(path, finalDst); err != nil {
		if useLog {
			log.Printf("[ERROR] Copy failed: %s -> %s : %v", path, finalDst, err)
		}
		return err
	}
	if useLog {
		log.Printf("Copied: %s", finalDst)
	}
	bar.Add(1)
	return nil
}
