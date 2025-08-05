// main.go — параллельная обработка файлов с пулом горутин и вспомогательными функциями
package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/schollz/progressbar/v3"
)

var (
	// глобальный прогресс-бар
	bar *progressbar.ProgressBar
)

func main() {

	start := time.Now()

	logFlag := flag.Bool("log", false, "Enable logging to file")
	moveFlag := flag.Bool("move", false, "Move files instead of copying")
	helpFlag := flag.Bool("help", false, "Show usage")
	workersFlag := flag.Int("workers", runtime.NumCPU(), "maximum number of parallel workers")

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
		SrcDir:     args[0],
		DstDir:     args[1],
		UseLog:     *logFlag,
		MoveFiles:  *moveFlag,
		NumWorkers: *workersFlag,
	}
	if cfg.UseLog {
		initLog(cfg.SrcDir, cfg.DstDir)
	}

	if err := checkTargetDirectory(cfg.DstDir); err != nil {
		logf(LogErr, "cannot prepare target directory: %v", err)
		os.Exit(1)
	}

	paths := make([]string, 0, 1000)
	filepath.WalkDir(cfg.SrcDir, func(path string, d os.DirEntry, err error) error {

		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		media := lookupExtType(path)
		if media == MediaUnknown {
			return nil
		}

		paths = append(paths, path)

		return nil
	})

	total := len(paths)

	bar = progressbar.NewOptions(total,
		progressbar.OptionSetDescription("Processing"),
		progressbar.OptionShowCount(),
		progressbar.OptionShowIts(),
		progressbar.OptionSetPredictTime(true),
		progressbar.OptionClearOnFinish(),
	)

	workers := cfg.NumWorkers

	if workers < 1 {
		workers = 1
	}

	maxByFiles := (total + 100 - 1) / 100
	if maxByFiles < 1 {
		maxByFiles = 1
	}
	if workers > maxByFiles {
		workers = maxByFiles
	}

	maxByCPU := runtime.NumCPU() * 2
	if workers > maxByCPU {
		workers = maxByCPU
	}

	jobs := make(chan string)
	results := make(chan error)
	var wg sync.WaitGroup

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for p := range jobs {
				results <- processFile(cfg, p)
			}
		}()
	}
	go func() {
		wg.Wait()
		close(results)
	}()

	go func() {
		for _, p := range paths {
			jobs <- p
		}
		close(jobs)
	}()

	for err := range results {
		bar.Add(1)
		if err != nil {
			logf(LogErr, "%v", err)
		}
	}

	if len(cfg.SkipList) > 0 {
		fmt.Println("Skipped identical files:")
		for _, p := range cfg.SkipList {
			fmt.Println("  ", p)
			logf(LogNotice, "Skipped identical: %s", p)
		}
	}
	if len(cfg.RenamedList) > 0 {
		fmt.Println("Renamed files:")
		for _, m := range cfg.RenamedList {
			fmt.Println("  ", m)
			logf(LogNotice, "Renamed: %s", m)
		}
	}

	logf(LogNotice, "Done. Finished at %s", time.Now().Format(time.RFC3339))
	elapsed := time.Since(start)
	logf(LogNotice, "Processed %d files in %s", total, elapsed)
}

// checkTargetDirectory проверяет и создаёт целевой каталог
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

// processFile обрабатывает один файл: копирует либо перемещает
func processFile(cfg *AppConfig, path string) error {

	media := lookupExtType(path)
	isVideo := media == MediaVideo

	t, err := getFileDate(path)
	if err != nil {
		logf(LogWarning, "failed to get date for %s: %v", path, err)
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
	destDir := filepath.Join(cfg.DstDir, relDir)
	destPath := filepath.Join(destDir, filename)

	finalDst, skip, renamed := resolveDestination(path, destPath)

	if skip {
		cfg.SkipList = append(cfg.SkipList, path)
		logf(LogNotice, "Skipped identical: %s", path)
		return nil
	}

	if renamed {
		msg := fmt.Sprintf("%s -> %s", path, finalDst)
		cfg.RenamedList = append(cfg.RenamedList, msg)
		logf(LogNotice, "Renamed: %s", msg)
	}

	logf(LogInfo, "Copying: %s -> %s", path, finalDst)
	if err := copyFile(path, finalDst); err != nil {
		return fmt.Errorf("processing %s failed: %w", path, err)
	}
	if cfg.MoveFiles {
		if err := os.Remove(path); err != nil {
			return fmt.Errorf("failed to remove original %s: %w", path, err)
		}
	}

	return nil
}
