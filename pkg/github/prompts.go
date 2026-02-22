package github

import (
	"github.com/github/omnigit-mcp/pkg/inventory"
	"github.com/github/omnigit-mcp/pkg/translations"
)

// AllPrompts returns all prompts with their embedded toolset metadata.
// Prompt functions return ServerPrompt directly with toolset info.
func AllPrompts(t translations.TranslationHelperFunc) []inventory.ServerPrompt {
	return []inventory.ServerPrompt{
		// Issue prompts
		AssignCodingAgentPrompt(t),
		IssueToFixWorkflowPrompt(t),
	}
}
