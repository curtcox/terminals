# ui

Server-driven UI rendering primitives.

`server_driven_renderer.dart` is the root widget that interprets `UiNode` trees from the server and builds the Flutter widget hierarchy. `primitive_props.dart` maps server-defined style properties to Flutter widget properties. `renderer_policy.dart` controls render behavior flags. `server_driven_action.dart` and `server_driven_node_key.dart` handle action dispatch and stable widget keying. `idle_main_layer_placeholder.dart` renders the placeholder shown before the first server frame arrives.
