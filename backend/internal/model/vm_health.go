package model

import (
	"time"

	"github.com/google/uuid"
)

// HealthScore represents a VM's overall health score (0-100).
type HealthScore struct {
	NodeID    uuid.UUID        `json:"node_id"`
	VMID      int              `json:"vmid"`
	VMName    string           `json:"vm_name"`
	VMType    string           `json:"vm_type"`
	Score     int              `json:"score"`
	Status    string           `json:"status"` // healthy, warning, critical
	Breakdown HealthBreakdown  `json:"breakdown"`
	UpdatedAt time.Time        `json:"updated_at"`
}

// HealthBreakdown shows the individual components of a health score.
type HealthBreakdown struct {
	CPUScore       int     `json:"cpu_score"`
	CPUAvg         float64 `json:"cpu_avg"`
	RAMScore       int     `json:"ram_score"`
	RAMAvg         float64 `json:"ram_avg"`
	DiskScore      int     `json:"disk_score"`
	DiskUsage      float64 `json:"disk_usage"`
	StabilityScore int     `json:"stability_score"`
	UptimeDays     float64 `json:"uptime_days"`
	CrashCount     int     `json:"crash_count"`
}

// RightsizingRecommendation is a per-VM recommendation from the VM-level
// rightsizing analyzer (different from the node-level ResourceRecommendation).
type VMRightsizingRecommendation struct {
	NodeID       uuid.UUID                  `json:"node_id"`
	VMID         int                        `json:"vmid"`
	VMName       string                     `json:"vm_name"`
	VMType       string                     `json:"vm_type"`
	Resources    []VMResourceRecommendation `json:"resources"`
	AnalyzedAt   time.Time                  `json:"analyzed_at"`
}

type VMResourceRecommendation struct {
	Resource         string  `json:"resource"` // cpu, memory, disk
	CurrentValue     string  `json:"current_value"`
	RecommendedValue string  `json:"recommended_value"`
	AvgUsage         float64 `json:"avg_usage"`
	MaxUsage         float64 `json:"max_usage"`
	Status           string  `json:"status"` // optimal, reduce, increase
	Reason           string  `json:"reason"`
}

// VMAnomaly represents a detected anomaly for a specific VM.
type VMAnomaly struct {
	NodeID     uuid.UUID `json:"node_id"`
	VMID       int       `json:"vmid"`
	VMName     string    `json:"vm_name"`
	Metric     string    `json:"metric"`
	Value      float64   `json:"value"`
	Mean       float64   `json:"mean"`
	StdDev     float64   `json:"stddev"`
	ZScore     float64   `json:"z_score"`
	Severity   string    `json:"severity"` // warning, critical
	Message    string    `json:"message"`
	DetectedAt time.Time `json:"detected_at"`
}

// SnapshotPolicy defines an automatic snapshot rotation policy for a VM.
type SnapshotPolicy struct {
	ID           uuid.UUID  `json:"id"`
	NodeID       uuid.UUID  `json:"node_id"`
	VMID         int        `json:"vmid"`
	VMType       string     `json:"vm_type"`
	Name         string     `json:"name"`
	KeepDaily    int        `json:"keep_daily"`
	KeepWeekly   int        `json:"keep_weekly"`
	KeepMonthly  int        `json:"keep_monthly"`
	ScheduleCron string     `json:"schedule_cron"`
	IsActive     bool       `json:"is_active"`
	LastRun      *time.Time `json:"last_run,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
}

type CreateSnapshotPolicyRequest struct {
	NodeID       uuid.UUID `json:"node_id" validate:"required"`
	VMID         int       `json:"vmid" validate:"required"`
	VMType       string    `json:"vm_type" validate:"required"`
	Name         string    `json:"name" validate:"required"`
	KeepDaily    int       `json:"keep_daily"`
	KeepWeekly   int       `json:"keep_weekly"`
	KeepMonthly  int       `json:"keep_monthly"`
	ScheduleCron string    `json:"schedule_cron"`
	IsActive     *bool     `json:"is_active"`
}

type UpdateSnapshotPolicyRequest struct {
	Name         *string `json:"name"`
	KeepDaily    *int    `json:"keep_daily"`
	KeepWeekly   *int    `json:"keep_weekly"`
	KeepMonthly  *int    `json:"keep_monthly"`
	ScheduleCron *string `json:"schedule_cron"`
	IsActive     *bool   `json:"is_active"`
}

// ScheduledAction defines a cron-based action for a VM.
type ScheduledAction struct {
	ID           uuid.UUID `json:"id"`
	NodeID       uuid.UUID `json:"node_id"`
	VMID         *int      `json:"vmid,omitempty"`
	VMType       string    `json:"vm_type,omitempty"`
	Action       string    `json:"action"`
	ScheduleCron string    `json:"schedule_cron"`
	IsActive     bool      `json:"is_active"`
	Description  string    `json:"description,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

type CreateScheduledActionRequest struct {
	NodeID       uuid.UUID `json:"node_id" validate:"required"`
	VMID         *int      `json:"vmid"`
	VMType       string    `json:"vm_type"`
	Action       string    `json:"action" validate:"required"`
	ScheduleCron string    `json:"schedule_cron" validate:"required"`
	IsActive     *bool     `json:"is_active"`
	Description  string    `json:"description"`
}

// VMDependency represents a dependency between two VMs.
type VMDependency struct {
	ID             uuid.UUID `json:"id"`
	SourceNodeID   uuid.UUID `json:"source_node_id"`
	SourceVMID     int       `json:"source_vmid"`
	TargetNodeID   uuid.UUID `json:"target_node_id"`
	TargetVMID     int       `json:"target_vmid"`
	DependencyType string    `json:"dependency_type"`
	Description    string    `json:"description,omitempty"`
	CreatedAt      time.Time `json:"created_at"`

	// Enriched fields (not stored in DB)
	SourceVMName string `json:"source_vm_name,omitempty"`
	TargetVMName string `json:"target_vm_name,omitempty"`
	SourceVMType string `json:"source_vm_type,omitempty"`
	TargetVMType string `json:"target_vm_type,omitempty"`
	SourceStatus string `json:"source_status,omitempty"`
	TargetStatus string `json:"target_status,omitempty"`
}

type CreateVMDependencyRequest struct {
	SourceNodeID   uuid.UUID `json:"source_node_id" validate:"required"`
	SourceVMID     int       `json:"source_vmid" validate:"required"`
	TargetNodeID   uuid.UUID `json:"target_node_id" validate:"required"`
	TargetVMID     int       `json:"target_vmid" validate:"required"`
	DependencyType string    `json:"dependency_type"`
	Description    string    `json:"description"`
}
