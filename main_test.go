package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHealthzHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()

	healthzHandler(rec, req)

	res := rec.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, res.StatusCode)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("failed to read body: %v", err)
	}
	if got := string(body); got != "ok" {
		t.Fatalf("expected body %q, got %q", "ok", got)
	}
}

func TestRootHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	rootHandler(rec, req)

	res := rec.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, res.StatusCode)
	}
}

func TestRootHandlerNotFound(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/nope", nil)
	rec := httptest.NewRecorder()

	rootHandler(rec, req)

	res := rec.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, res.StatusCode)
	}
}

func TestRunHealthCheck(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(healthzHandler))
	defer srv.Close()

	// httptest URLs look like http://127.0.0.1:PORT — extract the port.
	port := srv.URL[strings.LastIndex(srv.URL, ":")+1:]

	if code := runHealthCheck(port); code != 0 {
		t.Fatalf("expected healthcheck exit code 0, got %d", code)
	}
}

func TestRunHealthCheckFailsWhenDown(t *testing.T) {
	// Nothing is listening on this port, so the probe must fail.
	if code := runHealthCheck("1"); code == 0 {
		t.Fatalf("expected non-zero exit code when server is unreachable")
	}
}
