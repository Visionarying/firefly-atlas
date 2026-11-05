package database

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log"
	"os"
	"path/filepath"

	"github.com/glebarez/sqlite"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"firefly-panel/global"
	"firefly-panel/models"
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
		&models.ItemIconGroup{},
		&models.ItemIcon{},
		&models.UserConfig{},
		&models.SystemSetting{},
		&models.File{},
	); err != nil {
		return err
	}

	seed()
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
			log.Println("[firefly-panel] admin account created")
			log.Printf("[firefly-panel] initial password: %s", adminPwd)
			log.Println("[firefly-panel] change it after first login, shown only once")
		} else {
			log.Println("[firefly-panel] admin account created with ADMIN_PASSWORD env")
		}
		seedPanel(admin.ID)
	}

	settings := []models.SystemSetting{
		{ConfigName: "public_visit_mode", Title: "公开访客模式", Content: "0"},
		{ConfigName: "site_title", Title: "站点标题", Content: "Firefly Panel"},
		{ConfigName: "allow_register", Title: "允许注册", Content: "0"},
	}
	for _, s := range settings {
		var exist models.SystemSetting
		if db.Where("config_name = ?", s.ConfigName).First(&exist).Error != nil {
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
		{group2.ID, "面板设置", "https://demo.example.com", "", "Firefly Panel 文档", "F", "linear-gradient(135deg,#fbbf24,#f59e0b)"},
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

func RandomToken(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return ""
	}
	return hex.EncodeToString(b)
}
