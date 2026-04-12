package bodyfilter

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFilterBody(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "no patterns to filter",
			input:    "This is a normal PR description\n\nWith multiple lines",
			expected: "This is a normal PR description\n\nWith multiple lines",
		},
		{
			name:     "single Co-Authored-By line",
			input:    "Fix bug in authentication\n\nCo-Authored-By: John Doe <john@example.com>",
			expected: "Fix bug in authentication",
		},
		{
			name: "multiple Co-Authored-By lines",
			input: `Fix bug in authentication

Co-Authored-By: John Doe <john@example.com>
Co-Authored-By: Jane Smith <jane@example.com>`,
			expected: "Fix bug in authentication",
		},
		{
			name: "Co-Authored-By in middle of text",
			input: `This is the description

Co-Authored-By: John Doe <john@example.com>

More content here`,
			expected: "This is the description\n\nMore content here",
		},
		{
			name: "Co-Authored-By with different formats",
			input: `PR description

Co-Authored-By: John Doe <john@example.com>
Co-Authored-By: Jane Smith <jane.smith@company.org>
Co-Authored-By: Bob <bob123@test.co.uk>`,
			expected: "PR description",
		},
		{
			name:     "text containing 'Co-Authored-By' but not at line start",
			input:    "This mentions Co-Authored-By: but not at start",
			expected: "This mentions Co-Authored-By: but not at start",
		},
		{
			name: "multiple blank lines cleanup",
			input: `Description


Co-Authored-By: John Doe <john@example.com>


More text`,
			expected: "Description\n\nMore text",
		},
		{
			name: "real-world git commit message",
			input: `Add new feature for user authentication

This PR implements OAuth2 authentication flow.

Fixes #123

Co-Authored-By: Alice Developer <alice@example.com>
Co-Authored-By: Bob Reviewer <bob@example.com>`,
			expected: "Add new feature for user authentication\n\nThis PR implements OAuth2 authentication flow.\n\nFixes #123",
		},
		{
			name:     "only Co-Authored-By lines",
			input:    "Co-Authored-By: John Doe <john@example.com>\nCo-Authored-By: Jane Smith <jane@example.com>",
			expected: "",
		},
		{
			name: "preserves other trailers",
			input: `Fix critical bug

Signed-off-by: Developer <dev@example.com>
Co-Authored-By: John Doe <john@example.com>
Reviewed-by: Reviewer <reviewer@example.com>`,
			expected: "Fix critical bug\n\nSigned-off-by: Developer <dev@example.com>\n\nReviewed-by: Reviewer <reviewer@example.com>",
		},
		{
			name:     "whitespace only",
			input:    "   \n\n   ",
			expected: "",
		},
		{
			name: "Co-Authored-By with various spacing",
			input: `Description

Co-Authored-By:    John Doe    <john@example.com>
Co-Authored-By: Jane<jane@example.com>`,
			expected: "Description",
		},
		{
			name: "filters out John Doe Co-Authored-By",
			input: `Fix authentication bug

This PR fixes the authentication issue.

Co-Authored-By: John <john@doe.example>`,
			expected: "Fix authentication bug\n\nThis PR fixes the authentication issue.",
		},
		{
			name: "filters out John Doe with other co-authors",
			input: `Add new feature

Co-Authored-By: Developer <dev@example.com>
Co-Authored-By: John <john@doe.example>
Co-Authored-By: Reviewer <reviewer@example.com>`,
			expected: "Add new feature",
		},
		{
			name: "filters out generic PR footer - AI Coder",
			input: `Fix authentication bug

This PR fixes the authentication issue.

---
Pull Request opened by [AI Coder](https://aicoder.example/) with guidance from the PR author`,
			expected: "Fix authentication bug\n\nThis PR fixes the authentication issue.",
		},
		{
			name: "filters out generic PR footer with extra whitespace",
			input: `Add new feature

Implementation details here.

---
Pull Request opened by [DevBot](https://devbot.example/) with guidance from the PR author
`,
			expected: "Add new feature\n\nImplementation details here.",
		},
		{
			name: "filters out generic PR footer with other content",
			input: `Update documentation

Changes:
- Updated README
- Added examples

---
Pull Request opened by [CodeAssist](https://codeassist.example/) with guidance from the PR author

Co-Authored-By: Developer <dev@example.com>`,
			expected: "Update documentation\n\nChanges:\n- Updated README\n- Added examples",
		},
		{
			name: "filters out generic PR footer - different tool name",
			input: `Implement new feature

Feature description here.

---
Pull Request opened by [AI Assistant](https://example.com/) with guidance from the developer`,
			expected: "Implement new feature\n\nFeature description here.",
		},
		{
			name: "filters out generic PR footer - different URL format",
			input: `Bug fix

Fixed the issue.

---
Pull Request opened by [CodeBot](https://codebot.io/app) with guidance from the team`,
			expected: "Bug fix\n\nFixed the issue.",
		},
		{
			name: "filters out generic PR footer - minimal format",
			input: `Quick fix

---
Pull Request opened by [X](http://x.com) with guidance from Y`,
			expected: "Quick fix",
		},
		{
			name: "filters out Generated with line - with emoji and link",
			input: `Fix authentication bug

This PR fixes the authentication issue.

🤖 Generated with [Claude Code](https://claude.ai/claude-code)`,
			expected: "Fix authentication bug\n\nThis PR fixes the authentication issue.",
		},
		{
			name: "filters out Generated with line - without emoji, with link",
			input: `Add new feature

Implementation details.

Generated with [Claude Code](https://claude.ai/claude-code)`,
			expected: "Add new feature\n\nImplementation details.",
		},
		{
			name: "filters out Generated with line - with emoji, no link",
			input: `Update docs

🤖 Generated with Claude Code`,
			expected: "Update docs",
		},
		{
			name: "filters out Generated with line - no emoji, no link",
			input: `Bug fix

Generated with Claude Code`,
			expected: "Bug fix",
		},
		{
			name: "filters out Generated with line - any emoji and any tool/url",
			input: `Refactor code

✨ Generated with [MyAITool](https://myaitool.example.com/some/path)`,
			expected: "Refactor code",
		},
		{
			name: "filters out Generated with combined with Co-Authored-By",
			input: `Fix bug

Co-Authored-By: Claude <noreply@anthropic.com>
🤖 Generated with [Claude Code](https://claude.ai/claude-code)`,
			expected: "Fix bug",
		},
		{
			name: "filters out Warp.dev conversation and plan links",
			input: `Fix authentication issue

This PR addresses the bug in the login flow.

---
*Conversation: https://app.warp.dev/conversation/abc123-def4-5678-90gh-ijklmnopqrst*
*Plan: https://app.warp.dev/drive/notebook/xyz789FakeNotebookId*`,
			expected: "Fix authentication issue\n\nThis PR addresses the bug in the login flow.",
		},
		{
			name: "filters out Warp.dev links with different IDs",
			input: `Update documentation

---
*Conversation: https://app.warp.dev/conversation/fake-conversation-uuid-1234*
*Plan: https://app.warp.dev/drive/notebook/fakePlanId567*`,
			expected: "Update documentation",
		},
		{
			name: "filters out Warp.dev links combined with other patterns",
			input: `Implement new feature

Co-Authored-By: Developer <dev@example.com>
🤖 Generated with [Claude Code](https://claude.ai/claude-code)

---
*Conversation: https://app.warp.dev/conversation/example-conversation-id*
*Plan: https://app.warp.dev/drive/notebook/example-plan-id*`,
			expected: "Implement new feature",
		},
		{
			name: "filters out Warp.dev conversation link - new plain format without separator or asterisks",
			input: `Recreates the configurable buckets feature.

Supersedes #122.

Conversation: https://app.warp.dev/conversation/f2974b5a-e1ce-4428-b1ff-21735457acd5`,
			expected: "Recreates the configurable buckets feature.\n\nSupersedes #122.",
		},
		{
			name: "filters out Warp.dev conversation link with preceding --- but no asterisks or plan",
			input: `Fix a bug.

---
Conversation: https://app.warp.dev/conversation/abc123-def456`,
			expected: "Fix a bug.",
		},
		{
			name: "filters out Warp.dev conversation link only (no plan) with asterisks",
			input: `Add feature.

*Conversation: https://app.warp.dev/conversation/abc123*`,
			expected: "Add feature.",
		},
		{
			name: "filters out Warp.dev ## Artifacts block with list items",
			input: `Fix bug in login flow.

## Artifacts
- Conversation: https://app.warp.dev/conversation/7a680e91-3ac5-4cc6-a170-dd976dcabe33
- Plan: https://app.warp.dev/drive/notebook/yI1A8F434ZAhCpTfW3LANZ`,
			expected: "Fix bug in login flow.",
		},
		{
			name: "filters out Warp.dev ## Artifacts block - conversation only",
			input: `Update readme.

## Artifacts
- Conversation: https://app.warp.dev/conversation/abc123`,
			expected: "Update readme.",
		},
		{
			name: "filters out Warp.dev list-item links without heading",
			input: `Refactor auth.

- Conversation: https://app.warp.dev/conversation/abc123
- Plan: https://app.warp.dev/drive/notebook/planxyz`,
			expected: "Refactor auth.",
		},
		{
			name: "catch-all: removes any line containing a warp.dev URL",
			input: `Fix something.

See https://app.warp.dev/conversation/some-unknown-format for context.`,
			expected: "Fix something.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FilterBody(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFilterBodyIdempotent(t *testing.T) {
	input := `PR description

Co-Authored-By: John Doe <john@example.com>

More content`

	firstPass := FilterBody(input)
	secondPass := FilterBody(firstPass)

	assert.Equal(t, firstPass, secondPass, "FilterBody should be idempotent")
}

func TestFilterBodyPreservesFormatting(t *testing.T) {
	input := `# Title

## Description

This is a **bold** statement.

- Item 1
- Item 2

Co-Authored-By: John Doe <john@example.com>

## More sections

Content here.`

	expected := `# Title

## Description

This is a **bold** statement.

- Item 1
- Item 2

## More sections

Content here.`

	result := FilterBody(input)
	assert.Equal(t, expected, result)
}
