package ui

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

type Server struct {
	collector *Collector
	addr      string
	mux       *http.ServeMux
}

func NewServer(addr string, collector *Collector) *Server {
	server := &Server{
		addr:      addr,
		collector: collector,
		mux:       http.NewServeMux(),
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
	s.mux.HandleFunc("/api/snapshot", s.handleSnapshot)
	s.mux.HandleFunc("/api/tools", s.handleTools)
	s.mux.HandleFunc("/api/logs", s.handleLogs)
	s.mux.HandleFunc("/api/history", s.handleHistory)
	s.mux.HandleFunc("/api/documents", s.handleDocuments)
	s.mux.HandleFunc("/api/check", s.handleCheck)
	s.mux.HandleFunc("/api/stream", s.handleStream)
	s.mux.HandleFunc("/a2ui/render", s.handleA2UIRender)
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

func (s *Server) handleDocuments(w http.ResponseWriter, r *http.Request) {
	snapshot, err := s.collector.Snapshot(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, snapshot.Documents)
}

func (s *Server) handleCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	s.handleSnapshot(w, r)
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
