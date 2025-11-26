package cmdb

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// CreateExecution creates a new execution record
func (c *CMDB) CreateExecution(exec *ExecutionHistory) error {
	exec.StartTime = time.Now()
	exec.ExecutionStatus = "Running"

	query := `
		INSERT INTO execution_history (
			ci_id, script_path, execution_status, start_time,
			task_count, success_count, failed_count, skipped_count, executed_by
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	result, err := c.db.Exec(query,
		exec.CIID, exec.ScriptPath, exec.ExecutionStatus, exec.StartTime,
		exec.TaskCount, exec.SuccessCount, exec.FailedCount, exec.SkippedCount, exec.ExecutedBy,
	)
	if err != nil {
		return fmt.Errorf("failed to create execution: %w", err)
	}

	execID, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get execution ID: %w", err)
	}
	exec.ExecID = execID

	c.logger.Infof("Created execution record: %d", exec.ExecID)
	return nil
}

// UpdateExecution updates an execution record
func (c *CMDB) UpdateExecution(exec *ExecutionHistory) error {
	endTime := time.Now()
	exec.EndTime = &endTime

	if exec.StartTime.IsZero() {
		exec.StartTime = time.Now()
	}

	duration := endTime.Sub(exec.StartTime).Milliseconds()
	exec.DurationMS = &duration

	query := `
		UPDATE execution_history
		SET execution_status = ?, end_time = ?, duration_ms = ?,
		    task_count = ?, success_count = ?, failed_count = ?, skipped_count = ?,
		    error_message = ?
		WHERE exec_id = ?
	`

	_, err := c.db.Exec(query,
		exec.ExecutionStatus, exec.EndTime, exec.DurationMS,
		exec.TaskCount, exec.SuccessCount, exec.FailedCount, exec.SkippedCount,
		exec.ErrorMessage, exec.ExecID,
	)
	if err != nil {
		return fmt.Errorf("failed to update execution: %w", err)
	}

	c.logger.Infof("Updated execution record: %d (status: %s)", exec.ExecID, exec.ExecutionStatus)
	return nil
}

// GetExecution retrieves an execution by ID
func (c *CMDB) GetExecution(execID int64) (*ExecutionHistory, error) {
	query := `
		SELECT exec_id, ci_id, script_path, execution_status, start_time, end_time,
		       duration_ms, task_count, success_count, failed_count, skipped_count,
		       error_message, executed_by
		FROM execution_history
		WHERE exec_id = ?
	`

	exec := &ExecutionHistory{}
	err := c.db.QueryRow(query, execID).Scan(
		&exec.ExecID, &exec.CIID, &exec.ScriptPath, &exec.ExecutionStatus,
		&exec.StartTime, &exec.EndTime, &exec.DurationMS,
		&exec.TaskCount, &exec.SuccessCount, &exec.FailedCount, &exec.SkippedCount,
		&exec.ErrorMessage, &exec.ExecutedBy,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("execution not found: %d", execID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get execution: %w", err)
	}

	return exec, nil
}

// ListExecutions retrieves executions with optional filtering
func (c *CMDB) ListExecutions(filter *ExecutionFilter) ([]*ExecutionHistory, error) {
	query := `
		SELECT exec_id, ci_id, script_path, execution_status, start_time, end_time,
		       duration_ms, task_count, success_count, failed_count, skipped_count,
		       error_message, executed_by
		FROM execution_history
		WHERE 1=1
	`
	args := []interface{}{}

	if filter != nil {
		if filter.CIID != nil {
			query += " AND ci_id = ?"
			args = append(args, *filter.CIID)
		}
		if filter.Status != "" {
			query += " AND execution_status = ?"
			args = append(args, filter.Status)
		}
		if filter.StartDate != nil {
			query += " AND start_time >= ?"
			args = append(args, *filter.StartDate)
		}
		if filter.EndDate != nil {
			query += " AND start_time <= ?"
			args = append(args, *filter.EndDate)
		}
		if filter.ExecutedBy != "" {
			query += " AND executed_by = ?"
			args = append(args, filter.ExecutedBy)
		}

		query += " ORDER BY start_time DESC"

		if filter.Limit > 0 {
			query += " LIMIT ?"
			args = append(args, filter.Limit)
		}
		if filter.Offset > 0 {
			query += " OFFSET ?"
			args = append(args, filter.Offset)
		}
	} else {
		query += " ORDER BY start_time DESC LIMIT 100"
	}

	rows, err := c.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list executions: %w", err)
	}
	defer rows.Close()

	var execs []*ExecutionHistory
	for rows.Next() {
		exec := &ExecutionHistory{}
		if err := rows.Scan(
			&exec.ExecID, &exec.CIID, &exec.ScriptPath, &exec.ExecutionStatus,
			&exec.StartTime, &exec.EndTime, &exec.DurationMS,
			&exec.TaskCount, &exec.SuccessCount, &exec.FailedCount, &exec.SkippedCount,
			&exec.ErrorMessage, &exec.ExecutedBy,
		); err != nil {
			return nil, fmt.Errorf("failed to scan execution: %w", err)
		}
		execs = append(execs, exec)
	}

	return execs, nil
}

// CreateTaskExecution creates a task execution record
func (c *CMDB) CreateTaskExecution(task *TaskExecution) error {
	query := `
		INSERT INTO task_executions (
			exec_id, task_name, task_type, task_status,
			start_time, end_time, duration_ms, changes, comment, error_message
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	result, err := c.db.Exec(query,
		task.ExecID, task.TaskName, task.TaskType, task.TaskStatus,
		task.StartTime, task.EndTime, task.DurationMS,
		task.Changes, task.Comment, task.ErrorMessage,
	)
	if err != nil {
		return fmt.Errorf("failed to create task execution: %w", err)
	}

	taskExecID, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get task execution ID: %w", err)
	}
	task.TaskExecID = taskExecID

	return nil
}

// GetTaskExecutions retrieves all task executions for an execution
func (c *CMDB) GetTaskExecutions(execID int64) ([]*TaskExecution, error) {
	query := `
		SELECT task_exec_id, exec_id, task_name, task_type, task_status,
		       start_time, end_time, duration_ms, changes, comment, error_message
		FROM task_executions
		WHERE exec_id = ?
		ORDER BY task_exec_id
	`

	rows, err := c.db.Query(query, execID)
	if err != nil {
		return nil, fmt.Errorf("failed to get task executions: %w", err)
	}
	defer rows.Close()

	var tasks []*TaskExecution
	for rows.Next() {
		task := &TaskExecution{}
		if err := rows.Scan(
			&task.TaskExecID, &task.ExecID, &task.TaskName, &task.TaskType, &task.TaskStatus,
			&task.StartTime, &task.EndTime, &task.DurationMS,
			&task.Changes, &task.Comment, &task.ErrorMessage,
		); err != nil {
			return nil, fmt.Errorf("failed to scan task execution: %w", err)
		}
		tasks = append(tasks, task)
	}

	return tasks, nil
}

// RecordPackageInstallation records a package installation
func (c *CMDB) RecordPackageInstallation(pkg *InstalledPackage) error {
	now := time.Now()
	pkg.InstallDate = now
	pkg.LastUpdated = now

	query := `
		INSERT INTO installed_packages (ci_id, package_name, package_version, package_manager, install_date, last_updated)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(ci_id, package_name, package_manager) DO UPDATE SET
			package_version = excluded.package_version,
			last_updated = excluded.last_updated
	`

	_, err := c.db.Exec(query, pkg.CIID, pkg.PackageName, pkg.PackageVersion, pkg.PackageManager, pkg.InstallDate, pkg.LastUpdated)
	if err != nil {
		return fmt.Errorf("failed to record package installation: %w", err)
	}

	c.logger.Debugf("Recorded package: %s (%s) on CI %d", pkg.PackageName, pkg.PackageVersion, pkg.CIID)
	return nil
}

// RemovePackageInstallation removes a package installation record
func (c *CMDB) RemovePackageInstallation(ciID int64, packageName, packageManager string) error {
	query := `
		DELETE FROM installed_packages
		WHERE ci_id = ? AND package_name = ? AND package_manager = ?
	`

	_, err := c.db.Exec(query, ciID, packageName, packageManager)
	if err != nil {
		return fmt.Errorf("failed to remove package: %w", err)
	}

	c.logger.Debugf("Removed package: %s from CI %d", packageName, ciID)
	return nil
}

// GetInstalledPackages retrieves all packages for a CI
func (c *CMDB) GetInstalledPackages(ciID int64) ([]*InstalledPackage, error) {
	query := `
		SELECT pkg_id, ci_id, package_name, package_version, package_manager, install_date, last_updated
		FROM installed_packages
		WHERE ci_id = ?
		ORDER BY package_name
	`

	rows, err := c.db.Query(query, ciID)
	if err != nil {
		return nil, fmt.Errorf("failed to get installed packages: %w", err)
	}
	defer rows.Close()

	var packages []*InstalledPackage
	for rows.Next() {
		pkg := &InstalledPackage{}
		if err := rows.Scan(&pkg.PkgID, &pkg.CIID, &pkg.PackageName, &pkg.PackageVersion, &pkg.PackageManager, &pkg.InstallDate, &pkg.LastUpdated); err != nil {
			return nil, fmt.Errorf("failed to scan package: %w", err)
		}
		packages = append(packages, pkg)
	}

	return packages, nil
}

// UpdateSystemInfo updates system information for a CI
func (c *CMDB) UpdateSystemInfo(info *SystemInfo) error {
	info.LastDiscovered = time.Now()

	query := `
		INSERT INTO system_info (
			ci_id, hostname, os_name, os_version, os_platform, os_family,
			architecture, cpu_cores, memory_total_mb, disk_total_gb, last_discovered
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(ci_id) DO UPDATE SET
			hostname = excluded.hostname,
			os_name = excluded.os_name,
			os_version = excluded.os_version,
			os_platform = excluded.os_platform,
			os_family = excluded.os_family,
			architecture = excluded.architecture,
			cpu_cores = excluded.cpu_cores,
			memory_total_mb = excluded.memory_total_mb,
			disk_total_gb = excluded.disk_total_gb,
			last_discovered = excluded.last_discovered
	`

	_, err := c.db.Exec(query,
		info.CIID, info.Hostname, info.OSName, info.OSVersion, info.OSPlatform, info.OSFamily,
		info.Architecture, info.CPUCores, info.MemoryTotalMB, info.DiskTotalGB, info.LastDiscovered,
	)
	if err != nil {
		return fmt.Errorf("failed to update system info: %w", err)
	}

	c.logger.Debugf("Updated system info for CI %d", info.CIID)
	return nil
}

// GetSystemInfo retrieves system information for a CI
func (c *CMDB) GetSystemInfo(ciID int64) (*SystemInfo, error) {
	query := `
		SELECT info_id, ci_id, hostname, os_name, os_version, os_platform, os_family,
		       architecture, cpu_cores, memory_total_mb, disk_total_gb, last_discovered
		FROM system_info
		WHERE ci_id = ?
	`

	info := &SystemInfo{}
	err := c.db.QueryRow(query, ciID).Scan(
		&info.InfoID, &info.CIID, &info.Hostname, &info.OSName, &info.OSVersion,
		&info.OSPlatform, &info.OSFamily, &info.Architecture, &info.CPUCores,
		&info.MemoryTotalMB, &info.DiskTotalGB, &info.LastDiscovered,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("system info not found for CI: %d", ciID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get system info: %w", err)
	}

	return info, nil
}

// ExecutionSummary provides aggregated execution statistics
type ExecutionSummary struct {
	TotalExecutions int
	SuccessCount    int
	FailedCount     int
	PartialCount    int
	AverageDuration float64
	TotalTasks      int
}

// GetExecutionSummary retrieves aggregated execution statistics
func (c *CMDB) GetExecutionSummary(ciID *int64, days int) (*ExecutionSummary, error) {
	query := `
		SELECT 
			COUNT(*) as total_executions,
			SUM(CASE WHEN execution_status = 'Success' THEN 1 ELSE 0 END) as success_count,
			SUM(CASE WHEN execution_status = 'Failed' THEN 1 ELSE 0 END) as failed_count,
			SUM(CASE WHEN execution_status = 'Partial' THEN 1 ELSE 0 END) as partial_count,
			AVG(duration_ms) as avg_duration,
			SUM(task_count) as total_tasks
		FROM execution_history
		WHERE 1=1
	`
	args := []interface{}{}

	if ciID != nil {
		query += " AND ci_id = ?"
		args = append(args, *ciID)
	}

	if days > 0 {
		query += " AND start_time >= datetime('now', '-' || ? || ' days')"
		args = append(args, days)
	}

	summary := &ExecutionSummary{}
	err := c.db.QueryRow(query, args...).Scan(
		&summary.TotalExecutions,
		&summary.SuccessCount,
		&summary.FailedCount,
		&summary.PartialCount,
		&summary.AverageDuration,
		&summary.TotalTasks,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get execution summary: %w", err)
	}

	return summary, nil
}

// Helper function to marshal task changes to JSON
func MarshalChanges(changes interface{}) (string, error) {
	data, err := json.Marshal(changes)
	if err != nil {
		return "", fmt.Errorf("failed to marshal changes: %w", err)
	}
	return string(data), nil
}

// Helper function to unmarshal task changes from JSON
func UnmarshalChanges(changesJSON string, target interface{}) error {
	if changesJSON == "" {
		return nil
	}
	if err := json.Unmarshal([]byte(changesJSON), target); err != nil {
		return fmt.Errorf("failed to unmarshal changes: %w", err)
	}
	return nil
}
