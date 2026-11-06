package router

import (
	"embed"
	"io/fs"
	"log"
	"mime"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"

	"firefly-atlas/api"
	"firefly-atlas/global"
)

//go:embed all:webdist
var webDist embed.FS

// trustedProxies reads the optional TRUSTED_PROXIES environment variable
// (comma separated CIDRs or IPs). Returning nil tells Gin to ignore
// X-Forwarded-For entirely.
func trustedProxies() []string {
	raw := strings.TrimSpace(os.Getenv("TRUSTED_PROXIES"))
	if raw == "" {
		return nil
	}
	out := make([]string, 0, 4)
	for _, part := range strings.Split(raw, ",") {
		if part = strings.TrimSpace(part); part != "" {
			out = append(out, part)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func Init() *gin.Engine {
	_ = mime.AddExtensionType(".webmanifest", "application/manifest+json")
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery(), Cors())
	r.MaxMultipartMemory = 32 << 20

	// Gin trusts every proxy by default, which lets any client spoof
	// X-Forwarded-For and escape the login rate limiter (it keys on ClientIP).
	// Only the deployments that list their reverse proxy in TRUSTED_PROXIES get
	// forwarded headers honoured; everything else uses the socket address.
	if err := r.SetTrustedProxies(trustedProxies()); err != nil {
		log.Printf("[firefly-atlas] TRUSTED_PROXIES 无效，已改为不信任任何代理: %v", err)
		_ = r.SetTrustedProxies(nil)
	}

	r.Static("/uploads", filepath.Join(global.DataDir, "uploads"))

	v1 := r.Group("/api/v1")
	{
		v1.POST("/auth/login", api.Login)
		v1.GET("/panel/public", api.PanelPublic)
		v1.GET("/panel/site", api.SiteInfo)
		v1.GET("/panel/monitor", api.MonitorPublic)
		v1.GET("/panel/cloud/image", api.CloudImageProxy)
		v1.GET("/health", func(c *gin.Context) { c.JSON(200, gin.H{"status": "ok", "version": global.Version}) })

		auth := v1.Group("")
		auth.Use(LoginInterceptor())
		{
			auth.GET("/auth/info", api.Info)
			auth.POST("/auth/logout", api.Logout)
			auth.POST("/auth/password", api.ChangePassword)

			auth.GET("/panel/index", api.PanelIndex)
			auth.POST("/panel/config/save", api.SaveUserConfig)

			auth.POST("/panel/itemIcon/save", api.SaveItemIcon)
			auth.DELETE("/panel/itemIcon/:id", api.DeleteItemIcon)
			auth.POST("/panel/itemIcon/sort", api.SortItemIcons)

			auth.POST("/panel/itemIconGroup/save", api.SaveItemIconGroup)
			auth.DELETE("/panel/itemIconGroup/:id", api.DeleteItemIconGroup)
			auth.POST("/panel/itemIconGroup/sort", api.SortItemIconGroups)

			auth.GET("/panel/backup/export", api.ExportBackup)
			auth.POST("/panel/backup/import", api.ImportBackup)
			auth.POST("/panel/import/bookmarks", api.ImportBookmarks)
			auth.POST("/panel/health", api.CheckHealth)

			auth.GET("/system/file/list", api.ListFiles)
			auth.POST("/system/file/upload", api.UploadFile)
			auth.DELETE("/system/file/:id", api.DeleteFile)
			auth.POST("/system/file/batch-delete", api.BatchDeleteFiles)
			auth.POST("/system/file/move", api.MoveFiles)
			auth.POST("/system/file/rename", api.RenameFile)
			auth.POST("/system/file/apply-sync", api.ApplySyncFiles)
			auth.POST("/system/file/upload-cloud", api.UploadFilesToCloud)
			auth.GET("/system/file/audit", api.AuditList)
			auth.POST("/system/file/audit", api.AuditFiles)
			auth.GET("/system/file/public-icons", api.PublicIcons)

			auth.POST("/system/backup/export", api.ExportSiteBackup)
			auth.POST("/system/backup/import", api.ImportSiteBackup)

			auth.GET("/system/cloud/config", api.GetCloudConfig)
			auth.POST("/system/cloud/config", api.SaveCloudConfig)
			auth.POST("/system/cloud/test", api.TestCloudConfig)
			auth.GET("/system/cloud/images", api.CloudListImages)
			auth.POST("/system/cloud/sync", api.CloudSyncImages)
			auth.POST("/system/cloud/sync-all", api.CloudSyncAll)
			auth.POST("/system/cloud/delete", api.CloudDeleteImages)
			auth.POST("/system/cloud/clear", api.CloudClearImages)
			auth.GET("/system/cloud/backups", api.CloudListBackups)
			auth.POST("/system/cloud/backup", api.CloudUploadBackup)
			auth.POST("/system/cloud/backup/restore", api.CloudRestoreBackup)
			auth.GET("/system/cloud/backup/download", api.CloudDownloadBackup)
			auth.POST("/system/cloud/backup/delete", api.CloudDeleteBackup)

			auth.GET("/system/setting", api.GetSettings)
			auth.POST("/system/setting/save", api.SaveSettings)

			auth.GET("/system/monitor", api.MonitorAuth)
			auth.GET("/system/favicon", api.FetchFavicon)

			auth.GET("/system/user/list", api.ListUsers)
			auth.POST("/system/user/save", api.SaveUser)
			auth.DELETE("/system/user/:id", api.DeleteUser)
		}
	}

	registerSPA(r)
	return r
}

func registerSPA(r *gin.Engine) {
	sub, err := fs.Sub(webDist, "webdist")
	if err != nil {
		return
	}
	index, _ := fs.ReadFile(sub, "index.html")
	fileServer := http.FileServer(http.FS(sub))

	r.NoRoute(func(c *gin.Context) {
		p := c.Request.URL.Path
		if strings.HasPrefix(p, "/api/") {
			c.JSON(http.StatusNotFound, gin.H{"code": -1, "msg": "接口不存在"})
			return
		}
		clean := strings.TrimPrefix(path.Clean(p), "/")
		if clean != "" && clean != "." {
			if f, err := sub.Open(clean); err == nil {
				_ = f.Close()
				if strings.HasPrefix(clean, "assets/") {
					c.Header("Cache-Control", "public, max-age=31536000, immutable")
				}
				fileServer.ServeHTTP(c.Writer, c.Request)
				return
			}
		}
		c.Header("Cache-Control", "no-cache")
		c.Data(http.StatusOK, "text/html; charset=utf-8", index)
	})
}
