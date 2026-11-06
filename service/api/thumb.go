package api

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/disintegration/imaging"
	_ "golang.org/x/image/webp"

	"firefly-atlas/database"
	"firefly-atlas/global"
	"firefly-atlas/models"
)

// ensureThumb creates a 320px preview for an uploaded image and stores its path
// on the file record. SVG, GIF and ICO are skipped so animation/vector data is
// never modified.
func ensureThumb(file *models.File) string {
	if file == nil || file.ID == 0 || file.Src == "" {
		return file.Thumb
	}
	if file.Thumb != "" {
		if _, err := localUploadPath(file.Thumb); err == nil {
			return file.Thumb
		}
	}
	ext := strings.ToLower(filepath.Ext(file.Src))
	if ext == ".svg" || ext == ".gif" || ext == ".ico" {
		return file.Thumb
	}
	local, err := localUploadPath(file.Src)
	if err != nil {
		return file.Thumb
	}
	img, err := imaging.Open(local, imaging.AutoOrientation(true))
	if err != nil {
		return file.Thumb
	}
	thumb := imaging.Fit(img, 320, 320, imaging.Lanczos)
	dir := filepath.Join(global.DataDir, "uploads", "thumbs")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return file.Thumb
	}
	name := fmt.Sprintf("%d_%s", time.Now().UnixNano(), database.RandomToken(3))
	var dst string
	if file.FileType == 2 {
		name += ".png"
		dst = filepath.Join(dir, name)
		if err := imaging.Save(thumb, dst); err != nil {
			return file.Thumb
		}
	} else {
		name += ".jpg"
		dst = filepath.Join(dir, name)
		if err := imaging.Save(thumb, dst, imaging.JPEGQuality(82)); err != nil {
			return file.Thumb
		}
	}
	file.Thumb = "/uploads/thumbs/" + name
	global.DB.Model(&models.File{}).Where("id = ?", file.ID).Update("thumb", file.Thumb)
	return file.Thumb
}

func removeThumb(file models.File) {
	if file.Thumb == "" {
		return
	}
	if local, err := localUploadPath(file.Thumb); err == nil {
		_ = os.Remove(local)
	}
}
