package api

import (
	"context"
	"errors"
	"net"
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
			// Validate every hop. Checking only the first address would let a
			// public URL redirect the server into a cloud metadata endpoint.
			if _, err := isSafeHealthURL(req.URL.String()); err != nil {
				return err
			}
			return nil
		},
	}
)

const healthCacheTTL = 30 * time.Second

// metadataHosts are the well-known cloud metadata endpoints. Ordinary LAN and
// loopback addresses stay allowed on purpose: probing a service next to the
// panel is the main use case, and users may legitimately monitor 127.0.0.1.
var metadataHosts = map[string]bool{
	"169.254.169.254":          true,
	"100.100.100.200":          true,
	"metadata.google.internal": true,
	"metadata.goog":            true,
}

func isSafeHealthURL(raw string) (*url.URL, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Host == "" {
		return nil, errors.New("地址格式不正确")
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, errors.New("仅支持 http / https 地址")
	}
	host := strings.ToLower(u.Hostname())
	if metadataHosts[host] {
		return nil, errors.New("不允许探测云元数据地址")
	}
	if ip := net.ParseIP(host); ip != nil && ip.IsLinkLocalUnicast() {
		// Link-local covers 169.254.0.0/16, including IPv4-mapped forms such
		// as ::ffff:169.254.169.254.
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
	build := func(method string) (*http.Request, error) {
		req, err := http.NewRequestWithContext(ctx, method, u.String(), nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("User-Agent", "Firefly-Atlas-Health/1.0")
		if method == http.MethodGet {
			req.Header.Set("Range", "bytes=0-0")
		}
		return req, nil
	}
	req, err := build(http.MethodHead)
	if err != nil {
		res.LatencyMs = time.Since(start).Milliseconds()
		res.Error = err.Error()
		return res
	}
	resp, err := healthHTTP.Do(req)
	if err == nil {
		if resp.StatusCode == http.StatusMethodNotAllowed || resp.StatusCode == http.StatusNotImplemented {
			resp.Body.Close()
			fallback, buildErr := build(http.MethodGet)
			if buildErr != nil {
				res.LatencyMs = time.Since(start).Milliseconds()
				res.Error = buildErr.Error()
				return res
			}
			resp, err = healthHTTP.Do(fallback)
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
