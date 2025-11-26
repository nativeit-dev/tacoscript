package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/nativeit-dev/tacoscript/cmdb"
	"github.com/spf13/cobra"
)

var (
	cmdbPath    string
	ciType      string
	ciStatus    string
	environment string
	limitRows   int
)

var cmdbCmd = &cobra.Command{
	Use:   "cmdb",
	Short: "Configuration Management Database operations",
	Long: `Manage the Configuration Management Database (CMDB) for tracking
system configurations, changes, and execution history.`,
}

var cmdbInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize the CMDB database",
	Long:  `Initialize the CMDB database at the default or specified location.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		dbPath := cmdbPath
		if dbPath == "" {
			dbPath = cmdb.GetDefaultDBPath()
		}

		db, err := cmdb.NewCMDB(cmdb.Config{DBPath: dbPath})
		if err != nil {
			return fmt.Errorf("failed to initialize CMDB: %w", err)
		}
		defer db.Close()

		fmt.Printf("CMDB initialized at: %s\n", dbPath)
		return nil
	},
}

var cmdbDiscoverCmd = &cobra.Command{
	Use:   "discover",
	Short: "Discover and register the current system",
	Long:  `Discover system information and register it as a Configuration Item.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		db, err := cmdb.OpenDefaultCMDB()
		if err != nil {
			return fmt.Errorf("failed to open CMDB: %w", err)
		}
		defer db.Close()

		ci, err := db.DiscoverSystem()
		if err != nil {
			return fmt.Errorf("failed to discover system: %w", err)
		}

		fmt.Printf("System discovered and registered:\n")
		fmt.Printf("  CI ID:       %d\n", ci.CIID)
		fmt.Printf("  Name:        %s\n", ci.CIName)
		fmt.Printf("  Type:        %s\n", ci.CIType)
		fmt.Printf("  Status:      %s\n", ci.CIStatus)
		fmt.Printf("  Environment: %s\n", ci.Environment)

		return nil
	},
}

var cmdbListCmd = &cobra.Command{
	Use:   "list",
	Short: "List Configuration Items",
	Long:  `List all Configuration Items in the CMDB with optional filtering.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		db, err := cmdb.OpenDefaultCMDB()
		if err != nil {
			return fmt.Errorf("failed to open CMDB: %w", err)
		}
		defer db.Close()

		filter := &cmdb.CIFilter{
			CIType:      ciType,
			CIStatus:    ciStatus,
			Environment: environment,
			Limit:       limitRows,
		}

		cis, err := db.ListCIs(filter)
		if err != nil {
			return fmt.Errorf("failed to list CIs: %w", err)
		}

		if len(cis) == 0 {
			fmt.Println("No Configuration Items found.")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "ID\tName\tType\tStatus\tEnvironment\tOwner\tUpdated")
		fmt.Fprintln(w, "--\t----\t----\t------\t-----------\t-----\t-------")

		for _, ci := range cis {
			fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\t%s\t%s\n",
				ci.CIID, ci.CIName, ci.CIType, ci.CIStatus,
				ci.Environment, ci.Owner, ci.UpdatedAt.Format("2006-01-02"),
			)
		}

		w.Flush()
		fmt.Printf("\nTotal: %d Configuration Items\n", len(cis))
		return nil
	},
}

var cmdbShowCmd = &cobra.Command{
	Use:   "show <ci-name>",
	Short: "Show detailed information about a CI",
	Long:  `Display detailed information about a specific Configuration Item.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		db, err := cmdb.OpenDefaultCMDB()
		if err != nil {
			return fmt.Errorf("failed to open CMDB: %w", err)
		}
		defer db.Close()

		ciName := args[0]
		ci, err := db.GetCIByName(ciName)
		if err != nil {
			return fmt.Errorf("failed to get CI: %w", err)
		}

		fmt.Printf("Configuration Item Details:\n")
		fmt.Printf("  CI ID:         %d\n", ci.CIID)
		fmt.Printf("  Name:          %s\n", ci.CIName)
		fmt.Printf("  Type:          %s\n", ci.CIType)
		fmt.Printf("  Class:         %s\n", ci.CIClass)
		fmt.Printf("  Status:        %s\n", ci.CIStatus)
		fmt.Printf("  Environment:   %s\n", ci.Environment)
		fmt.Printf("  Owner:         %s\n", ci.Owner)
		fmt.Printf("  Description:   %s\n", ci.Description)
		fmt.Printf("  Serial Number: %s\n", ci.SerialNumber)
		fmt.Printf("  Asset Tag:     %s\n", ci.AssetTag)
		fmt.Printf("  Location:      %s\n", ci.Location)
		fmt.Printf("  Criticality:   %s\n", ci.Criticality)
		fmt.Printf("  Created:       %s by %s\n", ci.CreatedAt.Format(time.RFC3339), ci.CreatedBy)
		fmt.Printf("  Updated:       %s by %s\n", ci.UpdatedAt.Format(time.RFC3339), ci.UpdatedBy)

		// Get attributes
		attrs, err := db.GetCIAttributes(ci.CIID)
		if err == nil && len(attrs) > 0 {
			fmt.Printf("\nAttributes:\n")
			for _, attr := range attrs {
				fmt.Printf("  %s: %s\n", attr.AttrKey, attr.AttrValue)
			}
		}

		// Get system info
		sysInfo, err := db.GetSystemInfo(ci.CIID)
		if err == nil {
			fmt.Printf("\nSystem Information:\n")
			fmt.Printf("  Hostname:     %s\n", sysInfo.Hostname)
			fmt.Printf("  OS:           %s %s\n", sysInfo.OSName, sysInfo.OSVersion)
			fmt.Printf("  Platform:     %s (%s)\n", sysInfo.OSPlatform, sysInfo.OSFamily)
			fmt.Printf("  Architecture: %s\n", sysInfo.Architecture)
			fmt.Printf("  CPU Cores:    %d\n", sysInfo.CPUCores)
			fmt.Printf("  Memory:       %d MB\n", sysInfo.MemoryTotalMB)
			fmt.Printf("  Disk:         %d GB\n", sysInfo.DiskTotalGB)
			fmt.Printf("  Discovered:   %s\n", sysInfo.LastDiscovered.Format(time.RFC3339))
		}

		// Get installed packages count
		packages, err := db.GetInstalledPackages(ci.CIID)
		if err == nil {
			fmt.Printf("\nInstalled Packages: %d\n", len(packages))
		}

		return nil
	},
}

var cmdbHistoryCmd = &cobra.Command{
	Use:   "history <ci-name>",
	Short: "Show change history for a CI",
	Long:  `Display the change history for a specific Configuration Item.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		db, err := cmdb.OpenDefaultCMDB()
		if err != nil {
			return fmt.Errorf("failed to open CMDB: %w", err)
		}
		defer db.Close()

		ciName := args[0]
		ci, err := db.GetCIByName(ciName)
		if err != nil {
			return fmt.Errorf("failed to get CI: %w", err)
		}

		history, err := db.GetCIHistory(ci.CIID, limitRows)
		if err != nil {
			return fmt.Errorf("failed to get CI history: %w", err)
		}

		if len(history) == 0 {
			fmt.Println("No change history found.")
			return nil
		}

		fmt.Printf("Change History for %s:\n\n", ci.CIName)

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "Changed At\tType\tField\tOld Value\tNew Value\tChanged By")
		fmt.Fprintln(w, "----------\t----\t-----\t---------\t---------\t----------")

		for _, h := range history {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
				h.ChangedAt.Format("2006-01-02 15:04:05"),
				h.ChangeType, h.FieldName, h.OldValue, h.NewValue, h.ChangedBy,
			)
		}

		w.Flush()
		fmt.Printf("\nTotal: %d changes\n", len(history))
		return nil
	},
}

var cmdbExecutionsCmd = &cobra.Command{
	Use:   "executions [ci-name]",
	Short: "Show execution history",
	Long:  `Display execution history, optionally filtered by CI name.`,
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		db, err := cmdb.OpenDefaultCMDB()
		if err != nil {
			return fmt.Errorf("failed to open CMDB: %w", err)
		}
		defer db.Close()

		filter := &cmdb.ExecutionFilter{
			Limit: limitRows,
		}

		if len(args) > 0 {
			ciName := args[0]
			ci, err := db.GetCIByName(ciName)
			if err != nil {
				return fmt.Errorf("failed to get CI: %w", err)
			}
			filter.CIID = &ci.CIID
		}

		executions, err := db.ListExecutions(filter)
		if err != nil {
			return fmt.Errorf("failed to list executions: %w", err)
		}

		if len(executions) == 0 {
			fmt.Println("No executions found.")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "ID\tStart Time\tStatus\tDuration\tTasks\tSuccess\tFailed\tScript")
		fmt.Fprintln(w, "--\t----------\t------\t--------\t-----\t-------\t------\t------")

		for _, exec := range executions {
			duration := "N/A"
			if exec.DurationMS != nil {
				duration = fmt.Sprintf("%dms", *exec.DurationMS)
			}

			fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%d\t%d\t%d\t%s\n",
				exec.ExecID,
				exec.StartTime.Format("2006-01-02 15:04"),
				exec.ExecutionStatus,
				duration,
				exec.TaskCount,
				exec.SuccessCount,
				exec.FailedCount,
				exec.ScriptPath,
			)
		}

		w.Flush()
		fmt.Printf("\nTotal: %d executions\n", len(executions))
		return nil
	},
}

var cmdbStatsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show CMDB statistics",
	Long:  `Display statistics about the CMDB.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		db, err := cmdb.OpenDefaultCMDB()
		if err != nil {
			return fmt.Errorf("failed to open CMDB: %w", err)
		}
		defer db.Close()

		stats, err := db.GetStats()
		if err != nil {
			return fmt.Errorf("failed to get stats: %w", err)
		}

		fmt.Printf("CMDB Statistics:\n")
		fmt.Printf("  Database:             %s\n", db.GetDBPath())
		fmt.Printf("  Configuration Items:  %d\n", stats["total_cis"])
		fmt.Printf("  Active CIs:           %d\n", stats["active_cis"])
		fmt.Printf("  Change Records:       %d\n", stats["total_changes"])
		fmt.Printf("  Executions:           %d\n", stats["total_executions"])
		fmt.Printf("  Installed Packages:   %d\n", stats["total_packages"])

		// Get execution summary
		summary, err := db.GetExecutionSummary(nil, 30)
		if err == nil {
			fmt.Printf("\nExecution Summary (Last 30 Days):\n")
			fmt.Printf("  Total:         %d\n", summary.TotalExecutions)
			fmt.Printf("  Success:       %d\n", summary.SuccessCount)
			fmt.Printf("  Failed:        %d\n", summary.FailedCount)
			fmt.Printf("  Partial:       %d\n", summary.PartialCount)
			fmt.Printf("  Avg Duration:  %.0fms\n", summary.AverageDuration)
			fmt.Printf("  Total Tasks:   %d\n", summary.TotalTasks)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(cmdbCmd)

	// Add subcommands
	cmdbCmd.AddCommand(cmdbInitCmd)
	cmdbCmd.AddCommand(cmdbDiscoverCmd)
	cmdbCmd.AddCommand(cmdbListCmd)
	cmdbCmd.AddCommand(cmdbShowCmd)
	cmdbCmd.AddCommand(cmdbHistoryCmd)
	cmdbCmd.AddCommand(cmdbExecutionsCmd)
	cmdbCmd.AddCommand(cmdbStatsCmd)

	// Global flags
	cmdbCmd.PersistentFlags().StringVar(&cmdbPath, "db", "", "Path to CMDB database (default: user config dir)")

	// List command flags
	cmdbListCmd.Flags().StringVar(&ciType, "type", "", "Filter by CI type")
	cmdbListCmd.Flags().StringVar(&ciStatus, "status", "", "Filter by CI status")
	cmdbListCmd.Flags().StringVar(&environment, "environment", "", "Filter by environment")
	cmdbListCmd.Flags().IntVar(&limitRows, "limit", 100, "Maximum number of rows to return")

	// History command flags
	cmdbHistoryCmd.Flags().IntVar(&limitRows, "limit", 50, "Maximum number of history records")

	// Executions command flags
	cmdbExecutionsCmd.Flags().IntVar(&limitRows, "limit", 50, "Maximum number of executions")
}
