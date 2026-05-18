package agent

import (
	"context"
	"encoding/json"

	"github.com/antigravity/prometheus/internal/model"
)

type ToolRisk string

const (
	ToolRiskLow      ToolRisk = "low"
	ToolRiskMedium   ToolRisk = "medium"
	ToolRiskHigh     ToolRisk = "high"
	ToolRiskCritical ToolRisk = "critical"
)

type ToolSecurity struct {
	Risk           ToolRisk         `json:"risk"`
	Permission     model.Permission `json:"permission"`
	Action         string           `json:"action"`
	RequiresDryRun bool             `json:"requires_dry_run"`
}

type SecureTool interface {
	Security() ToolSecurity
}

type DryRunTool interface {
	Preview(ctx context.Context, args json.RawMessage) (json.RawMessage, error)
}

func securityForTool(tool Tool) ToolSecurity {
	if secure, ok := tool.(SecureTool); ok {
		return secure.Security()
	}
	if security, ok := builtInToolSecurity[tool.Name()]; ok {
		return security
	}
	if tool.ReadOnly() {
		return ToolSecurity{
			Risk:       ToolRiskLow,
			Permission: model.PermissionAgentUse,
			Action:     "read",
		}
	}
	return ToolSecurity{
		Risk:           ToolRiskHigh,
		Permission:     model.PermissionAgentExecute,
		Action:         "execute",
		RequiresDryRun: true,
	}
}

func toolSupportsDryRun(tool Tool) bool {
	_, ok := tool.(DryRunTool)
	return ok
}

// argsAreRealRun reports true when the security policy requires a dry-run
// for the tool AND the LLM-provided args do NOT explicitly opt into dry-run
// (i.e. `dry_run` is missing or false). When this is the case the call is
// destructive and must always be routed through the approval workflow, even
// for users with AutonomyFullAuto. This closes the bypass where an LLM
// hallucinates `dry_run: false` and skips approval that was meant to guard
// the destructive code path.
func argsAreRealRun(security ToolSecurity, args json.RawMessage) bool {
	if !security.RequiresDryRun {
		return false
	}
	if len(args) == 0 {
		// No args at all → defaults apply; we treat that as real-run because
		// the tool isn't being explicitly told to preview.
		return true
	}
	var probe struct {
		DryRun *bool `json:"dry_run"`
	}
	if err := json.Unmarshal(args, &probe); err != nil {
		// Malformed args → fall through to caller's normal handling; don't
		// pretend it's a dry-run.
		return true
	}
	if probe.DryRun == nil {
		return true
	}
	return !*probe.DryRun
}

var builtInToolSecurity = map[string]ToolSecurity{
	"list_nodes":       {Risk: ToolRiskLow, Permission: model.PermissionNodesRead, Action: "read"},
	"node_status":      {Risk: ToolRiskLow, Permission: model.PermissionNodesRead, Action: "read"},
	"get_vms":          {Risk: ToolRiskLow, Permission: model.PermissionVMsRead, Action: "read"},
	"get_metrics":      {Risk: ToolRiskLow, Permission: model.PermissionNodesRead, Action: "read"},
	"get_storage":      {Risk: ToolRiskLow, Permission: model.PermissionNodesRead, Action: "read"},
	"get_network":      {Risk: ToolRiskLow, Permission: model.PermissionNodesRead, Action: "read"},
	"get_anomalies":    {Risk: ToolRiskLow, Permission: model.PermissionSecurityRead, Action: "read"},
	"get_predictions":  {Risk: ToolRiskLow, Permission: model.PermissionNodesRead, Action: "read"},
	"get_briefing":     {Risk: ToolRiskLow, Permission: model.PermissionNodesRead, Action: "read"},
	"check_drift":      {Risk: ToolRiskLow, Permission: model.PermissionNodesRead, Action: "read"},
	"check_updates":    {Risk: ToolRiskMedium, Permission: model.PermissionNodesRead, Action: "read"},
	"rightsizing":      {Risk: ToolRiskLow, Permission: model.PermissionVMsRead, Action: "read"},
	"recall_knowledge": {Risk: ToolRiskLow, Permission: model.PermissionAgentUse, Action: "read"},
	"save_knowledge":   {Risk: ToolRiskMedium, Permission: model.PermissionAgentManage, Action: "write"},

	"create_backup":   {Risk: ToolRiskMedium, Permission: model.PermissionBackupsCreate, Action: "backup"},
	"start_vm":        {Risk: ToolRiskHigh, Permission: model.PermissionVMPower, Action: "power"},
	"stop_vm":         {Risk: ToolRiskHigh, Permission: model.PermissionVMPower, Action: "power"},
	"migrate_vm":      {Risk: ToolRiskCritical, Permission: model.PermissionVMsWrite, Action: "migration", RequiresDryRun: true},
	"restore_config":  {Risk: ToolRiskCritical, Permission: model.PermissionBackupsRestore, Action: "restore", RequiresDryRun: true},
	"run_ssh_command": {Risk: ToolRiskCritical, Permission: model.PermissionNodesWrite, Action: "ssh", RequiresDryRun: true},
	"vm_exec":         {Risk: ToolRiskCritical, Permission: model.PermissionVMsWrite, Action: "vm_exec", RequiresDryRun: true},
	"vm_file_write":   {Risk: ToolRiskCritical, Permission: model.PermissionVMsWrite, Action: "file_write", RequiresDryRun: true},

	"vm_processes":      {Risk: ToolRiskLow, Permission: model.PermissionVMsRead, Action: "read"},
	"vm_services":       {Risk: ToolRiskLow, Permission: model.PermissionVMsRead, Action: "read"},
	"vm_disk_usage":     {Risk: ToolRiskLow, Permission: model.PermissionVMsRead, Action: "read"},
	"vm_network_info":   {Risk: ToolRiskLow, Permission: model.PermissionVMsRead, Action: "read"},
	"vm_service_action": {Risk: ToolRiskHigh, Permission: model.PermissionVMsWrite, Action: "service"},
}
