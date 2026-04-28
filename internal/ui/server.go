package ui

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

type Server struct {
	collector *Collector
	addr      string
	mux       *http.ServeMux
	authToken string
}

func NewServer(addr string, collector *Collector) *Server {
	server := &Server{
		addr:      addr,
		collector: collector,
		mux:       http.NewServeMux(),
	}
	if collector != nil && collector.cfg != nil {
		server.authToken = collector.cfg.UI.AuthToken
	}
	server.routes()
	return server
}

func (s *Server) ListenAndServe(ctx context.Context) error {
	httpServer := &http.Server{
		Addr:              s.addr,
		Handler:           s.mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			log.Printf("UI server shutdown error: %v", err)
		}
	}()

	err := httpServer.ListenAndServe()
	if err == http.ErrServerClosed {
		return nil
	}
	return err
}

func (s *Server) routes() {
	s.mux.HandleFunc("/", s.handleIndex)
	s.mux.HandleFunc("/api/snapshot", s.withAuth(s.requireMethod(http.MethodGet, s.handleSnapshot)))
	s.mux.HandleFunc("/api/tools", s.withAuth(s.requireMethod(http.MethodGet, s.handleTools)))
	s.mux.HandleFunc("/api/logs", s.withAuth(s.requireMethod(http.MethodGet, s.handleLogs)))
	s.mux.HandleFunc("/api/history", s.withAuth(s.requireMethod(http.MethodGet, s.handleHistory)))
	s.mux.HandleFunc("/api/runs", s.withAuth(s.requireMethod(http.MethodGet, s.handleRuns)))
	s.mux.HandleFunc("/api/runs/", s.withAuth(s.requireMethod(http.MethodGet, s.handleRun)))
	s.mux.HandleFunc("/api/approvals", s.withAuth(s.requireMethod(http.MethodGet, s.handleApprovals)))
	s.mux.HandleFunc("/api/approvals/", s.withAuth(s.requireMethod(http.MethodPost, s.handleApprovalDecision)))
	s.mux.HandleFunc("/api/documents", s.withAuth(s.requireMethod(http.MethodGet, s.handleDocuments)))
	s.mux.HandleFunc("/api/check", s.withAuth(s.requireMethod(http.MethodPost, s.handleCheck)))
	s.mux.HandleFunc("/api/stream", s.withAuth(s.requireMethod(http.MethodGet, s.handleStream)))
	s.mux.HandleFunc("/a2ui/render", s.withAuth(s.requireMethod(http.MethodGet, s.handleA2UIRender)))
}

func (s *Server) requireMethod(method string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != method {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		next(w, r)
	}
}

func (s *Server) withAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if s.authToken == "" {
			next(w, r)
			return
		}
		authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
		if !strings.HasPrefix(authHeader, "Bearer ") {
			w.Header().Set("WWW-Authenticate", `Bearer realm="sysguard"`)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		got := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
		if subtle.ConstantTimeCompare([]byte(got), []byte(s.authToken)) != 1 {
			w.Header().Set("WWW-Authenticate", `Bearer realm="sysguard"`)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(indexHTML))
}

func (s *Server) handleSnapshot(w http.ResponseWriter, r *http.Request) {
	snapshot, err := s.collector.Snapshot(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, snapshot)
}

func (s *Server) handleTools(w http.ResponseWriter, r *http.Request) {
	snapshot, err := s.collector.Snapshot(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, snapshot.Tools)
}

func (s *Server) handleLogs(w http.ResponseWriter, r *http.Request) {
	snapshot, err := s.collector.Snapshot(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, snapshot.Logs)
}

func (s *Server) handleHistory(w http.ResponseWriter, r *http.Request) {
	snapshot, err := s.collector.Snapshot(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, snapshot.History)
}

func (s *Server) handleRuns(w http.ResponseWriter, r *http.Request) {
	runs, err := s.collector.Runs(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, runs)
}

func (s *Server) handleRun(w http.ResponseWriter, r *http.Request) {
	runID := strings.TrimPrefix(r.URL.Path, "/api/runs/")
	if strings.TrimSpace(runID) == "" || strings.Contains(runID, "/") {
		http.NotFound(w, r)
		return
	}
	run, ok, err := s.collector.Run(r.Context(), runID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !ok {
		http.NotFound(w, r)
		return
	}
	writeJSON(w, run)
}

func (s *Server) handleApprovals(w http.ResponseWriter, r *http.Request) {
	approvals, err := s.collector.Approvals(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, approvals)
}

func (s *Server) handleApprovalDecision(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/approvals/")
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 2 || parts[0] == "" {
		http.NotFound(w, r)
		return
	}
	var approved bool
	switch parts[1] {
	case "approve":
		approved = true
	case "deny":
		approved = false
	default:
		http.NotFound(w, r)
		return
	}
	req, err := s.collector.DecideApproval(r.Context(), parts[0], approved, "ui")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, req)
}

func (s *Server) handleDocuments(w http.ResponseWriter, r *http.Request) {
	snapshot, err := s.collector.Snapshot(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, snapshot.Documents)
}

func (s *Server) handleCheck(w http.ResponseWriter, r *http.Request) {
	snapshot, err := s.collector.TriggerCheck(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, snapshot)
}

func (s *Server) handleA2UIRender(w http.ResponseWriter, r *http.Request) {
	snapshot, err := s.collector.Snapshot(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, NewA2UIDashboardMessage(snapshot))
}

func (s *Server) handleStream(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		if err := s.writeA2UIEvent(w); err != nil {
			return
		}
		flusher.Flush()

		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
		}
	}
}

func (s *Server) writeA2UIEvent(w http.ResponseWriter) error {
	snapshot, err := s.collector.Snapshot(context.Background())
	if err != nil {
		return err
	}
	data, err := json.Marshal(NewA2UIDashboardMessage(snapshot))
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "event: a2ui\ndata: %s\n\n", data)
	return err
}

func writeJSON(w http.ResponseWriter, value interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	_ = encoder.Encode(value)
}
