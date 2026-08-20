package render

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"

	"github.com/Romsnack/PersonalBlog/internal/config"
	"github.com/Romsnack/PersonalBlog/internal/content"
)

// Site is the whole blog in memory: the source of truth every output file is
// derived from. Build the model once, then write from it.
type Site struct {
	Config *config.Config
	Posts  []*content.Post // newest first
	Pages  []*content.Post // standalone pages, e.g. about
	Tags   []Tag           // alphabetical
}

// Tag is one tag and the posts carrying it.
type Tag struct {
	Name  string
	Path  string
	Posts []*content.Post
}

// pageData is what every template receives. Keeping one shape means base.html
// can always reach Site and Title without each page inventing its own.
type pageData struct {
	Site        *Site
	Title       string // page title; empty means "use the site title alone"
	Description string
	Path        string // site-relative path of the page being rendered
	Post        *content.Post
	Tag         *Tag
	Posts       []*content.Post
}

// Load reads content/ and assembles the site model.
func Load(cfg *config.Config, contentDir string, includeDrafts bool) (*Site, error) {
	posts, err := content.ParseDir(filepath.Join(contentDir, "posts"), includeDrafts)
	if err != nil {
		return nil, fmt.Errorf("posts: %w", err)
	}

	// Standalone pages are optional.
	var pages []*content.Post
	pagesDir := filepath.Join(contentDir, "pages")
	if _, err := os.Stat(pagesDir); err == nil {
		pages, err = content.ParseDir(pagesDir, includeDrafts)
		if err != nil {
			return nil, fmt.Errorf("pages: %w", err)
		}
		for _, p := range pages {
			p.Path = "/" + p.Slug + "/"
		}
	}

	return &Site{
		Config: cfg,
		Posts:  posts,
		Pages:  pages,
		Tags:   indexTags(posts),
	}, nil
}

func indexTags(posts []*content.Post) []Tag {
	byName := map[string][]*content.Post{}
	for _, p := range posts {
		for _, t := range p.Tags {
			byName[t] = append(byName[t], p)
		}
	}

	tags := make([]Tag, 0, len(byName))
	for name, ps := range byName {
		tags = append(tags, Tag{Name: name, Path: "/tags/" + name + "/", Posts: ps})
	}
	sort.Slice(tags, func(i, j int) bool { return tags[i].Name < tags[j].Name })
	return tags
}

// Build writes the complete site to outDir, replacing whatever was there.
func (s *Site) Build(outDir, staticDir string) error {
	if err := os.RemoveAll(outDir); err != nil {
		return err
	}

	tmpl, err := parseTemplates(s.Config)
	if err != nil {
		return fmt.Errorf("templates: %w", err)
	}

	// Index.
	if err := s.writePage(tmpl["index.html"], "/", pageData{
		Site:        s,
		Description: s.Config.Description,
		Path:        "/",
		Posts:       s.Posts,
	}, outDir); err != nil {
		return err
	}

	// One page per post.
	for _, p := range s.Posts {
		if err := s.writePage(tmpl["post.html"], p.Path, pageData{
			Site:        s,
			Title:       p.Title,
			Description: p.Summary,
			Path:        p.Path,
			Post:        p,
		}, outDir); err != nil {
			return err
		}
	}

	// Standalone pages.
	for _, p := range s.Pages {
		if err := s.writePage(tmpl["page.html"], p.Path, pageData{
			Site:        s,
			Title:       p.Title,
			Description: p.Summary,
			Path:        p.Path,
			Post:        p,
		}, outDir); err != nil {
			return err
		}
	}

	// Tag list and one page per tag.
	if err := s.writePage(tmpl["tags.html"], "/tags/", pageData{
		Site:  s,
		Title: "Tags",
		Path:  "/tags/",
	}, outDir); err != nil {
		return err
	}
	for i := range s.Tags {
		t := &s.Tags[i]
		if err := s.writePage(tmpl["tag.html"], t.Path, pageData{
			Site:  s,
			Title: "#" + t.Name,
			Path:  t.Path,
			Tag:   t,
			Posts: t.Posts,
		}, outDir); err != nil {
			return err
		}
	}

	if err := s.writeFeed(outDir); err != nil {
		return fmt.Errorf("feed: %w", err)
	}
	if err := s.writeSitemap(outDir); err != nil {
		return fmt.Errorf("sitemap: %w", err)
	}
	if err := copyDir(staticDir, outDir); err != nil {
		return fmt.Errorf("static: %w", err)
	}
	return nil
}

// writePage renders one template to <outDir>/<path>/index.html, so URLs need
// no file extension and no trailing-slash redirect.
func (s *Site) writePage(t *template.Template, path string, data pageData, outDir string) error {
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return fmt.Errorf("render %s: %w", path, err)
	}
	dst := filepath.Join(outDir, filepath.FromSlash(path), "index.html")
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	return os.WriteFile(dst, buf.Bytes(), 0o644)
}

// copyDir copies src into dst verbatim. Missing src is not an error — a site
// with no static assets is legal.
func copyDir(src, dst string) error {
	if _, err := os.Stat(src); os.IsNotExist(err) {
		return nil
	}
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		defer in.Close()
		out, err := os.Create(target)
		if err != nil {
			return err
		}
		defer out.Close()
		_, err = io.Copy(out, in)
		return err
	})
}
