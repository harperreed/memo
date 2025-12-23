// ABOUTME: Sync subcommand for Charm cloud integration.
// ABOUTME: Provides link, unlink, status, and wipe commands for Charm sync.

package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	charmkv "github.com/charmbracelet/charm/kv"
	"github.com/fatih/color"
	"github.com/harper/memo/internal/charm"
	"github.com/spf13/cobra"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Manage Charm cloud sync",
	Long: `Sync your memo notes to the Charm cloud.

Charm uses SSH key authentication - no passwords needed.
Data syncs automatically after each change.

Commands:
  status  - Show sync configuration and connection status
  link    - Connect this device to Charm cloud
  unlink  - Disconnect from Charm cloud
  repair  - Repair database corruption issues
  reset   - Reset local sync data (keeps cloud data)
  wipe    - Delete all synced data and start fresh

Examples:
  memo sync status
  memo sync link
  memo sync link --host charm.example.com
  memo sync repair
  memo sync reset`,
}

var syncStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show sync status",
	Long:  `Display Charm sync configuration and connection status.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := charm.LoadConfig()
		if err != nil {
			cfg = &charm.Config{AutoSync: true}
		}

		fmt.Println("Charm Sync Status")
		fmt.Println(strings.Repeat("-", 40))

		// Show config
		fmt.Printf("Config:    %s\n", charm.ConfigPath())
		if cfg.CharmHost != "" {
			fmt.Printf("Host:      %s\n", cfg.CharmHost)
		} else {
			fmt.Printf("Host:      %s\n", color.New(color.Faint).Sprint("(default: cloud.charm.sh)"))
		}

		if cfg.AutoSync {
			fmt.Printf("Auto-sync: %s\n", color.GreenString("enabled"))
		} else {
			fmt.Printf("Auto-sync: %s\n", color.YellowString("disabled"))
		}

		// Try to get charm user info
		if charmClient != nil {
			user, err := charmClient.User()
			if err == nil && user != nil {
				fmt.Println()
				fmt.Printf("User ID:   %s\n", user.CharmID)
				fmt.Printf("Name:      %s\n", valueOrNone(user.Name))
				fmt.Printf("Status:    %s\n", color.GreenString("connected"))
			} else {
				fmt.Println()
				fmt.Printf("Status:    %s\n", color.YellowString("not linked"))
				fmt.Println("\nRun 'memo sync link' to connect to Charm cloud.")
			}
		} else {
			fmt.Println()
			fmt.Printf("Status:    %s\n", color.RedString("client not initialized"))
		}

		return nil
	},
}

var syncLinkCmd = &cobra.Command{
	Use:   "link",
	Short: "Connect to Charm cloud",
	Long: `Link this device to Charm cloud for sync.

Charm uses SSH key authentication. On first link, you'll see
a code to verify on another device, or you can create a new account.

Your SSH keys are used automatically - no passwords needed.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		host, _ := cmd.Flags().GetString("host")

		// Load or create config
		cfg, err := charm.LoadConfig()
		if err != nil {
			cfg = &charm.Config{AutoSync: true}
		}

		if host != "" {
			cfg.CharmHost = host
		}

		// Save config before linking (in case host changed)
		if err := charm.SaveConfig(cfg); err != nil {
			return fmt.Errorf("save config: %w", err)
		}

		// Re-initialize client with new config
		if err := charm.ResetClient(); err != nil {
			return fmt.Errorf("reset client: %w", err)
		}

		client, err := charm.GetClient()
		if err != nil {
			return fmt.Errorf("get client: %w", err)
		}

		// Link will prompt for authentication if needed
		if err := client.Link(); err != nil {
			return fmt.Errorf("link failed: %w", err)
		}

		user, err := client.User()
		if err != nil {
			return fmt.Errorf("get user: %w", err)
		}

		color.Green("\n✓ Linked to Charm cloud")
		fmt.Printf("  User ID: %s\n", user.CharmID)
		if user.Name != "" {
			fmt.Printf("  Name:    %s\n", user.Name)
		}
		fmt.Println("\nYour notes will now sync automatically.")

		return nil
	},
}

var syncUnlinkCmd = &cobra.Command{
	Use:   "unlink",
	Short: "Disconnect from Charm cloud",
	Long: `Unlink this device from Charm cloud.

This removes your SSH key association but keeps local data.
You can re-link anytime with 'memo sync link'.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if charmClient == nil {
			fmt.Println("Not linked to Charm cloud.")
			return nil
		}

		// Confirm with user
		fmt.Println("This will disconnect this device from Charm cloud.")
		fmt.Println("Your local notes will be preserved.")
		fmt.Print("\nType 'unlink' to confirm: ")

		reader := bufio.NewReader(os.Stdin)
		confirmation, _ := reader.ReadString('\n')
		confirmation = strings.TrimSpace(confirmation)

		if confirmation != "unlink" {
			fmt.Println("Aborted.")
			return nil
		}

		if err := charmClient.Unlink(); err != nil {
			return fmt.Errorf("unlink failed: %w", err)
		}

		color.Green("\n✓ Unlinked from Charm cloud")
		fmt.Println("Run 'memo sync link' to reconnect.")

		return nil
	},
}

var syncRepairCmd = &cobra.Command{
	Use:   "repair",
	Short: "Repair database corruption issues",
	Long: `Repair the local KV database if it's corrupted.

This command:
- Checkpoints the WAL (write-ahead log)
- Removes shared memory files
- Runs integrity checks
- Vacuums the database if needed

Use --force to attempt repair even if integrity check fails.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		force, _ := cmd.Flags().GetBool("force")

		fmt.Println("Repairing database...")
		result, err := charmkv.Repair(charm.DBName, force)
		if err != nil {
			return fmt.Errorf("repair failed: %w", err)
		}

		fmt.Println("\nRepair Results:")
		if result.WalCheckpointed {
			fmt.Println("  ✓ WAL checkpointed")
		}
		if result.ShmRemoved {
			fmt.Println("  ✓ SHM file removed")
		}
		if result.IntegrityOK {
			color.Green("  ✓ Integrity check passed")
		} else {
			color.Red("  ✗ Integrity check failed")
		}
		if result.Vacuumed {
			fmt.Println("  ✓ Database vacuumed")
		}

		if result.IntegrityOK {
			color.Green("\n✓ Database repaired successfully")
		} else {
			color.Yellow("\n⚠ Repair completed but integrity issues remain")
			fmt.Println("Consider running 'memo sync reset' or 'memo sync wipe'")
		}

		return nil
	},
}

var syncResetCmd = &cobra.Command{
	Use:   "reset",
	Short: "Reset local sync data",
	Long: `Reset the local KV database while keeping cloud data intact.

This removes all local sync state and forces a fresh sync from the cloud.
Use this when:
- Local database is corrupted
- You want to re-sync from cloud
- Sync state has diverged

Your cloud data is preserved.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Confirm with user
		fmt.Println("This will reset local sync data.")
		fmt.Println("Cloud data will be preserved and re-synced.")
		fmt.Print("\nContinue? [y/N]: ")

		reader := bufio.NewReader(os.Stdin)
		confirmation, _ := reader.ReadString('\n')
		confirmation = strings.TrimSpace(strings.ToLower(confirmation))

		if confirmation != "y" && confirmation != "yes" {
			fmt.Println("Aborted.")
			return nil
		}

		fmt.Println("\nResetting local data...")

		if err := charmkv.Reset(charm.DBName); err != nil {
			return fmt.Errorf("reset failed: %w", err)
		}

		color.Green("✓ Local sync data reset")
		fmt.Println("\nRun any memo command to re-sync from cloud.")

		return nil
	},
}

var syncWipeCmd = &cobra.Command{
	Use:   "wipe",
	Short: "Wipe all sync data and start fresh",
	Long: `Delete all synced data from Charm cloud and local KV store.

This is the nuclear option - use when:
- Sync data is corrupted beyond repair
- You want to start completely fresh
- You're cleaning up after development/testing

This deletes BOTH cloud backups and local files.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Confirm with user
		fmt.Println("This will DELETE all sync data:")
		fmt.Println("  - All notes in Charm cloud")
		fmt.Println("  - Local KV database")
		fmt.Println()
		color.Yellow("This cannot be undone!")
		fmt.Print("\nType 'wipe' to confirm: ")

		reader := bufio.NewReader(os.Stdin)
		confirmation, _ := reader.ReadString('\n')
		confirmation = strings.TrimSpace(confirmation)

		if confirmation != "wipe" {
			fmt.Println("Aborted.")
			return nil
		}

		fmt.Println("\nWiping data...")

		result, err := charmkv.Wipe(charm.DBName)
		if err != nil {
			return fmt.Errorf("wipe failed: %w", err)
		}

		fmt.Println("\nWipe Results:")
		if result.CloudBackupsDeleted > 0 {
			fmt.Printf("  ✓ Deleted %d cloud backups\n", result.CloudBackupsDeleted)
		}
		if result.LocalFilesDeleted > 0 {
			fmt.Printf("  ✓ Deleted %d local files\n", result.LocalFilesDeleted)
		}

		color.Green("\n✓ All sync data wiped")
		fmt.Println("\nRun any memo command to start fresh.")

		return nil
	},
}

func init() {
	syncLinkCmd.Flags().String("host", "", "Charm server host (default: cloud.charm.sh)")
	syncRepairCmd.Flags().Bool("force", false, "Force repair even if integrity check fails")

	syncCmd.AddCommand(syncStatusCmd)
	syncCmd.AddCommand(syncLinkCmd)
	syncCmd.AddCommand(syncUnlinkCmd)
	syncCmd.AddCommand(syncRepairCmd)
	syncCmd.AddCommand(syncResetCmd)
	syncCmd.AddCommand(syncWipeCmd)

	rootCmd.AddCommand(syncCmd)
}

// valueOrNone returns "(not set)" if the string is empty.
func valueOrNone(s string) string {
	if s == "" {
		return "(not set)"
	}
	return s
}
