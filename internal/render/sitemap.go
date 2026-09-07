package render

import (
	"encoding/xml"
	"os"
	"path/filepath"
	"time"
)

type urlSet struct {
	XMLName xml.Name     `xml:"http://www.sitemaps.org/schemas/sitemap/0.9 urlset"`
	URLs    []sitemapURL `xml:"url"`
}

type sitemapURL struct {
	Loc     string `xml:"loc"`
	LastMod string `xml:"lastmod,omitempty"`
	// Alternates are the same page in the other languages. Search engines use
	// these to serve a reader the edition in their own language instead of
	// treating the translations as duplicate content.
	Alternates []xhtmlLink `xml:"http://www.w3.org/1999/xhtml link,omitempty"`
}

type xhtmlLink struct {
	Rel      string `xml:"rel,attr"`
	HrefLang string `xml:"hreflang,attr"`
	Href     string `xml:"href,attr"`
}

// writeSitemap writes one sitemap covering every language, which is what the
// protocol expects: a single file listing every URL on the host.
func (s *Site) writeSitemap(outDir string) error {
	var set urlSet

	index := make(map[string]map[string]string, len(s.Editions))
	for _, ed := range s.Editions {
		index[ed.Lang.Code] = ed.paths()
	}

	// alternates lists a page in every language that actually has it. A page
	// with no translation gets none, rather than pointing at a homepage.
	alternates := func(key string) []xhtmlLink {
		var out []xhtmlLink
		for _, ed := range s.Editions {
			path, ok := index[ed.Lang.Code][key]
			if !ok {
				continue
			}
			out = append(out, xhtmlLink{
				Rel:      "alternate",
				HrefLang: ed.Lang.Code,
				Href:     s.Config.LangAbsURL(ed.Lang.Code, path),
			})
		}
		if len(out) < 2 {
			return nil
		}
		return out
	}

	for _, ed := range s.Editions {
		lang := ed.Lang.Code

		add := func(path, key string, mod time.Time) {
			u := sitemapURL{
				Loc:        s.Config.LangAbsURL(lang, path),
				Alternates: alternates(key),
			}
			if !mod.IsZero() {
				u.LastMod = mod.UTC().Format("2006-01-02")
			}
			set.URLs = append(set.URLs, u)
		}

		var newest time.Time
		if len(ed.Posts) > 0 {
			newest = ed.Posts[0].Date
		}
		add("/", "index", newest)
		add("/tags/", "tags", newest)
		for _, p := range ed.Posts {
			add(p.Path, "post:"+p.TranslationKey, p.Date)
		}
		for _, p := range ed.Pages {
			add(p.Path, "page:"+p.TranslationKey, p.Date)
		}
		for _, t := range ed.Tags {
			add(t.Path, "tag:"+t.Name, t.Posts[0].Date)
		}
	}

	b, err := xml.MarshalIndent(set, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(outDir, "sitemap.xml"), append([]byte(xml.Header), b...), 0o644)
}
