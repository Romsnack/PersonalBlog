package render

import (
	"encoding/xml"
	"os"
	"path/filepath"
	"time"
)

// Atom rather than RSS: dates are unambiguously RFC 3339 and every reader
// supports it.
type atomFeed struct {
	XMLName  xml.Name    `xml:"http://www.w3.org/2005/Atom feed"`
	Title    string      `xml:"title"`
	Subtitle string      `xml:"subtitle,omitempty"`
	ID       string      `xml:"id"`
	Updated  string      `xml:"updated"`
	Lang     string      `xml:"http://www.w3.org/XML/1998/namespace lang,attr,omitempty"`
	Links    []atomLink  `xml:"link"`
	Author   atomAuthor  `xml:"author"`
	Entries  []atomEntry `xml:"entry"`
}

type atomLink struct {
	Rel  string `xml:"rel,attr,omitempty"`
	Type string `xml:"type,attr,omitempty"`
	Href string `xml:"href,attr"`
}

type atomAuthor struct {
	Name string `xml:"name"`
}

type atomEntry struct {
	Title     string      `xml:"title"`
	ID        string      `xml:"id"`
	Updated   string      `xml:"updated"`
	Published string      `xml:"published"`
	Link      atomLink    `xml:"link"`
	Summary   string      `xml:"summary"`
	Content   atomContent `xml:"content"`
}

type atomContent struct {
	Type string `xml:"type,attr"`
	Body string `xml:",cdata"`
}

// writeFeed writes one language's feed under that language's prefix:
// /atom.xml for the default language, /fr/atom.xml for the rest. A reader
// subscribes to the edition they read, not to a mixed-language stream.
func (e *Edition) writeFeed(outDir string) error {
	cfg := e.Site.Config
	lang := e.Lang.Code

	updated := time.Now().UTC()
	if len(e.Posts) > 0 {
		updated = e.Posts[0].Date.UTC()
	}

	home := cfg.LangAbsURL(lang, "/")
	f := atomFeed{
		Title:    cfg.Title,
		Subtitle: e.Description(),
		ID:       home,
		Updated:  updated.Format(time.RFC3339),
		Lang:     lang,
		Links: []atomLink{
			{Rel: "alternate", Type: "text/html", Href: home},
			{Rel: "self", Type: "application/atom+xml", Href: cfg.LangAbsURL(lang, "/atom.xml")},
		},
		Author: atomAuthor{Name: cfg.Author},
	}

	for _, p := range e.Posts {
		href := cfg.LangAbsURL(lang, p.Path)
		f.Entries = append(f.Entries, atomEntry{
			Title:     p.Title,
			ID:        href,
			Updated:   p.Date.UTC().Format(time.RFC3339),
			Published: p.Date.UTC().Format(time.RFC3339),
			Link:      atomLink{Rel: "alternate", Type: "text/html", Href: href},
			Summary:   p.Summary,
			Content:   atomContent{Type: "html", Body: string(p.HTML)},
		})
	}

	b, err := xml.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	dst := e.outPath(outDir, "/atom.xml")
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	return os.WriteFile(dst, append([]byte(xml.Header), b...), 0o644)
}
