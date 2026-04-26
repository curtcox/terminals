# Messaging Service

Typed operations for room chat, direct messages, boards, bulletins, threads, and unread state.

## Room operations

- Create room: `POST /admin/api/message/room` (`name`)
- List rooms: `GET /admin/api/message/rooms`
- Get room: `GET /admin/api/message/room?room=<id-or-name>`

## Message operations

- List messages: `GET /admin/api/message?room=<optional-id-or-name>`
- Get message: `GET /admin/api/message/get?message_id=<id>`
- Post room message: `POST /admin/api/message/post` (`room`, `text`)
- Send direct message: `POST /admin/api/message/dm` (`target_ref`, `text`)
- Reply in thread: `POST /admin/api/message/thread` (`root_ref`, `text`)
- List unread for identity: `GET /admin/api/message/unread?identity_id=<id>&room=<optional>`
- Acknowledge message: `POST /admin/api/message/ack` (`identity_id`, `message_id`)

## Board operations

- List board entries: `GET /admin/api/board?board=<optional>`
- Post board entry: `POST /admin/api/board/post` (`board`, `text`)
- Pin board entry: `POST /admin/api/board/pin` (`board`, `text`)
