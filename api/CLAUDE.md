# api Agent Notes

## Scope

This folder contains protobuf contracts and code generation config used by server and client.

## Commands

```bash
buf format -w
buf lint
buf generate
buf breaking --against '.git#branch=main'
```

## Guardrails

- Keep wire contracts backward compatible.
- Use explicit field numbering and avoid field reuse.
- Regenerate bindings after proto changes.

