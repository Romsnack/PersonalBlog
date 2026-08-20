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
| `.golangci.yml` | linter selection and exclusions |
| `.github/workflows/` | CI, security and deploy pipelines |

The pipeline runs in one direction — parse, model, write — and every build is a
full rebuild, so there is no cache to invalidate.

## Checks

Three workflows run on every push and pull request. Everything they do can be
run locally:

```sh
gofmt -l .                  # formatting
go vet ./...                # the standard vet suite
golangci-lint run ./...     # the linters in .golangci.yml
go test -race ./...         # unit tests
govulncheck ./...           # known CVEs, call-graph aware
gosec ./...                 # static analysis for insecure patterns
gitleaks dir .              # committed credentials
zizmor .github/workflows/   # the workflows' own security
```

`Security` also runs weekly on a schedule, because the vulnerability and secret
rule sets change without this repository changing — a commit that was clean in
August can be vulnerable in September.

### How the pipelines are hardened

The workflows have write access to a token and are therefore treated as code:

- **Runner images are pinned** to `ubuntu-24.04`, never `ubuntu-latest`, so a
  GitHub image roll cannot change what runs.
- **Actions are pinned to full commit SHAs**, never tags. A tag is a mutable
  pointer; a maintainer or an attacker who compromises the account can repoint
  `v4` at new code, and every workflow using it picks that up silently.
  Dependabot proposes the bumps as reviewable PRs.
- **Go tooling is installed with `go install tool@vX.Y.Z`** rather than through
  third-party actions. The module proxy and checksum database verify each
  download, and it keeps the set of actions to trust small. The one exception
  is golangci-lint, which needs a newer Go to compile than the one the project
  builds with; it comes from its official action, SHA-pinned like the rest, in
  the mode that downloads a checksum-verified prebuilt binary rather than
  building from source.
- **`GITHUB_TOKEN` is read-only by default.** Jobs widen it themselves, so the
  Pages and OIDC write scopes exist only in the job that deploys.
- **`persist-credentials: false` on every checkout**, so the token is never left
  in `.git/config` for a later step to pick up.
- **No secrets.** Nothing here needs one: Pages authenticates with a short-lived
  OIDC token minted per deployment.

CI builds with the newest 1.25 patch release rather than the exact version in
`go.mod`, whose `go` directive is a minimum language version rather than a build
pin. Standard-library security fixes therefore land without a commit.

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
