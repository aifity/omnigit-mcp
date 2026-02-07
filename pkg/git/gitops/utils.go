// Package gitops provides Git operations interface and implementations.
//
// This code is adapted from git-mcp-go by Gero Posmyk-Leinemann and contributors.
// Original source: https://github.com/geropl/git-mcp-go
// Copyright (c) Gero Posmyk-Leinemann <gero@gitpod.io>
package gitops

import (
	"fmt"
	"os/exec"
)

// RunGitCommand runs a git command and returns its output
func RunGitCommand(repoPath string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = repoPath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git command failed: %w\nOutput: %s", err, string(output))
	}
	return string(output), nil
}

