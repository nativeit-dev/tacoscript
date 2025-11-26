# Configuration Management Database (CMDB)

Tacoscript now includes a comprehensive Configuration Management Database (CMDB) based on ITIL/ITSM standards. The CMDB tracks configuration items (CIs), their relationships, changes, and execution history.

## Overview

The CMDB provides:

- **Configuration Item (CI) Management**: Track servers, workstations, network devices, applications, and services
- **Relationship Tracking**: Define dependencies and relationships between CIs
- **Change Auditing**: Complete audit trail of all CI changes
- **Execution History**: Track all Tacoscript executions and their results
- **Package Inventory**: Maintain a record of installed software packages
- **System Discovery**: Automatic discovery and registration of system information

## Database

The CMDB uses SQLite for storage, providing a lightweight, embeddable database with no external dependencies. The database is stored at:

- **Windows**: `%APPDATA%\tacoscript\tacoscript.cmdb.db`
- **Linux/macOS**: `~/.config/tacoscript/tacoscript.cmdb.db`

## ITIL Compliance

The CMDB follows ITIL (Information Technology Infrastructure Library) standards:

### Configuration Items (CIs)

CIs are uniquely identifiable components that need to be managed. Each CI has:

- **Identification**: Unique ID, name, serial number, asset tag
- **Classification**: Type (Server, Workstation, etc.), Class (Physical, Logical)
- **Status**: Active, Inactive, Retired, Under Change
- **Ownership**: Owner, environment (Production, Dev, Test)
- **Metadata**: Description, location, criticality

### CI Types

- **Server**: Production servers, database servers, web servers
- **Workstation**: Desktop computers, laptops
- **Network Device**: Routers, switches, firewalls
- **Application**: Software applications
- **Service**: IT services
- **Database**: Database instances
- **Storage**: Storage systems

### CI Relationships

Relationships define how CIs interact:

- **DependsOn**: CI A depends on CI B
- **RunsOn**: Application runs on a server
- **ConnectsTo**: Network connectivity
- **Contains**: Physical containment
- **Uses**: Service usage

## CLI Commands

### Initialize CMDB

```bash
taco cmdb init [--db <path>]
```

Creates and initializes the CMDB database.

### Discover System

```bash
taco cmdb discover
```

Discovers the current system and registers it as a CI. Collects:
- Hostname
- Operating system information
- Hardware specifications (CPU, memory, disk)
- Platform details

### List Configuration Items

```bash
taco cmdb list [flags]
```

Lists all CIs with optional filtering.

**Flags:**
- `--type <type>`: Filter by CI type
- `--status <status>`: Filter by status
- `--environment <env>`: Filter by environment
- `--limit <n>`: Limit number of results (default: 100)

**Example:**
```bash
taco cmdb list --type Server --status Active
```

### Show CI Details

```bash
taco cmdb show <ci-name>
```

Displays detailed information about a specific CI including:
- Basic properties
- Attributes
- System information
- Installed packages count

**Example:**
```bash
taco cmdb show web-server-01
```

### View Change History

```bash
taco cmdb history <ci-name> [--limit <n>]
```

Shows the change history for a CI.

**Example:**
```bash
taco cmdb history web-server-01 --limit 50
```

### View Execution History

```bash
taco cmdb executions [ci-name] [--limit <n>]
```

Shows Tacoscript execution history, optionally filtered by CI.

**Example:**
```bash
taco cmdb executions web-server-01
```

### Show Statistics

```bash
taco cmdb stats
```

Displays CMDB statistics including:
- Total and active CIs
- Change records
- Execution counts
- Package inventory
- 30-day execution summary

## Integration with Tacoscript

The CMDB automatically integrates with Tacoscript execution:

### Automatic Discovery

When running scripts, Tacoscript can automatically discover and register systems as CIs.

### Execution Tracking

Every script execution is recorded with:
- Execution status (Running, Success, Failed, Partial)
- Start and end times
- Task counts (success, failed, skipped)
- Error messages

### Task Recording

Individual task executions are tracked:
- Task name and type
- Status and timing
- Changes made
- Error details

### Package Management Integration

Package operations are automatically recorded:
- `pkg.installed`: Records package installation
- `pkg.removed`: Updates package records
- Tracks package manager used (winget, choco, apt, etc.)

## Programmatic Usage

### Opening the CMDB

```go
import "github.com/nativeit-dev/tacoscript/cmdb"

// Use default location
db, err := cmdb.OpenDefaultCMDB()
if err != nil {
    return err
}
defer db.Close()

// Use custom location
db, err := cmdb.NewCMDB(cmdb.Config{
    DBPath: "/path/to/database.db",
})
```

### Creating a CI

```go
ci := &cmdb.ConfigurationItem{
    CIName:      "web-server-01",
    CIType:      "Server",
    CIClass:     "Physical",
    CIStatus:    "Active",
    Environment: "Production",
    Owner:       "ops-team",
    Criticality: "High",
    CreatedBy:   "admin",
    UpdatedBy:   "admin",
}

err := db.CreateCI(ci)
```

### Recording Execution

```go
exec := &cmdb.ExecutionHistory{
    CIID:       &ciID,
    ScriptPath: "/path/to/script.yaml",
    TaskCount:  10,
    ExecutedBy: "admin",
}

err := db.CreateExecution(exec)

// ... run tasks ...

exec.ExecutionStatus = "Success"
exec.SuccessCount = 8
exec.FailedCount = 2
err = db.UpdateExecution(exec)
```

### Recording Package Installation

```go
pkg := &cmdb.InstalledPackage{
    CIID:           ciID,
    PackageName:    "nginx",
    PackageVersion: "1.24.0",
    PackageManager: "apt",
}

err := db.RecordPackageInstallation(pkg)
```

## Database Schema

The CMDB uses 10 tables to track all aspects of configuration management:

1. **configuration_items**: Core CI records
2. **ci_attributes**: Flexible key-value attributes for CIs
3. **ci_relationships**: CI-to-CI relationships
4. **ci_change_history**: Audit trail of all CI changes
5. **execution_history**: Tacoscript execution records
6. **task_executions**: Individual task execution details
7. **installed_packages**: Software package inventory
8. **system_info**: Hardware and OS information
9. **schema_version**: Database schema version tracking

## Best Practices

### CI Naming Conventions

Use consistent naming conventions:
- **Servers**: `<env>-<role>-<number>` (e.g., `prod-web-01`)
- **Workstations**: `<location>-<user>-<type>` (e.g., `hq-jsmith-laptop`)
- **Services**: `<name>-<env>` (e.g., `api-prod`)

### Environment Classification

Use standard environment names:
- **Production**: Live production systems
- **Staging**: Pre-production staging
- **Development**: Development environment
- **Test**: Testing environment
- **DR**: Disaster recovery

### Criticality Levels

Assign appropriate criticality:
- **Critical**: Business-critical systems
- **High**: Important systems with significant impact
- **Medium**: Standard business systems
- **Low**: Non-critical systems

### Regular Discovery

Run discovery regularly to keep system information current:

```bash
taco cmdb discover
```

### Change Tracking

All CI updates are automatically tracked in change history, providing complete audit trails for compliance.

## Reporting

### CI Summary

```bash
taco cmdb list --limit 0  # List all CIs
```

### System Health

```bash
taco cmdb executions --limit 100  # Recent executions
```

### Change Audit

```bash
taco cmdb history <ci-name> --limit 0  # All changes
```

## Backup and Restore

### Backup

Simply copy the database file:

```bash
# Windows
copy %APPDATA%\tacoscript\tacoscript.cmdb.db <backup-location>

# Linux/macOS
cp ~/.config/tacoscript/tacoscript.cmdb.db <backup-location>
```

### Restore

Copy the backup file back to the CMDB location.

## Performance

The SQLite database provides excellent performance for typical CMDB operations:

- Indexes on all foreign keys and commonly queried fields
- Efficient change tracking with minimal overhead
- Supports thousands of CIs and millions of records

## Security

- Database file permissions restrict access to the owner
- No network exposure (local file-based)
- Audit trail for all changes
- SQL injection protection through parameterized queries

## Future Enhancements

Planned features:
- CMDB query language for complex reports
- Export to CMDB-compatible formats (JSON, XML)
- Integration with external CMDBs
- Web-based CMDB viewer
- Automated compliance reporting
- Relationship visualization
