# Template audit

- **Last audited:** 2026-08-04
- **Audited by:** Onklave platform maintenance (automated, Claude Code)
- **Next review due:** 2026-11-04 (quarterly, or sooner on a dependency alert)

## Why this file exists
So we know when this template was last deliberately checked, and what was true at
the time. Apps are generated from this repo — stale or vulnerable dependencies
here propagate to every app created from it.

## Scope of this audit
- Go toolchain currency and the `go.mod` ↔ Dockerfile version drift.
- Third-party dependency currency (`go list -m -u all`) and known vulnerabilities
  (`govulncheck`, stdlib included).
- Base image currency for both stages (`golang:1.26-alpine`, `gcr.io/distroless/static:nonroot`).
- Dockerfile security posture: non-root user, shell-less final image, static
  binary, `HEALTHCHECK` correctness, build-context hygiene.
- Runtime hardening compatibility: read-only root filesystem, `cap-drop=ALL`,
  `no-new-privileges`, graceful SIGTERM shutdown.
- HTTP server hardening: timeouts and header limits (slowloris exposure).
- `.dockerignore` / `.gitignore` correctness.
- Secrets: full git history scanned for committed credentials.

Not in scope: `onklave.yaml` (written and verified in a previous pass, deliberately
untouched).

## Verification run
Go is not installed on the audit host, so every Go command was executed inside the
pinned `golang:1.26-alpine` image (go1.26.5) against a bind mount of the repo —
i.e. the same toolchain the Dockerfile builds with. Commands are recorded as run.

| Check | Command | Result |
|---|---|---|
| Vet (baseline, before changes) | `go vet ./...` | Pass — exit 0, no diagnostics |
| Test (baseline, before changes) | `go test ./...` | Pass — `ok github.com/onklave/template-container 0.002s` |
| Build (baseline, before changes) | `go build ./...` | Pass — exit 0 |
| Vet (final) | `go vet ./...` | Pass — exit 0, no diagnostics |
| Test (final) | `go test ./...` | Pass — 6/6 tests pass, exit 0 |
| Build (final) | `go build ./...` | Pass — exit 0 |
| Cross-toolchain build | `go vet/test/build` under `golang:1.25-alpine`, `GOTOOLCHAIN=local` | Pass — go1.25.12, no toolchain download needed |
| Dependency updates | `go list -m -u all` | Only the main module — no third-party dependencies, nothing to update |
| Vulnerability scan | `govulncheck ./...` (v1.6.0, DB 2026-07-27) | **No vulnerabilities found** — exit 0 |
| Image build | `docker build -t template-container .` | Pass — exit 0, final image 14.7 MB |
| Image config | `docker image inspect` | `User=nonroot:nonroot`, `Entrypoint=[/app]`, `ExposedPorts=8080/tcp` |
| Hardened runtime | `docker run --read-only --cap-drop=ALL --security-opt=no-new-privileges` | Pass — starts and serves normally |
| Endpoint `/healthz` | `curl :8080/healthz` | `200` / body `ok` |
| Endpoint `/` | `curl :8080/` | `200` / greeting |
| Endpoint `/nope` | `curl :8080/nope` | `404` |
| Docker HEALTHCHECK | `docker inspect --format '{{.State.Health.Status}}'` | `healthy` |
| Slowloris probe | raw socket, partial headers, never terminated | Connection closed by server after 5.0s — `ReadHeaderTimeout` enforced |
| Graceful shutdown | `docker stop -t 15` | `shutting down` → `stopped`, exit code 0 |
| Secret scan | `git grep -E '<credential patterns>' $(git rev-list --all)` | No matches across all history |
| GODEBUG baseline (directive `1.25`) | `go version -m /out/app` | `build DefaultGODEBUG=cryptocustomrand=1,tlssecpmlkem=0,urlstrictcolons=0` |
| GODEBUG baseline (directive `1.26`) | `go version -m /out/app` | **No `DefaultGODEBUG` line** — binary runs on 1.26 defaults |
| Vet / test / build (after raising to `1.26`) | `go vet ./...`, `go test ./...`, `go build ./...` | Pass — exit 0, `ok github.com/onklave/template-container 0.003s` |

`govulncheck` was installed with `go install golang.org/x/vuln/cmd/govulncheck@latest`
inside the build container; it installed and ran cleanly.

The last three rows are from the **cross-template reconciliation pass** (2026-08-04,
same audit date): seven templates were audited independently and made inconsistent
choices, which were then reconciled. See finding 9.

## Dependency status
**Third-party dependencies: none.** `go.mod` has no `require` block and the service
is pure standard library, so there is no dependency tree to upgrade and no `go.sum`
to verify. `go list -m -u all` returns only `github.com/onklave/template-container`.
The entire third-party attack surface of this template is therefore the two base
images plus the Go standard library.

| Item | Before | After | Note |
|---|---|---|---|
| `go` directive in `go.mod` | `go 1.22` | `go 1.26` | Go 1.22 is end-of-life; `1.26` matches the builder (see below) |
| Build image | `golang:1.26-alpine` | `golang:1.26-alpine` (unchanged) | Already current — resolves to go1.26.5 |
| Runtime image | `gcr.io/distroless/static:nonroot` | unchanged | Already current — re-pulled, `uid 65532` |
| Go stdlib (via builder) | go1.26.5 | go1.26.5 | Clean under govulncheck |

**Toolchain drift — assessed and resolved.** `go.mod` declared `go 1.22` while the
Dockerfile built with `golang:1.26-alpine`; the deleted CI pinned setup-go to 1.22,
so nothing ever forced the two to agree. Raising the directive is safe here because
the module has zero dependencies and uses only long-stable stdlib APIs, so there is
no transitive constraint to satisfy.

Initially raised to **1.25 rather than 1.26**, on the reasoning that the directive is
a hard *minimum* and `1.25` states the supported floor: satisfied by the 1.26 builder
the image actually uses, while still building for a developer on 1.25 with
`GOTOOLCHAIN=local` (verified above). Pinning to `1.26` costs those developers a
toolchain download — or a hard failure under `GOTOOLCHAIN=local`.

**Superseded — raised again to `1.26` in the reconciliation pass.** That reasoning
treated the directive as *only* a language floor, which is the part it got wrong: the
directive also fixes the GODEBUG compatibility baseline compiled into the binary, so a
directive behind the builder silently bakes in opt-outs from the newer release's
hardening defaults. Measured directly on this module by diffing `go version -m` across
the two directives (both built in `golang:1.26-alpine`, go1.26.5):

| GODEBUG baked in at `go 1.25` | Effect of the opt-out |
|---|---|
| `cryptocustomrand=1` | Honours a caller-replaced `crypto/rand.Reader` instead of always using the OS CSPRNG |
| `tlssecpmlkem=0` | Disables the hybrid post-quantum TLS key exchange, leaving sessions harvest-now-decrypt-later exposed |
| `urlstrictcolons=0` | Keeps the lax URL colon parsing, the classic source of parser-differential bugs between proxy and app |

At `go 1.26` the `DefaultGODEBUG` line disappears from the binary entirely — it runs on
1.26 defaults. The cost is real but small and one-sided: a developer pinned to a 1.25
toolchain with `GOTOOLCHAIN=local` must upgrade. Every *shipped* binary is built by the
1.26 builder either way, so the directive was buying compatibility for local builds at
the price of weaker crypto defaults in production. `template-go-web-service` and
`template-static-site` had already chosen `1.26`; this also makes three of three Go
templates agree.

**Not upgraded, deliberately:** Go 1.27 (only `go1.27rc2` exists as of this audit —
no stable release; `golang:1.27-alpine` is not published). Revisit at the next review.

**Nothing was reverted** — no change made during this audit broke vet, test or build.

## Findings
1. **(medium) `go.mod` declared an end-of-life Go version.** `go 1.22` left the
   supported window when Go 1.24 shipped; it receives no security patches. Because
   the directive is only a floor and the Docker build pins 1.26, released images
   were never actually built with 1.22 — but any developer or agent building
   locally with a 1.22 toolchain installed would have satisfied the directive and
   produced a binary against an unpatched stdlib. **Action taken:** raised to
   `go 1.25`, then to `go 1.26` in the reconciliation pass once the directive's
   effect on the GODEBUG hardening baseline was measured (rationale and evidence
   above, and finding 9). Re-verified under 1.26; the `1.25`-floor claim no longer
   holds and `GOTOOLCHAIN=local` on 1.25 will now refuse to build.

2. **(medium) HTTP server had only one of five relevant timeouts set.** Only
   `ReadHeaderTimeout` was configured. `ReadTimeout`, `WriteTimeout` and
   `IdleTimeout` were all zero-valued, meaning a connection that completed its
   headers could then stall mid-body, stall mid-response, or idle on keep-alive
   indefinitely — each holding a goroutine and socket open. `MaxHeaderBytes` was
   left at the 1 MB default. **Action taken:** set `ReadTimeout` 15s,
   `WriteTimeout` 15s, `IdleTimeout` 60s and `MaxHeaderBytes` 64 KiB, with a
   comment telling adopters to raise rather than delete them for streaming or
   long-polling workloads. Added `TestNewServerHasTimeouts` so a future edit that
   drops them fails the test suite. Verified empirically: a half-open request is
   now dropped after 5.0s.

3. **(medium) Dockerfile would break as soon as a dependency was added.** The build
   stage did `COPY go.mod ./` only. With no `go.sum` in the build context, the
   first customer to add any third-party module would hit a build failure inside
   the image — the exact moment a template should not be fighting them. **Action
   taken:** changed to `COPY go.mod go.sum* ./`; the glob is a no-op today (no
   `go.sum` exists) and picks the file up automatically once there is one.
   Verified the glob does not error when the file is absent.

4. **(low) `.dockerignore` did not exclude secret-bearing files.** It covered VCS
   and docs but not `.env`, `*.pem`, `*.key`, `credentials.json` or
   `service-account*.json`. Since `COPY . .` pulls the whole context into the build
   stage, a customer with a local `.env` would silently bake it into an image
   layer, where it stays recoverable even if a later step deletes it. **Action
   taken:** added secret patterns, locally built binaries, and `.onklave/`.
   Also changed `*_test.go` to `**/*_test.go` — the unanchored form only matched
   the repository root, so tests in subpackages were being copied in.

5. **(low) `.gitignore` missed the binary `go build ./...` actually produces.** It
   ignored `/app` (the in-image path) but not `/template-container`, which is what
   a root-level `go build ./...` emits, since Go names the output after the module.
   Confirmed during this audit: the command left an untracked 8 MB binary in the
   working tree. **Action taken:** added `/template-container`.

6. **(low, accepted — not fixed) Base images are floating tags, not digest-pinned.**
   `golang:1.26-alpine` and `gcr.io/distroless/static:nonroot` are both mutable
   tags, so builds are not byte-reproducible and a compromised or regressed
   upstream tag would flow straight into generated apps. Digest pinning is the
   stricter posture, but this repo has no CI and no image-update automation (the
   GitHub Actions workflow was deliberately removed — the platform builds the repo
   itself). A pinned digest with nothing to bump it would rot, and a *stale* pinned
   base is a worse outcome than a floating one that picks up upstream patches.
   **Action:** left floating by design, recorded here, and raised under Open items
   for a human to revisit once image-update automation exists.

7. **(none) No secrets committed.** Full-history scan across all reachable commits
   for API keys, tokens, passwords and private key headers returned no matches.
   The only file ever removed from history was the inert CI workflow.

8. **(none) Container security posture is sound.** Final image is
   `distroless/static:nonroot` — no shell, no package manager, no libc beyond the
   static binary. Runs as uid 65532 via an explicit `USER nonroot:nonroot`. The
   binary is `CGO_ENABLED=0` static and `-trimpath` stripped (no host paths
   leaked). `HEALTHCHECK` correctly self-probes via the binary's own `-healthcheck`
   flag rather than assuming a shell or curl exists. Confirmed compatible with
   `readOnlyRootFilesystem`, `cap-drop=ALL` and `no-new-privileges` — the service
   writes nothing to disk. SIGTERM triggers a clean 10s-budget graceful shutdown,
   exit code 0.

9. **(medium) `go` directive trailed the builder, weakening the binary's GODEBUG
   defaults — FIXED in the cross-template reconciliation pass.** Seven templates were
   audited independently on 2026-08-04 and reached different conclusions about the
   same question; this template landed on `go 1.25` while `template-go-web-service`
   and `template-static-site` landed on `1.26`. Shipping the same starter code with
   different security postures is itself the defect. On the merits the `1.26` side is
   right: the directive is not only a language floor, it fixes the GODEBUG
   compatibility baseline compiled into the binary, so trailing the builder bakes in
   opt-outs from newer hardening defaults (`cryptocustomrand=1`, `tlssecpmlkem=0`,
   `urlstrictcolons=0` — table above). **Action taken:** raised to `go 1.26`, verified
   by diffing `go version -m` before and after (the `DefaultGODEBUG` line disappears),
   with `go vet`/`go test`/`go build` green after the change. Accepted cost: a
   developer on a 1.25 toolchain with `GOTOOLCHAIN=local` must now upgrade.

## Changes made in this audit
- `go.mod`: raised the `go` directive from `1.22` to `1.25`, and then to `1.26` in the
  reconciliation pass (finding 9) — ending the go.mod ↔ Dockerfile drift, leaving the
  end-of-life 1.22 floor behind, and dropping the `DefaultGODEBUG` hardening opt-outs
  that a trailing directive compiles into the binary. Added a comment naming the
  Dockerfile builder as the source of truth, matching `template-static-site`.
- `main.go`: added `ReadTimeout`, `WriteTimeout`, `IdleTimeout` and
  `MaxHeaderBytes` to the `http.Server`, with a comment on how to adapt them for
  streaming workloads.
- `main_test.go`: added `TestNewServerHasTimeouts` to lock the hardening in.
- `Dockerfile`: `COPY go.mod ./` → `COPY go.mod go.sum* ./` so the build survives
  the first added dependency.
- `.dockerignore`: excluded secret-bearing files (`.env`, `*.pem`, `*.key`,
  `*.p12`, `*.pfx`, `credentials.json`, `service-account*.json`), locally built
  binaries and `.onklave/`; corrected `*_test.go` to `**/*_test.go`.
- `.gitignore`: added `/template-container`.
- `README.md`: corrected the stale `golang:1.22-alpine` claim to `golang:1.26-alpine`,
  and described the `go.mod` row as declaring `go 1.26` to match the builder — noting
  that the directive sets the binary's GODEBUG hardening baseline — plus the absence
  of third-party dependencies.
- Added this file.

`onklave.yaml` was not modified.

## Open items
1. **Decide on base-image digest pinning** (finding 6). Pinning
   `golang:1.26-alpine` and `gcr.io/distroless/static:nonroot` by digest gives
   reproducible builds and protects against upstream tag compromise, but only pays
   off with something to bump the digests. Worth doing as one change alongside
   whatever image-update automation the platform adopts for templates — pinning
   without it just guarantees staleness.
2. **Go 1.27 is at rc2.** Once it goes stable and `golang:1.27-alpine` is published,
   bump the Dockerfile builder **and the `go` directive together** — the directive now
   tracks the builder rather than trailing it (finding 9), and the same move is due in
   `template-go-web-service` and `template-static-site` so the three stay aligned.
3. **No automated recheck exists for this template.** This audit was manual and the
   CI workflow was intentionally removed, so nothing will notice the next stdlib
   CVE or EOL toolchain. Consider a scheduled platform-side job that runs
   `govulncheck` across all templates and opens a ticket on findings — otherwise
   the "next review due" date above is the only control.
4. **Confirm the timeout values suit real generated apps.** 15s read/write is right
   for a JSON API and wrong for file uploads or SSE. If common Onklave workloads
   skew long-lived, revisit the defaults in `newServer` so customers are not forced
   to edit them immediately. Note the reconciliation pass deliberately kept
   `WriteTimeout` **here and in `template-go-web-service`** while leaving it off the
   two static-file servers (`template-static-site`, `template-3d-web-app`): a JSON API
   can bound its response writes, a file server serving large assets to a slow client
   cannot without truncating legitimate downloads. That split is intentional, not
   drift — do not "harmonise" it away.
