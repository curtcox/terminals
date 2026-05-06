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
import androidx.compose.runtime.Composable
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.unit.dp
import com.curtcox.terminals.android.ui.DeviceControlEffects
import com.curtcox.terminals.android.ui.ServerDrivenRenderer
import com.curtcox.terminals.android.ui.ServerDrivenRendererPlaceholder

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
                        modifier = Modifier.weight(1f),
                    )
                    Button(onClick = viewModel::connect, enabled = state.connectionState == ConnectionState.ReadyToConnect) {
                        Text("Connect")
                    }
                }

                state.lastError?.let {
                    Text(it, color = MaterialTheme.colorScheme.error)
                }

                Text("Status: ${state.connectionState}", style = MaterialTheme.typography.titleMedium)
                Spacer(modifier = Modifier.height(8.dp))
                SelectionContainer {
                    Text(state.diagnosticsText, fontFamily = FontFamily.Monospace)
                }

                state.serverRoot?.let { root ->
                    ServerDrivenRenderer(
                        root = root,
                        onAction = viewModel::sendUiAction,
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
