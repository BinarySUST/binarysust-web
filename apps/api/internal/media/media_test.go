package media_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/sakib412/ekman"
	"github.com/sakib412/ekman/ekmantest"
	"github.com/sakib412/ekman/rbac"
	"github.com/sakib412/ekman/storage"

	"github.com/BinarySUST/binarysust-web/apps/api/internal/blog"
	"github.com/BinarySUST/binarysust-web/apps/api/internal/media"
	"github.com/BinarySUST/binarysust-web/apps/api/internal/people"
	"github.com/BinarySUST/binarysust-web/apps/api/internal/sponsors"
)

// fakeContents is a tiny in-memory GitHub Contents API.
type fakeContents struct {
	mu    sync.Mutex
	files map[string][]byte
}

func (f *fakeContents) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if r.Header.Get("Authorization") != "Bearer secret" {
		http.Error(w, "bad credentials", http.StatusUnauthorized)
		return
	}
	const prefix = "/repos/BinarySUST/binarysust-media/contents/"
	if !strings.HasPrefix(r.URL.Path, prefix) {
		http.NotFound(w, r)
		return
	}
	p := strings.TrimPrefix(r.URL.Path, prefix)
	var body struct{ Content, SHA, Branch, Message string }
	_ = json.NewDecoder(r.Body).Decode(&body)
	switch r.Method {
	case http.MethodPut:
		if _, exists := f.files[p]; exists {
			http.Error(w, "sha required", http.StatusUnprocessableEntity)
			return
		}
		data, err := base64.StdEncoding.DecodeString(body.Content)
		if err != nil || body.Branch != "main" {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		f.files[p] = data
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"content":{"sha":"abc"}}`))
	case http.MethodGet:
		if _, ok := f.files[p]; !ok {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(`{"sha":"abc"}`))
	case http.MethodDelete:
		if body.SHA != "abc" {
			http.Error(w, "sha mismatch", http.StatusConflict)
			return
		}
		delete(f.files, p)
		_, _ = w.Write([]byte(`{}`))
	}
}

func TestGitHubBucket(t *testing.T) {
	fake := &fakeContents{files: map[string][]byte{}}
	srv := httptest.NewServer(fake)
	defer srv.Close()
	b := &media.GitHubBucket{Token: "secret", Repository: "BinarySUST/binarysust-media", APIBaseURL: srv.URL,
		PublicBaseURL: "https://media.binarysust.org/"}
	ctx := context.Background()
	obj, err := b.Put(ctx, "avatars/ab cd.png", strings.NewReader("pngdata"), "image/png")
	if err != nil {
		t.Fatal(err)
	}
	if obj.URL != "https://media.binarysust.org/avatars/ab%20cd.png" || obj.Size != 7 || string(fake.files["avatars/ab cd.png"]) != "pngdata" {
		t.Fatalf("put: %+v %v", obj, fake.files)
	}
	if err := b.Delete(ctx, "avatars/ab cd.png"); err != nil || len(fake.files) != 0 {
		t.Fatalf("delete: %v %v", err, fake.files)
	}
	if err := b.Delete(ctx, "avatars/missing.png"); err != nil {
		t.Fatalf("deleting a missing key: %v", err)
	}
	if _, err := b.Put(ctx, "../escape", strings.NewReader("x"), "text/plain"); !errors.Is(err, storage.ErrInvalidKey) {
		t.Fatalf("key escape: %v", err)
	}
	b.Token = "wrong"
	if _, err := b.Put(ctx, "a/b.png", strings.NewReader("x"), "image/png"); err == nil || !strings.Contains(err.Error(), "401") {
		t.Fatalf("auth failure: %v", err)
	}
}

// png is the 8-byte PNG signature plus a little padding: enough for content sniffing.
var png = append([]byte("\x89PNG\r\n\x1a\n"), make([]byte, 64)...)

func multipartBody(t *testing.T, data []byte) (io.Reader, string) {
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, err := mw.CreateFormFile("file", "x.png")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = fw.Write(data)
	_ = mw.Close()
	return &buf, mw.FormDataContentType()
}

// upload posts a multipart file as user holding perms. ekmantest.Client.Do cannot set a
// multipart Content-Type, so this builds the request itself and passes the identity the way
// ekmantest's test server reads it (an ekman gap noted in the README).
func upload(t *testing.T, base, purpose string, data []byte, user string, perms ...string) (int, media.Object, string) {
	t.Helper()
	body, ct := multipartBody(t, data)
	req, err := http.NewRequest(http.MethodPost, base+"/api/v1/media?purpose="+purpose, body)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", ct)
	if user != "" {
		id, _ := json.Marshal(map[string]any{"UserID": user, "Perms": perms})
		req.Header.Set("Ekmantest-Identity", string(id))
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	var obj media.Object
	_ = json.Unmarshal(raw, &obj)
	return resp.StatusCode, obj, string(raw)
}

func TestUploadPurposes(t *testing.T) {
	dir := t.TempDir()
	// The modules that declare the purposes' permissions, without their routes.
	decl := ekman.Module{Name: "decl", Permissions: []rbac.Permission{people.EditOwn, blog.Write, sponsors.Manage}}
	c := ekmantest.New(t, ekman.Options{Baseline: []rbac.Permission{people.EditOwn, blog.Write}},
		media.New(storage.Dir(dir, "http://media.test")).Module(), decl)

	if code, _, _ := upload(t, c.URL(), "avatar", png, ""); code != 401 {
		t.Fatalf("anonymous: %d", code)
	}
	code, obj, raw := upload(t, c.URL(), "avatar", png, "member")
	if code != 201 || !strings.HasPrefix(obj.Key, "avatars/") || !strings.HasSuffix(obj.Key, ".png") || obj.URL != "http://media.test/"+obj.Key {
		t.Fatalf("member avatar: %d %s", code, raw)
	}
	if got, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(obj.Key))); err != nil || !bytes.Equal(got, png) {
		t.Fatalf("stored file: %v", err)
	}
	if code, _, _ := upload(t, c.URL(), "logo", png, "member"); code != 403 {
		t.Fatalf("logo needs sponsors.manage: %d", code)
	}
	if code, _, raw := upload(t, c.URL(), "logo", png, "sm", string(sponsors.Manage)); code != 201 {
		t.Fatalf("sponsor manager logo: %d %s", code, raw)
	}
	if code, _, _ := upload(t, c.URL(), "file", png, "member"); code != 403 {
		t.Fatalf("general uploads need media.upload: %d", code)
	}
	if code, _, _ := upload(t, c.URL(), "file", png, "editor", string(media.Upload)); code != 201 {
		t.Fatalf("media.upload: %d", code)
	}
	if code, _, _ := upload(t, c.URL(), "cv", png, "member"); code != 415 {
		t.Fatalf("a png is not a CV: %d", code)
	}
	if code, _, _ := upload(t, c.URL(), "nope", png, "member"); code != 422 {
		t.Fatalf("unknown purpose: %d", code)
	}
}
