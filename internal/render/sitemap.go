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
}

func (s *Site) writeSitemap(outDir string) error {
	var set urlSet

	add := func(path string, mod time.Time) {
		u := sitemapURL{Loc: s.Config.AbsURL(path)}
		if !mod.IsZero() {
			u.LastMod = mod.UTC().Format("2006-01-02")
		}
		set.URLs = append(set.URLs, u)
	}

	var newest time.Time
	if len(s.Posts) > 0 {
		newest = s.Posts[0].Date
	}
	add("/", newest)
	add("/tags/", newest)
	for _, p := range s.Posts {
		add(p.Path, p.Date)
	}
	for _, p := range s.Pages {
		add(p.Path, p.Date)
	}
	for _, t := range s.Tags {
		add(t.Path, t.Posts[0].Date)
	}

	b, err := xml.MarshalIndent(set, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(outDir, "sitemap.xml"), append([]byte(xml.Header), b...), 0o644)
}
