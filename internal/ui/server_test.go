package ui

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sysguard/sysguard/internal/config"
)

func TestServerExposesDashboardResourceEndpoints(t *testing.T) {
	t.Parallel()

	collector := NewCollector(config.Default(), nil, nil, nil)
	server := NewServer(":0", collector)

	cases := []struct {
		path string
		key  string
	}{
		{path: "/api/tools", key: "recent"},
		{path: "/api/logs", key: "recent"},
		{path: "/api/history", key: "recent"},
		{path: "/api/documents", key: "items"},
	}

	for _, tc := range cases {
		t.Run(tc.path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			res := httptest.NewRecorder()
			server.mux.ServeHTTP(res, req)
			if res.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200 body=%s", res.Code, res.Body.String())
			}
			var body map[string]interface{}
			if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
				t.Fatalf("response is not JSON: %v", err)
			}
			if _, ok := body[tc.key]; !ok {
				t.Fatalf("response missing key %q: %#v", tc.key, body)
			}
		})
	}
}

func TestServerRequiresBearerTokenForAPIWhenConfigured(t *testing.T) {
	t.Parallel()

	cfg := config.Default()
	cfg.UI.AuthToken = "secret-token"
	collector := NewCollector(cfg, nil, nil, nil)
	server := NewServer(":0", collector)

	req := httptest.NewRequest(http.MethodGet, "/api/snapshot", nil)
	res := httptest.NewRecorder()
	server.mux.ServeHTTP(res, req)
	if res.Code != http.StatusUnauthorized {
		t.Fatalf("status without token = %d, want 401", res.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/snapshot", nil)
	req.Header.Set("Authorization", "secret-token")
	res = httptest.NewRecorder()
	server.mux.ServeHTTP(res, req)
	if res.Code != http.StatusUnauthorized {
		t.Fatalf("status with bare token = %d, want 401", res.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/snapshot", nil)
	req.Header.Set("Authorization", "Bearer secret-token")
	res = httptest.NewRecorder()
	server.mux.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status with bearer token = %d, want 200 body=%s", res.Code, res.Body.String())
	}
}

func TestServerRejectsPostOnReadOnlyEndpoints(t *testing.T) {
	t.Parallel()

	collector := NewCollector(config.Default(), nil, nil, nil)
	server := NewServer(":0", collector)

	req := httptest.NewRequest(http.MethodPost, "/api/logs", nil)
	res := httptest.NewRecorder()
	server.mux.ServeHTTP(res, req)
	if res.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", res.Code)
	}
}
