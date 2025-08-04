// files.go — копирование, сравнение и разрешение конфликтов имён
package main

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// filesAreEqual возвращает true, если два файла идентичны по содержимому
// или ошибку, если не удалось провести сравнение.
func filesAreEqual(path1, path2 string) (bool, error) {
	// Быстрая проверка размера
	info1, err := os.Stat(path1)
	if err != nil {
		return false, err
	}
	info2, err := os.Stat(path2)
	if err != nil {
		return false, err
	}
	if info1.Size() != info2.Size() {
		return false, nil
	}

	// Открываем оба файла
	f1, err := os.Open(path1)
	if err != nil {
		return false, err
	}
	defer func() {
		if cerr := f1.Close(); cerr != nil {
			logf(LogWarning, "Failed to close file %s: %v", path1, cerr)
		}
	}()

	f2, err := os.Open(path2)
	if err != nil {
		return false, err
	}
	defer func() {
		if cerr := f2.Close(); cerr != nil {
			logf(LogWarning, "Failed to close file %s: %v", path2, cerr)
		}
	}()

	// Вычисляем MD5-хеши
	h1 := md5.New()
	h2 := md5.New()

	if _, err := io.Copy(h1, f1); err != nil {
		return false, err
	}
	if _, err := io.Copy(h2, f2); err != nil {
		return false, err
	}

	// Сравниваем
	return bytes.Equal(h1.Sum(nil), h2.Sum(nil)), nil
}

// resolveDestination решает, куда копировать src -> dst:
// 1) если dst не существует — вернёт (dst, skip=false, renamed=false).
// 2) если dst существует и идентичен src — вернёт (dst, skip=true, renamed=false).
// 3) если dst существует, но отличается — найдёт dst_1, dst_2… и вернёт (uniqueDst, skip=false, renamed=true).
func resolveDestination(src, dst string) (finalDst string, skip bool, renamed bool) {
	// Если нет такого файла — копируем прямо
	if _, err := os.Stat(dst); os.IsNotExist(err) {
		return dst, false, false
	}
	// Файл есть — сравниваем
	equal, err := filesAreEqual(src, dst)
	if err != nil {
		logf(LogErr, "Failed to compare files: %s <-> %s : %v", src, dst, err)
		// в случае ошибки будем считать, что файл не совпадает
		return dst, false, false
	}
	if equal {
		return dst, true, false // идентичный файл — пропустить
	}

	// файл есть, но отличается — ищем новое имя
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

// copyFile копирует содержимое src в dst, создавая все необходимые директории.
func copyFile(src, dst string) error {
	// Открываем исходный файл
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file %q: %w", src, err)
	}
	defer in.Close()

	// Создаём папку для dst
	if err := os.MkdirAll(filepath.Dir(dst), os.ModePerm); err != nil {
		return fmt.Errorf("failed to create directory %q: %w", filepath.Dir(dst), err)
	}

	// Создаём файл назначения
	out, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file %q: %w", dst, err)
	}
	defer out.Close()

	// Копируем данные
	if _, err := io.Copy(out, in); err != nil {
		return fmt.Errorf("failed copy from %q to %q: %w", src, dst, err)
	}
	return nil
}
