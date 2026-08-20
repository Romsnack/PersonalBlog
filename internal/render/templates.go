// Package render turns parsed posts into a complete static site.
package render

import (
	"embed"
	"html/template"
	"strings"
	"time"

	"github.com/Romsnack/PersonalBlog/internal/config"
)

//go:embed templates/*.html
var templateFS embed.FS

// parseTemplates parses every template in templates/ against one FuncMap.
// Each page template defines the "content" block that base.html renders.
func parseTemplates(cfg *config.Config) (map[string]*template.Template, error) {
	funcs := template.FuncMap{
		"url":     cfg.URL,
		"absURL":  cfg.AbsURL,
		"date":    func(t time.Time) string { return t.Format("2006-01-02") },
		"rfc3339": func(t time.Time) string { return t.Format(time.RFC3339) },
		"year":    func() int { return time.Now().Year() },
		"join":    strings.Join,
	}

	pages := []string{"index.html", "post.html", "tag.html", "tags.html", "page.html"}
	out := make(map[string]*template.Template, len(pages))
	for _, name := range pages {
		t, err := template.New("base.html").Funcs(funcs).
			ParseFS(templateFS, "templates/base.html", "templates/"+name)
		if err != nil {
			return nil, err
		}
		out[name] = t
	}
	return out, nil
}
