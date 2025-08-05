// media.go
package main

import (
	"path/filepath"
	"strings"
)

type MediaType int

const (
	MediaUnknown MediaType = iota
	MediaImage
	MediaVideo
)

var extType = map[string]MediaType{
	// изображения
	".jpg":  MediaImage,
	".jpeg": MediaImage,
	".png":  MediaImage,
	".heic": MediaImage,
	// видео
	".mp4": MediaVideo,
	".avi": MediaVideo,
	".mov": MediaVideo,
	".mkv": MediaVideo,
	".mts": MediaVideo,
	".3gp": MediaVideo,
}

// lookupExtType возвращает тип по расширению
func lookupExtType(path string) MediaType {
	ext := strings.ToLower(filepath.Ext(path))
	if t, ok := extType[ext]; ok {
		return t
	}
	return MediaUnknown
}
