package republish

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// ErrNotFound is returned by KV.Get for a key that does not exist (or has expired).
var ErrNotFound = errors.New("kv: key not found")

// KV is one Cloudflare Workers KV namespace, reached through Cloudflare's REST API
// (https://developers.cloudflare.com/api/resources/kv/). Workers KV is free without a card
// (1,000 writes and 100,000 reads a day), which is why the snapshot lives here and not in R2.
type KV struct {
	AccountID   string
	NamespaceID string
	Token       string       // an API token with "Workers KV Storage: Edit"
	BaseURL     string       // default https://api.cloudflare.com/client/v4
	Client      *http.Client // default: 15 s timeout
}

func (k *KV) url(key string) string {
	base := k.BaseURL
	if base == "" {
		base = "https://api.cloudflare.com/client/v4"
	}
	return strings.TrimSuffix(base, "/") + "/accounts/" + url.PathEscape(k.AccountID) +
		"/storage/kv/namespaces/" + url.PathEscape(k.NamespaceID) + "/values/" + url.PathEscape(key)
}

func (k *KV) do(req *http.Request) (*http.Response, error) {
	client := k.Client
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	req.Header.Set("Authorization", "Bearer "+k.Token)
	return client.Do(req)
}

// Put stores value under key. A positive ttl makes the key expire (KV's minimum is 60 s);
// zero keeps it forever.
func (k *KV) Put(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	u := k.url(key)
	if ttl > 0 {
		u += "?expiration_ttl=" + strconv.Itoa(int(ttl/time.Second))
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, u, bytes.NewReader(value))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/octet-stream")
	resp, err := k.do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return statusErr("put "+key, resp)
	}
	return nil
}

// Get returns the value under key, or ErrNotFound.
func (k *KV) Get(ctx context.Context, key string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, k.url(key), nil)
	if err != nil {
		return nil, err
	}
	resp, err := k.do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrNotFound
	}
	if resp.StatusCode/100 != 2 {
		return nil, statusErr("get "+key, resp)
	}
	return io.ReadAll(io.LimitReader(resp.Body, 25<<20)) // KV's value limit
}

func statusErr(op string, resp *http.Response) error {
	msg, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
	return fmt.Errorf("kv %s: %s: %s", op, resp.Status, bytes.TrimSpace(msg))
}
