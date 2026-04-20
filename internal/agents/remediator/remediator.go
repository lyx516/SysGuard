package remediator

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/sysguard/sysguard/internal/config"
	"github.com/sysguard/sysguard/internal/monitor"
	"github.com/sysguard/sysguard/internal/observability"
	"github.com/sysguard/sysguard/internal/rag"
	"github.com/sysguard/sysguard/internal/security"
	"github.com/sysguard/sysguard/pkg/utils"
)

type Remediator struct {
	cfg         *config.Config
	kb          *rag.KnowledgeBase
	historyKB   *rag.HistoryKnowledgeBase
	interceptor *security.CommandInterceptor
	obs         *observability.GlobalCallback
	executor    *utils.ShellExecutor
	verifier    RemediationVerifier
	approvalCh  chan *ApprovalRequest
	cancel      context.CancelFunc
	wg          sync.WaitGroup
}

type RemediationVerifier func(ctx context.Context, anomaly monitor.Anomaly) error

type ApprovalRequest struct {
	Command    string
	Reason     string
	ResponseCh chan bool
}

type RemediationPlan struct {
	Title    string
	Commands []string
	Notes    []string
}

func NewRemediator(
	cfg *config.Config,
	kb *rag.KnowledgeBase,
	historyKB *rag.HistoryKnowledgeBase,
	interceptor *security.CommandInterceptor,
	obs *observability.GlobalCallback,
) *Remediator {
	return &Remediator{
		cfg:         cfg,
		kb:          kb,
		historyKB:   historyKB,
		interceptor: interceptor,
		obs:         obs,
		executor:    utils.NewShellExecutor(cfg.Agents.Remediator.CommandTimeout),
		approvalCh:  make(chan *ApprovalRequest, 32),
	}
}

func (r *Remediator) SetVerifier(verifier RemediationVerifier) {
	r.verifier = verifier
}

func (r *Remediator) Remediate(ctx context.Context, anomaly monitor.Anomaly) error {
	log.Printf("Remediator: Starting remediation for - %v", anomaly)

	callbackID := r.obs.OnCallbackStarted("Remediator.remediate")
	sops, err := r.kb.Retrieve(ctx, anomaly.Description)
	if err != nil {
		r.obs.OnCallbackError(callbackID, err)
		return err
	}

	plan, err := r.createRemediationPlan(ctx, anomaly, sops)
	if err != nil {
		r.obs.OnCallbackError(callbackID, err)
		return err
	}
	log.Printf("Remediator: Plan %q with %d command(s)", plan.Title, len(plan.Commands))

	if len(plan.Commands) == 0 {
		log.Printf("Remediator: No executable commands generated for anomaly %q", anomaly.Description)
		r.obs.OnCallbackCompleted(callbackID, map[string]interface{}{
			"anomaly": anomaly,
			"notes":   plan.Notes,
		})
		return nil
	}

	if r.cfg.Agents.Remediator.DryRun {
		log.Printf("Remediator: Dry-run enabled; planned %d command(s) without execution", len(plan.Commands))
		if err := r.recordHistory(ctx, anomaly, plan, false, map[string]string{"dry_run": "true"}); err != nil {
			log.Printf("Remediator: Failed to persist dry-run history - %v", err)
		}
		r.obs.OnCallbackCompleted(callbackID, map[string]interface{}{
			"anomaly":          anomaly,
			"plan":             plan.Title,
			"dry_run":          true,
			"planned_commands": len(plan.Commands),
		})
		return nil
	}

	if err := r.executeRemediation(ctx, plan); err != nil {
		r.obs.OnCallbackError(callbackID, err)
		return err
	}

	if r.cfg.Agents.Remediator.VerifyAfterRemediation && r.verifier != nil {
		if err := r.verifier(ctx, anomaly); err != nil {
			verifyErr := fmt.Errorf("verification failed: %w", err)
			if historyErr := r.recordHistory(ctx, anomaly, plan, false, map[string]string{"verify_error": err.Error()}); historyErr != nil {
				log.Printf("Remediator: Failed to persist failed verification history - %v", historyErr)
			}
			r.obs.OnCallbackError(callbackID, verifyErr)
			return verifyErr
		}
	}

	if err := r.recordHistory(ctx, anomaly, plan, true, nil); err != nil {
		log.Printf("Remediator: Failed to persist history - %v", err)
	}

	r.obs.OnCallbackCompleted(callbackID, map[string]interface{}{
		"anomaly": anomaly,
		"plan":    plan.Title,
	})
	log.Println("Remediator: Remediation completed")
	return nil
}

func (r *Remediator) recordHistory(ctx context.Context, anomaly monitor.Anomaly, plan *RemediationPlan, success bool, extra map[string]string) error {
	metadata := make(map[string]string, len(anomaly.Metadata)+len(extra))
	for k, v := range anomaly.Metadata {
		metadata[k] = v
	}
	for k, v := range extra {
		metadata[k] = v
	}

	record := &rag.HistoryRecord{
		ProblemType: anomaly.Source,
		Description: anomaly.Description,
		Solution:    plan.Title,
		Steps:       append([]string(nil), plan.Commands...),
		Success:     success,
		Timestamp:   time.Now().UTC(),
		Metadata:    metadata,
	}
	return r.historyKB.AddRecord(ctx, record)
}

func (r *Remediator) Start(ctx context.Context) error {
	runCtx, cancel := context.WithCancel(ctx)
	r.cancel = cancel
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		r.handleApprovals(runCtx)
	}()
	return nil
}

func (r *Remediator) Stop(ctx context.Context) error {
	if r.cancel != nil {
		r.cancel()
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		r.wg.Wait()
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-done:
		return nil
	}
}

func (r *Remediator) createRemediationPlan(ctx context.Context, anomaly monitor.Anomaly, sops []string) (*RemediationPlan, error) {
	similarRecords, err := r.historyKB.SearchSimilarRecords(ctx, anomaly.Description, 0.45)
	if err == nil {
		for _, record := range similarRecords {
			if record.Success && len(record.Steps) > 0 {
				return &RemediationPlan{
					Title:    "Reuse historical remediation plan",
					Commands: append([]string(nil), record.Steps...),
					Notes:    []string{"historical_match=" + record.ID},
				}, nil
			}
		}
	}

	if service := anomaly.Metadata["service_name"]; service != "" {
		commands := r.defaultServiceRecoveryCommands(service)
		return &RemediationPlan{
			Title:    fmt.Sprintf("Recover service %s", service),
			Commands: commands,
			Notes:    []string{"generated_from=service_anomaly"},
		}, nil
	}

	commands := r.parseCommands(strings.Join(sops, "\n\n"))
	commands = r.expandCommands(commands, anomaly.Metadata)
	return &RemediationPlan{
		Title:    "Execute SOP remediation plan",
		Commands: commands,
		Notes:    []string{"generated_from=sop"},
	}, nil
}

func (r *Remediator) executeRemediation(ctx context.Context, plan *RemediationPlan) error {
	validator := utils.NewDefaultValidator()
	for _, cmd := range plan.Commands {
		if strings.TrimSpace(cmd) == "" {
			continue
		}

		if err := validator.Validate(cmd); err != nil {
			return fmt.Errorf("unsafe command %q: %w", cmd, err)
		}

		if r.interceptor.IsDangerous(cmd) {
			approved := r.requestApproval(ctx, cmd, plan.Title)
			if !approved {
				return fmt.Errorf("command not approved: %s", cmd)
			}
		}

		result, err := r.executeCommand(ctx, cmd)
		if err != nil {
			return err
		}
		log.Printf("Remediator: Command succeeded cmd=%q exit=%d duration=%s", cmd, result.ExitCode, result.Duration)
	}
	return nil
}

func (r *Remediator) requestApproval(ctx context.Context, command, reason string) bool {
	if !r.cfg.Security.EnableApproval {
		return true
	}
	if !r.cfg.Agents.Remediator.AllowInteractiveInput {
		return false
	}

	req := &ApprovalRequest{
		Command:    command,
		Reason:     reason,
		ResponseCh: make(chan bool, 1),
	}

	select {
	case <-ctx.Done():
		return false
	case r.approvalCh <- req:
	}

	timeout := time.NewTimer(r.cfg.Security.ApprovalTimeout)
	defer timeout.Stop()

	select {
	case <-ctx.Done():
		return false
	case approved := <-req.ResponseCh:
		return approved
	case <-timeout.C:
		return false
	}
}

func (r *Remediator) parseCommands(plan string) []string {
	lines := strings.Split(plan, "\n")
	commands := make([]string, 0)
	inCodeBlock := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") {
			inCodeBlock = !inCodeBlock
			continue
		}
		if !inCodeBlock || trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "#") || strings.ContainsAny(trimmed, "|;&><$`") {
			continue
		}
		commands = append(commands, trimmed)
	}
	return commands
}

func (r *Remediator) expandCommands(commands []string, metadata map[string]string) []string {
	expanded := make([]string, 0, len(commands))
	service := metadata["service_name"]
	port := metadata["port"]
	for _, cmd := range commands {
		cmd = strings.ReplaceAll(cmd, "<service_name>", service)
		cmd = strings.ReplaceAll(cmd, "<port>", port)
		if strings.Contains(cmd, "<") || strings.Contains(cmd, ">") {
			continue
		}
		expanded = append(expanded, cmd)
	}
	return expanded
}

func (r *Remediator) executeCommand(ctx context.Context, cmd string) (*utils.ExecutionResult, error) {
	log.Printf("Remediator: Executing command - %s", cmd)
	result, err := r.executor.Execute(ctx, cmd)
	if err != nil {
		return result, fmt.Errorf("execute %q: %w stderr=%s", cmd, err, result.Stderr)
	}
	return result, nil
}

func (r *Remediator) handleApprovals(ctx context.Context) {
	reader := bufio.NewReader(os.Stdin)
	for {
		select {
		case <-ctx.Done():
			return
		case req := <-r.approvalCh:
			if req == nil {
				continue
			}
			if !r.cfg.Security.EnableApproval {
				req.ResponseCh <- true
				continue
			}
			if !isInteractiveTerminal() {
				log.Printf("Remediator: Approval denied for %q because no interactive terminal is available", req.Command)
				req.ResponseCh <- false
				continue
			}

			log.Printf("\n=== APPROVAL REQUIRED ===")
			log.Printf("Command: %s", req.Command)
			log.Printf("Reason: %s", req.Reason)
			log.Printf("Approve? (y/n): ")
			answer, err := reader.ReadString('\n')
			if err != nil {
				req.ResponseCh <- false
				continue
			}
			req.ResponseCh <- strings.EqualFold(strings.TrimSpace(answer), "y")
		}
	}
}

func (r *Remediator) defaultServiceRecoveryCommands(service string) []string {
	if runtime.GOOS == "linux" {
		return []string{
			fmt.Sprintf("journalctl -u %s -n 100 --no-pager", service),
			fmt.Sprintf("systemctl restart %s", service),
			fmt.Sprintf("systemctl is-active %s", service),
		}
	}

	return nil
}

func isInteractiveTerminal() bool {
	info, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (info.Mode() & os.ModeCharDevice) != 0
}
