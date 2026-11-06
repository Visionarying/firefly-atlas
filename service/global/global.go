package global

import (
	"os"
	"path/filepath"

	"gorm.io/gorm"
)

var (
	// Version is injected at build time through -ldflags.
	Version = "1.1.2"
	Port    = "3002"
	DataDir = "./data"
	DB      *gorm.DB

	// PublicVisitMode is a hot cache of the public_visit_mode system setting.
	PublicVisitMode = false
)

func Init() {
	if p := os.Getenv("PORT"); p != "" {
		Port = p
	}
	if d := os.Getenv("DATA_DIR"); d != "" {
		DataDir = d
	}
	abs, err := filepath.Abs(DataDir)
	if err == nil {
		DataDir = abs
	}
	_ = os.MkdirAll(filepath.Join(DataDir, "uploads", "wallpapers"), 0o755)
	_ = os.MkdirAll(filepath.Join(DataDir, "uploads", "icons"), 0o755)
}
