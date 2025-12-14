// ABOUTME: Database operations for tags and note-tag associations.
// ABOUTME: Provides tag creation, assignment, removal, and listing.

package db

import (
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/harper/memo/internal/models"
)

func GetOrCreateTag(db *sql.DB, name string) (*models.Tag, error) {
	tag := models.NewTag(name)

	// Try to get existing
	err := db.QueryRow(`SELECT id FROM tags WHERE name = ?`, tag.Name).Scan(&tag.ID)
	if err == nil {
		return tag, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	// Create new
	result, err := db.Exec(`INSERT INTO tags (name) VALUES (?)`, tag.Name)
	if err != nil {
		return nil, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}
	tag.ID = id
	return tag, nil
}

func AddTagToNote(db *sql.DB, noteID uuid.UUID, tagName string) error {
	tag, err := GetOrCreateTag(db, tagName)
	if err != nil {
		return err
	}

	_, err = db.Exec(
		`INSERT OR IGNORE INTO note_tags (note_id, tag_id) VALUES (?, ?)`,
		noteID.String(), tag.ID,
	)
	return err
}

func RemoveTagFromNote(db *sql.DB, noteID uuid.UUID, tagName string) error {
	tag := models.NewTag(tagName)
	_, err := db.Exec(
		`DELETE FROM note_tags WHERE note_id = ? AND tag_id = (SELECT id FROM tags WHERE name = ?)`,
		noteID.String(), tag.Name,
	)
	return err
}

func GetNoteTags(db *sql.DB, noteID uuid.UUID) ([]*models.Tag, error) {
	rows, err := db.Query(
		`SELECT t.id, t.name FROM tags t
		 JOIN note_tags nt ON t.id = nt.tag_id
		 WHERE nt.note_id = ?
		 ORDER BY t.name`,
		noteID.String(),
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var tags []*models.Tag
	for rows.Next() {
		tag := &models.Tag{}
		if err := rows.Scan(&tag.ID, &tag.Name); err != nil {
			return nil, err
		}
		tags = append(tags, tag)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return tags, nil
}

type TagWithCount struct {
	Tag   *models.Tag
	Count int
}

func ListAllTags(db *sql.DB) ([]*TagWithCount, error) {
	rows, err := db.Query(
		`SELECT t.id, t.name, COUNT(nt.note_id) as count
		 FROM tags t
		 LEFT JOIN note_tags nt ON t.id = nt.tag_id
		 GROUP BY t.id
		 ORDER BY t.name`,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var tags []*TagWithCount
	for rows.Next() {
		tc := &TagWithCount{Tag: &models.Tag{}}
		if err := rows.Scan(&tc.Tag.ID, &tc.Tag.Name, &tc.Count); err != nil {
			return nil, err
		}
		tags = append(tags, tc)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return tags, nil
}
