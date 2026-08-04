# template-container

A generic **"bring your own Dockerfile"** container starter for [Onklave](https://onklave.app). This is an **Onklave project template** — use it as the starting point for a containerized service in any language or stack.

The example here is a tiny self-contained Go HTTP service, chosen only because it compiles to a small static binary. **The language is not the point** — the point is the build/run contract below.

## The contract

Onklave builds and runs your container from the repository itself. Whatever stack you bring, keep these three things true:

1. **An `onklave.yaml` at the repository root.** This is the only file the platform reads to learn what to build, which port to serve, which health path to probe and which route to expose. GitHub Actions is not used — the platform builds, tests and deploys the repo itself. To add a second service, add another entry under `services`.
2. **A `Dockerfile` at the repository root.** Onklave builds your image from it, using the `build.context` and `build.dockerfile` declared in `onklave.yaml`.
3. **The service listens on port `8080`.** Honour the `PORT` environment variable if set, but default to `8080` — the value declared as `runtime.port`.
4. **`GET /healthz` returns `200`.** Declared as `runtime.healthPath`; used for liveness/readiness probes.

As long as those hold, swap in any language, framework, or base image you like.

## What's in here

| File            | Purpose                                                                 |
| --------------- | ----------------------------------------------------------------------- |
| `onklave.yaml`  | Build/run manifest the platform reads: build context, port, health path, exposed route. |
| `main.go`       | stdlib `net/http` server: `GET /` greeting, `GET /healthz` → `200 ok`, graceful shutdown on SIGTERM/SIGINT. |
| `main_test.go`  | Handler + healthcheck tests.                                            |
| `Dockerfile`    | Multi-stage build: static binary in `golang:1.26-alpine`, final `gcr.io/distroless/static:nonroot` image running as a non-root user, with `EXPOSE 8080` and a `HEALTHCHECK`. |
| `go.mod`        | Go module definition. Declares `go 1.26`, matching the `golang:1.26-alpine` builder — the directive sets the GODEBUG hardening baseline compiled into the binary, so it tracks the builder rather than trailing it. No third-party dependencies. |
| `.dockerignore` | Keeps the build context small.                                          |
| `.gitignore`    | Standard Go ignores.                                                    |

## Run locally

```bash
# With Go installed
go test ./...
go run .          # serves on http://localhost:8080
curl localhost:8080/healthz   # -> ok

# Or via Docker (the path Onklave uses)
docker build -t template-container .
docker run --rm -p 8080:8080 template-container
curl localhost:8080/healthz   # -> ok
```

## Make it yours

1. Replace `main.go` (and `go.mod`/`main_test.go`) with your own application.
2. Update the `Dockerfile` for your stack — keep it multi-stage, run as non-root, and `EXPOSE 8080`.
3. Keep `GET /healthz` returning `200` and the service listening on `:8080`.

That's the whole contract. Onklave handles the rest.
