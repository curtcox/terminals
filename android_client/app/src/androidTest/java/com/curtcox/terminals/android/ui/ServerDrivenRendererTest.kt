package com.curtcox.terminals.android.ui

import androidx.compose.foundation.layout.Box
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.semantics.SemanticsProperties
import androidx.compose.ui.test.assertIsDisplayed
import androidx.compose.ui.test.assertIsFocused
import androidx.compose.ui.test.fetchSemanticsNode
import androidx.compose.ui.test.junit4.createComposeRule
import androidx.compose.ui.test.onNodeWithTag
import androidx.compose.ui.test.onNodeWithText
import androidx.compose.ui.test.performClick
import androidx.compose.ui.test.performImeAction
import androidx.compose.ui.test.performTextInput
import androidx.compose.ui.test.performTouchInput
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNotNull
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

        assertEquals(listOf(ServerDrivenAction("start", "begin")), actions)
    }

    @Test
    fun buttonWithoutComponentIdFallsBackToWidgetName() {
        val actions = mutableListOf<ServerDrivenAction>()
        val root = Ui.Node.newBuilder()
            .setButton(Ui.ButtonWidget.newBuilder().setLabel("Tap").build())
            .build()

        compose.setContent { render(root, actions::add) }
        compose.onNodeWithText("Tap").performClick()

        assertEquals(listOf(ServerDrivenAction("button", "tap")), actions)
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
    fun videoAndAudioVisualizerUseSeparateShellSurfacesWhenBothProvided() {
        val root = Ui.Node.newBuilder()
            .setId("root")
            .setStack(Ui.StackWidget.getDefaultInstance())
            .addChildren(node("cam") {
                videoSurface = Ui.VideoSurfaceWidget.newBuilder().setTrackId("track-a").build()
            })
            .addChildren(node("meter") {
                audioVisualizer = Ui.AudioVisualizerWidget.newBuilder().setStreamId("stream-b").build()
            })
            .build()

        compose.setContent {
            ServerDrivenRenderer(
                root = root,
                onAction = {},
                mediaSurface = { Text("video:$it") },
                audioVisualizerSurface = { Text("audio:$it") },
                imageLoader = { _, _ -> Box {} },
            )
        }

        compose.onNodeWithText("video:track-a").assertIsDisplayed()
        compose.onNodeWithText("audio:stream-b").assertIsDisplayed()
    }

    @Test
    fun imageLoaderReceivesContentDescriptionFromProps() {
        val root = Ui.Node.newBuilder()
            .setId("hero")
            .putProps("contentDescription", "Terminal logo")
            .setImage(Ui.ImageWidget.newBuilder().setUrl("https://example.test/logo.svg").build())
            .build()

        compose.setContent {
            ServerDrivenRenderer(
                root = root,
                onAction = {},
                imageLoader = { url, desc -> Text("image:$url|alt:${desc ?: "null"}") },
            )
        }

        compose.onNodeWithText("image:https://example.test/logo.svg|alt:Terminal logo").assertIsDisplayed()
    }

    @Test
    fun videoSurfaceWithoutBuilderUsesFlutterStylePlaceholder() {
        val root = node("cam") {
            videoSurface = Ui.VideoSurfaceWidget.newBuilder().setTrackId("track-z").build()
        }

        compose.setContent {
            ServerDrivenRenderer(
                root = root,
                onAction = {},
                imageLoader = { _, _ -> Box {} },
            )
        }

        compose.onNodeWithText("Video surface").assertIsDisplayed()
        compose.onNodeWithText("track-z").assertIsDisplayed()
    }

    @Test
    fun audioVisualizerWithoutBuilderUsesFlutterStylePlaceholder() {
        val root = node("viz") {
            audioVisualizer = Ui.AudioVisualizerWidget.newBuilder().setStreamId("s9").build()
        }

        compose.setContent {
            ServerDrivenRenderer(
                root = root,
                onAction = {},
                imageLoader = { _, _ -> Box {} },
            )
        }

        compose.onNodeWithText("Audio level").assertIsDisplayed()
        compose.onNodeWithText("s9").assertIsDisplayed()
    }

    @Test
    fun malformedNodeUsesFallbackPolicy() {
        compose.setContent { render(Ui.Node.newBuilder().setId("unknown").build()) }

        compose.onNodeWithText("Unsupported UI node").assertIsDisplayed()
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

        compose.onNodeWithText("Keep awake enabled").assertIsDisplayed()
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

        compose.onNodeWithText("Fullscreen enabled").assertIsDisplayed()
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

        compose.onNodeWithText("Brightness hint").assertIsDisplayed()
        compose.onNodeWithText("0.42").assertIsDisplayed()
        compose.waitUntil { calls == listOf(0.42) }
    }

    @Test
    fun brightnessWidgetClampsValueToFlutterRangeForLabelAndEffect() {
        val calls = mutableListOf<Double>()
        val root = node("brightness") {
            brightness = Ui.BrightnessWidget.newBuilder().setValue(-0.5).build()
        }

        compose.setContent {
            ServerDrivenRenderer(
                root = root,
                onAction = {},
                imageLoader = { url, _ -> Text(url) },
                deviceControlEffects = DeviceControlEffects(setBrightness = calls::add),
            )
        }

        compose.onNodeWithText("Brightness hint").assertIsDisplayed()
        compose.onNodeWithText("0.00").assertIsDisplayed()
        compose.waitUntil { calls == listOf(0.0) }
    }

    @Test
    fun keepAwakeWidgetRendersChildContentLikeFlutter() {
        val root = Ui.Node.newBuilder()
            .setId("wake")
            .setKeepAwake(Ui.KeepAwakeWidget.newBuilder().setEnabled(true).build())
            .addChildren(node("kiosk-banner") {
                text = Ui.TextWidget.newBuilder().setValue("Kiosk active").build()
            })
            .build()

        compose.setContent { render(root) }

        compose.onNodeWithText("Keep awake enabled").assertIsDisplayed()
        compose.onNodeWithText("Kiosk active").assertIsDisplayed()
        compose.onNodeWithTag("terminal-node-kiosk-banner").assertIsDisplayed()
    }

    @Test
    fun fullscreenWidgetRendersChildContentLikeFlutter() {
        val root = Ui.Node.newBuilder()
            .setId("full")
            .setFullscreen(Ui.FullscreenWidget.newBuilder().setEnabled(true).build())
            .addChildren(node("immersive-label") {
                text = Ui.TextWidget.newBuilder().setValue("Immersive content").build()
            })
            .build()

        compose.setContent { render(root) }

        compose.onNodeWithText("Fullscreen enabled").assertIsDisplayed()
        compose.onNodeWithText("Immersive content").assertIsDisplayed()
        compose.onNodeWithTag("terminal-node-immersive-label").assertIsDisplayed()
    }

    @Test
    fun brightnessWidgetRendersChildContentLikeFlutter() {
        val root = Ui.Node.newBuilder()
            .setId("brightness")
            .setBrightness(Ui.BrightnessWidget.newBuilder().setValue(0.6).build())
            .addChildren(node("brightness-hint") {
                text = Ui.TextWidget.newBuilder().setValue("Bright wall mode").build()
            })
            .build()

        compose.setContent { render(root) }

        compose.onNodeWithText("Brightness hint").assertIsDisplayed()
        compose.onNodeWithText("0.60").assertIsDisplayed()
        compose.onNodeWithText("Bright wall mode").assertIsDisplayed()
        compose.onNodeWithTag("terminal-node-brightness-hint").assertIsDisplayed()
    }

    @Test
    fun paddingWithMultipleChildrenRendersThemAsColumnLikeFlutter() {
        val root = Ui.Node.newBuilder()
            .setId("padded")
            .setPadding(Ui.PaddingWidget.newBuilder().setAll(8).build())
            .addChildren(node("first-line") {
                text = Ui.TextWidget.newBuilder().setValue("First line").build()
            })
            .addChildren(node("second-line") {
                text = Ui.TextWidget.newBuilder().setValue("Second line").build()
            })
            .build()

        compose.setContent { render(root) }

        compose.onNodeWithText("First line").assertIsDisplayed()
        compose.onNodeWithText("Second line").assertIsDisplayed()
        compose.onNodeWithTag("terminal-node-first-line").assertIsDisplayed()
        compose.onNodeWithTag("terminal-node-second-line").assertIsDisplayed()
    }

    @Test
    fun textInputSubmitsOnImeDone() {
        val actions = mutableListOf<ServerDrivenAction>()
        val root = node("name") {
            textInput = Ui.TextInputWidget.newBuilder().setPlaceholder("Name").build()
        }

        compose.setContent { render(root, actions::add) }
        compose.onNodeWithTag("terminal-node-name").performTextInput("Ada")
        compose.onNodeWithTag("terminal-node-name").performImeAction()

        assertEquals(ServerDrivenAction("name", "submit", "Ada"), actions.last())
    }

    @Test
    fun textInputAutofocusRequestsFocus() {
        val root = node("focus") {
            textInput = Ui.TextInputWidget.newBuilder()
                .setPlaceholder("Endpoint")
                .setAutofocus(true)
                .build()
        }

        compose.setContent { render(root) }

        compose.waitForIdle()
        compose.onNodeWithTag("terminal-node-focus").assertIsFocused()
    }

    @Test
    fun toggleEmitsCheckedState() {
        val actions = mutableListOf<ServerDrivenAction>()
        val root = node("enabled") {
            toggle = Ui.ToggleWidget.newBuilder().setValue(false).build()
        }

        compose.setContent { render(root, actions::add) }
        compose.onNodeWithTag("terminal-node-enabled").performClick()

        assertEquals(listOf(ServerDrivenAction("enabled", "toggle", "true")), actions)
    }

    @Test
    fun toggleWithoutComponentIdFallsBackToWidgetName() {
        val actions = mutableListOf<ServerDrivenAction>()
        val root = Ui.Node.newBuilder()
            .setToggle(Ui.ToggleWidget.newBuilder().setValue(true).build())
            .build()

        compose.setContent { render(root, actions::add) }
        compose.onNodeWithTag("terminal-node-toggle").performClick()

        assertEquals(listOf(ServerDrivenAction("toggle", "toggle", "false")), actions)
    }

    @Test
    fun dropdownEmitsSelectedOption() {
        val actions = mutableListOf<ServerDrivenAction>()
        val root = node("mode") {
            dropdown = Ui.DropdownWidget.newBuilder()
                .setValue("Manual")
                .addOptions("Manual")
                .addOptions("Auto")
                .build()
        }

        compose.setContent { render(root, actions::add) }
        compose.onNodeWithText("Manual").performClick()
        compose.onNodeWithText("Auto").performClick()

        assertEquals(listOf(ServerDrivenAction("mode", "select", "Auto")), actions)
    }

    @Test
    fun dropdownValueNotInOptionsShowsFirstOptionLikeFlutter() {
        val actions = mutableListOf<ServerDrivenAction>()
        val root = node("mode") {
            dropdown = Ui.DropdownWidget.newBuilder()
                .setValue("Unknown")
                .addOptions("Alpha")
                .addOptions("Beta")
                .build()
        }

        compose.setContent { render(root, actions::add) }
        compose.onNodeWithText("Alpha").assertIsDisplayed()
        compose.onNodeWithText("Alpha").performClick()
        compose.onNodeWithText("Beta").performClick()

        assertEquals(listOf(ServerDrivenAction("mode", "select", "Beta")), actions)
    }

    @Test
    fun dropdownWithNoOptionsShowsSelectHint() {
        val root = node("empty") {
            dropdown = Ui.DropdownWidget.newBuilder().setValue("x").build()
        }

        compose.setContent { render(root) }

        compose.onNodeWithText("Select option").assertIsDisplayed()
    }

    @Test
    fun gestureAreaEmitsConfiguredAction() {
        val actions = mutableListOf<ServerDrivenAction>()
        val root = Ui.Node.newBuilder()
            .setId("surface")
            .setGestureArea(Ui.GestureAreaWidget.newBuilder().setAction("primary"))
            .addChildren(node("label") {
                text = Ui.TextWidget.newBuilder().setValue("Tap target").build()
            })
            .build()

        compose.setContent { render(root, actions::add) }
        compose.onNodeWithText("Tap target").performClick()

        assertEquals(listOf(ServerDrivenAction("surface", "primary")), actions)
    }

    @Test
    fun gestureAreaWithoutComponentIdFallsBackToWidgetName() {
        val actions = mutableListOf<ServerDrivenAction>()
        val root = Ui.Node.newBuilder()
            .setGestureArea(Ui.GestureAreaWidget.newBuilder().setAction("tap-anywhere"))
            .addChildren(node("label") {
                text = Ui.TextWidget.newBuilder().setValue("Tap area").build()
            })
            .build()

        compose.setContent { render(root, actions::add) }
        compose.onNodeWithText("Tap area").performClick()

        assertEquals(listOf(ServerDrivenAction("gesture_area", "tap-anywhere")), actions)
    }

    @Test
    fun gestureAreaWithNoChildrenExposesMinimumTapTarget() {
        val actions = mutableListOf<ServerDrivenAction>()
        val root = Ui.Node.newBuilder()
            .setId("tap-target")
            .setGestureArea(Ui.GestureAreaWidget.newBuilder().setAction("tap-empty").build())
            .build()

        compose.setContent { render(root, actions::add) }
        compose.onNodeWithTag("terminal-node-tap-target").performClick()

        assertEquals(listOf(ServerDrivenAction("tap-target", "tap-empty")), actions)
    }

    @Test
    fun stackPreservesChildOrderAndStableTags() {
        val root = Ui.Node.newBuilder()
            .setId("root")
            .setStack(Ui.StackWidget.getDefaultInstance())
            .addChildren(node("first") {
                text = Ui.TextWidget.newBuilder().setValue("Alpha").build()
            })
            .addChildren(node("second") {
                text = Ui.TextWidget.newBuilder().setValue("Beta").build()
            })
            .build()

        compose.setContent { render(root) }

        compose.onNodeWithTag("terminal-node-first").assertIsDisplayed()
        compose.onNodeWithTag("terminal-node-second").assertIsDisplayed()
        compose.onNodeWithText("Alpha").assertIsDisplayed()
        compose.onNodeWithText("Beta").assertIsDisplayed()
    }

    @Test
    fun rowRendersChildrenWithStableTags() {
        val root = Ui.Node.newBuilder()
            .setId("root")
            .setRow(Ui.RowWidget.getDefaultInstance())
            .addChildren(node("left") {
                text = Ui.TextWidget.newBuilder().setValue("Left").build()
            })
            .addChildren(node("right") {
                text = Ui.TextWidget.newBuilder().setValue("Right").build()
            })
            .build()

        compose.setContent { render(root) }

        compose.onNodeWithTag("terminal-node-left").assertIsDisplayed()
        compose.onNodeWithTag("terminal-node-right").assertIsDisplayed()
        compose.onNodeWithText("Left").assertIsDisplayed()
        compose.onNodeWithText("Right").assertIsDisplayed()
    }

    @Test
    fun gridRendersChildrenWithStableTags() {
        val root = Ui.Node.newBuilder()
            .setId("root")
            .setGrid(Ui.GridWidget.newBuilder().setColumns(2).build())
            .addChildren(node("c0") {
                text = Ui.TextWidget.newBuilder().setValue("Cell0").build()
            })
            .addChildren(node("c1") {
                text = Ui.TextWidget.newBuilder().setValue("Cell1").build()
            })
            .build()

        compose.setContent { render(root) }

        compose.onNodeWithTag("terminal-node-c0").assertIsDisplayed()
        compose.onNodeWithTag("terminal-node-c1").assertIsDisplayed()
    }

    @Test
    fun gridWithMoreChildrenThanColumnsRendersAllCells() {
        val root = Ui.Node.newBuilder()
            .setId("root")
            .setGrid(Ui.GridWidget.newBuilder().setColumns(2).build())
            .addChildren(node("c0") {
                text = Ui.TextWidget.newBuilder().setValue("Cell0").build()
            })
            .addChildren(node("c1") {
                text = Ui.TextWidget.newBuilder().setValue("Cell1").build()
            })
            .addChildren(node("c2") {
                text = Ui.TextWidget.newBuilder().setValue("Cell2").build()
            })
            .build()

        compose.setContent { render(root) }

        compose.onNodeWithText("Cell0").assertIsDisplayed()
        compose.onNodeWithText("Cell1").assertIsDisplayed()
        compose.onNodeWithText("Cell2").assertIsDisplayed()
        compose.onNodeWithTag("terminal-node-c2").assertIsDisplayed()
    }

    @Test
    fun verticalScrollRendersChild() {
        val root = Ui.Node.newBuilder()
            .setId("root")
            .setScroll(
                Ui.ScrollWidget.newBuilder()
                    .setDirectionEnum(Ui.ScrollDirection.SCROLL_DIRECTION_VERTICAL)
                    .build(),
            )
            .addChildren(node("body") {
                text = Ui.TextWidget.newBuilder().setValue("Scrollable content").build()
            })
            .build()

        compose.setContent { render(root) }

        compose.onNodeWithText("Scrollable content").assertIsDisplayed()
    }

    @Test
    fun horizontalScrollDeprecatedStringDirectionRendersChildrenInRow() {
        val root = Ui.Node.newBuilder()
            .setId("scroll")
            .setScroll(Ui.ScrollWidget.newBuilder().setDirection("horizontal").build())
            .addChildren(node("a") {
                text = Ui.TextWidget.newBuilder().setValue("One").build()
            })
            .addChildren(node("b") {
                text = Ui.TextWidget.newBuilder().setValue("Two").build()
            })
            .build()

        compose.setContent { render(root) }

        compose.onNodeWithText("One").assertIsDisplayed()
        compose.onNodeWithText("Two").assertIsDisplayed()
    }

    @Test
    fun horizontalScrollEnumDirectionRendersChildrenInRow() {
        val root = Ui.Node.newBuilder()
            .setId("scroll")
            .setScroll(
                Ui.ScrollWidget.newBuilder()
                    .setDirectionEnum(Ui.ScrollDirection.SCROLL_DIRECTION_HORIZONTAL)
                    .build(),
            )
            .addChildren(node("a") {
                text = Ui.TextWidget.newBuilder().setValue("East").build()
            })
            .addChildren(node("b") {
                text = Ui.TextWidget.newBuilder().setValue("West").build()
            })
            .build()

        compose.setContent { render(root) }

        compose.onNodeWithText("East").assertIsDisplayed()
        compose.onNodeWithText("West").assertIsDisplayed()
    }

    @Test
    fun paddingWrapsChildWithTag() {
        val root = Ui.Node.newBuilder()
            .setId("root")
            .setPadding(Ui.PaddingWidget.newBuilder().setAll(12).build())
            .addChildren(node("inner") {
                text = Ui.TextWidget.newBuilder().setValue("Inset").build()
            })
            .build()

        compose.setContent { render(root) }

        compose.onNodeWithTag("terminal-node-inner").assertIsDisplayed()
        compose.onNodeWithText("Inset").assertIsDisplayed()
    }

    @Test
    fun centerWrapsChild() {
        val root = Ui.Node.newBuilder()
            .setId("root")
            .setCenter(Ui.CenterWidget.getDefaultInstance())
            .addChildren(node("mid") {
                text = Ui.TextWidget.newBuilder().setValue("Centered").build()
            })
            .build()

        compose.setContent { render(root) }

        compose.onNodeWithText("Centered").assertIsDisplayed()
    }

    @Test
    fun expandWrapsChildInsideRow() {
        val root = Ui.Node.newBuilder()
            .setId("root")
            .setRow(Ui.RowWidget.getDefaultInstance())
            .addChildren(
                Ui.Node.newBuilder()
                    .setId("grow")
                    .setExpand(Ui.ExpandWidget.getDefaultInstance())
                    .addChildren(node("inner") {
                        text = Ui.TextWidget.newBuilder().setValue("Flexible").build()
                    })
                    .build(),
            )
            .build()

        compose.setContent { render(root) }

        compose.onNodeWithTag("terminal-node-inner").assertIsDisplayed()
        compose.onNodeWithText("Flexible").assertIsDisplayed()
    }

    @Test
    fun expandInsideStackColumnUsesWeightLikeFlutterExpanded() {
        val root = Ui.Node.newBuilder()
            .setId("root")
            .setStack(Ui.StackWidget.getDefaultInstance())
            .addChildren(
                Ui.Node.newBuilder()
                    .setId("grow")
                    .setExpand(Ui.ExpandWidget.getDefaultInstance())
                    .addChildren(node("inner") {
                        text = Ui.TextWidget.newBuilder().setValue("StackExpand").build()
                    })
                    .build(),
            )
            .build()

        compose.setContent { render(root) }

        compose.onNodeWithText("StackExpand").assertIsDisplayed()
        compose.onNodeWithTag("terminal-node-inner").assertIsDisplayed()
    }

    @Test
    fun overlayRendersChild() {
        val root = Ui.Node.newBuilder()
            .setId("root")
            .setOverlay(Ui.OverlayWidget.getDefaultInstance())
            .addChildren(node("layer") {
                text = Ui.TextWidget.newBuilder().setValue("Overlay text").build()
            })
            .build()

        compose.setContent { render(root) }

        compose.onNodeWithText("Overlay text").assertIsDisplayed()
    }

    @Test
    fun progressWidgetRenders() {
        val root = node("prog") {
            progress = Ui.ProgressWidget.newBuilder().setValue(0.35).build()
        }

        compose.setContent { render(root) }

        compose.onNodeWithTag("terminal-node-prog").assertIsDisplayed()
    }

    @Test
    fun progressWidgetClampsValueAboveOneToMatchFlutter() {
        val root = node("prog") {
            progress = Ui.ProgressWidget.newBuilder().setValue(1.8).build()
        }

        compose.setContent { render(root) }

        val rangeInfo =
            compose.onNodeWithTag("terminal-node-prog").fetchSemanticsNode()
                .config[SemanticsProperties.ProgressBarRangeInfo]
        assertNotNull(rangeInfo)
        assertEquals(1f, rangeInfo!!.current, 0.001f)
    }

    @Test
    fun progressWidgetClampsNegativeValueToMatchFlutter() {
        val root = node("prog") {
            progress = Ui.ProgressWidget.newBuilder().setValue(-0.5).build()
        }

        compose.setContent { render(root) }

        val rangeInfo =
            compose.onNodeWithTag("terminal-node-prog").fetchSemanticsNode()
                .config[SemanticsProperties.ProgressBarRangeInfo]
        assertNotNull(rangeInfo)
        assertEquals(0f, rangeInfo!!.current, 0.001f)
    }

    @Test
    fun sliderWidgetIsDisplayedWithStableTag() {
        val root = node("level") {
            slider = Ui.SliderWidget.newBuilder().setMin(0.0).setMax(10.0).setValue(3.0).build()
        }

        compose.setContent { render(root) }

        compose.onNodeWithTag("terminal-node-level").assertIsDisplayed()
    }

    @Test
    fun sliderEmitsChangeActionWhenAdjusted() {
        val actions = mutableListOf<ServerDrivenAction>()
        val root = node("level") {
            slider = Ui.SliderWidget.newBuilder().setMin(0.0).setMax(100.0).setValue(50.0).build()
        }

        compose.setContent { render(root, actions::add) }

        compose.onNodeWithTag("terminal-node-level").performTouchInput {
            down(center)
            moveTo(center + Offset(160f, 0f))
            up()
        }

        compose.waitUntil(timeoutMillis = 5_000) {
            actions.any { it.componentId == "level" && it.action == "change" && it.value != "50.0" }
        }
    }

    @Test
    fun canvasWithDrawLineRendersWithoutCrash() {
        val lineOp = Ui.DrawOp.newBuilder()
            .setLine(
                Ui.DrawLine.newBuilder()
                    .setX1(0.0)
                    .setY1(0.0)
                    .setX2(20.0)
                    .setY2(20.0)
                    .setStroke("#FF000000")
                    .setStrokeWidth(2.0)
                    .build(),
            )
            .build()
        val root = node("canvas") {
            canvas = Ui.CanvasWidget.newBuilder().addDrawOps(lineOp).build()
        }

        compose.setContent { render(root) }

        compose.onNodeWithTag("terminal-node-canvas").assertExists()
    }

    @Test
    fun canvasWithDrawTextRendersWithoutCrash() {
        val textOp =
            Ui.DrawOp.newBuilder()
                .setText(
                    Ui.DrawText.newBuilder()
                        .setX(4.0)
                        .setY(24.0)
                        .setText("hi")
                        .setFill("#FF000000")
                        .setFontSize(14.0)
                        .setFontFamily("monospace")
                        .build(),
                )
                .build()
        val root = node("canvas") {
            canvas = Ui.CanvasWidget.newBuilder().addDrawOps(textOp).build()
        }

        compose.setContent { render(root) }

        compose.onNodeWithTag("terminal-node-canvas").assertExists()
    }

    @Test
    fun canvasWithDrawPathRendersWithoutCrash() {
        val pathOp =
            Ui.DrawOp.newBuilder()
                .setPath(
                    Ui.DrawPath.newBuilder()
                        .setD("M10,10 L50,50")
                        .setStroke("#FF0000FF")
                        .setStrokeWidth(2.0)
                        .build(),
                )
                .build()
        val root = node("canvas") {
            canvas = Ui.CanvasWidget.newBuilder().addDrawOps(pathOp).build()
        }

        compose.setContent { render(root) }

        compose.onNodeWithTag("terminal-node-canvas").assertExists()
    }

    @Test
    fun canvasWithMalformedDrawPathDoesNotCrash() {
        val pathOp =
            Ui.DrawOp.newBuilder()
                .setPath(
                    Ui.DrawPath.newBuilder()
                        .setD("this is not valid path data")
                        .setStroke("#FF0000FF")
                        .build(),
                )
                .build()
        val root = node("canvas") {
            canvas = Ui.CanvasWidget.newBuilder().addDrawOps(pathOp).build()
        }

        compose.setContent { render(root) }

        compose.onNodeWithTag("terminal-node-canvas").assertExists()
    }

    @Test
    fun audioVisualizerDelegatesToMediaSurface() {
        val root = node("viz") {
            audioVisualizer = Ui.AudioVisualizerWidget.newBuilder().setStreamId("pcm-1").build()
        }

        compose.setContent {
            ServerDrivenRenderer(
                root = root,
                onAction = {},
                mediaSurface = { track -> Text("audio:$track") },
                imageLoader = { url, _ -> Text(url) },
            )
        }

        compose.onNodeWithText("audio:pcm-1").assertIsDisplayed()
    }

    @Composable
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
