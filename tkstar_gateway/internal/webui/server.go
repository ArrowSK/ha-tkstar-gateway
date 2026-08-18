package webui

import (
	"embed"
	"encoding/json"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/ArrowSK/ha-tkstar-gateway/internal/gateway"
	"github.com/ArrowSK/ha-tkstar-gateway/internal/store"
)

//go:embed assets/*
var assets embed.FS

type Server struct {
	gw      *gateway.Gateway
	log     *slog.Logger
	started time.Time
}

func New(gw *gateway.Gateway, log *slog.Logger) *Server {
	return &Server{gw: gw, log: log, started: time.Now()}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	sub, _ := fs.Sub(assets, "assets")
	mux.Handle("GET /", http.FileServer(http.FS(sub)))
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("GET /api/status", s.status)
	mux.HandleFunc("GET /api/devices", s.devices)
	mux.HandleFunc("POST /api/pairing/start", s.pairStart)
	mux.HandleFunc("POST /api/pairing/stop", s.pairStop)
	mux.HandleFunc("GET /api/devices/{key}", s.device)
	mux.HandleFunc("PATCH /api/devices/{key}", s.update)
	mux.HandleFunc("GET /api/devices/{key}/positions", s.positions)
	mux.HandleFunc("GET /api/devices/{key}/stats", s.stats)
	mux.HandleFunc("GET /api/devices/{key}/alarms", s.alarms)
	return secure(mux)
}

func secure(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "SAMEORIGIN")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r)
	})
}

func out(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func (s *Server) status(w http.ResponseWriter, _ *http.Request) {
	out(w, map[string]any{
		"version":               "0.1.0",
		"uptime_seconds":        int(time.Since(s.started).Seconds()),
		"pairing_until":         s.gw.PairingUntil(),
		"learned_cells":         s.gw.LearnedCells(),
		"offline_after_seconds": int(s.gw.OfflineAfter().Seconds()),
	})
}

func (s *Server) devices(w http.ResponseWriter, _ *http.Request) { out(w, s.gw.Devices()) }

func (s *Server) pairStart(w http.ResponseWriter, _ *http.Request) {
	out(w, map[string]any{"pairing_until": s.gw.StartPairing(10 * time.Minute)})
}

func (s *Server) pairStop(w http.ResponseWriter, _ *http.Request) {
	s.gw.StopPairing()
	out(w, map[string]any{"pairing_until": nil})
}

func (s *Server) device(w http.ResponseWriter, r *http.Request) {
	d, ok := s.gw.DeviceByKey(r.PathValue("key"))
	if !ok {
		http.NotFound(w, r)
		return
	}
	out(w, d)
}

func (s *Server) update(w http.ResponseWriter, r *http.Request) {
	var p map[string]any
	if json.NewDecoder(http.MaxBytesReader(w, r.Body, 32<<10)).Decode(&p) != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	d, err := s.gw.Update(r.PathValue("key"), p)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	out(w, d)
}

func period(r *http.Request) (time.Time, time.Time) {
	hours := 24.0
	if v, err := strconv.ParseFloat(r.URL.Query().Get("hours"), 64); err == nil && v > 0 && v <= 24*365 {
		hours = v
	}
	to := time.Now().UTC()
	return to.Add(-time.Duration(hours * float64(time.Hour))), to
}

func (s *Server) positions(w http.ResponseWriter, r *http.Request) {
	d, ok := s.gw.DeviceByKey(r.PathValue("key"))
	if !ok {
		http.NotFound(w, r)
		return
	}
	from, to := period(r)
	p, err := s.gw.Store().Positions(d.Key, from, to, 5000)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	out(w, p)
}

func (s *Server) alarms(w http.ResponseWriter, r *http.Request) {
	d, ok := s.gw.DeviceByKey(r.PathValue("key"))
	if !ok {
		http.NotFound(w, r)
		return
	}
	from, to := period(r)
	p, err := s.gw.Store().Alarms(d.Key, from, to, 500)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	out(w, p)
}

func (s *Server) stats(w http.ResponseWriter, r *http.Request) {
	d, ok := s.gw.DeviceByKey(r.PathValue("key"))
	if !ok {
		http.NotFound(w, r)
		return
	}
	from, to := period(r)
	p, err := s.gw.Store().Positions(d.Key, from, to, 20000)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	out(w, store.ComputeStats(p))
}

func Listen(addr string, h http.Handler, log *slog.Logger) error {
	srv := &http.Server{
		Addr:              addr,
		Handler:           h,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	log.Info("web interface started", "addr", addr)
	return srv.Serve(ln)
}
