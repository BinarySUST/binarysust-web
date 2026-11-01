// Package blog holds member-written posts behind an editor review gate (see Transition).
package blog

import (
	"context"
	"embed"
	"net/http"
	"strings"
	"time"
	"unicode"

	"github.com/sakib412/ekman"
	"github.com/sakib412/ekman/db"
	"github.com/sakib412/ekman/rbac"

	"github.com/BinarySUST/binarysust-web/apps/api/internal/access"
	"github.com/BinarySUST/binarysust-web/apps/api/internal/md"
	"github.com/BinarySUST/binarysust-web/apps/api/internal/pg"
	"github.com/BinarySUST/binarysust-web/apps/api/internal/republish"
)

//go:embed migrations/*.sql
var migrations embed.FS

// Permissions.
const (
	Write  rbac.Permission = "blog.write"  // baseline: write and submit your own posts
	Review rbac.Permission = "blog.review" // publish, request changes, unpublish, edit any post
)

// Service is the blog module.
type Service struct {
	db  *db.DB
	acc *access.Service
}

// New returns the blog service.
func New(d *db.DB, acc *access.Service) *Service { return &Service{db: d, acc: acc} }

// Module wires the blog into the app.
func (s *Service) Module() ekman.Module {
	return ekman.Module{
		Name:        "blog",
		Migrations:  migrations,
		Permissions: []rbac.Permission{Write, Review},
		Routes: func(r *ekman.Router) {
			ekman.Handle(r, "GET /posts", ekman.Requires(Write), s.list, ekman.Summary("Your posts; every post for reviewers"))
			ekman.Handle(r, "POST /posts", ekman.Requires(Write), s.create, ekman.Summary("Start a draft"))
			ekman.Handle(r, "GET /posts/{id}", ekman.Requires(Write), s.get)
			ekman.Handle(r, "PUT /posts/{id}", ekman.Requires(Write), s.update)
			ekman.Handle(r, "DELETE /posts/{id}", ekman.Requires(Write), s.remove)
			ekman.Handle(r, "POST /posts/{id}/transitions", ekman.Requires(Write), s.transition,
				ekman.Status(200), ekman.Summary("submit, withdraw, request_changes, publish or unpublish"))
			ekman.Handle(r, "GET /posts/{id}/preview", ekman.Requires(Write), s.preview,
				ekman.Summary("The saved post exactly as the public site will render it"))
			// The editor's live preview: renders unsaved markdown, writes nothing, never republishes.
			ekman.Handle(r, "POST /posts/preview", ekman.Requires(Write), s.renderPreview,
				ekman.Status(http.StatusOK), republish.Skip(),
				ekman.Summary("Render markdown with the public site's renderer (no save)"))
		},
	}
}

// PublicPost is a published post on the public site.
type PublicPost struct {
	Slug           string    `json:"slug"`
	Title          string    `json:"title"`
	Excerpt        string    `json:"excerpt"`
	BodyHTML       string    `json:"bodyHtml"`
	CoverURL       *string   `json:"coverUrl"`
	Author         string    `json:"author"` // person handle
	Tags           []string  `json:"tags"`
	PublishedAt    time.Time `json:"publishedAt"`
	ReadingMinutes int       `json:"readingMinutes"`
}

// PostRow is a line in the dashboard list.
type PostRow struct {
	ID         string    `json:"id"`
	Slug       string    `json:"slug"`
	Title      string    `json:"title"`
	State      State     `json:"state"`
	Author     string    `json:"author"` // person handle
	AuthorName string    `json:"authorName"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

// Post is the editor view.
type Post struct {
	PostRow
	Excerpt      string     `json:"excerpt"`
	BodyMarkdown string     `json:"bodyMarkdown"`
	CoverURL     *string    `json:"coverUrl"`
	Tags         []string   `json:"tags"`
	ReviewNote   string     `json:"reviewNote"`
	PublishedAt  *time.Time `json:"publishedAt"`
	CanEdit      bool       `json:"canEdit"`
}

type loaded struct {
	Post
	authorID string
}

const selectPost = `
	SELECT p.id, p.slug, p.title, p.state, a.handle, a.name, p.updated_at, p.excerpt, p.body_md, p.cover_url,
	       coalesce((SELECT array_agg(tag ORDER BY tag) FROM post_tags WHERE post_id = p.id), '{}'),
	       p.review_note, p.first_published_at, p.author_id
	  FROM posts p JOIN persons a ON a.id = p.author_id`

func (s *Service) load(ctx context.Context, id string, lock bool) (loaded, error) {
	var l loaded
	if !pg.ValidID(id) {
		return l, pg.NotFound("post")
	}
	q := selectPost + ` WHERE p.id = $1`
	if lock {
		q += ` FOR UPDATE OF p`
	}
	err := s.db.Q(ctx).QueryRowContext(ctx, q, id).Scan(&l.ID, &l.Slug, &l.Title, &l.State, &l.Author, &l.AuthorName,
		&l.UpdatedAt, &l.Excerpt, &l.BodyMarkdown, &l.CoverURL, pg.Array(&l.Tags), &l.ReviewNote, &l.PublishedAt, &l.authorID)
	if err != nil {
		return l, err
	}
	l.UpdatedAt = l.UpdatedAt.UTC()
	if l.PublishedAt != nil {
		t := l.PublishedAt.UTC()
		l.PublishedAt = &t
	}
	if l.Tags == nil {
		l.Tags = []string{}
	}
	return l, nil
}

// actor works out who the caller is to the post. Non-members (no person) can still review.
func (s *Service) actor(ctx context.Context, c ekman.Caller, authorID string) Actor {
	pid, err := s.acc.PersonOf(ctx, c.UserID())
	return Actor{IsAuthor: err == nil && pid == authorID, CanReview: c.Can(Review)}
}

// view loads a post the caller may see (its author or a reviewer); others get 404.
func (s *Service) view(ctx context.Context, c ekman.Caller, id string, lock bool) (loaded, Actor, error) {
	l, err := s.load(ctx, id, lock)
	if err != nil {
		return l, Actor{}, err
	}
	who := s.actor(ctx, c, l.authorID)
	if !who.IsAuthor && !who.CanReview {
		return l, who, pg.NotFound("post")
	}
	l.CanEdit = CanEdit(l.State, who)
	return l, who, nil
}

// ListIn filters the list.
type ListIn struct {
	State *State `query:"state"`
}

func (s *Service) list(ctx context.Context, c ekman.Caller, in *ListIn) ([]PostRow, error) {
	var author *string
	if !c.Can(Review) {
		pid, err := s.acc.PersonOf(ctx, c.UserID())
		if err != nil {
			return []PostRow{}, nil // not a member yet: no posts
		}
		author = &pid
	}
	rows, err := s.db.Q(ctx).QueryContext(ctx, `
		SELECT p.id, p.slug, p.title, p.state, a.handle, a.name, p.updated_at
		  FROM posts p JOIN persons a ON a.id = p.author_id
		 WHERE ($1::uuid IS NULL OR p.author_id = $1::uuid) AND ($2::text IS NULL OR p.state = $2::text)
		 ORDER BY p.updated_at DESC`, author, in.State)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []PostRow
	for rows.Next() {
		var r PostRow
		if err := rows.Scan(&r.ID, &r.Slug, &r.Title, &r.State, &r.Author, &r.AuthorName, &r.UpdatedAt); err != nil {
			return nil, err
		}
		r.UpdatedAt = r.UpdatedAt.UTC()
		out = append(out, r)
	}
	return out, rows.Err()
}

// PostIn is the editable content of a post.
type PostIn struct {
	Slug         string   `json:"slug"` // empty: derived from the title
	Title        string   `json:"title"`
	Excerpt      string   `json:"excerpt"`
	BodyMarkdown string   `json:"bodyMarkdown"`
	CoverURL     *string  `json:"coverUrl"`
	Tags         []string `json:"tags"`
}

// Validate implements ekman.Validator.
func (in *PostIn) Validate(v *ekman.Validation) {
	in.Title = strings.TrimSpace(in.Title)
	v.Required("title", in.Title)
	v.MaxLen("title", in.Title, 200)
	if strings.TrimSpace(in.Slug) == "" {
		in.Slug = Slugify(in.Title)
	}
	in.Slug = strings.ToLower(strings.TrimSpace(in.Slug))
	v.Check(in.Slug == Slugify(in.Slug) && len(in.Slug) >= 2, "slug", "use lowercase letters, digits and dashes")
	v.MaxLen("slug", in.Slug, 100)
	v.MaxLen("excerpt", in.Excerpt, 400)
	v.MaxLen("bodyMarkdown", in.BodyMarkdown, 100_000)
	if in.CoverURL != nil {
		v.URL("coverUrl", *in.CoverURL)
	}
	v.Check(len(in.Tags) <= 8, "tags", "at most 8 tags")
	for i, t := range in.Tags {
		in.Tags[i] = Slugify(t)
		v.Check(in.Tags[i] != "" && len(in.Tags[i]) <= 30, "tags", "tags are short words like go or icpc")
	}
}

func (s *Service) create(ctx context.Context, c ekman.Caller, in *PostIn) (Post, error) {
	pid, err := s.acc.PersonOf(ctx, c.UserID())
	if err != nil {
		return Post{}, err
	}
	var id string
	err = s.db.Tx(ctx, func(ctx context.Context) error {
		if err := s.db.Q(ctx).QueryRowContext(ctx, `
			INSERT INTO posts (slug, author_id, title, excerpt, body_md, cover_url) VALUES ($1, $2, $3, $4, $5, $6)
			RETURNING id`, in.Slug, pid, in.Title, in.Excerpt, in.BodyMarkdown, in.CoverURL).Scan(&id); err != nil {
			return pg.Friendly(err, "a post with that slug exists")
		}
		return s.setTags(ctx, id, in.Tags)
	})
	if err != nil {
		return Post{}, err
	}
	l, _, err := s.view(ctx, c, id, false)
	return l.Post, err
}

func (s *Service) setTags(ctx context.Context, id string, tags []string) error {
	if _, err := s.db.Q(ctx).ExecContext(ctx, `DELETE FROM post_tags WHERE post_id = $1`, id); err != nil {
		return err
	}
	for _, t := range tags {
		if _, err := s.db.Q(ctx).ExecContext(ctx,
			`INSERT INTO post_tags (post_id, tag) VALUES ($1, $2) ON CONFLICT DO NOTHING`, id, t); err != nil {
			return err
		}
	}
	return nil
}

// IDIn is a path id.
type IDIn struct {
	ID string `path:"id"`
}

func (s *Service) get(ctx context.Context, c ekman.Caller, in *IDIn) (Post, error) {
	l, _, err := s.view(ctx, c, in.ID, false)
	return l.Post, err
}

// UpdateIn edits a post.
type UpdateIn struct {
	ID string `path:"id"`
	PostIn
}

func (s *Service) update(ctx context.Context, c ekman.Caller, in *UpdateIn) (Post, error) {
	err := s.db.Tx(ctx, func(ctx context.Context) error {
		l, who, err := s.view(ctx, c, in.ID, true)
		if err != nil {
			return err
		}
		if !CanEdit(l.State, who) {
			return ekman.Conflict("not_editable", "this post is "+string(l.State)+"; only a reviewer can edit it now")
		}
		if _, err := s.db.Q(ctx).ExecContext(ctx, `
			UPDATE posts SET slug = $2, title = $3, excerpt = $4, body_md = $5, cover_url = $6, updated_at = now()
			 WHERE id = $1`, in.ID, in.Slug, in.Title, in.Excerpt, in.BodyMarkdown, in.CoverURL); err != nil {
			return pg.Friendly(err, "a post with that slug exists")
		}
		return s.setTags(ctx, in.ID, in.Tags)
	})
	if err != nil {
		return Post{}, err
	}
	return s.get(ctx, c, &IDIn{ID: in.ID})
}

func (s *Service) remove(ctx context.Context, c ekman.Caller, in *IDIn) (ekman.NoContent, error) {
	return ekman.NoContent{}, s.db.Tx(ctx, func(ctx context.Context) error {
		l, who, err := s.view(ctx, c, in.ID, true)
		if err != nil {
			return err
		}
		if !who.CanReview && l.State != Draft {
			return ekman.Conflict("not_deletable", "only drafts can be deleted by their author")
		}
		_, err = s.db.Q(ctx).ExecContext(ctx, `DELETE FROM posts WHERE id = $1`, in.ID)
		return err
	})
}

// TransitionIn applies an action.
type TransitionIn struct {
	ID     string `path:"id"`
	Action Action `json:"action"`
	Note   string `json:"note"` // shown to the author with request_changes
}

// Validate implements ekman.Validator.
func (in *TransitionIn) Validate(v *ekman.Validation) {
	v.Required("action", string(in.Action))
	v.OneOf("action", string(in.Action), Actions()...)
	v.Check(in.Action != RequestChanges || strings.TrimSpace(in.Note) != "", "note", "say what should change")
	v.MaxLen("note", in.Note, 2000)
}

func (s *Service) transition(ctx context.Context, c ekman.Caller, in *TransitionIn) (Post, error) {
	err := s.db.Tx(ctx, func(ctx context.Context) error {
		l, who, err := s.view(ctx, c, in.ID, true)
		if err != nil {
			return err
		}
		to, err := Transition(l.State, in.Action, who)
		if err != nil || to == l.State {
			return err
		}
		note := l.ReviewNote
		switch in.Action {
		case RequestChanges:
			note = in.Note
		case Publish:
			note = ""
		}
		_, err = s.db.Q(ctx).ExecContext(ctx, `
			UPDATE posts SET state = $2, review_note = $3, updated_at = now(),
			       first_published_at = CASE WHEN $2 = 'published' THEN coalesce(first_published_at, now())
			                                 ELSE first_published_at END
			 WHERE id = $1`, in.ID, to, note)
		return err
	})
	if err != nil {
		return Post{}, err
	}
	return s.get(ctx, c, &IDIn{ID: in.ID})
}

func (s *Service) preview(ctx context.Context, c ekman.Caller, in *IDIn) (PublicPost, error) {
	l, _, err := s.view(ctx, c, in.ID, false)
	if err != nil {
		return PublicPost{}, err
	}
	at := time.Now().UTC()
	if l.PublishedAt != nil {
		at = *l.PublishedAt
	}
	return toPublic(l.Post, at), nil
}

// PreviewIn is markdown to render.
type PreviewIn struct {
	Markdown string `json:"markdown"`
}

// Validate implements ekman.Validator.
func (in *PreviewIn) Validate(v *ekman.Validation) {
	v.MaxLen("markdown", in.Markdown, 100_000) // the same limit as a post body
}

// Preview is rendered markdown.
type Preview struct {
	HTML           string `json:"html"` // sanitized by internal/md, like everything public
	ReadingMinutes int    `json:"readingMinutes"`
}

func (s *Service) renderPreview(_ context.Context, _ ekman.Caller, in *PreviewIn) (Preview, error) {
	return Preview{HTML: md.Render(in.Markdown), ReadingMinutes: md.ReadingMinutes(in.Markdown)}, nil
}

func toPublic(p Post, at time.Time) PublicPost {
	return PublicPost{Slug: p.Slug, Title: p.Title, Excerpt: p.Excerpt, BodyHTML: md.Render(p.BodyMarkdown),
		CoverURL: p.CoverURL, Author: p.Author, Tags: p.Tags, PublishedAt: at,
		ReadingMinutes: md.ReadingMinutes(p.BodyMarkdown)}
}

// Published is the read side for site.Build: published posts, newest first.
func (s *Service) Published(ctx context.Context) ([]PublicPost, error) {
	rows, err := s.db.Q(ctx).QueryContext(ctx, selectPost+` WHERE p.state = 'published' ORDER BY p.first_published_at DESC, p.slug`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []PublicPost{}
	for rows.Next() {
		var l loaded
		if err := rows.Scan(&l.ID, &l.Slug, &l.Title, &l.State, &l.Author, &l.AuthorName, &l.UpdatedAt, &l.Excerpt,
			&l.BodyMarkdown, &l.CoverURL, pg.Array(&l.Tags), &l.ReviewNote, &l.PublishedAt, &l.authorID); err != nil {
			return nil, err
		}
		if l.Tags == nil {
			l.Tags = []string{}
		}
		out = append(out, toPublic(l.Post, l.PublishedAt.UTC()))
	}
	return out, rows.Err()
}

// Slugify lowercases s and joins its ASCII letters and digits with single dashes.
func Slugify(s string) string {
	var b strings.Builder
	dash := false
	for _, r := range strings.ToLower(s) {
		switch {
		case r < unicode.MaxASCII && (unicode.IsLetter(r) || unicode.IsDigit(r)):
			if dash && b.Len() > 0 {
				b.WriteByte('-')
			}
			b.WriteRune(r)
			dash = false
		default:
			dash = true
		}
	}
	return b.String()
}
