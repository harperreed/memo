// ABOUTME: Tag command for managing note tags.
// ABOUTME: Provides add, rm, and list subcommands.

package main

import (
	"fmt"

	"github.com/harper/memo/internal/db"
	"github.com/harper/memo/internal/ui"
	"github.com/spf13/cobra"
)

var tagCmd = &cobra.Command{
	Use:   "tag",
	Short: "Manage tags",
	Long:  `Add, remove, or list tags on notes.`,
}

var tagAddCmd = &cobra.Command{
	Use:   "add <id-prefix> <tag>",
	Short: "Add a tag to a note",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		prefix := args[0]
		tagName := args[1]

		note, err := db.GetNoteByPrefix(dbConn, prefix)
		if err != nil {
			return fmt.Errorf("failed to get note: %w", err)
		}

		if err := db.AddTagToNote(dbConn, note.ID, tagName); err != nil {
			return fmt.Errorf("failed to add tag: %w", err)
		}

		fmt.Println(ui.Success(fmt.Sprintf("Added tag %q to note %s", tagName, note.ID.String()[:6])))
		return nil
	},
}

var tagRmCmd = &cobra.Command{
	Use:   "rm <id-prefix> <tag>",
	Short: "Remove a tag from a note",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		prefix := args[0]
		tagName := args[1]

		note, err := db.GetNoteByPrefix(dbConn, prefix)
		if err != nil {
			return fmt.Errorf("failed to get note: %w", err)
		}

		if err := db.RemoveTagFromNote(dbConn, note.ID, tagName); err != nil {
			return fmt.Errorf("failed to remove tag: %w", err)
		}

		fmt.Println(ui.Success(fmt.Sprintf("Removed tag %q from note %s", tagName, note.ID.String()[:6])))
		return nil
	},
}

var tagListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all tags",
	RunE: func(cmd *cobra.Command, args []string) error {
		tags, err := db.ListAllTags(dbConn)
		if err != nil {
			return fmt.Errorf("failed to list tags: %w", err)
		}

		if len(tags) == 0 {
			fmt.Println("No tags found.")
			return nil
		}

		var tagCounts []ui.TagCount
		for _, t := range tags {
			tagCounts = append(tagCounts, ui.TagCount{
				Name:  t.Tag.Name,
				Count: t.Count,
			})
		}
		fmt.Print(ui.FormatTagList(tagCounts))
		return nil
	},
}

func init() {
	tagCmd.AddCommand(tagAddCmd)
	tagCmd.AddCommand(tagRmCmd)
	tagCmd.AddCommand(tagListCmd)
	rootCmd.AddCommand(tagCmd)
}
