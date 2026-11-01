// Package media is the one upload endpoint. Each purpose maps to the permission it needs and
// to size and type rules; ekman's storage.Receive does the rest (size limit, sniffed type,
// random key). Development stores files in MEDIA_DIR (storage.Dir, served by Caddy); production
// commits them to the binarysust-media GitHub repository (GitHubBucket), deployed as its own
// Pages project so media never counts against the site's per-deploy file limit.
package media

import (
	"context"
	"errors"
	"maps"
	"net/http"
	"slices"
	"strings"

	"github.com/sakib412/ekman"
	"github.com/sakib412/ekman/rbac"
	"github.com/sakib412/ekman/storage"

	"github.com/BinarySUST/binarysust-web/apps/api/internal/blog"
	"github.com/BinarySUST/binarysust-web/apps/api/internal/events"
	"github.com/BinarySUST/binarysust-web/apps/api/internal/people"
	"github.com/BinarySUST/binarysust-web/apps/api/internal/republish"
	"github.com/BinarySUST/binarysust-web/apps/api/internal/sponsors"
	"github.com/BinarySUST/binarysust-web/apps/api/internal/talks"
)

// Upload allows general-purpose uploads (the dashboard's Media section).
const Upload rbac.Permission = "media.upload"

// Purpose is one kind of upload.
type Purpose struct {
	Requires rbac.Permission
	Policy   storage.Policy
}

var (
	images = []string{"image/jpeg", "image/png", "image/webp", "image/gif"}
	pdf    = []string{"application/pdf"}
)

// Purposes is the single table of upload kinds: adding one is one line.
var Purposes = map[string]Purpose{
	"avatar": {people.EditOwn, storage.Policy{MaxBytes: 1 << 20, Types: images, Prefix: "avatars/"}},
	"cv":     {people.EditOwn, storage.Policy{MaxBytes: 4 << 20, Types: pdf, Prefix: "cv/"}},
	"post":   {blog.Write, storage.Policy{MaxBytes: 2 << 20, Types: images, Prefix: "posts/"}},
	"event":  {events.Manage, storage.Policy{MaxBytes: 3 << 20, Types: images, Prefix: "events/"}},
	"talk":   {talks.Manage, storage.Policy{MaxBytes: 2 << 20, Types: images, Prefix: "talks/"}},
	"logo":   {sponsors.Manage, storage.Policy{MaxBytes: 1 << 20, Types: images, Prefix: "logos/"}},
	"file":   {Upload, storage.Policy{MaxBytes: 5 << 20, Types: append(slices.Clone(images), pdf...), Prefix: "files/"}},
}

// Service is the media module.
type Service struct{ bucket storage.Bucket }

// New returns the media service storing into b.
func New(b storage.Bucket) *Service { return &Service{bucket: b} }

// Module wires media into the app.
func (s *Service) Module() ekman.Module {
	return ekman.Module{
		Name:        "media",
		Permissions: []rbac.Permission{Upload},
		Routes: func(r *ekman.Router) {
			// SignedIn at the route; the purpose decides the permission (see Purposes).
			// Skip: an upload alone changes nothing public; the save that uses its URL does.
			// ekman.Handle (not Raw) so the purpose and the result are typed in OpenAPI; the
			// multipart body is read from the request withRequest keeps (an ekman v0.1 gap).
			ekman.Handle(r, "POST /media", ekman.SignedIn(), s.upload, ekman.Use(withRequest), republish.Skip(),
				ekman.Summary("Upload one file (multipart field \"file\")"))
		},
	}
}

// PurposeName names an entry of Purposes.
type PurposeName string

// Enum lists the purposes (OpenAPI and the web's types are generated from it).
func (PurposeName) Enum() []string { return slices.Sorted(maps.Keys(Purposes)) }

// MultipartFile names the form field holding the file, for the OpenAPI request body.
func (UploadIn) MultipartFile() string { return "file" }

// UploadIn picks the purpose; the file itself is the multipart field "file".
type UploadIn struct {
	Purpose PurposeName `query:"purpose"` // default "file"
}

// Validate implements ekman.Validator.
func (in *UploadIn) Validate(v *ekman.Validation) {
	if in.Purpose == "" {
		in.Purpose = "file"
	}
	_, ok := Purposes[string(in.Purpose)]
	v.Check(ok, "purpose", "one of "+strings.Join(PurposeName("").Enum(), ", "))
}

// Object is the upload result.
type Object struct {
	Key         string `json:"key"`
	URL         string `json:"url"`
	ContentType string `json:"contentType"`
	Size        int64  `json:"size"`
}

type requestKey struct{}

// withRequest keeps the request in the context, so a typed handler can read a multipart body.
func withRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), requestKey{}, r)))
	})
}

func (s *Service) upload(ctx context.Context, c ekman.Caller, in *UploadIn) (Object, error) {
	if err := c.Require(Purposes[string(in.Purpose)].Requires); err != nil {
		return Object{}, err
	}
	if s.bucket == nil {
		return Object{}, &ekman.Error{Status: http.StatusServiceUnavailable, Code: "no_storage", Message: "uploads are not configured"}
	}
	r, ok := ctx.Value(requestKey{}).(*http.Request)
	if !ok {
		return Object{}, errors.New("media: request missing from context")
	}
	obj, err := storage.Receive(r, "file", s.bucket, Purposes[string(in.Purpose)].Policy)
	if err != nil {
		return Object{}, err
	}
	return Object{Key: obj.Key, URL: obj.URL, ContentType: obj.ContentType, Size: obj.Size}, nil
}
