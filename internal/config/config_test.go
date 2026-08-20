package config

import (
	"os"
	"path/filepath"
	"testing"
)

// writeConfig drops a config.yaml into a temp dir and returns its path.
func writeConfig(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadDerivesBasePathFromBaseURL(t *testing.T) {
	c, err := Load(writeConfig(t, "title: Romsnack\nbaseURL: https://romsnack.github.io/PersonalBlog\n"))
	if err != nil {
		t.Fatal(err)
	}
	if c.BasePath != "/PersonalBlog" {
		t.Errorf("BasePath = %q, want /PersonalBlog", c.BasePath)
	}
}

func TestLoadBasePathIsEmptyOnACustomDomain(t *testing.T) {
	c, err := Load(writeConfig(t, "title: Romsnack\nbaseURL: https://romsnack.dev\n"))
	if err != nil {
		t.Fatal(err)
	}
	if c.BasePath != "" {
		t.Errorf("BasePath = %q, want empty", c.BasePath)
	}
}

func TestLoadTrimsTrailingSlash(t *testing.T) {
	c, err := Load(writeConfig(t, "title: T\nbaseURL: https://romsnack.dev/blog/\n"))
	if err != nil {
		t.Fatal(err)
	}
	if c.BaseURL != "https://romsnack.dev/blog" {
		t.Errorf("BaseURL = %q", c.BaseURL)
	}
	if c.BasePath != "/blog" {
		t.Errorf("BasePath = %q, want /blog", c.BasePath)
	}
}

func TestLoadAppliesDefaults(t *testing.T) {
	c, err := Load(writeConfig(t, "baseURL: https://romsnack.dev\n"))
	if err != nil {
		t.Fatal(err)
	}
	if c.Title != "Blog" {
		t.Errorf("Title = %q, want the Blog default", c.Title)
	}
	if c.Language != "en" {
		t.Errorf("Language = %q, want the en default", c.Language)
	}
}

func TestLoadRequiresBaseURL(t *testing.T) {
	if _, err := Load(writeConfig(t, "title: Romsnack\n")); err == nil {
		t.Error("want an error when baseURL is missing, got nil")
	}
}

func TestLoadRejectsInvalidYAML(t *testing.T) {
	if _, err := Load(writeConfig(t, "title: [unclosed\n")); err == nil {
		t.Error("want an error on malformed YAML, got nil")
	}
}

func TestLoadReportsAMissingFile(t *testing.T) {
	if _, err := Load(filepath.Join(t.TempDir(), "absent.yaml")); err == nil {
		t.Error("want an error for a missing config, got nil")
	}
}

func TestURLAndAbsURL(t *testing.T) {
	c := &Config{BaseURL: "https://romsnack.github.io/PersonalBlog", BasePath: "/PersonalBlog"}

	cases := []struct{ in, wantURL, wantAbs string }{
		{"/posts/hello/", "/PersonalBlog/posts/hello/", "https://romsnack.github.io/PersonalBlog/posts/hello/"},
		{"posts/hello/", "/PersonalBlog/posts/hello/", "https://romsnack.github.io/PersonalBlog/posts/hello/"},
		{"/", "/PersonalBlog/", "https://romsnack.github.io/PersonalBlog/"},
	}
	for _, tc := range cases {
		if got := c.URL(tc.in); got != tc.wantURL {
			t.Errorf("URL(%q) = %q, want %q", tc.in, got, tc.wantURL)
		}
		if got := c.AbsURL(tc.in); got != tc.wantAbs {
			t.Errorf("AbsURL(%q) = %q, want %q", tc.in, got, tc.wantAbs)
		}
	}
}

func TestURLOnACustomDomainAddsNoPrefix(t *testing.T) {
	c := &Config{BaseURL: "https://romsnack.dev", BasePath: ""}
	if got := c.URL("/about/"); got != "/about/" {
		t.Errorf("URL = %q, want /about/", got)
	}
}
