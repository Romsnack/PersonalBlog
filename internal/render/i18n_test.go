package render

import "testing"

func TestTranslateUsesTheRequestedLanguage(t *testing.T) {
	if got := translate("fr", "en", "nav.posts"); got != "articles" {
		t.Errorf("fr nav.posts = %q, want articles", got)
	}
	if got := translate("en", "en", "nav.posts"); got != "posts" {
		t.Errorf("en nav.posts = %q, want posts", got)
	}
}

// A half-translated site should degrade to the default language rather than
// render blanks, so an unknown language is not a broken page.
func TestTranslateFallsBackToTheDefaultLanguage(t *testing.T) {
	if got := translate("de", "en", "nav.posts"); got != "posts" {
		t.Errorf("de nav.posts = %q, want the English fallback", got)
	}
}

// A regional tag should find its base language's strings.
func TestTranslateReducesARegionalTag(t *testing.T) {
	if got := translate("fr-CA", "en", "nav.posts"); got != "articles" {
		t.Errorf("fr-CA nav.posts = %q, want the French string", got)
	}
}

// An unknown key returns itself, so a typo is visible on the page instead of
// silently rendering an empty element.
func TestTranslateReturnsAnUnknownKeyVerbatim(t *testing.T) {
	if got := translate("en", "en", "nav.nope"); got != "nav.nope" {
		t.Errorf("unknown key = %q, want the key itself", got)
	}
}

// Every language has to define the same keys, or a page in one language would
// silently show another's wording.
func TestEveryLanguageDefinesTheSameKeys(t *testing.T) {
	for lang, strs := range uiStrings {
		for key := range uiStrings["en"] {
			if strs[key] == "" {
				t.Errorf("language %q is missing the key %q", lang, key)
			}
		}
		for key := range strs {
			if uiStrings["en"][key] == "" {
				t.Errorf("language %q defines %q, which en does not", lang, key)
			}
		}
	}
}
