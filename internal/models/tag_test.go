// ABOUTME: Tests for Tag model.
// ABOUTME: Validates tag creation and name normalization.

package models

import "testing"

func TestNewTag(t *testing.T) {
	tag := NewTag("TestTag")

	if tag.Name != "testtag" {
		t.Errorf("expected lowercase name 'testtag', got %q", tag.Name)
	}
}

func TestNewTagWithSpaces(t *testing.T) {
	tag := NewTag("  My Tag  ")

	if tag.Name != "my tag" {
		t.Errorf("expected trimmed lowercase 'my tag', got %q", tag.Name)
	}
}
