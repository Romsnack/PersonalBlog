package render

import (
	"html/template"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Romsnack/PersonalBlog/internal/config"
	"github.com/Romsnack/PersonalBlog/internal/content"
)

// testConfig is a two-language site: English at the root, French under /fr/.
func testConfig() *config.Config {
	return &config.Config{
		Title:           "Romsnack",
		Author:          "Romsnack",
		BaseURL:         "https://romsnack.github.io/PersonalBlog",
		BasePath:        "/PersonalBlog",
		Language:        "en",
		DefaultLanguage: "en",
		Languages: []config.Language{
			{Code: "en", Name: "English", Description: "Notes on DevSecOps.", Tagline: "A mindset."},
			{Code: "fr", Name: "Français", Description: "Notes sur le DevSecOps.", Tagline: "Un état d'esprit."},
		},
	}
}

// testConfigSingle is the same site with the languages: block left out, which
// is how a single-language config arrives from config.Load.
func testConfigSingle() *config.Config {
	return &config.Config{
		Title:           "Romsnack",
		Author:          "Romsnack",
		BaseURL:         "https://romsnack.github.io/PersonalBlog",
		BasePath:        "/PersonalBlog",
		Language:        "en",
		DefaultLanguage: "en",
		Languages:       []config.Language{{Code: "en", Name: "English", Description: "Notes on DevSecOps."}},
	}
}

func day(s string) time.Time {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		panic(err)
	}
	return t
}

// translated builds a post with an explicit translation key, so the same
// article can carry a different slug in each language. A post with no
// translation simply passes its own slug as the key, which is the default
// ParseFile applies.
func translated(title, slug, key, date string, tags ...string) *content.Post {
	return &content.Post{
		Title:          title,
		Date:           day(date),
		Tags:           tags,
		Summary:        title + " summary.",
		Slug:           slug,
		Path:           "/posts/" + slug + "/",
		TranslationKey: key,
		Nav:            slug,
		HTML:           template.HTML("<p>Body of " + title + ".</p>"),
	}
}

// testSite is a bilingual site. English has two posts and an about page;
// French translates only one of the posts, so both switcher branches — a real
// translation and a fallback to the homepage — are exercised.
func testSite() *Site {
	s := &Site{Config: testConfig()}

	enPosts := []*content.Post{
		translated("Layers", "layers", "layers", "2026-08-20", "containers", "docker"),
		translated("Hello", "hello", "hello", "2026-01-01", "docker"),
	}
	enAbout := translated("About", "about", "about", "2025-06-01")
	enAbout.Path = "/about/"

	frPosts := []*content.Post{
		translated("Couches", "couches", "layers", "2026-08-20", "containers", "docker"),
	}
	frAbout := translated("À propos", "a-propos", "about", "2025-06-01")
	frAbout.Path = "/a-propos/"

	s.Editions = []*Edition{
		{Site: s, Lang: s.Config.Languages[0], Posts: enPosts, Pages: []*content.Post{enAbout}, Tags: indexTags(enPosts)},
		{Site: s, Lang: s.Config.Languages[1], Posts: frPosts, Pages: []*content.Post{frAbout}, Tags: indexTags(frPosts)},
	}
	return s
}

// en and fr name the two editions, so tests read as prose.
func en(s *Site) *Edition { return s.Editions[0] }
func fr(s *Site) *Edition { return s.Editions[1] }

// build renders the whole site into a fresh temp dir and returns it.
func build(t *testing.T, s *Site) string {
	t.Helper()
	out := filepath.Join(t.TempDir(), "public")
	if err := s.Build(out, filepath.Join(t.TempDir(), "absent-static")); err != nil {
		t.Fatal(err)
	}
	return out
}

func read(t *testing.T, parts ...string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(parts...))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestIndexTagsGroupsAndSorts(t *testing.T) {
	tags := en(testSite()).Tags
	if len(tags) != 2 {
		t.Fatalf("got %d tags, want 2", len(tags))
	}
	if tags[0].Name != "containers" || tags[1].Name != "docker" {
		t.Errorf("tags = %q, %q; want them alphabetical", tags[0].Name, tags[1].Name)
	}
	if tags[0].Path != "/tags/containers/" {
		t.Errorf("Path = %q", tags[0].Path)
	}
	if len(tags[1].Posts) != 2 {
		t.Errorf("docker carries %d posts, want 2", len(tags[1].Posts))
	}
}

func TestIndexTagsOnNoPosts(t *testing.T) {
	if got := indexTags(nil); len(got) != 0 {
		t.Errorf("got %d tags for no posts, want 0", len(got))
	}
}

func TestParseTemplatesCoversEveryPageType(t *testing.T) {
	tmpl, err := parseTemplates(testConfig(), "en")
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"index.html", "post.html", "tag.html", "tags.html", "page.html"} {
		if tmpl[name] == nil {
			t.Errorf("no template parsed for %s", name)
		}
	}
}

func TestBuildWritesEveryExpectedFile(t *testing.T) {
	out := build(t, testSite())

	for _, rel := range []string{
		// The default language keeps the root, so existing URLs never move.
		"index.html",
		"posts/layers/index.html",
		"posts/hello/index.html",
		"about/index.html",
		"tags/index.html",
		"tags/containers/index.html",
		"tags/docker/index.html",
		"atom.xml",
		// Every other language lives under its own prefix, feed included.
		"fr/index.html",
		"fr/posts/couches/index.html",
		"fr/a-propos/index.html",
		"fr/tags/index.html",
		"fr/tags/containers/index.html",
		"fr/tags/docker/index.html",
		"fr/atom.xml",
		// One sitemap covers both.
		"sitemap.xml",
	} {
		if _, err := os.Stat(filepath.Join(out, rel)); err != nil {
			t.Errorf("missing %s: %v", rel, err)
		}
	}
}

func TestBuildDoesNotWriteAnUntranslatedPost(t *testing.T) {
	out := build(t, testSite())
	if _, err := os.Stat(filepath.Join(out, "fr", "posts", "hello")); !os.IsNotExist(err) {
		t.Error("a post with no French translation should not produce a French page")
	}
}

func TestBuildRendersPostContentAndSubpathLinks(t *testing.T) {
	out := build(t, testSite())
	html := read(t, out, "posts", "layers", "index.html")

	if !strings.Contains(html, "<p>Body of Layers.</p>") {
		t.Error("rendered post is missing its body HTML")
	}
	if !strings.Contains(html, "Layers") {
		t.Error("rendered post is missing its title")
	}
	// Every internal link must carry the project-page subpath.
	if !strings.Contains(html, "/PersonalBlog/") {
		t.Error("rendered post has no BasePath-prefixed links")
	}
}

func TestBuildPrefixesEveryLinkOnANonDefaultLanguage(t *testing.T) {
	out := build(t, testSite())
	html := read(t, out, "fr", "posts", "couches", "index.html")

	if !strings.Contains(html, `<html lang="fr">`) {
		t.Error("the French page does not declare lang=fr")
	}
	// The nav, the tag links and the feed link all have to land inside /fr/.
	for _, want := range []string{
		`href="/PersonalBlog/fr/"`,
		`href="/PersonalBlog/fr/tags/"`,
		`href="/PersonalBlog/fr/tags/docker/"`,
		`href="/PersonalBlog/fr/atom.xml"`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("French page is missing %s", want)
		}
	}
}

// static/ is copied to the root once and shared by every language, so an asset
// link must not take the language prefix — with one, every page outside the
// default language loads unstyled.
func TestBuildDoesNotPrefixSharedAssets(t *testing.T) {
	out := build(t, testSite())

	for _, page := range [][]string{
		{out, "index.html"},
		{out, "fr", "index.html"},
		{out, "fr", "posts", "couches", "index.html"},
	} {
		html := read(t, page...)
		for _, want := range []string{
			`href="/PersonalBlog/style.css"`,
			`href="/PersonalBlog/chroma.css"`,
		} {
			if !strings.Contains(html, want) {
				t.Errorf("%s is missing %s", filepath.Join(page[1:]...), want)
			}
		}
		if strings.Contains(html, "/fr/style.css") {
			t.Errorf("%s points at a stylesheet that is never written", filepath.Join(page[1:]...))
		}
	}
}

func TestBuildTranslatesTheChrome(t *testing.T) {
	out := build(t, testSite())
	frHTML := read(t, out, "fr", "index.html")
	enHTML := read(t, out, "index.html")

	if !strings.Contains(frHTML, ">articles<") {
		t.Error("French nav does not use the translated label for posts")
	}
	if !strings.Contains(enHTML, ">posts<") {
		t.Error("English nav lost its label")
	}
	if !strings.Contains(frHTML, "Notes sur le DevSecOps.") {
		t.Error("French homepage does not carry the French description")
	}
	if !strings.Contains(enHTML, "Notes on DevSecOps.") {
		t.Error("English homepage does not carry the English description")
	}
}

func TestBuildLinksTranslationsByKeyNotBySlug(t *testing.T) {
	out := build(t, testSite())

	// The English post points at the French slug, which differs from its own.
	enHTML := read(t, out, "posts", "layers", "index.html")
	if !strings.Contains(enHTML, `href="/PersonalBlog/fr/posts/couches/"`) {
		t.Error("the English post does not link to its French translation")
	}
	// And back again.
	frHTML := read(t, out, "fr", "posts", "couches", "index.html")
	if !strings.Contains(frHTML, `href="/PersonalBlog/posts/layers/"`) {
		t.Error("the French post does not link back to its English original")
	}
}

func TestBuildFallsBackToTheHomepageWhenAPageIsUntranslated(t *testing.T) {
	out := build(t, testSite())
	html := read(t, out, "posts", "hello", "index.html")

	if !strings.Contains(html, `class="untranslated"`) {
		t.Error("a post with no translation should mark its switcher link untranslated")
	}
	if !strings.Contains(html, `href="/PersonalBlog/fr/" hreflang="fr"`) {
		t.Error("an untranslated post should fall back to the French homepage")
	}
	// A fallback is not a translation, so it must not claim one to a crawler.
	if strings.Contains(html, `rel="alternate" hreflang="fr"`) {
		t.Error("a fallback link must not be advertised as an hreflang alternate")
	}
}

func TestBuildEmitsHreflangForRealTranslations(t *testing.T) {
	out := build(t, testSite())
	html := read(t, out, "posts", "layers", "index.html")

	const base = "https://romsnack.github.io/PersonalBlog"
	for _, want := range []string{
		`<link rel="alternate" hreflang="en" href="` + base + `/posts/layers/">`,
		`<link rel="alternate" hreflang="fr" href="` + base + `/fr/posts/couches/">`,
		`<link rel="alternate" hreflang="x-default" href="` + base + `/posts/layers/">`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("missing hreflang tag:\n%s", want)
		}
	}
}

func TestBuildOmitsTheSwitcherOnASingleLanguageSite(t *testing.T) {
	s := testSite()
	s.Config = testConfigSingle()
	s.Editions = s.Editions[:1]
	s.Editions[0].Lang = s.Config.Languages[0]

	out := build(t, s)
	if strings.Contains(read(t, out, "index.html"), "lang-switch") {
		t.Error("a one-language site should not render a language switcher")
	}
}

func TestBuildReplacesPreviousOutput(t *testing.T) {
	out := filepath.Join(t.TempDir(), "public")
	stale := filepath.Join(out, "posts", "deleted-post", "index.html")
	if err := os.MkdirAll(filepath.Dir(stale), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(stale, []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := testSite().Build(out, filepath.Join(t.TempDir(), "absent-static")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Error("a page removed from content/ survived the rebuild")
	}
}

func TestBuildCopiesStaticVerbatim(t *testing.T) {
	static := t.TempDir()
	if err := os.MkdirAll(filepath.Join(static, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(static, "style.css"), []byte("body{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(static, "sub", "f.txt"), []byte("nested"), 0o600); err != nil {
		t.Fatal(err)
	}

	out := filepath.Join(t.TempDir(), "public")
	if err := testSite().Build(out, static); err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(filepath.Join(out, "sub", "f.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "nested" {
		t.Errorf("nested static file = %q, want nested", got)
	}
}

func TestCopyDirTreatsMissingSourceAsLegal(t *testing.T) {
	if err := copyDir(filepath.Join(t.TempDir(), "nope"), t.TempDir()); err != nil {
		t.Errorf("missing static dir should not be an error, got %v", err)
	}
}

// writeContent lays out posts/ and pages/ under dir.
func writeContent(t *testing.T, dir string, files map[string]string) {
	t.Helper()
	for rel, body := range files {
		path := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
}

func TestLoadReadsPostsAndPages(t *testing.T) {
	dir := t.TempDir()
	writeContent(t, dir, map[string]string{
		"posts/2026-08-20-layers.md": "---\ntitle: Layers\ntags: [docker]\n---\n\nBody.\n",
		"posts/2026-05-05-wip.md":    "---\ntitle: WIP\ndraft: true\n---\n\nBody.\n",
		"pages/about.md":             "---\ntitle: About\ndate: 2025-06-01\n---\n\nAbout me.\n",
	})

	// A single-language site may keep content/posts directly, with no
	// content/en/ directory in between.
	site, err := Load(testConfigSingle(), dir, false)
	if err != nil {
		t.Fatal(err)
	}
	ed := en(site)
	if len(ed.Posts) != 1 {
		t.Errorf("got %d posts, want 1 (the draft should be dropped)", len(ed.Posts))
	}
	if len(ed.Pages) != 1 {
		t.Fatalf("got %d pages, want 1", len(ed.Pages))
	}
	// Pages live at the root, not under /posts/.
	if ed.Pages[0].Path != "/about/" {
		t.Errorf("page Path = %q, want /about/", ed.Pages[0].Path)
	}
	if len(ed.Tags) != 1 || ed.Tags[0].Name != "docker" {
		t.Errorf("Tags = %+v, want one docker tag", ed.Tags)
	}
}

func TestLoadReadsOneDirectoryPerLanguage(t *testing.T) {
	dir := t.TempDir()
	writeContent(t, dir, map[string]string{
		"en/posts/2026-08-20-layers.md": "---\ntitle: Layers\ntranslationKey: layers\ntags: [docker]\n---\n\nBody.\n",
		"en/pages/about.md":             "---\ntitle: About\ndate: 2025-06-01\n---\n\nAbout me.\n",
		"fr/posts/2026-08-20-couches.md": "---\ntitle: Couches\nslug: couches\n" +
			"translationKey: layers\ntags: [docker]\n---\n\nCorps.\n",
	})

	site, err := Load(testConfig(), dir, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(site.Editions) != 2 {
		t.Fatalf("got %d editions, want 2", len(site.Editions))
	}
	if got := en(site).Posts[0].Title; got != "Layers" {
		t.Errorf("English post = %q", got)
	}
	if got := fr(site).Posts[0].Title; got != "Couches" {
		t.Errorf("French post = %q", got)
	}
	// The two share a key, which is what makes them each other's translation.
	if en(site).Posts[0].TranslationKey != fr(site).Posts[0].TranslationKey {
		t.Error("the two editions of the same article do not share a translation key")
	}
	// A language may have fewer pages than another.
	if len(fr(site).Pages) != 0 {
		t.Errorf("got %d French pages, want 0", len(fr(site).Pages))
	}
}

func TestLoadWithoutAPagesDirectory(t *testing.T) {
	dir := t.TempDir()
	writeContent(t, dir, map[string]string{
		"posts/2026-08-20-p.md": "---\ntitle: P\n---\n\nBody.\n",
	})

	site, err := Load(testConfigSingle(), dir, false)
	if err != nil {
		t.Fatalf("a site with no pages/ directory is legal, got %v", err)
	}
	if len(en(site).Pages) != 0 {
		t.Errorf("got %d pages, want 0", len(en(site).Pages))
	}
}

func TestLoadReportsAMissingPostsDirectory(t *testing.T) {
	if _, err := Load(testConfigSingle(), t.TempDir(), false); err == nil {
		t.Error("want an error when content/posts is absent, got nil")
	}
}

func TestLoadNamesTheLanguageThatFailed(t *testing.T) {
	dir := t.TempDir()
	// English is complete; French has no posts directory at all.
	writeContent(t, dir, map[string]string{
		"en/posts/2026-08-20-layers.md": "---\ntitle: Layers\n---\n\nBody.\n",
	})

	_, err := Load(testConfig(), dir, false)
	if err == nil {
		t.Fatal("want an error when a language has no content, got nil")
	}
	if !strings.Contains(err.Error(), "fr") {
		t.Errorf("error = %q, want it to name the offending language", err)
	}
}
