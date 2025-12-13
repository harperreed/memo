// ABOUTME: MCP prompts for common note-taking workflows.
// ABOUTME: Provides pre-configured prompts for AI agent interactions.

package mcp

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func (s *Server) registerPrompts() {
	// Register individual prompts - SDK will automatically handle listing
	s.server.AddPrompt(&mcp.Prompt{
		Name:        "create-meeting-notes",
		Description: "Create structured meeting notes with attendees, agenda, and action items",
		Arguments: []*mcp.PromptArgument{
			{
				Name:        "meeting_title",
				Description: "Title of the meeting",
				Required:    true,
			},
		},
	}, s.getMeetingNotesPrompt)

	s.server.AddPrompt(&mcp.Prompt{
		Name:        "create-daily-journal",
		Description: "Create a daily journal entry with prompts for reflection",
		Arguments: []*mcp.PromptArgument{
			{
				Name:        "date",
				Description: "Date for the journal entry (YYYY-MM-DD)",
				Required:    false,
			},
		},
	}, s.getDailyJournalPrompt)

	s.server.AddPrompt(&mcp.Prompt{
		Name:        "summarize-note",
		Description: "Generate a summary of an existing note",
		Arguments: []*mcp.PromptArgument{
			{
				Name:        "note_id",
				Description: "ID of the note to summarize",
				Required:    true,
			},
		},
	}, s.getSummarizeNotePrompt)

	s.server.AddPrompt(&mcp.Prompt{
		Name:        "organize-notes",
		Description: "Get suggestions for organizing and tagging notes",
	}, s.getOrganizeNotesPrompt)

	s.server.AddPrompt(&mcp.Prompt{
		Name:        "create-project-note",
		Description: "Create a project planning note with goals, tasks, and milestones",
		Arguments: []*mcp.PromptArgument{
			{
				Name:        "project_name",
				Description: "Name of the project",
				Required:    true,
			},
		},
	}, s.getProjectNotePrompt)
}

func (s *Server) getMeetingNotesPrompt(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	meetingTitle, ok := req.Params.Arguments["meeting_title"]
	if !ok || meetingTitle == "" {
		meetingTitle = "Meeting"
	}

	template := fmt.Sprintf(`Create meeting notes for: %s

Please structure the notes with the following sections:

## Attendees
- [List attendees]

## Agenda
1. [Topic 1]
2. [Topic 2]

## Discussion Notes
[Key points discussed]

## Decisions Made
- [Decision 1]
- [Decision 2]

## Action Items
- [ ] [Action 1] - @owner - Due: [date]
- [ ] [Action 2] - @owner - Due: [date]

## Next Steps
[What happens next]

Use the add_note tool to create this note with appropriate tags like "meeting", "work".`, meetingTitle)

	return &mcp.GetPromptResult{
		Messages: []*mcp.PromptMessage{
			{
				Role: "user",
				Content: &mcp.TextContent{
					Text: template,
				},
			},
		},
	}, nil
}

func (s *Server) getDailyJournalPrompt(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	date, ok := req.Params.Arguments["date"]
	if !ok || date == "" {
		date = "today"
	}

	template := fmt.Sprintf(`Create a daily journal entry for %s.

Please include reflections on:

## Today's Highlights
- What went well today?
- What am I grateful for?

## Challenges
- What difficulties did I face?
- What can I learn from them?

## Progress
- What did I accomplish?
- What progress did I make on my goals?

## Tomorrow's Focus
- What are my top 3 priorities?
- What do I want to accomplish?

Use the add_note tool to create this journal entry with tags like "journal", "daily-notes".`, date)

	return &mcp.GetPromptResult{
		Messages: []*mcp.PromptMessage{
			{
				Role: "user",
				Content: &mcp.TextContent{
					Text: template,
				},
			},
		},
	}, nil
}

func (s *Server) getSummarizeNotePrompt(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	noteID, ok := req.Params.Arguments["note_id"]
	if !ok || noteID == "" {
		return nil, fmt.Errorf("note_id argument is required")
	}

	template := fmt.Sprintf(`Please summarize the note with ID: %s

1. Use the get_note tool to retrieve the note content
2. Read and analyze the note
3. Create a concise summary highlighting:
   - Main topic or theme
   - Key points or takeaways
   - Important details or action items
4. Use the update_note tool to add a "Summary" section at the top of the note`, noteID)

	return &mcp.GetPromptResult{
		Messages: []*mcp.PromptMessage{
			{
				Role: "user",
				Content: &mcp.TextContent{
					Text: template,
				},
			},
		},
	}, nil
}

func (s *Server) getOrganizeNotesPrompt(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	template := `Help me organize my notes by:

1. Use the list_notes tool to see all my notes
2. Analyze the content and identify common themes
3. Suggest a tagging system that would help categorize them
4. Recommend which notes could be merged or split
5. Identify notes that might need updating or archiving

Please provide specific recommendations with note IDs and suggested tags.`

	return &mcp.GetPromptResult{
		Messages: []*mcp.PromptMessage{
			{
				Role: "user",
				Content: &mcp.TextContent{
					Text: template,
				},
			},
		},
	}, nil
}

func (s *Server) getProjectNotePrompt(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	projectName, ok := req.Params.Arguments["project_name"]
	if !ok || projectName == "" {
		projectName = "Project"
	}

	template := fmt.Sprintf(`Create a project planning note for: %s

Please structure the note with:

## Project Overview
[Brief description of the project]

## Goals
- [ ] Goal 1
- [ ] Goal 2
- [ ] Goal 3

## Key Milestones
1. [Milestone 1] - [Date]
2. [Milestone 2] - [Date]
3. [Milestone 3] - [Date]

## Tasks
- [ ] Task 1
- [ ] Task 2
- [ ] Task 3

## Resources
- [Resource 1]
- [Resource 2]

## Notes
[Additional context or considerations]

Use the add_note tool to create this project note with tags like "project", "planning".`, projectName)

	return &mcp.GetPromptResult{
		Messages: []*mcp.PromptMessage{
			{
				Role: "user",
				Content: &mcp.TextContent{
					Text: template,
				},
			},
		},
	}, nil
}
