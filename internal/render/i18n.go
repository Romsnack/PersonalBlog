package render

import "strings"

// The chrome around the content — nav labels, the "back" links, the empty
// states — is not in any Markdown file, so it lives here. One map per
// language, keyed identically; templates reach it through the `t` func.
//
// Adding a language means adding an entry with the same keys. A missing key
// falls back to the default language rather than rendering blank, so a
// half-translated site degrades to English instead of to nothing.
var uiStrings = map[string]map[string]string{
	"en": {
		"nav.posts":     "posts",
		"nav.tags":      "tags",
		"tags.title":    "Tags",
		"tags.empty":    "No tags yet.",
		"posts.empty":   "No posts yet.",
		"back.posts":    "← all posts",
		"back.tags":     "← all tags",
		"readingTime":   "min",
		"theme.dark":    "dark",
		"theme.light":   "light",
		"theme.toDark":  "Switch to dark theme",
		"theme.toLight": "Switch to light theme",
		"lang.label":    "Language",
		"feed":          "atom",
	},
	"fr": {
		"nav.posts":     "articles",
		"nav.tags":      "tags",
		"tags.title":    "Tags",
		"tags.empty":    "Aucun tag pour l'instant.",
		"posts.empty":   "Aucun article pour l'instant.",
		"back.posts":    "← tous les articles",
		"back.tags":     "← tous les tags",
		"readingTime":   "min",
		"theme.dark":    "sombre",
		"theme.light":   "clair",
		"theme.toDark":  "Passer au thème sombre",
		"theme.toLight": "Passer au thème clair",
		"lang.label":    "Langue",
		"feed":          "atom",
	},
}

// translate looks up one UI string. It tries the exact language, then the base
// of a regional tag ("fr-CA" -> "fr"), then the site default, and finally
// returns the key itself so a typo is visible on the page rather than silent.
func translate(lang, defaultLang, key string) string {
	for _, l := range []string{lang, baseTag(lang), defaultLang, baseTag(defaultLang)} {
		if s, ok := uiStrings[l][key]; ok && s != "" {
			return s
		}
	}
	return key
}

// baseTag reduces a regional tag to its language subtag: "pt-BR" -> "pt".
func baseTag(lang string) string {
	if i := strings.IndexAny(lang, "-_"); i > 0 {
		return lang[:i]
	}
	return lang
}
