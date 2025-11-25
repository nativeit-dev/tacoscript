//go:build windows
// +build windows

package pkgmanager

import (
	"context"
	"fmt"
	"strings"

	"github.com/nativeit-dev/tacoscript/tasks/pkgtask"
)

// WingetCmdProvider provides commands for Windows Package Manager (winget)
type WingetCmdProvider struct{}

// ChocoCmdProvider provides commands for Chocolatey package manager
type ChocoCmdProvider struct{}

func BuildManagementCmdsProviders() ([]ManagementCmdsProvider, error) {
	// Winget is tried first as it's the modern, built-in Windows package manager
	// Falls back to Chocolatey if winget is not available
	return []ManagementCmdsProvider{
		WingetCmdProvider{},
		ChocoCmdProvider{},
	}, nil
}

func (w WingetCmdProvider) GetManagementCmds(t *pkgtask.Task) (*ManagementCmds, error) {
	rawCmds := t.Named.GetNames()

	// Build install commands - winget requires separate commands per package
	installCmds := make([]string, 0, len(rawCmds))
	for _, cmd := range rawCmds {
		if t.Version != "" {
			// Winget version specification: --version flag
			installCmds = append(installCmds, fmt.Sprintf("winget install --exact --id %s --version %s --silent --accept-package-agreements --accept-source-agreements", cmd, t.Version))
		} else {
			installCmds = append(installCmds, fmt.Sprintf("winget install --exact --id %s --silent --accept-package-agreements --accept-source-agreements", cmd))
		}
	}

	// Build uninstall commands
	uninstallCmds := make([]string, 0, len(rawCmds))
	for _, cmd := range rawCmds {
		uninstallCmds = append(uninstallCmds, fmt.Sprintf("winget uninstall --exact --id %s --silent", cmd))
	}

	// Build upgrade commands
	upgradeCmds := make([]string, 0, len(rawCmds))
	for _, cmd := range rawCmds {
		upgradeCmds = append(upgradeCmds, fmt.Sprintf("winget upgrade --exact --id %s --silent --accept-package-agreements --accept-source-agreements", cmd))
	}

	return &ManagementCmds{
		VersionCmd:    "winget --version",
		UpgradeCmd:    "winget source update",
		InstallCmds:   installCmds,
		UninstallCmds: uninstallCmds,
		UpgradeCmds:   upgradeCmds,
		ListCmd:       "winget list",
		FilterFunc: func(ctx context.Context, rawPackages []string) []string {
			res := make([]string, 0, len(rawPackages))
			for i, rawPackage := range rawPackages {
				// Skip header lines (first 2 lines) and separator/empty lines
				if i < 2 || strings.HasPrefix(rawPackage, "-") || strings.TrimSpace(rawPackage) == "" {
					continue
				}
				res = append(res, rawPackage)
			}
			return res
		},
	}, nil
}

func (c ChocoCmdProvider) GetManagementCmds(t *pkgtask.Task) (*ManagementCmds, error) {
	rawCmds := t.Named.GetNames()

	versionStr := ""
	if t.Version != "" {
		// Chocolatey uses --version= format
		versionStr += " --version=" + t.Version
	}

	return &ManagementCmds{
		VersionCmd:    "choco --version",
		UpgradeCmd:    "choco upgrade -y chocolatey",
		InstallCmds:   []string{fmt.Sprintf("choco install -y %s%s", strings.Join(rawCmds, " "), versionStr)},
		UninstallCmds: []string{fmt.Sprintf("choco uninstall -y %s", strings.Join(rawCmds, " "))},
		UpgradeCmds:   []string{fmt.Sprintf("choco upgrade -y %s", strings.Join(rawCmds, " "))},
		ListCmd:       "choco list --local-only",
		FilterFunc: func(ctx context.Context, rawPackages []string) []string {
			res := make([]string, 0, len(rawPackages))
			for _, rawPackage := range rawPackages {
				if strings.Contains(rawPackage, "packages installed") {
					continue
				}
				res = append(res, rawPackage)
			}

			return res
		},
	}, nil
}
