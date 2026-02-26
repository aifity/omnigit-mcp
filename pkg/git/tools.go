// Package git provides local Git repository tools for the GitHub MCP Server.
//
// This code is adapted from git-mcp-go by Gero Posmyk-Leinemann and contributors.
// Original source: https://github.com/geropl/git-mcp-go
// Copyright (c) Gero Posmyk-Leinemann <gero@gitpod.io>
package git

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/aifity/omnigit-mcp/pkg/git/gitops"
	"github.com/aifity/omnigit-mcp/pkg/inventory"
	"github.com/aifity/omnigit-mcp/pkg/translations"
	"github.com/aifity/omnigit-mcp/pkg/utils"
	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ToolsetMetadataLocalGit defines the local git toolset metadata
var ToolsetMetadataLocalGit = inventory.ToolsetMetadata{
	ID:          "local_git",
	Description: "Local Git repository operations - work with git repositories on your local machine (status, diff, commit, push, pull, branches, etc.)",
	Icon:        "git-branch",
}

// ToolDependencies defines the dependencies needed by git tools
type ToolDependencies interface {
	GetGitOps() gitops.GitOperations
	GetRepoPaths() []string
}

// gitDepsContextKey is the context key for ToolDependencies.
// Using a private type prevents collisions with other packages.
type gitDepsContextKey struct{}

// ErrGitDepsNotInContext is returned when ToolDependencies is not found in context.
var ErrGitDepsNotInContext = errors.New("ToolDependencies not found in context; use ContextWithGitDeps to inject")

// ContextWithGitDeps returns a new context with the ToolDependencies stored in it.
// This is used to inject dependencies at request time rather than at registration time,
// avoiding expensive closure creation during server initialization.
func ContextWithGitDeps(ctx context.Context, deps ToolDependencies) context.Context {
	return context.WithValue(ctx, gitDepsContextKey{}, deps)
}

// MustGitDepsFromContext extracts ToolDependencies from context.
// Panics if deps are not found - callers must ensure ContextWithGitDeps was called.
func MustGitDepsFromContext(ctx context.Context) ToolDependencies {
	deps, ok := ctx.Value(gitDepsContextKey{}).(ToolDependencies)
	if !ok {
		panic(ErrGitDepsNotInContext)
	}
	return deps
}

// newToolFromHandler creates a ServerTool that retrieves ToolDependencies from context at call time.
// Use this when you have a handler that conforms to mcp.ToolHandler directly.
//
// The handler function receives deps extracted from context via MustGitDepsFromContext.
// Ensure ContextWithGitDeps is called to inject deps before any tool handlers are invoked.
func newToolFromHandler(
	tool mcp.Tool,
	handler func(ctx context.Context, deps ToolDependencies, req *mcp.CallToolRequest) (*mcp.CallToolResult, error),
) inventory.ServerTool {
	return inventory.NewServerToolWithRawContextHandler(tool, ToolsetMetadataLocalGit, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		deps := MustGitDepsFromContext(ctx)
		return handler(ctx, deps, req)
	})
}

// validateRepoPath validates and normalizes a repository path
func validateRepoPath(requestedPath string, allowedPaths []string) (string, error) {
	// If no specific path is provided, but we have repositories configured
	if requestedPath == "" {
		if len(allowedPaths) > 0 {
			// Use the first repository as default
			return allowedPaths[0], nil
		}
		return "", fmt.Errorf("no repository specified and no defaults configured")
	}

	// Always convert to absolute path first
	absPath, err := filepath.Abs(requestedPath)
	if err != nil {
		return "", fmt.Errorf("invalid path: %w", err)
	}

	// Check if path is within allowed repositories
	isAllowed := false
	for _, repoPath := range allowedPaths {
		if strings.HasPrefix(absPath, repoPath) {
			isAllowed = true
			break
		}
	}

	if !isAllowed && len(allowedPaths) > 0 {
		return "", fmt.Errorf("access denied - path outside allowed repositories: %s", absPath)
	}

	// Ensure it's a valid git repository
	gitDirPath := filepath.Join(absPath, ".git")
	if _, err := os.Stat(gitDirPath); os.IsNotExist(err) {
		return "", fmt.Errorf("not a git repository: %s", absPath)
	}

	return absPath, nil
}

// Status creates a tool to show the working tree status
func Status(t translations.TranslationHelperFunc) inventory.ServerTool {
	return newToolFromHandler(
		mcp.Tool{
			Name:        "git_status",
			Description: t("TOOL_GIT_STATUS_DESCRIPTION", "Shows the working tree status of a local Git repository"),
			Annotations: &mcp.ToolAnnotations{
				Title:        t("TOOL_GIT_STATUS_USER_TITLE", "Git status"),
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"repo_path": {
						Type:        "string",
						Description: "Path to Git repository (optional if default repository is configured)",
					},
				},
			},
		},
		func(_ context.Context, gitDeps ToolDependencies, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			// Unmarshal arguments
			var args map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &args); err != nil {
				return utils.NewToolResultError(fmt.Sprintf("Failed to parse arguments: %v", err)), nil
			}

			requestedPath := ""
			if val, ok := args["repo_path"].(string); ok {
				requestedPath = val
			}

			repoPath, err := validateRepoPath(requestedPath, gitDeps.GetRepoPaths())
			if err != nil {
				return utils.NewToolResultError(fmt.Sprintf("Repository path error: %v", err)), nil
			}

			status, err := gitDeps.GetGitOps().GetStatus(repoPath)
			if err != nil {
				return utils.NewToolResultError(fmt.Sprintf("Failed to get status: %v", err)), nil
			}

			return utils.NewToolResultText(fmt.Sprintf("Repository status for %s:\n%s", repoPath, status)), nil
		},
	)
}

// DiffUnstaged creates a tool to show unstaged changes
func DiffUnstaged(t translations.TranslationHelperFunc) inventory.ServerTool {
	return newToolFromHandler(
		mcp.Tool{
			Name:        "git_diff_unstaged",
			Description: t("TOOL_GIT_DIFF_UNSTAGED_DESCRIPTION", "Shows changes in the working directory that are not yet staged"),
			Annotations: &mcp.ToolAnnotations{
				Title:        t("TOOL_GIT_DIFF_UNSTAGED_USER_TITLE", "Git diff unstaged"),
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"repo_path": {
						Type:        "string",
						Description: "Path to Git repository (optional if default repository is configured)",
					},
				},
			},
		},
		func(_ context.Context, gitDeps ToolDependencies, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			// Unmarshal arguments
			var args map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &args); err != nil {
				return utils.NewToolResultError(fmt.Sprintf("Failed to parse arguments: %v", err)), nil
			}

			requestedPath := ""
			if val, ok := args["repo_path"].(string); ok {
				requestedPath = val
			}

			repoPath, err := validateRepoPath(requestedPath, gitDeps.GetRepoPaths())
			if err != nil {
				return utils.NewToolResultError(fmt.Sprintf("Repository path error: %v", err)), nil
			}

			diff, err := gitDeps.GetGitOps().GetDiffUnstaged(repoPath)
			if err != nil {
				return utils.NewToolResultError(fmt.Sprintf("Failed to get unstaged diff: %v", err)), nil
			}

			return utils.NewToolResultText(fmt.Sprintf("Unstaged changes for %s:\n%s", repoPath, diff)), nil
		},
	)
}

// DiffStaged creates a tool to show staged changes
func DiffStaged(t translations.TranslationHelperFunc) inventory.ServerTool {
	return newToolFromHandler(
		mcp.Tool{
			Name:        "git_diff_staged",
			Description: t("TOOL_GIT_DIFF_STAGED_DESCRIPTION", "Shows changes that are staged for commit"),
			Annotations: &mcp.ToolAnnotations{
				Title:        t("TOOL_GIT_DIFF_STAGED_USER_TITLE", "Git diff staged"),
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"repo_path": {
						Type:        "string",
						Description: "Path to Git repository (optional if default repository is configured)",
					},
				},
			},
		},
		func(_ context.Context, gitDeps ToolDependencies, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			// Unmarshal arguments
			var args map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &args); err != nil {
				return utils.NewToolResultError(fmt.Sprintf("Failed to parse arguments: %v", err)), nil
			}

			requestedPath := ""
			if val, ok := args["repo_path"].(string); ok {
				requestedPath = val
			}

			repoPath, err := validateRepoPath(requestedPath, gitDeps.GetRepoPaths())
			if err != nil {
				return utils.NewToolResultError(fmt.Sprintf("Repository path error: %v", err)), nil
			}

			diff, err := gitDeps.GetGitOps().GetDiffStaged(repoPath)
			if err != nil {
				return utils.NewToolResultError(fmt.Sprintf("Failed to get staged diff: %v", err)), nil
			}

			return utils.NewToolResultText(fmt.Sprintf("Staged changes for %s:\n%s", repoPath, diff)), nil
		},
	)
}

// Diff creates a tool to show differences between branches or commits
func Diff(t translations.TranslationHelperFunc) inventory.ServerTool {
	return newToolFromHandler(
		mcp.Tool{
			Name:        "git_diff",
			Description: t("TOOL_GIT_DIFF_DESCRIPTION", "Shows differences between branches or commits"),
			Annotations: &mcp.ToolAnnotations{
				Title:        t("TOOL_GIT_DIFF_USER_TITLE", "Git diff"),
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"repo_path": {
						Type:        "string",
						Description: "Path to Git repository (optional if default repository is configured)",
					},
					"target": {
						Type:        "string",
						Description: "Target branch or commit to compare with",
					},
				},
				Required: []string{"target"},
			},
		},
		func(_ context.Context, gitDeps ToolDependencies, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			// Unmarshal arguments
			var args map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &args); err != nil {
				return utils.NewToolResultError(fmt.Sprintf("Failed to parse arguments: %v", err)), nil
			}

			requestedPath := ""
			if val, ok := args["repo_path"].(string); ok {
				requestedPath = val
			}

			repoPath, err := validateRepoPath(requestedPath, gitDeps.GetRepoPaths())
			if err != nil {
				return utils.NewToolResultError(fmt.Sprintf("Repository path error: %v", err)), nil
			}

			target, ok := args["target"].(string)
			if !ok {
				return utils.NewToolResultError("target must be a string"), nil
			}

			diff, err := gitDeps.GetGitOps().GetDiff(repoPath, target)
			if err != nil {
				return utils.NewToolResultError(fmt.Sprintf("Failed to get diff: %v", err)), nil
			}

			return utils.NewToolResultText(fmt.Sprintf("Diff with %s for %s:\n%s", target, repoPath, diff)), nil
		},
	)
}

// Commit creates a tool to commit changes
func Commit(t translations.TranslationHelperFunc) inventory.ServerTool {
	return newToolFromHandler(
		mcp.Tool{
			Name:        "git_commit",
			Description: t("TOOL_GIT_COMMIT_DESCRIPTION", "Records changes to the repository"),
			Annotations: &mcp.ToolAnnotations{
				Title:        t("TOOL_GIT_COMMIT_USER_TITLE", "Git commit"),
				ReadOnlyHint: false,
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"repo_path": {
						Type:        "string",
						Description: "Path to Git repository (optional if default repository is configured)",
					},
					"message": {
						Type:        "string",
						Description: "Commit message",
					},
				},
				Required: []string{"message"},
			},
		},
		func(_ context.Context, gitDeps ToolDependencies, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			// Unmarshal arguments
			var args map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &args); err != nil {
				return utils.NewToolResultError(fmt.Sprintf("Failed to parse arguments: %v", err)), nil
			}

			requestedPath := ""
			if val, ok := args["repo_path"].(string); ok {
				requestedPath = val
			}

			repoPath, err := validateRepoPath(requestedPath, gitDeps.GetRepoPaths())
			if err != nil {
				return utils.NewToolResultError(fmt.Sprintf("Repository path error: %v", err)), nil
			}

			message, ok := args["message"].(string)
			if !ok {
				return utils.NewToolResultError("message must be a string"), nil
			}

			result, err := gitDeps.GetGitOps().CommitChanges(repoPath, message)
			if err != nil {
				return utils.NewToolResultError(fmt.Sprintf("Failed to commit: %v", err)), nil
			}

			return utils.NewToolResultText(result), nil
		},
	)
}

// Add creates a tool to add files to staging area
func Add(t translations.TranslationHelperFunc) inventory.ServerTool {
	return newToolFromHandler(
		mcp.Tool{
			Name:        "git_add",
			Description: t("TOOL_GIT_ADD_DESCRIPTION", "Adds file contents to the staging area"),
			Annotations: &mcp.ToolAnnotations{
				Title:        t("TOOL_GIT_ADD_USER_TITLE", "Git add"),
				ReadOnlyHint: false,
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"repo_path": {
						Type:        "string",
						Description: "Path to Git repository (optional if default repository is configured)",
					},
					"files": {
						Type:        "string",
						Description: "Comma-separated list of file paths to stage",
					},
				},
				Required: []string{"files"},
			},
		},
		func(_ context.Context, gitDeps ToolDependencies, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			// Unmarshal arguments
			var argsMap map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &argsMap); err != nil {
				return utils.NewToolResultError(fmt.Sprintf("Failed to parse arguments: %v", err)), nil
			}

			requestedPath := ""
			if val, ok := argsMap["repo_path"].(string); ok {
				requestedPath = val
			}

			repoPath, err := validateRepoPath(requestedPath, gitDeps.GetRepoPaths())
			if err != nil {
				return utils.NewToolResultError(fmt.Sprintf("Repository path error: %v", err)), nil
			}

			filesStr, ok := argsMap["files"].(string)
			if !ok {
				return utils.NewToolResultError("files must be a string"), nil
			}

			// Support either single file, comma-separated, or space-delimited
			var files []string
			switch {
			case strings.Contains(filesStr, ","):
				files = strings.Split(filesStr, ",")
				for i, file := range files {
					files[i] = strings.TrimSpace(file)
				}
			case strings.Contains(filesStr, " "):
				files = strings.Split(filesStr, " ")
				for i, file := range files {
					files[i] = strings.TrimSpace(file)
				}
			default:
				files = []string{filesStr}
			}

			result, err := gitDeps.GetGitOps().AddFiles(repoPath, files)
			if err != nil {
				return utils.NewToolResultError(fmt.Sprintf("Failed to add files: %v", err)), nil
			}

			return utils.NewToolResultText(result), nil
		},
	)
}

// Reset creates a tool to unstage changes
func Reset(t translations.TranslationHelperFunc) inventory.ServerTool {
	return newToolFromHandler(
		mcp.Tool{
			Name:        "git_reset",
			Description: t("TOOL_GIT_RESET_DESCRIPTION", "Unstages all staged changes"),
			Annotations: &mcp.ToolAnnotations{
				Title:        t("TOOL_GIT_RESET_USER_TITLE", "Git reset"),
				ReadOnlyHint: false,
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"repo_path": {
						Type:        "string",
						Description: "Path to Git repository (optional if default repository is configured)",
					},
				},
			},
		},
		func(_ context.Context, gitDeps ToolDependencies, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			// Unmarshal arguments
			var args map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &args); err != nil {
				return utils.NewToolResultError(fmt.Sprintf("Failed to parse arguments: %v", err)), nil
			}

			requestedPath := ""
			if val, ok := args["repo_path"].(string); ok {
				requestedPath = val
			}

			repoPath, err := validateRepoPath(requestedPath, gitDeps.GetRepoPaths())
			if err != nil {
				return utils.NewToolResultError(fmt.Sprintf("Repository path error: %v", err)), nil
			}

			result, err := gitDeps.GetGitOps().ResetStaged(repoPath)
			if err != nil {
				return utils.NewToolResultError(fmt.Sprintf("Failed to reset: %v", err)), nil
			}

			return utils.NewToolResultText(result), nil
		},
	)
}

// Log creates a tool to show commit logs
func Log(t translations.TranslationHelperFunc) inventory.ServerTool {
	return newToolFromHandler(
		mcp.Tool{
			Name:        "git_log",
			Description: t("TOOL_GIT_LOG_DESCRIPTION", "Shows the commit logs"),
			Annotations: &mcp.ToolAnnotations{
				Title:        t("TOOL_GIT_LOG_USER_TITLE", "Git log"),
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"repo_path": {
						Type:        "string",
						Description: "Path to Git repository (optional if default repository is configured)",
					},
					"max_count": {
						Type:        "number",
						Description: "Maximum number of commits to show (default: 10)",
					},
				},
			},
		},
		func(_ context.Context, gitDeps ToolDependencies, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			// Unmarshal arguments
			var args map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &args); err != nil {
				return utils.NewToolResultError(fmt.Sprintf("Failed to parse arguments: %v", err)), nil
			}

			requestedPath := ""
			if val, ok := args["repo_path"].(string); ok {
				requestedPath = val
			}

			repoPath, err := validateRepoPath(requestedPath, gitDeps.GetRepoPaths())
			if err != nil {
				return utils.NewToolResultError(fmt.Sprintf("Repository path error: %v", err)), nil
			}

			maxCount := 10
			if maxCountInterface, ok := args["max_count"]; ok {
				if maxCountFloat, ok := maxCountInterface.(float64); ok {
					maxCount = int(maxCountFloat)
				}
			}

			logs, err := gitDeps.GetGitOps().GetLog(repoPath, maxCount)
			if err != nil {
				return utils.NewToolResultError(fmt.Sprintf("Failed to get log: %v", err)), nil
			}

			return utils.NewToolResultText(fmt.Sprintf("Commit history for %s:\n%s", repoPath, strings.Join(logs, "\n"))), nil
		},
	)
}

// CreateBranch creates a tool to create a new branch
func CreateBranch(t translations.TranslationHelperFunc) inventory.ServerTool {
	return newToolFromHandler(
		mcp.Tool{
			Name:        "git_create_branch",
			Description: t("TOOL_GIT_CREATE_BRANCH_DESCRIPTION", "Creates a new branch from an optional base branch and automatically checks it out"),
			Annotations: &mcp.ToolAnnotations{
				Title:        t("TOOL_GIT_CREATE_BRANCH_USER_TITLE", "Git create branch"),
				ReadOnlyHint: false,
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"repo_path": {
						Type:        "string",
						Description: "Path to Git repository (optional if default repository is configured)",
					},
					"branch_name": {
						Type:        "string",
						Description: "Name of the new branch",
					},
					"base_branch": {
						Type:        "string",
						Description: "Starting point for the new branch (optional)",
					},
				},
				Required: []string{"branch_name"},
			},
		},
		func(_ context.Context, gitDeps ToolDependencies, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			// Unmarshal arguments
			var args map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &args); err != nil {
				return utils.NewToolResultError(fmt.Sprintf("Failed to parse arguments: %v", err)), nil
			}

			requestedPath := ""
			if val, ok := args["repo_path"].(string); ok {
				requestedPath = val
			}

			repoPath, err := validateRepoPath(requestedPath, gitDeps.GetRepoPaths())
			if err != nil {
				return utils.NewToolResultError(fmt.Sprintf("Repository path error: %v", err)), nil
			}

			branchName, ok := args["branch_name"].(string)
			if !ok {
				return utils.NewToolResultError("branch_name must be a string"), nil
			}

			baseBranch := ""
			if baseBranchInterface, ok := args["base_branch"]; ok {
				if baseBranchStr, ok := baseBranchInterface.(string); ok {
					baseBranch = baseBranchStr
				}
			}

			result, err := gitDeps.GetGitOps().CreateBranch(repoPath, branchName, baseBranch)
			if err != nil {
				return utils.NewToolResultError(fmt.Sprintf("Failed to create branch: %v", err)), nil
			}

			return utils.NewToolResultText(result), nil
		},
	)
}

// Checkout creates a tool to switch branches
func Checkout(t translations.TranslationHelperFunc) inventory.ServerTool {
	return newToolFromHandler(
		mcp.Tool{
			Name:        "git_checkout",
			Description: t("TOOL_GIT_CHECKOUT_DESCRIPTION", "Switches branches"),
			Annotations: &mcp.ToolAnnotations{
				Title:        t("TOOL_GIT_CHECKOUT_USER_TITLE", "Git checkout"),
				ReadOnlyHint: false,
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"repo_path": {
						Type:        "string",
						Description: "Path to Git repository (optional if default repository is configured)",
					},
					"branch_name": {
						Type:        "string",
						Description: "Name of branch to checkout",
					},
				},
				Required: []string{"branch_name"},
			},
		},
		func(_ context.Context, gitDeps ToolDependencies, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			// Unmarshal arguments
			var args map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &args); err != nil {
				return utils.NewToolResultError(fmt.Sprintf("Failed to parse arguments: %v", err)), nil
			}

			requestedPath := ""
			if val, ok := args["repo_path"].(string); ok {
				requestedPath = val
			}

			repoPath, err := validateRepoPath(requestedPath, gitDeps.GetRepoPaths())
			if err != nil {
				return utils.NewToolResultError(fmt.Sprintf("Repository path error: %v", err)), nil
			}

			branchName, ok := args["branch_name"].(string)
			if !ok {
				return utils.NewToolResultError("branch_name must be a string"), nil
			}

			result, err := gitDeps.GetGitOps().CheckoutBranch(repoPath, branchName)
			if err != nil {
				return utils.NewToolResultError(fmt.Sprintf("Failed to checkout branch: %v", err)), nil
			}

			return utils.NewToolResultText(result), nil
		},
	)
}

// Show creates a tool to show commit contents
func Show(t translations.TranslationHelperFunc) inventory.ServerTool {
	return newToolFromHandler(
		mcp.Tool{
			Name:        "git_show",
			Description: t("TOOL_GIT_SHOW_DESCRIPTION", "Shows the contents of a commit"),
			Annotations: &mcp.ToolAnnotations{
				Title:        t("TOOL_GIT_SHOW_USER_TITLE", "Git show"),
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"repo_path": {
						Type:        "string",
						Description: "Path to Git repository (optional if default repository is configured)",
					},
					"revision": {
						Type:        "string",
						Description: "The revision (commit hash, branch name, tag) to show",
					},
				},
				Required: []string{"revision"},
			},
		},
		func(_ context.Context, gitDeps ToolDependencies, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			// Unmarshal arguments
			var args map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &args); err != nil {
				return utils.NewToolResultError(fmt.Sprintf("Failed to parse arguments: %v", err)), nil
			}

			requestedPath := ""
			if val, ok := args["repo_path"].(string); ok {
				requestedPath = val
			}

			repoPath, err := validateRepoPath(requestedPath, gitDeps.GetRepoPaths())
			if err != nil {
				return utils.NewToolResultError(fmt.Sprintf("Repository path error: %v", err)), nil
			}

			revision, ok := args["revision"].(string)
			if !ok {
				return utils.NewToolResultError("revision must be a string"), nil
			}

			result, err := gitDeps.GetGitOps().ShowCommit(repoPath, revision)
			if err != nil {
				return utils.NewToolResultError(fmt.Sprintf("Failed to show commit: %v", err)), nil
			}

			return utils.NewToolResultText(result), nil
		},
	)
}

// Init creates a tool to initialize a new repository
func Init(t translations.TranslationHelperFunc) inventory.ServerTool {
	return newToolFromHandler(
		mcp.Tool{
			Name:        "git_init",
			Description: t("TOOL_GIT_INIT_DESCRIPTION", "Initialize a new Git repository"),
			Annotations: &mcp.ToolAnnotations{
				Title:        t("TOOL_GIT_INIT_USER_TITLE", "Git init"),
				ReadOnlyHint: false,
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"repo_path": {
						Type:        "string",
						Description: "Path to directory to initialize git repo",
					},
				},
				Required: []string{"repo_path"},
			},
		},
		func(_ context.Context, gitDeps ToolDependencies, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			// Unmarshal arguments
			var args map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &args); err != nil {
				return utils.NewToolResultError(fmt.Sprintf("Failed to parse arguments: %v", err)), nil
			}

			requestedPath, ok := args["repo_path"].(string)
			if !ok || requestedPath == "" {
				return utils.NewToolResultError("repo_path must be specified for initialization"), nil
			}

			// Ensure the path is absolute
			absPath, err := filepath.Abs(requestedPath)
			if err != nil {
				return utils.NewToolResultError(fmt.Sprintf("Failed to get absolute path: %v", err)), nil
			}

			result, err := gitDeps.GetGitOps().InitRepo(absPath)
			if err != nil {
				return utils.NewToolResultError(fmt.Sprintf("Failed to initialize repository: %v", err)), nil
			}

			return utils.NewToolResultText(result), nil
		},
	)
}

// Push creates a tool to push changes to remote
func Push(t translations.TranslationHelperFunc) inventory.ServerTool {
	return newToolFromHandler(
		mcp.Tool{
			Name:        "git_push",
			Description: t("TOOL_GIT_PUSH_DESCRIPTION", "Pushes local commits to a remote repository and automatically sets up tracking"),
			Annotations: &mcp.ToolAnnotations{
				Title:        t("TOOL_GIT_PUSH_USER_TITLE", "Git push"),
				ReadOnlyHint: false,
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"repo_path": {
						Type:        "string",
						Description: "Path to Git repository (optional if default repository is configured)",
					},
					"remote": {
						Type:        "string",
						Description: "Remote name (default: origin)",
					},
					"branch": {
						Type:        "string",
						Description: "Branch name to push (default: current branch)",
					},
				},
			},
		},
		func(_ context.Context, gitDeps ToolDependencies, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			// Unmarshal arguments
			var args map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &args); err != nil {
				return utils.NewToolResultError(fmt.Sprintf("Failed to parse arguments: %v", err)), nil
			}

			requestedPath := ""
			if val, ok := args["repo_path"].(string); ok {
				requestedPath = val
			}

			repoPath, err := validateRepoPath(requestedPath, gitDeps.GetRepoPaths())
			if err != nil {
				return utils.NewToolResultError(fmt.Sprintf("Repository path error: %v", err)), nil
			}

			remote := ""
			if remoteInterface, ok := args["remote"]; ok {
				if remoteStr, ok := remoteInterface.(string); ok {
					remote = remoteStr
				}
			}

			branch := ""
			if branchInterface, ok := args["branch"]; ok {
				if branchStr, ok := branchInterface.(string); ok {
					branch = branchStr
				}
			}

			result, err := gitDeps.GetGitOps().PushChanges(repoPath, remote, branch)
			if err != nil {
				return utils.NewToolResultError(fmt.Sprintf("Failed to push changes: %v", err)), nil
			}

			return utils.NewToolResultText(result), nil
		},
	)
}

// Pull creates a tool to pull changes from remote
func Pull(t translations.TranslationHelperFunc) inventory.ServerTool {
	return newToolFromHandler(
		mcp.Tool{
			Name:        "git_pull",
			Description: t("TOOL_GIT_PULL_DESCRIPTION", "Pulls changes from a remote repository with automatic rebase and prune"),
			Annotations: &mcp.ToolAnnotations{
				Title:        t("TOOL_GIT_PULL_USER_TITLE", "Git pull"),
				ReadOnlyHint: false,
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"repo_path": {
						Type:        "string",
						Description: "Path to Git repository (optional if default repository is configured)",
					},
					"remote": {
						Type:        "string",
						Description: "Remote name (default: origin)",
					},
					"branch": {
						Type:        "string",
						Description: "Branch name to pull (default: current branch's upstream)",
					},
				},
			},
		},
		func(_ context.Context, gitDeps ToolDependencies, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			// Unmarshal arguments
			var args map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &args); err != nil {
				return utils.NewToolResultError(fmt.Sprintf("Failed to parse arguments: %v", err)), nil
			}

			requestedPath := ""
			if val, ok := args["repo_path"].(string); ok {
				requestedPath = val
			}

			repoPath, err := validateRepoPath(requestedPath, gitDeps.GetRepoPaths())
			if err != nil {
				return utils.NewToolResultError(fmt.Sprintf("Repository path error: %v", err)), nil
			}

			remote := ""
			if remoteInterface, ok := args["remote"]; ok {
				if remoteStr, ok := remoteInterface.(string); ok {
					remote = remoteStr
				}
			}

			branch := ""
			if branchInterface, ok := args["branch"]; ok {
				if branchStr, ok := branchInterface.(string); ok {
					branch = branchStr
				}
			}

			result, err := gitDeps.GetGitOps().PullChanges(repoPath, remote, branch)
			if err != nil {
				return utils.NewToolResultError(fmt.Sprintf("Failed to pull changes: %v", err)), nil
			}

			return utils.NewToolResultText(result), nil
		},
	)
}

// ListRepositories creates a tool to list all available repositories
func ListRepositories(t translations.TranslationHelperFunc) inventory.ServerTool {
	return newToolFromHandler(
		mcp.Tool{
			Name:        "git_list_repositories",
			Description: t("TOOL_GIT_LIST_REPOSITORIES_DESCRIPTION", "Lists all available Git repositories"),
			Annotations: &mcp.ToolAnnotations{
				Title:        t("TOOL_GIT_LIST_REPOSITORIES_USER_TITLE", "Git list repositories"),
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type:       "object",
				Properties: map[string]*jsonschema.Schema{},
			},
		},
		func(_ context.Context, gitDeps ToolDependencies, _ *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			repoPaths := gitDeps.GetRepoPaths()
			if len(repoPaths) == 0 {
				return utils.NewToolResultText("No repositories configured"), nil
			}

			var result strings.Builder
			fmt.Fprintf(&result, "Available repositories (%d):\n\n", len(repoPaths))

			for i, repoPath := range repoPaths {
				// Get the repository name (last part of the path)
				repoName := filepath.Base(repoPath)
				fmt.Fprintf(&result, "%d. %s (%s)\n", i+1, repoName, repoPath)
			}

			return utils.NewToolResultText(result.String()), nil
		},
	)
}

// ApplyPatchString creates a tool to apply a patch from a string
func ApplyPatchString(t translations.TranslationHelperFunc) inventory.ServerTool {
	return newToolFromHandler(
		mcp.Tool{
			Name:        "git_apply_patch_string",
			Description: t("TOOL_GIT_APPLY_PATCH_STRING_DESCRIPTION", "Applies a patch from a string to a git repository"),
			Annotations: &mcp.ToolAnnotations{
				Title:        t("TOOL_GIT_APPLY_PATCH_STRING_USER_TITLE", "Git apply patch string"),
				ReadOnlyHint: false,
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"repo_path": {
						Type:        "string",
						Description: "Path to Git repository (optional if default repository is configured)",
					},
					"patch_string": {
						Type:        "string",
						Description: "Patch string to apply",
					},
				},
				Required: []string{"patch_string"},
			},
		},
		func(_ context.Context, gitDeps ToolDependencies, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			// Unmarshal arguments
			var args map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &args); err != nil {
				return utils.NewToolResultError(fmt.Sprintf("Failed to parse arguments: %v", err)), nil
			}

			requestedPath := ""
			if val, ok := args["repo_path"].(string); ok {
				requestedPath = val
			}

			repoPath, err := validateRepoPath(requestedPath, gitDeps.GetRepoPaths())
			if err != nil {
				return utils.NewToolResultError(fmt.Sprintf("Repository path error: %v", err)), nil
			}

			patchString, ok := args["patch_string"].(string)
			if !ok {
				return utils.NewToolResultError("patch_string must be a string"), nil
			}

			if strings.TrimSpace(patchString) == "" {
				return utils.NewToolResultError("patch_string cannot be empty"), nil
			}

			result, err := gitDeps.GetGitOps().ApplyPatchFromString(repoPath, patchString)
			if err != nil {
				return utils.NewToolResultError(fmt.Sprintf("Failed to apply patch: %v", err)), nil
			}

			return utils.NewToolResultText(result), nil
		},
	)
}

// ApplyPatchFile creates a tool to apply a patch from a file
func ApplyPatchFile(t translations.TranslationHelperFunc) inventory.ServerTool {
	return newToolFromHandler(
		mcp.Tool{
			Name:        "git_apply_patch_file",
			Description: t("TOOL_GIT_APPLY_PATCH_FILE_DESCRIPTION", "Applies a patch from a file to a git repository"),
			Annotations: &mcp.ToolAnnotations{
				Title:        t("TOOL_GIT_APPLY_PATCH_FILE_USER_TITLE", "Git apply patch file"),
				ReadOnlyHint: false,
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"repo_path": {
						Type:        "string",
						Description: "Path to Git repository (optional if default repository is configured)",
					},
					"patch_file": {
						Type:        "string",
						Description: "Path to the patch file",
					},
				},
				Required: []string{"patch_file"},
			},
		},
		func(_ context.Context, gitDeps ToolDependencies, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			// Unmarshal arguments
			var args map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &args); err != nil {
				return utils.NewToolResultError(fmt.Sprintf("Failed to parse arguments: %v", err)), nil
			}

			requestedPath := ""
			if val, ok := args["repo_path"].(string); ok {
				requestedPath = val
			}

			repoPath, err := validateRepoPath(requestedPath, gitDeps.GetRepoPaths())
			if err != nil {
				return utils.NewToolResultError(fmt.Sprintf("Repository path error: %v", err)), nil
			}

			patchFile, ok := args["patch_file"].(string)
			if !ok {
				return utils.NewToolResultError("patch_file must be a string"), nil
			}

			if strings.TrimSpace(patchFile) == "" {
				return utils.NewToolResultError("patch_file cannot be empty"), nil
			}

			// Ensure the patch file exists
			absPath, err := filepath.Abs(patchFile)
			if err != nil {
				return utils.NewToolResultError(fmt.Sprintf("Invalid patch file path: %v", err)), nil
			}

			if _, err := os.Stat(absPath); os.IsNotExist(err) {
				return utils.NewToolResultError(fmt.Sprintf("Patch file does not exist: %s", absPath)), nil
			}

			result, err := gitDeps.GetGitOps().ApplyPatchFromFile(repoPath, absPath)
			if err != nil {
				return utils.NewToolResultError(fmt.Sprintf("Failed to apply patch: %v", err)), nil
			}

			return utils.NewToolResultText(result), nil
		},
	)
}

// ListWorktrees creates a tool to list all worktrees
func ListWorktrees(t translations.TranslationHelperFunc) inventory.ServerTool {
	return newToolFromHandler(
		mcp.Tool{
			Name:        "git_worktree_list",
			Description: t("TOOL_GIT_WORKTREE_LIST_DESCRIPTION", "Lists all worktrees in the repository with their paths and branches"),
			Annotations: &mcp.ToolAnnotations{
				Title:        t("TOOL_GIT_WORKTREE_LIST_USER_TITLE", "Git worktree list"),
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"repo_path": {
						Type:        "string",
						Description: "Path to Git repository (optional if default repository is configured)",
					},
				},
			},
		},
		func(_ context.Context, gitDeps ToolDependencies, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var args map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &args); err != nil {
				return utils.NewToolResultError(fmt.Sprintf("Failed to parse arguments: %v", err)), nil
			}

			requestedPath := ""
			if val, ok := args["repo_path"].(string); ok {
				requestedPath = val
			}

			repoPath, err := validateRepoPath(requestedPath, gitDeps.GetRepoPaths())
			if err != nil {
				return utils.NewToolResultError(fmt.Sprintf("Repository path error: %v", err)), nil
			}

			result, err := gitDeps.GetGitOps().ListWorktrees(repoPath)
			if err != nil {
				return utils.NewToolResultError(fmt.Sprintf("Failed to list worktrees: %v", err)), nil
			}

			return utils.NewToolResultText(result), nil
		},
	)
}

// AddWorktree creates a tool to add a new worktree
func AddWorktree(t translations.TranslationHelperFunc) inventory.ServerTool {
	return newToolFromHandler(
		mcp.Tool{
			Name:        "git_worktree_add",
			Description: t("TOOL_GIT_WORKTREE_ADD_DESCRIPTION", "Creates a new worktree at the specified path. A worktree allows you to have multiple working directories from the same repository"),
			Annotations: &mcp.ToolAnnotations{
				Title:        t("TOOL_GIT_WORKTREE_ADD_USER_TITLE", "Git worktree add"),
				ReadOnlyHint: false,
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"repo_path": {
						Type:        "string",
						Description: "Path to Git repository (optional if default repository is configured)",
					},
					"worktree_path": {
						Type:        "string",
						Description: "Path where the new worktree should be created",
					},
					"commitish": {
						Type:        "string",
						Description: "Branch name, tag, or commit SHA to checkout in the new worktree (optional)",
					},
					"new_branch": {
						Type:        "string",
						Description: "Create a new branch with this name in the worktree (optional)",
					},
					"detach": {
						Type:        "boolean",
						Description: "Create a detached HEAD worktree (optional, default: false)",
					},
					"force": {
						Type:        "boolean",
						Description: "Force creation even if worktree path already exists (optional, default: false)",
					},
				},
				Required: []string{"worktree_path"},
			},
		},
		func(_ context.Context, gitDeps ToolDependencies, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var args map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &args); err != nil {
				return utils.NewToolResultError(fmt.Sprintf("Failed to parse arguments: %v", err)), nil
			}

			requestedPath := ""
			if val, ok := args["repo_path"].(string); ok {
				requestedPath = val
			}

			repoPath, err := validateRepoPath(requestedPath, gitDeps.GetRepoPaths())
			if err != nil {
				return utils.NewToolResultError(fmt.Sprintf("Repository path error: %v", err)), nil
			}

			rawWorktreePath, ok := args["worktree_path"]
			if !ok {
				return utils.NewToolResultError("Missing required argument: worktree_path"), nil
			}

			worktreePathStr, ok := rawWorktreePath.(string)
			if !ok {
				return utils.NewToolResultError("Invalid type for worktree_path: expected string"), nil
			}

			worktreePath := strings.TrimSpace(worktreePathStr)
			if worktreePath == "" {
				return utils.NewToolResultError("Invalid worktree_path: value must be a non-empty string"), nil
			}

			commitish := ""
			if val, ok := args["commitish"].(string); ok {
				commitish = val
			}

			newBranch := ""
			if val, ok := args["new_branch"].(string); ok {
				newBranch = val
			}

			detach := false
			if val, ok := args["detach"].(bool); ok {
				detach = val
			}

			force := false
			if val, ok := args["force"].(bool); ok {
				force = val
			}

			// Build options array
			var options []string
			if newBranch != "" {
				options = append(options, "-b", newBranch)
			}
			if detach {
				options = append(options, "--detach")
			}
			if force {
				options = append(options, "--force")
			}

			result, err := gitDeps.GetGitOps().AddWorktree(repoPath, worktreePath, commitish, options)
			if err != nil {
				return utils.NewToolResultError(fmt.Sprintf("Failed to add worktree: %v", err)), nil
			}

			return utils.NewToolResultText(result), nil
		},
	)
}

// RemoveWorktree creates a tool to remove a worktree
func RemoveWorktree(t translations.TranslationHelperFunc) inventory.ServerTool {
	return newToolFromHandler(
		mcp.Tool{
			Name:        "git_worktree_remove",
			Description: t("TOOL_GIT_WORKTREE_REMOVE_DESCRIPTION", "Removes a worktree. The worktree must be clean (no uncommitted changes) unless force is used"),
			Annotations: &mcp.ToolAnnotations{
				Title:        t("TOOL_GIT_WORKTREE_REMOVE_USER_TITLE", "Git worktree remove"),
				ReadOnlyHint: false,
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"repo_path": {
						Type:        "string",
						Description: "Path to Git repository (optional if default repository is configured)",
					},
					"worktree": {
						Type:        "string",
						Description: "Path to the worktree directory to remove",
					},
					"force": {
						Type:        "boolean",
						Description: "Force removal even if worktree is dirty or locked (optional, default: false)",
					},
				},
				Required: []string{"worktree"},
			},
		},
		func(_ context.Context, gitDeps ToolDependencies, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var args map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &args); err != nil {
				return utils.NewToolResultError(fmt.Sprintf("Failed to parse arguments: %v", err)), nil
			}

			requestedPath := ""
			if val, ok := args["repo_path"].(string); ok {
				requestedPath = val
			}

			repoPath, err := validateRepoPath(requestedPath, gitDeps.GetRepoPaths())
			if err != nil {
				return utils.NewToolResultError(fmt.Sprintf("Repository path error: %v", err)), nil
			}

			rawWorktree, ok := args["worktree"]
			if !ok {
				return utils.NewToolResultError("Invalid 'worktree' argument: missing required field"), nil
			}

			worktreeStr, ok := rawWorktree.(string)
			if !ok {
				return utils.NewToolResultError("Invalid 'worktree' argument: must be a string"), nil
			}

			worktree := strings.TrimSpace(worktreeStr)
			if worktree == "" {
				return utils.NewToolResultError("Invalid 'worktree' argument: must be a non-empty string"), nil
			}

			force := false
			if val, ok := args["force"].(bool); ok {
				force = val
			}

			result, err := gitDeps.GetGitOps().RemoveWorktree(repoPath, worktree, force)
			if err != nil {
				return utils.NewToolResultError(fmt.Sprintf("Failed to remove worktree: %v", err)), nil
			}

			return utils.NewToolResultText(result), nil
		},
	)
}

// LockWorktree creates a tool to lock a worktree
func LockWorktree(t translations.TranslationHelperFunc) inventory.ServerTool {
	return newToolFromHandler(
		mcp.Tool{
			Name:        "git_worktree_lock",
			Description: t("TOOL_GIT_WORKTREE_LOCK_DESCRIPTION", "Locks a worktree to prevent it from being pruned or removed. Useful for worktrees on portable devices"),
			Annotations: &mcp.ToolAnnotations{
				Title:        t("TOOL_GIT_WORKTREE_LOCK_USER_TITLE", "Git worktree lock"),
				ReadOnlyHint: false,
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"repo_path": {
						Type:        "string",
						Description: "Path to Git repository (optional if default repository is configured)",
					},
					"worktree": {
						Type:        "string",
						Description: "Path to the worktree directory to lock",
					},
					"reason": {
						Type:        "string",
						Description: "Reason for locking the worktree (optional)",
					},
				},
				Required: []string{"worktree"},
			},
		},
		func(_ context.Context, gitDeps ToolDependencies, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var args map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &args); err != nil {
				return utils.NewToolResultError(fmt.Sprintf("Failed to parse arguments: %v", err)), nil
			}

			requestedPath := ""
			if val, ok := args["repo_path"].(string); ok {
				requestedPath = val
			}

			repoPath, err := validateRepoPath(requestedPath, gitDeps.GetRepoPaths())
			if err != nil {
				return utils.NewToolResultError(fmt.Sprintf("Repository path error: %v", err)), nil
			}

			rawWorktree, ok := args["worktree"]
			if !ok {
				return utils.NewToolResultError("'worktree' is required and must be a non-empty string"), nil
			}

			worktree, ok := rawWorktree.(string)
			if !ok || strings.TrimSpace(worktree) == "" {
				return utils.NewToolResultError("'worktree' is required and must be a non-empty string"), nil
			}
			worktree = strings.TrimSpace(worktree)

			reason := ""
			if val, ok := args["reason"].(string); ok {
				reason = val
			}

			result, err := gitDeps.GetGitOps().LockWorktree(repoPath, worktree, reason)
			if err != nil {
				return utils.NewToolResultError(fmt.Sprintf("Failed to lock worktree: %v", err)), nil
			}

			return utils.NewToolResultText(result), nil
		},
	)
}

// UnlockWorktree creates a tool to unlock a worktree
func UnlockWorktree(t translations.TranslationHelperFunc) inventory.ServerTool {
	return newToolFromHandler(
		mcp.Tool{
			Name:        "git_worktree_unlock",
			Description: t("TOOL_GIT_WORKTREE_UNLOCK_DESCRIPTION", "Unlocks a previously locked worktree, allowing it to be pruned or removed"),
			Annotations: &mcp.ToolAnnotations{
				Title:        t("TOOL_GIT_WORKTREE_UNLOCK_USER_TITLE", "Git worktree unlock"),
				ReadOnlyHint: false,
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"repo_path": {
						Type:        "string",
						Description: "Path to Git repository (optional if default repository is configured)",
					},
					"worktree": {
						Type:        "string",
						Description: "Path to the worktree directory to unlock",
					},
				},
				Required: []string{"worktree"},
			},
		},
		func(_ context.Context, gitDeps ToolDependencies, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var args map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &args); err != nil {
				return utils.NewToolResultError(fmt.Sprintf("Failed to parse arguments: %v", err)), nil
			}

			requestedPath := ""
			if val, ok := args["repo_path"].(string); ok {
				requestedPath = val
			}

			repoPath, err := validateRepoPath(requestedPath, gitDeps.GetRepoPaths())
			if err != nil {
				return utils.NewToolResultError(fmt.Sprintf("Repository path error: %v", err)), nil
			}

			worktreeVal, ok := args["worktree"]
			if !ok {
				return utils.NewToolResultError("Missing required 'worktree' argument"), nil
			}

			worktreeStr, ok := worktreeVal.(string)
			if !ok {
				return utils.NewToolResultError("Invalid 'worktree' argument: expected a string"), nil
			}

			worktree := strings.TrimSpace(worktreeStr)
			if worktree == "" {
				return utils.NewToolResultError("Invalid 'worktree' argument: must be a non-empty string"), nil
			}

			result, err := gitDeps.GetGitOps().UnlockWorktree(repoPath, worktree)
			if err != nil {
				return utils.NewToolResultError(fmt.Sprintf("Failed to unlock worktree: %v", err)), nil
			}

			return utils.NewToolResultText(result), nil
		},
	)
}

// PruneWorktrees creates a tool to prune worktree information
func PruneWorktrees(t translations.TranslationHelperFunc) inventory.ServerTool {
	return newToolFromHandler(
		mcp.Tool{
			Name:        "git_worktree_prune",
			Description: t("TOOL_GIT_WORKTREE_PRUNE_DESCRIPTION", "Removes stale worktree administrative files. Use this to clean up worktree metadata for worktrees that have been manually deleted"),
			Annotations: &mcp.ToolAnnotations{
				Title:        t("TOOL_GIT_WORKTREE_PRUNE_USER_TITLE", "Git worktree prune"),
				ReadOnlyHint: false,
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"repo_path": {
						Type:        "string",
						Description: "Path to Git repository (optional if default repository is configured)",
					},
					"dry_run": {
						Type:        "boolean",
						Description: "Show what would be pruned without actually pruning (optional, default: false)",
					},
					"verbose": {
						Type:        "boolean",
						Description: "Show verbose output (optional, default: false)",
					},
				},
			},
		},
		func(_ context.Context, gitDeps ToolDependencies, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var args map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &args); err != nil {
				return utils.NewToolResultError(fmt.Sprintf("Failed to parse arguments: %v", err)), nil
			}

			requestedPath := ""
			if val, ok := args["repo_path"].(string); ok {
				requestedPath = val
			}

			repoPath, err := validateRepoPath(requestedPath, gitDeps.GetRepoPaths())
			if err != nil {
				return utils.NewToolResultError(fmt.Sprintf("Repository path error: %v", err)), nil
			}

			dryRun := false
			if val, ok := args["dry_run"].(bool); ok {
				dryRun = val
			}

			verbose := false
			if val, ok := args["verbose"].(bool); ok {
				verbose = val
			}

			result, err := gitDeps.GetGitOps().PruneWorktrees(repoPath, dryRun, verbose)
			if err != nil {
				return utils.NewToolResultError(fmt.Sprintf("Failed to prune worktrees: %v", err)), nil
			}

			return utils.NewToolResultText(result), nil
		},
	)
}

// AllGitTools returns all git tools
func AllGitTools(t translations.TranslationHelperFunc) []inventory.ServerTool {
	return []inventory.ServerTool{
		Status(t),
		DiffUnstaged(t),
		DiffStaged(t),
		Diff(t),
		Commit(t),
		Add(t),
		Reset(t),
		Log(t),
		CreateBranch(t),
		Checkout(t),
		Show(t),
		Init(t),
		Push(t),
		Pull(t),
		ListRepositories(t),
		ApplyPatchString(t),
		ApplyPatchFile(t),
		ListWorktrees(t),
		AddWorktree(t),
		RemoveWorktree(t),
		LockWorktree(t),
		UnlockWorktree(t),
		PruneWorktrees(t),
	}
}
