# Example: send-direct-message

Send a direct message to one identity and confirm acknowledgement state.

```text
identity show mom
message dm mom "Can you check the front door?"
message ls dm:mom
message unread mom dm:mom
message ack mom msg_1
message unread mom dm:mom
```

Expected outcome:

- `message dm` creates a DM-scoped message thread for the target identity,
- unread state appears for the recipient until acknowledgement,
- unread state clears after `message ack`.
