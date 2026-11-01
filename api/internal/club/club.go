// Package club is the composition root: THE module list and THE role table.
//
// Add a module: write internal/<name> (copy internal/sponsors), construct it in New and add its
// Module() to Modules. Give a role a permission: add it to Roles below; naming a permission no
// module declares stops the API at boot.
package club

import (
	"log/slog"
	"reflect"

	"github.com/sakib412/ekman"
	"github.com/sakib412/ekman/db"
	"github.com/sakib412/ekman/mail"
	"github.com/sakib412/ekman/rbac"
	"github.com/sakib412/ekman/storage"

	"github.com/BinarySUST/binarysust-web/api/internal/access"
	"github.com/BinarySUST/binarysust-web/api/internal/apidoc"
	"github.com/BinarySUST/binarysust-web/api/internal/blog"
	"github.com/BinarySUST/binarysust-web/api/internal/events"
	"github.com/BinarySUST/binarysust-web/api/internal/media"
	"github.com/BinarySUST/binarysust-web/api/internal/people"
	"github.com/BinarySUST/binarysust-web/api/internal/site"
	"github.com/BinarySUST/binarysust-web/api/internal/sponsors"
	"github.com/BinarySUST/binarysust-web/api/internal/talks"
)

// Roles are defined in code. Who holds them is data: position_roles (via current-term seats)
// and role_grants, managed in the dashboard's "Committee & roles".
var Roles = []rbac.Role{
	{Name: "admin", Title: "Administrator", Permissions: []rbac.Permission{rbac.All}},
	{Name: "blog_editor", Title: "Blog editor", Permissions: []rbac.Permission{blog.Review, media.Upload}},
	{Name: "event_manager", Title: "Event manager", Permissions: []rbac.Permission{events.Manage, events.CheckIn, media.Upload}},
	{Name: "people_manager", Title: "People manager", Permissions: []rbac.Permission{people.Manage, media.Upload}},
	{Name: "sponsor_manager", Title: "Sponsor manager", Permissions: []rbac.Permission{sponsors.Manage, media.Upload}},
	{Name: "talks_manager", Title: "Talks manager", Permissions: []rbac.Permission{talks.Manage, media.Upload}},
	{Name: "checkin_volunteer", Title: "Check-in volunteer", Permissions: []rbac.Permission{events.CheckIn}},
}

// OpenAPI is the API's OpenAPI document: ekman's, with enums filled in (internal/apidoc).
// Role names are only known here, so they are passed explicitly.
func OpenAPI(app *ekman.App) ([]byte, error) {
	return apidoc.Document(app, map[reflect.Type][]string{reflect.TypeFor[access.Role](): access.RoleNames(Roles)})
}

// Baseline is held by every signed-in user.
var Baseline = []rbac.Permission{people.EditOwn, blog.Write}

// Deps are the outside-world drivers. Zero values are fine for read-only tools (cmd/snapshot).
type Deps struct {
	Mail      mail.Sender    // nil: no emails
	Bucket    storage.Bucket // nil: uploads answer 503
	PublicURL string         // the public site's origin, for email links and /version.json
	Logger    *slog.Logger
}

// Services holds one instance of every module service.
type Services struct {
	Access   *access.Service
	People   *people.Service
	Events   *events.Service
	Blog     *blog.Service
	Talks    *talks.Service
	Sponsors *sponsors.Service
	Media    *media.Service
	Site     *site.Service
	Sources  site.Sources
}

// New builds every service over d.
func New(d *db.DB, deps Deps) *Services {
	s := &Services{Access: access.New(d, Roles)}
	s.People = people.New(d, s.Access)
	s.Events = events.New(d, s.People, deps.Mail, deps.PublicURL, deps.Logger)
	s.Blog = blog.New(d, s.Access)
	s.Talks = talks.New(d, s.People)
	s.Sponsors = sponsors.New(d)
	s.Media = media.New(deps.Bucket)
	s.Sources = site.Sources{DB: d, Access: s.Access, People: s.People, Events: s.Events,
		Blog: s.Blog, Talks: s.Talks, Sponsors: s.Sponsors}
	s.Site = site.New(s.Sources, deps.PublicURL)
	var declared []rbac.Permission
	for _, m := range s.Modules() {
		declared = append(declared, m.Permissions...)
	}
	s.Access.SetDeclared(declared)
	return s
}

// Modules is the single module list. Order is migration order: people first (persons, terms,
// positions), then access (accounts, grants and site_settings reference them), then the rest.
func (s *Services) Modules() []ekman.Module {
	return []ekman.Module{
		s.People.Module(),
		s.Access.Module(),
		s.Events.Module(),
		s.Blog.Module(),
		s.Talks.Module(),
		s.Sponsors.Module(),
		s.Media.Module(),
		s.Site.Module(),
	}
}
