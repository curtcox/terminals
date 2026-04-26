# artifact

Manage durable shared artifacts (notes, lessons, canvases, templates) via the typed control plane.

## Commands

- `artifact ls`
- `artifact show <artifact-id>`
- `artifact history <artifact-id>`
- `artifact create <kind> <title>`
- `artifact patch <artifact-id> <title>`

## Examples

```text
artifact create lesson fractions basics
artifact show art-1
artifact patch art-1 fractions mastery
artifact history art-1
```
