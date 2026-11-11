// Package htmlsafe cleans Job description HTML (written by the Quill editor)
// so it is safe to render, and derives plain-text excerpts from it.
package htmlsafe

import (
	"html"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/microcosm-cc/bluemonday"
)

// policy allows only the markup Quill 2's default toolbar produces:
// headings, bold/italic/underline, links, lists, and alignment/indent classes.
var policy = func() *bluemonday.Policy {
	p := bluemonday.NewPolicy()
	p.AllowElements("p", "br", "strong", "b", "em", "i", "u", "s",
		"h1", "h2", "h3", "h4", "h5", "h6", "ol", "ul", "li", "span",
		"blockquote", "pre", "code")

	// Quill classes such as "ql-align-center ql-indent-1"; anything else drops the attribute.
	p.AllowAttrs("class").
		Matching(regexp.MustCompile(`^ql-[a-z0-9-]+( ql-[a-z0-9-]+)*$`)).
		Globally()
	// Quill 2 renders every list as <ol> and marks the kind on each <li>.
	p.AllowAttrs("data-list").
		Matching(regexp.MustCompile(`^(bullet|ordered|checked|unchecked)$`)).
		OnElements("li")

	p.AllowAttrs("href").OnElements("a")
	p.AllowStandardURLs() // http, https, mailto only; drops javascript: URLs
	p.RequireNoFollowOnLinks(true)
	p.AddTargetBlankToFullyQualifiedLinks(true)
	return p
}()

var strip = bluemonday.StrictPolicy()

// Sanitize returns html with everything outside the allowlist removed.
func Sanitize(s string) string {
	return policy.Sanitize(s)
}

// blockEnd puts a space after block-level closing tags so that
// "<p>a</p><p>b</p>" becomes "a b", not "ab", once tags are stripped.
var blockEnd = regexp.MustCompile(`(?i)(</(p|li|h[1-6]|blockquote|pre)>|<br\s*/?>)`)

// Excerpt returns the plain text of s, whitespace collapsed, cut to at most
// max runes (ending in "…" when cut).
func Excerpt(s string, max int) string {
	text := strip.Sanitize(blockEnd.ReplaceAllString(s, "$1 "))
	text = strings.Join(strings.Fields(html.UnescapeString(text)), " ")

	if utf8.RuneCountInString(text) <= max {
		return text
	}
	runes := []rune(text)
	return strings.TrimRight(string(runes[:max-1]), " ") + "…"
}
