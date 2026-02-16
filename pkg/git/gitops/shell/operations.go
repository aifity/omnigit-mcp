// Package shell provides Git operations using git CLI commands.
//
// This code is adapted from git-mcp-go by Gero Posmyk-Leinemann and contributors.
// Original source: https://github.com/geropl/git-mcp-go
// Copyright (c) Gero Posmyk-Leinemann <gero@gitpod.io>
package shell

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/github/github-mcp-server/pkg/bodyfilter"
	"github.com/github/github-mcp-server/pkg/git/gitops"
)

// GitOperations implements GitOperations using git CLI commands
type GitOperations struct{}

// NewGitOperations creates a new GitOperations instance
func NewGitOperations() *GitOperations {
	return &GitOperations{}
}

// GetStatus returns the status of the working tree
func (s *GitOperations) GetStatus(repoPath string) (string, error) {
	return gitops.RunGitCommand(repoPath, "status")
}

// GetDiffUnstaged returns the diff of unstaged changes
func (s *GitOperations) GetDiffUnstaged(repoPath string) (string, error) {
	return gitops.RunGitCommand(repoPath, "diff")
}

// GetDiffStaged returns the diff of staged changes
func (s *GitOperations) GetDiffStaged(repoPath string) (string, error) {
	return gitops.RunGitCommand(repoPath, "diff", "--cached")
}

// GetDiff returns the diff between the current state and a target
func (s *GitOperations) GetDiff(repoPath string, target string) (string, error) {
	return gitops.RunGitCommand(repoPath, "diff", target)
}

// CommitChanges commits the staged changes
func (s *GitOperations) CommitChanges(repoPath string, message string) (string, error) {
	// Filter out unwanted patterns from the commit message
	filteredMessage := bodyfilter.FilterBody(message)

	output, err := gitops.RunGitCommand(repoPath, "commit", "-m", filteredMessage)
	if err != nil {
		return "", fmt.Errorf("failed to commit: %w", err)
	}
	return output, nil
}

// AddFiles adds files to the staging area
func (s *GitOperations) AddFiles(repoPath string, files []string) (string, error) {
	args := append([]string{"add"}, files...)
	_, err := gitops.RunGitCommand(repoPath, args...)
	if err != nil {
		return "", fmt.Errorf("failed to add files: %w", err)
	}
	return "Files staged successfully", nil
}

// ResetStaged unstages all staged changes
func (s *GitOperations) ResetStaged(repoPath string) (string, error) {
	_, err := gitops.RunGitCommand(repoPath, "reset")
	if err != nil {
		return "", fmt.Errorf("failed to reset staged changes: %w", err)
	}
	return "All staged changes reset", nil
}

// GetLog returns the commit history
func (s *GitOperations) GetLog(repoPath string, maxCount int) ([]string, error) {
	args := []string{"log", "--pretty=format:Commit: %H%nAuthor: %an <%ae>%nDate: %ad%nMessage: %s%n"}
	if maxCount > 0 {
		args = append(args, fmt.Sprintf("-n%d", maxCount))
	}

	output, err := gitops.RunGitCommand(repoPath, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get log: %w", err)
	}

	// Split the output into individual commit entries
	logs := strings.Split(strings.TrimSpace(output), "\n\n")
	return logs, nil
}

// CreateBranch creates a new branch and automatically checks it out
func (s *GitOperations) CreateBranch(repoPath string, branchName string, baseBranch string) (string, error) {
	// Use checkout -b to create and switch to the new branch in one command
	args := []string{"checkout", "-b", branchName}
	if baseBranch != "" {
		args = append(args, baseBranch)
	}

	_, err := gitops.RunGitCommand(repoPath, args...)
	if err != nil {
		return "", fmt.Errorf("failed to create and checkout branch: %w", err)
	}

	baseRef := baseBranch
	if baseRef == "" {
		// If no base branch was specified, it was created from the current HEAD
		baseRef = "HEAD"
	}

	return fmt.Sprintf("Created and switched to branch '%s' from '%s'", branchName, baseRef), nil
}

// CheckoutBranch switches to a branch
func (s *GitOperations) CheckoutBranch(repoPath string, branchName string) (string, error) {
	_, err := gitops.RunGitCommand(repoPath, "checkout", branchName)
	if err != nil {
		return "", fmt.Errorf("failed to checkout branch: %w", err)
	}

	return fmt.Sprintf("Switched to branch '%s'", branchName), nil
}

// InitRepo initializes a new Git repository
func (s *GitOperations) InitRepo(repoPath string) (string, error) {
	// Create directory if it doesn't exist
	err := os.MkdirAll(repoPath, 0755)
	if err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}

	_, err = gitops.RunGitCommand(repoPath, "init")
	if err != nil {
		return "", fmt.Errorf("failed to initialize repository: %w", err)
	}

	gitDir := filepath.Join(repoPath, ".git")
	return fmt.Sprintf("Initialized empty Git repository in %s", gitDir), nil
}

// ShowCommit shows the contents of a commit
func (s *GitOperations) ShowCommit(repoPath string, revision string) (string, error) {
	return gitops.RunGitCommand(repoPath, "show", revision)
}

// PushChanges pushes local commits to a remote repository with automatic upstream tracking
func (s *GitOperations) PushChanges(repoPath string, remote string, branch string) (string, error) {
	// Default to "origin" if no remote is specified
	if remote == "" {
		remote = "origin"
	}

	// If no branch is specified, get the current branch
	if branch == "" {
		currentBranch, err := gitops.RunGitCommand(repoPath, "rev-parse", "--abbrev-ref", "HEAD")
		if err != nil {
			return "", fmt.Errorf("failed to get current branch: %w", err)
		}
		branch = strings.TrimSpace(currentBranch)
	}

	// Use --set-upstream to automatically track the remote branch
	args := []string{"push", "--set-upstream", remote, branch}

	output, err := gitops.RunGitCommand(repoPath, args...)
	if err != nil {
		return "", fmt.Errorf("failed to push changes: %w", err)
	}

	// Check if the output indicates that everything is up-to-date
	if strings.Contains(output, "up-to-date") {
		return output, nil
	}

	// Format the output to match the expected format
	return fmt.Sprintf("Successfully pushed to %s/%s\n%s",
		remote,
		branch,
		output), nil
}

// PullChanges pulls changes from a remote repository with automatic rebase and prune
func (s *GitOperations) PullChanges(repoPath string, remote string, branch string) (string, error) {
	// Default to "origin" if no remote is specified
	if remote == "" {
		remote = "origin"
	}

	// Build the pull command with --prune and --rebase flags
	args := []string{"pull", "--prune", "--rebase", remote}

	// Add branch if specified
	if branch != "" {
		args = append(args, branch)
	}

	output, err := gitops.RunGitCommand(repoPath, args...)
	if err != nil {
		return "", fmt.Errorf("failed to pull changes: %w", err)
	}

	// Check if the output indicates that everything is up-to-date
	if strings.Contains(output, "up-to-date") || strings.Contains(output, "Already up to date") {
		return output, nil
	}

	// Format the output to match the expected format
	return fmt.Sprintf("Successfully pulled from %s\n%s", remote, output), nil
}

// ApplyPatchFromFile applies a patch from a file to the repository
func (s *GitOperations) ApplyPatchFromFile(repoPath string, patchFilePath string) (string, error) {
	// Ensure the patch file exists
	if _, err := os.Stat(patchFilePath); os.IsNotExist(err) {
		return "", fmt.Errorf("patch file does not exist: %s", patchFilePath)
	}

	// Apply the patch using git apply
	output, err := gitops.RunGitCommand(repoPath, "apply", patchFilePath)
	if err != nil {
		return "", fmt.Errorf("failed to apply patch: %w", err)
	}

	return fmt.Sprintf("Patch from file '%s' applied successfully\n%s", patchFilePath, output), nil
}

// ApplyPatchFromString applies a patch from a string to the repository
func (s *GitOperations) ApplyPatchFromString(repoPath string, patchString string) (string, error) {
	// Create a temporary file to store the patch
	tmpFile, err := os.CreateTemp("", "git-mcp-patch-*.patch")
	if err != nil {
		return "", fmt.Errorf("failed to create temporary file: %w", err)
	}
	defer os.Remove(tmpFile.Name()) // Clean up the temp file when done

	// Write the patch content to the temporary file
	if _, err := tmpFile.WriteString(patchString); err != nil {
		return "", fmt.Errorf("failed to write patch to temporary file: %w", err)
	}

	// Close the file to ensure all data is written
	if err := tmpFile.Close(); err != nil {
		return "", fmt.Errorf("failed to close temporary file: %w", err)
	}

	// Delegate to the file-based method
	result, err := s.ApplyPatchFromFile(repoPath, tmpFile.Name())
	if err != nil {
		return "", err
	}

	// Modify the result to remove the file path reference since it's a temporary file
	return strings.Replace(result, fmt.Sprintf("from file '%s' ", tmpFile.Name()), "", 1), nil
}
