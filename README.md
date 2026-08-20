# PersonalBlog

A personal blog and the static site generator that builds it — about 700 lines
of Go, no framework, no JavaScript shipped to readers. Deployed to GitHub Pages
by Actions on every push to `main`.

## Usage

```sh
go run . serve          # preview on http://localhost:8080, rebuilds + reloads on save
go run . build          # render content/ into public/
go run . build -drafts  # include posts with draft: true
```

## Writing a post

Create `content/posts/YYYY-MM-DD-some-slug.md`. The date prefix orders the
directory listing and is stripped from the URL, so that file is served at
`/posts/some-slug/`.

```markdown
---
title: Writing a static site generator in Go
date: 2026-08-20
tags: [go, meta]
summary: Optional. Falls back to the first paragraph.
draft: false
---

Body in Markdown. GFM tables, footnotes and fenced code blocks all work.
```

Standalone pages (about, uses, …) go in `content/pages/` and are served at the
root: `content/pages/about.md` → `/about/`. They appear in the header nav
automatically.

## Layout

| Path | What it is |
| --- | --- |
| `main.go` | CLI: `build` and `serve` |
| `config.yaml` | title, author, description, `baseURL` |
| `internal/config` | config loading; derives the deployment subpath from `baseURL` |
| `internal/content` | Markdown → `Post` (goldmark + frontmatter + chroma) |
| `internal/render` | site model → HTML, `atom.xml`, `sitemap.xml` |
| `internal/render/templates` | embedded `html/template` files |
| `internal/serve` | preview server, file watcher, SSE live reload |
| `static/` | copied verbatim into the output |
| `public/` | build output (gitignored) |

The pipeline runs in one direction — parse, model, write — and every build is a
full rebuild, so there is no cache to invalidate.

## Deployment

Set the repository's **Settings → Pages → Source** to **GitHub Actions**
(not a branch). `.github/workflows/deploy.yml` handles the rest.

### Moving to a custom domain

1. Set `baseURL: https://romsnack.dev` in `config.yaml`.
2. Create `static/CNAME` containing `romsnack.dev` — it ships with every build.
3. DNS: an `ALIAS`/`ANAME` on the apex to `romsnack.github.io` (or GitHub's
   four Pages `A` records), plus a `www` CNAME to the same.
4. Enable **Enforce HTTPS** in Settings → Pages once the certificate issues.

Nothing in the templates hardcodes a host; every link goes through the `url` /
`absURL` helpers, so step 1 is the only code change.

## Design

Dark-first terminal aesthetic: monospace headings and metadata, a readable sans
for body copy, one accent color, a 46rem measure. All of it in
`static/style.css` (~240 lines) and `static/chroma.css` (generated from chroma's
`github-dark` / `github` styles, then trimmed so `style.css` owns the code-block
surface). Both files respond to `prefers-color-scheme`.

## License

Two licenses, because this repository holds two different kinds of work:

- **Code** — the generator, templates and CSS: [MIT](LICENSE). Take it, fork it,
  build your own blog on it.
- **Content** — everything under `content/`: [CC BY 4.0](LICENSE-CONTENT). Quote,
  translate or republish the posts freely, with credit and a link back.
