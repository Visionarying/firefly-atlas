package api

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"

	"firefly-panel/global"
	"firefly-panel/models"
)

// MonitorData is a compact snapshot of the host running the panel.
type MonitorData struct {
	CPUPercent float64 `json:"cpuPercent"`
	CPUCores   int     `json:"cpuCores"`
	CPUModel   string  `json:"cpuModel"`

	MemTotal   uint64  `json:"memTotal"`
	MemUsed    uint64  `json:"memUsed"`
	MemPercent float64 `json:"memPercent"`

	DiskPath    string  `json:"diskPath"`
	DiskTotal   uint64  `json:"diskTotal"`
	DiskUsed    uint64  `json:"diskUsed"`
	DiskFree    uint64  `json:"diskFree"`
	DiskPercent float64 `json:"diskPercent"`

	NetSent uint64 `json:"netSent"`
	NetRecv uint64 `json:"netRecv"`

	Uptime    uint64 `json:"uptime"`
	Hostname  string `json:"hostname"`
	Platform  string `json:"platform"`
	UpdatedAt int64  `json:"updatedAt"`
}

var (
	monitorMu    sync.Mutex
	monitorCache *MonitorData
	monitorAt    time.Time
)

const monitorCacheTTL = 3 * time.Second

func collectMonitor() MonitorData {
	now := time.Now()
	monitorMu.Lock()
	if monitorCache != nil && now.Sub(monitorAt) < monitorCacheTTL {
		data := *monitorCache
		monitorMu.Unlock()
		return data
	}
	monitorMu.Unlock()

	data := MonitorData{UpdatedAt: now.UnixMilli()}

	if infos, err := cpu.Info(); err == nil && len(infos) > 0 {
		data.CPUModel = infos[0].ModelName
	}
	if cores, err := cpu.Counts(true); err == nil {
		data.CPUCores = cores
	}
	// A 300ms sample yields a usable value even on the first request.
	if percents, err := cpu.Percent(300*time.Millisecond, false); err == nil && len(percents) > 0 {
		data.CPUPercent = percents[0]
	}

	if vm, err := mem.VirtualMemory(); err == nil {
		data.MemTotal = vm.Total
		data.MemUsed = vm.Used
		data.MemPercent = vm.UsedPercent
	}

	path := global.DataDir
	if usage, err := disk.Usage(path); err == nil {
		data.DiskPath = usage.Path
		data.DiskTotal = usage.Total
		data.DiskUsed = usage.Used
		data.DiskFree = usage.Free
		data.DiskPercent = usage.UsedPercent
	} else if parts, err := disk.Partitions(false); err == nil && len(parts) > 0 {
		if usage, err := disk.Usage(parts[0].Mountpoint); err == nil {
			data.DiskPath = usage.Path
			data.DiskTotal = usage.Total
			data.DiskUsed = usage.Used
			data.DiskFree = usage.Free
			data.DiskPercent = usage.UsedPercent
		}
	}

	if counters, err := net.IOCounters(false); err == nil && len(counters) > 0 {
		data.NetSent = counters[0].BytesSent
		data.NetRecv = counters[0].BytesRecv
	}

	if uptime, err := host.Uptime(); err == nil {
		data.Uptime = uptime
	}
	if info, err := host.Info(); err == nil {
		data.Hostname = info.Hostname
		data.Platform = info.Platform + " " + info.PlatformVersion
	}

	monitorMu.Lock()
	monitorCache = &data
	monitorAt = time.Now()
	monitorMu.Unlock()
	return data
}

// MonitorAuth serves host metrics to signed-in users.
func MonitorAuth(c *gin.Context) {
	OK(c, collectMonitor())
}

// MonitorPublic serves host metrics to guests only when the admin opted in.
func MonitorPublic(c *gin.Context) {
	if !global.PublicVisitMode || !panelConfigFlag(1, "showSystemMonitor") || !panelConfigFlag(1, "systemMonitorPublic") {
		OK(c, nil)
		return
	}
	OK(c, collectMonitor())
}

// panelConfigFlag reads a boolean flag from a user's saved panel configuration.
func panelConfigFlag(userID uint, key string) bool {
	var cfg models.UserConfig
	if err := global.DB.Where("user_id = ?", userID).First(&cfg).Error; err != nil {
		return false
	}
	var raw map[string]interface{}
	if err := json.Unmarshal([]byte(cfg.PanelJSON), &raw); err != nil {
		return false
	}
	value, _ := raw[key].(bool)
	return value
}
