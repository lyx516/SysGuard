package eino

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	einotool "github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"github.com/sysguard/sysguard/internal/skills"
)

type SkillTool struct {
	def skills.ToolDefinition
}

var _ einotool.InvokableTool = (*SkillTool)(nil)

func NewSkillTool(def skills.ToolDefinition) *SkillTool {
	return &SkillTool{def: def}
}

func BuildTools(definitions []skills.ToolDefinition) []einotool.BaseTool {
	out := make([]einotool.BaseTool, 0, len(definitions))
	for _, def := range definitions {
		out = append(out, NewSkillTool(def))
	}
	return out
}

func (t *SkillTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: t.def.Name,
		Desc: t.def.Description,
		Extra: map[string]any{
			"permission":        t.def.Permission,
			"toolset":           t.def.Toolset,
			"side_effects":      t.def.SideEffects,
			"requires_approval": t.def.RequiresApproval,
			"allowed_platforms": t.def.AllowedPlatforms,
			"output_budget":     t.def.OutputBudget,
			"redaction_policy":  t.def.RedactionPolicy,
		},
		ParamsOneOf: schema.NewParamsOneOfByParams(paramsFromSchema(t.def.Parameters)),
	}, nil
}

func (t *SkillTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...einotool.Option) (string, error) {
	if t.def.Handler == nil {
		return "", fmt.Errorf("tool %q handler is required", t.def.Name)
	}
	args := make(map[string]interface{})
	if argumentsInJSON != "" {
		if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
			return toolFailureObservation(fmt.Sprintf("parse tool arguments for %s: %v", t.def.Name, err)), nil
		}
	}
	if err := validateArgs(t.def.Parameters, args); err != nil {
		return toolFailureObservation(err.Error()), nil
	}
	result, err := t.def.Handler(ctx, args)
	if err != nil {
		return toolFailureObservation(err.Error()), nil
	}
	payload, err := json.Marshal(result)
	if err != nil {
		return toolFailureObservation(fmt.Sprintf("marshal tool result for %s: %v", t.def.Name, err)), nil
	}
	return string(payload), nil
}

func toolFailureObservation(message string) string {
	payload, err := json.Marshal(skills.ToolResult{
		Success: false,
		Error:   message,
	})
	if err != nil {
		return `{"success":false,"error":"tool failed"}`
	}
	return string(payload)
}

func paramsFromSchema(schemaDef skills.JSONSchema) map[string]*schema.ParameterInfo {
	if len(schemaDef.Properties) == 0 {
		return map[string]*schema.ParameterInfo{}
	}
	required := make(map[string]bool, len(schemaDef.Required))
	for _, key := range schemaDef.Required {
		required[key] = true
	}
	keys := make([]string, 0, len(schemaDef.Properties))
	for key := range schemaDef.Properties {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	params := make(map[string]*schema.ParameterInfo, len(keys))
	for _, key := range keys {
		prop := schemaDef.Properties[key]
		params[key] = &schema.ParameterInfo{
			Type:     dataType(prop.Type),
			Desc:     prop.Description,
			Enum:     append([]string(nil), prop.Enum...),
			Required: required[key],
		}
	}
	return params
}

func dataType(value string) schema.DataType {
	switch value {
	case "boolean":
		return schema.Boolean
	case "integer":
		return schema.Integer
	case "number":
		return schema.Number
	case "array":
		return schema.Array
	case "object":
		return schema.Object
	default:
		return schema.String
	}
}

func validateArgs(schemaDef skills.JSONSchema, args map[string]interface{}) error {
	for _, key := range schemaDef.Required {
		value, ok := args[key]
		if !ok || value == nil || fmt.Sprintf("%v", value) == "" {
			return fmt.Errorf("tool argument validation failed: missing required argument %q", key)
		}
	}
	for key, prop := range schemaDef.Properties {
		value, ok := args[key]
		if !ok || len(prop.Enum) == 0 {
			continue
		}
		got := fmt.Sprintf("%v", value)
		matched := false
		for _, allowed := range prop.Enum {
			if got == allowed {
				matched = true
				break
			}
		}
		if !matched {
			return fmt.Errorf("tool argument validation failed: %q must be one of %v", key, prop.Enum)
		}
	}
	return nil
}
