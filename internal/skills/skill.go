package skills

import (
	"context"
	"fmt"
)

// Skill 表示一个可复用的 Agent 能力单元
type Skill interface {
	// Name 返回 Skill 名称
	Name() string

	// Description 返回 Skill 描述
	Description() string

	// Execute 执行 Skill
	Execute(ctx context.Context, input *SkillInput) (*SkillOutput, error)

	// Tools 返回该 Skill 使用的工具集
	Tools() []Tool

	// Metadata 返回 Skill 元数据
	Metadata() *SkillMetadata
}

// Tool 表示 Agent 可以调用的工具
type Tool interface {
	// Name 返回工具名称
	Name() string

	// Description 返回工具描述
	Description() string

	// Execute 执行工具
	Execute(ctx context.Context, input *ToolInput) (*ToolOutput, error)
}

// SkillInput Skill 输入参数
type SkillInput struct {
	Command  string                 // 要执行的命令
	Params   map[string]interface{} // 参数
	Context  map[string]interface{} // 上下文信息
}

// SkillOutput Skill 输出结果
type SkillOutput struct {
	Success   bool                   // 是否成功
	Message   string                 // 结果消息
	Data      map[string]interface{} // 结果数据
	Errors    []string               // 错误信息
	Duration  int64                  // 执行时长（毫秒）
	ToolsUsed []string               // 使用的工具列表
}

// ToolInput 工具输入参数
type ToolInput struct {
	Params map[string]interface{} // 参数
}

// ToolOutput 工具输出结果
type ToolOutput struct {
	Success bool                   // 是否成功
	Data    map[string]interface{} // 结果数据
	Error   error                  // 错误信息
}

// SkillMetadata Skill 元数据
type SkillMetadata struct {
	Version     string   // 版本号
	Category    string   // 类别
	Tags        []string // 标签
	Author      string   // 作者
	Permissions []string // 所需权限
}

// SkillRegistry Skill 注册表
type SkillRegistry struct {
	skills map[string]Skill
}

// NewSkillRegistry 创建新的 Skill 注册表
func NewSkillRegistry() *SkillRegistry {
	return &SkillRegistry{
		skills: make(map[string]Skill),
	}
}

// Register 注册 Skill
func (sr *SkillRegistry) Register(skill Skill) error {
	if skill == nil {
		return fmt.Errorf("skill cannot be nil")
	}
	sr.skills[skill.Name()] = skill
	return nil
}

// Get 获取 Skill
func (sr *SkillRegistry) Get(name string) (Skill, bool) {
	skill, ok := sr.skills[name]
	return skill, ok
}

// List 列出所有 Skills
func (sr *SkillRegistry) List() []Skill {
	skills := make([]Skill, 0, len(sr.skills))
	for _, skill := range sr.skills {
		skills = append(skills, skill)
	}
	return skills
}

// Search 搜索 Skills（按类别或标签）
func (sr *SkillRegistry) Search(category string, tags []string) []Skill {
	var results []Skill

	for _, skill := range sr.skills {
		meta := skill.Metadata()

		// 检查类别
		if category != "" && meta.Category != category {
			continue
		}

		// 检查标签
		if len(tags) > 0 {
			found := false
			for _, tag := range tags {
				for _, metaTag := range meta.Tags {
					if tag == metaTag {
						found = true
						break
					}
				}
				if found {
					break
				}
			}
			if !found {
				continue
			}
		}

		results = append(results, skill)
	}

	return results
}

// GetCategories 获取所有类别
func (sr *SkillRegistry) GetCategories() []string {
	categories := make(map[string]bool)
	for _, skill := range sr.skills {
		meta := skill.Metadata()
		if meta.Category != "" {
			categories[meta.Category] = true
		}
	}

	result := make([]string, 0, len(categories))
	for category := range categories {
		result = append(result, category)
	}

	return result
}
