package render

import (
	"encoding/xml"
	"os"
	"path/filepath"
	"testing"
)

func writeAndParseSitemap(t *testing.T, s *Site) urlSet {
	t.Helper()
	out := t.TempDir()
	if err := s.writeSitemap(out); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(out, "sitemap.xml"))
	if err != nil {
		t.Fatal(err)
	}
	var set urlSet
	if err := xml.Unmarshal(raw, &set); err != nil {
		t.Fatalf("the generated sitemap is not valid XML: %v", err)
	}
	return set
}

func TestSitemapListsEveryPage(t *testing.T) {
	set := writeAndParseSitemap(t, testSite())

	got := map[string]string{}
	for _, u := range set.URLs {
		got[u.Loc] = u.LastMod
	}

	const base = "https://romsnack.github.io/PersonalBlog"
	for _, want := range []string{
		base + "/",
		base + "/tags/",
		base + "/posts/layers/",
		base + "/posts/hello/",
		base + "/about/",
		base + "/tags/containers/",
		base + "/tags/docker/",
	} {
		if _, ok := got[want]; !ok {
			t.Errorf("sitemap is missing %s", want)
		}
	}
	if len(set.URLs) != 7 {
		t.Errorf("got %d URLs, want 7", len(set.URLs))
	}
}

func TestSitemapLastModIsADate(t *testing.T) {
	set := writeAndParseSitemap(t, testSite())

	for _, u := range set.URLs {
		if u.LastMod == "" {
			t.Errorf("%s has no lastmod", u.Loc)
			continue
		}
		if len(u.LastMod) != len("2006-01-02") {
			t.Errorf("%s lastmod = %q, want YYYY-MM-DD", u.Loc, u.LastMod)
		}
	}
}

func TestSitemapOmitsLastModWhenThereAreNoPosts(t *testing.T) {
	s := testSite()
	s.Posts = nil
	s.Tags = nil
	set := writeAndParseSitemap(t, s)

	for _, u := range set.URLs {
		// The root and /tags/ derive their date from the newest post; with no
		// posts the zero time must be omitted rather than serialised.
		if u.Loc == "https://romsnack.github.io/PersonalBlog/" && u.LastMod != "" {
			t.Errorf("root lastmod = %q, want it omitted", u.LastMod)
		}
	}
}
