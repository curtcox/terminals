Make `AudioMonitorScenario.Stop()` cancel the live audio subscription and
classifier goroutine so explicit stops release the DeviceAudio hub subscriber
immediately. Add a control-stream integration test that activates the monitor,
issues a stop (via `StopTrigger` / preempting scenario), and asserts
`Hub.SubscriberCount` drops to zero without waiting for a matching sound event.
