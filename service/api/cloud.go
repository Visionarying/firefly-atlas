package api

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"mime"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	cloudstore "firefly-panel/cloud"
	"firefly-panel/database"
	"firefly-panel/global"
	"firefly-panel/models"
)

const cloudConfigRowID = 1

var (
	cloudNameCleaner = regexp.MustCompile(`[\\/:*?"<>|\s]+`)
	cloudImageSecret = make([]byte, 32)
)

func init() {
	if _, err := rand.Read(cloudImageSecret); err != nil {
		copy(cloudImageSecret, []byte("firefly-panel-image-secret-fallback"))
	}
}

func cloudImageSignature(key string, exp int64) string {
	mac := hmac.New(sha256.New, cloudImageSecret)
	_, _ = fmt.Fprintf(mac, "%s|%d", key, exp)
	return hex.EncodeToString(mac.Sum(nil))
}

func signedCloudImageURL(key string) string {
	exp := time.Now().Add(time.Hour).Unix()
	return fmt.Sprintf("/api/v1/panel/cloud/image?key=%s&exp=%d&sig=%s",
		url.QueryEscape(key), exp, cloudImageSignature(key, exp))
}

func defaultCloudConfig() cloudstore.Config {
	return cloudstore.Config{
		Provider: "github",
		GitHub: cloudstore.GitHubConfig{
			Branch:   "main",
			BasePath: "firefly-panel",
		},
		S3: cloudstore.S3Config{Region: "auto"},
	}
}

func loadCloudConfig() (cloudstore.Config, error) {
	var row models.CloudStorage
	if err := global.DB.Where("id = ?", cloudConfigRowID).First(&row).Error; err != nil {
		return defaultCloudConfig(), nil
	}
	cfg := defaultCloudConfig()
	if row.ConfigJSON == "" {
		return cfg, nil
	}
	if err := json.Unmarshal([]byte(row.ConfigJSON), &cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}

func saveCloudConfig(cfg cloudstore.Config) error {
	body, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	var row models.CloudStorage
	err = global.DB.Where("id = ?", cloudConfigRowID).First(&row).Error
	if err != nil {
		row = models.CloudStorage{ID: cloudConfigRowID, ConfigJSON: string(body)}
		return global.DB.Create(&row).Error
	}
	return global.DB.Model(&row).Update("config_json", string(body)).Error
}

func cloudClientFromConfig(cfg cloudstore.Config) (cloudstore.Client, error) {
	if !cfg.Enabled {
		return nil, errors.New("云存储未启用，请先在管理员面板开启")
	}
	return cloudstore.New(cfg)
}

func currentCloudClient() (cloudstore.Client, cloudstore.Config, error) {
	cfg, err := loadCloudConfig()
	if err != nil {
		return nil, cfg, err
	}
	client, err := cloudClientFromConfig(cfg)
	return client, cfg, err
}

// ---------------------------------------------------------------- admin config

func GetCloudConfig(c *gin.Context) {
	if adminOnly(c) == nil {
		return
	}
	cfg, err := loadCloudConfig()
	if err != nil {
		Fail(c, "读取云存储配置失败: "+err.Error())
		return
	}
	OK(c, cfg)
}

type cloudConfigReq struct {
	Config cloudstore.Config `json:"config"`
}

func SaveCloudConfig(c *gin.Context) {
	if adminOnly(c) == nil {
		return
	}
	var req cloudConfigReq
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, "参数格式错误")
		return
	}
	if req.Config.Provider != "github" && req.Config.Provider != "s3" {
		Fail(c, "请选择 GitHub 或 S3 / R2 存储")
		return
	}
	if err := saveCloudConfig(req.Config); err != nil {
		Fail(c, "保存云存储配置失败: "+err.Error())
		return
	}
	OK(c, nil)
}

type cloudTestResp struct {
	Private     bool   `json:"private"`
	Description string `json:"description"`
}

func TestCloudConfig(c *gin.Context) {
	if adminOnly(c) == nil {
		return
	}
	cfg, err := loadCloudConfig()
	if err != nil {
		Fail(c, "读取云存储配置失败: "+err.Error())
		return
	}
	if c.Request.ContentLength > 0 {
		var body cloudstore.Config
		if bindErr := c.ShouldBindJSON(&body); bindErr == nil && (body.Provider != "" || body.Enabled) {
			cfg = body
		}
	}
	if cfg.Provider == "" {
		cfg = defaultCloudConfig()
	}
	client, err := cloudstore.New(cfg)
	if err != nil {
		Fail(c, "初始化云存储客户端失败: "+err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	private, err := client.Test(ctx)
	if err != nil {
		Fail(c, "连接测试失败: "+err.Error())
		return
	}
	desc := "公有存储 (Public) - 文件可被外部直接访问"
	if private {
		desc = "私有存储 (Private) - 仅通过 Token / 密钥访问"
	}
	OK(c, cloudTestResp{Private: private, Description: desc})
}

// ---------------------------------------------------------------- helpers

func localUploadPath(src string) (string, error) {
	if !strings.HasPrefix(src, "/uploads/") {
		return "", errors.New("不是本地上传文件")
	}
	clean := filepath.Clean(filepath.Join(global.DataDir, filepath.FromSlash(strings.TrimPrefix(src, "/"))))
	base := filepath.Clean(filepath.Join(global.DataDir, "uploads"))
	if clean != base && !strings.HasPrefix(clean, base+string(os.PathSeparator)) {
		return "", errors.New("非法文件路径")
	}
	return clean, nil
}

func cloudFileName(f models.File) string {
	ext := strings.ToLower(filepath.Ext(f.Src))
	if ext == "" {
		ext = strings.ToLower(filepath.Ext(f.FileName))
	}
	base := strings.TrimSuffix(f.FileName, filepath.Ext(f.FileName))
	base = strings.TrimSpace(cloudNameCleaner.ReplaceAllString(base, "_"))
	if base == "" {
		base = "image"
	}
	if len([]rune(base)) > 48 {
		base = string([]rune(base)[:48])
	}
	return fmt.Sprintf("%s_%d%s", base, f.ID, ext)
}

func cloudKeyForFile(f models.File) string {
	dir := "images/wallpapers/"
	if f.FileType == 2 {
		dir = "images/icons/"
	}
	return dir + cloudFileName(f)
}

func contentTypeForFile(name string) string {
	if ct := mime.TypeByExtension(strings.ToLower(filepath.Ext(name))); ct != "" {
		return ct
	}
	return "application/octet-stream"
}

func uploadFileRecordToCloud(ctx context.Context, client cloudstore.Client, f models.File) (string, string, error) {
	local, err := localUploadPath(f.Src)
	if err != nil {
		return "", "", err
	}
	data, err := os.ReadFile(local)
	if err != nil {
		return "", "", err
	}
	key := cloudKeyForFile(f)
	url, err := client.Upload(ctx, key, data, contentTypeForFile(local))
	if err != nil {
		return "", "", err
	}
	return key, url, nil
}

func markFileSynced(id uint, key, url string) {
	now := time.Now()
	global.DB.Model(&models.File{}).Where("id = ?", id).Updates(map[string]interface{}{
		"sync_status":  2,
		"cloud_key":    key,
		"cloud_url":    url,
		"synced_at":    &now,
		"audit_remark": "",
	})
}

func pushFilesToCloud(ctx context.Context, client cloudstore.Client, files []models.File) (int, []string) {
	success := 0
	var failed []string
	for _, f := range files {
		key, url, err := uploadFileRecordToCloud(ctx, client, f)
		if err != nil {
			failed = append(failed, f.FileName+": "+err.Error())
			continue
		}
		markFileSynced(f.ID, key, url)
		success++
	}
	return success, failed
}

// dualWriteNewFile mirrors a freshly uploaded local file when the admin enabled
// local + cloud dual write. It is intentionally best-effort: a cloud outage must
// never make the local upload fail.
func dualWriteNewFile(fileID uint) {
	cfg, err := loadCloudConfig()
	if err != nil || !cfg.Enabled || !cfg.DualWrite {
		return
	}
	client, err := cloudstore.New(cfg)
	if err != nil {
		return
	}
	var f models.File
	if err := global.DB.First(&f, fileID).Error; err != nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	key, url, err := uploadFileRecordToCloud(ctx, client, f)
	if err == nil {
		markFileSynced(f.ID, key, url)
	}
}

// ---------------------------------------------------------------- cloud images

type cloudImage struct {
	Key      string    `json:"key"`
	Name     string    `json:"name"`
	Size     int64     `json:"size"`
	URL      string    `json:"url"`
	Category int       `json:"category"` // 1 wallpaper, 2 icon
	Updated  time.Time `json:"updatedAt"`
}

func listCloudImages(ctx context.Context, client cloudstore.Client) ([]cloudImage, error) {
	prefixes := []string{"images/wallpapers", "images/icons", "images"}
	seen := map[string]bool{}
	var out []cloudImage
	for _, prefix := range prefixes {
		items, err := client.List(ctx, prefix)
		if err != nil {
			continue
		}
		for _, it := range items {
			if it.IsDirectory || seen[it.Key] {
				continue
			}
			ext := strings.ToLower(path.Ext(it.Name))
			if !allowedImageExt[ext] {
				continue
			}
			seen[it.Key] = true
			category := 1
			if strings.Contains(it.Key, "/icons/") || strings.HasSuffix(prefix, "icons") {
				category = 2
			}
			out = append(out, cloudImage{
				Key: it.Key, Name: it.Name, Size: it.Size, URL: it.URL,
				Category: category, Updated: it.UpdatedAt,
			})
		}
	}
	return out, nil
}

func CloudListImages(c *gin.Context) {
	if adminOnly(c) == nil {
		return
	}
	client, _, err := currentCloudClient()
	if err != nil {
		Fail(c, err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	images, err := listCloudImages(ctx, client)
	if err != nil {
		Fail(c, "读取云端图库失败: "+err.Error())
		return
	}
	for i := range images {
		images[i].URL = signedCloudImageURL(images[i].Key)
	}
	OK(c, images)
}

// CloudImageProxy streams a private cloud image through a short-lived signed
// URL. The URL is generated by the admin-only gallery endpoint, so provider
// credentials and user tokens never appear in image URLs or access logs.
func CloudImageProxy(c *gin.Context) {
	key := strings.TrimSpace(c.Query("key"))
	exp, _ := strconv.ParseInt(c.Query("exp"), 10, 64)
	sig := strings.TrimSpace(c.Query("sig"))
	if key == "" || strings.Contains(key, "..") || !strings.HasPrefix(key, "images/") ||
		exp == 0 || time.Now().Unix() > exp || !hmac.Equal([]byte(sig), []byte(cloudImageSignature(key, exp))) {
		FailCode(c, 403, "图片链接无效或已过期")
		return
	}
	client, _, err := currentCloudClient()
	if err != nil {
		Fail(c, err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	data, err := client.Download(ctx, key)
	if err != nil {
		Fail(c, "读取云端图片失败: "+err.Error())
		return
	}
	c.Header("Cache-Control", "private, max-age=600")
	c.Data(200, contentTypeForFile(key), data)
}

type cloudSyncReq struct {
	Keys []string `json:"keys"`
}

func syncCloudKeys(ctx context.Context, client cloudstore.Client, keys []string, ownerID uint) (int, []string) {
	success := 0
	var failed []string
	for _, key := range keys {
		clean := path.Clean(strings.TrimPrefix(key, "/"))
		if clean == "." || strings.HasPrefix(clean, "../") || strings.Contains(clean, "/../") {
			continue
		}
		if !strings.HasPrefix(clean, "images/") {
			continue
		}
		data, err := client.Download(ctx, clean)
		if err != nil || len(data) == 0 {
			if err == nil {
				err = errors.New("云端文件为空")
			}
			failed = append(failed, clean+": "+err.Error())
			continue
		}
		category := 2
		sub := "icons"
		if !strings.Contains(clean, "/icons/") {
			category = 1
			sub = "wallpapers"
		}
		name := fmt.Sprintf("%d_%s%s", time.Now().UnixMilli(), database.RandomToken(6), strings.ToLower(path.Ext(clean)))
		dstDir := filepath.Join(global.DataDir, "uploads", sub)
		if err := os.MkdirAll(dstDir, 0o755); err != nil {
			failed = append(failed, clean+": "+err.Error())
			continue
		}
		dst := filepath.Join(dstDir, name)
		if err := os.WriteFile(dst, data, 0o644); err != nil {
			failed = append(failed, clean+": "+err.Error())
			continue
		}
		src := "/uploads/" + sub + "/" + name
		var existing models.File
		err = global.DB.Where("cloud_key = ?", clean).First(&existing).Error
		if err == nil {
			global.DB.Model(&existing).Updates(map[string]interface{}{
				"src": src, "size": int64(len(data)), "sync_status": 2,
				"cloud_url": client.PublicURL(clean), "synced_at": time.Now(),
			})
		} else {
			record := models.File{
				UserID: ownerID, FileName: path.Base(clean), Src: src,
				FileType: category, Size: int64(len(data)), SyncStatus: 2,
				CloudKey: clean, CloudURL: client.PublicURL(clean), SyncedAt: ptrTime(time.Now()),
			}
			global.DB.Create(&record)
		}
		success++
	}
	return success, failed
}

func ptrTime(t time.Time) *time.Time { return &t }

func CloudSyncImages(c *gin.Context) {
	if adminOnly(c) == nil {
		return
	}
	client, _, err := currentCloudClient()
	if err != nil {
		Fail(c, err.Error())
		return
	}
	var req cloudSyncReq
	if err := c.ShouldBindJSON(&req); err != nil || len(req.Keys) == 0 {
		Fail(c, "请选择要拉取到本地的图片")
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	success, failed := syncCloudKeys(ctx, client, req.Keys, CurrentUser(c).ID)
	OK(c, gin.H{"success": success, "failed": failed})
}

func CloudSyncAll(c *gin.Context) {
	if adminOnly(c) == nil {
		return
	}
	client, _, err := currentCloudClient()
	if err != nil {
		Fail(c, err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
	defer cancel()
	images, err := listCloudImages(ctx, client)
	if err != nil {
		Fail(c, "读取云端图库失败: "+err.Error())
		return
	}
	keys := make([]string, 0, len(images))
	for _, img := range images {
		keys = append(keys, img.Key)
	}
	success, failed := syncCloudKeys(ctx, client, keys, CurrentUser(c).ID)
	OK(c, gin.H{"success": success, "failed": failed, "total": len(keys)})
}

func CloudDeleteImages(c *gin.Context) {
	if adminOnly(c) == nil {
		return
	}
	client, cfg, err := currentCloudClient()
	if err != nil {
		Fail(c, err.Error())
		return
	}
	var req cloudSyncReq
	if err := c.ShouldBindJSON(&req); err != nil || len(req.Keys) == 0 {
		Fail(c, "请选择要删除的云端图片")
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	success := 0
	var failed []string
	for i, key := range req.Keys {
		if !strings.HasPrefix(key, "images/") {
			continue
		}
		if i > 0 && cfg.Provider == "github" {
			time.Sleep(350 * time.Millisecond) // stay clear of GitHub secondary limits
		}
		if err := client.Delete(ctx, key); err != nil {
			failed = append(failed, key+": "+err.Error())
			continue
		}
		success++
	}
	if len(req.Keys) > 0 {
		global.DB.Model(&models.File{}).Where("cloud_key IN ?", req.Keys).Updates(map[string]interface{}{
			"sync_status": 0, "cloud_key": "", "cloud_url": "", "synced_at": nil,
		})
	}
	OK(c, gin.H{"success": success, "failed": failed})
}

func CloudClearImages(c *gin.Context) {
	if adminOnly(c) == nil {
		return
	}
	client, cfg, err := currentCloudClient()
	if err != nil {
		Fail(c, err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
	defer cancel()
	images, err := listCloudImages(ctx, client)
	if err != nil {
		Fail(c, "读取云端图库失败: "+err.Error())
		return
	}
	success := 0
	for i, img := range images {
		if i > 0 && cfg.Provider == "github" {
			time.Sleep(350 * time.Millisecond)
		}
		if err := client.Delete(ctx, img.Key); err == nil {
			success++
		}
	}
	global.DB.Model(&models.File{}).Where("cloud_key <> ''").Updates(map[string]interface{}{
		"sync_status": 0, "cloud_key": "", "cloud_url": "", "synced_at": nil,
	})
	OK(c, gin.H{"success": success, "total": len(images)})
}

// ---------------------------------------------------------------- cloud backups

type cloudBackup struct {
	Key     string    `json:"key"`
	Name    string    `json:"name"`
	Size    int64     `json:"size"`
	URL     string    `json:"url"`
	Updated time.Time `json:"updatedAt"`
}

func CloudListBackups(c *gin.Context) {
	if adminOnly(c) == nil {
		return
	}
	client, _, err := currentCloudClient()
	if err != nil {
		Fail(c, err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	items, err := client.List(ctx, "backups")
	if err != nil {
		Fail(c, "读取云端备份列表失败: "+err.Error())
		return
	}
	out := make([]cloudBackup, 0, len(items))
	for _, it := range items {
		if it.IsDirectory {
			continue
		}
		out = append(out, cloudBackup{Key: it.Key, Name: it.Name, Size: it.Size, URL: it.URL, Updated: it.UpdatedAt})
	}
	OK(c, out)
}

func CloudUploadBackup(c *gin.Context) {
	if adminOnly(c) == nil {
		return
	}
	var req backupPasswordReq
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Password) == "" {
		Fail(c, "上传云端备份必须设置加密密码")
		return
	}
	if len(req.Password) < 6 {
		Fail(c, "备份密码至少 6 位")
		return
	}
	client, cfg, err := currentCloudClient()
	if err != nil {
		Fail(c, err.Error())
		return
	}
	data, err := buildSiteBackup()
	if err != nil {
		Fail(c, "读取备份数据失败: "+err.Error())
		return
	}
	tmp, err := os.CreateTemp("", "firefly-cloud-backup-*.zip")
	if err != nil {
		Fail(c, "创建临时文件失败")
		return
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	defer os.Remove(tmpPath + ".ffbak")
	if err := writeSiteBackupZip(tmp, data); err != nil {
		tmp.Close()
		Fail(c, "生成备份包失败: "+err.Error())
		return
	}
	tmp.Close()
	if err := encryptFileToFile(tmpPath, tmpPath+".ffbak", req.Password); err != nil {
		Fail(c, err.Error())
		return
	}
	info, _ := os.Stat(tmpPath + ".ffbak")
	if cfg.Provider == "github" && info != nil && info.Size() > 60<<20 {
		Fail(c, "GitHub Contents API 对单文件体积有限制，请改用 S3 / R2 或先导出到本地")
		return
	}
	body, err := os.ReadFile(tmpPath + ".ffbak")
	if err != nil {
		Fail(c, "读取加密备份失败")
		return
	}
	name := "firefly-backup-" + time.Now().Format("20060102-150405") + ".ffbak"
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	key := "backups/" + name
	url, err := client.Upload(ctx, key, body, "application/octet-stream")
	if err != nil {
		Fail(c, "上传云端备份失败: "+err.Error())
		return
	}
	OK(c, gin.H{"key": key, "name": name, "size": len(body), "url": url})
}

type cloudRestoreReq struct {
	Key      string `json:"key"`
	Password string `json:"password"`
}

func CloudRestoreBackup(c *gin.Context) {
	current := adminOnly(c)
	if current == nil {
		return
	}
	var req cloudRestoreReq
	if err := c.ShouldBindJSON(&req); err != nil || req.Key == "" || req.Password == "" {
		Fail(c, "请选择备份并填写解密密码")
		return
	}
	if !strings.HasPrefix(req.Key, "backups/") {
		Fail(c, "备份文件不存在")
		return
	}
	client, _, err := currentCloudClient()
	if err != nil {
		Fail(c, err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	body, err := client.Download(ctx, req.Key)
	if err != nil {
		Fail(c, "下载云端备份失败: "+err.Error())
		return
	}
	tmp, err := os.CreateTemp("", "firefly-cloud-restore-*.ffbak")
	if err != nil {
		Fail(c, "创建临时文件失败")
		return
	}
	encPath := tmp.Name()
	defer os.Remove(encPath)
	if _, err := tmp.Write(body); err != nil {
		tmp.Close()
		Fail(c, "写入临时文件失败")
		return
	}
	tmp.Close()
	zipPath := encPath + ".zip"
	defer os.Remove(zipPath)
	if err := decryptFileToFile(encPath, zipPath, req.Password); err != nil {
		Fail(c, err.Error())
		return
	}
	stats, err := restoreSiteBackupZip(zipPath, current)
	if err != nil {
		Fail(c, "恢复失败: "+err.Error())
		return
	}
	OK(c, stats)
}

func CloudDownloadBackup(c *gin.Context) {
	if adminOnly(c) == nil {
		return
	}
	key := strings.TrimSpace(c.Query("key"))
	if key == "" || !strings.HasPrefix(key, "backups/") {
		Fail(c, "备份文件不存在")
		return
	}
	client, _, err := currentCloudClient()
	if err != nil {
		Fail(c, err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	body, err := client.Download(ctx, key)
	if err != nil {
		Fail(c, "下载备份失败: "+err.Error())
		return
	}
	c.Header("Content-Disposition", "attachment; filename="+path.Base(key))
	c.Header("Content-Type", "application/octet-stream")
	c.Data(200, "application/octet-stream", body)
}

func CloudDeleteBackup(c *gin.Context) {
	if adminOnly(c) == nil {
		return
	}
	var req cloudSyncReq
	if err := c.ShouldBindJSON(&req); err != nil || len(req.Keys) == 0 {
		Fail(c, "请选择要删除的备份")
		return
	}
	client, _, err := currentCloudClient()
	if err != nil {
		Fail(c, err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	for _, key := range req.Keys {
		if !strings.HasPrefix(key, "backups/") {
			continue
		}
		_ = client.Delete(ctx, key)
	}
	OK(c, nil)
}
