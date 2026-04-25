# Shared Artifacts Plan

See `repl-capability-closure.md` for the umbrella capability-closure rationale.

## Design Principle

Server-driven UI is not enough to model durable, reusable, collaboratively edited content. The platform needs shared artifacts that can be rendered through UI primitives but are stored and evolved as first-class resources.

## Goals

- represent durable documents and canvases,
- support templates and reusable signs or symbols,
- support annotation and version history,
- support app-authored lesson content and visual routines,
- keep the client generic.

## Artifact Kinds

Minimum built-in kinds:

- `note`
- `board`
- `canvas`
- `template`
- `lesson`
- `quiz`
- `sign`
- `checklist`

## Data Model

### Artifact

- `artifact_id`
- `kind`
- title
- owner ref
- visibility or audience
- content payload
- metadata
- current version
- created and updated timestamps

### ArtifactVersion

- version number
- patch or replacement payload
- author ref
- created time

### Annotation

- annotation id
- target artifact ref
- author ref
- kind
- payload

## TAL Host Module

Add `artifact`.

Suggested functions:

- `artifact.create(kind, spec)`
- `artifact.get(id)`
- `artifact.list(filters)`
- `artifact.patch(id, patch)`
- `artifact.replace(id, content)`
- `artifact.history(id)`
- `artifact.annotate(id, annotation)`
- `artifact.template.save(name, source_ref)`
- `artifact.template.apply(name, target_ref)`

Optional canvas helpers:

- `artifact.canvas.stroke(...)`
- `artifact.canvas.shape(...)`
- `artifact.canvas.clear(...)`

## Services

### ArtifactService

- `CreateArtifact`
- `GetArtifact`
- `ListArtifacts`
- `PatchArtifact`
- `ReplaceArtifact`
- `GetArtifactHistory`
- `AddAnnotation`
- `SaveTemplate`
- `ApplyTemplate`

## REPL Surface

Add `artifact` and `canvas` command groups.

Examples:

```text
artifact ls
artifact new --kind lesson
artifact show art_42
artifact patch art_42 ...
artifact history art_42
artifact template save bedtime-sign art_42
artifact template apply bedtime-sign --to zone:kids

canvas show art_77
canvas annotate art_77 ...
canvas export art_77
```

## Use Cases Enabled

This plan directly supports:

- shared canvases,
- saved icons and visual templates,
- lesson content and reusable guided practice materials,
- quick visual cues and hand-drawn messages,
- durable activity artifacts that outlive a single UI session.

## Acceptance Criteria

- artifacts are durable, versioned, and addressable independently of any current screen view,
- TAL apps can render and update artifacts through `ui` without conflating view and storage,
- REPL can inspect artifact state and version history,
- templates are reusable across scenarios and zones.
