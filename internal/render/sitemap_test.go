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

const base = "https://romsnack.github.io/PersonalBlog"

// One sitemap covers every language, which is what the protocol expects.
func TestSitemapListsEveryPageInEveryLanguage(t *testing.T) {
	set := writeAndParseSitemap(t, testSite())

	got := map[string]bool{}
	for _, u := range set.URLs {
		got[u.Loc] = true
	}

	for _, want := range []string{
		base + "/",
		base + "/tags/",
		base + "/posts/layers/",
		base + "/posts/hello/",
		base + "/about/",
		base + "/tags/containers/",
		base + "/tags/docker/",
		base + "/fr/",
		base + "/fr/tags/",
		base + "/fr/posts/couches/",
		base + "/fr/a-propos/",
		base + "/fr/tags/containers/",
		base + "/fr/tags/docker/",
	} {
		if !got[want] {
			t.Errorf("sitemap is missing %s", want)
		}
	}
	// 7 English URLs and 6 French ones — French has no counterpart for /hello/.
	if len(set.URLs) != 13 {
		t.Errorf("got %d URLs, want 13", len(set.URLs))
	}
}

func TestSitemapPairsTranslationsAsAlternates(t *testing.T) {
	set := writeAndParseSitemap(t, testSite())

	byLoc := map[string]sitemapURL{}
	for _, u := range set.URLs {
		byLoc[u.Loc] = u
	}

	// A translated post carries an alternate for each language, itself
	// included — which is what the protocol asks for.
	got := map[string]string{}
	for _, a := range byLoc[base+"/posts/layers/"].Alternates {
		if a.Rel != "alternate" {
			t.Errorf("rel = %q, want alternate", a.Rel)
		}
		got[a.HrefLang] = a.Href
	}
	if got["en"] != base+"/posts/layers/" {
		t.Errorf("en alternate = %q", got["en"])
	}
	if got["fr"] != base+"/fr/posts/couches/" {
		t.Errorf("fr alternate = %q", got["fr"])
	}

	// A post that exists in one language only has nothing to point at, so it
	// must not claim an alternate.
	if alts := byLoc[base+"/posts/hello/"].Alternates; len(alts) != 0 {
		t.Errorf("an untranslated post has %d alternates, want 0", len(alts))
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
	for _, ed := range s.Editions {
		ed.Posts = nil
		ed.Tags = nil
	}
	set := writeAndParseSitemap(t, s)

	for _, u := range set.URLs {
		// The root and /tags/ derive their date from the newest post; with no
		// posts the zero time must be omitted rather than serialised.
		if u.Loc == base+"/" && u.LastMod != "" {
			t.Errorf("root lastmod = %q, want it omitted", u.LastMod)
		}
	}
}
