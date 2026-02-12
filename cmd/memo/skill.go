// ABOUTME: Install Claude Code skill for memo
// ABOUTME: Embeds and installs the skill definition to ~/.claude/skills/

package main

import (
	"bufio"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

//go:embed skill/SKILL.md
var skillFS embed.FS

var skillSkipConfirm bool

var installSkillCmd = &cobra.Command{
	Use:   "install-skill",
	Short: "Install Claude Code skill",
	Long: `Install the memo skill for Claude Code.

This copies the skill definition to ~/.claude/skills/memo/
so Claude Code can use memo commands contextually.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return installSkill()
	},
}

func init() {
	installSkillCmd.Flags().BoolVarP(&skillSkipConfirm, "yes", "y", false, "Skip confirmation prompt")
	rootCmd.AddCommand(installSkillCmd)
}

func installSkill() error {
	return installSkillWithHome("")
}

// installSkillWithHome installs the skill using a custom home directory.
// If homeDir is empty, it uses os.UserHomeDir().
func installSkillWithHome(homeDir string) error {
	// Determine destination
	var err error
	if homeDir == "" {
		homeDir, err = os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
	}

	skillDir := filepath.Join(homeDir, ".claude", "skills", "memo")
	skillPath := filepath.Join(skillDir, "SKILL.md")

	// Show explanation
	fmt.Println("┌─────────────────────────────────────────────────────────────┐")
	fmt.Println("│              Memo Skill for Claude Code                     │")
	fmt.Println("└─────────────────────────────────────────────────────────────┘")
	fmt.Println()
	fmt.Println("This will install the memo skill, enabling Claude Code to:")
	fmt.Println()
	fmt.Println("  • Create and search markdown notes")
	fmt.Println("  • Organize notes with tags")
	fmt.Println("  • Quickly capture thoughts and ideas")
	fmt.Println("  • Use the /memo slash command")
	fmt.Println()
	fmt.Println("Destination:")
	fmt.Printf("  %s\n", skillPath)
	fmt.Println()

	// Check if already installed
	if _, err := os.Stat(skillPath); err == nil {
		fmt.Println("Note: A skill file already exists and will be overwritten.")
		fmt.Println()
	}

	// Ask for confirmation unless --yes flag is set
	if !skillSkipConfirm {
		fmt.Print("Install the memo skill? [y/N] ")
		reader := bufio.NewReader(os.Stdin)
		response, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read response: %w", err)
		}
		response = strings.TrimSpace(strings.ToLower(response))
		if response != "y" && response != "yes" {
			fmt.Println("Installation canceled.")
			return nil
		}
		fmt.Println()
	}

	// Read embedded skill file
	content, err := skillFS.ReadFile("skill/SKILL.md")
	if err != nil {
		return fmt.Errorf("failed to read embedded skill: %w", err)
	}

	// Create directory
	if err := os.MkdirAll(skillDir, 0750); err != nil {
		return fmt.Errorf("failed to create skill directory: %w", err)
	}

	// Write skill file
	if err := os.WriteFile(skillPath, content, 0600); err != nil {
		return fmt.Errorf("failed to write skill file: %w", err)
	}

	fmt.Println("✓ Installed memo skill successfully!")
	fmt.Println()
	fmt.Println("Claude Code will now recognize /memo commands.")
	fmt.Println("Try asking Claude: \"Save a note about today's meeting\" or \"Find my notes on Go\"")
	return nil
}
