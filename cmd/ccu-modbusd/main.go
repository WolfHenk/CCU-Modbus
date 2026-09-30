package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/WolfHenk/ccu-modbus/internal/config"
	"github.com/WolfHenk/ccu-modbus/internal/engine"
)

var version = "0.1.0-dev"

func main() {
	configPath := flag.String("config", "/usr/local/etc/config/addons/ccu-modbus/config.json", "Konfigurationsdatei")
	check := flag.Bool("check", false, "Konfiguration pruefen und beenden")
	flag.Parse()

	log.SetPrefix("ccu-modbus: ")
	log.SetFlags(log.Ldate | log.Ltime)

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("Konfiguration kann nicht geladen werden: %v", err)
	}
	if *check {
		fmt.Printf("Konfiguration OK: %d Geraet(e)\n", len(cfg.Devices))
		return
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	eng := engine.New()
	eng.Start(ctx, cfg)
	defer eng.Stop()

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"ok":true,"version":%q}`, version)
	})
	mux.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(eng.Snapshot())
	})

	server := &http.Server{
		Addr:              cfg.Listen,
		Handler:           mux,
		ReadHeaderTimeout: 2 * time.Second,
	}

	go func() {
		log.Printf("gestartet, Version %s, API auf %s", version, cfg.Listen)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("API-Fehler: %v", err)
			stop()
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_ = server.Shutdown(shutdownCtx)
	log.Printf("beendet")
	_ = os.Stdout.Sync()
}
