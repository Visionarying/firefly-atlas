package api

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/disintegration/imaging"
	"github.com/gin-gonic/gin"

	"firefly-atlas/global"
	"firefly-atlas/models"
)

type compressReq struct {
	IDs []uint `json:"ids"`
}

// originalPathFor keeps a lossless copy of the pre-compression file so the
// destructive step can be reversed manually from uploads/originals/.
func originalPathFor(src string) (string, bool) {
	if !strings.HasPrefix(src, "/uploads/") {
		return "", false
	}
	rel := strings.TrimPrefix(src, "/uploads/")
	clean := filepath.Clean(filepath.FromSlash(rel))
	if clean == "." || strings.HasPrefix(clean, "..") {
		return "", false
	}
	return filepath.Join(global.DataDir, "uploads", "originals", clean), true
}

func removeOriginal(src string) {
	if path, ok := originalPathFor(src); ok {
		_ = os.Remove(path)
	}
}

func compressOne(file *models.File) (bool, int64, error) {
	if file == nil || file.Src == "" || file.FileType != 1 && file.FileType != 2 {
		return false, 0, nil
	}
	ext := strings.ToLower(filepath.Ext(file.Src))
	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" {
		return false, 0, nil
	}
	local, err := localUploadPath(file.Src)
	if err != nil {
		return false, 0, err
	}
	info, err := os.Stat(local)
	if err != nil {
		return false, 0, err
	}

	backup, ok := originalPathFor(file.Src)
	var source string
	createdBackup := false
	if ok {
		if _, err := os.Stat(backup); err == nil {
			// Already compressed before: re-encode from the preserved original.
			source = backup
		} else {
			if err := os.MkdirAll(filepath.Dir(backup), 0o755); err != nil {
				return false, 0, err
			}
			if err := copyFile(local, backup); err != nil {
				return false, 0, err
			}
			source = backup
			createdBackup = true
		}
	} else {
		source = local
	}

	img, err := imaging.Open(source, imaging.AutoOrientation(true))
	if err != nil {
		return false, 0, err
	}
	tmp, err := os.CreateTemp(filepath.Dir(local), ".compress-*")
	if err != nil {
		return false, 0, err
	}
	tmpPath := tmp.Name()
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return false, 0, err
	}
	out, err := os.Create(tmpPath)
	if err != nil {
		_ = os.Remove(tmpPath)
		return false, 0, err
	}
	switch ext {
	case ".png":
		err = imaging.Encode(out, img, imaging.PNG)
	default:
		err = imaging.Encode(out, img, imaging.JPEG, imaging.JPEGQuality(82))
	}
	closeErr := out.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		_ = os.Remove(tmpPath)
		return false, 0, err
	}

	newInfo, err := os.Stat(tmpPath)
	if err != nil {
		_ = os.Remove(tmpPath)
		return false, 0, err
	}
	if newInfo.Size() >= info.Size() {
		_ = os.Remove(tmpPath)
		if createdBackup {
			_ = os.Remove(backup)
		}
		return false, 0, nil
	}
	if err := os.Rename(tmpPath, local); err != nil {
		_ = os.Remove(local)
		if err := os.Rename(tmpPath, local); err != nil {
			_ = os.Remove(tmpPath)
			return false, 0, err
		}
	}
	removeThumb(*file)
	file.Size = newInfo.Size()
	global.DB.Model(&models.File{}).Where("id = ?", file.ID).Update("size", newInfo.Size())
	ensureThumb(file)
	return true, info.Size() - newInfo.Size(), nil
}

// CompressFiles re-encodes selected JPEG / PNG uploads at quality 82 and keeps
// the original under uploads/originals/. SVG, GIF, ICO and WebP are skipped.
func CompressFiles(c *gin.Context) {
	user := CurrentUser(c)
	if user == nil {
		Fail(c, "未登录")
		return
	}
	var req compressReq
	if err := c.ShouldBindJSON(&req); err != nil || len(req.IDs) == 0 {
		Fail(c, "请选择要压缩的图片")
		return
	}
	if len(req.IDs) > 500 {
		Fail(c, "一次最多压缩 500 个文件")
		return
	}
	query := global.DB.Where("id IN ?", req.IDs)
	if user.Role != 1 {
		query = query.Where("user_id = ?", user.ID)
	}
	var files []models.File
	if err := query.Find(&files).Error; err != nil {
		Fail(c, "读取文件失败")
		return
	}

	compressed := 0
	skipped := 0
	failed := 0
	var saved int64
	for i := range files {
		changed, bytes, err := compressOne(&files[i])
		if err != nil {
			failed++
			continue
		}
		if changed {
			compressed++
			saved += bytes
		} else {
			skipped++
		}
	}
	OK(c, gin.H{
		"compressed": compressed,
		"skipped":    skipped,
		"failed":     failed,
		"saved":      saved,
		"message":    fmt.Sprintf("已压缩 %d 个，节省 %s", compressed, humanBytes(saved)),
	})
}

func humanBytes(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	units := []string{"KB", "MB", "GB"}
	value := float64(size)
	for _, name := range units {
		value /= unit
		if value < unit {
			return fmt.Sprintf("%.1f %s", value, name)
		}
	}
	return fmt.Sprintf("%.1f TB", value/unit)
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	if _, err := out.ReadFrom(in); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}
