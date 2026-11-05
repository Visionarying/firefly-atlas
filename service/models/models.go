package models

import "time"

// User represents a panel account. Role 1 = admin, 2 = standard user.
type User struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Username  string    `gorm:"uniqueIndex;size:64" json:"username"`
	Password  string    `gorm:"size:255" json:"-"`
	Nickname  string    `gorm:"size:64" json:"nickname"`
	Role      int       `gorm:"default:2" json:"role"`
	Status    int       `gorm:"default:1" json:"status"`
	Token     string    `gorm:"size:128;index" json:"-"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type ItemIconGroup struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"index" json:"userId"`
	Title     string    `gorm:"size:128" json:"title"`
	Sort      int       `gorm:"default:0" json:"sort"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type ItemIconIcon struct {
	ItemType        int    `json:"itemType"` // 1 image, 2 text, 3 svg
	Src             string `json:"src"`
	Text            string `json:"text"`
	BackgroundColor string `json:"backgroundColor"`
}

type ItemIcon struct {
	ID              uint         `gorm:"primaryKey" json:"id"`
	UserID          uint         `gorm:"index" json:"userId"`
	ItemIconGroupID uint         `gorm:"index" json:"itemIconGroupId"`
	Title           string       `gorm:"size:128" json:"title"`
	Url             string       `gorm:"size:1024" json:"url"`
	LanUrl          string       `gorm:"size:1024" json:"lanUrl"`
	Description     string       `gorm:"size:512" json:"description"`
	OpenMethod      int          `gorm:"default:1" json:"openMethod"`
	HealthCheck     bool         `gorm:"default:false" json:"healthCheck"`
	IconJSON        string       `gorm:"type:text" json:"-"`
	Icon            ItemIconIcon `gorm:"-" json:"icon"`
	Sort            int          `gorm:"default:0" json:"sort"`
	CreatedAt       time.Time    `json:"createdAt"`
	UpdatedAt       time.Time    `json:"updatedAt"`
}

type UserConfig struct {
	ID               uint      `gorm:"primaryKey" json:"id"`
	UserID           uint      `gorm:"uniqueIndex" json:"userId"`
	PanelJSON        string    `gorm:"type:text" json:"-"`
	SearchEngineJSON string    `gorm:"type:text" json:"-"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
}

type SystemSetting struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	ConfigName string    `gorm:"uniqueIndex;size:64" json:"configName"`
	Title      string    `gorm:"size:128" json:"title"`
	Content    string    `gorm:"type:text" json:"content"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

type File struct {
	ID          uint       `gorm:"primaryKey" json:"id"`
	UserID      uint       `gorm:"index" json:"userId"`
	FileName    string     `gorm:"size:255" json:"fileName"`
	Src         string     `gorm:"size:1024" json:"src"`
	FileType    int        `gorm:"index" json:"fileType"` // 1 wallpaper, 2 icon
	Size        int64      `json:"size"`
	SyncStatus  int        `gorm:"default:0" json:"syncStatus"` // 0 local, 1 pending, 2 synced, 3 rejected
	AuditRemark string     `gorm:"size:255" json:"auditRemark"`
	CloudKey    string     `gorm:"size:512" json:"cloudKey"`
	CloudURL    string     `gorm:"size:1024" json:"cloudUrl"`
	SyncedAt    *time.Time `json:"syncedAt"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
}

// CloudStorage holds the single encrypted-at-rest (file permissions) cloud
// provider configuration row. It is deliberately kept out of SystemSetting so
// normal users never receive provider secrets with the public panel payload.
type CloudStorage struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	ConfigJSON string    `gorm:"type:text" json:"-"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

// DefaultPanelConfig is the fallback visual configuration for a fresh install.
const DefaultPanelConfig = `{
  "logoText": "Firefly Panel",
  "logoColor": "",
  "logoFontFamily": "'LXGW WenKai', 'Microsoft YaHei', sans-serif",
  "logoGlassEffect": false,
  "headerOpacity": 100,
  "cardGlassOpacity": 45,
  "footerOffsetY": 0,
  "footerHtml": "<span>Firefly Panel · 私有云导航</span>",
  "iconStyle": 0,
  "maxWidth": "1280",
  "maxWidthUnit": "px",
  "marginTop": 8,
  "marginBottom": 4,
  "marginX": 24,
  "itemGap": 16,
  "showHeader": true,
  "showClock": true,
  "showSearch": true,
  "showFooter": true,
  "hideTitle": false,
  "hideDescription": false,
  "searchFilterItems": true,
  "showSystemMonitor": false,
  "systemMonitorShowTitle": true,
  "systemMonitorPublic": false,
  "cardRadius": 18,
  "backgroundImage": "",
  "backgroundBlur": 0,
  "backgroundMask": 30,
  "clockShowSeconds": false,
  "clockShowDate": true,
  "searchPlaceholder": "搜索，或输入网址后回车",
  "searchOpenInNewTab": true
}`

const DefaultSearchEngines = `[
  {"name": "百度", "url": "https://www.baidu.com/s?wd=", "icon": ""},
  {"name": "Google", "url": "https://www.google.com/search?q=", "icon": ""},
  {"name": "Bing", "url": "https://www.bing.com/search?q=", "icon": ""},
  {"name": "GitHub", "url": "https://github.com/search?q=", "icon": ""}
]`
