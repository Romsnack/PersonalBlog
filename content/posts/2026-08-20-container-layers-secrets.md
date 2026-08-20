---
title: Your container deleted the secret. The layer kept it.
date: 2026-08-20
tags: [containers, docker, security]
summary: What an image actually is on disk, why RUN rm never removes anything, and three ways to pull credentials back out of a layer that a running container swears is empty.
---

A container image is not a machine and not a filesystem. It is a stack of
tarballs and a JSON file describing the order to unpack them in. Once you
believe that sentence, a whole class of "how did that key end up on GitHub"
stops being mysterious.

The short version: **layers are append-only**. Deleting a file writes a new
layer that says "ignore that file", and the layer holding the original bytes
stays exactly where it was, shipped to everyone who pulls the image.

## A build that looks fine

Here is a Dockerfile with three mistakes in it. All three are common and none
of them are visible in the finished container.

```dockerfile
FROM alpine:3.20

# The whole build context is copied in, .env and all.
COPY . /app
WORKDIR /app

# The "fix": deleting it in a later layer.
RUN rm -f /app/.env

# A secret passed as a build arg.
ARG NPM_TOKEN
RUN echo "//registry.npmjs.org/:_authToken=${NPM_TOKEN}" > /root/.npmrc \
 && rm -f /root/.npmrc

CMD ["sh"]
```

The `.env` next to it is the usual thing:

```ini
AWS_ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE
AWS_SECRET_ACCESS_KEY=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
```

Build it, run it, and go looking. The container is spotless:

```console
$ docker run --rm leaky-demo:1 sh -c 'ls -a /app; cat /app/.env; cat /root/.npmrc'
.
..
Dockerfile
cat: can't open '/app/.env': No such file or directory
cat: can't open '/root/.npmrc': No such file or directory
```

Both files are gone. Ship it, right?

## How layers are actually built

Each instruction that changes the filesystem produces one layer: a tar archive
of *just what changed* during that step. The runtime stacks them with a union
filesystem, so the container sees the merged result — later layers shadowing
earlier ones.

That shadowing is the whole trick. When a `RUN rm` deletes a file, the union
filesystem cannot reach into a lower layer and edit it, because lower layers
are immutable and shared between images by digest. So it does the only thing it
can: it writes a **whiteout file** into the new layer. An empty marker named
`.wh.<filename>` that means "when you merge the stack, pretend the thing below
me isn't there."

The bytes below are untouched. They are just covered up.

`docker history` shows the layers, newest first — read it bottom to top:

```console
$ docker history leaky-demo:1
IMAGE          CREATED CREATED BY                                      SIZE
4c8b70fe6c4d   ...     /bin/sh -c #(nop)  CMD ["sh"]                   0B
6734cc35b915   ...     |1 NPM_TOKEN=npm_S3cr3tT0k3nExample012345678…   0B
da79965154e7   ...     /bin/sh -c #(nop)  ARG NPM_TOKEN                0B
92d0e72f7de6   ...     /bin/sh -c rm -f /app/.env                      0B
e545a7f805f5   ...     /bin/sh -c #(nop) WORKDIR /app                  0B
9c25a27592af   ...     /bin/sh -c #(nop) COPY dir:e944c2b43443471d0…   8.19kB
d9e853e87e55   4 months ago  CMD ["/bin/sh"]                           0B
<missing>      4 months ago  ADD alpine-minirootfs-3.20.10-x86_64.…    9.44MB
```

Note the `rm -f /app/.env` layer costs **0B**. It didn't reclaim the 8.19kB the
`COPY` added. Deleting a file in a container image makes the image *bigger*,
never smaller.

## Attack 1: the build arg is already visible

Look again at that second line. The truncation is hiding it, so ask for the
full command:

```console
$ docker history leaky-demo:1 --no-trunc --format '{{.CreatedBy}}'
```

```text
|1 NPM_TOKEN=npm_S3cr3tT0k3nExample0123456789 /bin/sh -c echo
"//registry.npmjs.org/:_authToken=${NPM_TOKEN}" > /root/.npmrc && rm -f /root/.npmrc
```

There it is, in plain text, in metadata that ships with the image. No unpacking
required, and `docker history` works against a registry pull. **A build arg is
not a secret.** It is a comment stapled to your image with the value inlined.

This is the cheapest possible check and it costs one command, so it belongs in
your pipeline whether or not you do anything else in this post.

## Attack 2: docker save and go digging

`docker history` only sees metadata. For file contents you need the layers
themselves, and `docker save` streams the whole image as a tar:

```console
$ mkdir unpacked && docker save leaky-demo:1 | tar -x -C unpacked
$ ls unpacked
blobs  index.json  manifest.json  oci-layout
```

`manifest.json` names the config blob and lists the layers in stacking order.
Each entry under `blobs/sha256/` is either JSON metadata or one gzipped layer
tarball.

Now, the trap. The obvious next move fails:

```console
$ grep -rl 'AKIA' unpacked/blobs/sha256/
$
```

Nothing. Not because the key is absent, but because the layers are **gzipped** —
`grep` is scanning compressed bytes. This false negative is worth internalising,
because it is exactly the result that makes people declare an image clean. You
have to decompress first:

```console
$ for f in unpacked/blobs/sha256/*; do
    file -b --mime-type "$f" | grep -q gzip || continue
    gzip -dc "$f" | grep -qa 'AKIA' && echo "HIT: $(basename $f)"
  done
HIT: 862c774e0067e3fae7ab9dd17fdbeb64f5e888ab8f0f0b97545cd60be8fec9b7
```

List that layer against the one stacked on top of it, and the whole story is
right there:

```console
$ gzip -dc unpacked/blobs/sha256/862c774e0067* | tar -t
app/
app/.env
app/Dockerfile

$ gzip -dc unpacked/blobs/sha256/e5019b483794* | tar -t
app/
app/.wh..env
```

Layer two carries `app/.env`. Layer three carries `app/.wh..env` — the whiteout,
the "pretend it's gone" marker, and nothing else. The `rm` produced a tombstone,
not a deletion.

Read the file straight out of the tar, no container involved:

```console
$ gzip -dc unpacked/blobs/sha256/862c774e0067* | tar -xO app/.env
AWS_ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE
AWS_SECRET_ACCESS_KEY=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
```

Full recovery, from the image that showed an empty directory two commands after
it started.

## Attack 3: let dive do it

The manual route is worth doing once so you know what the tooling is doing for
you. After that, use [dive](https://github.com/wagoodman/dive). It reconstructs
the filesystem at every layer and shows you what each step added, changed and
deleted. Interactively:

```sh
dive leaky-demo:1
```

But the interesting mode for a pipeline is `--ci`, which is non-interactive and
returns an exit code:

```console
$ dive leaky-demo:1 --ci
Analyzing image...
  efficiency: 99.9987 %
  wastedBytes: 102 bytes (102 B)
  userWastedPercent: 20.5231 %
Inefficient Files:
Count  Wasted Space  File Path
    2         102 B  /app/.env
Result:FAIL [Total:3] [Passed:1] [Failed:1] [Skipped:1]
```

`Count 2` is the tell: `/app/.env` was touched in two different layers — added
in one, whited out in another. dive is measuring wasted space, not hunting
secrets, but **"this file exists in one layer and is deleted in a later one" is
the exact shape of a leaked credential**, which makes an efficiency tool a
surprisingly good detector.

If you don't want dive installed locally, it runs from its own image:

```sh
docker run --rm -v /var/run/docker.sock:/var/run/docker.sock \
  wagoodman/dive:latest leaky-demo:1 --ci
```

Rootless Docker puts its socket at `/run/user/$(id -u)/docker.sock` — mount that
path instead, or dive will tell you the daemon isn't running when it is.

## Fixing it

Three mistakes, three fixes, and none of them is a better `rm`.

**Keep it out of the context.** `COPY . /app` copies whatever is in the
directory, including files you forgot about. A `.dockerignore` stops them at the
door:

```text
.env
.git
```

`.git` matters as much as `.env` — it carries every secret anyone ever committed
and reverted.

**Mount build-time secrets instead of passing them.** BuildKit exposes a file
under `/run/secrets/` for the lifetime of one `RUN` and never records it in a
layer or in the history:

```dockerfile
RUN --mount=type=secret,id=npm_token \
    npm config set //registry.npmjs.org/:_authToken="$(cat /run/secrets/npm_token)" \
 && npm ci
```

```sh
docker build --secret id=npm_token,src=token.txt -t app:1 .
```

**Use multi-stage builds.** Do the work that needs credentials in a builder
stage and `COPY --from=builder` only the artifact. Layers you don't copy from
never reach the final image.

Rebuilt with all three, the same audit comes back empty — no `.env` in any
layer, and the history records the mount path rather than the token:

```console
$ docker history fixed-demo:2 --no-trunc --format '{{.CreatedBy}}' | head -2
CMD ["sh"]
RUN /bin/sh -c echo "//registry.npmjs.org/:_authToken=$(cat /run/secrets/npm_token)" ... # buildkit
```

The literal `$(cat ...)` is what got stored. The value never existed anywhere
but in that one command's memory.

## The part people skip

If a secret ever reached a published layer, **rebuilding does not fix it.**
Anyone who pulled that tag still has the bytes, and registries keep blobs
addressed by digest even after a tag moves. Rewriting the image is cleanup, not
remediation.

Rotate the credential. Then rebuild.

That's the whole thing: images are append-only, `rm` is a tombstone, and three
commands will tell you what you actually shipped.
