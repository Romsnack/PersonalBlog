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

// Config holds everything the templates and generators need to know about the
// site as a whole. Per-post data lives in content.Post instead.
type Config struct {
	Title       string `yaml:"title"`
	Author      string `yaml:"author"`
	Description string `yaml:"description"`
	Tagline     string `yaml:"tagline"`
	BaseURL     string `yaml:"baseURL"`
	Language    string `yaml:"language"`
	Links       []Link `yaml:"links"`

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

	u, err := url.Parse(c.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("%s: baseURL is not a valid URL: %w", path, err)
	}
	c.BaseURL = strings.TrimSuffix(c.BaseURL, "/")
	c.BasePath = strings.TrimSuffix(u.Path, "/")

	return &c, nil
}

// URL turns a site-relative path into one the browser can follow, accounting
// for a subpath deployment. URL("/posts/hello/") -> "/PersonalBlog/posts/hello/".
func (c *Config) URL(p string) string {
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	return c.BasePath + p
}

// AbsURL is URL with the scheme and host, for feeds, sitemaps and OG tags.
func (c *Config) AbsURL(p string) string {
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	// BaseURL already carries BasePath, so append to it directly.
	return c.BaseURL + p
}
