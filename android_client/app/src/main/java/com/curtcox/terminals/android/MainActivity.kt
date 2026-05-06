package com.curtcox.terminals.android

import android.content.res.Configuration
import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.activity.viewModels
import androidx.lifecycle.ViewModel
import androidx.lifecycle.ViewModelProvider
import com.curtcox.terminals.android.app.AndroidClientDependencies
import com.curtcox.terminals.android.app.AndroidTerminalApp
import com.curtcox.terminals.android.app.AndroidTerminalViewModel

class MainActivity : ComponentActivity() {
    private val viewModel: AndroidTerminalViewModel by viewModels {
        AndroidTerminalViewModelFactory(AndroidClientDependencies.fromContext(this))
    }

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContent {
            AndroidTerminalApp(viewModel = viewModel)
        }
    }

    override fun onResume() {
        super.onResume()
        viewModel.refreshCapabilities("activity-resume")
    }

    override fun onConfigurationChanged(newConfig: Configuration) {
        super.onConfigurationChanged(newConfig)
        viewModel.refreshCapabilities("configuration")
    }

    private class AndroidTerminalViewModelFactory(
        private val dependencies: AndroidClientDependencies,
    ) : ViewModelProvider.Factory {
        @Suppress("UNCHECKED_CAST")
        override fun <T : ViewModel> create(modelClass: Class<T>): T {
            require(modelClass.isAssignableFrom(AndroidTerminalViewModel::class.java)) {
                "Unsupported ViewModel: ${modelClass.name}"
            }
            return AndroidTerminalViewModel(dependencies) as T
        }
    }
}
