package api

import (
	"github.com/gin-gonic/gin"

	"firefly-atlas/global"
	"firefly-atlas/models"
)

// publicSettingKeys are safe to expose to unauthenticated visitors so the login
// page can share the site title and custom styling with the main panel.
var publicSettingKeys = map[string]bool{
	"site_title":     true,
	"custom_css":     true,
	"custom_js":      true,
	"notice_enabled": true,
	"notice_title":   true,
	"notice_content": true,
	"notice_scope":   true,
}

// SiteInfo returns the public part of the system settings.
func SiteInfo(c *gin.Context) {
	settings := map[string]string{}
	var rows []models.SystemSetting
	global.DB.Find(&rows)
	for _, row := range rows {
		if publicSettingKeys[row.ConfigName] {
			settings[row.ConfigName] = row.Content
		}
	}
	OK(c, gin.H{
		"version":         global.Version,
		"publicVisitMode": global.PublicVisitMode,
		"settings":        settings,
	})
}
