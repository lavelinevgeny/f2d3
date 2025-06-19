package main

import (
	"crypto/md5"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
)

func filesAreEqual(path1, path2 string) bool {
	f1, err1 := os.Open(path1)
	if err1 != nil {
		return false
	}
	defer f1.Close()

	f2, err2 := os.Open(path2)
	if err2 != nil {
		return false
	}
	defer f2.Close()

	h1 := md5.New()
	h2 := md5.New()

	_, err1 = io.Copy(h1, f1)
	_, err2 = io.Copy(h2, f2)

	if err1 != nil || err2 != nil {
		return false
	}

	return string(h1.Sum(nil)) == string(h2.Sum(nil))
}

func ensureUniqueFilename(path string) (string, bool) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return path, false
	} else {
		// Файл существует — проверим содержимое
		srcCopy := path
		if filesAreEqual(srcCopy, path) {
			if useLog {
				log.Printf("Skipped (identical): %s", path)
			}
			return path, false // не копируем и не переименовываем
		}
	}

	dir := filepath.Dir(path)
	name := filepath.Base(path)
	ext := filepath.Ext(name)
	base := name[:len(name)-len(ext)]

	newPath := path
	count := 1
	for {
		newName := fmt.Sprintf("%s_%d%s", base, count, ext)
		newPath = filepath.Join(dir, newName)
		if _, err := os.Stat(newPath); os.IsNotExist(err) {
			return newPath, true
		}
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
