# Terminals Web Client

Plain HTML/CSS/JavaScript terminal client for browser-first smoke tests and static deployments. It is a second generic renderer for the same server-owned terminal protocol, not a product fork.

## Run

```bash
make web-client-test
make run-web-client
```

The endpoint can come from `?ws=ws://host:port/control`, local storage key `terminals.controlWsEndpoint`, or manual entry in the browser chrome.

## Boundary Rules

1. `web_client/src/ui/**` may import generated UI protobuf bindings and DOM helper modules.
2. `web_client/src/ui/**` may not import transport sockets, server orchestration concepts, scenario names, placement, claims, REPL, MCP, or app runtime modules.
3. `web_client/src/ui/**` may emit generic renderer actions, but may not send protobuf messages directly.
4. `web_client/src/protocol/**` may translate renderer actions to protobuf messages.
5. `web_client/src/transport/**` may move protobuf envelopes over WebSocket, but may not interpret scenario semantics.
6. Browser permission prompts belong in capabilities, media, or diagnostics modules, not renderer primitives.
7. New UI primitives require a protocol plan, generated bindings, renderer support, and focused tests.
8. Scenario names and application IDs are allowed in tests and fixtures only when they are data received from the server.

## Protocol Note

`src/protocol/generated/**` is intentionally isolated behind `src/protocol/codec.js` and `src/transport/envelope_codec.js`. The current dependency-free bindings cover the web client boundary and tests; replacing them with a richer generator should not require renderer or app changes.
