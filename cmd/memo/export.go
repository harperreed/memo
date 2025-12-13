// ABOUTME: Export command for backing up notes.
// ABOUTME: Supports JSON and markdown export formats.

package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/harper/memo/internal/db"
	"github.com/harper/memo/internal/ui"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

type ExportNote struct {
	ID          string             `json:"id" yaml:"id"`
	Title       string             `json:"title" yaml:"title"`
	Content     string             `json:"content" yaml:"content"`
	Tags        []string           `json:"tags" yaml:"tags"`
	CreatedAt   time.Time          `json:"created_at" yaml:"created"`
	UpdatedAt   time.Time          `json:"updated_at" yaml:"updated"`
	Attachments []ExportAttachment `json:"attachments,omitempty" yaml:"-"`
}

type ExportAttachment struct {
	ID       string `json:"id"`
	Filename string `json:"filename"`
	MimeType string `json:"mime_type"`
	Data     string `json:"data"` // base64 encoded
}

type ExportData struct {
	ExportedAt time.Time    `json:"exported_at"`
	Version    string       `json:"version"`
	Notes      []ExportNote `json:"notes"`
}

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export notes",
	Long:  `Export notes to JSON or markdown format.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		format, _ := cmd.Flags().GetString("format")
		outputPath, _ := cmd.Flags().GetString("output")
		notePrefix, _ := cmd.Flags().GetString("note")

		var notes []*db.SearchResult
		var err error

		if notePrefix != "" {
			note, err := db.GetNoteByPrefix(dbConn, notePrefix)
			if err != nil {
				return fmt.Errorf("failed to get note: %w", err)
			}
			notes = append(notes, &db.SearchResult{Note: note})
		} else {
			allNotes, err := db.ListNotes(dbConn, nil, 10000)
			if err != nil {
				return fmt.Errorf("failed to list notes: %w", err)
			}
			for _, n := range allNotes {
				notes = append(notes, &db.SearchResult{Note: n})
			}
		}

		switch format {
		case "json":
			return exportJSON(notes, outputPath)
		case "md":
			return exportMarkdown(notes, outputPath)
		default:
			return fmt.Errorf("unknown format: %s", format)
		}
	},
}

func exportJSON(notes []*db.SearchResult, outputPath string) error {
	export := ExportData{
		ExportedAt: time.Now(),
		Version:    "1.0",
	}

	for _, n := range notes {
		tags, _ := db.GetNoteTags(dbConn, n.ID)
		attachments, _ := db.ListNoteAttachments(dbConn, n.ID)

		en := ExportNote{
			ID:        n.ID.String(),
			Title:     n.Title,
			Content:   n.Content,
			CreatedAt: n.CreatedAt,
			UpdatedAt: n.UpdatedAt,
		}

		for _, t := range tags {
			en.Tags = append(en.Tags, t.Name)
		}

		for _, a := range attachments {
			att, _ := db.GetAttachment(dbConn, a.ID)
			if att != nil {
				en.Attachments = append(en.Attachments, ExportAttachment{
					ID:       att.ID.String(),
					Filename: att.Filename,
					MimeType: att.MimeType,
					Data:     base64.StdEncoding.EncodeToString(att.Data),
				})
			}
		}

		export.Notes = append(export.Notes, en)
	}

	data, err := json.MarshalIndent(export, "", "  ")
	if err != nil {
		return err
	}

	if outputPath == "" || outputPath == "-" {
		fmt.Println(string(data))
		return nil
	}

	return os.WriteFile(outputPath, data, 0644)
}

func exportMarkdown(notes []*db.SearchResult, outputDir string) error {
	if outputDir == "" {
		outputDir = "export"
	}

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return err
	}

	for _, n := range notes {
		tags, _ := db.GetNoteTags(dbConn, n.ID)
		attachments, _ := db.ListNoteAttachments(dbConn, n.ID)

		en := ExportNote{
			ID:        n.ID.String(),
			Title:     n.Title,
			CreatedAt: n.CreatedAt,
			UpdatedAt: n.UpdatedAt,
		}
		for _, t := range tags {
			en.Tags = append(en.Tags, t.Name)
		}

		// Write markdown file with frontmatter
		var sb strings.Builder
		sb.WriteString("---\n")

		frontmatter, _ := yaml.Marshal(en)
		sb.Write(frontmatter)
		sb.WriteString("---\n\n")
		sb.WriteString(n.Content)

		filename := sanitizeFilename(n.Title) + ".md"
		filePath := filepath.Join(outputDir, filename)
		if err := os.WriteFile(filePath, []byte(sb.String()), 0644); err != nil {
			return err
		}

		// Export attachments
		if len(attachments) > 0 {
			attDir := filepath.Join(outputDir, "attachments", n.ID.String()[:8])
			os.MkdirAll(attDir, 0755)

			for _, a := range attachments {
				att, _ := db.GetAttachment(dbConn, a.ID)
				if att != nil {
					attPath := filepath.Join(attDir, att.Filename)
					os.WriteFile(attPath, att.Data, 0644)
				}
			}
		}
	}

	fmt.Println(ui.Success(fmt.Sprintf("Exported %d notes to %s", len(notes), outputDir)))
	return nil
}

func sanitizeFilename(name string) string {
	// Replace unsafe characters
	replacer := strings.NewReplacer(
		"/", "-", "\\", "-", ":", "-", "*", "-",
		"?", "-", "\"", "-", "<", "-", ">", "-", "|", "-",
	)
	name = replacer.Replace(name)
	if len(name) > 100 {
		name = name[:100]
	}
	return name
}

func init() {
	exportCmd.Flags().StringP("format", "f", "json", "export format (json|md)")
	exportCmd.Flags().StringP("output", "o", "", "output path")
	exportCmd.Flags().StringP("note", "n", "", "single note ID to export")
	rootCmd.AddCommand(exportCmd)
}
