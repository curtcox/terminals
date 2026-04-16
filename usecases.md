# Terminals — Use Cases

Use cases derived from the [Master Plan](masterplan.md). The first section covers use cases explicitly described in the plan. The second section covers adjacent use cases that the architecture naturally enables.

---

## Planned Use Cases

### Communication

| # | As a ... | I would like to ... | So that I can ... |
|---|----------|---------------------|-------------------|
| C1 | **Parent at home** | press a button or say "intercom to kitchen" and have two-way audio with the kitchen device | talk to my family in another room without shouting |
| C2 | **Household member** | say "announce: dinner is ready" and have it broadcast to every device | make a whole-house announcement from wherever I am |
| C3 | **Parent** | activate PA mode so my microphone streams to all speakers simultaneously | address everyone in the house in real time, like a PA system |
| C4 | **Home user** | say "call Mom" and have the system place a phone call via SIP | make external phone calls from any device without picking up my phone |
| C5 | **Family member** | start a video call between two devices in the house | have a face-to-face conversation with someone in another room |
| C6 | **Remote worker at home** | use a tablet as a dedicated video call terminal | join calls from a fixed station without tying up my laptop |

### Voice Assistant

| # | As a ... | I would like to ... | So that I can ... |
|---|----------|---------------------|-------------------|
| V1 | **Home user** | say a wake word and ask a question on any device | get a spoken and visual answer without touching a keyboard |
| V2 | **Cook in the kitchen** | ask for a recipe by voice and see it displayed on the nearest screen | follow instructions hands-free while cooking |
| V3 | **Household member** | ask the system about the weather, news, or general knowledge | get quick answers from any room |

### Timers, Reminders, and Scheduling

| # | As a ... | I would like to ... | So that I can ... |
|---|----------|---------------------|-------------------|
| T1 | **Cook** | say "set a timer for 10 minutes" | be alerted when my food is ready without watching the clock |
| T2 | **Home user** | say "remind me to check the oven at 3 PM" | get a spoken and visual reminder at the right time |
| T3 | **Parent** | configure the system to monitor my child's morning routine via camera at 7 AM on school days | be told if my child is running late, with escalating alerts |
| T4 | **Parent** | have the system warn my child via speaker ("the bus comes in 10 minutes") | help my child stay on schedule without nagging in person |

### Monitoring and Alerts

| # | As a ... | I would like to ... | So that I can ... |
|---|----------|---------------------|-------------------|
| M1 | **Home user** | say "tell me when the dishwasher stops" and have the system listen for silence | be notified the moment a noisy appliance finishes its cycle |
| M2 | **Home user** | say "tell me when the dryer beeps" and have the system listen for a specific sound | be alerted by a sound event I'd otherwise miss in another room |
| M3 | **Household member** | say "red alert" and have every screen turn red with an alarm sound | immediately get everyone's attention in an emergency or for fun |
| M4 | **Any user** | say "stand down" or tap any device to dismiss a red alert | restore all devices to their previous state after an alert |
| M5 | **Parent** | have the system watch a room's camera for activity during certain hours | know if someone is where they should (or shouldn't) be |

### Display and Ambient

| # | As a ... | I would like to ... | So that I can ... |
|---|----------|---------------------|-------------------|
| D1 | **Homeowner** | set an idle tablet to photo frame mode that rotates pictures | use unused screens as ambient displays showing family photos |
| D2 | **Home user** | have the photo frame automatically yield to higher-priority scenarios and resume afterward | never miss an alert or call because a photo was on screen |
| D3 | **Home user** | see a clock or standby screen on idle devices | have useful ambient information visible at a glance |

### Security and Surveillance

| # | As a ... | I would like to ... | So that I can ... |
|---|----------|---------------------|-------------------|
| S1 | **Homeowner** | say "show all cameras" and see a grid of every device's camera on one screen | get a multi-feed security view of my home from a single device |
| S2 | **Homeowner** | tap a camera cell in the grid to isolate its audio | hear what's happening in a specific room without mixed audio |
| S3 | **Homeowner** | have camera feeds mixed into a single audio track by default | hear an overview of activity across all monitored rooms |

### Terminal and Productivity

| # | As a ... | I would like to ... | So that I can ... |
|---|----------|---------------------|-------------------|
| P1 | **Developer / power user** | open a text terminal on a laptop or Chromebook that connects to the server's shell | use any nearby device as a terminal into the central server |
| P2 | **Power user** | have multiple terminal sessions on one device or access one session from multiple devices | work flexibly across screens without losing context |

### System and Infrastructure

| # | As a ... | I would like to ... | So that I can ... |
|---|----------|---------------------|-------------------|
| I1 | **Device (client program)** | discover the server automatically via mDNS on the local network | connect without manual configuration on a trusted LAN |
| I2 | **Device (client program)** | fall back to manual server address entry if mDNS fails | still connect when network configuration blocks discovery |
| I3 | **Device (client program)** | report my full capability manifest (screen, mic, camera, sensors, etc.) on connection | let the server know exactly what I can do and never receive commands I can't handle |
| I4 | **Server (scenario engine)** | query the device registry for devices matching a required capability set | route scenarios to appropriate devices automatically |
| I5 | **Server (IO router)** | consume, produce, forward, fork, mix, composite, record, and analyze any IO stream dynamically | orchestrate arbitrarily complex media flows without client changes |
| I6 | **Server (scenario engine)** | preempt lower-priority scenarios and suspend them for later resumption | handle competing demands for device IO gracefully |
| I7 | **Server (AI backend)** | swap between local and cloud AI implementations via configuration | adapt to the user's hardware, cost, and privacy preferences |
| I8 | **CI pipeline (program)** | run `make all-check` to validate the entire codebase (lint, test, proto) | catch regressions before code is merged |
| I9 | **Development agent (Claude Code / Codex)** | read CLAUDE.md and AGENTS.md to understand the project structure and rules | contribute new scenarios and features with full context |
| I10 | **Development agent** | add a new server-side scenario without touching client code | extend system behavior while respecting the thin-client architecture |
| I11 | **Device (client program)** | automatically reconnect and have my previous state restored after a disconnect | resume where I left off without manual intervention |

### Bug Reporting and Diagnostics

| # | As a ... | I would like to ... | So that I can ... |
|---|----------|---------------------|-------------------|
| B1 | **Home occupant** | file a bug report from the device I am currently using, via whatever input it has (button, gesture, shake, keyboard shortcut, or voice) | report a problem without having to find a different device or a specific input modality |
| B2 | **Home occupant** | file a bug report about a *different* device that has no working input by scanning its QR code, tapping an NFC tag, using the admin dashboard on another device, or speaking the target device's name into voice | report problems on devices whose own IO is broken |
| B3 | **Server (diagnostics engine)** | autodetect probable device failures (heartbeat timeout, never-registered, reconnect loop, repeated control errors) and open a pending bug report with the subject's last-known state | surface problems even before a human notices them, and let the first observer confirm the report with one tap |
| B4 | **Home network admin** | view every filed bug report together with the subject device's last-known capabilities, UI, state, and event-log trace via `/admin/bugs` | reproduce and fix the problem without a synchronous conversation with the reporter |
| B5 | **Home occupant** | dial a reserved extension on the SIP bug line and describe the problem verbally | report a bug from any phone when no screen or keyboard is usable, including when I am not at home |

---

## Adjacent Use Cases

These are not explicitly described in the master plan but are natural extensions of the architecture.

### Home

| # | As a ... | I would like to ... | So that I can ... |
|---|----------|---------------------|-------------------|
| AH1 | **Homeowner** | have the system announce when someone arrives (doorbell camera + sound detection) | know when visitors or deliveries arrive without being near the door |
| AH2 | **Elderly family member** | call for help by voice from any room and have it alert a caregiver's device | get assistance quickly in an emergency |
| AH3 | **Home user** | stream music from the server to one or more speakers simultaneously | play synchronized audio throughout the house |
| AH4 | **Home user** | use voice commands to control smart home devices (lights, thermostat) via server-side integrations | manage my home without a separate smart home app |
| AH5 | **Parent** | have the system read a bedtime story aloud via TTS on the child's room device | provide a consistent bedtime routine from any location |
| AH6 | **Homeowner** | receive a spoken summary of the day's weather and schedule each morning on the kitchen device | start my day informed without checking my phone |
| AH7 | **Pet owner** | say "check on the dog" and see the camera feed from the room the pet is in | monitor my pet from another room |
| AH8 | **Home user** | have the system detect smoke alarm or CO detector sounds and alert all devices | be notified of safety events even in distant rooms |
| AH9 | **Home occupant** | ask whether unusual accelerometer events were recorded recently | confirm whether what I just felt was detected by the system |
| AH10 | **Home occupant** | ask the system to identify a sound I just heard | understand what the sound was |
| AH11 | **Home occupant** | ask the system to locate where a sound I just heard came from | know where the sound originated |
| AH12 | **Home occupant** | ask who is currently in the house and where they are | find and talk to the right person quickly |
| AH13 | **Home occupant** | be notified when a relevant change is detected by sensors or devices | take action quickly when something changes |
| AH14 | **Home occupant** | be notified when unusual behavior or anomalies are detected | take action when something seems wrong |
| AH15 | **Home occupant** | track the location of important objects in the home | find needed items when they are misplaced |
| AH16 | **Home network admin** | view which Bluetooth devices are active in the house and where they are observed | understand what devices are operating and where |
| AH17 | **Home network admin** | verify the physical locations of terminal devices against reported placement | trust and validate the system's location data |

### Office

| # | As a ... | I would like to ... | So that I can ... |
|---|----------|---------------------|-------------------|
| AO1 | **Office worker** | use the intercom to reach a colleague in another office or conference room | communicate without walking across the building or sending a message |
| AO2 | **Receptionist** | see all office camera feeds in a grid on a dedicated monitor | monitor building entry points from a single station |
| AO3 | **Office manager** | broadcast announcements to all office devices via PA mode | communicate building-wide information (fire drill, office closure) |
| AO4 | **Meeting organizer** | display a shared screen or agenda on a conference room device driven by the server | present information without connecting a laptop to a projector |
| AO5 | **IT administrator** | use terminal sessions on any device to manage the server | perform administrative tasks from any workstation |
| AO6 | **Office manager** | set idle lobby displays to show company branding or visitor welcome messages | use available screens for ambient communication |
| AO7 | **Team lead** | set up a persistent intercom channel between two team rooms | enable low-friction ongoing communication between teams |

### Business / Retail / Hospitality

| # | As a ... | I would like to ... | So that I can ... |
|---|----------|---------------------|-------------------|
| AB1 | **Restaurant manager** | have kitchen and front-of-house devices intercomed together | coordinate between servers and kitchen staff without shouting |
| AB2 | **Retail store manager** | broadcast announcements to the sales floor via PA mode | communicate with staff during business hours |
| AB3 | **Hotel front desk** | display guest welcome messages or room information on lobby devices | provide personalized hospitality using idle screens |
| AB4 | **Warehouse supervisor** | monitor camera feeds from loading docks and aisles in a multi-window grid | maintain situational awareness of the facility |
| AB5 | **Business owner** | set up audio monitoring to detect alarm or glass-break sounds after hours | be alerted to security events without a dedicated alarm system |
| AB6 | **Restaurant staff** | say "timer 12 minutes table 5" and be reminded when food should be checked | track multiple cooking or service timers by voice |
| AB7 | **Clinic receptionist** | use voice to announce "patient Smith, room 3 is ready" to the waiting area device | direct patients without leaving the desk |

### Software Agents and Automation

| # | As a ... | I would like to ... | So that I can ... |
|---|----------|---------------------|-------------------|
| AA1 | **Automation agent (program)** | trigger scenarios via the server API based on external events (calendar, webhook, sensor) | integrate the terminal system with external automation platforms |
| AA2 | **Monitoring agent (program)** | subscribe to sound classification events and route notifications to other systems (Slack, email) | bridge the terminal system's awareness into existing workflows |
| AA3 | **AI agent (program)** | use the LLM backend to interpret ambiguous voice commands and map them to scenarios | handle natural-language requests that don't match a fixed trigger |
| AA4 | **Scheduling agent (program)** | create, modify, and cancel timers and reminders via the server API | manage scheduled events programmatically on behalf of users |
| AA5 | **Vision analysis agent (program)** | process camera frames and generate alerts or annotations on the viewing device | provide real-time intelligent overlays (e.g., package detected at door) |
| AA6 | **Development agent** | run integration tests that simulate multiple devices connecting and exchanging IO | validate multi-device scenarios in CI without physical hardware |
