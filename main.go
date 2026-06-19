// Package main implements a tiny self-contained HTTP service that demonstrates
// the Onklave "bring your own Dockerfile" container contract.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// runHealthCheck probes the local /healthz endpoint and exits 0 on success,
// 1 on failure. It is used by the Docker HEALTHCHECK directive because the
// distroless final image ships no shell or curl/wget.
func runHealthCheck(port string) int {
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(fmt.Sprintf("http://127.0.0.1:%s/healthz", port))
	if err != nil {
		return 1
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 1
	}
	return 0
}

// newServer wires up the HTTP routes and returns a configured *http.Server.
func newServer(addr string) *http.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", healthzHandler)
	mux.HandleFunc("/", rootHandler)

	return &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
}

// healthzHandler responds 200 "ok" for liveness/readiness probes.
func healthzHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

// rootHandler returns a simple greeting.
func rootHandler(w http.ResponseWriter, r *http.Request) {
	// Only the exact root path gets the greeting; everything else is 404.
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("Hello from the Onklave container template!\n"))
}

func main() {
	healthcheck := flag.Bool("healthcheck", false, "probe /healthz and exit (used by Docker HEALTHCHECK)")
	flag.Parse()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// In healthcheck mode, probe the running server and exit immediately.
	if *healthcheck {
		os.Exit(runHealthCheck(port))
	}

	addr := ":" + port

	srv := newServer(addr)

	// Run the server in a goroutine so main can wait for shutdown signals.
	go func() {
		log.Printf("listening on %s", addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server error: %v", err)
		}
	}()

	// Wait for SIGTERM/SIGINT, then shut down gracefully.
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT)
	<-stop

	log.Println("shutting down")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("graceful shutdown failed: %v", err)
	}
	log.Println("stopped")
}
