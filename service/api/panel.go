package api

import (
	"encoding/json"

	"github.com/gin-gonic/gin"

	"firefly-atlas/global"
	"firefly-atlas/models"
)

type PanelData struct {
	PublicVisitMode bool                   `json:"publicVisitMode"`
	PanelConfig     json.RawMessage        `json:"panelConfig"`
	SearchEngines   json.RawMessage        `json:"searchEngines"`
	Groups          []models.ItemIconGroup `json:"groups"`
	Items           []models.ItemIcon      `json:"items"`
	Settings        map[string]string      `json:"settings"`
}

// PanelIndex returns the authenticated user's own panel data.
func PanelIndex(c *gin.Context) {
	user := CurrentUser(c)
	if user == nil {
		Fail(c, "未登录")
		return
	}
	OK(c, buildPanelData(user.ID))
}

// PanelPublic serves the administrator panel for guests when public mode is on.
func PanelPublic(c *gin.Context) {
	if !global.PublicVisitMode {
		OK(c, gin.H{"publicVisitMode": false})
		return
	}
	OK(c, buildPanelData(1))
}

func buildPanelData(userID uint) PanelData {
	data := PanelData{
		PublicVisitMode: global.PublicVisitMode,
		Groups:          []models.ItemIconGroup{},
		Items:           []models.ItemIcon{},
		Settings:        map[string]string{},
	}

	var cfg models.UserConfig
	if err := global.DB.Where("user_id = ?", userID).First(&cfg).Error; err != nil {
		cfg = models.UserConfig{
			UserID:           userID,
			PanelJSON:        models.DefaultPanelConfig,
			SearchEngineJSON: models.DefaultSearchEngines,
		}
		global.DB.Create(&cfg)
	}
	data.PanelConfig = json.RawMessage(cfg.PanelJSON)
	data.SearchEngines = json.RawMessage(cfg.SearchEngineJSON)

	global.DB.Where("user_id = ?", userID).Order("sort asc, id asc").Find(&data.Groups)
	global.DB.Where("user_id = ?", userID).Order("sort asc, id asc").Find(&data.Items)
	for i := range data.Items {
		_ = json.Unmarshal([]byte(data.Items[i].IconJSON), &data.Items[i].Icon)
	}

	var settings []models.SystemSetting
	global.DB.Find(&settings)
	for _, s := range settings {
		data.Settings[s.ConfigName] = s.Content
	}
	return data
}

type panelConfigReq struct {
	PanelConfig   json.RawMessage `json:"panelConfig"`
	SearchEngines json.RawMessage `json:"searchEngines"`
}

func SaveUserConfig(c *gin.Context) {
	user := CurrentUser(c)
	if user == nil {
		Fail(c, "未登录")
		return
	}
	var req panelConfigReq
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, "参数错误")
		return
	}
	var cfg models.UserConfig
	err := global.DB.Where("user_id = ?", user.ID).First(&cfg).Error
	if err != nil {
		cfg = models.UserConfig{UserID: user.ID}
	}
	if len(req.PanelConfig) > 0 {
		cfg.PanelJSON = string(req.PanelConfig)
	}
	if len(req.SearchEngines) > 0 {
		cfg.SearchEngineJSON = string(req.SearchEngines)
	}
	if cfg.PanelJSON == "" {
		cfg.PanelJSON = models.DefaultPanelConfig
	}
	if cfg.SearchEngineJSON == "" {
		cfg.SearchEngineJSON = models.DefaultSearchEngines
	}
	global.DB.Save(&cfg)
	OK(c, nil)
}
