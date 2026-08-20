package render

import (
	"encoding/xml"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// writeAndParseFeed builds the site's feed in a temp dir and unmarshals it, so
// the assertions run against the same XML a reader would receive.
func writeAndParseFeed(t *testing.T, s *Site) (atomFeed, string) {
	t.Helper()
	out := t.TempDir()
	if err := s.writeFeed(out); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(out, "atom.xml"))
	if err != nil {
		t.Fatal(err)
	}
	var f atomFeed
	if err := xml.Unmarshal(raw, &f); err != nil {
		t.Fatalf("the generated feed is not valid XML: %v", err)
	}
	return f, string(raw)
}

func TestFeedCarriesSiteIdentity(t *testing.T) {
	f, raw := writeAndParseFeed(t, testSite())

	if !strings.HasPrefix(raw, xml.Header) {
		t.Error("feed is missing the XML declaration")
	}
	if f.Title != "Romsnack" {
		t.Errorf("Title = %q", f.Title)
	}
	if f.Author.Name != "Romsnack" {
		t.Errorf("Author = %q", f.Author.Name)
	}
	if f.ID != "https://romsnack.github.io/PersonalBlog/" {
		t.Errorf("ID = %q, want the absolute site root", f.ID)
	}
}

func TestFeedHasAlternateAndSelfLinks(t *testing.T) {
	f, _ := writeAndParseFeed(t, testSite())

	byRel := map[string]string{}
	for _, l := range f.Links {
		byRel[l.Rel] = l.Href
	}
	if byRel["alternate"] != "https://romsnack.github.io/PersonalBlog/" {
		t.Errorf("alternate = %q", byRel["alternate"])
	}
	if byRel["self"] != "https://romsnack.github.io/PersonalBlog/atom.xml" {
		t.Errorf("self = %q", byRel["self"])
	}
}

func TestFeedUpdatedMatchesTheNewestPost(t *testing.T) {
	f, _ := writeAndParseFeed(t, testSite())

	// testSite's newest post is 2026-08-20.
	want := day("2026-08-20").UTC().Format(time.RFC3339)
	if f.Updated != want {
		t.Errorf("Updated = %q, want %q", f.Updated, want)
	}
}

func TestFeedEntriesAreAbsoluteAndOrdered(t *testing.T) {
	f, _ := writeAndParseFeed(t, testSite())

	if len(f.Entries) != 2 {
		t.Fatalf("got %d entries, want 2", len(f.Entries))
	}
	if f.Entries[0].Title != "Layers" {
		t.Errorf("first entry = %q, want the newest post", f.Entries[0].Title)
	}

	e := f.Entries[0]
	if e.ID != "https://romsnack.github.io/PersonalBlog/posts/layers/" {
		t.Errorf("entry ID = %q, want an absolute URL", e.ID)
	}
	if e.Link.Href != e.ID {
		t.Errorf("entry link %q does not match its ID %q", e.Link.Href, e.ID)
	}
	if e.Content.Type != "html" {
		t.Errorf("content type = %q, want html", e.Content.Type)
	}
	if !strings.Contains(e.Content.Body, "<p>Body of Layers.</p>") {
		t.Errorf("entry content = %q, want the post body", e.Content.Body)
	}
}

func TestFeedWithNoPostsIsStillValid(t *testing.T) {
	s := testSite()
	s.Posts = nil
	f, _ := writeAndParseFeed(t, s)

	if len(f.Entries) != 0 {
		t.Errorf("got %d entries, want 0", len(f.Entries))
	}
	// With no posts there is no newest date, so Updated falls back to now.
	if _, err := time.Parse(time.RFC3339, f.Updated); err != nil {
		t.Errorf("Updated = %q, which is not RFC 3339: %v", f.Updated, err)
	}
}
