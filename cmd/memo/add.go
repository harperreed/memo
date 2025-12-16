// ABOUTME: Add command for creating new notes.
// ABOUTME: Supports inline content, file input, or $EDITOR.

package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"suitesync/vault"

	"github.com/harper/memo/internal/db"
	"github.com/harper/memo/internal/models"
	"github.com/harper/memo/internal/sync"
	"github.com/harper/memo/internal/ui"
	"github.com/spf13/cobra"
)

var addCmd = &cobra.Command{
	Use:   "add <title>",
	Short: "Add a new note",
	Long:  `Create a new note with the given title. Content can be provided via --content, --file, or $EDITOR.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		title := args[0]

		tagsFlag, _ := cmd.Flags().GetString("tags")
		contentFlag, _ := cmd.Flags().GetString("content")
		fileFlag, _ := cmd.Flags().GetString("file")
		hereFlag, _ := cmd.Flags().GetBool("here")

		var content string
		var err error

		switch {
		case contentFlag != "":
			content = contentFlag
		case fileFlag != "":
			data, err := os.ReadFile(fileFlag) //nolint:gosec // User-specified file path is expected CLI behavior
			if err != nil {
				return fmt.Errorf("failed to read file: %w", err)
			}
			content = string(data)
		default:
			content, err = openEditor("")
			if err != nil {
				return fmt.Errorf("failed to open editor: %w", err)
			}
		}

		if strings.TrimSpace(content) == "" {
			return fmt.Errorf("note content cannot be empty")
		}

		note := models.NewNote(title, content)
		if err := db.CreateNote(dbConn, note); err != nil {
			return fmt.Errorf("failed to create note: %w", err)
		}

		// Add tags if provided
		if tagsFlag != "" {
			for _, tag := range strings.Split(tagsFlag, ",") {
				tag = strings.TrimSpace(tag)
				if tag != "" {
					if err := db.AddTagToNote(dbConn, note.ID, tag); err != nil {
						return fmt.Errorf("failed to add tag %q: %w", tag, err)
					}
				}
			}
		}

		// Add directory tag if --here flag is set
		if hereFlag {
			pwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("failed to get current directory: %w", err)
			}
			dirTag := "dir:" + pwd
			if err := db.AddTagToNote(dbConn, note.ID, dirTag); err != nil {
				return fmt.Errorf("failed to add directory tag: %w", err)
			}
		}

		// Queue sync change (best-effort, won't fail if sync not configured)
		allTags := collectTags(tagsFlag, hereFlag)
		if err := sync.TryQueueNoteChange(
			context.Background(),
			dbConn,
			note.ID,
			note.Title,
			note.Content,
			allTags,
			note.CreatedAt,
			note.UpdatedAt,
			vault.OpUpsert,
		); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to queue sync: %v\n", err)
		}

		fmt.Println(ui.Success(fmt.Sprintf("Created note %s", note.ID.String()[:6])))
		return nil
	},
}

// collectTags gathers all tags that will be applied to a note.
func collectTags(tagsFlag string, hereFlag bool) []string {
	var tags []string
	if tagsFlag != "" {
		for _, tag := range strings.Split(tagsFlag, ",") {
			tag = strings.TrimSpace(tag)
			if tag != "" {
				tags = append(tags, strings.ToLower(tag))
			}
		}
	}
	if hereFlag {
		pwd, err := os.Getwd()
		if err == nil {
			tags = append(tags, strings.ToLower("dir:"+pwd))
		}
	}
	return tags
}

func openEditor(initial string) (string, error) {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vim"
	}

	tmpFile, err := os.CreateTemp("", "memo-*.md")
	if err != nil {
		return "", err
	}
	defer func() {
		_ = os.Remove(tmpFile.Name()) // Best-effort cleanup
	}()

	if initial != "" {
		if _, err := tmpFile.WriteString(initial); err != nil {
			_ = tmpFile.Close()
			return "", fmt.Errorf("failed to write initial content: %w", err)
		}
	}
	if err := tmpFile.Close(); err != nil {
		return "", fmt.Errorf("failed to close temp file: %w", err)
	}

	cmd := exec.Command(editor, tmpFile.Name()) //nolint:gosec // Launching $EDITOR is expected CLI behavior
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return "", err
	}

	data, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func init() {
	addCmd.Flags().String("tags", "", "comma-separated tags")
	addCmd.Flags().String("content", "", "note content (inline)")
	addCmd.Flags().String("file", "", "read content from file")
	addCmd.Flags().Bool("here", false, "tag note with current directory")
	rootCmd.AddCommand(addCmd)
}
