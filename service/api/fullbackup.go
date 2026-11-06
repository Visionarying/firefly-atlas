package api

import (
	"archive/zip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"firefly-atlas/database"
	"firefly-atlas/global"
	"firefly-atlas/models"
)

const maxBackupUncompressed = int64(2 << 30) // 2 GiB

type backupUser struct {
	ID        uint      `json:"id"`
	Username  string    `json:"username"`
	Password  string    `json:"password"`
	Nickname  string    `json:"nickname"`
	Role      int       `json:"role"`
	Status    int       `json:"status"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type backupUserConfig struct {
	UserID           uint            `json:"userId"`
	PanelJSON        json.RawMessage `json:"panelJson"`
	SearchEngineJSON json.RawMessage `json:"searchEngineJson"`
	CreatedAt        time.Time       `json:"createdAt"`
	UpdatedAt        time.Time       `json:"updatedAt"`
}

type siteBackup struct {
	Version     string                 `json:"version"`
	ExportedAt  string                 `json:"exportedAt"`
	Users       []backupUser           `json:"users"`
	Pages       []models.ItemIconPage  `json:"pages"`
	Groups      []models.ItemIconGroup `json:"groups"`
	Items       []backupItem           `json:"items"`
	UserConfigs []backupUserConfig     `json:"userConfigs"`
	Settings    []models.SystemSetting `json:"settings"`
	Files       []models.File          `json:"files"`
}

func adminOnly(c *gin.Context) *models.User {
	user := CurrentUser(c)
	if user == nil {
		Fail(c, "未登录")
		return nil
	}
	if user.Role != 1 {
		FailCode(c, 403, "仅管理员可执行此操作")
		return nil
	}
	return user
}

func buildSiteBackup() (*siteBackup, error) {
	data := &siteBackup{
		Version:     global.Version,
		ExportedAt:  time.Now().Format(time.RFC3339),
		Users:       []backupUser{},
		Pages:       []models.ItemIconPage{},
		Groups:      []models.ItemIconGroup{},
		Items:       []backupItem{},
		UserConfigs: []backupUserConfig{},
		Settings:    []models.SystemSetting{},
		Files:       []models.File{},
	}

	var users []models.User
	if err := global.DB.Order("id asc").Find(&users).Error; err != nil {
		return nil, err
	}
	for _, u := range users {
		data.Users = append(data.Users, backupUser{
			ID: u.ID, Username: u.Username, Password: u.Password, Nickname: u.Nickname,
			Role: u.Role, Status: u.Status, CreatedAt: u.CreatedAt, UpdatedAt: u.UpdatedAt,
		})
	}

	if err := global.DB.Order("sort asc, id asc").Find(&data.Pages).Error; err != nil {
		return nil, err
	}
	if err := global.DB.Order("page_id asc, sort asc, id asc").Find(&data.Groups).Error; err != nil {
		return nil, err
	}
	var items []models.ItemIcon
	if err := global.DB.Order("sort asc, id asc").Find(&items).Error; err != nil {
		return nil, err
	}
	for _, it := range items {
		bi := backupItem{ItemIcon: it}
		if it.IconJSON != "" {
			_ = json.Unmarshal([]byte(it.IconJSON), &bi.Icon)
		}
		data.Items = append(data.Items, bi)
	}

	var configs []models.UserConfig
	if err := global.DB.Order("id asc").Find(&configs).Error; err != nil {
		return nil, err
	}
	for _, cfg := range configs {
		data.UserConfigs = append(data.UserConfigs, backupUserConfig{
			UserID: cfg.UserID, PanelJSON: json.RawMessage(cfg.PanelJSON),
			SearchEngineJSON: json.RawMessage(cfg.SearchEngineJSON),
			CreatedAt:        cfg.CreatedAt, UpdatedAt: cfg.UpdatedAt,
		})
	}

	if err := global.DB.Order("id asc").Find(&data.Settings).Error; err != nil {
		return nil, err
	}
	if err := global.DB.Order("id asc").Find(&data.Files).Error; err != nil {
		return nil, err
	}
	return data, nil
}

func writeSiteBackupZip(w io.Writer, data *siteBackup) error {
	zw := zip.NewWriter(w)
	manifest, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		_ = zw.Close()
		return err
	}
	f, err := zw.Create("backup.json")
	if err != nil {
		_ = zw.Close()
		return err
	}
	if _, err := f.Write(manifest); err != nil {
		_ = zw.Close()
		return err
	}

	uploads := filepath.Join(global.DataDir, "uploads")
	walkErr := filepath.Walk(uploads, func(p string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			if os.IsNotExist(walkErr) {
				return nil
			}
			return walkErr
		}
		if info == nil || info.IsDir() {
			return nil
		}
		rel, relErr := filepath.Rel(uploads, p)
		if relErr != nil {
			return relErr
		}
		name := "uploads/" + filepath.ToSlash(rel)
		entry, createErr := zw.Create(name)
		if createErr != nil {
			return createErr
		}
		src, openErr := os.Open(p)
		if openErr != nil {
			return openErr
		}
		_, copyErr := io.Copy(entry, src)
		_ = src.Close()
		return copyErr
	})
	if walkErr != nil && !os.IsNotExist(walkErr) {
		_ = zw.Close()
		return walkErr
	}
	return zw.Close()
}

type backupPasswordReq struct {
	Password string `json:"password"`
}

// ExportSiteBackup streams a full ZIP backup. When a password is supplied the
// ZIP is wrapped in the chunked AES-256-GCM container.
func ExportSiteBackup(c *gin.Context) {
	if adminOnly(c) == nil {
		return
	}
	var req backupPasswordReq
	_ = c.ShouldBindJSON(&req)

	data, err := buildSiteBackup()
	if err != nil {
		Fail(c, "读取备份数据失败: "+err.Error())
		return
	}

	tmp, err := os.CreateTemp("", "firefly-atlas-backup-*.zip")
	if err != nil {
		Fail(c, "创建临时文件失败")
		return
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	defer tmp.Close()

	if err := writeSiteBackupZip(tmp, data); err != nil {
		Fail(c, "生成备份包失败: "+err.Error())
		return
	}
	if err := tmp.Close(); err != nil {
		Fail(c, "写入备份包失败")
		return
	}

	name := "firefly-atlas-backup-" + time.Now().Format("20060102-150405")
	if strings.TrimSpace(req.Password) != "" {
		encPath := tmpPath + ".ffbak"
		defer os.Remove(encPath)
		if err := encryptFileToFile(tmpPath, encPath, req.Password); err != nil {
			Fail(c, err.Error())
			return
		}
		c.FileAttachment(encPath, name+".ffbak")
		return
	}
	c.FileAttachment(tmpPath, name+".zip")
}

func ImportSiteBackup(c *gin.Context) {
	if adminOnly(c) == nil {
		return
	}
	file, err := c.FormFile("file")
	if err != nil {
		Fail(c, "请选择备份文件")
		return
	}
	if file.Size > 2<<30 {
		Fail(c, "备份文件不能超过 2GB")
		return
	}
	password := strings.TrimSpace(c.PostForm("password"))

	src, err := file.Open()
	if err != nil {
		Fail(c, "读取备份文件失败")
		return
	}
	defer src.Close()
	tmp, err := os.CreateTemp("", "firefly-restore-*")
	if err != nil {
		Fail(c, "创建临时文件失败")
		return
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if _, err := io.Copy(tmp, src); err != nil {
		tmp.Close()
		Fail(c, "保存备份文件失败")
		return
	}
	if err := tmp.Close(); err != nil {
		Fail(c, "保存备份文件失败")
		return
	}

	zipPath := tmpPath
	encrypted, _ := IsEncryptedBackup(tmpPath)
	if encrypted {
		if password == "" {
			Fail(c, "这是加密备份，请输入备份密码")
			return
		}
		zipPath = tmpPath + ".zip"
		defer os.Remove(zipPath)
		if err := decryptFileToFile(tmpPath, zipPath, password); err != nil {
			Fail(c, err.Error())
			return
		}
	} else if password != "" {
		// A password may be supplied for a plain ZIP; ignore it rather than fail.
		password = ""
	}

	stats, err := restoreSiteBackupZip(zipPath, CurrentUser(c))
	if err != nil {
		Fail(c, "恢复失败: "+err.Error())
		return
	}
	OK(c, stats)
}

type restoreStats struct {
	Users   int `json:"users"`
	Pages   int `json:"pages"`
	Groups  int `json:"groups"`
	Items   int `json:"items"`
	Files   int `json:"files"`
	Uploads int `json:"uploads"`
}

func restoreSiteBackupZip(zipPath string, current *models.User) (*restoreStats, error) {
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, errors.New("无法解压备份文件，文件可能已损坏")
	}
	defer zr.Close()

	if len(zr.File) > 100000 {
		return nil, errors.New("备份包文件数量异常")
	}
	var total int64
	for _, f := range zr.File {
		total += int64(f.UncompressedSize64)
		if total > maxBackupUncompressed {
			return nil, errors.New("备份包解压后超过 2GB 限制")
		}
	}

	stagingRoot := filepath.Join(global.DataDir, ".restore-"+database.RandomToken(6))
	stagingUploads := filepath.Join(stagingRoot, "uploads")
	defer os.RemoveAll(stagingRoot)

	var data *siteBackup
	uploads := 0
	remaining := maxBackupUncompressed
	for _, f := range zr.File {
		name := strings.ReplaceAll(f.Name, "\\", "/")
		name = strings.TrimPrefix(name, "./")
		clean := path.Clean(name)
		if clean == "backup.json" {
			rc, err := f.Open()
			if err != nil {
				return nil, err
			}
			body, err := io.ReadAll(io.LimitReader(rc, 64<<20))
			rc.Close()
			if err != nil {
				return nil, err
			}
			var parsed siteBackup
			if err := json.Unmarshal(body, &parsed); err != nil {
				return nil, fmt.Errorf("解析备份描述失败: %w", err)
			}
			data = &parsed
			continue
		}
		if !strings.HasPrefix(clean, "uploads/") {
			continue
		}
		rel := strings.TrimPrefix(clean, "uploads/")
		if rel == "" || rel == "." || strings.HasPrefix(rel, "../") || strings.Contains(rel, "/../") {
			continue
		}
		if f.FileInfo().IsDir() || f.FileInfo().Mode()&os.ModeSymlink != 0 {
			continue
		}
		if int64(f.UncompressedSize64) > remaining {
			return nil, errors.New("备份包解压后超过 2GB 限制")
		}
		dest := filepath.Join(stagingUploads, filepath.FromSlash(rel))
		relCheck, err := filepath.Rel(stagingUploads, dest)
		if err != nil || strings.HasPrefix(relCheck, "..") {
			continue
		}
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return nil, err
		}
		rc, err := f.Open()
		if err != nil {
			return nil, err
		}
		out, err := os.OpenFile(dest, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
		if err != nil {
			rc.Close()
			return nil, err
		}
		written, copyErr := io.Copy(out, io.LimitReader(rc, remaining+1))
		out.Close()
		rc.Close()
		if copyErr != nil {
			return nil, copyErr
		}
		if written > remaining {
			return nil, errors.New("备份包解压后超过 2GB 限制")
		}
		remaining -= written
		uploads++
	}
	if data == nil {
		return nil, errors.New("备份包中未找到 backup.json")
	}

	stats, err := applySiteRestore(data, current, stagingUploads)
	if err != nil {
		return nil, err
	}
	stats.Uploads = uploads
	return stats, nil
}

func applySiteRestore(data *siteBackup, current *models.User, stagingUploads string) (*restoreStats, error) {
	if len(data.Users) == 0 && len(data.Pages) == 0 && len(data.Groups) == 0 && len(data.Items) == 0 && len(data.Settings) == 0 {
		return nil, errors.New("备份内容为空，已停止还原")
	}
	stats := &restoreStats{}
	tx := global.DB.Begin()
	if tx.Error != nil {
		return nil, tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Where("1 = 1").Delete(&models.ItemIcon{}).Error; err != nil {
		tx.Rollback()
		return nil, err
	}
	if err := tx.Where("1 = 1").Delete(&models.ItemIconGroup{}).Error; err != nil {
		tx.Rollback()
		return nil, err
	}
	if err := tx.Where("1 = 1").Delete(&models.ItemIconPage{}).Error; err != nil {
		tx.Rollback()
		return nil, err
	}
	if err := tx.Where("1 = 1").Delete(&models.UserConfig{}).Error; err != nil {
		tx.Rollback()
		return nil, err
	}
	if err := tx.Where("1 = 1").Delete(&models.File{}).Error; err != nil {
		tx.Rollback()
		return nil, err
	}
	if err := tx.Where("1 = 1").Delete(&models.SystemSetting{}).Error; err != nil {
		tx.Rollback()
		return nil, err
	}
	if err := tx.Where("id <> ?", current.ID).Delete(&models.User{}).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	userMap := map[uint]uint{}
	for _, u := range data.Users {
		if strings.EqualFold(u.Username, current.Username) {
			updates := map[string]interface{}{
				"nickname": u.Nickname,
				"role":     current.Role,
				"status":   1,
			}
			if err := tx.Model(&models.User{}).Where("id = ?", current.ID).Updates(updates).Error; err != nil {
				tx.Rollback()
				return nil, err
			}
			userMap[u.ID] = current.ID
			stats.Users++
			continue
		}
		role := u.Role
		if role != 1 {
			role = 2
		}
		user := models.User{
			Username: u.Username, Password: u.Password, Nickname: u.Nickname,
			Role: role, Status: u.Status, Token: database.RandomToken(32),
			CreatedAt: u.CreatedAt, UpdatedAt: u.UpdatedAt,
		}
		if user.Password == "" {
			continue
		}
		if err := tx.Create(&user).Error; err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("恢复用户 %s 失败: %w", u.Username, err)
		}
		userMap[u.ID] = user.ID
		stats.Users++
	}
	if _, ok := userMap[0]; !ok {
		userMap[0] = current.ID
	}

	pageMap := map[uint]uint{}
	defaultPages := map[uint]uint{}
	for _, p := range data.Pages {
		uid, ok := userMap[p.UserID]
		if !ok {
			uid = current.ID
		}
		columns := p.Columns
		if columns < 1 || columns > 3 {
			columns = 1
		}
		page := models.ItemIconPage{
			UserID: uid, Title: p.Title, Sort: p.Sort, Columns: columns,
			CreatedAt: p.CreatedAt, UpdatedAt: p.UpdatedAt,
		}
		if err := tx.Create(&page).Error; err != nil {
			tx.Rollback()
			return nil, err
		}
		pageMap[p.ID] = page.ID
		if defaultPages[uid] == 0 {
			defaultPages[uid] = page.ID
		}
		stats.Pages++
	}

	groupMap := map[uint]uint{}
	for _, g := range data.Groups {
		uid, ok := userMap[g.UserID]
		if !ok {
			uid = current.ID
		}
		pageID := pageMap[g.PageID]
		if pageID == 0 {
			pageID = defaultPages[uid]
			if pageID == 0 {
				page := models.ItemIconPage{UserID: uid, Title: "我的主页", Columns: 1}
				if err := tx.Create(&page).Error; err != nil {
					tx.Rollback()
					return nil, err
				}
				pageID = page.ID
				defaultPages[uid] = pageID
				stats.Pages++
			}
		}
		group := models.ItemIconGroup{
			UserID: uid, PageID: pageID, Title: g.Title, Sort: g.Sort,
			CreatedAt: g.CreatedAt, UpdatedAt: g.UpdatedAt,
		}
		if err := tx.Create(&group).Error; err != nil {
			tx.Rollback()
			return nil, err
		}
		groupMap[g.ID] = group.ID
		stats.Groups++
	}

	for _, it := range data.Items {
		uid, ok := userMap[it.UserID]
		if !ok {
			uid = current.ID
		}
		gid, ok := groupMap[it.ItemIconGroupID]
		if !ok {
			continue
		}
		iconJSON := it.IconJSON
		if iconJSON == "" {
			b, _ := json.Marshal(it.Icon)
			iconJSON = string(b)
		}
		item := models.ItemIcon{
			UserID: uid, ItemIconGroupID: gid, Title: it.Title, Url: it.Url,
			LanUrl: it.LanUrl, Description: it.Description, OpenMethod: it.OpenMethod,
			HealthCheck: it.HealthCheck, IconJSON: iconJSON, Sort: it.Sort,
			CreatedAt: it.CreatedAt, UpdatedAt: it.UpdatedAt,
		}
		if err := tx.Create(&item).Error; err != nil {
			tx.Rollback()
			return nil, err
		}
		stats.Items++
	}

	for _, cfg := range data.UserConfigs {
		uid, ok := userMap[cfg.UserID]
		if !ok {
			continue
		}
		panelJSON := string(cfg.PanelJSON)
		searchJSON := string(cfg.SearchEngineJSON)
		if panelJSON == "" {
			panelJSON = models.DefaultPanelConfig
		}
		if searchJSON == "" {
			searchJSON = models.DefaultSearchEngines
		}
		uc := models.UserConfig{
			UserID: uid, PanelJSON: panelJSON, SearchEngineJSON: searchJSON,
			CreatedAt: cfg.CreatedAt, UpdatedAt: cfg.UpdatedAt,
		}
		if err := tx.Create(&uc).Error; err != nil {
			tx.Rollback()
			return nil, err
		}
	}

	for _, s := range data.Settings {
		if s.ConfigName == "cloud_storage_config" {
			continue
		}
		setting := models.SystemSetting{ConfigName: s.ConfigName, Title: s.Title, Content: s.Content}
		if err := tx.Create(&setting).Error; err != nil {
			tx.Rollback()
			return nil, err
		}
	}

	for _, f := range data.Files {
		uid, ok := userMap[f.UserID]
		if !ok {
			uid = current.ID
		}
		if !strings.HasPrefix(f.Src, "/uploads/") || strings.Contains(f.Src, "..") {
			continue
		}
		file := models.File{
			UserID: uid, FileName: f.FileName, Src: f.Src, Thumb: f.Thumb, FileType: f.FileType,
			Size: f.Size, SyncStatus: f.SyncStatus, AuditRemark: f.AuditRemark,
			CloudKey: f.CloudKey, CloudURL: f.CloudURL, SyncedAt: f.SyncedAt,
			CreatedAt: f.CreatedAt, UpdatedAt: f.UpdatedAt,
		}
		if err := tx.Create(&file).Error; err != nil {
			tx.Rollback()
			return nil, err
		}
		stats.Files++
	}

	if dirHasFiles(stagingUploads) {
		if err := swapUploads(stagingUploads); err != nil {
			tx.Rollback()
			return nil, err
		}
	}

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}
	reloadPublicMode()
	return stats, nil
}

func dirHasFiles(dir string) bool {
	found := false
	_ = filepath.Walk(dir, func(_ string, info os.FileInfo, err error) error {
		if err == nil && info != nil && !info.IsDir() {
			found = true
			return io.EOF
		}
		return nil
	})
	return found
}

func swapUploads(staging string) error {
	target := filepath.Join(global.DataDir, "uploads")
	old := target + ".old-" + database.RandomToken(4)
	hadOld := false
	if _, err := os.Stat(target); err == nil {
		if err := os.Rename(target, old); err != nil {
			return fmt.Errorf("替换上传目录失败: %w", err)
		}
		hadOld = true
	}
	if err := os.Rename(staging, target); err != nil {
		if hadOld {
			_ = os.Rename(old, target)
		}
		return fmt.Errorf("写入新上传目录失败: %w", err)
	}
	if hadOld {
		_ = os.RemoveAll(old)
	}
	return nil
}

func reloadPublicMode() {
	var s models.SystemSetting
	if err := global.DB.Where("config_name = ?", "public_visit_mode").First(&s).Error; err == nil {
		global.PublicVisitMode = s.Content == "1"
	} else {
		global.PublicVisitMode = false
	}
}
