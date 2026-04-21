package skills

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCoreSkillsHaveClaudeStyleSkillDocs(t *testing.T) {
	root := repoRoot(t)
	for _, name := range []string{
		"log-analysis",
		"health-check",
		"service-management",
		"alerting",
		"metrics-collection",
		"network-diagnosis",
		"database-operation",
		"file-operation",
		"notification",
	} {
		path := filepath.Join(root, "skills", name, "SKILL.md")
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("expected Claude-style skill doc for %q at %s: %v", name, path, err)
		}

		content := string(data)
		if !strings.HasPrefix(content, "---\n") {
			t.Fatalf("skill doc %q must start with YAML frontmatter", name)
		}
		if !strings.Contains(content, "name: "+name+"\n") {
			t.Fatalf("skill doc %q must declare matching frontmatter name", name)
		}
		if !strings.Contains(content, "RegisterCoreSkills") || !strings.Contains(content, `"`+name+`"`) {
			t.Fatalf("skill doc %q must reference the Go runtime skill invocation", name)
		}
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("could not find repository root from %s", dir)
		}
		dir = parent
	}
}
