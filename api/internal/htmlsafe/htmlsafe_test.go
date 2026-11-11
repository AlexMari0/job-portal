package htmlsafe_test

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/AlexMari0/job-portal/api/internal/htmlsafe"
)

func TestSanitize(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "keeps Quill formatting",
			in:   `<h2>Role</h2><p><strong>Bold</strong> <em>it</em> <u>u</u></p>`,
			want: `<h2>Role</h2><p><strong>Bold</strong> <em>it</em> <u>u</u></p>`,
		},
		{
			name: "keeps Quill 2 list markup",
			in:   `<ol><li data-list="bullet"><span class="ql-ui" contenteditable="false"></span>One</li></ol>`,
			want: `<ol><li data-list="bullet"><span class="ql-ui"></span>One</li></ol>`,
		},
		{
			name: "keeps Quill alignment classes only",
			in:   `<p class="ql-align-center evil">x</p>`,
			want: `<p>x</p>`,
		},
		{
			name: "removes script tags",
			in:   `<p>hi</p><script>alert(1)</script>`,
			want: `<p>hi</p>`,
		},
		{
			name: "removes event handler attributes",
			in:   `<p onclick="alert(1)">hi</p><img src=x onerror="alert(1)">`,
			want: `<p>hi</p>`,
		},
		{
			name: "removes javascript: links and hardens external ones",
			in:   `<a href="javascript:alert(1)">a</a><a href="https://example.com">b</a>`,
			want: `a<a href="https://example.com" rel="nofollow noopener" target="_blank">b</a>`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := htmlsafe.Sanitize(tt.in); got != tt.want {
				t.Errorf("Sanitize()\n got: %s\nwant: %s", got, tt.want)
			}
		})
	}
}

func TestSanitizeAllowsQuillAlignment(t *testing.T) {
	got := htmlsafe.Sanitize(`<p class="ql-align-center">x</p>`)
	if got != `<p class="ql-align-center">x</p>` {
		t.Errorf("got %s", got)
	}
}

func TestExcerpt(t *testing.T) {
	t.Run("plain text with spaces between blocks", func(t *testing.T) {
		got := htmlsafe.Excerpt(`<p>We are <strong>hiring</strong>.</p><ol><li>Go</li><li>SQL &amp; Mongo</li></ol>`, 150)
		want := "We are hiring. Go SQL & Mongo"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("truncates long text on a rune boundary with an ellipsis", func(t *testing.T) {
		in := "<p>" + strings.Repeat("é", 200) + "</p>"
		got := htmlsafe.Excerpt(in, 150)
		if n := utf8.RuneCountInString(got); n != 150 {
			t.Errorf("got %d runes, want 150", n)
		}
		if !strings.HasSuffix(got, "…") {
			t.Errorf("got %q, want trailing ellipsis", got)
		}
		if !utf8.ValidString(got) {
			t.Error("excerpt is not valid UTF-8")
		}
	})

	t.Run("drops script content", func(t *testing.T) {
		if got := htmlsafe.Excerpt(`<p>ok</p><script>alert(1)</script>`, 150); got != "ok" {
			t.Errorf("got %q, want %q", got, "ok")
		}
	})
}
