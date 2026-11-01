// SPDX-License-Identifier: Apache-2.0
//
// Placeholder tailbondd for package tests only (never shipped): a static
// linux/arm64 binary that accepts the real agent's flags, serves / and
// /api/status on :8077 and exits cleanly on SIGTERM, so the init script,
// procd and the GL/LuCI links can be exercised without the real agent.
//
//	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o out/test/tailbondd main.go
package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	cfgPath := flag.String("config", "/etc/tailbond/tailbond.json", "")
	listen := flag.String("listen", ":8077", "")
	flag.Bool("quiet", false, "")
	flag.Parse()

	role := "unknown"
	if b, err := os.ReadFile(*cfgPath); err == nil {
		var c struct {
			Role string `json:"role"`
		}
		if json.Unmarshal(b, &c) == nil && c.Role != "" {
			role = c.Role
		}
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"role": role, "mode": "speed", "state": "disconnected", "placeholder": true,
			"paths": []any{}, "clients": []any{},
		})
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte("<!doctype html><title>tailbond</title><h1>tailbond placeholder</h1>\n"))
	})
	srv := &http.Server{Addr: *listen, Handler: mux}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		sctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		srv.Shutdown(sctx)
	}()
	log.Printf("tailbondd placeholder: role=%s listening on %s", role, *listen)
	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatal(err)
	}
	log.Print("tailbondd placeholder: stopped")
}
