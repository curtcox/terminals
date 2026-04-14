Teach `scenario.ParseVoiceTrigger` to recognize the Phase-6 milestone phrasing
"tell me when X stops" (and close variants) and route it to the
`AudioMonitorScenario` with `target = X` extracted as a trigger argument.
This is the phrasing that `plans/phase-6-monitoring.md` pins as the
end-to-end milestone — with the classifier + hub wiring and explicit stop
already in place, the remaining gap is the voice-trigger parse path.

Add voice-parser tests that cover the phrasing and at least one negative
case, plus a runtime-level test that spoken text activates
`audio_monitor` with the parsed target surfaced in the arming broadcast
("Audio monitor armed: dishwasher").
