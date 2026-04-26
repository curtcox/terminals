# artifact

Manage durable shared artifacts (notes, lessons, canvases, templates) via the typed control plane.

## Commands

- `artifact ls`
- `artifact show <artifact-id>`
- `artifact history <artifact-id>`
- `artifact create <kind> <title>`
- `artifact patch <artifact-id> <title>`
- `artifact replace <artifact-id> <title>`
- `artifact template save <name> <source-artifact-id>`
- `artifact template apply <name> <target-artifact-id>`

## Examples

```text
artifact create lesson fractions basics
artifact show art-1
artifact patch art-1 fractions mastery
artifact replace art-1 fractions complete refresh
artifact history art-1
artifact template save lesson-base art-1
artifact template apply lesson-base art-2
```
