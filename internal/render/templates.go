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
//
// The set is parsed once per language, and `url`, `absURL` and `t` are closed
// over that language. A template can therefore write {{url .Path}} and get the
// right edition's URL without ever naming a language.
func parseTemplates(cfg *config.Config, lang string) (map[string]*template.Template, error) {
	funcs := template.FuncMap{
		"url":    func(p string) string { return cfg.LangURL(lang, p) },
		"absURL": func(p string) string { return cfg.LangAbsURL(lang, p) },
		// asset is for files copied out of static/. They are shared by every
		// language and exist only at the root, so they must never take the
		// language prefix that url would add.
		"asset":   cfg.URL,
		"t":       func(key string) string { return translate(lang, cfg.DefaultLanguage, key) },
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
