package api

import (
	"context"
	"log"
	"math"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"

	"firefly-atlas/global"
	"firefly-atlas/models"
)

const (
	// healthSampleInterval is the fixed cadence required by v1.3.0 Sentinel.
	healthSampleInterval = 5 * time.Minute
	healthRetention      = 7 * 24 * time.Hour
	healthWindow24h      = 24 * time.Hour
	healthWorkers        = 6
	healthItemTimeout    = 12 * time.Second
)

var (
	healthRunMu    sync.Mutex
	healthSampling atomic.Bool
)

// StartHealthSampler launches the background five minute sampler. The first
// pass runs shortly after startup so a fresh deployment immediately has data.
func StartHealthSampler() {
	go func() {
		timer := time.NewTimer(10 * time.Second)
		defer timer.Stop()
		log.Printf("[firefly-atlas] health sampler started (every %s, retention %s)", healthSampleInterval, healthRetention)
		for {
			<-timer.C
			runHealthSample(false)
			timer.Reset(healthSampleInterval)
		}
	}()
}

func getHealthSetting(name string) string {
	var setting models.SystemSetting
	if err := global.DB.Where("config_name = ?", name).First(&setting).Error; err != nil {
		return ""
	}
	return strings.TrimSpace(setting.Content)
}

// maintenanceActive reports whether the current server-local time falls inside
// the configured daily maintenance window. Windows may wrap past midnight
// (for example 23:30 - 01:30). Equal start/end values disable the window.
func maintenanceActive(now time.Time) bool {
	if getHealthSetting("maintenance_enabled") != "1" {
		return false
	}
	start, okStart := parseClockMinutes(getHealthSetting("maintenance_start"))
	end, okEnd := parseClockMinutes(getHealthSetting("maintenance_end"))
	if !okStart || !okEnd || start == end {
		return false
	}
	current := now.Hour()*60 + now.Minute()
	if start < end {
		return current >= start && current < end
	}
	return current >= start || current < end
}

func parseClockMinutes(value string) (int, bool) {
	parts := strings.Split(value, ":")
	if len(parts) != 2 {
		return 0, false
	}
	hour, errHour := strconv.Atoi(strings.TrimSpace(parts[0]))
	minute, errMinute := strconv.Atoi(strings.TrimSpace(parts[1]))
	if errHour != nil || errMinute != nil || hour < 0 || hour > 23 || minute < 0 || minute > 59 {
		return 0, false
	}
	return hour*60 + minute, true
}

// healthURLs returns the addresses the server should probe. The LAN address is
// preferred because the panel usually runs next to the services it monitors;
// the public address is the fallback. Duplicates are removed.
func healthURLs(item models.ItemIcon) []string {
	seen := map[string]bool{}
	out := make([]string, 0, 2)
	for _, raw := range []string{item.LanUrl, item.Url} {
		raw = strings.TrimSpace(raw)
		if raw == "" || seen[raw] {
			continue
		}
		seen[raw] = true
		out = append(out, raw)
	}
	return out
}

func checkItemHealth(ctx context.Context, item models.ItemIcon) healthResult {
	urls := healthURLs(item)
	if len(urls) == 0 {
		return healthResult{}
	}
	var primary healthResult
	for index, raw := range urls {
		result := checkOneHealth(ctx, raw)
		if index == 0 {
			primary = result
		}
		if result.Online {
			return result
		}
	}
	return primary
}

func runHealthSample(force bool) {
	monitoringEnabled := getHealthSetting("health_monitor_enabled") == "1"
	if !force && !monitoringEnabled {
		return
	}
	if !healthRunMu.TryLock() {
		return
	}
	defer healthRunMu.Unlock()
	healthSampling.Store(true)
	defer healthSampling.Store(false)

	var items []models.ItemIcon
	if err := global.DB.Where("health_check = ?", true).Find(&items).Error; err != nil {
		log.Println("[firefly-atlas] health sample query:", err)
		return
	}
	if len(items) == 0 {
		pruneHealthSamples()
		return
	}

	ids := make([]uint, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}
	previous := loadLatestOnline(ids)
	inMaintenance := maintenanceActive(time.Now())

	results := make([]healthResult, len(items))
	sem := make(chan struct{}, healthWorkers)
	var wg sync.WaitGroup
	for index := range items {
		if len(healthURLs(items[index])) == 0 {
			continue
		}
		wg.Add(1)
		go func(idx int, item models.ItemIcon) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			ctx, cancel := context.WithTimeout(context.Background(), healthItemTimeout)
			defer cancel()
			results[idx] = checkItemHealth(ctx, item)
		}(index, items[index])
	}
	wg.Wait()

	samples := make([]models.HealthSample, 0, len(items))
	onlineCount := 0
	for index, item := range items {
		result := results[index]
		if result.URL == "" {
			continue
		}
		if result.Online {
			onlineCount++
		}
		alert := false
		// A manual run while background monitoring is off may still record a
		// sample, but it must not raise an alert the user asked not to receive.
		if wasOnline, ok := previous[item.ID]; ok && wasOnline && !result.Online && !inMaintenance && monitoringEnabled {
			alert = true
		}
		samples = append(samples, models.HealthSample{
			ItemID:    item.ID,
			Online:    result.Online,
			Status:    result.Status,
			LatencyMs: result.LatencyMs,
			CheckedAt: time.Now().UnixMilli(),
			Alert:     alert,
		})
	}

	if len(samples) > 0 {
		if err := global.DB.CreateInBatches(samples, 100).Error; err != nil {
			log.Println("[firefly-atlas] health sample insert:", err)
		}
	}
	pruneHealthSamples()
	log.Printf("[firefly-atlas] health sample: %d checked, %d online", len(samples), onlineCount)
}

func loadLatestOnline(ids []uint) map[uint]bool {
	out := map[uint]bool{}
	if len(ids) == 0 {
		return out
	}
	var rows []struct {
		ItemID uint
		Online bool
	}
	global.DB.Raw(`
		SELECT hs.item_id, hs.online
		FROM health_samples hs
		JOIN (
			SELECT item_id, MAX(id) AS max_id
			FROM health_samples
			WHERE item_id IN ?
			GROUP BY item_id
		) latest ON hs.id = latest.max_id`, ids).Scan(&rows)
	for _, row := range rows {
		out[row.ItemID] = row.Online
	}
	return out
}

func loadLatestSamples(ids []uint) map[uint]models.HealthSample {
	out := map[uint]models.HealthSample{}
	if len(ids) == 0 {
		return out
	}
	var rows []models.HealthSample
	global.DB.Raw(`
		SELECT hs.*
		FROM health_samples hs
		JOIN (
			SELECT item_id, MAX(id) AS max_id
			FROM health_samples
			WHERE item_id IN ?
			GROUP BY item_id
		) latest ON hs.id = latest.max_id`, ids).Scan(&rows)
	for _, row := range rows {
		out[row.ItemID] = row
	}
	return out
}

// pruneHealthSamples enforces the seven day retention and removes samples for
// deleted cards. It runs after every sampling pass.
func pruneHealthSamples() {
	cutoff := time.Now().Add(-healthRetention).UnixMilli()
	if err := global.DB.Where("checked_at < ?", cutoff).Delete(&models.HealthSample{}).Error; err != nil {
		log.Println("[firefly-atlas] health prune:", err)
	}
	global.DB.Exec("DELETE FROM health_samples WHERE item_id NOT IN (SELECT id FROM item_icons)")
}

type healthMaintenanceInfo struct {
	Enabled bool   `json:"enabled"`
	Start   string `json:"start"`
	End     string `json:"end"`
}

type healthOverviewItem struct {
	ItemID        uint     `json:"itemId"`
	Title         string   `json:"title"`
	GroupID       uint     `json:"groupId"`
	URL           string   `json:"url"`
	Monitored     bool     `json:"monitored"`
	Online        bool     `json:"online"`
	Status        int      `json:"status"`
	LatencyMs     int64    `json:"latencyMs"`
	CheckedAt     int64    `json:"checkedAt"`
	Alert         bool     `json:"alert"`
	Samples24h    int64    `json:"samples24h"`
	OnlineSamples int64    `json:"onlineSamples24h"`
	Uptime24h     *float64 `json:"uptime24h"`
	LastOnlineAt  int64    `json:"lastOnlineAt"`
}

type healthOverview struct {
	Enabled         bool                  `json:"enabled"`
	Sampling        bool                  `json:"sampling"`
	IntervalMinutes int                   `json:"intervalMinutes"`
	RetentionDays   int                   `json:"retentionDays"`
	Maintenance     healthMaintenanceInfo `json:"maintenance"`
	Items           []healthOverviewItem  `json:"items"`
}

// HealthOverview aggregates the last 24 hours with SQL. The full sample table
// is never loaded into memory: only one row per card leaves the database.
func HealthOverview(c *gin.Context) {
	user := CurrentUser(c)
	if user == nil {
		Fail(c, "未登录")
		return
	}

	var items []models.ItemIcon
	global.DB.Where("user_id = ?", user.ID).
		Order("item_icon_group_id asc, sort asc, id asc").Find(&items)

	ids := make([]uint, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}
	latest := loadLatestSamples(ids)

	type aggregateRow struct {
		ItemID        uint
		Samples       int64
		OnlineSamples int64
		LastOnlineAt  int64
	}
	aggregates := map[uint]aggregateRow{}
	if len(ids) > 0 {
		cutoff := time.Now().Add(-healthWindow24h).UnixMilli()
		var rows []aggregateRow
		global.DB.Raw(`
			SELECT item_id,
			       COUNT(*) AS samples,
			       SUM(CASE WHEN online THEN 1 ELSE 0 END) AS online_samples,
			       MAX(CASE WHEN online THEN checked_at ELSE 0 END) AS last_online_at
			FROM health_samples
			WHERE item_id IN ? AND checked_at >= ?
			GROUP BY item_id`, ids, cutoff).Scan(&rows)
		for _, row := range rows {
			aggregates[row.ItemID] = row
		}
	}

	out := make([]healthOverviewItem, 0, len(items))
	for _, item := range items {
		urls := healthURLs(item)
		displayURL := ""
		if len(urls) > 0 {
			displayURL = urls[0]
		}
		entry := healthOverviewItem{
			ItemID:    item.ID,
			Title:     item.Title,
			GroupID:   item.ItemIconGroupID,
			URL:       displayURL,
			Monitored: item.HealthCheck,
		}
		if sample, ok := latest[item.ID]; ok {
			entry.Online = sample.Online
			entry.Status = sample.Status
			entry.LatencyMs = sample.LatencyMs
			entry.CheckedAt = sample.CheckedAt
			entry.Alert = sample.Alert
		}
		if aggregate, ok := aggregates[item.ID]; ok {
			entry.Samples24h = aggregate.Samples
			entry.OnlineSamples = aggregate.OnlineSamples
			entry.LastOnlineAt = aggregate.LastOnlineAt
			if aggregate.Samples > 0 {
				uptime := math.Round(float64(aggregate.OnlineSamples)/float64(aggregate.Samples)*1000) / 10
				entry.Uptime24h = &uptime
			}
		}
		out = append(out, entry)
	}

	maintenanceStart := getHealthSetting("maintenance_start")
	if maintenanceStart == "" {
		maintenanceStart = "02:00"
	}
	maintenanceEnd := getHealthSetting("maintenance_end")
	if maintenanceEnd == "" {
		maintenanceEnd = "04:00"
	}

	OK(c, healthOverview{
		Enabled:         getHealthSetting("health_monitor_enabled") == "1",
		Sampling:        healthSampling.Load(),
		IntervalMinutes: int(healthSampleInterval / time.Minute),
		RetentionDays:   int(healthRetention / (24 * time.Hour)),
		Maintenance: healthMaintenanceInfo{
			Enabled: getHealthSetting("maintenance_enabled") == "1",
			Start:   maintenanceStart,
			End:     maintenanceEnd,
		},
		Items: out,
	})
}

// RunHealthSample triggers one immediate sampling pass. It is admin-only and
// runs in the background so the HTTP request never waits for slow targets.
func RunHealthSample(c *gin.Context) {
	user := CurrentUser(c)
	if user == nil {
		Fail(c, "未登录")
		return
	}
	if user.Role != 1 {
		FailCode(c, 403, "无权限")
		return
	}
	if healthSampling.Load() {
		OK(c, gin.H{"started": false, "running": true})
		return
	}
	go runHealthSample(true)
	OK(c, gin.H{"started": true, "running": true})
}
