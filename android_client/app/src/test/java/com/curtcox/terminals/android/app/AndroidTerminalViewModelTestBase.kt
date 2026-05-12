package com.curtcox.terminals.android.app

import com.curtcox.terminals.android.capabilities.AndroidCapabilityProbe
import com.curtcox.terminals.android.capabilities.AndroidCapabilitySnapshotInput
import com.curtcox.terminals.android.capabilities.AndroidHardwareCapabilities
import com.curtcox.terminals.android.capabilities.AndroidScreenMetrics
import com.curtcox.terminals.android.capabilities.PermissionCapabilityState
import com.curtcox.terminals.android.connection.AndroidControlResponseSink
import com.curtcox.terminals.android.connection.AndroidControlSession
import com.curtcox.terminals.android.connection.ControlSessionStatus
import com.curtcox.terminals.android.connection.EndpointResolution
import com.curtcox.terminals.android.diagnostics.AndroidBuildMetadata
import com.curtcox.terminals.android.discovery.AndroidNsdDiscovery
import com.curtcox.terminals.android.discovery.DiscoveredServer
import com.curtcox.terminals.android.media.AndroidLiveMediaSession
import com.curtcox.terminals.android.media.AndroidMediaEngine
import com.curtcox.terminals.android.media.AndroidMediaPermissionProbe
import com.curtcox.terminals.android.media.AndroidWebRtcAdapter
import com.curtcox.terminals.android.media.AndroidWebRtcSupport
import com.curtcox.terminals.android.platform.AndroidNetworkMonitor
import com.curtcox.terminals.android.platform.AndroidNetworkStateProvider
import com.curtcox.terminals.android.platform.AndroidNotificationDelivery
import com.curtcox.terminals.android.platform.AndroidPermissionRequester
import com.curtcox.terminals.android.platform.AndroidTerminalSpeech
import com.curtcox.terminals.android.ui.ServerDrivenAction
import kotlinx.coroutines.CompletableDeferred
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.ExperimentalCoroutinesApi
import kotlinx.coroutines.test.StandardTestDispatcher
import kotlinx.coroutines.test.resetMain
import kotlinx.coroutines.test.setMain
import org.junit.After
import org.junit.Before
import terminals.capabilities.v1.Capabilities
import terminals.control.v1.Control
import terminals.diagnostics.v1.Diagnostics

@OptIn(ExperimentalCoroutinesApi::class)
abstract class AndroidTerminalViewModelTestBase {
    /** Fresh queue per test so a leaked heartbeat/sensor loop cannot stall a later `advanceUntilIdle()`. */
    var testDispatcher = StandardTestDispatcher()

    @Before
    fun setUp() {
        testDispatcher = StandardTestDispatcher()
        Dispatchers.setMain(testDispatcher)
    }

    @After
    fun tearDown() {
        Dispatchers.resetMain()
    }

    fun viewModel(
        session: FakeSession,
        notificationDelivery: AndroidNotificationDelivery = AndroidNotificationDelivery.none(),
        speechDelivery: AndroidTerminalSpeech = AndroidTerminalSpeech.none(),
        webRtcAdapter: AndroidWebRtcAdapter = AndroidWebRtcAdapter { AndroidWebRtcSupport(supported = true) },
        mediaEngine: AndroidMediaEngine = AndroidMediaEngine(
            liveMedia = AndroidLiveMediaSession.fromAdapter(webRtcAdapter),
        ),
        networkStateProvider: AndroidNetworkStateProvider = AndroidNetworkStateProvider.unknown(),
        mediaPermissionProbe: AndroidMediaPermissionProbe = AndroidMediaPermissionProbe.unavailable(),
        permissionRequester: AndroidPermissionRequester = AndroidPermissionRequester.none(),
        networkMonitor: AndroidNetworkMonitor = AndroidNetworkMonitor.none(),
        heartbeatIntervalMillis: Long = 0,
        sensorTelemetryIntervalMillis: Long = 0,
        capabilityMonitorIntervalMillis: Long = 0,
        bugReportScreenshotCapture: () -> ByteArray? = { null },
    ): AndroidTerminalViewModel =
        AndroidTerminalViewModel(
            AndroidClientDependencies(
                buildMetadata = AndroidBuildMetadata("0.1.0-test", "sha", "date"),
                notificationDelivery = notificationDelivery,
                speechDelivery = speechDelivery,
                mediaEngine = mediaEngine,
                networkStateProvider = networkStateProvider,
                networkMonitor = networkMonitor,
                mediaPermissionProbe = mediaPermissionProbe,
                webRtcAdapter = webRtcAdapter,
                permissionRequester = permissionRequester,
                heartbeatIntervalMillis = heartbeatIntervalMillis,
                sensorTelemetryIntervalMillis = sensorTelemetryIntervalMillis,
                capabilityMonitorIntervalMillis = capabilityMonitorIntervalMillis,
                bugReportScreenshotCapture = bugReportScreenshotCapture,
                sessionFactory = { sink ->
                    session.sink = sink
                    session
                },
            ),
        )

    class FakeNetworkMonitor : AndroidNetworkMonitor {
        private var onChanged: (() -> Unit)? = null
        var startCount = 0
        var stopCount = 0

        override fun start(onChanged: () -> Unit) {
            startCount += 1
            this.onChanged = onChanged
        }

        override fun stop() {
            stopCount += 1
            onChanged = null
        }

        fun emitChange() {
            onChanged?.invoke()
        }
    }

    class FakePermissionRequester(
        val grantedPermissions: MutableSet<String> = mutableSetOf(),
    ) : AndroidPermissionRequester {
        val requests = mutableListOf<String>()
        var nextGrant: Boolean = false

        override fun hasPermission(permission: String): Boolean = grantedPermissions.contains(permission)

        override fun requestPermission(permission: String, onResult: (Boolean) -> Unit) {
            requests += permission
            onResult(nextGrant)
        }
    }

    class FakeSession(
        private val connectError: Throwable? = null,
        private val capabilityDeltaSent: Boolean = false,
        private val heartbeatError: Throwable? = null,
        private val sensorTelemetryError: Throwable? = null,
        private val connectGate: CompletableDeferred<Unit>? = null,
        override val lastRegisteredCapabilities: Capabilities.DeviceCapabilities? = null,
    ) : AndroidControlSession {
        override var status: ControlSessionStatus = ControlSessionStatus()
        lateinit var sink: AndroidControlResponseSink
        var connectedEndpoint: EndpointResolution? = null
        val actions = mutableListOf<ServerDrivenAction>()
        val bugReports = mutableListOf<Diagnostics.BugReport>()

        /** Per-call: `true` means [sendBugReport] throws for that attempt (flush continues). */
        var bugReportFailurePattern: List<Boolean> = emptyList()
        private var bugReportAttemptIndex = 0
        val keyTexts = mutableListOf<String>()
        val streamReadyIds = mutableListOf<String>()
        val systemCommands = mutableListOf<Pair<String, String>>()
        val applicationLaunchCommands = mutableListOf<Pair<String, String>>()
        val playbackMetadataQueries = mutableListOf<Triple<String, String, String>>()
        val capabilityDeltaReasons = mutableListOf<String>()
        val privacyModeCalls = mutableListOf<Boolean>()
        var rebaselineCount = 0
        var heartbeatCount = 0
        var sensorTelemetryCount = 0
        var closeCount = 0

        override fun setPrivacyMode(enabled: Boolean) {
            privacyModeCalls += enabled
        }

        override suspend fun connect(endpoint: EndpointResolution) {
            connectGate?.await()
            connectError?.let { throw it }
            connectedEndpoint = endpoint
            status = status.copy(connected = true, endpoint = endpoint)
        }

        override suspend fun sendHeartbeat() {
            heartbeatError?.let { throw it }
            heartbeatCount += 1
        }

        override suspend fun sendSensorTelemetry(): Boolean {
            sensorTelemetryError?.let { throw it }
            sensorTelemetryCount += 1
            return true
        }

        override suspend fun sendUiAction(action: ServerDrivenAction) {
            actions += action
        }

        override suspend fun sendWebRtcSignal(signal: Control.WebRTCSignal) = Unit

        override suspend fun sendStreamReady(streamId: String) {
            streamReadyIds += streamId
        }

        override suspend fun sendKeyText(text: String) {
            keyTexts += text
        }

        override suspend fun sendSystemCommand(requestId: String, intent: String) {
            systemCommands += requestId to intent
        }

        override suspend fun sendPlaybackMetadataQuery(
            requestId: String,
            artifactId: String,
            targetDeviceId: String,
        ) {
            playbackMetadataQueries += Triple(requestId, artifactId, targetDeviceId)
        }

        override suspend fun sendApplicationLaunchCommand(
            requestId: String,
            intent: String,
            arguments: Map<String, String>,
        ) {
            applicationLaunchCommands += requestId to intent
        }

        override suspend fun sendBugReport(report: Diagnostics.BugReport) {
            if (bugReportFailurePattern.isNotEmpty()) {
                val fail = bugReportFailurePattern.getOrElse(bugReportAttemptIndex) { false }
                bugReportAttemptIndex++
                if (fail) {
                    error("simulated-bug-report-send-failure")
                }
            }
            bugReports += report
        }

        override suspend fun sendCapabilityDeltaIfChanged(reason: String): Boolean {
            capabilityDeltaReasons += reason
            return capabilityDeltaSent
        }

        override suspend fun rebaselineCapabilitiesAfterStaleGeneration() {
            rebaselineCount += 1
        }

        override suspend fun close() {
            closeCount += 1
        }
    }

    class FakeDiscovery : AndroidNsdDiscovery {
        private var onServer: ((DiscoveredServer) -> Unit)? = null
        private var onError: ((String) -> Unit)? = null
        var startCount = 0
        var stopCount = 0

        override fun start(onServer: (DiscoveredServer) -> Unit, onError: (String) -> Unit) {
            startCount += 1
            this.onServer = onServer
            this.onError = onError
        }

        override fun stop() {
            stopCount += 1
        }

        fun emit(server: DiscoveredServer) {
            onServer?.invoke(server)
        }

        fun fail(message: String) {
            onError?.invoke(message)
        }
    }

    class FakeCapabilityProbe(
        var permissions: PermissionCapabilityState = PermissionCapabilityState(),
        var hardware: AndroidHardwareCapabilities = AndroidHardwareCapabilities(),
    ) : AndroidCapabilityProbe {
        override fun current(): AndroidCapabilitySnapshotInput =
            AndroidCapabilitySnapshotInput(
                identity = Capabilities.DeviceIdentity.newBuilder()
                    .setDeviceName("test-terminal")
                    .setDeviceType("tablet")
                    .setPlatform("android")
                    .build(),
                screenMetrics = AndroidScreenMetrics(
                    widthPx = 1280,
                    heightPx = 800,
                    density = 1.0f,
                    orientation = "landscape",
                ),
                permissions = permissions,
                hardware = hardware,
            )
    }
}
