package main

import (
	"bufio"
	"flag"
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
)

type AppConfig struct {
	SrcDir string
	DstDir string
	// флаги из CLI
	UseLog    bool
	MoveFiles bool
	// списки для итогового отчёта
	SkipList    []string
	RenamedList []string
}

var (
	logFlag  = flag.Bool("log", false, "Enable logging to file")
	moveFlag = flag.Bool("move", false, "Move files instead of copying")
	helpFlag = flag.Bool("help", false, "Show usage")
)
var cfg *AppConfig

func main() {

	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(),
			"Usage: %s [options] <sourceDir> <targetDir>\n", os.Args[0])
		flag.PrintDefaults()
	}

	flag.Parse()
	if *helpFlag {
		flag.Usage()
		os.Exit(0)
	}

	args := flag.Args()
	if len(args) < 2 {
		flag.Usage()
		os.Exit(1)
	}

	cfg = &AppConfig{
		SrcDir:    args[0],
		DstDir:    args[1],
		UseLog:    *logFlag,
		MoveFiles: *moveFlag,
	}

	// инициализация логирования
	initLog(cfg.SrcDir, cfg.DstDir)

	// проверка и подготовка целевой директории
	if err := checkTargetDirectory(cfg.DstDir); err != nil {
		logf(LogErr, "Cannot prepare target directory: %v", err)
		os.Exit(1)
	}

	// подсчет файлов
	total, err := countFiles(cfg.SrcDir)
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

	// обход дерева
	err = filepath.WalkDir(cfg.SrcDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		return processFile(cfg, path)
	})
	if err != nil {
		logf(LogErr, "Error walking directory: %v", err)
		os.Exit(1)
	}

	// вывод результатов
	if len(cfg.SkipList) > 0 {
		fmt.Println("\nSkipped identical files:")
		for _, p := range cfg.SkipList {
			fmt.Println("  ", p)
			logf(LogInfo, "Skipped identical: %s", p)
		}
	}
	if len(cfg.RenamedList) > 0 {
		fmt.Println("\nRenamed files:")
		for _, m := range cfg.RenamedList {
			fmt.Println("  ", m)
			logf(LogInfo, "Renamed: %s", m)
		}
	}

	logf(LogInfo, "Done. Finished at %s", time.Now().Format(time.RFC3339))
}

// checkTargetDirectory проверяет и создаёт целевую директорию, возвращая ошибку вместо выхода
func checkTargetDirectory(targetDir string) error {
	entries, err := os.ReadDir(targetDir)
	if err != nil {
		if os.IsNotExist(err) {
			if mkErr := os.MkdirAll(targetDir, os.ModePerm); mkErr != nil {
				return fmt.Errorf("failed to create target directory %q: %w", targetDir, mkErr)
			}
			return nil
		}
		return fmt.Errorf("failed to read target directory %q: %w", targetDir, err)
	}
	if len(entries) > 0 {
		fmt.Print("Target directory is not empty. Continue? (y/N): ")
		s := bufio.NewScanner(os.Stdin)
		s.Scan()
		if strings.ToLower(strings.TrimSpace(s.Text())) != "y" {
			return fmt.Errorf("operation cancelled by user")
		}
	}
	return nil
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
func processFile(cfg *AppConfig, path string) error {
	targetBase := cfg.DstDir
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
		cfg.SkipList = append(cfg.SkipList, path)
		logf(LogInfo, "Skipped identical: %s", path)
		bar.Add(1)
		return nil
	}

	if renamed {
		msg := fmt.Sprintf("%s -> %s", path, finalDst)
		cfg.RenamedList = append(cfg.RenamedList, msg)
		logf(LogInfo, "Renamed: %s", msg)
	}

	logf(LogInfo, "Copying: %s -> %s", path, finalDst)
	if err := copyFile(path, finalDst); err != nil {
		logf(LogErr, "Copy failed: %s -> %s : %v", path, finalDst, err)
		return err
	}
	logf(LogInfo, "Copied: %s", finalDst)

	if cfg.MoveFiles {
		if err := os.Remove(path); err != nil {
			logf(LogErr, "Failed to remove original file: %s : %v", path, err)
			return err
		}
		logf(LogInfo, "Removed: %s", path)
	}

	bar.Add(1)
	return nil
}
