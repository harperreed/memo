// ABOUTME: List command for displaying notes.
// ABOUTME: Supports filtering by tag, search, and directory-aware output.

package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/harper/memo/internal/charm"
	"github.com/harper/memo/internal/models"
	"github.com/harper/memo/internal/ui"
	"github.com/spf13/cobra"
)

const defaultGlobalLimit = 10

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List notes",
	Long:  `List all notes, optionally filtered by tag or search query. By default shows directory-specific notes first, then global notes.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		tagFlag, _ := cmd.Flags().GetString("tag")
		searchFlag, _ := cmd.Flags().GetString("search")
		limitFlag, _ := cmd.Flags().GetInt("limit")
		hereFlag, _ := cmd.Flags().GetBool("here")

		// Search mode - bypass sectioned output
		if searchFlag != "" {
			return listSearch(searchFlag, limitFlag)
		}

		// Tag filter mode - bypass sectioned output
		if tagFlag != "" {
			return listByTag(tagFlag, limitFlag)
		}

		// Here mode - only show pwd-tagged notes
		if hereFlag {
			return listHere(limitFlag)
		}

		// Default: sectioned output (pwd + global)
		return listSectioned(limitFlag)
	},
}

func listSearch(query string, limit int) error {
	filter := &charm.NoteFilter{
		Search: query,
		Limit:  limit,
	}
	notes, err := charmClient.ListNotes(filter)
	if err != nil {
		return fmt.Errorf("search failed: %w", err)
	}

	if len(notes) == 0 {
		fmt.Println("No notes found.")
		return nil
	}

	for _, note := range notes {
		tags, _ := charmClient.GetNoteTags(note.ID)
		fmt.Print(ui.FormatNoteListItem(note, tagsToModels(tags)))
	}
	return nil
}

func listByTag(tagName string, limit int) error {
	filter := &charm.NoteFilter{
		Tag:   &tagName,
		Limit: limit,
	}
	notes, err := charmClient.ListNotes(filter)
	if err != nil {
		return fmt.Errorf("failed to list notes: %w", err)
	}

	if len(notes) == 0 {
		fmt.Println("No notes found.")
		return nil
	}

	for _, note := range notes {
		tags, _ := charmClient.GetNoteTags(note.ID)
		fmt.Print(ui.FormatNoteListItem(note, tagsToModels(tags)))
	}
	return nil
}

func listHere(limit int) error {
	pwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	filter := &charm.NoteFilter{
		DirTag: &pwd,
		Limit:  limit,
	}
	notes, err := charmClient.ListNotes(filter)
	if err != nil {
		return fmt.Errorf("failed to list notes: %w", err)
	}

	if len(notes) == 0 {
		fmt.Println("No notes found for this directory.")
		return nil
	}

	fmt.Print(ui.FormatDirSectionHeader(pwd))
	for _, note := range notes {
		tags, _ := charmClient.GetNoteTags(note.ID)
		fmt.Print(ui.FormatNoteListItem(note, tagsToModels(tags)))
	}
	return nil
}

//nolint:funlen,nestif // Complex flow for sectioned listing
func listSectioned(limit int) error {
	pwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Get directory-specific notes
	dirFilter := &charm.NoteFilter{
		DirTag: &pwd,
		Limit:  limit,
	}
	dirNotes, err := charmClient.ListNotes(dirFilter)
	if err != nil {
		return fmt.Errorf("failed to list directory notes: %w", err)
	}

	// Get global notes (no dir: tag)
	globalFilter := &charm.NoteFilter{
		Global: true,
		Limit:  defaultGlobalLimit,
	}
	globalNotes, err := charmClient.ListNotes(globalFilter)
	if err != nil {
		return fmt.Errorf("failed to list global notes: %w", err)
	}

	// Get total count for "show more" logic
	totalGlobal, err := charmClient.CountGlobalNotes()
	if err != nil {
		return fmt.Errorf("failed to count global notes: %w", err)
	}

	// Handle empty case
	if len(dirNotes) == 0 && len(globalNotes) == 0 {
		fmt.Println("No notes found.")
		return nil
	}

	// Print directory section if there are notes
	if len(dirNotes) > 0 {
		fmt.Print(ui.FormatDirSectionHeader(pwd))
		for _, note := range dirNotes {
			tags, _ := charmClient.GetNoteTags(note.ID)
			fmt.Print(ui.FormatNoteListItem(note, tagsToModels(tags)))
		}
	}

	// Print global section
	if len(globalNotes) > 0 {
		fmt.Print(ui.FormatGlobalSectionHeader())
		for _, note := range globalNotes {
			tags, _ := charmClient.GetNoteTags(note.ID)
			fmt.Print(ui.FormatNoteListItem(note, tagsToModels(tags)))
		}

		// Show more prompt if there are more global notes
		remaining := totalGlobal - len(globalNotes)
		if remaining > 0 {
			fmt.Print(ui.FormatShowMorePrompt(remaining))

			reader := bufio.NewReader(os.Stdin)
			response, err := reader.ReadString('\n')
			if err != nil {
				// EOF or input error - just don't show more
				return nil //nolint:nilerr // Intentional: silently exit on stdin issues
			}

			response = strings.TrimSpace(strings.ToLower(response))
			if response == "y" || response == "yes" {
				// Fetch remaining notes
				allGlobalFilter := &charm.NoteFilter{
					Global: true,
					Limit:  totalGlobal,
				}
				allGlobal, err := charmClient.ListNotes(allGlobalFilter)
				if err != nil {
					return fmt.Errorf("failed to list remaining notes: %w", err)
				}

				// Print only the ones we haven't shown yet
				fmt.Println()
				for i := defaultGlobalLimit; i < len(allGlobal); i++ {
					note := allGlobal[i]
					tags, _ := charmClient.GetNoteTags(note.ID)
					fmt.Print(ui.FormatNoteListItem(note, tagsToModels(tags)))
				}
			}
		}
	}

	return nil
}

// tagsToModels converts string tags to model tags for UI formatting.
func tagsToModels(tags []string) []*models.Tag {
	result := make([]*models.Tag, len(tags))
	for i, t := range tags {
		result[i] = models.NewTag(t)
	}
	return result
}

func init() {
	listCmd.Flags().StringP("tag", "t", "", "filter by tag")
	listCmd.Flags().StringP("search", "s", "", "search query")
	listCmd.Flags().IntP("limit", "n", 20, "number of results")
	listCmd.Flags().Bool("here", false, "show only notes tagged with current directory")
	rootCmd.AddCommand(listCmd)
}
