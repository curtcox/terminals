# Skills Index

Quick lookup for repo-local skills under `.claude/skills/`.

## Rules

1. If a user explicitly names a skill, read `.claude/skills/<name>/SKILL.md` before taking action.
2. If a request clearly matches a skill's "must use when" row below, use that skill workflow.

## Available Skills

| Skill | Path | Trigger phrase examples | Must use when |
| --- | --- | --- | --- |
| `bugword` | `.claude/skills/bugword/SKILL.md` | "use the bugword skill", "work bug word photo", "debug word sky" | The user gives a bug token word/code and wants diagnosis or a fix from bug reports/logs. |
| `terminals-mcp` | `.claude/skills/terminals-mcp/SKILL.md` | "use terminals MCP", "list devices", "stop activation", "tail logs" | The user wants to operate or inspect the live Terminals server through MCP tools (not normal repo code edits). |
| `usecase-implement` | `.claude/skills/usecase-implement/SKILL.md` | "implement use case C2", "add use case M5", "promote S2 to automated" | The user wants to implement behavior and/or add automated validation for a use-case ID. |
| `usecase-validate` | `.claude/skills/usecase-validate/SKILL.md` | "validate C1", "run use-case gate", "does M3 pass?" | The user wants to run existing automated use-case validation and report pass/fail only. |

## Fast Lookup

```bash
rg --files .claude/skills | rg SKILL.md
```
