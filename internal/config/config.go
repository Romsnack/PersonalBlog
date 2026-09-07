// Package config loads the site-wide settings from config.yaml.
package config

import (
	"fmt"
	"net/url"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// Link is an external profile rendered in the site footer. Name is the visible
// label, lowercased by convention to match the rest of the chrome.
type Link struct {
	Name string `yaml:"name"`
	URL  string `yaml:"url"`
}

// Language is one edition of the site. The default language is served from the
// root ("/posts/…"); every other one lives under its own code ("/fr/posts/…"),
// so adding a language never moves the pages that already exist.
type Language struct {
	// Code is the BCP 47 tag used in <html lang>, in hreflang, and — for
	// non-default languages — as the URL prefix.
	Code string `yaml:"code"`
	// Name is what the language calls itself, for the header switcher.
	// "Français", not "French".
	Name string `yaml:"name"`
	// Description and Tagline are the per-language versions of the site's
	// own copy: the meta description and the homepage hero.
	Description string `yaml:"description"`
	Tagline     string `yaml:"tagline"`
}

// Config holds everything the templates and generators need to know about the
// site as a whole. Per-language copy lives in Language, per-post data in
// content.Post.
type Config struct {
	Title  string `yaml:"title"`
	Author string `yaml:"author"`

	// Description, Tagline and Language are the single-language shorthand.
	// They seed the sole Language entry when languages: is absent, and are
	// otherwise unused — the per-language fields win.
	Description string `yaml:"description"`
	Tagline     string `yaml:"tagline"`
	Language    string `yaml:"language"`

	BaseURL string `yaml:"baseURL"`
	Links   []Link `yaml:"links"`

	// DefaultLanguage is the code served from the root. Defaults to the first
	// entry in Languages.
	DefaultLanguage string     `yaml:"defaultLanguage"`
	Languages       []Language `yaml:"languages"`

	// BasePath is the path component of BaseURL ("/PersonalBlog" on a GitHub
	// project page, "" on a custom domain). Derived, not read from YAML.
	BasePath string `yaml:"-"`

	// Dev is set by `serve` and gates the live-reload snippet so it never
	// reaches a production build.
	Dev bool `yaml:"-"`
}

// Load reads path and fills in defaults for anything left blank.
func Load(path string) (*Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var c Config
	if err := yaml.Unmarshal(b, &c); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}

	if c.Title == "" {
		c.Title = "Blog"
	}
	if c.Language == "" {
		c.Language = "en"
	}
	if c.BaseURL == "" {
		return nil, fmt.Errorf("%s: baseURL is required", path)
	}

	if err := c.resolveLanguages(path); err != nil {
		return nil, err
	}

	u, err := url.Parse(c.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("%s: baseURL is not a valid URL: %w", path, err)
	}
	c.BaseURL = strings.TrimSuffix(c.BaseURL, "/")
	c.BasePath = strings.TrimSuffix(u.Path, "/")

	return &c, nil
}

// resolveLanguages normalises the language list: a config with no languages:
// block is a single-language site, and the default language is moved to the
// front so callers can rely on Languages[0] being the one served from the root.
func (c *Config) resolveLanguages(path string) error {
	if len(c.Languages) == 0 {
		c.Languages = []Language{{
			Code:        c.Language,
			Name:        c.Language,
			Description: c.Description,
			Tagline:     c.Tagline,
		}}
	}

	seen := make(map[string]bool, len(c.Languages))
	for i := range c.Languages {
		l := &c.Languages[i]
		if l.Code == "" {
			return fmt.Errorf("%s: languages[%d] has no code", path, i)
		}
		if seen[l.Code] {
			return fmt.Errorf("%s: language %q is listed twice", path, l.Code)
		}
		seen[l.Code] = true
		if l.Name == "" {
			l.Name = l.Code
		}
	}

	if c.DefaultLanguage == "" {
		c.DefaultLanguage = c.Languages[0].Code
	}
	if !seen[c.DefaultLanguage] {
		return fmt.Errorf("%s: defaultLanguage %q is not in languages", path, c.DefaultLanguage)
	}

	// Languages[0] is the default from here on.
	for i, l := range c.Languages {
		if l.Code == c.DefaultLanguage {
			c.Languages[0], c.Languages[i] = c.Languages[i], c.Languages[0]
			break
		}
	}

	// Language mirrors the default so single-language callers keep working.
	c.Language = c.DefaultLanguage
	return nil
}

// Prefix is the URL segment a language's pages sit under: empty for the
// default language, "/fr" for the rest.
func (c *Config) Prefix(lang string) string {
	if lang == "" || lang == c.DefaultLanguage {
		return ""
	}
	return "/" + lang
}

// URL turns a site-relative path into one the browser can follow, accounting
// for a subpath deployment. URL("/posts/hello/") -> "/PersonalBlog/posts/hello/".
// The path is in the default language; see LangURL for the others.
func (c *Config) URL(p string) string {
	return c.LangURL(c.DefaultLanguage, p)
}

// LangURL is URL for one specific language's edition of the path.
// LangURL("fr", "/posts/x/") -> "/PersonalBlog/fr/posts/x/".
func (c *Config) LangURL(lang, p string) string {
	return c.BasePath + c.Prefix(lang) + slashed(p)
}

// AbsURL is URL with the scheme and host, for feeds, sitemaps and OG tags.
func (c *Config) AbsURL(p string) string {
	return c.LangAbsURL(c.DefaultLanguage, p)
}

// LangAbsURL is AbsURL for one specific language.
func (c *Config) LangAbsURL(lang, p string) string {
	// BaseURL already carries BasePath, so append to it directly.
	return c.BaseURL + c.Prefix(lang) + slashed(p)
}

func slashed(p string) string {
	if !strings.HasPrefix(p, "/") {
		return "/" + p
	}
	return p
}
