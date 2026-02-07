// Package git provides local Git repository tools for the GitHub MCP Server.
//
// This code is adapted from git-mcp-go by Gero Posmyk-Leinemann and contributors.
// Original source: https://github.com/geropl/git-mcp-go
// Copyright (c) Gero Posmyk-Leinemann <gero@gitpod.io>
package git

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/github/github-mcp-server/pkg/git/gitops"
	"github.com/github/github-mcp-server/pkg/inventory"
	"github.com/github/github-mcp-server/pkg/translations"
	"github.com/github/github-mcp-server/pkg/utils"
	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ToolsetMetadataLocalGit defines the local git toolset metadata
var ToolsetMetadataLocalGit = inventory.ToolsetMetadata{
	ID:          "local_git",
	Description: "Local Git repository operations (adapted from git-mcp-go)",
	Icon:        "git-branch",
}

// GitToolDependencies defines the dependencies needed by git tools
type GitToolDependencies interface {
	GetGitOps() gitops.GitOperations
	GetRepoPaths() []string
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

// GitStatus creates a tool to show the working tree status
func GitStatus(t translations.TranslationHelperFunc) inventory.ServerTool {
	return inventory.NewServerToolFromHandler(
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
		ToolsetMetadataLocalGit,
		func(deps any) mcp.ToolHandler {
			gitDeps := deps.(GitToolDependencies)
			return func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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
			}
		},
	)
}

// GitDiffUnstaged creates a tool to show unstaged changes
func GitDiffUnstaged(t translations.TranslationHelperFunc) inventory.ServerTool {
	return inventory.NewServerToolFromHandler(
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
		ToolsetMetadataLocalGit,
		func(deps any) mcp.ToolHandler {
			gitDeps := deps.(GitToolDependencies)
			return func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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
			}
		},
	)
}

// GitDiffStaged creates a tool to show staged changes
func GitDiffStaged(t translations.TranslationHelperFunc) inventory.ServerTool {
	return inventory.NewServerToolFromHandler(
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
		ToolsetMetadataLocalGit,
		func(deps any) mcp.ToolHandler {
			gitDeps := deps.(GitToolDependencies)
			return func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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
			}
		},
	)
}

// GitDiff creates a tool to show differences between branches or commits
func GitDiff(t translations.TranslationHelperFunc) inventory.ServerTool {
	return inventory.NewServerToolFromHandler(
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
		ToolsetMetadataLocalGit,
		func(deps any) mcp.ToolHandler {
			gitDeps := deps.(GitToolDependencies)
			return func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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
			}
		},
	)
}


// GitCommit creates a tool to commit changes
func GitCommit(t translations.TranslationHelperFunc) inventory.ServerTool {
	return inventory.NewServerToolFromHandler(
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
		ToolsetMetadataLocalGit,
		func(deps any) mcp.ToolHandler {
			gitDeps := deps.(GitToolDependencies)
			return func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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
			}
		},
	)
}

// GitAdd creates a tool to add files to staging area
func GitAdd(t translations.TranslationHelperFunc) inventory.ServerTool {
	return inventory.NewServerToolFromHandler(
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
		ToolsetMetadataLocalGit,
		func(deps any) mcp.ToolHandler {
			gitDeps := deps.(GitToolDependencies)
			return func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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
				if strings.Contains(filesStr, ",") {
					files = strings.Split(filesStr, ",")
					for i, file := range files {
						files[i] = strings.TrimSpace(file)
					}
				} else if strings.Contains(filesStr, " ") {
					files = strings.Split(filesStr, " ")
					for i, file := range files {
						files[i] = strings.TrimSpace(file)
					}
				} else {
					files = []string{filesStr}
				}

				result, err := gitDeps.GetGitOps().AddFiles(repoPath, files)
				if err != nil {
					return utils.NewToolResultError(fmt.Sprintf("Failed to add files: %v", err)), nil
				}

				return utils.NewToolResultText(result), nil
			}
		},
	)
}

// GitReset creates a tool to unstage changes
func GitReset(t translations.TranslationHelperFunc) inventory.ServerTool {
	return inventory.NewServerToolFromHandler(
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
		ToolsetMetadataLocalGit,
		func(deps any) mcp.ToolHandler {
			gitDeps := deps.(GitToolDependencies)
			return func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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
			}
		},
	)
}

// GitLog creates a tool to show commit logs
func GitLog(t translations.TranslationHelperFunc) inventory.ServerTool {
	return inventory.NewServerToolFromHandler(
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
		ToolsetMetadataLocalGit,
		func(deps any) mcp.ToolHandler {
			gitDeps := deps.(GitToolDependencies)
			return func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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
			}
		},
	)
}

// GitCreateBranch creates a tool to create a new branch
func GitCreateBranch(t translations.TranslationHelperFunc) inventory.ServerTool {
	return inventory.NewServerToolFromHandler(
		mcp.Tool{
			Name:        "git_create_branch",
			Description: t("TOOL_GIT_CREATE_BRANCH_DESCRIPTION", "Creates a new branch from an optional base branch"),
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
		ToolsetMetadataLocalGit,
		func(deps any) mcp.ToolHandler {
			gitDeps := deps.(GitToolDependencies)
			return func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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
			}
		},
	)
}

// GitCheckout creates a tool to switch branches
func GitCheckout(t translations.TranslationHelperFunc) inventory.ServerTool {
	return inventory.NewServerToolFromHandler(
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
		ToolsetMetadataLocalGit,
		func(deps any) mcp.ToolHandler {
			gitDeps := deps.(GitToolDependencies)
			return func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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
			}
		},
	)
}

// GitShow creates a tool to show commit contents
func GitShow(t translations.TranslationHelperFunc) inventory.ServerTool {
	return inventory.NewServerToolFromHandler(
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
		ToolsetMetadataLocalGit,
		func(deps any) mcp.ToolHandler {
			gitDeps := deps.(GitToolDependencies)
			return func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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
			}
		},
	)
}

// GitInit creates a tool to initialize a new repository
func GitInit(t translations.TranslationHelperFunc) inventory.ServerTool {
	return inventory.NewServerToolFromHandler(
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
		ToolsetMetadataLocalGit,
		func(deps any) mcp.ToolHandler {
			gitDeps := deps.(GitToolDependencies)
			return func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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
			}
		},
	)
}

// GitPush creates a tool to push changes to remote
func GitPush(t translations.TranslationHelperFunc) inventory.ServerTool {
	return inventory.NewServerToolFromHandler(
		mcp.Tool{
			Name:        "git_push",
			Description: t("TOOL_GIT_PUSH_DESCRIPTION", "Pushes local commits to a remote repository"),
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
		ToolsetMetadataLocalGit,
		func(deps any) mcp.ToolHandler {
			gitDeps := deps.(GitToolDependencies)
			return func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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
			}
		},
	)
}

// GitListRepositories creates a tool to list all available repositories
func GitListRepositories(t translations.TranslationHelperFunc) inventory.ServerTool {
	return inventory.NewServerToolFromHandler(
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
		ToolsetMetadataLocalGit,
		func(deps any) mcp.ToolHandler {
			gitDeps := deps.(GitToolDependencies)
			return func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				repoPaths := gitDeps.GetRepoPaths()
				if len(repoPaths) == 0 {
					return utils.NewToolResultText("No repositories configured"), nil
				}

				var result strings.Builder
				result.WriteString(fmt.Sprintf("Available repositories (%d):\n\n", len(repoPaths)))

				for i, repoPath := range repoPaths {
					// Get the repository name (last part of the path)
					repoName := filepath.Base(repoPath)
					result.WriteString(fmt.Sprintf("%d. %s (%s)\n", i+1, repoName, repoPath))
				}

				return utils.NewToolResultText(result.String()), nil
			}
		},
	)
}

// GitApplyPatchString creates a tool to apply a patch from a string
func GitApplyPatchString(t translations.TranslationHelperFunc) inventory.ServerTool {
	return inventory.NewServerToolFromHandler(
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
		ToolsetMetadataLocalGit,
		func(deps any) mcp.ToolHandler {
			gitDeps := deps.(GitToolDependencies)
			return func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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
			}
		},
	)
}

// GitApplyPatchFile creates a tool to apply a patch from a file
func GitApplyPatchFile(t translations.TranslationHelperFunc) inventory.ServerTool {
	return inventory.NewServerToolFromHandler(
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
		ToolsetMetadataLocalGit,
		func(deps any) mcp.ToolHandler {
			gitDeps := deps.(GitToolDependencies)
			return func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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
			}
		},
	)
}

// AllGitTools returns all git tools
func AllGitTools(t translations.TranslationHelperFunc) []inventory.ServerTool {
	return []inventory.ServerTool{
		GitStatus(t),
		GitDiffUnstaged(t),
		GitDiffStaged(t),
		GitDiff(t),
		GitCommit(t),
		GitAdd(t),
		GitReset(t),
		GitLog(t),
		GitCreateBranch(t),
		GitCheckout(t),
		GitShow(t),
		GitInit(t),
		GitPush(t),
		GitListRepositories(t),
		GitApplyPatchString(t),
		GitApplyPatchFile(t),
	}
}


