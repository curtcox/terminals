# scenarios commands

Inline scenarios let operators prototype trigger-to-action behavior directly
from the REPL without building a TAR package.

## Commands

```text
scenarios ls
scenarios show <name>
scenarios define <name> [--match <intent|intent=x|event=y>]... [--priority <p>]
                        [--on-start <command>] [--on-input <command>]
                        [--on-event <kind> <command>]...
                        [--on-suspend <command>] [--on-resume <command>]
                        [--on-stop <command>]
scenarios undefine <name>
```

## Notes

- `--match` can be repeated. Bare values are treated as intent matches.
- Use `--match event=<kind>` to match a bus event kind.
- `--on-event` can be repeated to bind multiple event hooks.
- `scenarios undefine` removes only REPL-authored inline definitions.
