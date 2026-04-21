package security

import (
	"fmt"
	"strings"
	"time"
	"unicode"
)

const (
	PermissionReadOnly   = "read_only"
	PermissionPrivileged = "privileged"
	PermissionDangerous  = "dangerous"
)

type CommandTemplate struct {
	Name       string
	Permission string
	Program    string
	Args       []string
}

type CommandAudit struct {
	Command    string
	Template   string
	Permission string
	Allowed    bool
	Reason     string
	Timestamp  time.Time
}

type CommandPolicy struct {
	templates []CommandTemplate
}

func NewCommandPolicy(templates []CommandTemplate) *CommandPolicy {
	return &CommandPolicy{templates: append([]CommandTemplate(nil), templates...)}
}

func DefaultCommandPolicy() *CommandPolicy {
	return NewCommandPolicy([]CommandTemplate{
		{Name: "journalctl-service-tail", Permission: PermissionReadOnly, Program: "journalctl", Args: []string{"-u", "{service}", "-n", "{number}", "--no-pager"}},
		{Name: "systemctl-restart", Permission: PermissionPrivileged, Program: "systemctl", Args: []string{"restart", "{service}"}},
		{Name: "systemctl-is-active", Permission: PermissionReadOnly, Program: "systemctl", Args: []string{"is-active", "{service}"}},
		{Name: "pgrep-exact", Permission: PermissionReadOnly, Program: "pgrep", Args: []string{"-x", "{service}"}},
		{Name: "true", Permission: PermissionReadOnly, Program: "true"},
	})
}

func (p *CommandPolicy) Validate(command string) (CommandAudit, error) {
	audit := CommandAudit{Command: command, Timestamp: time.Now().UTC()}
	parts := strings.Fields(command)
	if len(parts) == 0 {
		audit.Reason = "empty command"
		return audit, fmt.Errorf(audit.Reason)
	}
	for _, part := range parts {
		if !safeToken(part) {
			audit.Reason = "unsafe command argument"
			return audit, fmt.Errorf("%s: %s", audit.Reason, part)
		}
	}
	for _, tmpl := range p.templates {
		if templateMatches(tmpl, parts) {
			audit.Template = tmpl.Name
			audit.Permission = tmpl.Permission
			audit.Allowed = true
			return audit, nil
		}
	}
	audit.Reason = "command does not match an allowlisted template"
	return audit, fmt.Errorf(audit.Reason)
}

func templateMatches(tmpl CommandTemplate, parts []string) bool {
	if len(parts) != len(tmpl.Args)+1 || parts[0] != tmpl.Program {
		return false
	}
	for i, expected := range tmpl.Args {
		got := parts[i+1]
		switch expected {
		case "{service}":
			if !safeIdentifier(got) {
				return false
			}
		case "{number}":
			if !allDigits(got) {
				return false
			}
		default:
			if got != expected {
				return false
			}
		}
	}
	return true
}

func safeToken(token string) bool {
	for _, r := range token {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			continue
		}
		switch r {
		case '-', '_', '.', '/', ':', '@':
			continue
		default:
			return false
		}
	}
	return true
}

func safeIdentifier(token string) bool {
	if token == "" {
		return false
	}
	for _, r := range token {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_' || r == '.' || r == '@' {
			continue
		}
		return false
	}
	return true
}

func allDigits(token string) bool {
	if token == "" {
		return false
	}
	for _, r := range token {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}
