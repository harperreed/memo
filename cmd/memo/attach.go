// ABOUTME: Attach command for managing note attachments.
// ABOUTME: Provides add and get subcommands for binary files.

package main

import (
	"bytes"
	"fmt"
	"io"
	"mime"
	"os"
	"path/filepath"

	"github.com/harper/memo/internal/models"
	"github.com/harper/memo/internal/ui"
	"github.com/spf13/cobra"
)

var attachCmd = &cobra.Command{
	Use:   "attach <id-prefix> <file>",
	Short: "Add an attachment to a note",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		prefix := args[0]
		filePath := args[1]

		note, _, err := charmClient.GetNoteByPrefix(prefix)
		if err != nil {
			return fmt.Errorf("failed to get note: %w", err)
		}

		data, err := os.ReadFile(filePath) //nolint:gosec // User-specified file path is expected CLI behavior
		if err != nil {
			return fmt.Errorf("failed to read file: %w", err)
		}

		filename := filepath.Base(filePath)
		mimeType := mime.TypeByExtension(filepath.Ext(filePath))
		if mimeType == "" {
			mimeType = "application/octet-stream"
		}

		att := models.NewAttachment(note.ID, filename, mimeType, data)
		if err := charmClient.CreateAttachment(att); err != nil {
			return fmt.Errorf("failed to create attachment: %w", err)
		}

		fmt.Println(ui.Success(fmt.Sprintf("Added attachment %s to note %s", att.ID.String()[:6], note.ID.String()[:6])))
		return nil
	},
}

var attachGetCmd = &cobra.Command{
	Use:   "get <attachment-id-prefix>",
	Short: "Extract an attachment to a file",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		prefix := args[0]
		outputPath, _ := cmd.Flags().GetString("output")

		att, err := charmClient.GetAttachmentByPrefix(prefix)
		if err != nil {
			return fmt.Errorf("failed to get attachment: %w", err)
		}

		if outputPath == "" {
			outputPath = att.Filename
		}

		if outputPath == "-" {
			_, err = io.Copy(os.Stdout, bytes.NewReader(att.Data))
			return err
		}

		if err := os.WriteFile(outputPath, att.Data, 0600); err != nil {
			return fmt.Errorf("failed to write file: %w", err)
		}

		fmt.Println(ui.Success(fmt.Sprintf("Extracted %s to %s", att.Filename, outputPath)))
		return nil
	},
}

func init() {
	attachGetCmd.Flags().StringP("output", "o", "", "output path (default: original filename)")
	attachCmd.AddCommand(attachGetCmd)
	rootCmd.AddCommand(attachCmd)
}
