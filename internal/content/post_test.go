package content

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestSlugify(t *testing.T) {
	cases := map[string]string{
		"Hello World":             "hello-world",
		"  Trimmed  ":             "trimmed",
		"Go 1.25 & containers!":   "go-1-25-containers",
		"---already---slugged---": "already-slugged",
		"ÉÀÜ":                     "",
	}
	for in, want := range cases {
		if got := slugify(in); got != want {
			t.Errorf("slugify(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestNormalizeTagsDedupesAndSorts(t *testing.T) {
	got := normalizeTags([]string{"Go", "go", "GO", "App Sec", "", "containers"})
	want := []string{"app-sec", "containers", "go"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("normalizeTags = %v, want %v", got, want)
	}
}

func TestResolveDate(t *testing.T) {
	t.Run("frontmatter wins over filename", func(t *testing.T) {
		got, err := resolveDate("2026-01-02", "2020-12-31-post")
		if err != nil {
			t.Fatal(err)
		}
		if got.Format("2006-01-02") != "2026-01-02" {
			t.Errorf("got %s, want 2026-01-02", got)
		}
	})

	t.Run("falls back to the filename prefix", func(t *testing.T) {
		got, err := resolveDate("", "2026-08-20-container-layers")
		if err != nil {
			t.Fatal(err)
		}
		if got.Format("2006-01-02") != "2026-08-20" {
			t.Errorf("got %s, want 2026-08-20", got)
		}
	})

	t.Run("accepts an RFC 3339 timestamp", func(t *testing.T) {
		if _, err := resolveDate("2026-08-20T10:30:00Z", ""); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("errors when neither source has a date", func(t *testing.T) {
		if _, err := resolveDate("", "no-date-here"); err == nil {
			t.Error("want an error, got nil")
		}
	})

	t.Run("errors on an unparseable date", func(t *testing.T) {
		if _, err := resolveDate("last tuesday", ""); err == nil {
			t.Error("want an error, got nil")
		}
	})
}

func TestReadingTimeIsAtLeastOneMinute(t *testing.T) {
	if got := readingTime([]byte("three whole words")); got != 1 {
		t.Errorf("readingTime = %d, want 1", got)
	}
}

func TestReadingTimeScalesWithLength(t *testing.T) {
	src := make([]byte, 0, 4400)
	for i := 0; i < 440; i++ {
		src = append(src, []byte("word ")...)
	}
	if got := readingTime(src); got != 2 {
		t.Errorf("readingTime = %d, want 2", got)
	}
}

func TestFirstSentencesSkipsFrontmatterAndHeadings(t *testing.T) {
	src := []byte("---\ntitle: T\n---\n\n## A heading\n\nThe first real paragraph.\n\nSecond.\n")
	if got := firstSentences(src); got != "The first real paragraph." {
		t.Errorf("firstSentences = %q", got)
	}
}

func TestParseFileRequiresATitle(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "2026-08-20-untitled.md")
	if err := os.WriteFile(path, []byte("---\ntags: [go]\n---\n\nBody.\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := ParseFile(path); err == nil {
		t.Error("want an error for a post with no title, got nil")
	}
}

func TestParseFileDerivesSlugAndPath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "2026-08-20-container-layers.md")
	body := "---\ntitle: Layers\ntags: [Docker, docker]\n---\n\nSome body text.\n"
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}

	p, err := ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if p.Slug != "container-layers" {
		t.Errorf("Slug = %q, want container-layers", p.Slug)
	}
	if p.Path != "/posts/container-layers/" {
		t.Errorf("Path = %q", p.Path)
	}
	if !reflect.DeepEqual(p.Tags, []string{"docker"}) {
		t.Errorf("Tags = %v, want [docker]", p.Tags)
	}
	if p.Summary != "Some body text." {
		t.Errorf("Summary = %q", p.Summary)
	}
}

func TestParseDirSortsNewestFirstAndDropsDrafts(t *testing.T) {
	dir := t.TempDir()
	write := func(name, body string) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write("2026-01-01-old.md", "---\ntitle: Old\n---\n\nOld.\n")
	write("2026-08-20-new.md", "---\ntitle: New\n---\n\nNew.\n")
	write("2026-05-05-wip.md", "---\ntitle: WIP\ndraft: true\n---\n\nWIP.\n")
	write("notes.txt", "ignored")

	posts, err := ParseDir(dir, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(posts) != 2 {
		t.Fatalf("got %d posts, want 2 (the draft and the .txt should be skipped)", len(posts))
	}
	if posts[0].Title != "New" || posts[1].Title != "Old" {
		t.Errorf("order = %q, %q; want New, Old", posts[0].Title, posts[1].Title)
	}

	withDrafts, err := ParseDir(dir, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(withDrafts) != 3 {
		t.Errorf("got %d posts with drafts enabled, want 3", len(withDrafts))
	}
}
