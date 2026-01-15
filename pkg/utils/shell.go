package utils

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"syscall"
	"time"
)

// ShellExecutor Shell 命令执行器
type ShellExecutor struct {
	timeout time.Duration
}

// NewShellExecutor 创建新的 Shell 执行器
func NewShellExecutor(timeout time.Duration) *ShellExecutor {
	return &ShellExecutor{
		timeout: timeout,
	}
}

// Execute 执行 Shell 命令
func (se *ShellExecutor) Execute(ctx context.Context, command string) (*ExecutionResult, error) {
	// 设置超时
	ctx, cancel := context.WithTimeout(ctx, se.timeout)
	defer cancel()

	// 解析命令
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return nil, fmt.Errorf("empty command")
	}

	// 创建命令
	cmd := exec.CommandContext(ctx, parts[0], parts[1:]...)

	// 执行命令
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	startTime := time.Now()
	err := cmd.Run()
	endTime := time.Now()

	// 构建结果
	result := &ExecutionResult{
		Command:   command,
		Stdout:    stdout.String(),
		Stderr:    stderr.String(),
		StartTime: startTime,
		EndTime:   endTime,
		Duration:  endTime.Sub(startTime),
	}

	// 获取退出状态码
	if exitErr, ok := err.(*exec.ExitError); ok {
		if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
			result.ExitCode = status.ExitStatus()
		}
	}

	if err != nil {
		result.Success = false
		return result, fmt.Errorf("command execution failed: %w", err)
	}

	result.Success = true
	return result, nil
}

// ExecuteSafe 安全执行命令，包含参数验证
func (se *ShellExecutor) ExecuteSafe(ctx context.Context, command string, validator CommandValidator) (*ExecutionResult, error) {
	// 验证命令
	if err := validator.Validate(command); err != nil {
		return nil, fmt.Errorf("command validation failed: %w", err)
	}

	// 执行命令
	return se.Execute(ctx, command)
}

// ExecutionResult 执行结果
type ExecutionResult struct {
	Command   string
	Stdout    string
	Stderr    string
	Success   bool
	ExitCode  int
	StartTime time.Time
	EndTime   time.Time
	Duration  time.Duration
}

// CommandValidator 命令验证器接口
type CommandValidator interface {
	Validate(command string) error
}

// DefaultValidator 默认命令验证器
type DefaultValidator struct{}

// NewDefaultValidator 创建默认验证器
func NewDefaultValidator() *DefaultValidator {
	return &DefaultValidator{}
}

// Validate 验证命令
func (dv *DefaultValidator) Validate(command string) error {
	if command == "" {
		return fmt.Errorf("command cannot be empty")
	}

	// 检查是否包含潜在的命令注入
	if strings.Contains(command, ";") || strings.Contains(command, "&") || strings.Contains(command, "|") {
		return fmt.Errorf("command contains potentially dangerous characters")
	}

	return nil
}
