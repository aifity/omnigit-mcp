---
name: omnigit-mcp-only
description: Always use the omnigit-mcp server for git and GitHub operations — never the `git` CLI via Bash and never `gh`. Activates whenever the task involves committing, pushing, branching, merging, force-pushing, opening/reading/commenting on issues or PRs, fetching commits or diffs, listing workflows, downloading artifacts, or any other git/GitHub action. If the operation can't be expressed via the MCP, STOP and ask the user before falling back.
---

# Git/GitHub via MCP only

## The rule

For **every** git or GitHub operation, use the `omnigit-mcp` server. Do not shell out to `git` or `gh`.

This is a hard rule, not a preference.

## Tool naming convention

This skill assumes the MCP is registered as `github` in your Claude Code config — that's the name used in [`install-claude.md`](../../installation-guides/install-claude.md). If you registered the server under a different name (e.g. `omnigit`, `git`, `gh-mcp`), substitute that name everywhere the examples below say `mcp__github__*`. The MCP tool prefix is `mcp__<your-server-name>__<tool>`.

## Concrete mapping (common cases — NOT exhaustive)

The MCP exposes **far more tools than the table below**. The table only covers the most frequent everyday operations. If your task is more specific (sub-issues, copilot review requests, label management, project boards, releases-by-tag, file-level reads/writes via the API, branch protection, etc.), assume there is probably a tool for it — see "Discovering more tools" below before falling back to anything else.

| Operation | Use MCP tool, not… |
| --------- | ------------------ |
| `git status` | `mcp__github__git_status` (not `Bash: git status`) |
| `git diff` (working tree) | `mcp__github__git_diff_unstaged` |
| `git diff --staged` | `mcp__github__git_diff_staged` |
| `git diff <ref>...HEAD` | `mcp__github__git_diff` |
| `git log` | `mcp__github__git_log` |
| `git show <sha>` | `mcp__github__git_show` |
| `git add` | `mcp__github__git_add` |
| `git commit` | `mcp__github__git_commit` |
| `git push` | `mcp__github__git_push` |
| `git pull` | `mcp__github__git_pull` |
| `git checkout` | `mcp__github__git_checkout` |
| `git branch` (create) | `mcp__github__git_create_branch` |
| `git reset` | `mcp__github__git_reset` |
| `git worktree …` | `mcp__github__git_worktree_*` |
| `gh pr create` | `mcp__github__create_pull_request` |
| `gh pr view` / read | `mcp__github__pull_request_read` |
| `gh pr review` | `mcp__github__pull_request_review_write` |
| `gh pr comment` | `mcp__github__pull_request_comment_write` |
| `gh issue view` | `mcp__github__issue_read` |
| `gh issue create` / edit | `mcp__github__issue_write` |
| `gh issue comment` | `mcp__github__issue_comment_write` |
| `gh run list` / view | `mcp__github__actions_*` |
| `gh run download` (logs) | `mcp__github__get_job_logs` |
| `gh release …` | `mcp__github__list_releases` / `mcp__github__get_latest_release` / `mcp__github__get_release_by_tag` |

The MCP tool schemas are deferred — load them via `ToolSearch` with `query="select:mcp__github__<name>[,…]"` before the first call in a session.

## Discovering more tools

Before concluding "no MCP tool for this":

1. Run `ToolSearch` with a **keyword query** scoped to the server, e.g. `query="github <topic>"` (`max_results` 10–20). Examples: `"github label"`, `"github release tag"`, `"github sub_issue"`, `"github copilot review"`. The server has many wrappers around the GitHub REST API that aren't in the everyday table.
2. If a search turns up a plausible candidate, fetch its full schema via `ToolSearch` with `query="select:<full_tool_name>"` and read the description before calling it.
3. Only after a deliberate search comes up empty should you treat the operation as "missing" and trigger the stop-and-ask protocol below.

## What this rules out

- `Bash: git ...` — even for "read-only" calls like `git status` or `git log`. The MCP equivalents exist; use them.
- `Bash: gh ...` — even for "I just want to read this PR's diff."
- "I'll just do it via Bash this once because it's faster" — no. The session preference is consistent: MCP every time.

## Why

MCP-only for git/GitHub is enforced because:

- Structured, typed inputs/outputs over free-form shell parsing.
- No PATH dependency, no shell-quoting bugs in commit messages, no surprises from local `git` aliases or hooks.
- Consistent behavior across machines and worktrees.

## When the needed MCP tool doesn't exist — **STOP**

Some git/GitHub operations probably have no MCP wrapper. Examples that have come up:

- `git stash`
- `git rebase` / `git rebase -i`
- `git cherry-pick`
- `git tag` create (reading via `list_tags` / `get_tag` exists)
- `git remote …`
- `git revert`
- Anything sufficiently exotic

(`gh auth` is also missing — but it has its own dedicated section below because of the dual-auth split, not the same stop-and-ask flow.)

But **don't assume from this list**: the MCP may have grown a wrapper since this skill was written. Do the discovery search above first.

**The protocol when discovery confirms the operation is not in the MCP:**

1. **STOP.** Do not silently fall back to `Bash: git ...` or `Bash: gh ...`. **You must always ask the user — there is no implicit approval, ever, even for "small" operations.**
2. Tell the user clearly, listing all three options every time:
   > "I need to do X. I searched the MCP server and didn't find a wrapper.
   > 
   > Three options:
   > **(a)** I open an issue at https://github.com/aifity/omnigit-mcp asking for the tool. (long-term fix; suggested wording: …)
   > **(b)** I fall back to `git …` / `gh …` via Bash this once, if you approve.
   > **(c)** Both — file the issue *and* run the Bash fallback now.
   > 
   > Which do you want?"
3. **Wait for the user's answer.** Do not continue the surrounding task while waiting — treat this like any other blocker.
4. On `(b)` or `(c)` approval: run the Bash fallback, and only the specific command authorized. Don't generalize the approval to other operations or future calls.
5. On `(a)` or `(c)` — **only after the user picks an issue-filing option**:
   - Search `aifity/omnigit-mcp` for duplicates via `mcp__github__search_issues` (e.g. `query="repo:aifity/omnigit-mcp <keyword>"`) plus a quick `list_issues` scan.
   - If a matching open issue exists → surface it and ask whether to add a comment instead of filing a new one.
   - If none exists → file via `mcp__github__issue_write` (`method: "create"`).
   - In the body: describe what's missing, what the user had to do as a workaround, and — for destructive ops — explicitly suggest the new tool be opt-in (separate tool name, or a server-startup flag). A permission-conscious harness will want to allowlist safe tools and gate destructive ones; not everyone wants force-push / amend / reset --hard exposed by default.

**Don't pre-check for duplicates** before the user has chosen `(a)` or `(c)`. If they pick `(b)`, no GitHub lookup happens at all — saves tokens and round-trips when the user just wants the fallback.

**Always offer to file the issue.** Even if the user just wants the Bash fallback, surface option (a) — that's how the MCP eventually gets the wrapper. The friction of asking every time is the signal that drives the long-term fix.

This protocol applies to **anything unclear**, not just missing tools. If you're not sure whether the MCP supports a flag/option/edge case, ask first. Do not guess and proceed.

## Destructive ops (`--force`, `--amend`, `reset --hard`, branch deletion)

At time of writing, several common destructive operations have no MCP wrapper — `git_push` does not accept a `force` flag, `git_commit` has no `amend` mode, `git_reset` only unstages (no `--hard`). Run discovery first in case wrappers have been added since; if not, the stop-and-ask protocol above is the correct path. The opt-in shape these tools should take if/when added is tracked at https://github.com/aifity/omnigit-mcp/issues/56.

The base "destructive-action confirmation" rule still applies on top: even when a wrapper exists, explicit user authorization is required for each destructive call.

## `gh auth` — read this before suggesting it

`gh auth` is a special case worth flagging explicitly to the user before doing anything with it.

**The omnigit-mcp server uses its own authentication mechanism**, completely independent of the `gh` CLI's credential store. Practical consequences:

- Running `gh auth login` / `gh auth logout` / `gh auth switch` has **no effect on MCP operations**. The MCP keeps using whatever credentials it was configured with.
- Conversely, MCP-side auth changes have no effect on a user's `gh` CLI session.
- The two surfaces can be authenticated as **different GitHub accounts** without either side knowing. A `gh` user who later runs `gh pr view` may be surprised that the MCP-created PR was attributed to a different account.

If a task needs `gh auth` (e.g. the user genuinely wants to switch their CLI account, refresh a token, set scopes), follow the same stop-and-ask protocol as for any other missing tool — but in addition, **explain the dual-auth split before running anything**. The user should understand:

1. Whatever you run via `Bash: gh auth …` only affects their local `gh` CLI; the MCP will continue using its own credentials.
2. After the change, `gh` and the MCP may be authenticated as different identities — actions performed via one won't show up in the other's "me".
3. There is unlikely to ever be a `gh_auth` MCP wrapper, since auth lives outside the operations surface — so this is a permanent Bash-fallback case, not a "missing tool" to file an issue for.

## Scope

This rule applies **only** to `git` and `gh` commands. It says nothing about how to handle anything else — those choices are governed by other rules and the agent's normal judgement.
