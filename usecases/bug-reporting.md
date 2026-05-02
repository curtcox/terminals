---
title: "Bug Reporting and Diagnostics Use Cases"
family: B
ids: [B1, B2, B3, B4, B5]
---

# Bug Reporting and Diagnostics

| # | As a ... | I would like to ... | So that I can ... |
|---|----------|---------------------|-------------------|
| B1 | **Home occupant** | file a bug report from the device I am currently using, via whatever input it has (button, gesture, shake, keyboard shortcut, or voice) | report a problem without having to find a different device or a specific input modality |
| B2 | **Home occupant** | file a bug report about a *different* device that has no working input by scanning its QR code, tapping an NFC tag, using the admin dashboard on another device, or speaking the target device's name into voice | report problems on devices whose own IO is broken |
| B3 | **Server (diagnostics engine)** | autodetect probable device failures (heartbeat timeout, never-registered, reconnect loop, repeated control errors) and open a pending bug report with the subject's last-known state | surface problems even before a human notices them, and let the first observer confirm the report with one tap |
| B4 | **Home network admin** | view every filed bug report together with the subject device's last-known capabilities, UI, state, and event-log trace via `/admin/bugs` | reproduce and fix the problem without a synchronous conversation with the reporter |
| B5 | **Home occupant** | dial a reserved extension on the SIP bug line and describe the problem verbally | report a bug from any phone when no screen or keyboard is usable, including when I am not at home |
