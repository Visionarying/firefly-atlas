package api

import (
	"fmt"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"firefly-atlas/database"
	"firefly-atlas/global"
	"firefly-atlas/models"
)

type loginReq struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// Simple in-memory brute-force protection. Restarting the process clears it.
type loginAttempt struct {
	fails       int
	lockedUntil time.Time
	lastSeen    time.Time
}

const (
	loginMaxFails   = 5
	loginLockPeriod = 60 * time.Second
	loginAttemptTTL = 30 * time.Minute
)

var (
	loginMu       sync.Mutex
	loginAttempts = map[string]*loginAttempt{}
)

func loginLockRemaining(ip string) time.Duration {
	loginMu.Lock()
	defer loginMu.Unlock()
	attempt, ok := loginAttempts[ip]
	if !ok {
		return 0
	}
	if time.Now().Before(attempt.lockedUntil) {
		return time.Until(attempt.lockedUntil)
	}
	return 0
}

func recordLoginFailure(ip string) {
	loginMu.Lock()
	defer loginMu.Unlock()
	now := time.Now()
	attempt, ok := loginAttempts[ip]
	if !ok {
		attempt = &loginAttempt{}
		loginAttempts[ip] = attempt
	}
	attempt.lastSeen = now
	attempt.fails++
	if attempt.fails >= loginMaxFails {
		attempt.fails = 0
		attempt.lockedUntil = now.Add(loginLockPeriod)
	}
	if len(loginAttempts) > 2048 {
		for key, item := range loginAttempts {
			if now.Sub(item.lastSeen) > loginAttemptTTL {
				delete(loginAttempts, key)
			}
		}
	}
}

func clearLoginFailures(ip string) {
	loginMu.Lock()
	delete(loginAttempts, ip)
	loginMu.Unlock()
}

func Login(c *gin.Context) {
	ip := c.ClientIP()
	if remaining := loginLockRemaining(ip); remaining > 0 {
		Fail(c, fmt.Sprintf("尝试次数过多，请 %d 秒后再试", int(remaining.Seconds())+1))
		return
	}

	var req loginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, "请输入用户名和密码")
		return
	}

	var user models.User
	if err := global.DB.Where("username = ?", req.Username).First(&user).Error; err != nil {
		recordLoginFailure(ip)
		Fail(c, "用户名或密码错误")
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)) != nil {
		recordLoginFailure(ip)
		Fail(c, "用户名或密码错误")
		return
	}
	if user.Status != 1 {
		Fail(c, "账号已被禁用")
		return
	}

	clearLoginFailures(ip)
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
