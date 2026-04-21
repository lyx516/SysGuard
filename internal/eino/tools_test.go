package eino

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/sysguard/sysguard/internal/skills"
)

func TestSkillToolReturnsValidationFailureAsObservation(t *testing.T) {
	tool := NewSkillTool(skills.ToolDefinition{
		Name:       "demo",
		Permission: skills.PermissionReadOnly,
		Parameters: skills.JSONSchema{
			Type:     "object",
			Required: []string{"query"},
			Properties: map[string]skills.JSONSchemaProperty{
				"query": {Type: "string"},
			},
		},
		Handler: func(ctx context.Context, args map[string]interface{}) (skills.ToolResult, error) {
			return skills.ToolResult{Success: true}, nil
		},
	})

	out, err := tool.InvokableRun(context.Background(), `{}`)
	if err != nil {
		t.Fatalf("InvokableRun() error = %v, want nil tool observation", err)
	}

	var result skills.ToolResult
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("tool output is not JSON: %v", err)
	}
	if result.Success {
		t.Fatalf("tool output Success = true, want false")
	}
	if result.Error == "" {
		t.Fatalf("tool output Error is empty, want validation message")
	}
}

func TestSkillToolInfoPreservesPermissionMetadata(t *testing.T) {
	tool := NewSkillTool(skills.ToolDefinition{
		Name:        "demo",
		Description: "demo tool",
		Permission:  skills.PermissionPrivileged,
		Parameters:  skills.JSONSchema{Type: "object"},
		Handler: func(ctx context.Context, args map[string]interface{}) (skills.ToolResult, error) {
			return skills.ToolResult{Success: true}, nil
		},
	})

	info, err := tool.Info(context.Background())
	if err != nil {
		t.Fatalf("Info() error = %v", err)
	}
	if info.Extra["permission"] != skills.PermissionPrivileged {
		t.Fatalf("Info().Extra permission = %v, want %s", info.Extra["permission"], skills.PermissionPrivileged)
	}
}

func TestSkillToolReturnsHandlerFailureAsObservation(t *testing.T) {
	tool := NewSkillTool(skills.ToolDefinition{
		Name:       "demo",
		Permission: skills.PermissionReadOnly,
		Parameters: skills.JSONSchema{Type: "object"},
		Handler: func(ctx context.Context, args map[string]interface{}) (skills.ToolResult, error) {
			return skills.ToolResult{}, errors.New("backend unavailable")
		},
	})

	out, err := tool.InvokableRun(context.Background(), `{}`)
	if err != nil {
		t.Fatalf("InvokableRun() error = %v, want nil tool observation", err)
	}

	var result skills.ToolResult
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("tool output is not JSON: %v", err)
	}
	if result.Success {
		t.Fatalf("tool output Success = true, want false")
	}
	if result.Error != "backend unavailable" {
		t.Fatalf("tool output Error = %q, want backend unavailable", result.Error)
	}
}
