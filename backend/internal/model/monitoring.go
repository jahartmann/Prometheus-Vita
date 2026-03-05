package model

import (
	"time"

	"github.com/google/uuid"
)

type MetricsRecord struct {
	ID         int64     `json:"id"`
	NodeID     uuid.UUID `json:"node_id"`
	RecordedAt time.Time `json:"recorded_at"`
	CPUUsage   float64   `json:"cpu_usage"`
	MemUsed    int64     `json:"memory_used"`
	MemTotal   int64     `json:"memory_total"`
	DiskUsed   int64     `json:"disk_used"`
	DiskTotal  int64     `json:"disk_total"`
	NetIn      int64     `json:"net_in"`
	NetOut     int64     `json:"net_out"`
	LoadAvg    []float64 `json:"load_avg"`
}

type MetricsSummary struct {
	NodeID      uuid.UUID `json:"node_id"`
	Period      string    `json:"period"`
	CPUAvg      float64   `json:"cpu_avg"`
	CPUMax      float64   `json:"cpu_max"`
	CPUMin      float64   `json:"cpu_min"`
	CPUCurrent  float64   `json:"cpu_current"`
	MemAvg      float64   `json:"memory_avg_percent"`
	MemMax      float64   `json:"memory_max_percent"`
	MemMin      float64   `json:"memory_min_percent"`
	MemCurrent  float64   `json:"memory_current_percent"`
	DiskAvg     float64   `json:"disk_avg_percent"`
	DiskMax     float64   `json:"disk_max_percent"`
	DiskMin     float64   `json:"disk_min_percent"`
	DiskCurrent float64   `json:"disk_current_percent"`
}

type NetworkAlias struct {
	ID            uuid.UUID `json:"id"`
	NodeID        uuid.UUID `json:"node_id"`
	InterfaceName string    `json:"interface_name"`
	DisplayName   string    `json:"display_name"`
	Description   string    `json:"description,omitempty"`
	Color         string    `json:"color,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type UpsertAliasRequest struct {
	DisplayName string `json:"display_name"`
	Description string `json:"description,omitempty"`
	Color       string `json:"color,omitempty"`
}

type Tag struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Color     string    `json:"color"`
	Category  string    `json:"category,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type CreateTagRequest struct {
	Name     string `json:"name" validate:"required"`
	Color    string `json:"color,omitempty"`
	Category string `json:"category,omitempty"`
}

type NodeTag struct {
	NodeID uuid.UUID `json:"node_id"`
	TagID  uuid.UUID `json:"tag_id"`
}

type AssignTagRequest struct {
	TagID string `json:"tag_id" validate:"required"`
}

type VMTag struct {
	NodeID    uuid.UUID `json:"node_id"`
	VMID      int       `json:"vmid"`
	VMType    string    `json:"vm_type"`
	TagID     uuid.UUID `json:"tag_id"`
	CreatedAt time.Time `json:"created_at"`
}

type BulkVMTarget struct {
	NodeID string `json:"node_id"`
	VMID   int    `json:"vmid"`
	VMType string `json:"vm_type,omitempty"`
}

type BulkTagRequest struct {
	Targets []BulkVMTarget `json:"targets"`
}

// VMMetricsRecord stores per-VM metrics with network rates (bytes/sec).
type VMMetricsRecord struct {
	ID         string    `json:"id"`
	NodeID     uuid.UUID `json:"node_id"`
	VMID       int       `json:"vmid"`
	VMType     string    `json:"vm_type"`
	CPUUsage   float64   `json:"cpu_usage"`
	MemUsed    int64     `json:"memory_used"`
	MemTotal   int64     `json:"memory_total"`
	NetIn      int64     `json:"net_in"`
	NetOut     int64     `json:"net_out"`
	DiskRead   int64     `json:"disk_read"`
	DiskWrite  int64     `json:"disk_write"`
	RecordedAt time.Time `json:"recorded_at"`
}

// NetworkSummary provides aggregated network metrics for a period.
type NetworkSummary struct {
	TotalIn     int64 `json:"total_in"`
	TotalOut    int64 `json:"total_out"`
	AvgInRate   int64 `json:"avg_in_rate"`
	AvgOutRate  int64 `json:"avg_out_rate"`
	PeakInRate  int64 `json:"peak_in_rate"`
	PeakOutRate int64 `json:"peak_out_rate"`
}
