// ABOUTME: Tag model for categorizing notes.
// ABOUTME: Normalizes tag names to lowercase with trimmed whitespace.

package models

import "strings"

type Tag struct {
	ID   int64
	Name string
}

func NewTag(name string) *Tag {
	return &Tag{
		Name: strings.ToLower(strings.TrimSpace(name)),
	}
}
