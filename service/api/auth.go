package api

import (
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"firefly-panel/database"
	"firefly-panel/global"
	"firefly-panel/models"
)

type loginReq struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func Login(c *gin.Context) {
	var req loginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, "请输入用户名和密码")
		return
	}

	var user models.User
	if err := global.DB.Where("username = ?", req.Username).First(&user).Error; err != nil {
		Fail(c, "用户名或密码错误")
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)) != nil {
		Fail(c, "用户名或密码错误")
		return
	}
	if user.Status != 1 {
		Fail(c, "账号已被禁用")
		return
	}

	token := database.RandomToken(32)
	global.DB.Model(&user).Update("token", token)

	OK(c, gin.H{
		"token": token,
		"user":  user,
	})
}

func Logout(c *gin.Context) {
	user := CurrentUser(c)
	if user != nil {
		global.DB.Model(user).Update("token", "")
	}
	OK(c, nil)
}

func Info(c *gin.Context) {
	OK(c, CurrentUser(c))
}

type passwordReq struct {
	OldPassword string `json:"oldPassword" binding:"required"`
	NewPassword string `json:"newPassword" binding:"required,min=6"`
}

func ChangePassword(c *gin.Context) {
	var req passwordReq
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, "新密码至少 6 位")
		return
	}
	user := CurrentUser(c)
	if user == nil {
		Fail(c, "未登录")
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.OldPassword)) != nil {
		Fail(c, "原密码不正确")
		return
	}
	hash, _ := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	global.DB.Model(user).Update("password", string(hash))
	OK(c, nil)
}

// CurrentUser reads the authenticated user injected by LoginInterceptor.
func CurrentUser(c *gin.Context) *models.User {
	if v, ok := c.Get("userInfo"); ok {
		if u, ok := v.(*models.User); ok {
			return u
		}
	}
	return nil
}
