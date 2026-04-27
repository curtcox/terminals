# Example: start-room-chat

Create a durable room, post a first message, and verify unread/ack flow for one participant.

```text
identity resolve group:family
message room new family-room
message room show family-room
message post family-room "Dinner is ready in ten"
message ls family-room
message unread mom family-room
message ack mom msg_1
message unread mom family-room
```

Expected outcome:

- `identity resolve` shows eligible participants for the room audience,
- the room exists and receives the initial post,
- unread count decreases after `message ack` for the selected participant.
