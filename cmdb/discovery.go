package cmdb

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
)

// DiscoverSystem discovers system information and creates/updates a CI
func (c *CMDB) DiscoverSystem() (*ConfigurationItem, error) {
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}

	// Determine CI type based on platform
	ciType := "Workstation"
	if runtime.GOOS == "linux" {
		// Check if it's likely a server (no graphical environment, etc.)
		if _, err := os.Stat("/usr/bin/systemctl"); err == nil {
			ciType = "Server"
		}
	}

	// Get or create CI
	ci, err := c.GetOrCreateCI(hostname, ciType)
	if err != nil {
		return nil, fmt.Errorf("failed to get/create CI: %w", err)
	}

	// Gather system information
	sysInfo, err := c.gatherSystemInfo(ci.CIID, hostname)
	if err != nil {
		c.logger.Warnf("Failed to gather system info: %v", err)
	} else {
		if err := c.UpdateSystemInfo(sysInfo); err != nil {
			c.logger.Warnf("Failed to update system info: %v", err)
		}
	}

	// Add basic attributes
	attrs := map[string]string{
		"hostname":     hostname,
		"platform":     runtime.GOOS,
		"architecture": runtime.GOARCH,
	}

	for key, value := range attrs {
		attr := &CIAttribute{
			CIID:      ci.CIID,
			AttrKey:   key,
			AttrValue: value,
		}
		if err := c.AddCIAttribute(attr); err != nil {
			c.logger.Warnf("Failed to add attribute %s: %v", key, err)
		}
	}

	c.logger.Infof("Discovered system: %s (CI ID: %d)", hostname, ci.CIID)
	return ci, nil
}

// gatherSystemInfo collects system information using gopsutil
func (c *CMDB) gatherSystemInfo(ciID int64, hostname string) (*SystemInfo, error) {
	info := &SystemInfo{
		CIID:     ciID,
		Hostname: hostname,
	}

	// OS information
	hostInfo, err := host.Info()
	if err == nil {
		info.OSName = hostInfo.OS
		info.OSVersion = hostInfo.PlatformVersion
		info.OSPlatform = hostInfo.Platform
		info.OSFamily = hostInfo.PlatformFamily
	}

	// Architecture
	info.Architecture = runtime.GOARCH

	// CPU cores
	cpuCount, err := cpu.Counts(true)
	if err == nil {
		info.CPUCores = cpuCount
	}

	// Memory
	memInfo, err := mem.VirtualMemory()
	if err == nil {
		info.MemoryTotalMB = int(memInfo.Total / 1024 / 1024)
	}

	// Disk space
	diskInfo, err := disk.Usage("/")
	if err == nil {
		info.DiskTotalGB = int(diskInfo.Total / 1024 / 1024 / 1024)
	}

	return info, nil
}

// GetDefaultDBPath returns the default path for the CMDB database
func GetDefaultDBPath() string {
	userConfigDir, err := os.UserConfigDir()
	if err != nil {
		userConfigDir = "."
	}

	dbDir := filepath.Join(userConfigDir, "tacoscript")
	os.MkdirAll(dbDir, 0755)

	return filepath.Join(dbDir, "tacoscript.cmdb.db")
}

// OpenDefaultCMDB opens the CMDB at the default location
func OpenDefaultCMDB() (*CMDB, error) {
	dbPath := GetDefaultDBPath()
	return NewCMDB(Config{DBPath: dbPath})
}
