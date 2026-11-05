package api

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"firefly-panel/database"
	"firefly-panel/global"
	"firefly-panel/models"
)

var allowedImageExt = map[string]bool{
	".jpg": true, ".jpeg": true, ".png": true, ".gif": true,
	".webp": true, ".svg": true, ".ico": true, ".avif": true, ".bmp": true,
}

func UploadFile(c *gin.Context) {
	user := CurrentUser(c)
	if user == nil {
		Fail(c, "未登录")
		return
	}
	file, err := c.FormFile("file")
	if err != nil {
		Fail(c, "请选择文件")
		return
	}
	if file.Size > 30<<20 {
		Fail(c, "文件不能超过 30MB")
		return
	}
	ext := strings.ToLower(filepath.Ext(file.Filename))
	if !allowedImageExt[ext] {
		Fail(c, "仅支持图片格式")
		return
	}

	fileType := 1
	if c.PostForm("fileType") == "2" {
		fileType = 2
	}
	sub := "wallpapers"
	if fileType == 2 {
		sub = "icons"
	}
	name := fmt.Sprintf("%d_%s%s", time.Now().UnixMilli(), database.RandomToken(6), ext)
	dst := filepath.Join(global.DataDir, "uploads", sub, name)
	if err := c.SaveUploadedFile(file, dst); err != nil {
		Fail(c, "文件保存失败")
		return
	}

	src := "/uploads/" + sub + "/" + name
	record := models.File{
		UserID:   user.ID,
		FileName: file.Filename,
		Src:      src,
		FileType: fileType,
		Size:     file.Size,
	}
	global.DB.Create(&record)
	OK(c, record)
}

func ListFiles(c *gin.Context) {
	user := CurrentUser(c)
	if user == nil {
		Fail(c, "未登录")
		return
	}
	fileType, _ := strconv.Atoi(c.Query("fileType"))
	q := global.DB.Where("user_id = ?", user.ID)
	if fileType > 0 {
		q = q.Where("file_type = ?", fileType)
	}
	var files []models.File
	q.Order("id desc").Find(&files)
	OK(c, files)
}

func DeleteFile(c *gin.Context) {
	user := CurrentUser(c)
	if user == nil {
		Fail(c, "未登录")
		return
	}
	var f models.File
	if err := global.DB.Where("id = ? AND user_id = ?", c.Param("id"), user.ID).First(&f).Error; err != nil {
		Fail(c, "文件不存在")
		return
	}
	if strings.HasPrefix(f.Src, "/uploads/") {
		_ = os.Remove(filepath.Join(global.DataDir, filepath.FromSlash(strings.TrimPrefix(f.Src, "/"))))
	}
	global.DB.Delete(&f)
	OK(c, nil)
}

func GetSettings(c *gin.Context) {
	var settings []models.SystemSetting
	global.DB.Order("id asc").Find(&settings)
	OK(c, settings)
}

type settingItem struct {
	ConfigName string `json:"configName"`
	Content    string `json:"content"`
}

func SaveSettings(c *gin.Context) {
	user := CurrentUser(c)
	if user == nil || user.Role != 1 {
		FailCode(c, 403, "无权限")
		return
	}
	var items []settingItem
	if err := c.ShouldBindJSON(&items); err != nil {
		Fail(c, "参数错误")
		return
	}
	for _, item := range items {
		global.DB.Model(&models.SystemSetting{}).
			Where("config_name = ?", item.ConfigName).
			Update("content", item.Content)
		if item.ConfigName == "public_visit_mode" {
			global.PublicVisitMode = item.Content == "1"
		}
	}
	OK(c, nil)
}
