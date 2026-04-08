package skills

import (
	"github.com/sysguard/sysguard/internal/skills/alerting"
	"github.com/sysguard/sysguard/internal/skills/container_management"
	"github.com/sysguard/sysguard/internal/skills/database_operations"
	"github.com/sysguard/sysguard/internal/skills/file_operations"
	"github.com/sysguard/sysguard/internal/skills/health_check"
	"github.com/sysguard/sysguard/internal/skills/log_analysis"
	"github.com/sysguard/sysguard/internal/skills/metrics"
	"github.com/sysguard/sysguard/internal/skills/network_diagnosis"
	"github.com/sysguard/sysguard/internal/skills/notification"
	"github.com/sysguard/sysguard/internal/skills/remediation_workflow"
	"github.com/sysguard/sysguard/internal/skills/service_management"
)

// NewDefaultRegistry 创建默认的 Skill 注册表
func NewDefaultRegistry() *SkillRegistry {
	registry := NewSkillRegistry()

	// 注册所有默认 Skills
	registry.Register(log_analysis.NewLogAnalysisSkill(1000, []string{"error", "failed", "warning"}))
	registry.Register(health_check.NewHealthCheckSkill(nil)) // TODO: 传入实际的 monitor
	registry.Register(service_management.NewServiceManagementSkill())
	registry.Register(alerting.NewAlertingSkill(nil)) // TODO: 传入实际的 notifier
	registry.Register(metrics.NewMetricsSkill())
	registry.Register(network_diagnosis.NewNetworkDiagnosisSkill())
	registry.Register(container_management.NewContainerManagementSkill())
	registry.Register(database_operations.NewDatabaseOperationsSkill())
	registry.Register(file_operations.NewFileOperationsSkill())
	registry.Register(notification.NewNotificationSkill())

	// 注意：RemediationWorkflowSkill 需要在 Coordinator 中动态注册，
	// 因为它依赖于 HistoryKnowledgeBase 和 SOP KnowledgeBase

	return registry
}

// RegisterWorkflowSkill 注册工作流 Skill
func RegisterWorkflowSkill(
	registry *SkillRegistry,
	historyKB interface{},
	sopKB *rag.KnowledgeBase,
) error {
	workflowSkill := remediation_workflow.NewRemediationWorkflowSkill(
		historyKB,
		sopKB,
		registry,
	)
	return registry.Register(workflowSkill)
}

// GetSkillNames 获取所有 Skill 名称
func GetSkillNames(registry *SkillRegistry) []string {
	skills := registry.List()
	names := make([]string, 0, len(skills))
	for _, skill := range skills {
		names = append(names, skill.Name())
	}
	return names
}

// GetSkillsByCategory 获取指定类别的 Skills
func GetSkillsByCategory(registry *SkillRegistry, category string) []Skill {
	return registry.Search(category, nil)
}

// GetSkillsByTag 获取指定标签的 Skills
func GetSkillsByTag(registry *SkillRegistry, tags []string) []Skill {
	return registry.Search("", tags)
}
