package cmdb

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
	_ "modernc.org/sqlite" // SQLite driver
)

// CMDB represents the Configuration Management Database
// Based on ITIL/ITSM standards for tracking Configuration Items (CIs)
type CMDB struct {
	db       *sql.DB
	dbPath   string
	readOnly bool
	logger   *logrus.Logger
}

// Config holds CMDB configuration
type Config struct {
	DBPath   string
	ReadOnly bool
}

// NewCMDB creates a new CMDB instance
func NewCMDB(config Config) (*CMDB, error) {
	if config.DBPath == "" {
		// Default to user's config directory
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get user home directory: %w", err)
		}
		config.DBPath = filepath.Join(homeDir, ".tacoscript", "cmdb.db")
	}

	// Ensure directory exists
	dbDir := filepath.Dir(config.DBPath)
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create CMDB directory: %w", err)
	}

	// Open database connection
	dsn := config.DBPath
	if config.ReadOnly {
		dsn += "?mode=ro"
	}

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open CMDB database: %w", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping CMDB database: %w", err)
	}

	logger := logrus.New()
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})

	cmdb := &CMDB{
		db:       db,
		dbPath:   config.DBPath,
		readOnly: config.ReadOnly,
		logger:   logger,
	}

	// Initialize schema if not read-only
	if !config.ReadOnly {
		if err := cmdb.initializeSchema(); err != nil {
			db.Close()
			return nil, fmt.Errorf("failed to initialize CMDB schema: %w", err)
		}
	}

	logrus.Infof("CMDB initialized at %s", config.DBPath)
	return cmdb, nil
}

// Close closes the database connection
func (c *CMDB) Close() error {
	if c.db != nil {
		return c.db.Close()
	}
	return nil
}

// initializeSchema creates all necessary tables following ITIL CMDB structure
func (c *CMDB) initializeSchema() error {
	schema := `
	-- Configuration Items (CIs) - Core ITIL entity
	CREATE TABLE IF NOT EXISTS configuration_items (
		ci_id INTEGER PRIMARY KEY AUTOINCREMENT,
		ci_name TEXT NOT NULL UNIQUE,
		ci_type TEXT NOT NULL,
		ci_class TEXT,
		ci_status TEXT NOT NULL DEFAULT 'Active',
		environment TEXT,
		owner TEXT,
		description TEXT,
		serial_number TEXT,
		asset_tag TEXT,
		location TEXT,
		criticality TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		created_by TEXT,
		updated_by TEXT
	);

	CREATE INDEX IF NOT EXISTS idx_ci_type ON configuration_items(ci_type);
	CREATE INDEX IF NOT EXISTS idx_ci_status ON configuration_items(ci_status);
	CREATE INDEX IF NOT EXISTS idx_ci_name ON configuration_items(ci_name);

	-- CI Attributes - Flexible key-value storage for CI properties
	CREATE TABLE IF NOT EXISTS ci_attributes (
		attr_id INTEGER PRIMARY KEY AUTOINCREMENT,
		ci_id INTEGER NOT NULL,
		attr_key TEXT NOT NULL,
		attr_value TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (ci_id) REFERENCES configuration_items(ci_id) ON DELETE CASCADE,
		UNIQUE(ci_id, attr_key)
	);

	CREATE INDEX IF NOT EXISTS idx_ci_attr_key ON ci_attributes(attr_key);

	-- CI Relationships - Tracks dependencies and relationships between CIs
	CREATE TABLE IF NOT EXISTS ci_relationships (
		rel_id INTEGER PRIMARY KEY AUTOINCREMENT,
		source_ci_id INTEGER NOT NULL,
		target_ci_id INTEGER NOT NULL,
		relationship_type TEXT NOT NULL,
		description TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		created_by TEXT,
		FOREIGN KEY (source_ci_id) REFERENCES configuration_items(ci_id) ON DELETE CASCADE,
		FOREIGN KEY (target_ci_id) REFERENCES configuration_items(ci_id) ON DELETE CASCADE
	);

	CREATE INDEX IF NOT EXISTS idx_rel_source ON ci_relationships(source_ci_id);
	CREATE INDEX IF NOT EXISTS idx_rel_target ON ci_relationships(target_ci_id);

	-- Change History - Audit trail for all CI changes
	CREATE TABLE IF NOT EXISTS ci_change_history (
		change_id INTEGER PRIMARY KEY AUTOINCREMENT,
		ci_id INTEGER NOT NULL,
		change_type TEXT NOT NULL,
		field_name TEXT,
		old_value TEXT,
		new_value TEXT,
		changed_by TEXT,
		change_reason TEXT,
		changed_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (ci_id) REFERENCES configuration_items(ci_id) ON DELETE CASCADE
	);

	CREATE INDEX IF NOT EXISTS idx_change_ci ON ci_change_history(ci_id);
	CREATE INDEX IF NOT EXISTS idx_change_date ON ci_change_history(changed_at);

	-- Execution History - Tracks Tacoscript executions on CIs
	CREATE TABLE IF NOT EXISTS execution_history (
		exec_id INTEGER PRIMARY KEY AUTOINCREMENT,
		ci_id INTEGER,
		script_path TEXT NOT NULL,
		execution_status TEXT NOT NULL,
		start_time DATETIME DEFAULT CURRENT_TIMESTAMP,
		end_time DATETIME,
		duration_ms INTEGER,
		task_count INTEGER DEFAULT 0,
		success_count INTEGER DEFAULT 0,
		failed_count INTEGER DEFAULT 0,
		skipped_count INTEGER DEFAULT 0,
		error_message TEXT,
		executed_by TEXT,
		FOREIGN KEY (ci_id) REFERENCES configuration_items(ci_id) ON DELETE SET NULL
	);

	CREATE INDEX IF NOT EXISTS idx_exec_ci ON execution_history(ci_id);
	CREATE INDEX IF NOT EXISTS idx_exec_status ON execution_history(execution_status);
	CREATE INDEX IF NOT EXISTS idx_exec_time ON execution_history(start_time);

	-- Task Execution Details - Details of individual task executions
	CREATE TABLE IF NOT EXISTS task_executions (
		task_exec_id INTEGER PRIMARY KEY AUTOINCREMENT,
		exec_id INTEGER NOT NULL,
		task_name TEXT NOT NULL,
		task_type TEXT NOT NULL,
		task_status TEXT NOT NULL,
		start_time DATETIME,
		end_time DATETIME,
		duration_ms INTEGER,
		changes TEXT,
		comment TEXT,
		error_message TEXT,
		FOREIGN KEY (exec_id) REFERENCES execution_history(exec_id) ON DELETE CASCADE
	);

	CREATE INDEX IF NOT EXISTS idx_task_exec ON task_executions(exec_id);
	CREATE INDEX IF NOT EXISTS idx_task_type ON task_executions(task_type);

	-- Installed Packages - Track software/packages on CIs
	CREATE TABLE IF NOT EXISTS installed_packages (
		pkg_id INTEGER PRIMARY KEY AUTOINCREMENT,
		ci_id INTEGER NOT NULL,
		package_name TEXT NOT NULL,
		package_version TEXT,
		package_manager TEXT,
		install_date DATETIME DEFAULT CURRENT_TIMESTAMP,
		last_updated DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (ci_id) REFERENCES configuration_items(ci_id) ON DELETE CASCADE,
		UNIQUE(ci_id, package_name)
	);

	CREATE INDEX IF NOT EXISTS idx_pkg_ci ON installed_packages(ci_id);
	CREATE INDEX IF NOT EXISTS idx_pkg_name ON installed_packages(package_name);

	-- System Information - Hardware and OS details
	CREATE TABLE IF NOT EXISTS system_info (
		info_id INTEGER PRIMARY KEY AUTOINCREMENT,
		ci_id INTEGER NOT NULL UNIQUE,
		hostname TEXT,
		os_name TEXT,
		os_version TEXT,
		os_platform TEXT,
		os_family TEXT,
		architecture TEXT,
		cpu_cores INTEGER,
		memory_total_mb INTEGER,
		disk_total_gb INTEGER,
		last_discovered DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (ci_id) REFERENCES configuration_items(ci_id) ON DELETE CASCADE
	);

	CREATE INDEX IF NOT EXISTS idx_sysinfo_ci ON system_info(ci_id);

	-- Version tracking
	CREATE TABLE IF NOT EXISTS schema_version (
		version INTEGER PRIMARY KEY,
		applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	INSERT OR IGNORE INTO schema_version (version) VALUES (1);
	`

	_, err := c.db.Exec(schema)
	if err != nil {
		return fmt.Errorf("failed to execute schema: %w", err)
	}

	return nil
}

// GetDBPath returns the database file path
func (c *CMDB) GetDBPath() string {
	return c.dbPath
}

// IsReadOnly returns whether the CMDB is in read-only mode
func (c *CMDB) IsReadOnly() bool {
	return c.readOnly
}

// BeginTransaction starts a new database transaction
func (c *CMDB) BeginTransaction() (*sql.Tx, error) {
	if c.readOnly {
		return nil, fmt.Errorf("cannot begin transaction in read-only mode")
	}
	return c.db.Begin()
}

// GetStats returns basic statistics about the CMDB
func (c *CMDB) GetStats() (map[string]int, error) {
	stats := make(map[string]int)

	queries := map[string]string{
		"total_cis":        "SELECT COUNT(*) FROM configuration_items",
		"active_cis":       "SELECT COUNT(*) FROM configuration_items WHERE ci_status = 'Active'",
		"total_executions": "SELECT COUNT(*) FROM execution_history",
		"total_packages":   "SELECT COUNT(*) FROM installed_packages",
		"total_changes":    "SELECT COUNT(*) FROM ci_change_history",
	}

	for key, query := range queries {
		var count int
		err := c.db.QueryRow(query).Scan(&count)
		if err != nil {
			return nil, fmt.Errorf("failed to get %s: %w", key, err)
		}
		stats[key] = count
	}

	return stats, nil
}
