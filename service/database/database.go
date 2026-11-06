package database

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/glebarez/sqlite"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"firefly-atlas/global"
	"firefly-atlas/models"
)

// Init opens the embedded SQLite database, migrates the schema and seeds defaults.
func Init() error {
	db, err := gorm.Open(sqlite.Open(filepath.Join(global.DataDir, "database.db")), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return err
	}
	global.DB = db

	if err := db.AutoMigrate(
		&models.User{},
		&models.ItemIconPage{},
		&models.ItemIconGroup{},
		&models.ItemIcon{},
		&models.UserConfig{},
		&models.SystemSetting{},
		&models.File{},
		&models.CloudStorage{},
	); err != nil {
		return err
	}

	seed()
	migratePages()
	migrateBranding()
	return nil
}

func seed() {
	db := global.DB

	var count int64
	db.Model(&models.User{}).Count(&count)
	if count == 0 {
		adminPwd := os.Getenv("ADMIN_PASSWORD")
		generated := false
		if adminPwd == "" {
			adminPwd = RandomToken(9)
			generated = true
		}
		hash, _ := bcrypt.GenerateFromPassword([]byte(adminPwd), bcrypt.DefaultCost)
		admin := models.User{
			Username: "admin",
			Password: string(hash),
			Nickname: "管理员",
			Role:     1,
			Status:   1,
			Token:    RandomToken(32),
		}
		if err := db.Create(&admin).Error; err != nil {
			log.Println("seed admin:", err)
			return
		}
		if generated {
			log.Println("[firefly-atlas] admin account created")
			log.Printf("[firefly-atlas] initial password: %s", adminPwd)
			log.Println("[firefly-atlas] change it after first login, shown only once")
		} else {
			log.Println("[firefly-atlas] admin account created with ADMIN_PASSWORD env")
		}
		seedPanel(admin.ID)
	}

	settings := []models.SystemSetting{
		{ConfigName: "public_visit_mode", Title: "公开访客模式", Content: "0"},
		{ConfigName: "site_title", Title: "站点标题", Content: "Firefly Atlas"},
		{ConfigName: "allow_register", Title: "允许注册", Content: "0"},
		{ConfigName: "custom_css", Title: "自定义 CSS", Content: ""},
		{ConfigName: "custom_js", Title: "自定义 JS", Content: ""},
		{ConfigName: "notice_enabled", Title: "公告开关", Content: "0"},
		{ConfigName: "notice_title", Title: "公告标题", Content: ""},
		{ConfigName: "notice_content", Title: "公告内容", Content: ""},
		{ConfigName: "notice_scope", Title: "公告位置", Content: "home"},
	}
	for _, s := range settings {
		var count int64
		db.Model(&models.SystemSetting{}).Where("config_name = ?", s.ConfigName).Count(&count)
		if count == 0 {
			db.Create(&s)
		}
	}

	var ps models.SystemSetting
	if db.Where("config_name = ?", "public_visit_mode").First(&ps).Error == nil {
		global.PublicVisitMode = ps.Content == "1"
	}
}

// SeedDemoData creates a pleasant first-run experience.
func seedPanel(userID uint) {
	db := global.DB

	group := models.ItemIconGroup{UserID: userID, Title: "我的应用", Sort: 0}
	group2 := models.ItemIconGroup{UserID: userID, Title: "系统与服务", Sort: 1}
	db.Create(&group)
	db.Create(&group2)

	type demo struct {
		Group  uint
		Title  string
		Url    string
		LanUrl string
		Desc   string
		Text   string
		Color  string
	}
	demos := []demo{
		{group.ID, "NAS 控制台", "https://nas.example.com", "http://192.168.1.10:5000", "照片、文件与备份中心", "N", "linear-gradient(135deg,#5b8cff,#3a5cff)"},
		{group.ID, "影音中心", "https://media.example.com", "http://192.168.1.10:8096", "电影、剧集与音乐库", "M", "linear-gradient(135deg,#ff7a59,#ff3d71)"},
		{group.ID, "私人网盘", "https://cloud.example.com", "http://192.168.1.10:8080", "文件同步与在线预览", "C", "linear-gradient(135deg,#22d3ee,#0ea5e9)"},
		{group2.ID, "服务器", "https://server.example.com", "", "主机监控与容器管理", "S", "linear-gradient(135deg,#a78bfa,#7c3aed)"},
		{group2.ID, "路由器", "http://192.168.1.1", "http://192.168.1.1", "网络与设备管理", "R", "linear-gradient(135deg,#34d399,#059669)"},
		{group2.ID, "面板设置", "https://demo.example.com", "", "Firefly Atlas 文档", "F", "linear-gradient(135deg,#fbbf24,#f59e0b)"},
	}
	for i, d := range demos {
		iconBytes, _ := json.Marshal(models.ItemIconIcon{
			ItemType:        2,
			Text:            d.Text,
			BackgroundColor: d.Color,
		})
		db.Create(&models.ItemIcon{
			UserID:          userID,
			ItemIconGroupID: d.Group,
			Title:           d.Title,
			Url:             d.Url,
			LanUrl:          d.LanUrl,
			Description:     d.Desc,
			OpenMethod:      1,
			IconJSON:        string(iconBytes),
			Sort:            i,
		})
	}

	db.Create(&models.UserConfig{
		UserID:           userID,
		PanelJSON:        models.DefaultPanelConfig,
		SearchEngineJSON: models.DefaultSearchEngines,
	})
}

// EnsureDefaultPage returns the user's first page, creating a default one when
// the account has no pages yet.
func EnsureDefaultPage(userID uint) models.ItemIconPage {
	var page models.ItemIconPage
	if err := global.DB.Where("user_id = ?", userID).Order("sort asc, id asc").First(&page).Error; err == nil {
		return page
	}
	page = models.ItemIconPage{UserID: userID, Title: "我的主页", Sort: 0, Columns: 1}
	_ = global.DB.Create(&page).Error
	return page
}

// migratePages assigns every legacy group to a page. It is idempotent and only
// touches groups whose page_id is still zero.
func migratePages() {
	db := global.DB
	var count int64
	db.Model(&models.ItemIconGroup{}).Where("page_id = 0 OR page_id IS NULL").Count(&count)
	if count == 0 {
		return
	}
	backupDatabaseFile()

	var userIDs []uint
	db.Model(&models.ItemIconGroup{}).Where("page_id = 0 OR page_id IS NULL").
		Distinct().Pluck("user_id", &userIDs)
	for _, userID := range userIDs {
		page := EnsureDefaultPage(userID)
		if page.ID == 0 {
			continue
		}
		db.Model(&models.ItemIconGroup{}).
			Where("user_id = ? AND (page_id = 0 OR page_id IS NULL)", userID).
			Update("page_id", page.ID)
	}
}

func backupDatabaseFile() {
	src := filepath.Join(global.DataDir, "database.db")
	if _, err := os.Stat(src); err != nil {
		return
	}
	dst := src + ".bak-" + time.Now().Format("20060102-150405")
	in, err := os.Open(src)
	if err != nil {
		return
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return
	}
	if _, err := io.Copy(out, in); err == nil {
		_ = out.Sync()
		log.Printf("[firefly-atlas] database backup before page migration: %s", dst)
	}
	_ = out.Close()
}

func migrateBranding() {
	db := global.DB
	// Only replace the exact old defaults. Custom titles are left untouched.
	db.Model(&models.SystemSetting{}).
		Where("config_name = ? AND content = ?", "site_title", "Firefly Panel").
		Update("content", "Firefly Atlas")

	var configs []models.UserConfig
	db.Find(&configs)
	for _, cfg := range configs {
		next := strings.ReplaceAll(cfg.PanelJSON, `"logoText": "Firefly Panel"`, `"logoText": "Firefly Atlas"`)
		next = strings.ReplaceAll(next, `"logoText":"Firefly Panel"`, `"logoText":"Firefly Atlas"`)
		next = strings.ReplaceAll(next, "Firefly Panel · 私有云导航", "Firefly Atlas · 私有云导航")
		if next != cfg.PanelJSON {
			db.Model(&cfg).Update("panel_json", next)
		}
	}

	db.Model(&models.ItemIcon{}).
		Where("description = ?", "Firefly Panel 文档").
		Update("description", "Firefly Atlas 文档")
}

func RandomToken(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return ""
	}
	return hex.EncodeToString(b)
}
