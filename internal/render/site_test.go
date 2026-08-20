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

func testConfig() *config.Config {
	return &config.Config{
		Title:       "Romsnack",
		Author:      "Romsnack",
		Description: "Notes on DevSecOps.",
		BaseURL:     "https://romsnack.github.io/PersonalBlog",
		BasePath:    "/PersonalBlog",
		Language:    "en",
	}
}

func day(s string) time.Time {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		panic(err)
	}
	return t
}

func post(title, slug, date string, tags ...string) *content.Post {
	return &content.Post{
		Title:   title,
		Date:    day(date),
		Tags:    tags,
		Summary: title + " summary.",
		Slug:    slug,
		Path:    "/posts/" + slug + "/",
		HTML:    template.HTML("<p>Body of " + title + ".</p>"),
	}
}

// testSite is two posts sharing a tag, plus one standalone page.
func testSite() *Site {
	posts := []*content.Post{
		post("Layers", "layers", "2026-08-20", "containers", "docker"),
		post("Hello", "hello", "2026-01-01", "docker"),
	}
	about := post("About", "about", "2025-06-01")
	about.Path = "/about/"
	return &Site{
		Config: testConfig(),
		Posts:  posts,
		Pages:  []*content.Post{about},
		Tags:   indexTags(posts),
	}
}

func TestIndexTagsGroupsAndSorts(t *testing.T) {
	tags := testSite().Tags
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
	tmpl, err := parseTemplates(testConfig())
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
	out := filepath.Join(t.TempDir(), "public")
	if err := testSite().Build(out, filepath.Join(t.TempDir(), "absent-static")); err != nil {
		t.Fatal(err)
	}

	for _, rel := range []string{
		"index.html",
		"posts/layers/index.html",
		"posts/hello/index.html",
		"about/index.html",
		"tags/index.html",
		"tags/containers/index.html",
		"tags/docker/index.html",
		"atom.xml",
		"sitemap.xml",
	} {
		if _, err := os.Stat(filepath.Join(out, rel)); err != nil {
			t.Errorf("missing %s: %v", rel, err)
		}
	}
}

func TestBuildRendersPostContentAndSubpathLinks(t *testing.T) {
	out := filepath.Join(t.TempDir(), "public")
	if err := testSite().Build(out, filepath.Join(t.TempDir(), "absent-static")); err != nil {
		t.Fatal(err)
	}

	b, err := os.ReadFile(filepath.Join(out, "posts", "layers", "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	html := string(b)

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

func TestBuildReplacesPreviousOutput(t *testing.T) {
	out := filepath.Join(t.TempDir(), "public")
	if err := os.MkdirAll(out, 0o755); err != nil {
		t.Fatal(err)
	}
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

func TestLoadReadsPostsAndPages(t *testing.T) {
	dir := t.TempDir()
	postsDir := filepath.Join(dir, "posts")
	pagesDir := filepath.Join(dir, "pages")
	for _, d := range []string{postsDir, pagesDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	write := func(dir, name, body string) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write(postsDir, "2026-08-20-layers.md", "---\ntitle: Layers\ntags: [docker]\n---\n\nBody.\n")
	write(postsDir, "2026-05-05-wip.md", "---\ntitle: WIP\ndraft: true\n---\n\nBody.\n")
	write(pagesDir, "about.md", "---\ntitle: About\ndate: 2025-06-01\n---\n\nAbout me.\n")

	site, err := Load(testConfig(), dir, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(site.Posts) != 1 {
		t.Errorf("got %d posts, want 1 (the draft should be dropped)", len(site.Posts))
	}
	if len(site.Pages) != 1 {
		t.Fatalf("got %d pages, want 1", len(site.Pages))
	}
	// Pages live at the root, not under /posts/.
	if site.Pages[0].Path != "/about/" {
		t.Errorf("page Path = %q, want /about/", site.Pages[0].Path)
	}
	if len(site.Tags) != 1 || site.Tags[0].Name != "docker" {
		t.Errorf("Tags = %+v, want one docker tag", site.Tags)
	}
}

func TestLoadWithoutAPagesDirectory(t *testing.T) {
	dir := t.TempDir()
	postsDir := filepath.Join(dir, "posts")
	if err := os.MkdirAll(postsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(postsDir, "2026-08-20-p.md"),
		[]byte("---\ntitle: P\n---\n\nBody.\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	site, err := Load(testConfig(), dir, false)
	if err != nil {
		t.Fatalf("a site with no pages/ directory is legal, got %v", err)
	}
	if len(site.Pages) != 0 {
		t.Errorf("got %d pages, want 0", len(site.Pages))
	}
}

func TestLoadReportsAMissingPostsDirectory(t *testing.T) {
	if _, err := Load(testConfig(), t.TempDir(), false); err == nil {
		t.Error("want an error when content/posts is absent, got nil")
	}
}
