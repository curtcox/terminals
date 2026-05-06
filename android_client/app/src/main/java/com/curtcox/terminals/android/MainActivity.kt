package com.curtcox.terminals.android

import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.activity.viewModels
import com.curtcox.terminals.android.app.AndroidTerminalApp
import com.curtcox.terminals.android.app.AndroidTerminalViewModel

class MainActivity : ComponentActivity() {
    private val viewModel: AndroidTerminalViewModel by viewModels()

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContent {
            AndroidTerminalApp(viewModel = viewModel)
        }
    }
}
