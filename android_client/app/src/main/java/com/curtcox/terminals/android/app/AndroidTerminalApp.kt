package com.curtcox.terminals.android.app

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.text.selection.SelectionContainer
import androidx.compose.material3.Button
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.testTag
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.unit.dp
import com.curtcox.terminals.android.ui.DeviceControlEffects
import com.curtcox.terminals.android.ui.ServerDrivenRenderer
import com.curtcox.terminals.android.ui.ServerDrivenRendererPlaceholder
import com.curtcox.terminals.android.ui.widgets.TerminalShellAudioVisualizer
import com.curtcox.terminals.android.ui.widgets.TerminalShellVideoSurface

@Composable
fun AndroidTerminalApp(viewModel: AndroidTerminalViewModel) {
    val state by viewModel.state.collectAsState()
    MaterialTheme {
        Surface(modifier = Modifier.fillMaxSize()) {
            Column(
                modifier = Modifier
                    .fillMaxSize()
                    .padding(24.dp),
                verticalArrangement = Arrangement.spacedBy(16.dp),
            ) {
                Text("Terminals", style = MaterialTheme.typography.headlineMedium)
                Text("Native Android terminal shell", style = MaterialTheme.typography.bodyMedium)

                Row(horizontalArrangement = Arrangement.spacedBy(12.dp), modifier = Modifier.fillMaxWidth()) {
                    OutlinedTextField(
                        value = state.endpointText,
                        onValueChange = viewModel::updateEndpoint,
                        label = { Text("Server endpoint") },
                        singleLine = true,
                        modifier = Modifier
                            .weight(1f)
                            .testTag("terminal-endpoint-field"),
                    )
                    Button(
                        onClick = viewModel::connect,
                        enabled = state.connectionState == ConnectionState.ReadyToConnect,
                        modifier = Modifier.testTag("terminal-connect-button"),
                    ) {
                        Text("Connect")
                    }
                }

                Row(horizontalArrangement = Arrangement.spacedBy(12.dp), modifier = Modifier.fillMaxWidth()) {
                    Button(
                        onClick = viewModel::startDiscovery,
                        enabled = !state.discoveryState.scanning,
                        modifier = Modifier.testTag("terminal-discovery-start-button"),
                    ) {
                        Text("Discover")
                    }
                    Button(
                        onClick = viewModel::stopDiscovery,
                        enabled = state.discoveryState.scanning,
                        modifier = Modifier.testTag("terminal-discovery-stop-button"),
                    ) {
                        Text("Stop")
                    }
                    Text(state.discoveryState.statusText, style = MaterialTheme.typography.bodySmall)
                }

                if (state.discoveryState.servers.isNotEmpty()) {
                    Column(verticalArrangement = Arrangement.spacedBy(4.dp)) {
                        state.discoveryState.servers.forEach { server ->
                            TextButton(
                                onClick = { viewModel.selectDiscoveredServer(server) },
                                modifier = Modifier.testTag("terminal-discovered-server-${server.host}-${server.port}"),
                            ) {
                                Text("${server.name} (${server.host}:${server.port})")
                            }
                        }
                    }
                }

                state.lastError?.let {
                    Text(it, color = MaterialTheme.colorScheme.error)
                }

                if (state.permissionEducation.messages.isNotEmpty()) {
                    Column(verticalArrangement = Arrangement.spacedBy(4.dp)) {
                        state.permissionEducation.messages.forEach { message ->
                            Text(message, style = MaterialTheme.typography.bodySmall)
                        }
                    }
                }
                if (!state.permissionEducation.notificationsGranted ||
                    (state.permissionEducation.microphonePresent && !state.permissionEducation.microphoneAvailable) ||
                    (state.permissionEducation.cameraPresent && !state.permissionEducation.cameraAvailable)
                ) {
                    Button(
                        onClick = viewModel::requestMissingPermissions,
                        modifier = Modifier.testTag("terminal-request-missing-permissions-button"),
                    ) {
                        Text("Enable missing permissions")
                    }
                }
                if (!state.permissionEducation.notificationsGranted) {
                    Button(
                        onClick = viewModel::requestNotificationPermission,
                        modifier = Modifier.testTag("terminal-request-notification-permission-button"),
                    ) {
                        Text("Enable notifications")
                    }
                }
                if (state.permissionEducation.microphonePresent && !state.permissionEducation.microphoneAvailable) {
                    Button(
                        onClick = viewModel::requestMicrophonePermission,
                        modifier = Modifier.testTag("terminal-request-microphone-permission-button"),
                    ) {
                        Text("Enable microphone")
                    }
                }
                if (state.permissionEducation.cameraPresent && !state.permissionEducation.cameraAvailable) {
                    Button(
                        onClick = viewModel::requestCameraPermission,
                        modifier = Modifier.testTag("terminal-request-camera-permission-button"),
                    ) {
                        Text("Enable camera")
                    }
                }
                if (!state.mediaSupport.webRtcSupported) {
                    Text(
                        "Live media transport is unavailable: ${state.mediaSupport.webRtcReason}.",
                        style = MaterialTheme.typography.bodySmall,
                        modifier = Modifier.testTag("terminal-live-media-status"),
                    )
                }

                Text("Status: ${state.connectionState}", style = MaterialTheme.typography.titleMedium)
                Text(
                    "Last server activity: ${state.lastControlResponseActivity ?: "—"}",
                    style = MaterialTheme.typography.bodySmall,
                    modifier = Modifier.testTag("terminal-last-server-activity"),
                )
                Spacer(modifier = Modifier.height(8.dp))
                Button(
                    onClick = { viewModel.setLocalKeepAwake(!state.localKeepAwakeEnabled) },
                    modifier = Modifier.testTag("terminal-local-keep-awake-button"),
                ) {
                    Text(if (state.localKeepAwakeEnabled) "Keep awake on" else "Keep awake off")
                }
                Button(
                    onClick = { viewModel.setLocalFullscreen(!state.localFullscreenEnabled) },
                    modifier = Modifier.testTag("terminal-local-fullscreen-button"),
                ) {
                    Text(if (state.localFullscreenEnabled) "Fullscreen on" else "Fullscreen off")
                }
                Button(
                    onClick = { viewModel.setLocalBrightDisplay(!state.localBrightDisplayEnabled) },
                    modifier = Modifier.testTag("terminal-local-bright-display-button"),
                ) {
                    Text(if (state.localBrightDisplayEnabled) "Bright display on" else "Bright display off")
                }
                Button(
                    onClick = viewModel::copyDiagnostics,
                    modifier = Modifier.testTag("terminal-copy-diagnostics-button"),
                ) {
                    Text("Copy diagnostics")
                }
                state.lastDiagnosticsCopyStatus?.let {
                    Text("Diagnostics copy: $it", style = MaterialTheme.typography.bodySmall)
                }
                SelectionContainer {
                    Text(state.diagnosticsText, fontFamily = FontFamily.Monospace)
                }

                state.serverRoot?.let { root ->
                    ServerDrivenRenderer(
                        root = root,
                        onAction = viewModel::sendUiAction,
                        onTerminalKeyText = viewModel::sendTerminalKeyText,
                        mediaSurface = { trackId -> TerminalShellVideoSurface(trackId = trackId) },
                        audioVisualizerSurface = { streamId ->
                            TerminalShellAudioVisualizer(streamId = streamId)
                        },
                        imageLoader = { url, _ -> Text(url) },
                        deviceControlEffects = DeviceControlEffects(
                            setKeepAwake = viewModel::setKeepAwake,
                            setFullscreen = viewModel::setFullscreen,
                            setBrightness = viewModel::setBrightness,
                        ),
                    )
                } ?: ServerDrivenRendererPlaceholder()
            }
        }
    }
}
