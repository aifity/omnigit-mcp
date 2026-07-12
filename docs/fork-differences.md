# Fork Differences From Upstream

This document is the review checklist for differences that Omnigit intentionally keeps on top of `github/github-mcp-server`. Use it when backporting upstream changes, resolving conflicts, preparing releases, or checking whether a merge accidentally reverted fork behavior.

Last reviewed against upstream `main` at `c36e4e44` on 2026-07-12.

## Fork Identity

Omnigit is distributed as `aifity/omnigit-mcp`, not `github/github-mcp-server`.

Preserve these identifiers unless the fork is intentionally renamed:

- Go module and imports: `github.com/aifity/omnigit-mcp`
- Binary and command path: `omnigit-mcp`, `cmd/omnigit-mcp`
- Docker image: `ghcr.io/aifity/omnigit-mcp`
- MCP registry name: `io.github.aifity/omnigit-mcp`
- Default config file: `omnigit-mcp-config.json`
- User-visible product name: `Omnigit GitHub & Local Git MCP Server`

Important files:

- `go.mod`
- `server.json`
- `Dockerfile`
- `.goreleaser.yaml`
- `cmd/omnigit-mcp/`
- `internal/ghmcp/server.go`
- `README.md`
- `docs/installation-guides/`

## Local Git Toolset

Omnigit includes a local repository toolset named `local_git`. It lets agents work with local repositories without going through the GitHub API.

Preserve:

- Toolset metadata in `pkg/git/tools.go`
- Registration through `github.AllTools`
- Local git implementation under `pkg/git/`
- `local_git` in the README toolset table
- Local git guidance in `pkg/inventory/instructions.go`

Do not add `local_git` to remote hosted-server docs unless the remote server actually supports local filesystem git operations.

## Body Filtering

Omnigit filters AI-generated footers and configured patterns before sending issue bodies, pull request bodies, and local git commit messages to GitHub.

Preserve:

- `pkg/bodyfilter/`
- `bodyfilter.FilterBody` calls in issue, pull request, granular issue, granular pull request, and local git commit flows
- `filter_patterns` support in `pkg/translations/translations.go`
- README configuration docs for `filter_patterns`

Default behavior should continue to strip generic AI-generated PR footers and `Co-Authored-By:` trailers where the current tests expect that.

## Comment Management Tools

Omnigit adds write tools for existing comments:

- `issue_comment_write`: update or delete an existing issue comment
- `pull_request_comment_write`: update or delete an existing pull request review comment

Preserve:

- Tool registrations and implementations in `pkg/github/issues.go` and `pkg/github/pullrequests.go`
- Tool snapshots under `pkg/github/__toolsnaps__/`
- README tool docs
- Claude Code skill mappings in `docs/claude-code-skills/omnigit-mcp-only/SKILL.md`

## Review Comment IDs

`get_review_comments` must expose the numeric REST comment ID needed by write tools.

Preserve:

- `CommentID` in the review comment response model
- `database_id` / `DatabaseID` fields where they are part of the existing response contract
- Parameter descriptions that tell callers to use `database_id` from `get_review_comments` for `pull_request_comment_write` and reply operations
- Tests and snapshots that cover these fields

The goal is that an agent can read PR review comments and then update, delete, or reply to the correct comment without decoding opaque GraphQL node IDs or scraping URLs.

## OAuth And Packaging

Upstream may change OAuth behavior; Omnigit still needs fork-specific packaging and registry identity.

Preserve:

- Build-time OAuth client ID/secret ldflags using the `github.com/aifity/omnigit-mcp/internal/buildinfo` path
- Docker callback-port runtime arguments in `server.json`
- Docker and install docs that point at `ghcr.io/aifity/omnigit-mcp`
- Release packaging under the `omnigit-mcp` project name

## Claude Code Skills

Omnigit ships Claude Code skills that encode project-specific usage guidance.

Preserve:

- `docs/claude-code-skills/README.md`
- `docs/claude-code-skills/omnigit-mcp-only/SKILL.md`
- Any installation links that point at `aifity/omnigit-mcp`

## Generated Documentation Notes

`go run ./cmd/omnigit-mcp generate-docs` updates automated sections in:

- `README.md`
- `docs/remote-server.md`
- `docs/insiders-features.md`
- `docs/feature-flags.md`
- `docs/tool-renaming.md`

The README introduction and fork-differences section are manual and should remain outside automated markers.

After editing tool metadata, run:

```bash
go run ./cmd/omnigit-mcp generate-docs
git diff -- README.md docs/remote-server.md docs/insiders-features.md docs/feature-flags.md docs/tool-renaming.md
```

## Backport Audit Checklist

After merging or rebasing upstream, verify the fork deltas explicitly:

```bash
rg -n "aifity|omnigit|local_git|bodyfilter|filter_patterns|issue_comment_write|pull_request_comment_write|database_id|CommentID" \
  README.md docs server.json .goreleaser.yaml Dockerfile cmd internal pkg .github/workflows

git grep -n -E '^(<<<<<<<|=======|>>>>>>>)' -- .
git diff --check
go test ./pkg/bodyfilter ./pkg/git ./pkg/github ./internal/ghmcp
go run ./cmd/omnigit-mcp generate-docs
git diff --exit-code README.md docs/remote-server.md docs/insiders-features.md docs/feature-flags.md docs/tool-renaming.md
```

For a merge commit, also check what the backport deleted relative to the pre-backport fork side:

```bash
git diff --name-status --diff-filter=D HEAD^1
```

Deleted files should be understood and intentional. Fork-specific files should not disappear as a side effect of accepting upstream conflict resolutions.

