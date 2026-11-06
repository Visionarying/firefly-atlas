package api

import (
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"firefly-atlas/global"
	"firefly-atlas/models"
)

func setupRestoreTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "test.db")), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	if err := db.AutoMigrate(
		&models.User{},
		&models.ItemIconPage{},
		&models.ItemIconGroup{},
		&models.ItemIcon{},
		&models.UserConfig{},
		&models.SystemSetting{},
		&models.File{},
		&models.HealthSample{},
	); err != nil {
		t.Fatalf("migrate test db: %v", err)
	}
	previous := global.DB
	global.DB = db
	t.Cleanup(func() {
		global.DB = previous
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})
	return db
}

// Restoring a backup written by v1.2 must not leave the v1.3 health settings
// missing, otherwise background sampling silently stops and the toggles can no
// longer be saved. Old health samples must be dropped as well because every
// card gets a new ID.
func TestApplySiteRestoreRestoresHealthDefaults(t *testing.T) {
	db := setupRestoreTestDB(t)

	admin := models.User{Username: "admin", Password: "hash", Nickname: "管理员", Role: 1, Status: 1}
	if err := db.Create(&admin).Error; err != nil {
		t.Fatalf("create admin: %v", err)
	}

	for _, setting := range []models.SystemSetting{
		{ConfigName: "health_monitor_enabled", Title: "启用状态采样", Content: "1"},
		{ConfigName: "maintenance_enabled", Title: "启用维护窗口", Content: "0"},
		{ConfigName: "maintenance_start", Title: "维护开始时间", Content: "02:00"},
		{ConfigName: "maintenance_end", Title: "维护结束时间", Content: "04:00"},
		{ConfigName: "site_title", Title: "站点标题", Content: "旧站点"},
	} {
		if err := db.Create(&setting).Error; err != nil {
			t.Fatalf("seed setting %s: %v", setting.ConfigName, err)
		}
	}
	if err := db.Create(&models.HealthSample{ItemID: 999, Online: true, CheckedAt: 1}).Error; err != nil {
		t.Fatalf("seed health sample: %v", err)
	}

	// A v1.2 backup only knows the settings that existed back then.
	backup := &siteBackup{
		Settings: []models.SystemSetting{
			{ConfigName: "site_title", Title: "站点标题", Content: "还原后的站点"},
		},
	}

	stats, err := applySiteRestore(backup, &admin, t.TempDir())
	if err != nil {
		t.Fatalf("applySiteRestore: %v", err)
	}
	if stats == nil {
		t.Fatal("applySiteRestore returned nil stats")
	}

	var samples int64
	db.Model(&models.HealthSample{}).Count(&samples)
	if samples != 0 {
		t.Errorf("health_samples = %d after restore, want 0", samples)
	}

	defaults := map[string]string{
		"health_monitor_enabled": "1",
		"maintenance_enabled":    "0",
		"maintenance_start":      "02:00",
		"maintenance_end":        "04:00",
	}
	for name, want := range defaults {
		var setting models.SystemSetting
		if err := db.Where("config_name = ?", name).First(&setting).Error; err != nil {
			t.Errorf("setting %s missing after restore: %v", name, err)
			continue
		}
		if setting.Content != want {
			t.Errorf("setting %s = %q, want %q", name, setting.Content, want)
		}
	}

	var title models.SystemSetting
	if err := db.Where("config_name = ?", "site_title").First(&title).Error; err != nil {
		t.Fatalf("site_title missing: %v", err)
	}
	if title.Content != "还原后的站点" {
		t.Errorf("site_title = %q, want restored value", title.Content)
	}
}
