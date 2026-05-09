package com.curtcox.terminals.android.ui.widgets

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.LinearProgressIndicator
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp

/**
 * Rich video-surface chrome matching Flutter [terminal_client_shell] `_buildVideoSurface`.
 * Live attach state is not wired yet; surfaces show the same “waiting” copy as an unbound stream.
 */
@Composable
fun TerminalShellVideoSurface(trackId: String) {
    val outerShape = RoundedCornerShape(8.dp)
    Column(
        modifier =
            Modifier
                .padding(vertical = 6.dp)
                .border(1.dp, VideoSurfaceBorder, outerShape)
                .background(Color.Black, outerShape)
                .padding(8.dp),
    ) {
        Box(
            modifier =
                Modifier
                    .fillMaxWidth()
                    .height(160.dp),
        ) {
            Box(modifier = Modifier.fillMaxSize())
            Box(
                modifier =
                    Modifier
                        .align(Alignment.TopStart)
                        .padding(6.dp)
                        .background(OverlayScrim, RoundedCornerShape(4.dp))
                        .padding(horizontal = 6.dp, vertical = 2.dp),
            ) {
                Text(
                    text = "Waiting for media",
                    color = Color.White,
                    fontSize = 11.sp,
                )
            }
            if (trackId.isNotEmpty()) {
                Box(
                    modifier =
                        Modifier
                            .align(Alignment.BottomEnd)
                            .padding(6.dp)
                            .background(OverlayScrim, RoundedCornerShape(4.dp))
                            .padding(horizontal = 6.dp, vertical = 2.dp),
                ) {
                    Text(
                        text = trackId,
                        color = Color.White,
                        fontSize = 11.sp,
                    )
                }
            }
        }
    }
}

/**
 * Audio visualizer chrome matching Flutter [terminal_client_shell] `_buildAudioVisualizer`.
 * Level metering is not wired yet; progress is indeterminate while waiting for a stream.
 */
@Composable
fun TerminalShellAudioVisualizer(streamId: String) {
    val outerShape = RoundedCornerShape(8.dp)
    Column(
        modifier =
            Modifier
                .padding(vertical = 6.dp)
                .border(1.dp, AudioVisualizerBorder, outerShape)
                .padding(8.dp),
    ) {
        Row(verticalAlignment = Alignment.CenterVertically, modifier = Modifier.fillMaxWidth()) {
            Text(text = "Audio level")
            Spacer(modifier = Modifier.weight(1f))
            Box(
                modifier =
                    Modifier
                        .background(AudioChipBackground, RoundedCornerShape(4.dp))
                        .padding(horizontal = 6.dp, vertical = 2.dp),
            ) {
                Text(
                    text = "Waiting for media",
                    fontSize = 11.sp,
                )
            }
        }
        if (streamId.isNotEmpty()) {
            Text(text = streamId, fontSize = 12.sp)
        }
        Spacer(modifier = Modifier.height(8.dp))
        LinearProgressIndicator(modifier = Modifier.fillMaxWidth())
    }
}

private val VideoSurfaceBorder = Color(0xFF455A64)
private val AudioVisualizerBorder = Color(0xFFB0BEC5)
private val OverlayScrim = Color(0x8A000000)
private val AudioChipBackground = Color(0xFFECEFF1)
