package api

import (
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"firefly-panel/database"
	"firefly-panel/global"
	"firefly-panel/models"
)

var (
	linkTagRe = regexp.MustCompile(`(?is)<link\b[^>]*>`)
	attrRe    = regexp.MustCompile(`(?is)([a-z0-9:-]+)\s*=\s*["']([^"']*)["']`)
	// faviconClient deliberately does not follow redirects to exotic schemes.
	faviconClient = &http.Client{Timeout: 10 * time.Second}
)

const (
	maxHTMLBytes = 512 << 10
	maxIconBytes = 3 << 20
)

// FetchFavicon looks up a website icon, downloads it into local storage and
// returns the created file record. Keeping a local copy makes icons work even
// when the target site cannot be reached from the browser.
func FetchFavicon(c *gin.Context) {
	user := CurrentUser(c)
	if user == nil {
		Fail(c, "未登录")
		return
	}
	raw := strings.TrimSpace(c.Query("url"))
	if raw == "" {
		Fail(c, "请填写网站地址")
		return
	}
	pageURL, err := normalizeHTTPURL(raw)
	if err != nil {
		Fail(c, "请输入 http:// 或 https:// 开头的地址")
		return
	}

	iconURL := discoverFavicon(pageURL)
	if iconURL == nil {
		iconURL, _ = url.Parse(pageURL.Scheme + "://" + pageURL.Host + "/favicon.ico")
	}
	if iconURL == nil {
		Fail(c, "未找到网站图标")
		return
	}

	body, contentType, err := downloadLimited(iconURL.String(), maxIconBytes)
	if err != nil {
		Fail(c, "图标下载失败")
		return
	}
	ext := iconExtension(contentType, iconURL.Path)
	fileName := time.Now().Format("20060102150405") + "_" + database.RandomToken(4) + ext
	dstDir := filepath.Join(global.DataDir, "uploads", "icons")
	if err := os.MkdirAll(dstDir, 0o755); err != nil {
		Fail(c, "保存失败")
		return
	}
	dst := filepath.Join(dstDir, fileName)
	if err := os.WriteFile(dst, body, 0o644); err != nil {
		Fail(c, "保存失败")
		return
	}

	record := models.File{
		UserID:   user.ID,
		FileName: "favicon" + ext,
		Src:      "/uploads/icons/" + fileName,
		FileType: 2,
		Size:     int64(len(body)),
	}
	global.DB.Create(&record)
	OK(c, record)
}

func normalizeHTTPURL(raw string) (*url.URL, error) {
	if !strings.HasPrefix(strings.ToLower(raw), "http://") && !strings.HasPrefix(strings.ToLower(raw), "https://") {
		raw = "https://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return nil, err
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, os.ErrInvalid
	}
	return u, nil
}

// discoverFavicon fetches the page and parses <link rel="...icon..."> tags.
func discoverFavicon(pageURL *url.URL) *url.URL {
	body, _, err := downloadLimited(pageURL.String(), maxHTMLBytes)
	if err != nil {
		return nil
	}
	html := string(body)
	best := ""
	for _, tag := range linkTagRe.FindAllString(html, -1) {
		attrs := map[string]string{}
		for _, m := range attrRe.FindAllStringSubmatch(tag, -1) {
			attrs[strings.ToLower(m[1])] = m[2]
		}
		rel := strings.ToLower(attrs["rel"])
		href := strings.TrimSpace(attrs["href"])
		if href == "" || !strings.Contains(rel, "icon") || strings.Contains(rel, "mask-icon") {
			continue
		}
		if strings.HasPrefix(href, "data:") {
			continue
		}
		if best == "" || strings.Contains(rel, "shortcut") || strings.Contains(rel, "apple-touch") {
			best = href
		}
	}
	if best == "" {
		return nil
	}
	ref, err := url.Parse(best)
	if err != nil {
		return nil
	}
	abs := pageURL.ResolveReference(ref)
	if abs.Scheme != "http" && abs.Scheme != "https" {
		return nil
	}
	return abs
}

func downloadLimited(rawURL string, limit int64) ([]byte, string, error) {
	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,image/*,*/*")
	resp, err := faviconClient.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, "", os.ErrNotExist
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, limit))
	if err != nil {
		return nil, "", err
	}
	if len(body) == 0 {
		return nil, "", os.ErrNotExist
	}
	return body, resp.Header.Get("Content-Type"), nil
}

func iconExtension(contentType, iconPath string) string {
	ct := strings.ToLower(contentType)
	switch {
	case strings.Contains(ct, "svg"):
		return ".svg"
	case strings.Contains(ct, "png"):
		return ".png"
	case strings.Contains(ct, "jpeg"), strings.Contains(ct, "jpg"):
		return ".jpg"
	case strings.Contains(ct, "gif"):
		return ".gif"
	case strings.Contains(ct, "webp"):
		return ".webp"
	case strings.Contains(ct, "avif"):
		return ".avif"
	case strings.Contains(ct, "icon"):
		return ".ico"
	}
	ext := strings.ToLower(path.Ext(iconPath))
	if allowedImageExt[ext] {
		return ext
	}
	return ".png"
}
