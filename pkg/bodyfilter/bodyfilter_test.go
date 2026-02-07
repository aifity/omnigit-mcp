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
