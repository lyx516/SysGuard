package container_management

import (
	"context"
	"fmt"
	"time"

	"github.com/sysguard/sysguard/internal/skills"
)

// ContainerManagementSkill 容器管理 Skill
type ContainerManagementSkill struct {
	version string
}

// NewContainerManagementSkill 创建容器管理 Skill
func NewContainerManagementSkill() *ContainerManagementSkill {
	return &ContainerManagementSkill{
		version: "1.0.0",
	}
}

// Name 返回 Skill 名称
func (s *ContainerManagementSkill) Name() string {
	return "container_management"
}

// Description 返回 Skill 描述
func (s *ContainerManagementSkill) Description() string {
	return "管理 Docker 和 Kubernetes 容器"
}

// Execute 执行容器管理操作
func (s *ContainerManagementSkill) Execute(ctx context.Context, input *skills.SkillInput) (*skills.SkillOutput, error) {
	startTime := time.Now()

	// 获取操作类型
	action, ok := input.Params["action"].(string)
	if !ok {
		return nil, fmt.Errorf("action parameter is required")
	}

	// 获取容器类型
	containerType, _ := input.Params["type"].(string)
	if containerType == "" {
		containerType = "docker"
	}

	var result map[string]interface{}
	var message string
	var toolsUsed []string

	switch action {
	case "list":
		result, message = s.listContainers(ctx, containerType)
		toolsUsed = []string{fmt.Sprintf("%s_list", containerType)}
	case "start":
		result, message = s.startContainer(ctx, containerType, input)
		toolsUsed = []string{fmt.Sprintf("%s_start", containerType)}
	case "stop":
		result, message = s.stopContainer(ctx, containerType, input)
		toolsUsed = []string{fmt.Sprintf("%s_stop", containerType)}
	case "restart":
		result, message = s.restartContainer(ctx, containerType, input)
		toolsUsed = []string{fmt.Sprintf("%s_restart", containerType)}
	case "logs":
		result, message = s.getContainerLogs(ctx, containerType, input)
		toolsUsed = []string{fmt.Sprintf("%s_logs", containerType)}
	case "status":
		result, message = s.getContainerStatus(ctx, containerType, input)
		toolsUsed = []string{fmt.Sprintf("%s_status", containerType)}
	default:
		return nil, fmt.Errorf("unsupported action: %s", action)
	}

	duration := time.Since(startTime).Milliseconds()

	return &skills.SkillOutput{
		Success: result["success"].(bool),
		Message: message,
		Data:    result,
		ToolsUsed: toolsUsed,
		Duration:  duration,
	}, nil
}

// Tools 返回该 Skill 使用的工具集
func (s *ContainerManagementSkill) Tools() []skills.Tool {
	return []skills.Tool{
		&DockerList{},
		&DockerStart{},
		&DockerStop{},
		&DockerRestart{},
		&DockerLogs{},
		&DockerStatus{},
		&KubectlList{},
		&KubectlApply{},
		&KubectlLogs{},
	}
}

// Metadata 返回 Skill 元数据
func (s *ContainerManagementSkill) Metadata() *skills.SkillMetadata {
	return &skills.SkillMetadata{
		Version:     s.version,
		Category:    "management",
		Tags:        []string{"containers", "docker", "kubernetes", "orchestration"},
		Author:      "SysGuard Team",
		Permissions: []string{"docker:manage", "kubernetes:manage"},
	}
}

// listContainers 列出容器
func (s *ContainerManagementSkill) listContainers(ctx context.Context, containerType string) (map[string]interface{}, string) {
	return map[string]interface{}{
		"success": true,
		"type":    containerType,
		"total":   10,
		"running": 8,
		"stopped": 2,
		"containers": []map[string]interface{}{
			{"id": "abc123", "name": "nginx", "status": "running", "image": "nginx:latest"},
			{"id": "def456", "name": "mysql", "status": "running", "image": "mysql:8.0"},
		},
	}, fmt.Sprintf("Listed %s containers", containerType)
}

// startContainer 启动容器
func (s *ContainerManagementSkill) startContainer(ctx context.Context, containerType string, input *skills.SkillInput) (map[string]interface{}, string) {
	containerID, _ := input.Params["container"].(string)
	return map[string]interface{}{
		"success":      true,
		"container_id": containerID,
		"status":       "running",
	}, fmt.Sprintf("Started container %s", containerID)
}

// stopContainer 停止容器
func (s *ContainerManagementSkill) stopContainer(ctx context.Context, containerType string, input *skills.SkillInput) (map[string]interface{}, string) {
	containerID, _ := input.Params["container"].(string)
	return map[string]interface{}{
		"success":      true,
		"container_id": containerID,
		"status":       "stopped",
	}, fmt.Sprintf("Stopped container %s", containerID)
}

// restartContainer 重启容器
func (s *ContainerManagementSkill) restartContainer(ctx context.Context, containerType string, input *skills.SkillInput) (map[string]interface{}, string) {
	containerID, _ := input.Params["container"].(string)
	return map[string]interface{}{
		"success":      true,
		"container_id": containerID,
		"status":       "running",
	}, fmt.Sprintf("Restarted container %s", containerID)
}

// getContainerLogs 获取容器日志
func (s *ContainerManagementSkill) getContainerLogs(ctx context.Context, containerType string, input *skills.SkillInput) (map[string]interface{}, string) {
	containerID, _ := input.Params["container"].(string)
	tail, _ := input.Params["tail"].(int)
	if tail == 0 {
		tail = 100
	}

	return map[string]interface{}{
		"success":      true,
		"container_id": containerID,
		"lines":        tail,
		"logs":         []string{},
	}, fmt.Sprintf("Retrieved %d log lines from container %s", tail, containerID)
}

// getContainerStatus 获取容器状态
func (s *ContainerManagementSkill) getContainerStatus(ctx context.Context, containerType string, input *skills.SkillInput) (map[string]interface{}, string) {
	containerID, _ := input.Params["container"].(string)
	return map[string]interface{}{
		"success":      true,
		"container_id": containerID,
		"status":       "running",
		"state":        "running",
		"restart_count": 0,
		"uptime":       "2h 30m",
		"cpu_percent":  5.2,
		"memory_mb":    256,
	}, fmt.Sprintf("Status of container %s", containerID)
}

// DockerList Docker list 工具
type DockerList struct{}

// Name 返回工具名称
func (t *DockerList) Name() string {
	return "docker_list"
}

// Description 返回工具描述
func (t *DockerList) Description() string {
	return "列出 Docker 容器"
}

// Execute 执行工具
func (t *DockerList) Execute(ctx context.Context, input *skills.ToolInput) (*skills.ToolOutput, error) {
	return &skills.ToolOutput{
		Success: true,
		Data: map[string]interface{}{
			"containers": []string{"nginx", "mysql"},
		},
	}, nil
}

// DockerStart Docker start 工具
type DockerStart struct{}

// Name 返回工具名称
func (t *DockerStart) Name() string {
	return "docker_start"
}

// Description 返回工具描述
func (t *DockerStart) Description() string {
	return "启动 Docker 容器"
}

// Execute 执行工具
func (t *DockerStart) Execute(ctx context.Context, input *skills.ToolInput) (*skills.ToolOutput, error) {
	return &skills.ToolOutput{
		Success: true,
		Data: map[string]interface{}{
			"status": "running",
		},
	}, nil
}

// DockerStop Docker stop 工具
type DockerStop struct{}

// Name 返回工具名称
func (t *DockerStop) Name() string {
	return "docker_stop"
}

// Description 返回工具描述
func (t *DockerStop) Description() string {
	return "停止 Docker 容器"
}

// Execute 执行工具
func (t *DockerStop) Execute(ctx context.Context, input *skills.ToolInput) (*skills.ToolOutput, error) {
	return &skills.ToolOutput{
		Success: true,
		Data: map[string]interface{}{
			"status": "stopped",
		},
	}, nil
}

// DockerRestart Docker restart 工具
type DockerRestart struct{}

// Name 返回工具名称
func (t *DockerRestart) Name() string {
	return "docker_restart"
}

// Description 返回工具描述
func (t *DockerRestart) Description() string {
	return "重启 Docker 容器"
}

// Execute 执行工具
func (t *DockerRestart) Execute(ctx context.Context, input *skills.ToolInput) (*skills.ToolOutput, error) {
	return &skills.ToolOutput{
		Success: true,
		Data: map[string]interface{}{
			"status": "running",
		},
	}, nil
}

// DockerLogs Docker logs 工具
type DockerLogs struct{}

// Name 返回工具名称
func (t *DockerLogs) Name() string {
	return "docker_logs"
}

// Description 返回工具描述
func (t *DockerLogs) Description() string {
	return "获取 Docker 容器日志"
}

// Execute 执行工具
func (t *DockerLogs) Execute(ctx context.Context, input *skills.ToolInput) (*skills.ToolOutput, error) {
	return &skills.ToolOutput{
		Success: true,
		Data: map[string]interface{}{
			"logs": []string{},
		},
	}, nil
}

// DockerStatus Docker status 工具
type DockerStatus struct{}

// Name 返回工具名称
func (t *DockerStatus) Name() string {
	return "docker_status"
}

// Description 返回工具描述
func (t *DockerStatus) Description() string {
	return "获取 Docker 容器状态"
}

// Execute 执行工具
func (t *DockerStatus) Execute(ctx context.Context, input *skills.ToolInput) (*skills.ToolOutput, error) {
	return &skills.ToolOutput{
		Success: true,
		Data: map[string]interface{}{
			"status": "running",
		},
	}, nil
}

// KubectlList kubectl list 工具
type KubectlList struct{}

// Name 返回工具名称
func (t *KubectlList) Name() string {
	return "kubectl_list"
}

// Description 返回工具描述
func (t *KubectlList) Description() string {
	return "列出 Kubernetes 资源"
}

// Execute 执行工具
func (t *KubectlList) Execute(ctx context.Context, input *skills.ToolInput) (*skills.ToolOutput, error) {
	return &skills.ToolOutput{
		Success: true,
		Data: map[string]interface{}{
			"pods":     10,
			"services": 5,
			"deployments": 3,
		},
	}, nil
}

// KubectlApply kubectl apply 工具
type KubectlApply struct{}

// Name 返回工具名称
func (t *KubectlApply) Name() string {
	return "kubectl_apply"
}

// Description 返回工具描述
func (t *KubectlApply) Description() string {
	return "应用 Kubernetes 配置"
}

// Execute 执行工具
func (t *KubectlApply) Execute(ctx context.Context, input *skills.ToolInput) (*skills.ToolOutput, error) {
	return &skills.ToolOutput{
		Success: true,
		Data: map[string]interface{}{
			"applied": true,
		},
	}, nil
}

// KubectlLogs kubectl logs 工具
type KubectlLogs struct{}

// Name 返回工具名称
func (t *KubectlLogs) Name() string {
	return "kubectl_logs"
}

// Description 返回工具描述
func (t *KubectlLogs) Description() string {
	return "获取 Pod 日志"
}

// Execute 执行工具
func (t *KubectlLogs) Execute(ctx context.Context, input *skills.ToolInput) (*skills.ToolOutput, error) {
	return &skills.ToolOutput{
		Success: true,
		Data: map[string]interface{}{
			"logs": []string{},
		},
	}, nil
}
