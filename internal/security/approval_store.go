package security

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

const defaultApprovalStoreLimit = 200

type ApprovalStatus string

const (
	ApprovalPending  ApprovalStatus = "pending"
	ApprovalApproved ApprovalStatus = "approved"
	ApprovalDenied   ApprovalStatus = "denied"
	ApprovalUsed     ApprovalStatus = "used"
)

type ApprovalRequest struct {
	ID         string            `json:"id"`
	Tool       string            `json:"tool"`
	Action     string            `json:"action"`
	Command    string            `json:"command"`
	Reason     string            `json:"reason,omitempty"`
	Risk       string            `json:"risk,omitempty"`
	Status     ApprovalStatus    `json:"status"`
	CreatedAt  time.Time         `json:"created_at"`
	DecidedAt  time.Time         `json:"decided_at,omitempty"`
	UsedAt     time.Time         `json:"used_at,omitempty"`
	ExpiresAt  time.Time         `json:"expires_at,omitempty"`
	DecisionBy string            `json:"decision_by,omitempty"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

type ApprovalStore struct {
	path  string
	limit int
	mu    sync.RWMutex
	items map[string]ApprovalRequest
}

func NewApprovalStore(path string) (*ApprovalStore, error) {
	store := &ApprovalStore{
		path:  path,
		limit: defaultApprovalStoreLimit,
		items: make(map[string]ApprovalRequest),
	}
	if err := store.load(); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *ApprovalStore) Create(ctx context.Context, req ApprovalRequest) (ApprovalRequest, error) {
	if s == nil || s.path == "" {
		return ApprovalRequest{}, fmt.Errorf("approval store is not configured")
	}
	now := time.Now().UTC()
	if req.ID == "" {
		req.ID = fmt.Sprintf("approval-%d", now.UnixNano())
	}
	if req.CreatedAt.IsZero() {
		req.CreatedAt = now
	}
	req.Status = ApprovalPending
	if req.Metadata == nil {
		req.Metadata = map[string]string{}
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[req.ID] = req
	s.pruneLocked()
	return req, s.saveLocked()
}

func (s *ApprovalStore) Decide(ctx context.Context, id string, approved bool, actor string) (ApprovalRequest, error) {
	if s == nil {
		return ApprovalRequest{}, fmt.Errorf("approval store is not configured")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	req, ok := s.items[id]
	if !ok {
		return ApprovalRequest{}, fmt.Errorf("approval %q not found", id)
	}
	if req.Status != ApprovalPending {
		return ApprovalRequest{}, fmt.Errorf("approval %q is %s", id, req.Status)
	}
	now := time.Now().UTC()
	req.DecidedAt = now
	req.DecisionBy = actor
	if approved {
		req.Status = ApprovalApproved
	} else {
		req.Status = ApprovalDenied
	}
	s.items[id] = req
	return req, s.saveLocked()
}

func (s *ApprovalStore) Consume(ctx context.Context, id, command string) (ApprovalRequest, error) {
	if s == nil {
		return ApprovalRequest{}, fmt.Errorf("approval store is not configured")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	req, ok := s.items[id]
	if !ok {
		return ApprovalRequest{}, fmt.Errorf("approval %q not found", id)
	}
	if req.Status != ApprovalApproved {
		return ApprovalRequest{}, fmt.Errorf("approval %q is %s", id, req.Status)
	}
	if req.Command != command {
		return ApprovalRequest{}, fmt.Errorf("approval %q command mismatch", id)
	}
	if !req.ExpiresAt.IsZero() && time.Now().UTC().After(req.ExpiresAt) {
		req.Status = ApprovalDenied
		s.items[id] = req
		_ = s.saveLocked()
		return ApprovalRequest{}, fmt.Errorf("approval %q expired", id)
	}
	req.Status = ApprovalUsed
	req.UsedAt = time.Now().UTC()
	s.items[id] = req
	return req, s.saveLocked()
}

func (s *ApprovalStore) List(ctx context.Context, limit int) ([]ApprovalRequest, error) {
	if s == nil {
		return nil, nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]ApprovalRequest, 0, len(s.items))
	for _, item := range s.items {
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].CreatedAt.After(items[j].CreatedAt)
	})
	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}
	return items, nil
}

func (s *ApprovalStore) load() error {
	if s.path == "" {
		return nil
	}
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if len(data) == 0 {
		return nil
	}
	var items []ApprovalRequest
	if err := json.Unmarshal(data, &items); err != nil {
		return err
	}
	for _, item := range items {
		if item.ID != "" {
			s.items[item.ID] = item
		}
	}
	return nil
}

func (s *ApprovalStore) pruneLocked() {
	items := make([]ApprovalRequest, 0, len(s.items))
	for _, item := range s.items {
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].CreatedAt.After(items[j].CreatedAt)
	})
	for i := s.limit; i < len(items); i++ {
		delete(s.items, items[i].ID)
	}
}

func (s *ApprovalStore) saveLocked() error {
	if s.path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	items := make([]ApprovalRequest, 0, len(s.items))
	for _, item := range s.items {
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].CreatedAt.After(items[j].CreatedAt)
	})
	data, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(s.path), ".approvals-*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if _, err := tmp.Write(append(data, '\n')); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, s.path); err != nil {
		return err
	}
	return os.Chmod(s.path, 0o600)
}
