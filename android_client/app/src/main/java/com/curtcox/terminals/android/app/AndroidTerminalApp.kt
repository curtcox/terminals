package com.curtcox.terminals.android.app

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.text.selection.SelectionContainer
import androidx.compose.material3.Button
import androidx.compose.material3.DropdownMenu
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
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
                    onClick = { viewModel.setLocalImmersiveSticky(!state.localImmersiveStickyEnabled) },
                    modifier = Modifier.testTag("terminal-local-immersive-sticky-button"),
                ) {
                    Text(
                        if (state.localImmersiveStickyEnabled) {
                            "Immersive sticky on"
                        } else {
                            "Immersive sticky off"
                        },
                    )
                }
                Button(
                    onClick = { viewModel.setLocalBrightDisplay(!state.localBrightDisplayEnabled) },
                    modifier = Modifier.testTag("terminal-local-bright-display-button"),
                ) {
                    Text(if (state.localBrightDisplayEnabled) "Bright display on" else "Bright display off")
                }
                Button(
                    onClick = viewModel::togglePrivacyMode,
                    modifier = Modifier.testTag("terminal-privacy-toggle-button"),
                ) {
                    Text(if (state.privacyModeEnabled) "Privacy on" else "Privacy off")
                }
                if (state.connectionState == ConnectionState.Connected) {
                    Row(
                        horizontalArrangement = Arrangement.spacedBy(12.dp),
                        modifier = Modifier.fillMaxWidth(),
                    ) {
                        TextButton(
                            onClick = viewModel::sendRuntimeStatusQuery,
                            modifier = Modifier.testTag("terminal-debug-runtime-status-button"),
                        ) {
                            Text("Runtime status")
                        }
                        TextButton(
                            onClick = viewModel::sendDeviceStatusQuery,
                            modifier = Modifier.testTag("terminal-debug-device-status-button"),
                        ) {
                            Text("Device status")
                        }
                        TextButton(
                            onClick = viewModel::sendPlaybackArtifactsQuery,
                            modifier = Modifier.testTag("terminal-debug-playback-artifacts-button"),
                        ) {
                            Text("List playback artifacts")
                        }
                    }
                    Row(
                        horizontalArrangement = Arrangement.spacedBy(12.dp),
                        modifier = Modifier.fillMaxWidth(),
                    ) {
                        OutlinedTextField(
                            value = state.playbackArtifactIdText,
                            onValueChange = viewModel::updatePlaybackArtifactId,
                            label = { Text("Playback artifact ID") },
                            singleLine = true,
                            modifier = Modifier
                                .weight(1f)
                                .testTag("terminal-playback-artifact-field"),
                        )
                        OutlinedTextField(
                            value = state.playbackTargetDeviceIdText,
                            onValueChange = viewModel::updatePlaybackTargetDeviceId,
                            label = { Text("Target device (optional)") },
                            singleLine = true,
                            modifier = Modifier
                                .weight(1f)
                                .testTag("terminal-playback-target-device-field"),
                        )
                    }
                    TextButton(
                        onClick = viewModel::sendPlaybackMetadataQuery,
                        modifier = Modifier.testTag("terminal-debug-playback-metadata-button"),
                    ) {
                        Text("Playback metadata")
                    }
                    var applicationMenuExpanded by remember { mutableStateOf(false) }
                    Row(
                        horizontalArrangement = Arrangement.spacedBy(12.dp),
                        verticalAlignment = Alignment.CenterVertically,
                        modifier = Modifier.fillMaxWidth(),
                    ) {
                        Box {
                            OutlinedButton(
                                onClick = { applicationMenuExpanded = true },
                                modifier = Modifier.testTag("terminal-application-intent-menu"),
                            ) {
                                Text(state.selectedApplicationIntent)
                            }
                            DropdownMenu(
                                expanded = applicationMenuExpanded,
                                onDismissRequest = { applicationMenuExpanded = false },
                            ) {
                                state.availableApplicationIntents.forEach { intent ->
                                    DropdownMenuItem(
                                        text = { Text(intent) },
                                        onClick = {
                                            viewModel.updateSelectedApplicationIntent(intent)
                                            applicationMenuExpanded = false
                                        },
                                    )
                                }
                            }
                        }
                        TextButton(
                            onClick = viewModel::submitApplicationLaunchCommand,
                            modifier = Modifier.testTag("terminal-debug-launch-application-button"),
                        ) {
                            Text("Open application")
                        }
                        TextButton(
                            onClick = viewModel::sendScenarioRegistryQuery,
                            modifier = Modifier.testTag("terminal-debug-scenario-registry-button"),
                        ) {
                            Text("Refresh applications")
                        }
                    }
                }
                Button(
                    onClick = viewModel::copyDiagnostics,
                    modifier = Modifier.testTag("terminal-copy-diagnostics-button"),
                ) {
                    Text("Copy diagnostics")
                }
                Button(
                    onClick = viewModel::submitChromeBugReport,
                    modifier = Modifier.testTag("terminal-report-bug-button"),
                ) {
                    Text("Report bug")
                }
                state.lastBugReportSubmitStatus?.let {
                    Text(it, style = MaterialTheme.typography.bodySmall, modifier = Modifier.testTag("terminal-bug-report-status"))
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
