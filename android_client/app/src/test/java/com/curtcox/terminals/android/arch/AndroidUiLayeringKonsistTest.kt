package com.curtcox.terminals.android.arch

import com.lemonappdev.konsist.api.Konsist
import com.lemonappdev.konsist.api.ext.list.withPackage
import com.lemonappdev.konsist.api.verify.assertFalse
import org.junit.Test

/**
 * Mirrors [scripts/check-android-client-boundary.sh]: server-driven UI must stay generic and not
 * reach into connection, discovery, media, or platform subsystems.
 */
class AndroidUiLayeringKonsistTest {

    @Test
    fun `server driven ui does not import platform subsystems`() {
        val forbiddenPrefixes = listOf(
            "com.curtcox.terminals.android.connection",
            "com.curtcox.terminals.android.discovery",
            "com.curtcox.terminals.android.media",
            "com.curtcox.terminals.android.platform",
        )
        Konsist.scopeFromProduction()
            .files
            .withPackage("com.curtcox.terminals.android.ui..")
            .assertFalse { file ->
                file.imports.any { koImport ->
                    forbiddenPrefixes.any { prefix -> koImport.name.startsWith(prefix) }
                }
            }
    }
}
