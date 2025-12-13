// ABOUTME: FTS5 full-text search operations for notes.
// ABOUTME: Provides ranked search across note titles and content.

package db

import (
	"database/sql"

	"github.com/google/uuid"
	"github.com/harper/memo/internal/models"
)

type SearchResult struct {
	*models.Note
	Rank float64
}

func SearchNotes(db *sql.DB, query string, limit int) ([]*SearchResult, error) {
	rows, err := db.Query(
		`SELECT n.id, n.title, n.content, n.created_at, n.updated_at, rank
		 FROM notes_fts
		 JOIN notes n ON notes_fts.rowid = n.rowid
		 WHERE notes_fts MATCH ?
		 ORDER BY rank
		 LIMIT ?`,
		query, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*SearchResult
	for rows.Next() {
		result := &SearchResult{Note: &models.Note{}}
		var idStr string
		if err := rows.Scan(&idStr, &result.Title, &result.Content, &result.CreatedAt, &result.UpdatedAt, &result.Rank); err != nil {
			return nil, err
		}
		result.ID, _ = uuid.Parse(idStr)
		results = append(results, result)
	}
	return results, nil
}
