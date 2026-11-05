package router

import (
	"embed"
	"io/fs"
	"net/http"
	"path"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"

	"firefly-panel/api"
	"firefly-panel/global"
)

//go:embed all:webdist
var webDist embed.FS

func Init() *gin.Engine {
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery(), Cors())
	r.MaxMultipartMemory = 32 << 20

	r.Static("/uploads", filepath.Join(global.DataDir, "uploads"))

	v1 := r.Group("/api/v1")
	{
		v1.POST("/auth/login", api.Login)
		v1.GET("/panel/public", api.PanelPublic)
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

			auth.GET("/system/file/list", api.ListFiles)
			auth.POST("/system/file/upload", api.UploadFile)
			auth.DELETE("/system/file/:id", api.DeleteFile)

			auth.GET("/system/setting", api.GetSettings)
			auth.POST("/system/setting/save", api.SaveSettings)

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
