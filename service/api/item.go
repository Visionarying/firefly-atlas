package api

import (
	"encoding/json"
	"strings"

	"github.com/gin-gonic/gin"

	"firefly-atlas/database"
	"firefly-atlas/global"
	"firefly-atlas/models"
)

type itemPayload struct {
	ID              uint                `json:"id"`
	ItemIconGroupID uint                `json:"itemIconGroupId"`
	Title           string              `json:"title"`
	Url             string              `json:"url"`
	LanUrl          string              `json:"lanUrl"`
	Description     string              `json:"description"`
	OpenMethod      int                 `json:"openMethod"`
	HealthCheck     bool                `json:"healthCheck"`
	Icon            models.ItemIconIcon `json:"icon"`
	Sort            int                 `json:"sort"`
}

func SaveItemIcon(c *gin.Context) {
	user := CurrentUser(c)
	if user == nil {
		Fail(c, "未登录")
		return
	}
	var req itemPayload
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, "参数错误")
		return
	}
	req.Title = strings.TrimSpace(req.Title)
	if req.Title == "" {
		Fail(c, "请填写名称")
		return
	}
	if req.Icon.ItemType == 0 {
		req.Icon.ItemType = 2
	}
	if req.Icon.ItemType == 2 && req.Icon.Text == "" {
		req.Icon.Text = firstRuneTitle(req.Title)
	}
	var group models.ItemIconGroup
	if err := global.DB.Where("id = ? AND user_id = ?", req.ItemIconGroupID, user.ID).First(&group).Error; err != nil {
		Fail(c, "分组不存在")
		return
	}
	iconBytes, _ := json.Marshal(req.Icon)

	if req.ID > 0 {
		var item models.ItemIcon
		if err := global.DB.Where("id = ? AND user_id = ?", req.ID, user.ID).First(&item).Error; err != nil {
			Fail(c, "卡片不存在")
			return
		}
		updates := map[string]interface{}{
			"item_icon_group_id": req.ItemIconGroupID,
			"title":              req.Title,
			"url":                req.Url,
			"lan_url":            req.LanUrl,
			"description":        req.Description,
			"open_method":        req.OpenMethod,
			"health_check":       req.HealthCheck,
			"icon_json":          string(iconBytes),
			"sort":               req.Sort,
		}
		global.DB.Model(&item).Updates(updates)
		req.ID = item.ID
		OK(c, gin.H{"id": item.ID})
		return
	}

	var maxSort int
	global.DB.Model(&models.ItemIcon{}).
		Where("user_id = ? AND item_icon_group_id = ?", user.ID, req.ItemIconGroupID).
		Select("COALESCE(MAX(sort), -1)").Scan(&maxSort)

	item := models.ItemIcon{
		UserID:          user.ID,
		ItemIconGroupID: req.ItemIconGroupID,
		Title:           req.Title,
		Url:             req.Url,
		LanUrl:          req.LanUrl,
		Description:     req.Description,
		OpenMethod:      req.OpenMethod,
		HealthCheck:     req.HealthCheck,
		IconJSON:        string(iconBytes),
		Sort:            maxSort + 1,
	}
	if err := global.DB.Create(&item).Error; err != nil {
		Fail(c, "保存失败")
		return
	}
	OK(c, gin.H{"id": item.ID})
}

type moveItemPayload struct {
	ID      uint `json:"id"`
	GroupID uint `json:"groupId"`
}

// MoveItemIcon moves one card to another group (possibly on another page)
// without touching its title, URL or icon configuration.
func MoveItemIcon(c *gin.Context) {
	user := CurrentUser(c)
	if user == nil {
		Fail(c, "未登录")
		return
	}
	var req moveItemPayload
	if err := c.ShouldBindJSON(&req); err != nil || req.ID == 0 || req.GroupID == 0 {
		Fail(c, "参数错误")
		return
	}
	var item models.ItemIcon
	if err := global.DB.Where("id = ? AND user_id = ?", req.ID, user.ID).First(&item).Error; err != nil {
		Fail(c, "卡片不存在")
		return
	}
	var group models.ItemIconGroup
	if err := global.DB.Where("id = ? AND user_id = ?", req.GroupID, user.ID).First(&group).Error; err != nil {
		Fail(c, "目标分组不存在")
		return
	}
	var maxSort int
	global.DB.Model(&models.ItemIcon{}).
		Where("user_id = ? AND item_icon_group_id = ?", user.ID, group.ID).
		Select("COALESCE(MAX(sort), -1)").Scan(&maxSort)
	if err := global.DB.Model(&models.ItemIcon{}).Where("id = ?", item.ID).
		Updates(map[string]interface{}{"item_icon_group_id": group.ID, "sort": maxSort + 1}).Error; err != nil {
		Fail(c, "移动失败")
		return
	}
	OK(c, gin.H{"id": item.ID, "groupId": group.ID})
}

func DeleteItemIcon(c *gin.Context) {
	user := CurrentUser(c)
	if user == nil {
		Fail(c, "未登录")
		return
	}
	id := c.Param("id")
	res := global.DB.Where("id = ? AND user_id = ?", id, user.ID).Delete(&models.ItemIcon{})
	if res.RowsAffected == 0 {
		Fail(c, "卡片不存在")
		return
	}
	OK(c, nil)
}

type sortReq struct {
	IDs []uint `json:"ids"`
}

func SortItemIcons(c *gin.Context) {
	user := CurrentUser(c)
	if user == nil {
		Fail(c, "未登录")
		return
	}
	var req sortReq
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, "参数错误")
		return
	}
	for i, id := range req.IDs {
		global.DB.Model(&models.ItemIcon{}).
			Where("id = ? AND user_id = ?", id, user.ID).
			Update("sort", i)
	}
	OK(c, nil)
}

type groupPayload struct {
	ID     uint   `json:"id"`
	PageID uint   `json:"pageId"`
	Title  string `json:"title"`
}

func SaveItemIconGroup(c *gin.Context) {
	user := CurrentUser(c)
	if user == nil {
		Fail(c, "未登录")
		return
	}
	var req groupPayload
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, "参数错误")
		return
	}
	req.Title = strings.TrimSpace(req.Title)
	if req.Title == "" {
		Fail(c, "请填写分组名称")
		return
	}
	pageID := req.PageID
	if pageID == 0 {
		pageID = database.EnsureDefaultPage(user.ID).ID
	}
	var page models.ItemIconPage
	if err := global.DB.Where("id = ? AND user_id = ?", pageID, user.ID).First(&page).Error; err != nil {
		Fail(c, "页面不存在")
		return
	}
	if req.ID > 0 {
		res := global.DB.Model(&models.ItemIconGroup{}).
			Where("id = ? AND user_id = ?", req.ID, user.ID).
			Updates(map[string]interface{}{"title": req.Title, "page_id": pageID})
		if res.RowsAffected == 0 {
			Fail(c, "分组不存在")
			return
		}
		OK(c, gin.H{"id": req.ID})
		return
	}
	var maxSort int
	global.DB.Model(&models.ItemIconGroup{}).
		Where("user_id = ? AND page_id = ?", user.ID, pageID).
		Select("COALESCE(MAX(sort), -1)").Scan(&maxSort)
	group := models.ItemIconGroup{UserID: user.ID, PageID: pageID, Title: req.Title, Sort: maxSort + 1}
	if err := global.DB.Create(&group).Error; err != nil {
		Fail(c, "保存失败")
		return
	}
	OK(c, gin.H{"id": group.ID})
}

func DeleteItemIconGroup(c *gin.Context) {
	user := CurrentUser(c)
	if user == nil {
		Fail(c, "未登录")
		return
	}
	var group models.ItemIconGroup
	if err := global.DB.Where("id = ? AND user_id = ?", c.Param("id"), user.ID).First(&group).Error; err != nil {
		Fail(c, "分组不存在")
		return
	}
	global.DB.Where("item_icon_group_id = ? AND user_id = ?", group.ID, user.ID).Delete(&models.ItemIcon{})
	global.DB.Delete(&group)
	OK(c, nil)
}

func SortItemIconGroups(c *gin.Context) {
	user := CurrentUser(c)
	if user == nil {
		Fail(c, "未登录")
		return
	}
	var req sortReq
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, "参数错误")
		return
	}
	for i, id := range req.IDs {
		global.DB.Model(&models.ItemIconGroup{}).
			Where("id = ? AND user_id = ?", id, user.ID).
			Update("sort", i)
	}
	OK(c, nil)
}
