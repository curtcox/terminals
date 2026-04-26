---
title: "Bug Reporting and Diagnostics"
kind: plan
status: superseded
owner: copilot
validation: automated:B4
last-reviewed: 2026-04-26
---

# Bug Reporting and Diagnostics

Status: Completed core local pipeline and drained on 2026-04-26.

Durable implementation and operating details now live in:

- [docs/bug-reporting.md](../../docs/bug-reporting.md)
- [docs/usecase-validation-matrix.md](../../docs/usecase-validation-matrix.md)
- [plans/incidents/2026-04-16-bug-reporting.md](../incidents/2026-04-16-bug-reporting.md)

Automated coverage promoted in this pass:

- B3 via `make usecase-validate USECASE=B3`
- B4 via `make usecase-validate USECASE=B4`

Remaining stretch work (multi-modal intake expansions such as QR/NFC/SIP/voice
path hardening) has been scope-split into:

- [plans/features/bug-reporting-multimodal-expansion.md](../features/bug-reporting-multimodal-expansion.md)
