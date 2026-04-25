---
title: "PLATO-Inspired Use Cases for Terminals"
kind: plan
status: planned
owner: unowned
validation: none
last-reviewed: 2026-04-25
---

# PLATO-Inspired Use Cases for Terminals

These use cases are inspired by notable PLATO applications and interaction patterns, but adapted to the architecture and intent of **Terminals**: a server-orchestrated home system with generic client terminals. They are written in the same story format as `usecases.md`.

---

## PLATO-Inspired Use Cases

### Conversation and Messaging

| # | As a ... | I would like to ... | So that I can ... |
|---|----------|---------------------|-------------------|
| PL1 | **Household member** | open a live text chat room for a topic like "kitchen", "kids", or "movie night" from any device | have lightweight room-based conversation without needing everyone on the same commercial messaging app |
| PL2 | **Parent** | send a short typed message to one specific device or person, like "come downstairs" | quietly reach someone without broadcasting to the whole house |
| PL3 | **Household member** | leave a persistent note on a shared household board | let others read and respond later without requiring everyone to be present at once |
| PL4 | **Parent** | pin a high-priority family bulletin to every idle screen | make sure important information is seen repeatedly until it is acknowledged |
| PL5 | **Family member** | reply to a shared household note from any nearby terminal | keep one threaded conversation attached to the original topic instead of scattering context across rooms |
| PL6 | **Home user** | search past household notes, messages, and announcements by topic or date | recover decisions, reminders, and context that would otherwise be forgotten |

### Presence, Help, and Collaboration

| # | As a ... | I would like to ... | So that I can ... |
|---|----------|---------------------|-------------------|
| PL7 | **Parent helping a child** | request to see exactly what is on the child's terminal screen | provide help without walking to the room first |
| PL8 | **Household member** | invite another device into a shared live session where both participants can see the same server-driven view | collaborate on a recipe, checklist, homework screen, or settings page together |
| PL9 | **Remote helper in the house** | temporarily take control of navigation on another family member's screen after they approve it | help someone recover from confusion without physically handling the device |
| PL10 | **Parent** | observe whether a child is actively engaged with an assigned screen or has walked away | know when help or a reminder is needed |
| PL11 | **Home user** | escalate from typed help to voice or intercom from the same session | move to a richer channel only when the lighter one is not enough |

### Learning and Guided Practice

| # | As a ... | I would like to ... | So that I can ... |
|---|----------|---------------------|-------------------|
| PL12 | **Parent or tutor** | publish a guided lesson or drill that can run on any terminal | turn existing household devices into reusable learning stations |
| PL13 | **Learner** | answer short questions and get immediate feedback that adapts to common mistakes | learn interactively instead of just reading static material |
| PL14 | **Parent** | assign a lesson to a specific child at a specific time and nearby device | create routine practice without manual setup each day |
| PL15 | **Learner** | resume a lesson exactly where I left off from a different terminal | continue seamlessly when I move rooms or devices |
| PL16 | **Parent or tutor** | review progress, common wrong answers, and completion history | see whether the lesson is working and where the learner is struggling |
| PL17 | **Music student or speaker** | practice against timing or pitch feedback using a microphone-equipped device | get structured coaching from household terminals instead of a dedicated app |

### Shared Canvases and Expressive Tools

| # | As a ... | I would like to ... | So that I can ... |
|---|----------|---------------------|-------------------|
| PL18 | **Child or family member** | draw or annotate on a shared canvas from a tablet and show it on another screen | use the system for lightweight creativity and collaboration |
| PL19 | **Parent** | send a quick visual cue, symbol, or hand-drawn sketch to a room | communicate something faster than a spoken explanation |
| PL20 | **Household member** | save reusable icons, signs, and visual templates for announcements and routines | reuse familiar household visual language across scenarios |

### Social Play and Group Interaction

| # | As a ... | I would like to ... | So that I can ... |
|---|----------|---------------------|-------------------|
| PL21 | **Family member** | start a multiplayer text or simple graphical game across several terminals | turn spare devices into a shared family activity |
| PL22 | **Parent** | launch a trivia or quiz game on the nearest screens in a room | create an instant group activity without installing apps on each device |
| PL23 | **Household member** | join an ongoing game session from another room or device without losing identity or score | stay part of the activity as I move through the house |
| PL24 | **Parent** | restrict game availability by time, room, or device role | keep playful scenarios from conflicting with homework, bedtime, or shared displays |

### Community, Memory, and Household Knowledge

| # | As a ... | I would like to ... | So that I can ... |
|---|----------|---------------------|-------------------|
| PL25 | **Household member** | maintain topic boards for things like groceries, repairs, vacation ideas, and family traditions | build a shared household memory instead of relying on one person's phone |
| PL26 | **Parent** | have the system surface relevant older notes when a similar topic comes up again | reuse previous decisions and avoid repeating discussions |
| PL27 | **Home user** | browse a chronological activity stream of important household messages, lessons, alerts, and acknowledgements | understand what happened across the house while I was away |

---

## Design Notes

These use cases are adaptations of several PLATO patterns:

- **Notesfiles** -> shared household boards, persistent notes, threaded replies, searchable memory.
- **Talkomatic / Term-talk** -> room chat and direct device-to-device or person-to-person text conversation.
- **Monitor Mode** -> screen observation, shared sessions, and consent-based remote help.
- **Computer-assisted lessons and testing** -> guided practice, adaptive feedback, scheduled learning, progress review.
- **Picture-language / drawing tools** -> shared canvases, saved symbols, visual household communication.
- **Multiplayer games** -> lightweight multi-device family play orchestrated by the server.

The important adaptation is architectural: each experience remains **server-authored and server-orchestrated**, with clients acting only as terminals for rendering, capture, and input forwarding.

