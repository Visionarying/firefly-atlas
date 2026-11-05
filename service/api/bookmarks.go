package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
	"golang.org/x/net/html"

	"firefly-panel/global"
	"firefly-panel/models"
)

type bookmark struct {
	Folder string
	Title  string
	URL    string
}

func normalizeBookmarkURL(raw string) (string, bool) {
	raw = strings.TrimSpace(raw)
	lower := strings.ToLower(raw)
	if !strings.HasPrefix(lower, "http://") && !strings.HasPrefix(lower, "https://") {
		return "", false
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return "", false
	}
	return raw, true
}

func parseBookmarkHTML(r io.Reader) ([]bookmark, error) {
	z := html.NewTokenizer(r)
	var out []bookmark
	var folderStack []string
	pendingFolder := ""
	inH3 := false
	inA := false
	var h3Text, aText strings.Builder
	aHref := ""

	for {
		tt := z.Next()
		if tt == html.ErrorToken {
			if z.Err() == io.EOF {
				break
			}
			return out, z.Err()
		}
		switch tt {
		case html.StartTagToken:
			name, hasAttr := z.TagName()
			switch string(name) {
			case "h3":
				inH3 = true
				h3Text.Reset()
			case "dl":
				folderStack = append(folderStack, pendingFolder)
				pendingFolder = ""
			case "a":
				inA = true
				aText.Reset()
				aHref = ""
				if hasAttr {
					for {
						key, val, more := z.TagAttr()
						if string(key) == "href" {
							aHref = string(val)
						}
						if !more {
							break
						}
					}
				}
			}
		case html.TextToken:
			if inH3 {
				h3Text.Write(z.Text())
			} else if inA {
				aText.Write(z.Text())
			}
		case html.EndTagToken:
			name, _ := z.TagName()
			switch string(name) {
			case "h3":
				inH3 = false
				pendingFolder = strings.TrimSpace(h3Text.String())
			case "dl":
				if len(folderStack) > 0 {
					folderStack = folderStack[:len(folderStack)-1]
				}
			case "a":
				if inA {
					inA = false
					href, ok := normalizeBookmarkURL(aHref)
					if ok {
						title := strings.TrimSpace(aText.String())
						if title == "" {
							if u, err := url.Parse(href); err == nil {
								title = u.Host
							}
						}
						folder := ""
						var parts []string
						for _, s := range folderStack {
							if strings.TrimSpace(s) != "" {
								parts = append(parts, strings.TrimSpace(s))
							}
						}
						if len(parts) > 0 {
							folder = strings.Join(parts, " / ")
						}
						out = append(out, bookmark{Folder: folder, Title: title, URL: href})
						if len(out) >= 3000 {
							return out, nil
						}
					}
				}
			}
		}
	}
	return out, nil
}

var bookmarkGradients = []string{
	"linear-gradient(135deg,#5b8cff,#3a5cff)",
	"linear-gradient(135deg,#22d3ee,#0ea5e9)",
	"linear-gradient(135deg,#34d399,#059669)",
	"linear-gradient(135deg,#fbbf24,#f59e0b)",
	"linear-gradient(135deg,#fb7185,#e11d48)",
	"linear-gradient(135deg,#a78bfa,#7c3aed)",
}

func ImportBookmarks(c *gin.Context) {
	user := CurrentUser(c)
	if user == nil {
		Fail(c, "未登录")
		return
	}
	file, err := c.FormFile("file")
	if err != nil {
		Fail(c, "请选择浏览器导出的书签 HTML 文件")
		return
	}
	if file.Size > 8<<20 {
		Fail(c, "书签文件不能超过 8MB")
		return
	}
	f, err := file.Open()
	if err != nil {
		Fail(c, "读取书签文件失败")
		return
	}
	defer f.Close()
	bookmarks, err := parseBookmarkHTML(f)
	if err != nil {
		Fail(c, "解析书签文件失败: "+err.Error())
		return
	}
	if len(bookmarks) == 0 {
		Fail(c, "没有在文件中找到 http / https 书签")
		return
	}

	mode := c.PostForm("mode")
	if mode == "" {
		mode = "folders"
	}
	groupId := uint(0)
	_, _ = fmt.Sscanf(c.PostForm("groupId"), "%d", &groupId)

	// Existing URLs are used for cheap duplicate avoidance.
	tx := global.DB.Begin()
	if tx.Error != nil {
		Fail(c, "开始导入事务失败")
		return
	}
	committed := false
	defer func() {
		if !committed {
			tx.Rollback()
		}
	}()

	var existing []string
	tx.Model(&models.ItemIcon{}).Where("user_id = ?", user.ID).Pluck("url", &existing)
	existingSet := map[string]bool{}
	for _, u := range existing {
		existingSet[strings.ToLower(strings.TrimSpace(u))] = true
	}

	var maxGroupSort int
	tx.Model(&models.ItemIconGroup{}).Where("user_id = ?", user.ID).
		Select("COALESCE(MAX(sort), -1)").Scan(&maxGroupSort)

	groups := map[string][]bookmark{}
	if mode == "single" || groupId > 0 {
		groups["导入的书签"] = bookmarks
	} else {
		for _, b := range bookmarks {
			name := b.Folder
			if name == "" {
				name = "导入的书签"
			}
			if len([]rune(name)) > 60 {
				name = string([]rune(name)[:60])
			}
			groups[name] = append(groups[name], b)
		}
	}

	createdGroups := 0
	createdItems := 0
	skipped := 0
	for name, list := range groups {
		if createdGroups >= 200 {
			break
		}
		var group models.ItemIconGroup
		if groupId > 0 {
			if err := tx.Where("id = ? AND user_id = ?", groupId, user.ID).First(&group).Error; err != nil {
				Fail(c, "目标分组不存在")
				return
			}
		} else {
			maxGroupSort++
			group = models.ItemIconGroup{UserID: user.ID, Title: name, Sort: maxGroupSort}
			if err := tx.Create(&group).Error; err != nil {
				Fail(c, "创建分组失败")
				return
			}
			createdGroups++
		}

		var maxSort int
		tx.Model(&models.ItemIcon{}).
			Where("user_id = ? AND item_icon_group_id = ?", user.ID, group.ID).
			Select("COALESCE(MAX(sort), -1)").Scan(&maxSort)
		for _, b := range list {
			if createdItems >= 2000 {
				break
			}
			if existingSet[strings.ToLower(b.URL)] {
				skipped++
				continue
			}
			icon := models.ItemIconIcon{
				ItemType:        2,
				Text:            firstRuneTitle(b.Title),
				BackgroundColor: bookmarkGradients[createdItems%len(bookmarkGradients)],
			}
			iconBytes, _ := json.Marshal(icon)
			maxSort++
			item := models.ItemIcon{
				UserID: user.ID, ItemIconGroupID: group.ID, Title: b.Title, Url: b.URL,
				OpenMethod: 1, IconJSON: string(iconBytes), Sort: maxSort,
			}
			if err := tx.Create(&item).Error; err != nil {
				Fail(c, "写入书签卡片失败: "+err.Error())
				return
			}
			existingSet[strings.ToLower(b.URL)] = true
			createdItems++
		}
	}
	if err := tx.Commit().Error; err != nil {
		Fail(c, "提交导入失败: "+err.Error())
		return
	}
	committed = true
	OK(c, gin.H{
		"groups": createdGroups, "items": createdItems,
		"skipped": skipped, "total": len(bookmarks),
	})
}
