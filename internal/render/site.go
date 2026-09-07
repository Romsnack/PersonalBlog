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

// Site is the whole blog in memory, every language at once: the source of
// truth every output file is derived from. Build the model once, then write
// from it.
type Site struct {
	Config *config.Config
	// Editions are the per-language views, default language first. Each one
	// is a complete blog; they share only the config and the static assets.
	Editions []*Edition
}

// Edition is one language's complete view of the blog. Templates receive an
// Edition as .Site, so a template never has to ask which language it is in —
// the URLs and UI strings it is handed are already the right ones.
type Edition struct {
	Site  *Site
	Lang  config.Language
	Posts []*content.Post // newest first
	Pages []*content.Post // standalone pages, e.g. about
	Tags  []Tag           // alphabetical
}

// Title is the site title. It is shared across languages today, but templates
// go through the Edition so a per-language title stays a config change.
func (e *Edition) Title() string { return e.Site.Config.Title }

// Description is the language's own meta description.
func (e *Edition) Description() string { return e.Lang.Description }

// Tagline is the language's own homepage hero line.
func (e *Edition) Tagline() string { return e.Lang.Tagline }

// Config exposes the shared settings the chrome needs: author, footer links,
// the dev flag.
func (e *Edition) Config() *config.Config { return e.Site.Config }

// Tag is one tag and the posts carrying it. Tag names are shared across
// languages — "docker" is "docker" — but each language indexes only its own
// posts, so /tags/docker/ and /fr/tags/docker/ are separate pages.
type Tag struct {
	Name  string
	Path  string
	Posts []*content.Post
}

// altLink is one entry in the header's language switcher: where this same page
// lives in another language.
type altLink struct {
	Lang    config.Language
	URL     string
	AbsURL  string
	Current bool
	// Translated is false when the language has no counterpart for this page
	// and the link falls back to that language's homepage. The switcher dims
	// those so a reader is not promised a translation that isn't there.
	Translated bool
}

// pageData is what every template receives. Keeping one shape means base.html
// can always reach the edition and the title without each page inventing its
// own.
type pageData struct {
	Site        *Edition
	Title       string // page title; empty means "use the site title alone"
	Description string
	Path        string // language-agnostic site-relative path of this page
	Post        *content.Post
	Tag         *Tag
	Posts       []*content.Post
	// Alts is every language's version of this page, for the switcher and for
	// the hreflang tags. Always includes the current language.
	Alts []altLink
}

// Load reads contentDir and assembles the site model. Each language is read
// from its own subdirectory — content/en/posts, content/fr/posts — except for
// a single-language site, which may keep content/posts directly.
func Load(cfg *config.Config, contentDir string, includeDrafts bool) (*Site, error) {
	s := &Site{Config: cfg}

	for _, lang := range cfg.Languages {
		dir := filepath.Join(contentDir, lang.Code)
		// A single-language site needs no language directory at all.
		if len(cfg.Languages) == 1 {
			if _, err := os.Stat(dir); os.IsNotExist(err) {
				dir = contentDir
			}
		}

		ed, err := loadEdition(s, lang, dir, includeDrafts)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", lang.Code, err)
		}
		s.Editions = append(s.Editions, ed)
	}
	return s, nil
}

func loadEdition(s *Site, lang config.Language, dir string, includeDrafts bool) (*Edition, error) {
	posts, err := content.ParseDir(filepath.Join(dir, "posts"), includeDrafts)
	if err != nil {
		return nil, fmt.Errorf("posts: %w", err)
	}

	// Standalone pages are optional.
	var pages []*content.Post
	pagesDir := filepath.Join(dir, "pages")
	if _, err := os.Stat(pagesDir); err == nil {
		pages, err = content.ParseDir(pagesDir, includeDrafts)
		if err != nil {
			return nil, fmt.Errorf("pages: %w", err)
		}
		for _, p := range pages {
			p.Path = "/" + p.Slug + "/"
		}
	}

	return &Edition{
		Site:  s,
		Lang:  lang,
		Posts: posts,
		Pages: pages,
		Tags:  indexTags(posts),
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

// paths indexes one edition's pages by translation key, so the switcher can
// find the counterpart of a page whose slug differs between languages.
// Keys are namespaced ("post:x" cannot collide with "page:x").
func (e *Edition) paths() map[string]string {
	m := map[string]string{
		"index": "/",
		"tags":  "/tags/",
	}
	for _, p := range e.Posts {
		m["post:"+p.TranslationKey] = p.Path
	}
	for _, p := range e.Pages {
		m["page:"+p.TranslationKey] = p.Path
	}
	for _, t := range e.Tags {
		m["tag:"+t.Name] = t.Path
	}
	return m
}

// alts resolves one translation key across every language. A language with no
// counterpart falls back to its homepage, marked untranslated.
func (s *Site) alts(index map[string]map[string]string, current, key string) []altLink {
	out := make([]altLink, 0, len(s.Editions))
	for _, ed := range s.Editions {
		path, ok := index[ed.Lang.Code][key]
		if !ok {
			path = "/"
		}
		out = append(out, altLink{
			Lang:       ed.Lang,
			URL:        s.Config.LangURL(ed.Lang.Code, path),
			AbsURL:     s.Config.LangAbsURL(ed.Lang.Code, path),
			Current:    ed.Lang.Code == current,
			Translated: ok,
		})
	}
	return out
}

// Build writes the complete site to outDir, replacing whatever was there.
func (s *Site) Build(outDir, staticDir string) error {
	if err := os.RemoveAll(outDir); err != nil {
		return err
	}

	// One index per language up front: a page in one language needs to know
	// where its counterpart lives in all the others.
	index := make(map[string]map[string]string, len(s.Editions))
	for _, ed := range s.Editions {
		index[ed.Lang.Code] = ed.paths()
	}

	for _, ed := range s.Editions {
		if err := ed.build(outDir, index); err != nil {
			return fmt.Errorf("%s: %w", ed.Lang.Code, err)
		}
	}

	// One sitemap covers every language, which is what the spec expects.
	if err := s.writeSitemap(outDir); err != nil {
		return fmt.Errorf("sitemap: %w", err)
	}
	if err := copyDir(staticDir, outDir); err != nil {
		return fmt.Errorf("static: %w", err)
	}
	return nil
}

// build writes one language's pages under its own prefix.
func (e *Edition) build(outDir string, index map[string]map[string]string) error {
	cfg := e.Site.Config
	lang := e.Lang.Code

	tmpl, err := parseTemplates(cfg, lang)
	if err != nil {
		return fmt.Errorf("templates: %w", err)
	}

	alts := func(key string) []altLink { return e.Site.alts(index, lang, key) }

	// Index.
	if err := e.writePage(tmpl["index.html"], "/", pageData{
		Site:        e,
		Description: e.Description(),
		Path:        "/",
		Posts:       e.Posts,
		Alts:        alts("index"),
	}, outDir); err != nil {
		return err
	}

	// One page per post.
	for _, p := range e.Posts {
		if err := e.writePage(tmpl["post.html"], p.Path, pageData{
			Site:        e,
			Title:       p.Title,
			Description: p.Summary,
			Path:        p.Path,
			Post:        p,
			Alts:        alts("post:" + p.TranslationKey),
		}, outDir); err != nil {
			return err
		}
	}

	// Standalone pages.
	for _, p := range e.Pages {
		if err := e.writePage(tmpl["page.html"], p.Path, pageData{
			Site:        e,
			Title:       p.Title,
			Description: p.Summary,
			Path:        p.Path,
			Post:        p,
			Alts:        alts("page:" + p.TranslationKey),
		}, outDir); err != nil {
			return err
		}
	}

	// Tag list and one page per tag.
	if err := e.writePage(tmpl["tags.html"], "/tags/", pageData{
		Site:  e,
		Title: translate(lang, cfg.DefaultLanguage, "tags.title"),
		Path:  "/tags/",
		Alts:  alts("tags"),
	}, outDir); err != nil {
		return err
	}
	for i := range e.Tags {
		t := &e.Tags[i]
		if err := e.writePage(tmpl["tag.html"], t.Path, pageData{
			Site:  e,
			Title: "#" + t.Name,
			Path:  t.Path,
			Tag:   t,
			Posts: t.Posts,
			Alts:  alts("tag:" + t.Name),
		}, outDir); err != nil {
			return err
		}
	}

	if err := e.writeFeed(outDir); err != nil {
		return fmt.Errorf("feed: %w", err)
	}
	return nil
}

// outPath maps a language-agnostic site path to its file on disk, adding the
// language prefix. "/posts/x/" in fr -> "<outDir>/fr/posts/x/".
func (e *Edition) outPath(outDir, path string) string {
	prefix := e.Site.Config.Prefix(e.Lang.Code)
	return filepath.Join(outDir, filepath.FromSlash(prefix), filepath.FromSlash(path))
}

// writePage renders one template to <outDir>/<lang>/<path>/index.html, so URLs
// need no file extension and no trailing-slash redirect.
func (e *Edition) writePage(t *template.Template, path string, data pageData, outDir string) error {
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return fmt.Errorf("render %s: %w", path, err)
	}
	dst := filepath.Join(e.outPath(outDir, path), "index.html")
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
