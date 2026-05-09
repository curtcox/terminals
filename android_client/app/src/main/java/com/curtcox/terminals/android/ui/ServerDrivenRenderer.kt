package com.curtcox.terminals.android.ui

import android.graphics.Paint
import android.graphics.Typeface
import androidx.compose.foundation.Canvas
import androidx.compose.foundation.background
import androidx.compose.foundation.gestures.detectTapGestures
import androidx.compose.foundation.horizontalScroll
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.BoxWithConstraints
import androidx.compose.foundation.layout.FlowRow
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.layout.ColumnScope
import androidx.compose.foundation.layout.RowScope
import androidx.compose.foundation.layout.wrapContentSize
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Button
import androidx.compose.material3.DropdownMenu
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.LinearProgressIndicator
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Slider
import androidx.compose.material3.Switch
import androidx.compose.material3.Text
import androidx.compose.foundation.text.KeyboardActions
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.text.selection.SelectionContainer
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalConfiguration
import androidx.compose.ui.focus.FocusRequester
import androidx.compose.ui.focus.focusRequester
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.geometry.Size
import androidx.compose.ui.input.pointer.pointerInput
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.toArgb
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.unit.dp
import com.curtcox.terminals.android.ui.widgets.TerminalMediaPlaceholder
import androidx.core.graphics.PathParser
import terminals.ui.v1.Ui

private class MediaSurfaces(
    val video: (@Composable (String) -> Unit)?,
    val audioVisualizer: (@Composable (String) -> Unit)?,
) {
    val audio: (@Composable (String) -> Unit)?
        get() = audioVisualizer ?: video
}

@Composable
fun ServerDrivenRendererPlaceholder() {
    Text("Waiting for server-driven UI")
}

@Composable
fun ServerDrivenRenderer(
    root: Ui.Node,
    onAction: (ServerDrivenAction) -> Unit,
    mediaSurface: (@Composable (trackId: String) -> Unit)? = null,
    audioVisualizerSurface: (@Composable (streamId: String) -> Unit)? = null,
    imageLoader: @Composable (url: String, contentDescription: String?) -> Unit,
    deviceControlEffects: DeviceControlEffects = DeviceControlEffects.none(),
    policy: RendererPolicy = RendererPolicy.default(),
) {
    val media = MediaSurfaces(mediaSurface, audioVisualizerSurface)
    RenderNode(root, onAction, media, imageLoader, deviceControlEffects, policy)
}

@Composable
private fun RenderNode(
    node: Ui.Node,
    onAction: (ServerDrivenAction) -> Unit,
    media: MediaSurfaces,
    imageLoader: @Composable (url: String, contentDescription: String?) -> Unit,
    deviceControlEffects: DeviceControlEffects,
    policy: RendererPolicy,
) {
    val props = PrimitiveProps.from(node)
    when (node.widgetCase) {
        Ui.Node.WidgetCase.STACK -> {
            val stackMod = parseHexColor(node.propsMap["background"])
                ?.let { props.modifier().background(it) }
                ?: props.modifier()
            Column(
                modifier = stackMod,
                verticalArrangement = Arrangement.Top,
                horizontalAlignment = Alignment.Start,
            ) {
                RenderFlexColumnChildren(node, onAction, media, imageLoader, deviceControlEffects, policy)
            }
        }
        Ui.Node.WidgetCase.ROW -> Row(props.modifier()) {
            RenderFlexRowChildren(node, onAction, media, imageLoader, deviceControlEffects, policy)
        }
        Ui.Node.WidgetCase.GRID -> {
            // Match Flutter `server_driven_renderer.dart` grid: LayoutBuilder + Wrap with 8dp gaps
            // and fixed cell width so `columns` items fit per row.
            val columns = node.grid.columns.coerceAtLeast(1)
            val spacing = 8.dp
            BoxWithConstraints(modifier = props.modifier()) {
                val maxW =
                    if (maxWidth.value.isFinite() && maxWidth > 0.dp) {
                        maxWidth
                    } else {
                        LocalConfiguration.current.screenWidthDp.dp
                    }
                val totalSpacing = spacing * (columns - 1).coerceAtLeast(0)
                val itemWidth =
                    if (columns <= 1) {
                        maxW
                    } else {
                        ((maxW - totalSpacing) / columns).coerceAtLeast(0.dp)
                    }
                FlowRow(
                    modifier = Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.spacedBy(spacing),
                    verticalArrangement = Arrangement.spacedBy(spacing),
                ) {
                    for (child in node.childrenList) {
                        Box(Modifier.width(itemWidth)) {
                            RenderNode(child, onAction, media, imageLoader, deviceControlEffects, policy)
                        }
                    }
                }
            }
        }
        Ui.Node.WidgetCase.SCROLL -> {
            val scrollState = rememberScrollState()
            val isHorizontal = when (node.scroll.directionEnum) {
                Ui.ScrollDirection.SCROLL_DIRECTION_HORIZONTAL -> true
                Ui.ScrollDirection.SCROLL_DIRECTION_VERTICAL -> false
                else -> node.scroll.direction.trim().lowercase() == "horizontal"
            }
            if (isHorizontal) {
                Row(
                    modifier = props.modifier().horizontalScroll(scrollState),
                    verticalAlignment = Alignment.CenterVertically,
                ) {
                    RenderFlexRowChildren(node, onAction, media, imageLoader, deviceControlEffects, policy)
                }
            } else {
                Column(
                    modifier = props.modifier().verticalScroll(scrollState),
                    horizontalAlignment = Alignment.Start,
                ) {
                    RenderFlexColumnChildren(node, onAction, media, imageLoader, deviceControlEffects, policy)
                }
            }
        }
        Ui.Node.WidgetCase.PADDING -> Box(props.modifier().padding(node.padding.all.dp)) {
            RenderPlainChildren(node, onAction, media, imageLoader, deviceControlEffects, policy)
        }
        Ui.Node.WidgetCase.CENTER -> Box(props.modifier().fillMaxWidth(), contentAlignment = Alignment.Center) {
            RenderPlainChildren(node, onAction, media, imageLoader, deviceControlEffects, policy)
        }
        Ui.Node.WidgetCase.EXPAND -> Box(Modifier.fillMaxWidth().then(props.modifier())) {
            RenderPlainChildren(node, onAction, media, imageLoader, deviceControlEffects, policy)
        }
        Ui.Node.WidgetCase.TEXT -> SelectionContainer {
            Text(
                text = node.text.value,
                // Match Flutter `server_driven_renderer.dart`: symmetric vertical padding around text.
                modifier = props.modifier().padding(vertical = 4.dp),
                color = parseColorOrUnspecified(node.text.color),
                fontFamily = if (node.text.style == "monospace") FontFamily.Monospace else null,
            )
        }
        Ui.Node.WidgetCase.IMAGE -> imageLoader(node.image.url, node.propsMap["contentDescription"])
        Ui.Node.WidgetCase.VIDEO_SURFACE -> {
            val trackId = node.videoSurface.trackId.trim()
            val builder = media.video
            if (builder != null) {
                Box(props.modifier()) { builder(trackId) }
            } else {
                TerminalMediaPlaceholder(props, "Video surface", trackId)
            }
        }
        Ui.Node.WidgetCase.AUDIO_VISUALIZER -> {
            val streamId = node.audioVisualizer.streamId.trim()
            val builder = media.audio
            if (builder != null) {
                Box(props.modifier()) { builder(streamId) }
            } else {
                TerminalMediaPlaceholder(props, "Audio level", streamId)
            }
        }
        Ui.Node.WidgetCase.CANVAS -> TerminalCanvas(node, props.modifier().fillMaxSize())
        Ui.Node.WidgetCase.TEXT_INPUT -> TerminalTextInput(node, props, onAction)
        Ui.Node.WidgetCase.BUTTON -> Button(
            // Match Flutter: Padding(vertical: 4) around the button.
            modifier = props.modifier().padding(vertical = 4.dp),
            onClick = {
                onAction(
                    ServerDrivenAction(
                        actionComponentId(props.componentId, "button"),
                        node.button.action.ifBlank { "tap" },
                    ),
                )
            },
        ) { Text(node.button.label) }
        Ui.Node.WidgetCase.SLIDER -> {
            val min = node.slider.min.toFloat()
            val max = node.slider.max.toFloat().takeIf { it > min } ?: min + 1f
            Slider(
                value = node.slider.value.toFloat().coerceIn(min, max),
                onValueChange = {
                    onAction(
                        ServerDrivenAction(
                            actionComponentId(props.componentId, "slider"),
                            "change",
                            it.toString(),
                        ),
                    )
                },
                valueRange = min..max,
                modifier = props.modifier(),
            )
        }
        Ui.Node.WidgetCase.TOGGLE -> Switch(
            checked = node.toggle.value,
            onCheckedChange = {
                onAction(
                    ServerDrivenAction(
                        actionComponentId(props.componentId, "toggle"),
                        "toggle",
                        it.toString(),
                    ),
                )
            },
            modifier = props.modifier(),
        )
        Ui.Node.WidgetCase.DROPDOWN -> TerminalDropdown(node, props, onAction)
        Ui.Node.WidgetCase.GESTURE_AREA -> {
            val tapModifier = Modifier.pointerInput(node.id) {
                detectTapGestures {
                    onAction(
                        ServerDrivenAction(
                            actionComponentId(props.componentId, "gesture_area"),
                            node.gestureArea.action.ifBlank { "tap" },
                        ),
                    )
                }
            }
            // Match Flutter: empty gesture areas use a 48×48 minimum hit target.
            if (node.childrenList.isEmpty()) {
                Box(props.modifier().then(tapModifier).size(48.dp)) {}
            } else {
                Box(props.modifier().then(tapModifier)) {
                    RenderPlainChildren(node, onAction, media, imageLoader, deviceControlEffects, policy)
                }
            }
        }
        Ui.Node.WidgetCase.OVERLAY -> Box(props.modifier()) {
            RenderPlainChildren(node, onAction, media, imageLoader, deviceControlEffects, policy)
        }
        Ui.Node.WidgetCase.PROGRESS -> LinearProgressIndicator(
            progress = { node.progress.value.toFloat().coerceIn(0f, 1f) },
            modifier = props.modifier(),
        )
        Ui.Node.WidgetCase.FULLSCREEN -> DeviceControlNode(props, "fullscreen=${node.fullscreen.enabled}") {
            deviceControlEffects.setFullscreen(node.fullscreen.enabled)
        }
        Ui.Node.WidgetCase.KEEP_AWAKE -> DeviceControlNode(props, "keep_awake=${node.keepAwake.enabled}") {
            deviceControlEffects.setKeepAwake(node.keepAwake.enabled)
        }
        Ui.Node.WidgetCase.BRIGHTNESS -> DeviceControlNode(props, "brightness=${node.brightness.value}") {
            deviceControlEffects.setBrightness(node.brightness.value)
        }
        Ui.Node.WidgetCase.WIDGET_NOT_SET -> if (policy.showUnsupportedFallback) Text(policy.unsupportedText, modifier = props.modifier())
        null -> if (policy.showUnsupportedFallback) Text(policy.unsupportedText, modifier = props.modifier())
    }
}

@Composable
private fun RenderPlainChildren(
    node: Ui.Node,
    onAction: (ServerDrivenAction) -> Unit,
    media: MediaSurfaces,
    imageLoader: @Composable (url: String, contentDescription: String?) -> Unit,
    deviceControlEffects: DeviceControlEffects,
    policy: RendererPolicy,
) {
    for (child in node.childrenList) {
        RenderNode(child, onAction, media, imageLoader, deviceControlEffects, policy)
    }
}

/** Applies [Modifier.weight] to direct [EXPAND] children, matching Flutter [Expanded] inside [Row]. */
@Composable
private fun RowScope.RenderFlexRowChildren(
    node: Ui.Node,
    onAction: (ServerDrivenAction) -> Unit,
    media: MediaSurfaces,
    imageLoader: @Composable (url: String, contentDescription: String?) -> Unit,
    deviceControlEffects: DeviceControlEffects,
    policy: RendererPolicy,
) {
    for (child in node.childrenList) {
        if (child.widgetCase == Ui.Node.WidgetCase.EXPAND) {
            val expandProps = PrimitiveProps.from(child)
            Box(Modifier.weight(1f).then(expandProps.modifier())) {
                RenderPlainChildren(child, onAction, media, imageLoader, deviceControlEffects, policy)
            }
        } else {
            RenderNode(child, onAction, media, imageLoader, deviceControlEffects, policy)
        }
    }
}

/** Applies [Modifier.weight] to direct [EXPAND] children, matching Flutter [Expanded] inside [Column]. */
@Composable
private fun ColumnScope.RenderFlexColumnChildren(
    node: Ui.Node,
    onAction: (ServerDrivenAction) -> Unit,
    media: MediaSurfaces,
    imageLoader: @Composable (url: String, contentDescription: String?) -> Unit,
    deviceControlEffects: DeviceControlEffects,
    policy: RendererPolicy,
) {
    for (child in node.childrenList) {
        if (child.widgetCase == Ui.Node.WidgetCase.EXPAND) {
            val expandProps = PrimitiveProps.from(child)
            Box(Modifier.weight(1f).then(expandProps.modifier())) {
                RenderPlainChildren(child, onAction, media, imageLoader, deviceControlEffects, policy)
            }
        } else {
            RenderNode(child, onAction, media, imageLoader, deviceControlEffects, policy)
        }
    }
}

@Composable
private fun TerminalTextInput(node: Ui.Node, props: PrimitiveProps, onAction: (ServerDrivenAction) -> Unit) {
    var value by remember(node.id) { mutableStateOf(node.propsMap["value"].orEmpty()) }
    val focusRequester = remember(node.id) { FocusRequester() }
    if (node.textInput.autofocus) {
        LaunchedEffect(node.id) {
            focusRequester.requestFocus()
        }
    }
    OutlinedTextField(
        value = value,
        onValueChange = { value = it },
        placeholder = { Text(node.textInput.placeholder) },
        modifier = props.modifier().focusRequester(focusRequester),
        singleLine = true,
        keyboardOptions = KeyboardOptions(imeAction = ImeAction.Done),
        keyboardActions = KeyboardActions(
            onDone = {
                onAction(ServerDrivenAction(actionComponentId(props.componentId, "text_input"), "submit", value))
                value = ""
            },
        ),
    )
}

@Composable
private fun TerminalDropdown(node: Ui.Node, props: PrimitiveProps, onAction: (ServerDrivenAction) -> Unit) {
    var expanded by remember(node.id) { mutableStateOf(false) }
    val options = node.dropdown.optionsList
    val selected = when {
        options.isEmpty() -> null
        options.contains(node.dropdown.value) -> node.dropdown.value
        else -> options.first()
    }
    val label = selected?.takeIf { it.isNotEmpty() } ?: "Select option"
    Box(props.modifier().wrapContentSize(Alignment.TopStart)) {
        OutlinedButton(onClick = { expanded = true }) { Text(label) }
        DropdownMenu(expanded = expanded, onDismissRequest = { expanded = false }) {
            node.dropdown.optionsList.forEach { option ->
                DropdownMenuItem(
                    text = { Text(option) },
                    onClick = {
                        expanded = false
                        onAction(ServerDrivenAction(actionComponentId(props.componentId, "dropdown"), "select", option))
                    },
                )
            }
        }
    }
}

private fun actionComponentId(componentId: String, fallback: String): String = componentId.ifBlank { fallback }

@Composable
private fun DeviceControlNode(props: PrimitiveProps, label: String, apply: () -> Unit) {
    LaunchedEffect(label) {
        apply()
    }
    Text(label, modifier = props.modifier())
}

@Composable
private fun TerminalCanvas(node: Ui.Node, modifier: Modifier) {
    Canvas(modifier = modifier) {
        node.canvas.drawOpsList.forEach { op ->
            when (op.opCase) {
                Ui.DrawOp.OpCase.LINE -> drawLine(
                    color = parseColorOrUnspecified(op.line.stroke),
                    start = Offset(op.line.x1.toFloat(), op.line.y1.toFloat()),
                    end = Offset(op.line.x2.toFloat(), op.line.y2.toFloat()),
                    strokeWidth = op.line.strokeWidth.toFloat().coerceAtLeast(1f),
                )
                Ui.DrawOp.OpCase.RECT -> drawRect(
                    color = parseColorOrUnspecified(op.rect.fill),
                    topLeft = Offset(op.rect.x.toFloat(), op.rect.y.toFloat()),
                    size = Size(op.rect.width.toFloat(), op.rect.height.toFloat()),
                )
                Ui.DrawOp.OpCase.CIRCLE -> drawCircle(
                    color = parseColorOrUnspecified(op.circle.fill),
                    radius = op.circle.radius.toFloat(),
                    center = Offset(op.circle.cx.toFloat(), op.circle.cy.toFloat()),
                )
                Ui.DrawOp.OpCase.TEXT -> {
                    val t = op.text
                    val fill = parseColorOrUnspecified(t.fill)
                    val paint =
                        Paint().apply {
                            isAntiAlias = true
                            color =
                                (if (fill == Color.Unspecified) Color.Black else fill).toArgb()
                            textSize =
                                t.fontSize.toFloat().takeIf { it > 0f } ?: 42f
                            typeface =
                                when (t.fontFamily.lowercase()) {
                                    "monospace" -> Typeface.MONOSPACE
                                    else -> Typeface.DEFAULT
                                }
                        }
                    drawContext.canvas.nativeCanvas.drawText(
                        t.text,
                        t.x.toFloat(),
                        t.y.toFloat(),
                        paint,
                    )
                }
                Ui.DrawOp.OpCase.PATH -> {
                    val p = op.path
                    if (p.d.isBlank()) return@forEach
                    val path =
                        try {
                            PathParser.createPathFromPathData(p.d)
                        } catch (_: Throwable) {
                            null
                        } ?: return@forEach
                    val nc = drawContext.canvas.nativeCanvas
                    val fill = parseColorOrUnspecified(p.fill)
                    if (fill != Color.Unspecified) {
                        nc.drawPath(
                            path,
                            Paint().apply {
                                isAntiAlias = true
                                style = Paint.Style.FILL
                                color = fill.toArgb()
                            },
                        )
                    }
                    val stroke = parseColorOrUnspecified(p.stroke)
                    if (stroke != Color.Unspecified) {
                        nc.drawPath(
                            path,
                            Paint().apply {
                                isAntiAlias = true
                                style = Paint.Style.STROKE
                                strokeWidth = p.strokeWidth.toFloat().coerceAtLeast(1f)
                                color = stroke.toArgb()
                            },
                        )
                    }
                }
                else -> Unit
            }
        }
    }
}

