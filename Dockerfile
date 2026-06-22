# syntax=docker/dockerfile:1

# ---- Build stage ------------------------------------------------------------
FROM golang:1.26-alpine AS build

WORKDIR /src

# Cache module downloads when only sources change.
COPY go.mod ./
RUN go mod download

COPY . .

# Build a fully static binary so it runs in a distroless/scratch image.
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/app .

# ---- Final stage ------------------------------------------------------------
# distroless/static:nonroot ships no shell and runs as an unprivileged user.
FROM gcr.io/distroless/static:nonroot

WORKDIR /

COPY --from=build /out/app /app

# Run as the built-in non-root user (uid 65532).
USER nonroot:nonroot

EXPOSE 8080

# distroless has no shell; this binary self-checks its own /healthz endpoint.
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD ["/app", "-healthcheck"]

ENTRYPOINT ["/app"]
