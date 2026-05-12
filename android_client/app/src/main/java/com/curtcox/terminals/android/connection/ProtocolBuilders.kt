package com.curtcox.terminals.android.connection

import com.curtcox.terminals.android.ui.ServerDrivenAction
import terminals.capabilities.v1.Capabilities
import terminals.control.v1.Control
import terminals.diagnostics.v1.Diagnostics
import terminals.io.v1.Io

const val AndroidWireProtocolVersion: Int = 1

class ProtocolBuilders(
    private val protocolVersion: Int = AndroidWireProtocolVersion,
) {
    fun transportHello(
        desiredDeviceId: String,
        resumeToken: String = "",
        supportedCarriers: List<Control.CarrierKind> = listOf(Control.CarrierKind.CARRIER_KIND_GRPC),
    ): Control.WireEnvelope =
        Control.WireEnvelope.newBuilder()
            .setProtocolVersion(protocolVersion)
            .setTransportHello(
                Control.TransportHello.newBuilder()
                    .setProtocolVersion(protocolVersion)
                    .setDesiredDeviceId(desiredDeviceId)
                    .setResumeToken(resumeToken)
                    .addAllSupportedCarriers(supportedCarriers),
            )
            .build()

    fun hello(
        deviceId: String,
        identity: Capabilities.DeviceIdentity,
        clientVersion: String,
    ): Control.ConnectRequest =
        Control.ConnectRequest.newBuilder()
            .setHello(
                Control.Hello.newBuilder()
                    .setDeviceId(deviceId)
                    .setIdentity(identity)
                    .setClientVersion(clientVersion),
            )
            .build()

    fun capabilitySnapshot(
        deviceId: String,
        generation: Long,
        capabilities: Capabilities.DeviceCapabilities,
    ): Control.ConnectRequest =
        Control.ConnectRequest.newBuilder()
            .setCapabilitySnapshot(
                Control.CapabilitySnapshot.newBuilder()
                    .setDeviceId(deviceId)
                    .setGeneration(generation)
                    .setCapabilities(capabilities),
            )
            .build()

    fun capabilityDelta(
        deviceId: String,
        generation: Long,
        capabilities: Capabilities.DeviceCapabilities,
        reason: String,
    ): Control.ConnectRequest =
        Control.ConnectRequest.newBuilder()
            .setCapabilityDelta(
                Control.CapabilityDelta.newBuilder()
                    .setDeviceId(deviceId)
                    .setGeneration(generation)
                    .setCapabilities(capabilities)
                    .setReason(reason),
            )
            .build()

    fun heartbeat(deviceId: String, unixMs: Long): Control.ConnectRequest =
        Control.ConnectRequest.newBuilder()
            .setHeartbeat(
                Control.Heartbeat.newBuilder()
                    .setDeviceId(deviceId)
                    .setUnixMs(unixMs),
            )
            .build()

    /** Matches Flutter shell `ConnectRequest.streamReady` after `StartStream` with non-empty stream id. */
    fun streamReady(streamId: String): Control.ConnectRequest =
        Control.ConnectRequest.newBuilder()
            .setStreamReady(
                Control.StreamReady.newBuilder().setStreamId(streamId),
            )
            .build()

    /** Sends a WebRTC signaling message (ICE candidate, SDP answer) back to the server. */
    fun webRtcSignal(signal: Control.WebRTCSignal): Control.ConnectRequest =
        Control.ConnectRequest.newBuilder()
            .setWebrtcSignal(signal)
            .build()

    /**
     * Matches Flutter `buildSensorTelemetryRequest`: only battery fields from the last registered
     * capabilities snapshot; returns null when there is nothing to send.
     */
    fun sensorTelemetryFromCapabilities(
        deviceId: String,
        capabilities: Capabilities.DeviceCapabilities?,
        unixMs: Long,
    ): Control.ConnectRequest? {
        if (deviceId.isEmpty() || capabilities == null || !capabilities.hasBattery()) {
            return null
        }
        val values =
            mapOf(
                "battery.level" to capabilities.battery.level.toDouble(),
                "battery.charging" to if (capabilities.battery.charging) 1.0 else 0.0,
            )
        return Control.ConnectRequest.newBuilder()
            .setSensor(
                Io.SensorData.newBuilder()
                    .setDeviceId(deviceId)
                    .setUnixMs(unixMs)
                    .putAllValues(values),
            )
            .build()
    }

    fun uiAction(deviceId: String, action: ServerDrivenAction): Control.ConnectRequest =
        Control.ConnectRequest.newBuilder()
            .setInput(
                Io.InputEvent.newBuilder()
                    .setDeviceId(deviceId)
                    .setUiAction(
                        Io.UIAction.newBuilder()
                            .setComponentId(action.componentId)
                            .setAction(action.action)
                            .setValue(action.value),
                    ),
            )
            .build()

    /** Matches Flutter `buildKeyInputRequest` for shell `terminal_input` streaming. */
    fun keyInput(deviceId: String, text: String): Control.ConnectRequest =
        Control.ConnectRequest.newBuilder()
            .setInput(
                Io.InputEvent.newBuilder()
                    .setDeviceId(deviceId)
                    .setKey(Io.KeyEvent.newBuilder().setText(text)),
            )
            .build()

    fun bugReport(report: Diagnostics.BugReport): Control.ConnectRequest =
        Control.ConnectRequest.newBuilder().setBugReport(report).build()

    /** Matches Flutter `buildSystemCommandRequest` (system kind, intent only). */
    fun systemCommand(
        requestId: String,
        intent: String,
    ): Control.ConnectRequest =
        Control.ConnectRequest.newBuilder()
            .setCommand(
                Control.CommandRequest.newBuilder()
                    .setRequestId(requestId)
                    .setKind(Control.CommandKind.COMMAND_KIND_SYSTEM)
                    .setIntent(intent),
            )
            .build()

    /**
     * Matches Flutter `buildPlaybackMetadataQueryRequest`: manual command with map + typed string arguments.
     */
    fun playbackMetadataCommand(
        requestId: String,
        deviceId: String,
        artifactId: String,
        targetDeviceId: String,
    ): Control.ConnectRequest {
        val args =
            mapOf(
                "artifact_id" to artifactId,
                "target_device_id" to targetDeviceId,
            )
        val cmd =
            Control.CommandRequest.newBuilder()
                .setRequestId(requestId)
                .setDeviceId(deviceId)
                .setKind(Control.CommandKind.COMMAND_KIND_MANUAL)
                .setIntent("playback_metadata")
                .putAllArguments(args)
        for ((key, value) in args) {
            cmd.addTypedArguments(
                Control.CommandArgumentEntry.newBuilder()
                    .setKey(key)
                    .setValue(Control.CommandTypedValue.newBuilder().setStringValue(value).build())
                    .build(),
            )
        }
        return Control.ConnectRequest.newBuilder().setCommand(cmd).build()
    }

    /**
     * Matches Flutter `buildApplicationLaunchCommandRequest`: manual start command with optional
     * string arguments (mirrors map + typed string entries).
     */
    fun applicationLaunchCommand(
        requestId: String,
        deviceId: String,
        intent: String,
        arguments: Map<String, String> = emptyMap(),
    ): Control.ConnectRequest {
        val cmd =
            Control.CommandRequest.newBuilder()
                .setRequestId(requestId)
                .setDeviceId(deviceId)
                .setAction(Control.CommandAction.COMMAND_ACTION_START)
                .setKind(Control.CommandKind.COMMAND_KIND_MANUAL)
                .setIntent(intent)
                .putAllArguments(arguments)
        for ((key, value) in arguments) {
            cmd.addTypedArguments(
                Control.CommandArgumentEntry.newBuilder()
                    .setKey(key)
                    .setValue(Control.CommandTypedValue.newBuilder().setStringValue(value).build())
                    .build(),
            )
        }
        return Control.ConnectRequest.newBuilder().setCommand(cmd).build()
    }
}
