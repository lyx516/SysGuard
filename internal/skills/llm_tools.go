package skills

import (
	"context"
	"fmt"
)

const (
	PermissionReadOnly   = "read_only"
	PermissionPrivileged = "privileged"
	PermissionDangerous  = "dangerous"
)

type ToolHandler func(ctx context.Context, args map[string]interface{}) (ToolResult, error)

type ToolDefinition struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Permission  string      `json:"permission"`
	Parameters  JSONSchema  `json:"parameters"`
	Handler     ToolHandler `json:"-"`
}

type JSONSchema struct {
	Type       string                        `json:"type"`
	Required   []string                      `json:"required,omitempty"`
	Properties map[string]JSONSchemaProperty `json:"properties,omitempty"`
}

type JSONSchemaProperty struct {
	Type        string   `json:"type"`
	Description string   `json:"description,omitempty"`
	Enum        []string `json:"enum,omitempty"`
}

type ToolResult struct {
	Success  bool                   `json:"success"`
	Data     interface{}            `json:"data,omitempty"`
	Error    string                 `json:"error,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

func CoreSkillToolDefinitions(registry *SkillRegistry) ([]ToolDefinition, error) {
	if registry == nil {
		return nil, fmt.Errorf("skill registry is required")
	}

	definitions := []ToolDefinition{
		skillTool(registry, "log-analysis", PermissionReadOnly, JSONSchema{
			Type:     "object",
			Required: []string{"path"},
			Properties: map[string]JSONSchemaProperty{
				"path":       {Type: "string", Description: "Log file path"},
				"chunk_size": {Type: "number", Description: "Lines per chunk"},
				"keywords":   {Type: "array", Description: "Keywords to match"},
			},
		}),
		skillTool(registry, "health-check", PermissionReadOnly, JSONSchema{Type: "object"}),
		skillTool(registry, "metrics-collection", PermissionReadOnly, JSONSchema{Type: "object"}),
		skillTool(registry, "network-diagnosis", PermissionReadOnly, JSONSchema{
			Type:     "object",
			Required: []string{"operation"},
			Properties: map[string]JSONSchemaProperty{
				"operation": {Type: "string", Enum: []string{"interfaces", "dns", "tcp", "ping"}},
				"host":      {Type: "string"},
				"port":      {Type: "number"},
				"timeout":   {Type: "string"},
			},
		}),
		skillTool(registry, "service-management", PermissionPrivileged, JSONSchema{
			Type:     "object",
			Required: []string{"operation", "service"},
			Properties: map[string]JSONSchemaProperty{
				"operation":       {Type: "string", Enum: []string{"status", "logs", "start", "stop", "restart"}},
				"service":         {Type: "string"},
				"lines":           {Type: "number"},
				"allow_dangerous": {Type: "boolean", Description: "Requires explicit approval for state-changing operations"},
			},
		}),
		skillTool(registry, "database-operation", PermissionReadOnly, JSONSchema{
			Type:     "object",
			Required: []string{"driver", "dsn", "operation"},
			Properties: map[string]JSONSchemaProperty{
				"driver":    {Type: "string"},
				"dsn":       {Type: "string"},
				"operation": {Type: "string", Enum: []string{"ping", "query"}},
				"query":     {Type: "string"},
				"limit":     {Type: "number"},
			},
		}),
		skillTool(registry, "file-operation", PermissionReadOnly, JSONSchema{
			Type:     "object",
			Required: []string{"operation", "path"},
			Properties: map[string]JSONSchemaProperty{
				"operation": {Type: "string", Enum: []string{"read", "stat", "list", "tail"}},
				"path":      {Type: "string"},
				"lines":     {Type: "number"},
			},
		}),
		skillTool(registry, "alerting", PermissionReadOnly, JSONSchema{
			Type:     "object",
			Required: []string{"title", "message"},
			Properties: map[string]JSONSchemaProperty{
				"title":    {Type: "string"},
				"message":  {Type: "string"},
				"severity": {Type: "string", Enum: []string{"info", "warning", "critical"}},
				"source":   {Type: "string"},
			},
		}),
		skillTool(registry, "notification", PermissionPrivileged, JSONSchema{
			Type:     "object",
			Required: []string{"channel", "message"},
			Properties: map[string]JSONSchemaProperty{
				"channel": {Type: "string", Enum: []string{"stdout", "log", "webhook"}},
				"target":  {Type: "string"},
				"message": {Type: "string"},
			},
		}),
	}
	return definitions, nil
}

func skillTool(registry *SkillRegistry, name, permission string, schema JSONSchema) ToolDefinition {
	skill, _ := registry.Get(name)
	description := name
	if skill != nil {
		description = skill.Description()
	}
	return ToolDefinition{
		Name:        name,
		Description: description,
		Permission:  permission,
		Parameters:  schema,
		Handler: func(ctx context.Context, args map[string]interface{}) (ToolResult, error) {
			out, err := registry.Execute(ctx, name, &SkillInput{Params: args})
			if err != nil {
				return ToolResult{}, err
			}
			result := ToolResult{
				Success:  out.Success,
				Data:     out.Result,
				Metadata: map[string]interface{}{},
			}
			if out.Error != nil {
				result.Error = out.Error.Error()
			}
			for key, value := range out.Metadata {
				result.Metadata[key] = value
			}
			return result, nil
		},
	}
}
