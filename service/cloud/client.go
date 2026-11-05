package cloud

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// Config is the persisted cloud provider configuration. Secrets never leave
// the admin-only endpoints.
type Config struct {
	Enabled   bool         `json:"enabled"`
	Provider  string       `json:"provider"` // github | s3
	DualWrite bool         `json:"dualWrite"`
	GitHub    GitHubConfig `json:"github"`
	S3        S3Config     `json:"s3"`
}

type GitHubConfig struct {
	Owner    string `json:"owner"`
	Repo     string `json:"repo"`
	Branch   string `json:"branch"`
	Token    string `json:"token"`
	BasePath string `json:"basePath"`
	UseCDN   bool   `json:"useCdn"`
}

type S3Config struct {
	// Endpoint can be omitted when AccountID is set for Cloudflare R2.
	Endpoint        string `json:"endpoint"`
	AccountID       string `json:"accountId"`
	Region          string `json:"region"`
	AccessKeyID     string `json:"accessKeyId"`
	SecretAccessKey string `json:"secretAccessKey"`
	Bucket          string `json:"bucket"`
	PublicBaseURL   string `json:"publicBaseUrl"`
	Insecure        bool   `json:"insecure"`
}

type FileItem struct {
	Key         string    `json:"key"`
	Name        string    `json:"name"`
	Size        int64     `json:"size"`
	UpdatedAt   time.Time `json:"updatedAt"`
	URL         string    `json:"url"`
	IsDirectory bool      `json:"isDirectory"`
}

type Client interface {
	Test(ctx context.Context) (private bool, err error)
	Upload(ctx context.Context, key string, data []byte, contentType string) (string, error)
	Download(ctx context.Context, key string) ([]byte, error)
	List(ctx context.Context, prefix string) ([]FileItem, error)
	Delete(ctx context.Context, key string) error
	PublicURL(key string) string
}

func New(cfg Config) (Client, error) {
	switch strings.ToLower(strings.TrimSpace(cfg.Provider)) {
	case "github":
		return NewGitHub(cfg.GitHub), nil
	case "s3", "r2", "minio":
		return NewS3(cfg.S3)
	default:
		return nil, fmt.Errorf("不支持的云存储类型: %s", cfg.Provider)
	}
}

// ---------------------------------------------------------------- GitHub

type GitHub struct {
	cfg    GitHubConfig
	client *http.Client
}

func NewGitHub(cfg GitHubConfig) *GitHub {
	if cfg.Branch == "" {
		cfg.Branch = "main"
	}
	cfg.BasePath = strings.Trim(cfg.BasePath, "/")
	if cfg.BasePath == "" {
		cfg.BasePath = "firefly-panel"
	}
	cfg.Owner = strings.TrimSpace(cfg.Owner)
	cfg.Repo = strings.TrimSpace(cfg.Repo)
	return &GitHub{cfg: cfg, client: &http.Client{Timeout: 30 * time.Second}}
}

func (g *GitHub) fullPath(key string) string {
	key = strings.TrimPrefix(strings.ReplaceAll(key, "\\", "/"), "/")
	return path.Join(g.cfg.BasePath, key)
}

func escapePath(p string) string {
	parts := strings.Split(strings.Trim(p, "/"), "/")
	for i, s := range parts {
		parts[i] = url.PathEscape(s)
	}
	return strings.Join(parts, "/")
}

func (g *GitHub) apiURL(key string, raw bool) string {
	u := fmt.Sprintf("https://api.github.com/repos/%s/%s/contents/%s",
		url.PathEscape(g.cfg.Owner), url.PathEscape(g.cfg.Repo), escapePath(g.fullPath(key)))
	if raw {
		return u + "?ref=" + url.QueryEscape(g.cfg.Branch)
	}
	return u
}

func (g *GitHub) request(ctx context.Context, method, u string, body io.Reader, accept string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, u, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+g.cfg.Token)
	req.Header.Set("Accept", accept)
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return g.client.Do(req)
}

func (g *GitHub) Test(ctx context.Context) (bool, error) {
	if g.cfg.Owner == "" || g.cfg.Repo == "" || g.cfg.Token == "" {
		return false, errors.New("GitHub 用户名、仓库名和 Token 不能为空")
	}
	resp, err := g.request(ctx, http.MethodGet,
		fmt.Sprintf("https://api.github.com/repos/%s/%s", url.PathEscape(g.cfg.Owner), url.PathEscape(g.cfg.Repo)),
		nil, "application/vnd.github+json")
	if err != nil {
		return false, fmt.Errorf("网络连接失败: %w", err)
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusUnauthorized:
		return false, errors.New("GitHub Token 无效或已过期")
	case http.StatusNotFound:
		return false, errors.New("仓库不存在，或 Token 无权限访问该私有仓库")
	case http.StatusOK:
	default:
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return false, fmt.Errorf("GitHub API 返回 %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}
	var info struct {
		Private bool `json:"private"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return false, err
	}
	return info.Private, nil
}

func (g *GitHub) sha(ctx context.Context, key string) (string, error) {
	resp, err := g.request(ctx, http.MethodGet, g.apiURL(key, true), nil, "application/vnd.github+json")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return "", nil
	}
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return "", fmt.Errorf("查询 GitHub 文件失败 %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}
	var meta struct {
		SHA string `json:"sha"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&meta); err != nil {
		return "", err
	}
	return meta.SHA, nil
}

func (g *GitHub) Upload(ctx context.Context, key string, data []byte, contentType string) (string, error) {
	oldSHA, err := g.sha(ctx, key)
	if err != nil {
		return "", err
	}
	payload := map[string]interface{}{
		"message": "Sync " + path.Base(key) + " via Firefly Panel",
		"content": base64.StdEncoding.EncodeToString(data),
		"branch":  g.cfg.Branch,
	}
	if oldSHA != "" {
		payload["sha"] = oldSHA
	}
	body, _ := json.Marshal(payload)
	resp, err := g.request(ctx, http.MethodPut, g.apiURL(key, false), bytes.NewReader(body), "application/vnd.github+json")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return "", fmt.Errorf("GitHub 上传失败 %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}
	return g.PublicURL(key), nil
}

func (g *GitHub) Download(ctx context.Context, key string) ([]byte, error) {
	resp, err := g.request(ctx, http.MethodGet, g.apiURL(key, true), nil, "application/vnd.github.raw")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("GitHub 下载失败 %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}
	return io.ReadAll(io.LimitReader(resp.Body, 1<<30))
}

func (g *GitHub) List(ctx context.Context, prefix string) ([]FileItem, error) {
	resp, err := g.request(ctx, http.MethodGet, g.apiURL(prefix, true), nil, "application/vnd.github+json")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return []FileItem{}, nil
	}
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("GitHub 列目录失败 %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}
	var rows []struct {
		Name        string `json:"name"`
		Path        string `json:"path"`
		Size        int64  `json:"size"`
		Type        string `json:"type"`
		DownloadURL string `json:"download_url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&rows); err != nil {
		return nil, err
	}
	out := make([]FileItem, 0, len(rows))
	for _, row := range rows {
		rel := row.Path
		if g.cfg.BasePath != "" {
			rel = strings.TrimPrefix(rel, g.cfg.BasePath+"/")
		}
		item := FileItem{
			Key:         rel,
			Name:        row.Name,
			Size:        row.Size,
			IsDirectory: row.Type == "dir",
			UpdatedAt:   time.Now(),
			URL:         row.DownloadURL,
		}
		if g.cfg.UseCDN && row.Type == "file" {
			item.URL = fmt.Sprintf("https://cdn.jsdelivr.net/gh/%s/%s@%s/%s",
				g.cfg.Owner, g.cfg.Repo, g.cfg.Branch, row.Path)
		}
		out = append(out, item)
	}
	return out, nil
}

func (g *GitHub) Delete(ctx context.Context, key string) error {
	sha, err := g.sha(ctx, key)
	if err != nil || sha == "" {
		return err
	}
	body, _ := json.Marshal(map[string]string{
		"message": "Delete " + path.Base(key) + " via Firefly Panel",
		"sha":     sha,
		"branch":  g.cfg.Branch,
	})
	resp, err := g.request(ctx, http.MethodDelete, g.apiURL(key, false), bytes.NewReader(body), "application/vnd.github+json")
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("GitHub 删除失败 %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}
	return nil
}

func (g *GitHub) PublicURL(key string) string {
	if g.cfg.UseCDN {
		return fmt.Sprintf("https://cdn.jsdelivr.net/gh/%s/%s@%s/%s",
			g.cfg.Owner, g.cfg.Repo, g.cfg.Branch, g.fullPath(key))
	}
	return fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s/%s",
		g.cfg.Owner, g.cfg.Repo, g.cfg.Branch, g.fullPath(key))
}

// ---------------------------------------------------------------- S3 / R2 / MinIO

type S3 struct {
	cfg    S3Config
	client *minio.Client
}

func NewS3(cfg S3Config) (*S3, error) {
	endpoint := strings.TrimSpace(cfg.Endpoint)
	if endpoint == "" && cfg.AccountID != "" {
		endpoint = cfg.AccountID + ".r2.cloudflarestorage.com"
	}
	endpoint = strings.TrimPrefix(strings.TrimPrefix(endpoint, "https://"), "http://")
	endpoint = strings.TrimSuffix(endpoint, "/")
	if endpoint == "" {
		return nil, errors.New("S3 / R2 Endpoint 或 AccountID 不能为空")
	}
	if cfg.Bucket == "" {
		return nil, errors.New("存储桶名称不能为空")
	}
	region := strings.TrimSpace(cfg.Region)
	if region == "" {
		region = "auto"
	}
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		Secure: !cfg.Insecure,
		Region: region,
	})
	if err != nil {
		return nil, err
	}
	return &S3{cfg: cfg, client: client}, nil
}

func (s *S3) Test(ctx context.Context) (bool, error) {
	ok, err := s.client.BucketExists(ctx, s.cfg.Bucket)
	if err != nil {
		return false, err
	}
	if !ok {
		return false, fmt.Errorf("存储桶 %s 不存在", s.cfg.Bucket)
	}
	return strings.TrimSpace(s.cfg.PublicBaseURL) == "", nil
}

func (s *S3) Upload(ctx context.Context, key string, data []byte, contentType string) (string, error) {
	key = strings.TrimPrefix(strings.ReplaceAll(key, "\\", "/"), "/")
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	_, err := s.client.PutObject(ctx, s.cfg.Bucket, key, bytes.NewReader(data), int64(len(data)),
		minio.PutObjectOptions{ContentType: contentType})
	if err != nil {
		return "", err
	}
	return s.PublicURL(key), nil
}

func (s *S3) Download(ctx context.Context, key string) ([]byte, error) {
	key = strings.TrimPrefix(strings.ReplaceAll(key, "\\", "/"), "/")
	obj, err := s.client.GetObject(ctx, s.cfg.Bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}
	defer obj.Close()
	return io.ReadAll(io.LimitReader(obj, 1<<30))
}

func (s *S3) List(ctx context.Context, prefix string) ([]FileItem, error) {
	prefix = strings.TrimPrefix(prefix, "/")
	ch := s.client.ListObjects(ctx, s.cfg.Bucket, minio.ListObjectsOptions{
		Prefix: prefix, Recursive: true,
	})
	var out []FileItem
	for obj := range ch {
		if obj.Err != nil {
			return nil, obj.Err
		}
		out = append(out, FileItem{
			Key:         obj.Key,
			Name:        path.Base(obj.Key),
			Size:        obj.Size,
			UpdatedAt:   obj.LastModified,
			URL:         s.PublicURL(obj.Key),
			IsDirectory: strings.HasSuffix(obj.Key, "/"),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].UpdatedAt.After(out[j].UpdatedAt) })
	return out, nil
}

func (s *S3) Delete(ctx context.Context, key string) error {
	key = strings.TrimPrefix(strings.ReplaceAll(key, "\\", "/"), "/")
	return s.client.RemoveObject(ctx, s.cfg.Bucket, key, minio.RemoveObjectOptions{})
}

func (s *S3) PublicURL(key string) string {
	base := strings.TrimSuffix(strings.TrimSpace(s.cfg.PublicBaseURL), "/")
	if base == "" {
		return ""
	}
	return base + "/" + strings.TrimPrefix(key, "/")
}
