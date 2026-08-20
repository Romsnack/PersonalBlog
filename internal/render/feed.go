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

func (s *Site) writeFeed(outDir string) error {
	updated := time.Now().UTC()
	if len(s.Posts) > 0 {
		updated = s.Posts[0].Date.UTC()
	}

	f := atomFeed{
		Title:    s.Config.Title,
		Subtitle: s.Config.Description,
		ID:       s.Config.AbsURL("/"),
		Updated:  updated.Format(time.RFC3339),
		Links: []atomLink{
			{Rel: "alternate", Type: "text/html", Href: s.Config.AbsURL("/")},
			{Rel: "self", Type: "application/atom+xml", Href: s.Config.AbsURL("/atom.xml")},
		},
		Author: atomAuthor{Name: s.Config.Author},
	}

	for _, p := range s.Posts {
		href := s.Config.AbsURL(p.Path)
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
	return os.WriteFile(filepath.Join(outDir, "atom.xml"), append([]byte(xml.Header), b...), 0o644)
}
