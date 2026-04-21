# Repo Skills

This directory stores repo-local skills used by coding agents.

## Layout

- One skill per directory under `.claude/skills/`.
- Each skill directory must include a `SKILL.md`.
- Canonical path format: `.claude/skills/<skill-name>/SKILL.md`.

## Naming

- Directory name must match the frontmatter `name` field.
- Use lowercase kebab-case names (for example: `usecase-validate`).

## Required `SKILL.md` frontmatter

Each `SKILL.md` must begin with YAML frontmatter and include:

- `name`: skill identifier (must equal directory name)
- `description`: one-line summary used for routing/discovery

Example:

```md
---
name: my-skill
description: Short summary of when and why this skill is used.
---
```

## How to add a skill

1. Create `.claude/skills/<skill-name>/SKILL.md`.
2. Add required frontmatter (`name`, `description`).
3. Add workflow instructions to the skill body.
4. Add an entry to `/Users/curt/me/terminals/SKILLS.md`.
5. Run `make skills-validate`.

## Validation

Run:

```bash
make skills-validate
```

This checks:

- each skill directory contains `SKILL.md`
- frontmatter exists
- `name` exists and matches directory name
- `description` exists
