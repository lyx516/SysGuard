package security

import (
	"strings"
)

// CommandInterceptor 命令拦截器，拦截危险命令
type CommandInterceptor struct {
	dangerousCommands map[string]bool
	whitelist         map[string]bool
}

// NewCommandInterceptor 创建新的命令拦截器
func NewCommandInterceptor() *CommandInterceptor {
	return &CommandInterceptor{
		dangerousCommands: map[string]bool{
			"rm":      true,
			"kill":    true,
			"killall": true,
			"dd":      true,
			":(){:|:&};:": true, // fork bomb
			"mkfs":    true,
			"shutdown": true,
			"reboot":  true,
			"init 0":  true,
		},
		whitelist: make(map[string]bool),
	}
}

// IsDangerous 检查命令是否为危险命令
func (ci *CommandInterceptor) IsDangerous(command string) bool {
	// 检查是否在白名单中
	if ci.whitelist[command] {
		return false
	}

	// 检查是否为危险命令
	for dangerousCmd := range ci.dangerousCommands {
		if strings.HasPrefix(command, dangerousCmd) {
			return true
		}
	}

	return false
}

// AddToWhitelist 将命令添加到白名单
func (ci *CommandInterceptor) AddToWhitelist(command string) {
	ci.whitelist[command] = true
}

// RemoveFromWhitelist 从白名单中移除命令
func (ci *CommandInterceptor) RemoveFromWhitelist(command string) {
	delete(ci.whitelist, command)
}
