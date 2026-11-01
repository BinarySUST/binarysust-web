package media

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
	"strings"
	"time"

	"github.com/sakib412/ekman/storage"
)

// GitHubBucket stores objects as files in a GitHub repository through the Contents API
// (one commit per upload). The repository is deployed to a static host at PublicBaseURL.
type GitHubBucket struct {
	Token         string // a token with contents:write on Repository
	Repository    string // "owner/name", e.g. BinarySUST/binarysust-media
	Branch        string // default "main"
	PublicBaseURL string // where the repository is served, e.g. https://media.binarysust.org
	APIBaseURL    string // default https://api.github.com
	Client        *http.Client
}

var _ storage.Bucket = (*GitHubBucket)(nil)

func (g *GitHubBucket) client() *http.Client {
	if g.Client != nil {
		return g.Client
	}
	return &http.Client{Timeout: 30 * time.Second}
}

func (g *GitHubBucket) branch() string {
	if g.Branch == "" {
		return "main"
	}
	return g.Branch
}

func (g *GitHubBucket) contentsURL(key string) (string, error) {
	if key == "" || strings.HasPrefix(key, "/") || strings.Contains(key, "\\") || path.Clean(key) != key || strings.HasPrefix(key, "..") {
		return "", storage.ErrInvalidKey
	}
	base := g.APIBaseURL
	if base == "" {
		base = "https://api.github.com"
	}
	segs := strings.Split(key, "/")
	for i, s := range segs {
		segs[i] = url.PathEscape(s)
	}
	return strings.TrimSuffix(base, "/") + "/repos/" + g.Repository + "/contents/" + strings.Join(segs, "/"), nil
}

func (g *GitHubBucket) do(ctx context.Context, method, u string, body any) (*http.Response, error) {
	var rd io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		rd = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, u, rd)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+g.Token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return g.client().Do(req)
}

func apiError(op string, resp *http.Response) error {
	msg, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
	return fmt.Errorf("storage: github %s: %s: %s", op, resp.Status, bytes.TrimSpace(msg))
}

// Put commits r as key. Keys are random (storage.Receive), so Put always creates a file.
func (g *GitHubBucket) Put(ctx context.Context, key string, r io.Reader, contentType string) (storage.Object, error) {
	u, err := g.contentsURL(key)
	if err != nil {
		return storage.Object{}, err
	}
	data, err := io.ReadAll(r) // bounded by the upload policy's size limit
	if err != nil {
		return storage.Object{}, err
	}
	resp, err := g.do(ctx, http.MethodPut, u, map[string]string{
		"message": "upload " + key,
		"content": base64.StdEncoding.EncodeToString(data),
		"branch":  g.branch(),
	})
	if err != nil {
		return storage.Object{}, fmt.Errorf("storage: github put: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return storage.Object{}, apiError("put", resp)
	}
	return storage.Object{Key: key, URL: g.PublicURL(key), ContentType: contentType, Size: int64(len(data))}, nil
}

// Delete removes key; a missing key is not an error.
func (g *GitHubBucket) Delete(ctx context.Context, key string) error {
	u, err := g.contentsURL(key)
	if err != nil {
		return err
	}
	resp, err := g.do(ctx, http.MethodGet, u+"?ref="+url.QueryEscape(g.branch()), nil)
	if err != nil {
		return fmt.Errorf("storage: github stat: %w", err)
	}
	var meta struct {
		SHA string `json:"sha"`
	}
	switch {
	case resp.StatusCode == http.StatusNotFound:
		resp.Body.Close()
		return nil
	case resp.StatusCode != http.StatusOK:
		defer resp.Body.Close()
		return apiError("stat", resp)
	}
	err = json.NewDecoder(resp.Body).Decode(&meta)
	resp.Body.Close()
	if err != nil || meta.SHA == "" {
		return errors.Join(errors.New("storage: github stat: no sha"), err)
	}
	resp, err = g.do(ctx, http.MethodDelete, u, map[string]string{
		"message": "delete " + key, "sha": meta.SHA, "branch": g.branch(),
	})
	if err != nil {
		return fmt.Errorf("storage: github delete: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil
	}
	if resp.StatusCode != http.StatusOK {
		return apiError("delete", resp)
	}
	return nil
}

// PublicURL is where key is served once the media site deploys.
func (g *GitHubBucket) PublicURL(key string) string {
	segs := strings.Split(key, "/")
	for i, s := range segs {
		segs[i] = url.PathEscape(s)
	}
	return strings.TrimSuffix(g.PublicBaseURL, "/") + "/" + strings.Join(segs, "/")
}
