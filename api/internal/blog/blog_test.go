package blog_test

import (
	"strings"
	"testing"

	"github.com/BinarySUST/binarysust-web/api/internal/blog"
	"github.com/BinarySUST/binarysust-web/api/internal/testapp"
)

func TestReviewWorkflow(t *testing.T) {
	a := testapp.New(t)
	a.Exec(t, `INSERT INTO persons (handle, name, email) VALUES ('ann', 'Ann', 'ann@example.com')`)
	author := a.As(a.Account(t, "ann@example.com"), blog.Write)
	editor := a.As(a.Account(t, "ed@example.com"), blog.Write, blog.Review)
	stranger := a.As(a.Account(t, "zed@example.com"), blog.Write)

	if r := stranger.Do("POST", "/api/v1/posts", blog.PostIn{Title: "Hi"}); r.Status() != 403 || r.ErrorCode() != "not_a_member" {
		t.Fatalf("non-member draft: %d %s", r.Status(), r.ErrorCode())
	}
	r := author.Do("POST", "/api/v1/posts", blog.PostIn{Title: "How this site is built!", BodyMarkdown: "# Hello\n\n<script>x</script>", Tags: []string{"Go", "web"}})
	if r.Status() != 201 {
		t.Fatalf("create: %d %s", r.Status(), r.Body())
	}
	var p blog.Post
	r.JSON(&p)
	if p.Slug != "how-this-site-is-built" || p.State != blog.Draft || !p.CanEdit || len(p.Tags) != 2 || p.Tags[0] != "go" {
		t.Fatalf("draft: %+v", p)
	}
	path := "/api/v1/posts/" + p.ID
	if r := stranger.Do("GET", path, nil); r.Status() != 404 {
		t.Fatalf("others' drafts are hidden: %d", r.Status())
	}
	do := func(who string, action blog.Action, note string, want int) {
		t.Helper()
		c := author
		if who == "editor" {
			c = editor
		}
		r := c.Do("POST", path+"/transitions", blog.TransitionIn{Action: action, Note: note})
		if r.Status() != want {
			t.Fatalf("%s %s: %d %s", who, action, r.Status(), r.Body())
		}
	}
	do("editor", blog.Publish, "", 409) // not submitted yet
	do("author", blog.Submit, "", 200)
	if r := author.Do("PUT", path, blog.PostIn{Title: "Changed"}); r.Status() != 409 {
		t.Fatalf("author edits while in review: %d", r.Status())
	}
	do("editor", blog.RequestChanges, "", 422) // a note is required
	do("editor", blog.RequestChanges, "Add a diagram", 200)
	do("author", blog.Submit, "", 200)
	do("author", blog.Publish, "", 403)
	do("editor", blog.Publish, "", 200)
	do("editor", blog.Publish, "", 200) // retry is a no-op

	pub, err := a.Svc.Blog.Published(t.Context())
	if err != nil || len(pub) != 1 || pub[0].Author != "ann" || pub[0].ReadingMinutes != 1 {
		t.Fatalf("published: %v %+v", err, pub)
	}
	first := pub[0].PublishedAt
	if got := pub[0].BodyHTML; got != "<h1 id=\"hello\">Hello</h1>" && got != "<h1>Hello</h1>" {
		t.Fatalf("body html: %q", got)
	}

	do("editor", blog.Unpublish, "", 200)
	if pub, _ := a.Svc.Blog.Published(t.Context()); len(pub) != 0 {
		t.Fatal("unpublished posts leave the site")
	}
	do("editor", blog.Publish, "", 200)
	pub, _ = a.Svc.Blog.Published(t.Context())
	if !pub[0].PublishedAt.Equal(first) {
		t.Fatalf("first_published_at changed: %v -> %v", first, pub[0].PublishedAt)
	}

	var rows []blog.PostRow
	editor.Do("GET", "/api/v1/posts?state=published", nil).JSON(&rows)
	if len(rows) != 1 {
		t.Fatalf("editor list: %+v", rows)
	}
	stranger.Do("GET", "/api/v1/posts", nil).JSON(&rows)
	if len(rows) != 0 {
		t.Fatalf("stranger list: %+v", rows)
	}
	if r := author.Do("DELETE", path, nil); r.Status() != 409 {
		t.Fatalf("author deletes a published post: %d", r.Status())
	}
	if r := editor.Do("DELETE", path, nil); r.Status() != 204 {
		t.Fatalf("editor delete: %d", r.Status())
	}
}

func TestMarkdownPreview(t *testing.T) {
	a := testapp.New(t)
	r := a.As("anyone", blog.Write).Do("POST", "/api/v1/posts/preview", blog.PreviewIn{Markdown: "# Hi\n\n<script>alert(1)</script>\n\n**bold**"})
	if r.Status() != 200 {
		t.Fatalf("preview: %d %s", r.Status(), r.Body())
	}
	var p blog.Preview
	r.JSON(&p)
	if !strings.Contains(p.HTML, "<h1") || !strings.Contains(p.HTML, "<strong>bold</strong>") || strings.Contains(p.HTML, "<script") || p.ReadingMinutes != 1 {
		t.Fatalf("preview: %+v", p)
	}
	if r := a.Do("POST", "/api/v1/posts/preview", blog.PreviewIn{Markdown: "x"}); r.Status() != 401 {
		t.Fatalf("needs a signed-in writer: %d", r.Status())
	}
}
