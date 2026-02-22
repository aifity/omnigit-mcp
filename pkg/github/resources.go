package github

import (
	"github.com/github/omnigit-mcp/pkg/inventory"
	"github.com/github/omnigit-mcp/pkg/translations"
)

// AllResources returns all resource templates with their embedded toolset metadata.
// Resource definitions are stateless - handlers are generated on-demand during registration.
func AllResources(t translations.TranslationHelperFunc) []inventory.ServerResourceTemplate {
	return []inventory.ServerResourceTemplate{
		// Repository resources
		GetRepositoryResourceContent(t),
		GetRepositoryResourceBranchContent(t),
		GetRepositoryResourceCommitContent(t),
		GetRepositoryResourceTagContent(t),
		GetRepositoryResourcePrContent(t),
	}
}
