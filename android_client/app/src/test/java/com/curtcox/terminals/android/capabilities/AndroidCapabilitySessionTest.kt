package com.curtcox.terminals.android.capabilities

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test
import terminals.capabilities.v1.Capabilities

class AndroidCapabilitySessionTest {
    @Test
    fun snapshotBuildsScreenIdentityDisplayAndGeneration() {
        val probe = MutableProbe(baseInput())
        val session = AndroidCapabilitySession("device-1", probe)

        val report = session.snapshot()

        assertTrue(report.fullSnapshot)
        assertEquals(1, report.generation)
        assertEquals("device-1", report.capabilities.deviceId)
        assertEquals("Kitchen Fire", report.capabilities.identity.deviceName)
        assertEquals("android", report.capabilities.identity.platform)
        assertEquals(1280, report.capabilities.screen.width)
        assertEquals(800, report.capabilities.screen.height)
        assertEquals("landscape", report.capabilities.screen.orientation)
        assertEquals(1, report.capabilities.displaysCount)
        assertEquals("primary", report.capabilities.displaysList.first().displayId)
    }

    @Test
    fun permissionDeniedSensitiveHardwareIsNotAdvertised() {
        val report = AndroidCapabilitySession(
            "device-1",
            MutableProbe(
                baseInput().copy(
                    hardware = baseHardware.copy(microphone = true, frontCamera = true, backCamera = true),
                    permissions = PermissionCapabilityState(
                        microphoneGranted = false,
                        cameraGranted = false,
                    ),
                ),
            ),
        ).snapshot()

        assertFalse(report.capabilities.hasMicrophone())
        assertFalse(report.capabilities.hasCamera())
    }

    @Test
    fun permissionGrantedSensitiveHardwareIsAdvertised() {
        val report = AndroidCapabilitySession(
            "device-1",
            MutableProbe(
                baseInput().copy(
                    hardware = baseHardware.copy(microphone = true, frontCamera = true),
                    permissions = PermissionCapabilityState(
                        microphoneGranted = true,
                        cameraGranted = true,
                    ),
                ),
            ),
        ).snapshot()

        assertTrue(report.capabilities.hasMicrophone())
        assertTrue(report.capabilities.hasCamera())
        assertNotNull(report.capabilities.camera.front)
    }

    @Test
    fun deltaIsNullWhenCapabilitiesDoNotChange() {
        val probe = MutableProbe(baseInput())
        val session = AndroidCapabilitySession("device-1", probe)

        session.snapshot()

        assertNull(session.deltaIfChanged("unchanged"))
    }

    @Test
    fun deltaIncrementsGenerationWhenDisplayChanges() {
        val probe = MutableProbe(baseInput())
        val session = AndroidCapabilitySession("device-1", probe)

        session.snapshot()
        probe.input = baseInput().copy(
            screenMetrics = AndroidScreenMetrics(
                widthPx = 800,
                heightPx = 1280,
                density = 2f,
                orientation = "portrait",
            ),
        )

        val delta = session.deltaIfChanged("orientation")

        assertNotNull(delta)
        assertFalse(delta!!.fullSnapshot)
        assertEquals(2, delta.generation)
        assertEquals("orientation", delta.reason)
        assertEquals("portrait", delta.capabilities.screen.orientation)
    }

    @Test
    fun staleGenerationRebaselineProducesFullSnapshot() {
        val probe = MutableProbe(baseInput())
        val session = AndroidCapabilitySession("device-1", probe)

        session.snapshot()
        val report = session.rebaselineAfterStaleGeneration()

        assertTrue(report.fullSnapshot)
        assertEquals("stale-generation", report.reason)
        assertEquals(2, report.generation)
    }

    private class MutableProbe(
        var input: AndroidCapabilitySnapshotInput,
    ) : AndroidCapabilityProbe {
        override fun current(): AndroidCapabilitySnapshotInput = input
    }

    private companion object {
        val identity: Capabilities.DeviceIdentity = Capabilities.DeviceIdentity.newBuilder()
            .setDeviceName("Kitchen Fire")
            .setDeviceType("tablet")
            .setPlatform("android")
            .build()

        val baseHardware = AndroidHardwareCapabilities(
            touchSupported = true,
            maxTouchPoints = 10,
            audioOutput = true,
            accelerometer = true,
            haptics = true,
        )

        fun baseInput(): AndroidCapabilitySnapshotInput =
            AndroidCapabilitySnapshotInput(
                identity = identity,
                screenMetrics = AndroidScreenMetrics(
                    widthPx = 1280,
                    heightPx = 800,
                    density = 2f,
                    orientation = "landscape",
                    safeArea = AndroidInsets(topPx = 24),
                ),
                hardware = baseHardware,
                power = PowerCapabilityState(
                    batteryLevel = 1.5f,
                    charging = true,
                ),
            )
    }
}
