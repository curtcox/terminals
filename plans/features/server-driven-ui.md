# Server-Driven UI

See [masterplan.md](../masterplan.md) for overall system context.

The server sends UI descriptors that the client renders. This is the mechanism by which the client displays anything beyond the connection screen.

## UI Descriptor Format

```json
{
  "type": "stack",
  "children": [
    {
      "type": "text",
      "value": "user@home:~$ ",
      "style": "monospace",
      "color": "#00FF00"
    },
    {
      "type": "text_input",
      "id": "terminal_input",
      "style": "monospace",
      "autofocus": true
    }
  ],
  "background": "#000000"
}
```

## Supported UI Components

The client renders a fixed set of primitive UI components. These primitives are rich enough to compose any interface the server needs:

- **Layout**: `stack` (vertical), `row` (horizontal), `grid`, `scroll`, `padding`, `center`, `expand`
- **Content**: `text`, `image`, `video_surface`, `audio_visualizer`, `canvas`
- **Input**: `text_input`, `button`, `slider`, `toggle`, `dropdown`, `gesture_area`
- **Feedback**: `notification`, `overlay`, `progress`
- **System**: `fullscreen`, `keep_awake`, `brightness`

Because these are generic primitives, the server can compose them into:

- A terminal emulator (monospace text + text input)
- A video call UI (video surfaces + mute button)
- A photo frame (fullscreen image + timer-based rotation)
- An intercom panel (push-to-talk button + audio visualizer)
- An alert screen (red background + flashing text + alarm audio)
- A PA console (audio visualizer + mute toggle + end button)
- A multi-camera grid (grid of video surfaces + device labels + audio controls)
- Any future UI without client changes

## UI Updates

The server can:

- **Replace** the entire UI: `SetUI` with a full descriptor
- **Patch** the UI: `UpdateUI` targeting a component by ID
- **Animate** transitions: `TransitionUI` with from/to states and duration

## Contract

The set of primitive components is a **closed contract**. Adding a new primitive to satisfy a new scenario is forbidden — compose existing primitives instead. This is the contract that lets the client stay unchanged while the server evolves.

## Related Plans

- [protocol.md](protocol.md) — `SetUI`/`UpdateUI`/`TransitionUI` messages.
- [architecture-client.md](architecture-client.md) — Client-side UI renderer (`lib/ui/server_driven_ui.dart`).
- [architecture-server.md](architecture-server.md) — Server-side descriptor generator (`internal/ui/descriptor.go`).
- [scenario-engine.md](scenario-engine.md) — Scenarios compose descriptors.
