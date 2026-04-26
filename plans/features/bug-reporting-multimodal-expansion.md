---
title: "Bug Reporting Multimodal Expansion"
kind: plan
status: shipped-validated
owner: copilot
validation: automated:B1
last-reviewed: 2026-04-26
---

# Bug Reporting Multimodal Expansion

This plan tracks follow-on work intentionally split from
[bug-reporting.md](./bug-reporting.md) after the core local bug-reporting
pipeline was shipped and documented.

## Why This Exists

The core bug-reporting pipeline is now durable and validated for:

- server-side filing and storage
- admin list/detail visibility
- JSON and form intake paths
- autodetect merge behavior

The remaining scope is larger, cross-cutting, and not required for current
local diagnostics operation, so it is tracked separately to keep work items
small and finishable.

## Scope

- On-device modality fan-out for B1:
  - server-composed report affordance wrapper consistency across scenarios
  - gesture/keyboard/voice/shake trigger parity and fallback behavior
- Cross-device filing for broken subjects (B2):
  - robust subject resolution from alternate reporter devices
  - explicit dead-subject flows and recently-disconnected selection UX
- Additional off-device intake channels:
  - QR/NFC routing hardening and subject prefill behavior
  - SIP bug-line ingestion adapter and transcript normalization
  - email-in adapter behind server interfaces
- Operational hardening:
  - retention and rotation policy for `logs/bug_reports/`
  - optional admin live-tail for `bug.report.*` events
  - attachment size policy and overflow handling strategy

## Validation Targets

- Add dedicated automated mappings for remaining bug-reporting IDs:
  - B1
  - B2
  - B5
- Keep [docs/usecase-validation-matrix.md](../../docs/usecase-validation-matrix.md)
  and [scripts/usecase-validate.sh](../../scripts/usecase-validate.sh) in sync.

## Exit Criteria

- Remaining bug-reporting use cases (B1/B2/B5) have automated validation.
- Durable behavior and operational guidance are documented in `docs/`.
- Plan can be marked `shipped-validated` or drained/superseded with docs links.
