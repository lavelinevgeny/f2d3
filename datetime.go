// datetime.go — логика получения даты для файлов (EXIF или fallback)
package main

import (
	"os"
	"time"

	mp4 "github.com/abema/go-mp4"

	"github.com/rwcarlsen/goexif/exif"
)

const (
	// Оffset between mp4 epoch (1904) and Unix epoch (1970)
	mp4EpochOffset = 2082844800
	// Допустимый сдвиг в будущем (24 часа)
	futureThreshold = 24 * time.Hour
	// Минимальная разумная дата (например, 1970-01-01)
	minValidDate = 0
)

func getFileDate(path string) (time.Time, error) {
	media := lookupExtType(path)

	switch media {
	case MediaImage:
		// EXIF для картинок
		f, err := os.Open(path)
		if err == nil {
			defer f.Close()
			if x, err := exif.Decode(f); err == nil {
				return x.DateTime()
			}
		}

	case MediaVideo:

		f, err := os.Open(path)
		if err != nil {
			logf(LogWarning, "MP4: cannot open %s: %v", path, err)
		} else {
			defer f.Close()
			boxes, err := mp4.ExtractBoxWithPayload(
				f, nil,
				mp4.BoxPath{mp4.BoxTypeMoov(), mp4.BoxTypeMvhd()},
			)
			if err != nil {
				logf(LogWarning, "MP4: extract mvhd failed for %s: %v", path, err)
			} else if len(boxes) > 0 {
				if mvhd, ok := boxes[0].Payload.(*mp4.Mvhd); ok {
					var ct uint64
					if mvhd.Version > 0 {
						ct = mvhd.CreationTimeV1
					} else {
						ct = uint64(mvhd.CreationTimeV0)
					}
					unixSec := int64(ct) - mp4EpochOffset
					dt := time.Unix(unixSec, 0)
					if validDate(dt) {
						return dt, nil
					}
					logf(LogWarning, "MP4: got unrealistic date %v in %s", dt, path)
				}
			}
		}
	}

	// Общий fallback: модифицированное время файла
	info, err := os.Stat(path)
	if err != nil {
		return time.Time{}, err
	}
	return info.ModTime(), nil
}

// validDate проверяет, что дата лежит в разумных пределах:
// не раньше Unix-эпохи и не позже (сейчас + threshold)
func validDate(t time.Time) bool {
	unix := t.Unix()
	if unix < minValidDate {
		return false
	}
	if t.After(time.Now().Add(futureThreshold)) {
		return false
	}
	return true
}
