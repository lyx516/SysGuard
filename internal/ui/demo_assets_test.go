package ui

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestGitHubPagesDemoDataShowsRichSimulatedIncident(t *testing.T) {
	t.Parallel()

	root := filepath.Join("..", "..", "docs", "demo", "data")
	data, err := os.ReadFile(filepath.Join(root, "snapshot.json"))
	if err != nil {
		t.Fatalf("read demo snapshot: %v", err)
	}

	var snapshot struct {
		ProjectHighlights []struct {
			Title string `json:"title"`
		} `json:"project_highlights"`
		Incident struct {
			ID       string `json:"id"`
			Severity string `json:"severity"`
			Status   string `json:"status"`
			Story    []struct {
				Agent  string   `json:"agent"`
				Action string   `json:"action"`
				Steps  []string `json:"steps"`
			} `json:"story"`
		} `json:"incident"`
		Tools struct {
			Recent []struct {
				Name   string `json:"name"`
				Events []struct {
					Type string `json:"type"`
				} `json:"events"`
			} `json:"recent"`
		} `json:"tools"`
		Documents struct {
			Items []struct {
				Kind     string   `json:"kind"`
				Title    string   `json:"title"`
				Commands []string `json:"commands"`
			} `json:"items"`
		} `json:"documents"`
	}
	if err := json.Unmarshal(data, &snapshot); err != nil {
		t.Fatalf("parse demo snapshot: %v", err)
	}

	if len(snapshot.ProjectHighlights) < 5 {
		t.Fatalf("project highlights = %d, want at least 5", len(snapshot.ProjectHighlights))
	}
	if snapshot.Incident.ID == "" || snapshot.Incident.Severity != "critical" || snapshot.Incident.Status != "resolved" {
		t.Fatalf("incident summary is not a resolved critical incident: %#v", snapshot.Incident)
	}
	if len(snapshot.Incident.Story) < 4 {
		t.Fatalf("incident story steps = %d, want at least 4", len(snapshot.Incident.Story))
	}
	for _, item := range snapshot.Incident.Story {
		if item.Agent == "" || item.Action == "" || len(item.Steps) < 2 {
			t.Fatalf("incident story item lacks concrete execution steps: %#v", item)
		}
	}
	if len(snapshot.Tools.Recent) < 5 {
		t.Fatalf("tool calls = %d, want at least 5", len(snapshot.Tools.Recent))
	}
	for _, call := range snapshot.Tools.Recent {
		if len(call.Events) < 2 {
			t.Fatalf("tool call %q should include detailed trace events", call.Name)
		}
	}
	if len(snapshot.Documents.Items) < 4 {
		t.Fatalf("documents = %d, want at least 4", len(snapshot.Documents.Items))
	}
}
