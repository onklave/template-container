# template-container

A generic **"bring your own Dockerfile"** container starter for [Onklave](https://onklave.app). This is an **Onklave project template** — use it as the starting point for a containerized service in any language or stack.

The example here is a tiny self-contained Go HTTP service, chosen only because it compiles to a small static binary. **The language is not the point** — the point is the build/run contract below.

## The contract

Onklave builds and runs your container from the repository itself. Whatever stack you bring, keep these three things true:

1. **A `Dockerfile` at the repository root.** Onklave builds your image from it — there is no separate build configuration.
2. **The service listens on port `8080`.** Honour the `PORT` environment variable if set, but default to `8080`. Onklave serves your app on port 8080.
3. **`GET /healthz` returns `200`.** Used for liveness/readiness probes.

As long as those hold, swap in any language, framework, or base image you like.

## What's in here

| File            | Purpose                                                                 |
| --------------- | ----------------------------------------------------------------------- |
| `main.go`       | stdlib `net/http` server: `GET /` greeting, `GET /healthz` → `200 ok`, graceful shutdown on SIGTERM/SIGINT. |
| `main_test.go`  | Handler + healthcheck tests.                                            |
| `Dockerfile`    | Multi-stage build: static binary in `golang:1.22-alpine`, final `gcr.io/distroless/static:nonroot` image running as a non-root user, with `EXPOSE 8080` and a `HEALTHCHECK`. |
| `go.mod`        | Go module definition.                                                   |
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
