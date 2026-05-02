---
title: "Software Agents and Automation Use Cases"
family: AA
ids: [AA1, AA2, AA3, AA4, AA5, AA6]
---

# Software Agents and Automation

| # | As a ... | I would like to ... | So that I can ... |
|---|----------|---------------------|-------------------|
| AA1 | **Automation agent (program)** | trigger scenarios via the server API based on external events (calendar, webhook, sensor) | integrate the terminal system with external automation platforms |
| AA2 | **Monitoring agent (program)** | subscribe to sound classification events and route notifications to other systems (Slack, email) | bridge the terminal system's awareness into existing workflows |
| AA3 | **AI agent (program)** | use the LLM backend to interpret ambiguous voice commands and map them to scenarios | handle natural-language requests that don't match a fixed trigger |
| AA4 | **Scheduling agent (program)** | create, modify, and cancel timers and reminders via the server API | manage scheduled events programmatically on behalf of users |
| AA5 | **Vision analysis agent (program)** | process camera frames and generate alerts or annotations on the viewing device | provide real-time intelligent overlays (e.g., package detected at door) |
| AA6 | **Development agent** | run integration tests that simulate multiple devices connecting and exchanging IO | validate multi-device scenarios in CI without physical hardware |
