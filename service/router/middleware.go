package router

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"

	"firefly-atlas/global"
	"firefly-atlas/models"
)

func LoginInterceptor() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := strings.TrimSpace(c.GetHeader("token"))
		if token == "" {
			auth := c.GetHeader("Authorization")
			if strings.HasPrefix(strings.ToLower(auth), "bearer ") {
				token = strings.TrimSpace(auth[7:])
			}
		}
		if token == "" {
			token = c.Query("token")
		}
		if token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": -1, "data": nil, "msg": "未登录或登录已过期"})
			return
		}

		var user models.User
		if err := global.DB.Where("token = ? AND status = 1", token).First(&user).Error; err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": -1, "data": nil, "msg": "未登录或登录已过期"})
			return
		}
		c.Set("userInfo", &user)
		c.Next()
	}
}

// sameOriginHost reports whether the request Origin points back at the host that
// served it. Reverse proxies usually forward the original Host header, so this
// matches the normal single-node deployment as well.
func sameOriginHost(origin, host string) bool {
	u, err := url.Parse(origin)
	if err != nil || u.Host == "" {
		return false
	}
	return strings.EqualFold(u.Host, host)
}

// Cors only echoes an Origin that belongs to the panel itself. Reflecting every
// Origin together with Allow-Credentials would let any third-party website read
// authenticated API responses in a visitor's browser.
func Cors() gin.HandlerFunc {
	return func(c *gin.Context) {
		if origin := c.GetHeader("Origin"); origin != "" {
			c.Header("Vary", "Origin")
			if sameOriginHost(origin, c.Request.Host) {
				c.Header("Access-Control-Allow-Origin", origin)
				c.Header("Access-Control-Allow-Credentials", "true")
			}
		}
		c.Header("Access-Control-Allow-Headers", "Content-Type, token, Authorization")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}
