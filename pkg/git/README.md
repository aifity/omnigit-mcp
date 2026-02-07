# Local Git Tools for GitHub MCP Server

This package provides local git repository operations for the GitHub MCP Server. It enables AI assistants to interact with local git repositories using standard git commands.

## Attribution

This code is adapted from the **git-mcp-go** project by **Gero Posmyk-Leinemann**.

- **Original Author**: Gero Posmyk-Leinemann <gero@gitpod.io>
- **Original Repository**: https://github.com/geropl/git-mcp-go
- **License**: MIT License (see original repository)

The code has been adapted to integrate with the github-mcp-server architecture while preserving the original functionality and design.

## Overview

The git tools package provides 16 tools for working with local git repositories:

### Read-Only Tools
- **git_status** - Show the working tree status
- **git_diff_unstaged** - Show changes in the working directory
- **git_diff_staged** - Show changes staged for commit
- **git_diff** - Show differences between commits, branches, or files
- **git_log** - Show commit history
- **git_show** - Show contents of a specific commit
- **git_list_repositories** - List all configured repositories

### Write Tools
- **git_add** - Add file contents to the staging area
- **git_commit** - Record changes to the repository
- **git_reset** - Unstage all staged changes
- **git_create_branch** - Create a new branch
- **git_checkout** - Switch branches
- **git_init** - Initialize a new git repository
- **git_push** - Push local commits to remote repository
- **git_apply_patch_string** - Apply a patch from a string
- **git_apply_patch_file** - Apply a patch from a file

## Architecture

### Package Structure

```
pkg/git/
├── README.md                    # This file
├── tools.go                     # MCP tool definitions
└── gitops/
    ├── interface.go             # GitOperations interface
    ├── utils.go                 # Shared utilities
    └── shell/
        └── operations.go        # Shell-based git implementation
```

### Key Components

1. **GitOperations Interface** (`gitops/interface.go`)
   - Defines the contract for all git operations
   - Allows for different implementations (shell-based, go-git, etc.)

2. **Shell Implementation** (`gitops/shell/operations.go`)
   - Implements GitOperations using git CLI commands
   - Executes git commands via shell

3. **MCP Tools** (`tools.go`)
   - Wraps git operations as MCP tools
   - Handles parameter validation and error handling
   - Integrates with the inventory system

4. **GitToolDependencies Interface**
   - Provides dependency injection for tools
   - Supplies GitOperations implementation and repository paths

### Security

The package includes path validation to ensure:
- Only configured repositories can be accessed
- Paths are validated as actual git repositories (contain `.git` directory)
- Paths are converted to absolute paths for consistency

## Integration

To integrate these tools into the GitHub MCP Server:

1. Implement the `GitToolDependencies` interface
2. Register tools using `AllGitTools(translationHelper)`
3. Configure allowed repository paths
4. Add tools to the server's inventory

## Differences from Original

The main adaptations from git-mcp-go include:

1. **Tool Registration Pattern**: Uses `inventory.NewServerToolFromHandler` instead of direct MCP tool registration
2. **Dependency Injection**: Uses `GitToolDependencies` interface for cleaner separation
3. **Toolset Metadata**: Tools are grouped under `ToolsetMetadataLocalGit` toolset
4. **Translation Support**: Integrated with github-mcp-server's translation system
5. **Error Handling**: Uses `utils.NewToolResultError` and `utils.NewToolResultText` helpers

## License

This adapted code maintains the original MIT License from git-mcp-go. See the original repository for full license details.

## Credits

Special thanks to Gero Posmyk-Leinemann for creating the original git-mcp-go project, which provided the foundation for these local git tools.

