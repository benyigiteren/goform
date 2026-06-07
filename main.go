package main

import (
	"embed"
	"flag"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

//go:embed all:web
var webFS embed.FS

const Version = "1.1.0"

func main() {
	addr := flag.String("addr", envOr("GOFORM_ADDR", ":3000"), "listen address")
	dataDir := flag.String("data", envOr("GOFORM_DATA", "./data"), "data directory")
	publicURL := flag.String("public-url", envOr("GOFORM_PUBLIC_URL", ""), "public URL (used in API docs / share links when set)")
	flag.Parse()

	if err := os.MkdirAll(*dataDir, 0o755); err != nil {
		log.Fatalf("create data dir: %v", err)
	}

	dbPath := filepath.Join(*dataDir, "goform.db")
	store, err := OpenStore(dbPath)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer store.Close()

	staticFS, err := fs.Sub(webFS, "web/static")
	if err != nil {
		log.Fatalf("static fs: %v", err)
	}
	pageFS, err := fs.Sub(webFS, "web/pages")
	if err != nil {
		log.Fatalf("page fs: %v", err)
	}

	srv := &Server{Store: store, Pages: pageFS, PublicURL: strings.TrimRight(*publicURL, "/")}

	mux := http.NewServeMux()
	srv.Routes(mux, staticFS)

	httpSrv := &http.Server{
		Addr:              *addr,
		Handler:           withSecurityHeaders(mux),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		log.Printf("goform v%s listening on %s", Version, *addr)
		if srv.PublicURL != "" {
			log.Printf("public url: %s", srv.PublicURL)
		}
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server: %v", err)
		}
	}()

	// Periodic session cleanup
	stop := make(chan struct{})
	go func() {
		t := time.NewTicker(1 * time.Hour)
		defer t.Stop()
		for {
			select {
			case <-stop:
				return
			case <-t.C:
				store.PruneSessions()
			}
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	<-sig
	log.Println("shutting down...")
	close(stop)
	httpSrv.Close()
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func withSecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		h.Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
		h.Set("Content-Security-Policy",
			"default-src 'self'; "+
				"script-src 'self' 'unsafe-inline'; "+
				"style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; "+
				"font-src 'self' https://fonts.gstatic.com data:; "+
				"img-src 'self' data:; "+
				"connect-src 'self'; "+
				"frame-ancestors 'none'")
		next.ServeHTTP(w, r)
	})
}
