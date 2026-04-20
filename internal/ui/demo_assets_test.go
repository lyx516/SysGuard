package ui

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
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
		Operations struct {
			SLO struct {
				Availability string `json:"availability"`
			} `json:"slo"`
			Queue []struct {
				ID string `json:"id"`
			} `json:"queue"`
			Evidence []struct {
				Name string `json:"name"`
			} `json:"evidence"`
			Graph struct {
				Nodes []struct {
					ID string `json:"id"`
				} `json:"nodes"`
				Edges []struct {
					From string `json:"from"`
					To   string `json:"to"`
				} `json:"edges"`
			} `json:"graph"`
		} `json:"operations"`
		Incident struct {
			ID       string `json:"id"`
			Severity string `json:"severity"`
			Status   string `json:"status"`
			Story    []struct {
				Agent          string   `json:"agent"`
				Action         string   `json:"action"`
				StartedAt      string   `json:"started_at"`
				DurationMillis int      `json:"duration_millis"`
				Steps          []string `json:"steps"`
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

	if snapshot.Operations.SLO.Availability == "" {
		t.Fatal("operations SLO availability should be populated")
	}
	if len(snapshot.Operations.Queue) < 3 {
		t.Fatalf("operations queue = %d, want at least 3", len(snapshot.Operations.Queue))
	}
	if len(snapshot.Operations.Evidence) < 5 {
		t.Fatalf("operations evidence = %d, want at least 5", len(snapshot.Operations.Evidence))
	}
	if len(snapshot.Operations.Graph.Nodes) < 5 || len(snapshot.Operations.Graph.Edges) < 5 {
		t.Fatalf("operations graph nodes=%d edges=%d, want at least 5 each", len(snapshot.Operations.Graph.Nodes), len(snapshot.Operations.Graph.Edges))
	}
	if snapshot.Incident.ID == "" || snapshot.Incident.Severity == "" || snapshot.Incident.Status != "resolved" {
		t.Fatalf("incident summary is not a resolved realistic incident: %#v", snapshot.Incident)
	}
	if len(snapshot.Incident.Story) < 4 {
		t.Fatalf("incident story steps = %d, want at least 4", len(snapshot.Incident.Story))
	}
	for _, item := range snapshot.Incident.Story {
		if item.Agent == "" || item.Action == "" || len(item.Steps) < 2 {
			t.Fatalf("incident story item lacks concrete execution steps: %#v", item)
		}
		if item.StartedAt == "" || item.DurationMillis <= 0 {
			t.Fatalf("incident story item should use real timestamps and durations for gantt rendering: %#v", item)
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

	html, err := os.ReadFile(filepath.Join("..", "..", "docs", "demo", "index.html"))
	if err != nil {
		t.Fatalf("read demo html: %v", err)
	}
	htmlText := string(html)
	for _, needle := range []string{"处理甘特图", "Agent 信息传输图", "组件健康拓扑", "工具调用时序"} {
		if !strings.Contains(htmlText, needle) {
			t.Fatalf("demo html should include visualization %q", needle)
		}
	}
	for _, fakePattern := range []string{"index * 14", "22 - Math.min"} {
		if strings.Contains(htmlText, fakePattern) {
			t.Fatalf("demo gantt should not use fake index-based timing pattern %q", fakePattern)
		}
	}
}
