package api

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type healthResult struct {
	URL       string `json:"url"`
	Online    bool   `json:"online"`
	Status    int    `json:"status"`
	LatencyMs int64  `json:"latencyMs"`
	Error     string `json:"error,omitempty"`
	CheckedAt int64  `json:"checkedAt"`
}

type healthCacheEntry struct {
	result healthResult
	at     time.Time
}

var (
	healthMu    sync.Mutex
	healthCache = map[string]healthCacheEntry{}
	healthHTTP  = &http.Client{
		Timeout: 5 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return errors.New("重定向次数过多")
			}
			return nil
		},
	}
)

const healthCacheTTL = 30 * time.Second

func isSafeHealthURL(raw string) (*url.URL, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Host == "" {
		return nil, errors.New("地址格式不正确")
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, errors.New("仅支持 http / https 地址")
	}
	if u.Hostname() == "169.254.169.254" {
		return nil, errors.New("不允许探测云元数据地址")
	}
	return u, nil
}

func checkOneHealth(ctx context.Context, raw string) healthResult {
	res := healthResult{URL: raw, CheckedAt: time.Now().UnixMilli()}
	u, err := isSafeHealthURL(raw)
	if err != nil {
		res.Error = err.Error()
		return res
	}

	start := time.Now()
	req, _ := http.NewRequestWithContext(ctx, http.MethodHead, u.String(), nil)
	req.Header.Set("User-Agent", "Firefly-Panel-Health/1.0")
	resp, err := healthHTTP.Do(req)
	if err == nil {
		if resp.StatusCode == http.StatusMethodNotAllowed || resp.StatusCode == http.StatusNotImplemented {
			resp.Body.Close()
			req, _ = http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
			req.Header.Set("User-Agent", "Firefly-Panel-Health/1.0")
			req.Header.Set("Range", "bytes=0-0")
			resp, err = healthHTTP.Do(req)
		}
	}
	if err != nil {
		res.LatencyMs = time.Since(start).Milliseconds()
		res.Error = err.Error()
		return res
	}
	defer resp.Body.Close()
	res.LatencyMs = time.Since(start).Milliseconds()
	res.Status = resp.StatusCode
	res.Online = resp.StatusCode > 0 && resp.StatusCode < 500
	return res
}

func CheckHealth(c *gin.Context) {
	var req struct {
		URLs []string `json:"urls"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || len(req.URLs) == 0 {
		Fail(c, "请提供要探测的地址")
		return
	}
	if len(req.URLs) > 30 {
		req.URLs = req.URLs[:30]
	}

	seen := map[string]bool{}
	unique := make([]string, 0, len(req.URLs))
	for _, raw := range req.URLs {
		raw = strings.TrimSpace(raw)
		if raw == "" || seen[raw] {
			continue
		}
		seen[raw] = true
		unique = append(unique, raw)
	}

	now := time.Now()
	results := make([]healthResult, len(unique))
	sem := make(chan struct{}, 6)
	var wg sync.WaitGroup
	for i, raw := range unique {
		healthMu.Lock()
		entry, ok := healthCache[raw]
		healthMu.Unlock()
		if ok && now.Sub(entry.at) < healthCacheTTL {
			results[i] = entry.result
			continue
		}
		wg.Add(1)
		go func(index int, address string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
			defer cancel()
			result := checkOneHealth(ctx, address)
			results[index] = result
			healthMu.Lock()
			healthCache[address] = healthCacheEntry{result: result, at: time.Now()}
			if len(healthCache) > 512 {
				for key, item := range healthCache {
					if time.Since(item.at) > 10*time.Minute {
						delete(healthCache, key)
					}
				}
			}
			healthMu.Unlock()
		}(i, raw)
	}
	wg.Wait()
	OK(c, results)
}
