package md

import (
	"strings"
	"testing"
)

func TestRenderSanitizes(t *testing.T) {
	out := Render("# Hi\n\n<script>alert(1)</script>\n\n[x](javascript:alert(1)) **bold**")
	if strings.Contains(out, "<script") || strings.Contains(out, "javascript:") {
		t.Fatalf("unsafe output: %s", out)
	}
	if !strings.Contains(out, "<strong>bold</strong>") || !strings.Contains(out, "<h1") {
		t.Fatalf("markdown not rendered: %s", out)
	}
}

func TestRenderEmpty(t *testing.T) {
	if got := Render("  \n"); got != "" {
		t.Fatalf("got %q", got)
	}
}

func TestReadingMinutes(t *testing.T) {
	if got := ReadingMinutes("one two"); got != 1 {
		t.Fatalf("short: %d", got)
	}
	if got := ReadingMinutes(strings.Repeat("word ", 441)); got != 3 {
		t.Fatalf("441 words: %d", got)
	}
}
