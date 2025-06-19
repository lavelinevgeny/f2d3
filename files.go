// files.go — копирование, сравнение, переименование
package main

import (
	"crypto/md5"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
)

// filesAreEqual возвращает true, если два файла полностью идентичны по байтам.
func filesAreEqual(path1, path2 string) bool {
	f1, err := os.Open(path1)
	if err != nil {
		return false
	}
	defer f1.Close()

	f2, err := os.Open(path2)
	if err != nil {
		return false
	}
	defer f2.Close()

	h1 := md5.New()
	h2 := md5.New()
	if _, err := io.Copy(h1, f1); err != nil {
		return false
	}
	if _, err := io.Copy(h2, f2); err != nil {
		return false
	}
	return string(h1.Sum(nil)) == string(h2.Sum(nil))
}

// resolveDestination решает, куда копировать src -> dst:
// 1) если dst не существует — вернёт (dst, skip=false, renamed=false).
// 2) если dst существует и идентичен src — вернёт (dst, skip=true, renamed=false).
// 3) если dst существует, но отличается — найдёт dst_1, dst_2… и вернёт (uniqueDst, skip=false, renamed=true).
func resolveDestination(src, dst string) (finalDst string, skip, renamed bool) {
	// 1) dst нет — копируем прямо туда
	if _, err := os.Stat(dst); os.IsNotExist(err) {
		return dst, false, false
	}
	// 2) dst есть — сравниваем содержимое
	if filesAreEqual(src, dst) {
		return dst, true, false
	}
	// 3) dst есть и отличается — ищем уникальное имя
	dir := filepath.Dir(dst)
	base := filepath.Base(dst)
	ext := filepath.Ext(base)
	name := base[:len(base)-len(ext)]

	for i := 1; ; i++ {
		candidate := fmt.Sprintf("%s_%d%s", name, i, ext)
		newPath := filepath.Join(dir, candidate)
		if _, err := os.Stat(newPath); os.IsNotExist(err) {
			return newPath, false, true
		}
	}
}

// copyFile копирует содержимое src в dst, создавая директории при необходимости.
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
		log.Printf("Creating directory: %s", filepath.Dir(dst))
	}
	if err := os.MkdirAll(filepath.Dir(dst), os.ModePerm); err != nil {
		return fmt.Errorf("failed to create dir: %w", err)
	}

	if useLog {
		log.Printf("Creating destination file: %s", dst)
	}
	out, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create dst file: %w", err)
	}
	defer out.Close()

	if useLog {
		log.Printf("Copying data...")
	}
	if _, err := io.Copy(out, in); err != nil {
		return fmt.Errorf("copy failed: %w", err)
	}
	return nil
}
