package cmdb

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// CreateCI creates a new Configuration Item
func (c *CMDB) CreateCI(ci *ConfigurationItem) error {
	now := time.Now()
	ci.CreatedAt = now
	ci.UpdatedAt = now

	query := `
		INSERT INTO configuration_items (
			ci_name, ci_type, ci_class, ci_status, environment,
			owner, description, serial_number, asset_tag, location,
			criticality, created_at, updated_at, created_by, updated_by
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	result, err := c.db.Exec(query,
		ci.CIName, ci.CIType, ci.CIClass, ci.CIStatus, ci.Environment,
		ci.Owner, ci.Description, ci.SerialNumber, ci.AssetTag, ci.Location,
		ci.Criticality, ci.CreatedAt, ci.UpdatedAt, ci.CreatedBy, ci.UpdatedBy,
	)
	if err != nil {
		return fmt.Errorf("failed to create CI: %w", err)
	}

	ciID, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get CI ID: %w", err)
	}
	ci.CIID = ciID

	c.logger.Infof("Created CI: %s (ID: %d)", ci.CIName, ci.CIID)
	return nil
}

// GetCI retrieves a CI by ID
func (c *CMDB) GetCI(ciID int64) (*ConfigurationItem, error) {
	query := `
		SELECT ci_id, ci_name, ci_type, ci_class, ci_status, environment,
		       owner, description, serial_number, asset_tag, location,
		       criticality, created_at, updated_at, created_by, updated_by
		FROM configuration_items
		WHERE ci_id = ?
	`

	ci := &ConfigurationItem{}
	err := ci.ScanRow(c.db.QueryRow(query, ciID))
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("CI not found: %d", ciID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get CI: %w", err)
	}

	return ci, nil
}

// GetCIByName retrieves a CI by name
func (c *CMDB) GetCIByName(ciName string) (*ConfigurationItem, error) {
	query := `
		SELECT ci_id, ci_name, ci_type, ci_class, ci_status, environment,
		       owner, description, serial_number, asset_tag, location,
		       criticality, created_at, updated_at, created_by, updated_by
		FROM configuration_items
		WHERE ci_name = ?
	`

	ci := &ConfigurationItem{}
	err := ci.ScanRow(c.db.QueryRow(query, ciName))
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("CI not found: %s", ciName)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get CI: %w", err)
	}

	return ci, nil
}

// UpdateCI updates an existing CI
func (c *CMDB) UpdateCI(ci *ConfigurationItem) error {
	// First get the old values for audit trail
	oldCI, err := c.GetCI(ci.CIID)
	if err != nil {
		return err
	}

	ci.UpdatedAt = time.Now()

	query := `
		UPDATE configuration_items
		SET ci_name = ?, ci_type = ?, ci_class = ?, ci_status = ?,
		    environment = ?, owner = ?, description = ?,
		    serial_number = ?, asset_tag = ?, location = ?,
		    criticality = ?, updated_at = ?, updated_by = ?
		WHERE ci_id = ?
	`

	_, err = c.db.Exec(query,
		ci.CIName, ci.CIType, ci.CIClass, ci.CIStatus, ci.Environment,
		ci.Owner, ci.Description, ci.SerialNumber, ci.AssetTag, ci.Location,
		ci.Criticality, ci.UpdatedAt, ci.UpdatedBy, ci.CIID,
	)
	if err != nil {
		return fmt.Errorf("failed to update CI: %w", err)
	}

	// Record changes in history
	if err := c.recordCIChanges(oldCI, ci); err != nil {
		c.logger.Warnf("Failed to record CI changes: %v", err)
	}

	c.logger.Infof("Updated CI: %s (ID: %d)", ci.CIName, ci.CIID)
	return nil
}

// DeleteCI deletes a CI and all related records
func (c *CMDB) DeleteCI(ciID int64) error {
	ci, err := c.GetCI(ciID)
	if err != nil {
		return err
	}

	tx, err := c.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	// Delete related records
	tables := []string{
		"ci_attributes",
		"ci_relationships",
		"ci_change_history",
		"task_executions",
		"installed_packages",
		"system_info",
	}

	for _, table := range tables {
		_, err = tx.Exec(fmt.Sprintf("DELETE FROM %s WHERE ci_id = ?", table), ciID)
		if err != nil {
			return fmt.Errorf("failed to delete from %s: %w", table, err)
		}
	}

	// Delete CI itself
	_, err = tx.Exec("DELETE FROM configuration_items WHERE ci_id = ?", ciID)
	if err != nil {
		return fmt.Errorf("failed to delete CI: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit deletion: %w", err)
	}

	c.logger.Infof("Deleted CI: %s (ID: %d)", ci.CIName, ciID)
	return nil
}

// ListCIs retrieves CIs with optional filtering
func (c *CMDB) ListCIs(filter *CIFilter) ([]*ConfigurationItem, error) {
	query := `
		SELECT ci_id, ci_name, ci_type, ci_class, ci_status, environment,
		       owner, description, serial_number, asset_tag, location,
		       criticality, created_at, updated_at, created_by, updated_by
		FROM configuration_items
		WHERE 1=1
	`
	args := []interface{}{}

	if filter != nil {
		if filter.CIType != "" {
			query += " AND ci_type = ?"
			args = append(args, filter.CIType)
		}
		if filter.CIStatus != "" {
			query += " AND ci_status = ?"
			args = append(args, filter.CIStatus)
		}
		if filter.Environment != "" {
			query += " AND environment = ?"
			args = append(args, filter.Environment)
		}
		if filter.Owner != "" {
			query += " AND owner = ?"
			args = append(args, filter.Owner)
		}
		if filter.Search != "" {
			query += " AND (ci_name LIKE ? OR description LIKE ?)"
			searchPattern := "%" + filter.Search + "%"
			args = append(args, searchPattern, searchPattern)
		}

		query += " ORDER BY ci_name"

		if filter.Limit > 0 {
			query += " LIMIT ?"
			args = append(args, filter.Limit)
		}
		if filter.Offset > 0 {
			query += " OFFSET ?"
			args = append(args, filter.Offset)
		}
	} else {
		query += " ORDER BY ci_name"
	}

	rows, err := c.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list CIs: %w", err)
	}
	defer rows.Close()

	var cis []*ConfigurationItem
	for rows.Next() {
		ci := &ConfigurationItem{}
		if err := ci.ScanRows(rows); err != nil {
			return nil, fmt.Errorf("failed to scan CI: %w", err)
		}
		cis = append(cis, ci)
	}

	return cis, nil
}

// AddCIAttribute adds or updates an attribute for a CI
func (c *CMDB) AddCIAttribute(attr *CIAttribute) error {
	now := time.Now()
	attr.CreatedAt = now
	attr.UpdatedAt = now

	query := `
		INSERT INTO ci_attributes (ci_id, attr_key, attr_value, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(ci_id, attr_key) DO UPDATE SET
			attr_value = excluded.attr_value,
			updated_at = excluded.updated_at
	`

	_, err := c.db.Exec(query, attr.CIID, attr.AttrKey, attr.AttrValue, attr.CreatedAt, attr.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to add CI attribute: %w", err)
	}

	return nil
}

// GetCIAttributes retrieves all attributes for a CI
func (c *CMDB) GetCIAttributes(ciID int64) ([]*CIAttribute, error) {
	query := `
		SELECT attr_id, ci_id, attr_key, attr_value, created_at, updated_at
		FROM ci_attributes
		WHERE ci_id = ?
		ORDER BY attr_key
	`

	rows, err := c.db.Query(query, ciID)
	if err != nil {
		return nil, fmt.Errorf("failed to get CI attributes: %w", err)
	}
	defer rows.Close()

	var attrs []*CIAttribute
	for rows.Next() {
		attr := &CIAttribute{}
		if err := rows.Scan(&attr.AttrID, &attr.CIID, &attr.AttrKey, &attr.AttrValue, &attr.CreatedAt, &attr.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan attribute: %w", err)
		}
		attrs = append(attrs, attr)
	}

	return attrs, nil
}

// CreateRelationship creates a relationship between two CIs
func (c *CMDB) CreateRelationship(rel *CIRelationship) error {
	rel.CreatedAt = time.Now()

	query := `
		INSERT INTO ci_relationships (source_ci_id, target_ci_id, relationship_type, description, created_at, created_by)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	result, err := c.db.Exec(query, rel.SourceCIID, rel.TargetCIID, rel.RelationshipType, rel.Description, rel.CreatedAt, rel.CreatedBy)
	if err != nil {
		return fmt.Errorf("failed to create relationship: %w", err)
	}

	relID, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get relationship ID: %w", err)
	}
	rel.RelID = relID

	return nil
}

// GetCIRelationships retrieves all relationships for a CI
func (c *CMDB) GetCIRelationships(ciID int64) ([]*CIRelationship, error) {
	query := `
		SELECT rel_id, source_ci_id, target_ci_id, relationship_type, description, created_at, created_by
		FROM ci_relationships
		WHERE source_ci_id = ? OR target_ci_id = ?
		ORDER BY created_at DESC
	`

	rows, err := c.db.Query(query, ciID, ciID)
	if err != nil {
		return nil, fmt.Errorf("failed to get relationships: %w", err)
	}
	defer rows.Close()

	var rels []*CIRelationship
	for rows.Next() {
		rel := &CIRelationship{}
		if err := rows.Scan(&rel.RelID, &rel.SourceCIID, &rel.TargetCIID, &rel.RelationshipType, &rel.Description, &rel.CreatedAt, &rel.CreatedBy); err != nil {
			return nil, fmt.Errorf("failed to scan relationship: %w", err)
		}
		rels = append(rels, rel)
	}

	return rels, nil
}

// recordCIChanges compares old and new CI and records changes
func (c *CMDB) recordCIChanges(oldCI, newCI *ConfigurationItem) error {
	changes := []struct {
		field    string
		oldValue string
		newValue string
	}{
		{"ci_name", oldCI.CIName, newCI.CIName},
		{"ci_type", oldCI.CIType, newCI.CIType},
		{"ci_class", oldCI.CIClass, newCI.CIClass},
		{"ci_status", oldCI.CIStatus, newCI.CIStatus},
		{"environment", oldCI.Environment, newCI.Environment},
		{"owner", oldCI.Owner, newCI.Owner},
		{"description", oldCI.Description, newCI.Description},
		{"serial_number", oldCI.SerialNumber, newCI.SerialNumber},
		{"asset_tag", oldCI.AssetTag, newCI.AssetTag},
		{"location", oldCI.Location, newCI.Location},
		{"criticality", oldCI.Criticality, newCI.Criticality},
	}

	for _, change := range changes {
		if change.oldValue != change.newValue {
			history := &CIChangeHistory{
				CIID:       newCI.CIID,
				ChangeType: "Updated",
				FieldName:  change.field,
				OldValue:   change.oldValue,
				NewValue:   change.newValue,
				ChangedBy:  newCI.UpdatedBy,
				ChangedAt:  time.Now(),
			}
			if err := c.RecordChange(history); err != nil {
				return err
			}
		}
	}

	return nil
}

// RecordChange records a change to a CI
func (c *CMDB) RecordChange(change *CIChangeHistory) error {
	change.ChangedAt = time.Now()

	query := `
		INSERT INTO ci_change_history (ci_id, change_type, field_name, old_value, new_value, changed_by, change_reason, changed_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := c.db.Exec(query, change.CIID, change.ChangeType, change.FieldName, change.OldValue, change.NewValue, change.ChangedBy, change.ChangeReason, change.ChangedAt)
	if err != nil {
		return fmt.Errorf("failed to record change: %w", err)
	}

	return nil
}

// GetCIHistory retrieves change history for a CI
func (c *CMDB) GetCIHistory(ciID int64, limit int) ([]*CIChangeHistory, error) {
	query := `
		SELECT change_id, ci_id, change_type, field_name, old_value, new_value, changed_by, change_reason, changed_at
		FROM ci_change_history
		WHERE ci_id = ?
		ORDER BY changed_at DESC
	`

	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	rows, err := c.db.Query(query, ciID)
	if err != nil {
		return nil, fmt.Errorf("failed to get CI history: %w", err)
	}
	defer rows.Close()

	var history []*CIChangeHistory
	for rows.Next() {
		h := &CIChangeHistory{}
		if err := rows.Scan(&h.ChangeID, &h.CIID, &h.ChangeType, &h.FieldName, &h.OldValue, &h.NewValue, &h.ChangedBy, &h.ChangeReason, &h.ChangedAt); err != nil {
			return nil, fmt.Errorf("failed to scan history: %w", err)
		}
		history = append(history, h)
	}

	return history, nil
}

// GetOrCreateCI gets an existing CI by name or creates it if it doesn't exist
func (c *CMDB) GetOrCreateCI(ciName, ciType string) (*ConfigurationItem, error) {
	ci, err := c.GetCIByName(ciName)
	if err == nil {
		return ci, nil
	}

	if !strings.Contains(err.Error(), "not found") {
		return nil, err
	}

	// CI doesn't exist, create it
	ci = &ConfigurationItem{
		CIName:      ciName,
		CIType:      ciType,
		CIClass:     "Physical",
		CIStatus:    "Active",
		Environment: "Unknown",
		CreatedBy:   "tacoscript",
		UpdatedBy:   "tacoscript",
	}

	if err := c.CreateCI(ci); err != nil {
		return nil, err
	}

	return ci, nil
}
