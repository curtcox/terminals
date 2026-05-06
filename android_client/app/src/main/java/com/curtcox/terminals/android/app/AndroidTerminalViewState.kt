package com.curtcox.terminals.android.app

import terminals.ui.v1.Ui

data class AndroidTerminalViewState(
    val endpointText: String = "",
    val connectionState: ConnectionState = ConnectionState.Disconnected,
    val lastError: String? = null,
    val diagnosticsText: String = "",
    val serverRoot: Ui.Node? = null,
    val lastNotificationTitle: String? = null,
    val lastNotificationBody: String? = null,
    val lastMediaRequestId: String? = null,
    val lastMediaStatus: String? = null,
    val lastTransition: String? = null,
    val permissionEducation: PermissionEducationState = PermissionEducationState(),
)

enum class ConnectionState {
    Disconnected,
    InvalidEndpoint,
    ReadyToConnect,
    Connecting,
    Connected,
}

data class PermissionEducationState(
    val notificationsGranted: Boolean = false,
    val microphonePresent: Boolean = false,
    val microphoneAvailable: Boolean = false,
    val cameraPresent: Boolean = false,
    val cameraAvailable: Boolean = false,
) {
    val messages: List<String>
        get() = buildList {
            if (!notificationsGranted) {
                add("Notifications are disabled; server notifications will stay in terminal diagnostics.")
            }
            if (microphonePresent && !microphoneAvailable) {
                add("Microphone capture is unavailable until hardware and permission are both present.")
            }
            if (cameraPresent && !cameraAvailable) {
                add("Camera capture is unavailable until hardware and permission are both present.")
            }
        }
}
