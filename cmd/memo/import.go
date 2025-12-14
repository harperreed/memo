// ABOUTME: Import command for restoring notes from backup.
// ABOUTME: Supports JSON and markdown directory import.

package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/harper/memo/internal/db"
	"github.com/harper/memo/internal/models"
	"github.com/harper/memo/internal/ui"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var importCmd = &cobra.Command{
	Use:   "import <path>",
	Short: "Import notes",
	Long:  `Import notes from a JSON file or directory of markdown files.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		path := args[0]

		info, err := os.Stat(path)
		if err != nil {
			return fmt.Errorf("failed to stat path: %w", err)
		}

		if info.IsDir() {
			return importMarkdownDir(path)
		}

		if strings.HasSuffix(path, ".json") {
			return importJSON(path)
		}

		return importMarkdownFile(path)
	},
}

func importJSON(path string) error {
	data, err := os.ReadFile(path) //nolint:gosec // User-specified file path is expected CLI behavior
	if err != nil {
		return err
	}

	var export ExportData
	if err := json.Unmarshal(data, &export); err != nil {
		return err
	}

	count := 0
	for _, en := range export.Notes {
		note := models.NewNote(en.Title, en.Content)
		// Try to preserve original ID if valid
		if id, err := uuid.Parse(en.ID); err == nil {
			note.ID = id
		}
		note.CreatedAt = en.CreatedAt
		note.UpdatedAt = en.UpdatedAt

		if err := db.CreateNote(dbConn, note); err != nil {
			fmt.Printf("Warning: failed to import %q: %v\n", en.Title, err)
			continue
		}

		for _, tagName := range en.Tags {
			if err := db.AddTagToNote(dbConn, note.ID, tagName); err != nil {
				fmt.Printf("Warning: failed to add tag %q: %v\n", tagName, err)
			}
		}

		for _, att := range en.Attachments {
			data, _ := base64.StdEncoding.DecodeString(att.Data)
			attachment := models.NewAttachment(note.ID, att.Filename, att.MimeType, data)
			if id, err := uuid.Parse(att.ID); err == nil {
				attachment.ID = id
			}
			if err := db.CreateAttachment(dbConn, attachment); err != nil {
				fmt.Printf("Warning: failed to create attachment %q: %v\n", att.Filename, err)
			}
		}

		count++
	}

	fmt.Println(ui.Success(fmt.Sprintf("Imported %d notes", count)))
	return nil
}

func importMarkdownDir(dir string) error {
	count := 0

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !strings.HasSuffix(path, ".md") {
			return nil
		}

		if err := importMarkdownFile(path); err != nil {
			fmt.Printf("Warning: failed to import %s: %v\n", path, err)
			return nil
		}
		count++
		return nil
	})

	if err != nil {
		return err
	}

	fmt.Println(ui.Success(fmt.Sprintf("Imported %d notes", count)))
	return nil
}

func importMarkdownFile(path string) error {
	data, err := os.ReadFile(path) //nolint:gosec // User-specified file path is expected CLI behavior
	if err != nil {
		return err
	}

	content := string(data)
	var title string
	var tags []string

	// Try to parse frontmatter
	if strings.HasPrefix(content, "---\n") {
		parts := strings.SplitN(content, "---\n", 3)
		if len(parts) >= 3 {
			var frontmatter struct {
				Title string   `yaml:"title"`
				Tags  []string `yaml:"tags"`
			}
			if err := yaml.Unmarshal([]byte(parts[1]), &frontmatter); err == nil {
				title = frontmatter.Title
				tags = frontmatter.Tags
				content = parts[2]
			}
		}
	}

	if title == "" {
		title = strings.TrimSuffix(filepath.Base(path), ".md")
	}

	content = strings.TrimSpace(content)
	if content == "" {
		return fmt.Errorf("note content cannot be empty")
	}

	note := models.NewNote(title, content)
	if err := db.CreateNote(dbConn, note); err != nil {
		return err
	}

	for _, tag := range tags {
		if err := db.AddTagToNote(dbConn, note.ID, tag); err != nil {
			fmt.Printf("Warning: failed to add tag %q: %v\n", tag, err)
		}
	}

	return nil
}

func init() {
	rootCmd.AddCommand(importCmd)
}
