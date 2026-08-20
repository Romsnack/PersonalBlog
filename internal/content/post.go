// Package content reads Markdown files from disk and turns them into Posts.
package content

import (
	"bytes"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/yuin/goldmark/parser"
	"go.abhg.dev/goldmark/frontmatter"
)

// Post is one rendered Markdown file, ready to hand to a template.
type Post struct {
	Title       string
	Date        time.Time
	Tags        []string
	Draft       bool
	Summary     string
	Slug        string
	Path        string // site-relative, e.g. "/posts/hello-world/"
	HTML        template.HTML
	ReadingTime int // whole minutes, minimum 1
	SourcePath  string
}

// matter mirrors the YAML frontmatter block at the top of each post.
type matter struct {
	Title   string   `yaml:"title"`
	Date    string   `yaml:"date"`
	Tags    []string `yaml:"tags"`
	Draft   bool     `yaml:"draft"`
	Summary string   `yaml:"summary"`
	Slug    string   `yaml:"slug"`
}

// datePrefix matches the "2026-08-20-" that orders files in the directory
// listing but should not appear in the URL.
var datePrefix = regexp.MustCompile(`^(\d{4}-\d{2}-\d{2})-`)

// ParseFile reads one Markdown file and renders it.
func ParseFile(path string) (*Post, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	md := newMarkdown()
	ctx := parser.NewContext()
	var buf bytes.Buffer
	if err := md.Convert(src, &buf, parser.WithContext(ctx)); err != nil {
		return nil, fmt.Errorf("%s: render: %w", path, err)
	}

	var fm matter
	if d := frontmatter.Get(ctx); d != nil {
		if err := d.Decode(&fm); err != nil {
			return nil, fmt.Errorf("%s: frontmatter: %w", path, err)
		}
	}

	base := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))

	// The date comes from frontmatter when present, otherwise from the
	// filename prefix. One of the two must supply it.
	date, err := resolveDate(fm.Date, base)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}

	slug := fm.Slug
	if slug == "" {
		slug = slugify(datePrefix.ReplaceAllString(base, ""))
	}

	title := fm.Title
	if title == "" {
		return nil, fmt.Errorf("%s: frontmatter is missing a title", path)
	}

	p := &Post{
		Title:       title,
		Date:        date,
		Tags:        normalizeTags(fm.Tags),
		Draft:       fm.Draft,
		Summary:     fm.Summary,
		Slug:        slug,
		Path:        "/posts/" + slug + "/",
		HTML:        template.HTML(buf.String()),
		ReadingTime: readingTime(src),
		SourcePath:  path,
	}
	if p.Summary == "" {
		p.Summary = firstSentences(src)
	}
	return p, nil
}

// ParseDir reads every .md file directly under dir. Drafts are dropped unless
// includeDrafts is set, and the result is sorted newest first.
func ParseDir(dir string, includeDrafts bool) ([]*Post, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var posts []*Post
	for _, e := range entries {
		if e.IsDir() || !strings.EqualFold(filepath.Ext(e.Name()), ".md") {
			continue
		}
		p, err := ParseFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, err
		}
		if p.Draft && !includeDrafts {
			continue
		}
		posts = append(posts, p)
	}

	sort.SliceStable(posts, func(i, j int) bool {
		return posts[i].Date.After(posts[j].Date)
	})
	return posts, nil
}

func resolveDate(fmDate, filename string) (time.Time, error) {
	if fmDate != "" {
		// Accept a bare date or a full RFC 3339 timestamp.
		for _, layout := range []string{time.RFC3339, "2006-01-02 15:04:05", "2006-01-02"} {
			if t, err := time.Parse(layout, fmDate); err == nil {
				return t, nil
			}
		}
		return time.Time{}, fmt.Errorf("cannot parse date %q", fmDate)
	}
	if m := datePrefix.FindStringSubmatch(filename); m != nil {
		return time.Parse("2006-01-02", m[1])
	}
	return time.Time{}, fmt.Errorf("no date in frontmatter and none in the filename")
}

var nonSlug = regexp.MustCompile(`[^a-z0-9]+`)

func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = nonSlug.ReplaceAllString(s, "-")
	return strings.Trim(s, "-")
}

// normalizeTags lowercases, slugifies and de-duplicates, so "Go", "go" and
// "GO" all land on the same tag page.
func normalizeTags(tags []string) []string {
	seen := make(map[string]bool, len(tags))
	out := make([]string, 0, len(tags))
	for _, t := range tags {
		s := slugify(t)
		if s == "" || seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}

// readingTime estimates minutes at 220 words per minute.
func readingTime(src []byte) int {
	words := len(strings.Fields(string(src)))
	if m := words / 220; m > 0 {
		return m
	}
	return 1
}

// firstSentences is the fallback summary: the first prose paragraph of the
// body, stripped of Markdown punctuation and clipped to ~200 characters.
func firstSentences(src []byte) string {
	body := string(src)
	// Skip the frontmatter block if there is one.
	if strings.HasPrefix(body, "---") {
		if i := strings.Index(body[3:], "\n---"); i >= 0 {
			body = body[3+i+4:]
		}
	}

	for _, para := range strings.Split(body, "\n\n") {
		para = strings.TrimSpace(para)
		if para == "" || strings.HasPrefix(para, "#") || strings.HasPrefix(para, "```") {
			continue
		}
		para = strings.NewReplacer("*", "", "_", "", "`", "", "\n", " ").Replace(para)
		if len(para) > 200 {
			if i := strings.LastIndex(para[:200], " "); i > 0 {
				return para[:i] + "…"
			}
			return para[:200] + "…"
		}
		return para
	}
	return ""
}
