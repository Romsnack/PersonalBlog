---
title: Writing a static site generator in Go
date: 2026-08-20
tags: [go, meta]
summary: Why this blog is 600 lines of Go instead of a Hugo theme, and how the pipeline fits together.
---

GitHub Pages serves static files and nothing else, so Go can't run when someone
loads a page. That turns out to be freeing rather than limiting: Go runs at
**build time**, and what it produces is plain HTML that any host on earth can
serve.

## The pipeline

The whole thing runs in one direction, which is the only reason it stays small:

```go
posts, err := content.ParseDir("content/posts", includeDrafts)
if err != nil {
    return err
}

site := &render.Site{Config: cfg, Posts: posts, Tags: indexTags(posts)}
return site.Build("public", "static")
```

Markdown goes in, `[]*Post` comes out, the site model indexes it by tag, and
`Build` writes every page, the feed and the sitemap. There is no cache to
invalidate and no second pass, so a rebuild is always a full rebuild — which
takes a few milliseconds and removes an entire category of bug.

## What each dependency buys

| Module | Job |
| --- | --- |
| goldmark | CommonMark-compliant Markdown |
| goldmark-frontmatter | the YAML block at the top of this file |
| chroma | syntax highlighting, at build time |
| fsnotify | file watching for `go run . serve` |

Highlighting is worth dwelling on. Chroma is configured with `WithClasses(true)`,
so the generated HTML carries class names instead of inline styles and the
actual colors live in `static/chroma.css` next to the rest of the palette. The
consequence: **zero JavaScript ships to readers**. No Prism, no highlight.js, no
flash of unstyled code.

## Writing

`go run . serve` builds the site, serves `public/` on port 8080, and watches
`content/` and `static/`. Save a file and the browser reloads itself over an
`EventSource` connection. The snippet that does it is one line, and it is gated
behind a dev flag so it never appears in a production build.

That's the whole system. Everything else is CSS.
