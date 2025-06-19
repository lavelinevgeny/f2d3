package main

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rwcarlsen/goexif/exif"
)

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
