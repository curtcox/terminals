# app

Flutter app entry point and shell.

`terminal_client_app.dart` builds the MaterialApp and wires `ClientDependencies` into the widget tree. The `TerminalClientShell` and its sub-files (`_capabilities`, `_carrier`, `_connection`, `_diagnostics`, `_display`, `_media`, `_monitoring`, `_ui`) decompose the shell widget into feature-scoped mixins. `TerminalClientViewState` holds the client-side view model updated by incoming server messages.
