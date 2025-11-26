package cmdb

import (
	"database/sql"
	"time"
)

// ConfigurationItem represents a Configuration Item in the CMDB
// Based on ITIL standards
type ConfigurationItem struct {
	CIID         int64
	CIName       string
	CIType       string // Server, Workstation, Network Device, Application, Service, etc.
	CIClass      string // Physical, Logical, Service, etc.
	CIStatus     string // Active, Inactive, Retired, Under Change, etc.
	Environment  string // Production, Development, Test, etc.
	Owner        string
	Description  string
	SerialNumber string
	AssetTag     string
	Location     string
	Criticality  string // Critical, High, Medium, Low
	CreatedAt    time.Time
	UpdatedAt    time.Time
	CreatedBy    string
	UpdatedBy    string
}

// CIAttribute represents a key-value attribute for a CI
type CIAttribute struct {
	AttrID    int64
	CIID      int64
	AttrKey   string
	AttrValue string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// CIRelationship represents a relationship between two CIs
type CIRelationship struct {
	RelID            int64
	SourceCIID       int64
	TargetCIID       int64
	RelationshipType string // DependsOn, RunsOn, ConnectsTo, Contains, etc.
	Description      string
	CreatedAt        time.Time
	CreatedBy        string
}

// CIChangeHistory represents a change to a CI
type CIChangeHistory struct {
	ChangeID     int64
	CIID         int64
	ChangeType   string // Created, Updated, Deleted, Status Change, etc.
	FieldName    string
	OldValue     string
	NewValue     string
	ChangedBy    string
	ChangeReason string
	ChangedAt    time.Time
}

// ExecutionHistory represents a Tacoscript execution
type ExecutionHistory struct {
	ExecID          int64
	CIID            *int64 // Nullable - execution might not be tied to a specific CI
	ScriptPath      string
	ExecutionStatus string // Running, Success, Failed, Partial
	StartTime       time.Time
	EndTime         *time.Time
	DurationMS      *int64
	TaskCount       int
	SuccessCount    int
	FailedCount     int
	SkippedCount    int
	ErrorMessage    string
	ExecutedBy      string
}

// TaskExecution represents an individual task execution
type TaskExecution struct {
	TaskExecID   int64
	ExecID       int64
	TaskName     string
	TaskType     string
	TaskStatus   string // Success, Failed, Skipped
	StartTime    *time.Time
	EndTime      *time.Time
	DurationMS   *int64
	Changes      string // JSON string of changes
	Comment      string
	ErrorMessage string
}

// InstalledPackage represents a software package installed on a CI
type InstalledPackage struct {
	PkgID          int64
	CIID           int64
	PackageName    string
	PackageVersion string
	PackageManager string // winget, choco, apt, yum, brew, etc.
	InstallDate    time.Time
	LastUpdated    time.Time
}

// SystemInfo represents system/hardware information for a CI
type SystemInfo struct {
	InfoID        int64
	CIID          int64
	Hostname      string
	OSName        string
	OSVersion     string
	OSPlatform    string
	OSFamily      string
	Architecture  string
	CPUCores      int
	MemoryTotalMB int
	DiskTotalGB   int
	LastDiscovered time.Time
}

// CIFilter provides filtering options for querying CIs
type CIFilter struct {
	CIType      string
	CIStatus    string
	Environment string
	Owner       string
	Search      string // Search in name and description
	Limit       int
	Offset      int
}

// ExecutionFilter provides filtering options for querying executions
type ExecutionFilter struct {
	CIID        *int64
	Status      string
	StartDate   *time.Time
	EndDate     *time.Time
	ExecutedBy  string
	Limit       int
	Offset      int
}

// Scan methods to convert database rows to structs

func (ci *ConfigurationItem) ScanRow(row *sql.Row) error {
	return row.Scan(
		&ci.CIID, &ci.CIName, &ci.CIType, &ci.CIClass, &ci.CIStatus,
		&ci.Environment, &ci.Owner, &ci.Description, &ci.SerialNumber,
		&ci.AssetTag, &ci.Location, &ci.Criticality, &ci.CreatedAt,
		&ci.UpdatedAt, &ci.CreatedBy, &ci.UpdatedBy,
	)
}

func (ci *ConfigurationItem) ScanRows(rows *sql.Rows) error {
	return rows.Scan(
		&ci.CIID, &ci.CIName, &ci.CIType, &ci.CIClass, &ci.CIStatus,
		&ci.Environment, &ci.Owner, &ci.Description, &ci.SerialNumber,
		&ci.AssetTag, &ci.Location, &ci.Criticality, &ci.CreatedAt,
		&ci.UpdatedAt, &ci.CreatedBy, &ci.UpdatedBy,
	)
}
