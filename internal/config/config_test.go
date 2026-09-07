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

// --- languages -------------------------------------------------------------

// A config with no languages: block is a single-language site, seeded from the
// shorthand fields, so every existing config keeps working untouched.
func TestLoadSynthesisesASingleLanguage(t *testing.T) {
	c, err := Load(writeConfig(t, "title: T\nlanguage: en\ndescription: D\ntagline: G\n"+
		"baseURL: https://romsnack.dev\n"))
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Languages) != 1 {
		t.Fatalf("got %d languages, want 1", len(c.Languages))
	}
	l := c.Languages[0]
	if l.Code != "en" || l.Description != "D" || l.Tagline != "G" {
		t.Errorf("synthesised language = %+v, want the shorthand fields", l)
	}
	if c.DefaultLanguage != "en" {
		t.Errorf("DefaultLanguage = %q, want en", c.DefaultLanguage)
	}
}

func TestLoadReadsALanguageList(t *testing.T) {
	c, err := Load(writeConfig(t, `
title: T
baseURL: https://romsnack.dev
defaultLanguage: en
languages:
  - code: en
    name: English
    description: EN
  - code: fr
    name: Français
    description: FR
`))
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Languages) != 2 {
		t.Fatalf("got %d languages, want 2", len(c.Languages))
	}
	if c.Languages[1].Name != "Français" {
		t.Errorf("second language name = %q", c.Languages[1].Name)
	}
}

// Languages[0] is the default from here on, so callers can rely on it without
// searching the slice — including for x-default in the hreflang tags.
func TestLoadMovesTheDefaultLanguageFirst(t *testing.T) {
	c, err := Load(writeConfig(t, `
title: T
baseURL: https://romsnack.dev
defaultLanguage: fr
languages:
  - code: en
  - code: fr
`))
	if err != nil {
		t.Fatal(err)
	}
	if c.Languages[0].Code != "fr" {
		t.Errorf("Languages[0] = %q, want the default language first", c.Languages[0].Code)
	}
	// Language mirrors the default so single-language callers keep working.
	if c.Language != "fr" {
		t.Errorf("Language = %q, want it to mirror defaultLanguage", c.Language)
	}
}

func TestLoadDefaultsTheLanguageNameToItsCode(t *testing.T) {
	c, err := Load(writeConfig(t, "title: T\nbaseURL: https://romsnack.dev\nlanguages:\n  - code: fr\n"))
	if err != nil {
		t.Fatal(err)
	}
	if c.Languages[0].Name != "fr" {
		t.Errorf("Name = %q, want it to fall back to the code", c.Languages[0].Name)
	}
}

func TestLoadRejectsABadLanguageList(t *testing.T) {
	cases := map[string]string{
		"no code":         "languages:\n  - name: French\n",
		"duplicate code":  "languages:\n  - code: fr\n  - code: fr\n",
		"unknown default": "defaultLanguage: de\nlanguages:\n  - code: en\n",
	}
	for name, body := range cases {
		if _, err := Load(writeConfig(t, "title: T\nbaseURL: https://romsnack.dev\n"+body)); err == nil {
			t.Errorf("%s: want an error, got nil", name)
		}
	}
}

// --- per-language URLs -----------------------------------------------------

// The default language keeps the root, so adding a language never moves a page
// that already exists.
func TestPrefixIsEmptyForTheDefaultLanguage(t *testing.T) {
	c := &Config{DefaultLanguage: "en"}
	if got := c.Prefix("en"); got != "" {
		t.Errorf("Prefix(en) = %q, want empty", got)
	}
	if got := c.Prefix("fr"); got != "/fr" {
		t.Errorf("Prefix(fr) = %q, want /fr", got)
	}
}

func TestLangURLAndLangAbsURL(t *testing.T) {
	c := &Config{
		BaseURL:         "https://romsnack.github.io/PersonalBlog",
		BasePath:        "/PersonalBlog",
		DefaultLanguage: "en",
	}

	cases := []struct{ lang, in, wantURL, wantAbs string }{
		{"en", "/posts/hello/", "/PersonalBlog/posts/hello/", "https://romsnack.github.io/PersonalBlog/posts/hello/"},
		{"fr", "/posts/bonjour/", "/PersonalBlog/fr/posts/bonjour/", "https://romsnack.github.io/PersonalBlog/fr/posts/bonjour/"},
		{"fr", "posts/bonjour/", "/PersonalBlog/fr/posts/bonjour/", "https://romsnack.github.io/PersonalBlog/fr/posts/bonjour/"},
		{"fr", "/", "/PersonalBlog/fr/", "https://romsnack.github.io/PersonalBlog/fr/"},
	}
	for _, tc := range cases {
		if got := c.LangURL(tc.lang, tc.in); got != tc.wantURL {
			t.Errorf("LangURL(%q, %q) = %q, want %q", tc.lang, tc.in, got, tc.wantURL)
		}
		if got := c.LangAbsURL(tc.lang, tc.in); got != tc.wantAbs {
			t.Errorf("LangAbsURL(%q, %q) = %q, want %q", tc.lang, tc.in, got, tc.wantAbs)
		}
	}
}

// URL and AbsURL are the default language's forms, which is what shared assets
// and single-language callers want.
func TestURLIsTheDefaultLanguagesURL(t *testing.T) {
	c := &Config{BaseURL: "https://romsnack.dev", BasePath: "", DefaultLanguage: "fr"}
	if got := c.URL("/style.css"); got != "/style.css" {
		t.Errorf("URL = %q, want the unprefixed path", got)
	}
	if got := c.LangURL("en", "/style.css"); got != "/en/style.css" {
		t.Errorf("LangURL(en) = %q, want /en/style.css", got)
	}
}
