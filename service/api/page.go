package api

import (
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"firefly-atlas/global"
	"firefly-atlas/models"
)

type pagePayload struct {
	ID      uint   `json:"id"`
	Title   string `json:"title"`
	Columns int    `json:"columns"`
}

func SaveItemPage(c *gin.Context) {
	user := CurrentUser(c)
	if user == nil {
		Fail(c, "未登录")
		return
	}
	var req pagePayload
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, "参数错误")
		return
	}
	req.Title = strings.TrimSpace(req.Title)
	if req.Title == "" {
		Fail(c, "请填写页面名称")
		return
	}
	if len([]rune(req.Title)) > 64 {
		req.Title = string([]rune(req.Title)[:64])
	}
	if req.Columns < 1 || req.Columns > 3 {
		req.Columns = 1
	}
	if req.ID > 0 {
		res := global.DB.Model(&models.ItemIconPage{}).
			Where("id = ? AND user_id = ?", req.ID, user.ID).
			Updates(map[string]interface{}{"title": req.Title, "columns": req.Columns})
		if res.RowsAffected == 0 {
			Fail(c, "页面不存在")
			return
		}
		OK(c, gin.H{"id": req.ID})
		return
	}
	var maxSort int
	global.DB.Model(&models.ItemIconPage{}).Where("user_id = ?", user.ID).
		Select("COALESCE(MAX(sort), -1)").Scan(&maxSort)
	page := models.ItemIconPage{UserID: user.ID, Title: req.Title, Columns: req.Columns, Sort: maxSort + 1}
	if err := global.DB.Create(&page).Error; err != nil {
		Fail(c, "保存失败")
		return
	}
	OK(c, gin.H{"id": page.ID})
}

func DeleteItemPage(c *gin.Context) {
	user := CurrentUser(c)
	if user == nil {
		Fail(c, "未登录")
		return
	}
	var page models.ItemIconPage
	if err := global.DB.Where("id = ? AND user_id = ?", c.Param("id"), user.ID).First(&page).Error; err != nil {
		Fail(c, "页面不存在")
		return
	}
	var count int64
	global.DB.Model(&models.ItemIconPage{}).Where("user_id = ?", user.ID).Count(&count)
	if count <= 1 {
		Fail(c, "至少保留一个页面")
		return
	}
	err := global.DB.Transaction(func(tx *gorm.DB) error {
		var groups []models.ItemIconGroup
		if err := tx.Where("user_id = ? AND page_id = ?", user.ID, page.ID).Find(&groups).Error; err != nil {
			return err
		}
		if len(groups) > 0 {
			ids := make([]uint, 0, len(groups))
			for _, group := range groups {
				ids = append(ids, group.ID)
			}
			if err := tx.Where("user_id = ? AND item_icon_group_id IN ?", user.ID, ids).Delete(&models.ItemIcon{}).Error; err != nil {
				return err
			}
			if err := tx.Where("user_id = ? AND id IN ?", user.ID, ids).Delete(&models.ItemIconGroup{}).Error; err != nil {
				return err
			}
		}
		return tx.Where("id = ? AND user_id = ?", page.ID, user.ID).Delete(&models.ItemIconPage{}).Error
	})
	if err != nil {
		Fail(c, "删除页面失败")
		return
	}
	OK(c, nil)
}

func SortItemPages(c *gin.Context) {
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
		global.DB.Model(&models.ItemIconPage{}).
			Where("id = ? AND user_id = ?", id, user.ID).
			Update("sort", i)
	}
	OK(c, nil)
}
