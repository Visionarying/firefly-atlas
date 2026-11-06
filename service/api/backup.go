package api

import (
	"encoding/json"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"firefly-atlas/global"
	"firefly-atlas/models"
)

type backupItem struct {
	models.ItemIcon
	Icon models.ItemIconIcon `json:"icon"`
}

type BackupData struct {
	Version       string                 `json:"version"`
	ExportedAt    string                 `json:"exportedAt"`
	PanelConfig   json.RawMessage        `json:"panelConfig"`
	SearchEngines json.RawMessage        `json:"searchEngines"`
	Groups        []models.ItemIconGroup `json:"groups"`
	Items         []backupItem           `json:"items"`
	Settings      []models.SystemSetting `json:"settings"`
}

func ExportBackup(c *gin.Context) {
	user := CurrentUser(c)
	if user == nil {
		Fail(c, "未登录")
		return
	}
	data := BackupData{
		Version:    global.Version,
		ExportedAt: time.Now().Format(time.RFC3339),
		Groups:     []models.ItemIconGroup{},
		Items:      []backupItem{},
	}

	var cfg models.UserConfig
	if err := global.DB.Where("user_id = ?", user.ID).First(&cfg).Error; err == nil {
		data.PanelConfig = json.RawMessage(cfg.PanelJSON)
		data.SearchEngines = json.RawMessage(cfg.SearchEngineJSON)
	}

	global.DB.Where("user_id = ?", user.ID).Order("sort asc, id asc").Find(&data.Groups)
	var items []models.ItemIcon
	global.DB.Where("user_id = ?", user.ID).Order("sort asc, id asc").Find(&items)
	for _, it := range items {
		bi := backupItem{ItemIcon: it}
		_ = json.Unmarshal([]byte(it.IconJSON), &bi.Icon)
		data.Items = append(data.Items, bi)
	}

	if user.Role == 1 {
		global.DB.Order("id asc").Find(&data.Settings)
	}

	OK(c, data)
}

func ImportBackup(c *gin.Context) {
	user := CurrentUser(c)
	if user == nil {
		Fail(c, "未登录")
		return
	}
	var data BackupData
	if err := c.ShouldBindJSON(&data); err != nil {
		Fail(c, "备份文件格式不正确")
		return
	}
	if len(data.Groups) == 0 && len(data.Items) == 0 &&
		len(data.PanelConfig) == 0 && len(data.SearchEngines) == 0 {
		Fail(c, "备份内容为空")
		return
	}

	err := global.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_id = ?", user.ID).Delete(&models.ItemIcon{}).Error; err != nil {
			return err
		}
		if err := tx.Where("user_id = ?", user.ID).Delete(&models.ItemIconGroup{}).Error; err != nil {
			return err
		}

		groupMap := map[uint]uint{}
		for _, g := range data.Groups {
			oldID := g.ID
			g.ID = 0
			g.UserID = user.ID
			if err := tx.Create(&g).Error; err != nil {
				return err
			}
			groupMap[oldID] = g.ID
		}

		for _, it := range data.Items {
			icon := it.Icon
			iconBytes, _ := json.Marshal(icon)
			item := it.ItemIcon
			item.ID = 0
			item.UserID = user.ID
			item.IconJSON = string(iconBytes)
			if newGroup, ok := groupMap[item.ItemIconGroupID]; ok {
				item.ItemIconGroupID = newGroup
			}
			if err := tx.Create(&item).Error; err != nil {
				return err
			}
		}

		if len(data.PanelConfig) > 0 || len(data.SearchEngines) > 0 {
			var cfg models.UserConfig
			if err := tx.Where("user_id = ?", user.ID).First(&cfg).Error; err != nil {
				cfg = models.UserConfig{UserID: user.ID}
			}
			if len(data.PanelConfig) > 0 {
				cfg.PanelJSON = string(data.PanelConfig)
			}
			if len(data.SearchEngines) > 0 {
				cfg.SearchEngineJSON = string(data.SearchEngines)
			}
			if err := tx.Save(&cfg).Error; err != nil {
				return err
			}
		}

		if user.Role == 1 && len(data.Settings) > 0 {
			for _, s := range data.Settings {
				tx.Model(&models.SystemSetting{}).
					Where("config_name = ?", s.ConfigName).
					Update("content", s.Content)
				if s.ConfigName == "public_visit_mode" {
					global.PublicVisitMode = s.Content == "1"
				}
			}
		}
		return nil
	})
	if err != nil {
		Fail(c, "还原失败："+err.Error())
		return
	}
	OK(c, nil)
}
