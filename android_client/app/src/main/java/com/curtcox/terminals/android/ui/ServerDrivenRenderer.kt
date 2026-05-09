package com.curtcox.terminals.android.ui

import androidx.compose.foundation.Canvas
import androidx.compose.foundation.background
import androidx.compose.foundation.gestures.detectTapGestures
import androidx.compose.foundation.horizontalScroll
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.weight
import androidx.compose.foundation.layout.ColumnScope
import androidx.compose.foundation.layout.RowScope
import androidx.compose.foundation.layout.wrapContentSize
import androidx.compose.foundation.lazy.grid.GridCells
import androidx.compose.foundation.lazy.grid.LazyVerticalGrid
import androidx.compose.foundation.lazy.grid.items
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
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.focus.FocusRequester
import androidx.compose.ui.focus.focusRequester
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.geometry.Size
import androidx.compose.ui.input.pointer.pointerInput
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.unit.dp
import terminals.ui.v1.Ui

@Composable
fun ServerDrivenRendererPlaceholder() {
    Text("Waiting for server-driven UI")
}

@Composable
fun ServerDrivenRenderer(
    root: Ui.Node,
    onAction: (ServerDrivenAction) -> Unit,
    mediaSurface: @Composable (trackId: String) -> Unit = {},
    imageLoader: @Composable (url: String, contentDescription: String?) -> Unit,
    deviceControlEffects: DeviceControlEffects = DeviceControlEffects.none(),
    policy: RendererPolicy = RendererPolicy.default(),
) {
    RenderNode(root, onAction, mediaSurface, imageLoader, deviceControlEffects, policy)
}

@Composable
private fun RenderNode(
    node: Ui.Node,
    onAction: (ServerDrivenAction) -> Unit,
    mediaSurface: @Composable (trackId: String) -> Unit,
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
                RenderFlexColumnChildren(node, onAction, mediaSurface, imageLoader, deviceControlEffects, policy)
            }
        }
        Ui.Node.WidgetCase.ROW -> Row(props.modifier()) {
            RenderFlexRowChildren(node, onAction, mediaSurface, imageLoader, deviceControlEffects, policy)
        }
        Ui.Node.WidgetCase.GRID -> LazyVerticalGrid(
            columns = GridCells.Fixed(node.grid.columns.coerceAtLeast(1)),
            modifier = props.modifier(),
        ) {
            items(node.childrenList) { child ->
                RenderNode(child, onAction, mediaSurface, imageLoader, deviceControlEffects, policy)
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
                    RenderFlexRowChildren(node, onAction, mediaSurface, imageLoader, deviceControlEffects, policy)
                }
            } else {
                Column(
                    modifier = props.modifier().verticalScroll(scrollState),
                    horizontalAlignment = Alignment.Start,
                ) {
                    RenderFlexColumnChildren(node, onAction, mediaSurface, imageLoader, deviceControlEffects, policy)
                }
            }
        }
        Ui.Node.WidgetCase.PADDING -> Box(props.modifier().padding(node.padding.all.dp)) {
            RenderPlainChildren(node, onAction, mediaSurface, imageLoader, deviceControlEffects, policy)
        }
        Ui.Node.WidgetCase.CENTER -> Box(props.modifier().fillMaxWidth(), contentAlignment = Alignment.Center) {
            RenderPlainChildren(node, onAction, mediaSurface, imageLoader, deviceControlEffects, policy)
        }
        Ui.Node.WidgetCase.EXPAND -> Box(Modifier.fillMaxWidth().then(props.modifier())) {
            RenderPlainChildren(node, onAction, mediaSurface, imageLoader, deviceControlEffects, policy)
        }
        Ui.Node.WidgetCase.TEXT -> Text(
            text = node.text.value,
            modifier = props.modifier(),
            color = parseColorOrUnspecified(node.text.color),
            fontFamily = if (node.text.style == "monospace") FontFamily.Monospace else null,
        )
        Ui.Node.WidgetCase.IMAGE -> imageLoader(node.image.url, node.propsMap["contentDescription"])
        Ui.Node.WidgetCase.VIDEO_SURFACE -> mediaSurface(node.videoSurface.trackId)
        Ui.Node.WidgetCase.AUDIO_VISUALIZER -> mediaSurface(node.audioVisualizer.streamId)
        Ui.Node.WidgetCase.CANVAS -> TerminalCanvas(node, props.modifier().fillMaxSize())
        Ui.Node.WidgetCase.TEXT_INPUT -> TerminalTextInput(node, props, onAction)
        Ui.Node.WidgetCase.BUTTON -> Button(
            modifier = props.modifier(),
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
        Ui.Node.WidgetCase.GESTURE_AREA -> Box(
            props.modifier().pointerInput(node.id) {
                detectTapGestures {
                    onAction(
                        ServerDrivenAction(
                            actionComponentId(props.componentId, "gesture_area"),
                            node.gestureArea.action.ifBlank { "tap" },
                        ),
                    )
                }
            },
        ) { RenderPlainChildren(node, onAction, mediaSurface, imageLoader, deviceControlEffects, policy) }
        Ui.Node.WidgetCase.OVERLAY -> Box(props.modifier()) {
            RenderPlainChildren(node, onAction, mediaSurface, imageLoader, deviceControlEffects, policy)
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
    mediaSurface: @Composable (trackId: String) -> Unit,
    imageLoader: @Composable (url: String, contentDescription: String?) -> Unit,
    deviceControlEffects: DeviceControlEffects,
    policy: RendererPolicy,
) {
    for (child in node.childrenList) {
        RenderNode(child, onAction, mediaSurface, imageLoader, deviceControlEffects, policy)
    }
}

/** Applies [Modifier.weight] to direct [EXPAND] children, matching Flutter [Expanded] inside [Row]. */
@Composable
private fun RowScope.RenderFlexRowChildren(
    node: Ui.Node,
    onAction: (ServerDrivenAction) -> Unit,
    mediaSurface: @Composable (trackId: String) -> Unit,
    imageLoader: @Composable (url: String, contentDescription: String?) -> Unit,
    deviceControlEffects: DeviceControlEffects,
    policy: RendererPolicy,
) {
    for (child in node.childrenList) {
        if (child.widgetCase == Ui.Node.WidgetCase.EXPAND) {
            val expandProps = PrimitiveProps.from(child)
            Box(Modifier.weight(1f).then(expandProps.modifier())) {
                RenderPlainChildren(child, onAction, mediaSurface, imageLoader, deviceControlEffects, policy)
            }
        } else {
            RenderNode(child, onAction, mediaSurface, imageLoader, deviceControlEffects, policy)
        }
    }
}

/** Applies [Modifier.weight] to direct [EXPAND] children, matching Flutter [Expanded] inside [Column]. */
@Composable
private fun ColumnScope.RenderFlexColumnChildren(
    node: Ui.Node,
    onAction: (ServerDrivenAction) -> Unit,
    mediaSurface: @Composable (trackId: String) -> Unit,
    imageLoader: @Composable (url: String, contentDescription: String?) -> Unit,
    deviceControlEffects: DeviceControlEffects,
    policy: RendererPolicy,
) {
    for (child in node.childrenList) {
        if (child.widgetCase == Ui.Node.WidgetCase.EXPAND) {
            val expandProps = PrimitiveProps.from(child)
            Box(Modifier.weight(1f).then(expandProps.modifier())) {
                RenderPlainChildren(child, onAction, mediaSurface, imageLoader, deviceControlEffects, policy)
            }
        } else {
            RenderNode(child, onAction, mediaSurface, imageLoader, deviceControlEffects, policy)
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
                else -> Unit
            }
        }
    }
}

