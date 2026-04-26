# Identity Service

Typed operations for people, groups, aliases, audiences, acknowledgements, and preferences.

- `ListIdentities()`
- `GetIdentity(identityRef)`
- `ListGroups()`
- `ResolveAudience(audience)` where selectors include `all`, `id:<id>`, `group:<group>`, and `alias:<alias>`.
- `GetPreferences(identityRef)`
- `RecordAcknowledgement(subjectRef, actorRef, mode)`
- `GetAcknowledgements(subjectRef)`

## Admin API

- `GET /admin/api/identity`
- `GET /admin/api/identity/show?identity=<id-or-alias>`
- `GET /admin/api/identity/groups`
- `GET /admin/api/identity/resolve?audience=<selector>`
- `GET /admin/api/identity/prefs?identity=<id-or-alias>`
- `GET /admin/api/identity/ack?subject_ref=<subject-ref>`
- `POST /admin/api/identity/ack` with `subject_ref`, `actor`, and optional `mode`

Actor refs are typed (`person:<id>`, `device:<id>`, `agent:<id>`, `anonymous:<origin>`) and acknowledgement mode defaults to `read` when omitted.
