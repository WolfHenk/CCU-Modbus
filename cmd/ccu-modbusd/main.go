package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"log/syslog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/WolfHenk/ccu-modbus/internal/ccuvirtual"
	"github.com/WolfHenk/ccu-modbus/internal/config"
	"github.com/WolfHenk/ccu-modbus/internal/engine"
)

var version = "0.1.0-dev"

func main() {
	configPath := flag.String("config", "/usr/local/etc/config/addons/ccu-modbus/config.json", "Konfigurationsdatei")
	check := flag.Bool("check", false, "Konfiguration pruefen und beenden")
	flag.Parse()

	if sw, err := syslog.New(syslog.LOG_DAEMON|syslog.LOG_INFO, "ccu-modbus"); err == nil {
		log.SetOutput(sw)
		defer sw.Close()
		log.SetFlags(0)
	} else {
		log.SetPrefix("ccu-modbus: ")
		log.SetFlags(log.Ldate | log.Ltime)
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("Konfiguration kann nicht geladen werden: %v", err)
	}
	if *check {
		if errs := config.Validate(cfg); len(errs) > 0 {
			log.Fatalf("Konfiguration ungueltig: %v", errs[0])
		}
		fmt.Printf("Konfiguration OK: %d Geraet(e)\n", len(cfg.Devices))
		return
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	eng := engine.New()
	eng.Start(ctx, cfg)
	defer eng.Stop()

	mux := http.NewServeMux()
	virt := ccuvirtual.Attach(ctx, mux, cfg, eng)
	defer virt.Close()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		jsonReply(w, http.StatusOK, map[string]any{"ok": true, "version": version})
	})
	mux.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			jsonReply(w, http.StatusMethodNotAllowed, map[string]any{"ok": false, "error": "Methode nicht erlaubt"})
			return
		}
		jsonReply(w, http.StatusOK, eng.Snapshot())
	})

	mux.HandleFunc("/config", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			current, err := config.Load(*configPath)
			if err != nil {
				jsonReply(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": err.Error()})
				return
			}
			jsonReply(w, http.StatusOK, current)
		case http.MethodPost:
			var next config.Config
			dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1024*1024))
			if err := dec.Decode(&next); err != nil {
				jsonReply(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "Ungueltige JSON-Konfiguration: " + err.Error()})
				return
			}
			config.ApplyDefaults(&next)
			if errs := config.Validate(&next); len(errs) > 0 {
				jsonReply(w, http.StatusBadRequest, map[string]any{"ok": false, "error": errs[0].Error()})
				return
			}
			if err := config.WriteAtomic(*configPath, &next); err != nil {
				jsonReply(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": err.Error()})
				return
			}
			// Synchronize the virtual CCU device model while ReGa is still
			// connected. The CGI restarts the daemon immediately afterwards.
			virt.ReplaceConfig(&next, eng)
			jsonReply(w, http.StatusOK, map[string]any{"ok": true, "restart_required": true})
		default:
			jsonReply(w, http.StatusMethodNotAllowed, map[string]any{"ok": false, "error": "Methode nicht erlaubt"})
		}
	})
	mux.HandleFunc("/unregister-all", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			jsonReply(w, http.StatusMethodNotAllowed, map[string]any{"ok": false, "error": "Methode nicht erlaubt"})
			return
		}
		virt.RemoveAll()
		jsonReply(w, http.StatusOK, map[string]any{"ok": true})
	})
	mux.HandleFunc("/test-connection", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			jsonReply(w, http.StatusMethodNotAllowed, map[string]any{"ok": false, "error": "Methode nicht erlaubt"})
			return
		}
		var d config.Device
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 256*1024)).Decode(&d); err != nil {
			jsonReply(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "Ungueltige Geraetedaten"})
			return
		}
		res := engine.TestConnection(d)
		code := http.StatusOK
		if !res.OK {
			code = http.StatusBadRequest
		}
		jsonReply(w, code, res)
	})
	mux.HandleFunc("/test-register", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			jsonReply(w, http.StatusMethodNotAllowed, map[string]any{"ok": false, "error": "Methode nicht erlaubt"})
			return
		}
		var req struct {
			Device   config.Device   `json:"device"`
			Register config.Register `json:"register"`
		}
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 256*1024)).Decode(&req); err != nil {
			jsonReply(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "Ungueltige Testdaten"})
			return
		}
		res := engine.TestRegister(req.Device, req.Register)
		code := http.StatusOK
		if !res.OK {
			code = http.StatusBadRequest
		}
		jsonReply(w, code, res)
	})

	server := &http.Server{
		Addr:              cfg.Listen,
		Handler:           mux,
		ReadHeaderTimeout: 2 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       30 * time.Second,
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

func jsonReply(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
