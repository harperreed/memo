// ABOUTME: Sync subcommand for Charm cloud integration.
// ABOUTME: Provides link, unlink, status, and wipe commands for Charm sync.

package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

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
  wipe    - Delete all synced data and start fresh

Examples:
  memo sync status
  memo sync link
  memo sync link --host charm.example.com`,
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

var syncWipeCmd = &cobra.Command{
	Use:   "wipe",
	Short: "Wipe all sync data and start fresh",
	Long: `Delete all synced data from Charm cloud and local KV store.

This is the nuclear option - use when:
- Sync data is corrupted
- You want to start completely fresh
- You're cleaning up after development/testing

After wipe, your local notes will be re-synced on next operation.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if charmClient == nil {
			return fmt.Errorf("not connected to Charm - run 'memo sync link' first")
		}

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

		if err := charmClient.Reset(); err != nil {
			return fmt.Errorf("wipe failed: %w", err)
		}

		color.Green("✓ All sync data wiped")
		fmt.Println("\nRun any memo command to start fresh.")

		return nil
	},
}

func init() {
	syncLinkCmd.Flags().String("host", "", "Charm server host (default: cloud.charm.sh)")

	syncCmd.AddCommand(syncStatusCmd)
	syncCmd.AddCommand(syncLinkCmd)
	syncCmd.AddCommand(syncUnlinkCmd)
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
