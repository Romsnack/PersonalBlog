---
title: Your lockfile doesn't know which registry a name came from
date: 2026-10-26
tags: [supply-chain, npm, security]
summary: How package managers actually resolve a name to a package, why an internal package name without a reserved public counterpart is a public one, and a full reproduction — two real registries, one package name, the highest version wins — with nothing in either package but a line that says where it came from.
translationKey: dependency-confusion
---

In February 2021, security researcher Alex Birsan published [a writeup](https://medium.com/@alex.birsan/dependency-confusion-4a5d60fec610)
describing how he got code to run inside the networks of Apple, Microsoft,
Tesla, Uber and dozens of other companies — not by exploiting a
vulnerability in their code, but by publishing packages with the right
*names* to the public npm, PyPI and RubyGems registries. Some of those
internal package names leaked from `package.json` files, Slack messages and
error logs scraped off the open internet. He never touched a single internal
system. The package managers did the rest on their own, because that's
exactly what they're designed to do: find the newest version of a name,
wherever it lives.

That's the whole attack. No exploit, no malware delivery mechanism, no
phishing. Just a name collision and a resolver doing its job correctly.

## What resolution actually does

A package manager doesn't know what "internal" means. When you write
`@acme/build-tools` in a `package.json`, npm's only job is: find something
that answers to that name, matching that version range, and install the
newest match. By default it asks one registry — whichever one is configured
— and that registry doesn't care who you are. If a company never explicitly
tells the resolver *this name lives only here*, the resolver will happily
accept it from anywhere that responds first or wins on version number.

<figure class="diagram">
<svg viewBox="0 0 720 220" role="img" aria-labelledby="fig1t">
  <title id="fig1t">The intended flow: a scope mapping sends the internal package name to the internal registry specifically, bypassing the public one entirely.</title>
  <defs><marker id="b1" viewBox="0 0 8 8" refX="7" refY="4" markerWidth="6" markerHeight="6" orient="auto"><path d="M0,0 L8,4 L0,8 z" fill="currentColor"/></marker></defs>
  <rect class="d-box" x="8" y="16" width="160" height="40" rx="4"/>
  <text class="d-t d-b" x="88" y="40" text-anchor="middle">npm install</text>
  <rect class="d-box-a" x="280" y="16" width="200" height="40" rx="4"/>
  <text class="d-a d-b" x="380" y="40" text-anchor="middle">@acme:registry</text>
  <rect class="d-box" x="560" y="16" width="152" height="40" rx="4"/>
  <text class="d-t d-b" x="636" y="40" text-anchor="middle">public index</text>
  <g class="d-arrow-a" marker-end="url(#b1)">
    <path d="M168,36 H275"/>
  </g>
  <text class="d-m" x="222" y="28" text-anchor="middle">@acme/build-tools</text>
  <g class="d-dash">
    <path d="M480,36 H555"/>
  </g>
  <text class="d-m" x="518" y="28" text-anchor="middle">everything else</text>
  <rect class="d-box-a" x="280" y="110" width="200" height="60" rx="4"/>
  <text class="d-a d-b" x="380" y="134" text-anchor="middle">internal registry</text>
  <text class="d-m" x="380" y="152" text-anchor="middle">@acme/build-tools@1.2.0</text>
  <rect class="d-box" x="560" y="110" width="152" height="60" rx="4"/>
  <text class="d-t d-b" x="636" y="134" text-anchor="middle">public index</text>
  <text class="d-m" x="636" y="152" text-anchor="middle">@acme/build-tools@99.0.0</text>
  <g class="d-arrow-a" marker-end="url(#b1)">
    <path d="M380,56 V105"/>
  </g>
  <g class="d-dash">
    <path d="M636,56 V105"/>
  </g>
  <text class="d-m" x="380" y="196" text-anchor="middle">installed — the only candidate asked for</text>
  <text class="d-m" x="636" y="196" text-anchor="middle">never queried for this name</text>
</svg>
<figcaption>With a scope mapping in place, the resolver never even asks the public index about <code>@acme/*</code> — there's no race to win because there's no contest.</figcaption>
</figure>

Without that mapping, there's only one registry configured, and it will
answer for *any* name you ask it about — including the one you meant to keep
private.

## Reproducing it, with nothing dangerous in either package

To show the mechanism rather than just describe it, here's a full
reproduction using two throwaway registries on my own machine: [Gitea](https://about.gitea.com/)
running its built-in npm package registry (standing in for a company's
internal registry) and a local [Verdaccio](https://verdaccio.org/) instance
(standing in for the public npm registry — so nothing in this demo ever
touches the real npm). Both packages do exactly one thing on install: print a
line saying which registry they came from. No file access, no network calls,
no payload of any kind — the entire point is the *name resolution*, not what
a malicious package could do once it wins, which is a different post.

**The "legitimate" internal package**, published to Gitea under version
`1.2.0`:

```json
{
  "name": "@acme/build-tools",
  "version": "1.2.0",
  "scripts": { "postinstall": "node ./announce.js" }
}
```

```js
// announce.js
console.log("[@acme/build-tools@1.2.0] installed from the INTERNAL registry (Gitea).");
```

```console
$ npm publish
npm notice 📦  @acme/build-tools@1.2.0
npm notice Publishing to http://localhost:3000/api/packages/acme/npm/ with tag latest and default access
+ @acme/build-tools@1.2.0
```

**The "squatted" package**, published to the public stand-in under the exact
same name, with a higher version number — the one piece of information that
decides everything:

```json
{
  "name": "@acme/build-tools",
  "version": "99.0.0",
  "scripts": { "postinstall": "node ./announce.js" }
}
```

```js
// announce.js
console.log("[@acme/build-tools@99.0.0] installed from the PUBLIC registry (attacker-controlled) — THIS IS THE CONFUSION.");
```

```console
$ npm publish
npm notice 📦  @acme/build-tools@99.0.0
npm notice Publishing to http://localhost:4873/ with tag latest and default access
+ @acme/build-tools@99.0.0
```

Now the vulnerable side: a project whose `.npmrc` points at a single
registry, with no scope mapping for `@acme` at all — which is exactly what a
default, never-configured `.npmrc` looks like.

```ini
# .npmrc
registry=http://localhost:4873/
```

```json
{
  "dependencies": { "@acme/build-tools": "*" }
}
```

```console
$ npm install --foreground-scripts

> @acme/build-tools@99.0.0 postinstall
> node ./announce.js

[@acme/build-tools@99.0.0] installed from the PUBLIC registry (attacker-controlled) — THIS IS THE CONFUSION.

added 1 package, and audited 2 packages in 468ms
```

<figure class="diagram">
<svg viewBox="0 0 720 196" role="img" aria-labelledby="fig2t">
  <title id="fig2t">Without a scope mapping, the single configured registry answers for the internal name too, and the higher version wins regardless of which side published it.</title>
  <defs><marker id="b2" viewBox="0 0 8 8" refX="7" refY="4" markerWidth="6" markerHeight="6" orient="auto"><path d="M0,0 L8,4 L0,8 z" fill="currentColor"/></marker></defs>
  <rect class="d-box" x="8" y="16" width="160" height="40" rx="4"/>
  <text class="d-t d-b" x="88" y="40" text-anchor="middle">npm install</text>
  <text class="d-m" x="88" y="66" text-anchor="middle">"@acme/build-tools": "*"</text>
  <rect class="d-box" x="280" y="16" width="200" height="40" rx="4"/>
  <text class="d-t d-b" x="380" y="40" text-anchor="middle">registry=…</text>
  <text class="d-m" x="380" y="66" text-anchor="middle">one registry, no scope rule</text>
  <g class="d-arrow" marker-end="url(#b2)">
    <path d="M168,36 H275"/>
  </g>
  <rect class="d-box-a" x="560" y="90" width="152" height="52" rx="4"/>
  <text class="d-a d-b" x="636" y="112" text-anchor="middle">public index</text>
  <text class="d-m" x="636" y="128" text-anchor="middle">99.0.0 — highest</text>
  <g class="d-arrow-a" marker-end="url(#b2)">
    <path d="M380,56 V80 Q380,96 440,100 T556,110"/>
  </g>
  <rect class="d-box" x="280" y="90" width="200" height="52" rx="4" opacity="0.5"/>
  <text class="d-m" x="380" y="112" text-anchor="middle">internal registry 1.2.0</text>
  <text class="d-m" x="380" y="128" text-anchor="middle">never asked</text>
  <text class="d-m" x="636" y="168" text-anchor="middle">installed — postinstall runs as this user</text>
</svg>
<figcaption>"*" means "any version" to the resolver, and the only candidate it ever sees is whatever this one registry hands back. The name being internal is a convention the resolver was never told about.</figcaption>
</figure>

That `postinstall` line is not cosmetic. npm, like most package managers, runs
lifecycle scripts automatically on install, with the permissions of whoever
ran the install — which in a CI pipeline is usually a service account with
access to source, secrets and a deploy path. Swap the `console.log` for
anything else and the "demo" becomes the incident; that line is doing all the
work Birsan's real packages did.

## The fix: tell the resolver what it was never told

```ini
# .npmrc
registry=http://localhost:4873/
@acme:registry=http://localhost:3000/api/packages/acme/npm/
```

```console
$ npm install

$ cat node_modules/@acme/build-tools/package.json | grep version
  "version": "1.2.0",
```

Same project, same `"*"` range, same two registries both still holding a
package called `@acme/build-tools` — but the scope mapping means the public
registry is never even asked about that name. There's no race to lose
because there's no race.

That one line generalizes across ecosystems, though the exact mechanism
differs and the gotchas are different too:

- **npm** — map every internal scope explicitly (`@yourorg:registry=…`), for
  every `.npmrc` that matters: developer machines, CI, and any base Docker
  image that runs `npm install`.
- **pip** — `--extra-index-url` is the classic trap: pip queries *all*
  configured indexes and, like npm, takes the highest version it finds
  across all of them — it does not prefer the first index listed. Use
  `--index-url` pointed only at your internal mirror, with the public
  registry proxied *through* it (most internal registries — Artifactory,
  Nexus, Gitea's own index support — can do this), rather than queried
  alongside it.
- **Any registry that supports it** — reserve the internal names on the
  public registry too, even as empty placeholder packages. It's the one
  defense that doesn't depend on every config file downstream being correct.
- **Everywhere** — commit the lockfile and install from it (`npm ci`, not
  `npm install`, in CI) with a registry that enforces integrity hashes. A
  lockfile pins a version *and* the hash of the exact bytes it resolved to
  last time, so a same-name, same-version swap on a registry still fails the
  hash check — though it does nothing for the first install that created the
  lockfile, which is why the scope mapping or single-index fix has to exist
  independently.

## What people skip

Reserving names on the public registry gets talked about and rarely actually
done, because it means someone has to enumerate every internal package name
your company has ever used — including the ones nobody remembers, from
projects that shipped once and were never touched again. Those are exactly
the names most likely to still be sitting in an old `package-lock.json`
checked into a public fork of something, waiting.

The underlying fact doesn't go away by fixing the config, either: an internal
package name only means "internal" inside your own tooling's configuration.
To every registry, and to anyone who can read that configuration, it's just a
string, and strings are free to register.
