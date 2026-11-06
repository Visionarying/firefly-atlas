package api

import (
	"strings"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"firefly-atlas/database"
	"firefly-atlas/global"
	"firefly-atlas/models"
)

type userPayload struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
	Nickname string `json:"nickname"`
	Password string `json:"password"`
	Role     int    `json:"role"`
	Status   int    `json:"status"`
}

func ListUsers(c *gin.Context) {
	user := CurrentUser(c)
	if user == nil || user.Role != 1 {
		FailCode(c, 403, "无权限")
		return
	}
	var users []models.User
	global.DB.Order("id asc").Find(&users)
	OK(c, users)
}

func SaveUser(c *gin.Context) {
	current := CurrentUser(c)
	if current == nil || current.Role != 1 {
		FailCode(c, 403, "无权限")
		return
	}
	var req userPayload
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, "参数错误")
		return
	}
	req.Username = strings.TrimSpace(req.Username)
	req.Nickname = strings.TrimSpace(req.Nickname)
	if req.Nickname == "" {
		req.Nickname = req.Username
	}

	if req.ID > 0 {
		var target models.User
		if err := global.DB.First(&target, req.ID).Error; err != nil {
			Fail(c, "用户不存在")
			return
		}
		if target.ID == current.ID && (req.Role != 1 || req.Status != 1) {
			Fail(c, "不能禁用或降级当前登录账号")
			return
		}
		if target.ID == 1 && (req.Role != 1 || req.Status != 1) {
			Fail(c, "不能禁用或降级初始管理员")
			return
		}
		updates := map[string]interface{}{
			"nickname": req.Nickname,
			"role":     req.Role,
			"status":   req.Status,
		}
		if req.Password != "" {
			if len(req.Password) < 6 {
				Fail(c, "密码至少 6 位")
				return
			}
			hash, _ := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
			updates["password"] = string(hash)
			updates["token"] = ""
		}
		global.DB.Model(&target).Updates(updates)
		OK(c, gin.H{"id": target.ID})
		return
	}

	if req.Username == "" || len(req.Password) < 6 {
		Fail(c, "请填写用户名，密码至少 6 位")
		return
	}
	var count int64
	global.DB.Model(&models.User{}).Where("username = ?", req.Username).Count(&count)
	if count > 0 {
		Fail(c, "用户名已存在")
		return
	}
	if req.Role != 1 {
		req.Role = 2
	}
	if req.Status != 1 {
		req.Status = 1
	}
	hash, _ := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	newUser := models.User{
		Username: req.Username,
		Password: string(hash),
		Nickname: req.Nickname,
		Role:     req.Role,
		Status:   req.Status,
		Token:    database.RandomToken(32),
	}
	if err := global.DB.Create(&newUser).Error; err != nil {
		Fail(c, "创建失败")
		return
	}
	global.DB.Create(&models.UserConfig{
		UserID:           newUser.ID,
		PanelJSON:        models.DefaultPanelConfig,
		SearchEngineJSON: models.DefaultSearchEngines,
	})
	OK(c, gin.H{"id": newUser.ID})
}

func DeleteUser(c *gin.Context) {
	current := CurrentUser(c)
	if current == nil || current.Role != 1 {
		FailCode(c, 403, "无权限")
		return
	}
	var target models.User
	if err := global.DB.Where("id = ?", c.Param("id")).First(&target).Error; err != nil {
		Fail(c, "用户不存在")
		return
	}
	if target.ID == current.ID || target.ID == 1 {
		Fail(c, "不能删除当前账号或初始管理员")
		return
	}
	global.DB.Where("user_id = ?", target.ID).Delete(&models.ItemIcon{})
	global.DB.Where("user_id = ?", target.ID).Delete(&models.ItemIconGroup{})
	global.DB.Where("user_id = ?", target.ID).Delete(&models.UserConfig{})
	global.DB.Where("user_id = ?", target.ID).Delete(&models.File{})
	global.DB.Delete(&target)
	OK(c, nil)
}
