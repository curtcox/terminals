package com.curtcox.terminals.android.ui

import androidx.compose.material3.Text
import androidx.compose.ui.test.assertIsDisplayed
import androidx.compose.ui.test.junit4.createComposeRule
import androidx.compose.ui.test.onNodeWithTag
import androidx.compose.ui.test.onNodeWithText
import androidx.compose.ui.test.performClick
import org.junit.Assert.assertEquals
import org.junit.Rule
import org.junit.Test
import terminals.ui.v1.Ui

class ServerDrivenRendererTest {
    @get:Rule
    val compose = createComposeRule()

    @Test
    fun rendersTextNodeWithStableTag() {
        val root = node("title") {
            text = Ui.TextWidget.newBuilder().setValue("Kitchen terminal").build()
        }

        compose.setContent { render(root) }

        compose.onNodeWithText("Kitchen terminal").assertIsDisplayed()
        compose.onNodeWithTag("terminal-node-title").assertIsDisplayed()
    }

    @Test
    fun buttonEmitsGenericServerDrivenAction() {
        val actions = mutableListOf<ServerDrivenAction>()
        val root = node("start") {
            button = Ui.ButtonWidget.newBuilder().setLabel("Start").setAction("begin").build()
        }

        compose.setContent { render(root, actions::add) }
        compose.onNodeWithText("Start").performClick()

        assertEquals(listOf(ServerDrivenAction("start", "begin", "pressed")), actions)
    }

    @Test
    fun delegatesImageAndMediaSurfaces() {
        val root = Ui.Node.newBuilder()
            .setId("root")
            .setStack(Ui.StackWidget.getDefaultInstance())
            .addChildren(node("image") {
                image = Ui.ImageWidget.newBuilder().setUrl("https://example.test/image.png").build()
            })
            .addChildren(node("camera") {
                videoSurface = Ui.VideoSurfaceWidget.newBuilder().setTrackId("track-1").build()
            })
            .build()

        compose.setContent {
            ServerDrivenRenderer(
                root = root,
                onAction = {},
                mediaSurface = { Text("media:$it") },
                imageLoader = { url, _ -> Text("image:$url") },
            )
        }

        compose.onNodeWithText("image:https://example.test/image.png").assertIsDisplayed()
        compose.onNodeWithText("media:track-1").assertIsDisplayed()
    }

    @Test
    fun malformedNodeUsesFallbackPolicy() {
        compose.setContent { render(Ui.Node.newBuilder().setId("unknown").build()) }

        compose.onNodeWithText("Unsupported terminal widget").assertIsDisplayed()
    }

    @Test
    fun keepAwakeWidgetInvokesInjectedDeviceControlEffect() {
        val calls = mutableListOf<Boolean>()
        val root = node("wake") {
            keepAwake = Ui.KeepAwakeWidget.newBuilder().setEnabled(true).build()
        }

        compose.setContent {
            ServerDrivenRenderer(
                root = root,
                onAction = {},
                imageLoader = { url, _ -> Text(url) },
                deviceControlEffects = DeviceControlEffects(setKeepAwake = calls::add),
            )
        }

        compose.onNodeWithText("keep_awake=true").assertIsDisplayed()
        compose.waitUntil { calls == listOf(true) }
    }

    @Test
    fun fullscreenWidgetInvokesInjectedDeviceControlEffect() {
        val calls = mutableListOf<Boolean>()
        val root = node("full") {
            fullscreen = Ui.FullscreenWidget.newBuilder().setEnabled(true).build()
        }

        compose.setContent {
            ServerDrivenRenderer(
                root = root,
                onAction = {},
                imageLoader = { url, _ -> Text(url) },
                deviceControlEffects = DeviceControlEffects(setFullscreen = calls::add),
            )
        }

        compose.onNodeWithText("fullscreen=true").assertIsDisplayed()
        compose.waitUntil { calls == listOf(true) }
    }

    @Test
    fun brightnessWidgetInvokesInjectedDeviceControlEffect() {
        val calls = mutableListOf<Double>()
        val root = node("brightness") {
            brightness = Ui.BrightnessWidget.newBuilder().setValue(0.42).build()
        }

        compose.setContent {
            ServerDrivenRenderer(
                root = root,
                onAction = {},
                imageLoader = { url, _ -> Text(url) },
                deviceControlEffects = DeviceControlEffects(setBrightness = calls::add),
            )
        }

        compose.onNodeWithText("brightness=0.42").assertIsDisplayed()
        compose.waitUntil { calls == listOf(0.42) }
    }

    private fun render(
        root: Ui.Node,
        onAction: (ServerDrivenAction) -> Unit = {},
    ) {
        ServerDrivenRenderer(
            root = root,
            onAction = onAction,
            imageLoader = { url, _ -> Text(url) },
        )
    }

    private fun node(id: String, configure: Ui.Node.Builder.() -> Unit): Ui.Node =
        Ui.Node.newBuilder().setId(id).apply(configure).build()
}
