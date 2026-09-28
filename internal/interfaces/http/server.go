// Package http implements the HTTP statistics listener of the daemon:
// /healthz, /api/v1/stats (JSON) and /metrics (Prometheus text format).
package http

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	nethttp "net/http"
	"time"

	"github.com/burn-lab-dev/cantcp/internal/app/domain"
)

// Endpoint paths.
const (
	pathHealth  = "/healthz"
	pathStats   = "/api/v1/stats"
	pathMetrics = "/metrics"
)

// iStats is the statistics source of the endpoints.
type iStats interface {
	Snapshot(now time.Time) domain.Stats
}

// Server serves the statistics endpoints on a dedicated listener.
type Server struct {
	cfg      domain.ConfigStats
	stats    iStats
	log      *slog.Logger
	listener net.Listener
	http     *nethttp.Server
}

// NewServer builds the server.
func NewServer(cfg domain.ConfigStats, stats iStats, log *slog.Logger) *Server {
	return &Server{cfg: cfg, stats: stats, log: log}
}

// Listen opens the TCP listener and prepares the routes.
func (s *Server) Listen() error {
	ln, err := net.Listen("tcp", s.cfg.Listen)
	if err != nil {
		return fmt.Errorf("stats http: listen %s: %w", s.cfg.Listen, err)
	}
	s.listener = ln
	mux := nethttp.NewServeMux()
	mux.HandleFunc("GET "+pathHealth, s.handleHealth)
	mux.HandleFunc("GET "+pathStats, s.handleStats)
	mux.HandleFunc("GET "+pathMetrics, s.handleMetrics)
	s.http = &nethttp.Server{
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		ErrorLog:          slog.NewLogLogger(s.log.Handler(), slog.LevelWarn),
	}
	return nil
}

// Addr returns the actual listener address, nil before Listen.
func (s *Server) Addr() net.Addr {
	if s.listener == nil {
		return nil
	}
	return s.listener.Addr()
}

// Run serves until ctx is canceled. It must be called after Listen.
func (s *Server) Run(ctx context.Context) error {
	if s.http == nil {
		return errors.New("stats http: Run called before Listen")
	}
	errCh := make(chan error, 1)
	go func() { errCh <- s.http.Serve(s.listener) }()
	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = s.http.Shutdown(shutdownCtx)
		<-errCh
		return nil
	case err := <-errCh:
		if errors.Is(err, nethttp.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("stats http: %w", err)
	}
}

// handleHealth reports that the process is alive.
func (s *Server) handleHealth(w nethttp.ResponseWriter, _ *nethttp.Request) {
	writeJSON(w, nethttp.StatusOK, map[string]string{"status": "ok"})
}

// handleStats serves the JSON snapshot.
func (s *Server) handleStats(w nethttp.ResponseWriter, _ *nethttp.Request) {
	writeJSON(w, nethttp.StatusOK, s.stats.Snapshot(time.Now()))
}

// writeJSON writes a JSON response.
func writeJSON(w nethttp.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
