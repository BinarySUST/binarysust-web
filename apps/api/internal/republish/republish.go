// Package republish keeps the static site in step with the database, by default.
//
// After every 2xx response to a non-GET request, Middleware records the change
// (site_settings.content_changed_at = now()) and asks GitHub to rebuild the site with a
// repository_dispatch, unless the route carries Skip(). Default-on is deliberate: forgetting
// Skip costs one no-op CI run (the build skips the deploy when the snapshot hash is unchanged);
// forgetting an opt-in would leave the public site silently stale.
package republish

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/sakib412/ekman"
)

const skipTag = "republish:skip"

// Skip marks a route whose writes never change public content (RSVPs, check-ins, uploads,
// role grants). Pass it as a route option: ekman.Handle(r, "POST /x", access, h, republish.Skip()).
func Skip() ekman.RouteOption { return ekman.Tag(skipTag) }

// ShouldPublish is the whole rule: a successful write on a route that did not opt out.
// ekman's own routes (sign-in, sign-out) never publish.
func ShouldPublish(method string, status int, ri ekman.RouteInfo, matched bool) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return false
	}
	if status < 200 || status > 299 || !matched {
		return false
	}
	return ri.Module != "ekman" && !ri.HasTag(skipTag)
}

// Recorder stores "content changed now" (access.Service.TouchContent).
type Recorder interface {
	TouchContent(ctx context.Context) (time.Time, error)
}

// Notifier asks CI to rebuild the site.
type Notifier interface {
	Notify(ctx context.Context, reason string) error
}

// Middleware runs after the handler, inside the same request, so a free instance that sleeps
// right after responding cannot drop the notification. Failures are logged, never returned:
// the user's write already succeeded, and the next write or the nightly build catches up.
func Middleware(rec Recorder, n Notifier, log *slog.Logger) ekman.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sw := &statusWriter{ResponseWriter: w}
			next.ServeHTTP(sw, r)
			ri, ok := ekman.RouteFrom(r.Context())
			if !ShouldPublish(r.Method, sw.code(), ri, ok) {
				return
			}
			// The request context may already be canceled by a client that hung up.
			ctx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), 8*time.Second)
			defer cancel()
			reason := r.Method + " " + ri.Pattern
			if _, err := rec.TouchContent(ctx); err != nil {
				log.ErrorContext(ctx, "republish: record content change", "err", err.Error(), "route", reason, "request_id", ekman.RequestID(ctx))
			}
			if err := n.Notify(ctx, reason); err != nil {
				log.ErrorContext(ctx, "republish: dispatch failed", "err", err.Error(), "route", reason, "request_id", ekman.RequestID(ctx))
			}
		})
	}
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	if w.status == 0 {
		w.status = code
	}
	w.ResponseWriter.WriteHeader(code)
}

func (w *statusWriter) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	return w.ResponseWriter.Write(b)
}

func (w *statusWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }

func (w *statusWriter) code() int {
	if w.status == 0 {
		return http.StatusOK
	}
	return w.status
}

// GitHub sends repository_dispatch {event_type: "publish"} to Repository ("owner/name").
// The workflow's concurrency group merges bursts into one running and one pending build.
type GitHub struct {
	Token      string
	Repository string
	BaseURL    string       // default https://api.github.com
	Client     *http.Client // default: 5 s timeout
}

// Notify implements Notifier.
func (g *GitHub) Notify(ctx context.Context, reason string) error {
	base := g.BaseURL
	if base == "" {
		base = "https://api.github.com"
	}
	client := g.Client
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}
	body, _ := json.Marshal(map[string]any{
		"event_type":     "publish",
		"client_payload": map[string]string{"reason": reason},
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		strings.TrimSuffix(base, "/")+"/repos/"+g.Repository+"/dispatches", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+g.Token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		msg, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("github dispatch: %s: %s", resp.Status, bytes.TrimSpace(msg))
	}
	return nil
}

// Log is the Notifier used when no GitHub token is configured (local development).
type Log struct{ Logger *slog.Logger }

// Notify implements Notifier.
func (l Log) Notify(ctx context.Context, reason string) error {
	l.Logger.InfoContext(ctx, "republish: content changed (no GITHUB_DISPATCH_TOKEN; not dispatching)", "route", reason)
	return nil
}
