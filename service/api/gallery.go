package api

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"firefly-atlas/global"
	"firefly-atlas/models"
)

type idsReq struct {
	IDs []uint `json:"ids"`
}

func BatchDeleteFiles(c *gin.Context) {
	user := CurrentUser(c)
	if user == nil {
		Fail(c, "未登录")
		return
	}
	var req idsReq
	if err := c.ShouldBindJSON(&req); err != nil || len(req.IDs) == 0 {
		Fail(c, "请选择要删除的文件")
		return
	}
	q := global.DB.Where("id IN ?", req.IDs)
	if user.Role != 1 {
		q = q.Where("user_id = ?", user.ID)
	}
	var files []models.File
	if err := q.Find(&files).Error; err != nil {
		Fail(c, "查询文件失败")
		return
	}
	for _, f := range files {
		if local, err := localUploadPath(f.Src); err == nil {
			_ = os.Remove(local)
		}
	}
	if len(files) > 0 {
		ids := make([]uint, 0, len(files))
		for _, f := range files {
			ids = append(ids, f.ID)
		}
		global.DB.Where("id IN ?", ids).Delete(&models.File{})
		// Best-effort cloud cleanup; never block the local delete on a cloud outage.
		go func() {
			client, _, err := currentCloudClient()
			if err != nil {
				return
			}
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			defer cancel()
			for _, f := range files {
				if f.CloudKey != "" {
					_ = client.Delete(ctx, f.CloudKey)
					time.Sleep(350 * time.Millisecond)
				}
			}
		}()
	}
	OK(c, gin.H{"deleted": len(files)})
}

type moveFilesReq struct {
	IDs      []uint `json:"ids"`
	FileType int    `json:"fileType"`
}

func MoveFiles(c *gin.Context) {
	user := CurrentUser(c)
	if user == nil {
		Fail(c, "未登录")
		return
	}
	var req moveFilesReq
	if err := c.ShouldBindJSON(&req); err != nil || len(req.IDs) == 0 {
		Fail(c, "请选择要移动的文件")
		return
	}
	if req.FileType != 1 && req.FileType != 2 {
		Fail(c, "目标分类不正确")
		return
	}
	q := global.DB.Model(&models.File{}).Where("id IN ?", req.IDs)
	if user.Role != 1 {
		q = q.Where("user_id = ?", user.ID)
	}
	res := q.Update("file_type", req.FileType)
	OK(c, gin.H{"moved": res.RowsAffected})
}

type renameFileReq struct {
	ID       uint   `json:"id"`
	FileName string `json:"fileName"`
}

func RenameFile(c *gin.Context) {
	user := CurrentUser(c)
	if user == nil {
		Fail(c, "未登录")
		return
	}
	var req renameFileReq
	if err := c.ShouldBindJSON(&req); err != nil || req.ID == 0 {
		Fail(c, "参数错误")
		return
	}
	name := strings.TrimSpace(req.FileName)
	if name == "" {
		Fail(c, "文件名不能为空")
		return
	}
	if len([]rune(name)) > 120 {
		name = string([]rune(name)[:120])
	}
	q := global.DB.Model(&models.File{}).Where("id = ?", req.ID)
	if user.Role != 1 {
		q = q.Where("user_id = ?", user.ID)
	}
	res := q.Update("file_name", name)
	if res.RowsAffected == 0 {
		Fail(c, "文件不存在")
		return
	}
	OK(c, nil)
}

func ApplySyncFiles(c *gin.Context) {
	user := CurrentUser(c)
	if user == nil {
		Fail(c, "未登录")
		return
	}
	var req idsReq
	if err := c.ShouldBindJSON(&req); err != nil || len(req.IDs) == 0 {
		Fail(c, "请选择要申请同步的文件")
		return
	}
	res := global.DB.Model(&models.File{}).
		Where("id IN ? AND user_id = ?", req.IDs, user.ID).
		Update("sync_status", 1)
	OK(c, gin.H{"pending": res.RowsAffected})
}

func UploadFilesToCloud(c *gin.Context) {
	if adminOnly(c) == nil {
		return
	}
	var req idsReq
	if err := c.ShouldBindJSON(&req); err != nil || len(req.IDs) == 0 {
		Fail(c, "请选择要推送到云端的文件")
		return
	}
	client, _, err := currentCloudClient()
	if err != nil {
		Fail(c, err.Error())
		return
	}
	var files []models.File
	global.DB.Where("id IN ?", req.IDs).Find(&files)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	success, failed := pushFilesToCloud(ctx, client, files)
	OK(c, gin.H{"success": success, "failed": failed})
}

type auditReq struct {
	IDs    []uint `json:"ids"`
	Action string `json:"action"` // approve | reject
}

type auditFile struct {
	models.File
	UserName string `json:"userName"`
}

func AuditList(c *gin.Context) {
	if adminOnly(c) == nil {
		return
	}
	var files []models.File
	global.DB.Where("sync_status = 1").Order("id desc").Find(&files)
	userIDs := make([]uint, 0, len(files))
	for _, f := range files {
		userIDs = append(userIDs, f.UserID)
	}
	names := map[uint]string{}
	if len(userIDs) > 0 {
		var users []models.User
		global.DB.Where("id IN ?", userIDs).Find(&users)
		for _, u := range users {
			names[u.ID] = u.Username
		}
	}
	out := make([]auditFile, 0, len(files))
	for _, f := range files {
		out = append(out, auditFile{File: f, UserName: names[f.UserID]})
	}
	OK(c, out)
}

func AuditFiles(c *gin.Context) {
	if adminOnly(c) == nil {
		return
	}
	var req auditReq
	if err := c.ShouldBindJSON(&req); err != nil || len(req.IDs) == 0 {
		Fail(c, "请选择待审核文件")
		return
	}
	if req.Action == "reject" {
		res := global.DB.Model(&models.File{}).Where("id IN ?", req.IDs).Updates(map[string]interface{}{
			"sync_status": 3, "audit_remark": "管理员已驳回",
		})
		OK(c, gin.H{"rejected": res.RowsAffected})
		return
	}
	client, _, err := currentCloudClient()
	if err != nil {
		Fail(c, err.Error())
		return
	}
	var files []models.File
	global.DB.Where("id IN ?", req.IDs).Find(&files)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	success, failed := pushFilesToCloud(ctx, client, files)
	OK(c, gin.H{"success": success, "failed": failed})
}

func PublicIcons(c *gin.Context) {
	user := CurrentUser(c)
	if user == nil {
		Fail(c, "未登录")
		return
	}
	var files []models.File
	global.DB.Where("file_type = ? AND (user_id = 1 OR sync_status = 2)", 2).
		Order("id desc").Limit(200).Find(&files)
	OK(c, files)
}
