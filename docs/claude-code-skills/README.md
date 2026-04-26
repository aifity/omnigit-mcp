# Claude Code skills for omnigit-mcp

This directory ships ready-to-install [Claude Code](https://docs.claude.com/en/docs/claude-code) skills that codify recommended usage patterns for `omnigit-mcp`. A skill auto-loads in every Claude Code session where its trigger phrases match — once installed, the agent applies the skill's rules without the user having to repeat them.

## Available skills

| Skill | Purpose |
|-------|---------|
| [`omnigit-mcp-only`](omnigit-mcp-only/SKILL.md) | Enforces MCP-first git/GitHub usage. Stops the agent from silently shelling out to `git` / `gh` when MCP wrappers exist, and defines a stop-and-ask protocol (with an option to file a wrapper-request issue here) for the cases where they don't. |

## Installing a skill

Pick the scope:

### Global (applies to every project)

```bash
mkdir -p ~/.claude/skills/<skill-name>
curl -fsSL https://raw.githubusercontent.com/aifity/omnigit-mcp/main/docs/claude-code-skills/<skill-name>/SKILL.md \
  -o ~/.claude/skills/<skill-name>/SKILL.md
```

Restart Claude Code (or run `/skills` to confirm it loaded).

### Repo-local (committed to git, applies to anyone who clones the project)

From your project root:

```bash
mkdir -p .claude/skills/<skill-name>
curl -fsSL https://raw.githubusercontent.com/aifity/omnigit-mcp/main/docs/claude-code-skills/<skill-name>/SKILL.md \
  -o .claude/skills/<skill-name>/SKILL.md
git add .claude/skills/<skill-name>/SKILL.md
git commit -m "chore(claude-code): install <skill-name> skill"
```

Repo-local skills load automatically when a Claude Code session is opened in that working directory, on top of any global skills the user has installed.

## Verifying a skill is active

In a Claude Code session, type `/skills` — installed skills appear in the list with their description. The first time the skill's trigger phrase matches a user request (e.g. "commit this", "push the branch"), the skill's body is loaded into the model's context.

## Contributing a new skill

Follow the project's [CONTRIBUTING.md](../../CONTRIBUTING.md). For a Claude Code skill specifically: open an issue first describing the rule the skill enforces and what gap it closes, then submit a PR adding `docs/claude-code-skills/<your-skill-name>/SKILL.md` with the standard frontmatter:

```yaml
---
name: <your-skill-name>
description: <one or two sentences — used by the harness to decide when to load the skill, so include explicit trigger phrases>
---
```
