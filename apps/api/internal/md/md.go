// Package md is the one markdown renderer. Everything public that started as markdown (bios,
// event bodies, posts, talk summaries) goes through Render, both when the snapshot is built and
// when the dashboard previews a draft, so what an author previews is exactly what gets published.
package md

import (
	"bytes"
	"strings"
	"unicode"

	"github.com/microcosm-cc/bluemonday"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/renderer/html"
)

var (
	gm = goldmark.New(
		goldmark.WithExtensions(extension.GFM, extension.Typographer),
		// Raw HTML in markdown is dropped by goldmark; bluemonday below is the real guard.
		goldmark.WithRendererOptions(html.WithHardWraps()),
	)
	policy = newPolicy()
)

func newPolicy() *bluemonday.Policy {
	p := bluemonday.UGCPolicy()
	p.RequireNoFollowOnLinks(false)
	p.RequireNoReferrerOnLinks(true)
	p.AddTargetBlankToFullyQualifiedLinks(true)
	p.AllowAttrs("class").Matching(bluemonday.SpaceSeparatedTokens).OnElements("code", "pre")
	return p
}

// Render turns markdown into sanitized HTML. Empty input renders to "".
func Render(src string) string {
	if strings.TrimSpace(src) == "" {
		return ""
	}
	var buf bytes.Buffer
	if err := gm.Convert([]byte(src), &buf); err != nil {
		// goldmark only fails on writer errors, which a bytes.Buffer never returns.
		return policy.Sanitize("<p>" + src + "</p>")
	}
	return strings.TrimSpace(policy.Sanitize(buf.String()))
}

// ReadingMinutes estimates reading time at 220 words per minute, at least 1.
func ReadingMinutes(src string) int {
	words := len(strings.FieldsFunc(src, func(r rune) bool { return unicode.IsSpace(r) }))
	m := (words + 219) / 220
	if m < 1 {
		return 1
	}
	return m
}
