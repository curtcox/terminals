package com.curtcox.terminals.android.ui.widgets

import androidx.compose.foundation.border
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.curtcox.terminals.android.ui.PrimitiveProps

/** Matches Flutter [server_driven_renderer] `_placeholderPrimitive` for media nodes. */
@Composable
fun TerminalMediaPlaceholder(
    props: PrimitiveProps,
    title: String,
    detail: String,
) {
    val shape = RoundedCornerShape(8.dp)
    Column(
        modifier =
            props
                .modifier()
                .padding(vertical = 6.dp)
                .border(1.dp, PlaceholderBorder, shape)
                .padding(8.dp),
    ) {
        Text(text = title)
        if (detail.isNotEmpty()) {
            Spacer(modifier = Modifier.height(4.dp))
            Text(text = detail, fontSize = 12.sp)
        }
    }
}

private val PlaceholderBorder = Color(0xFFB0BEC5)
