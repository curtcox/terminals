plugins {
    id("com.android.application") version "8.11.1" apply false
    id("org.jetbrains.kotlin.android") version "2.2.20" apply false
    id("org.jetbrains.kotlin.plugin.compose") version "2.2.20" apply false
    id("com.google.protobuf") version "0.9.5" apply false
    id("io.gitlab.arturbosch.detekt") version "1.23.8" apply false
    id("org.owasp.dependencycheck") version "12.2.2" apply false
    id("com.github.ben-manes.versions") version "0.51.0"
}

tasks.register<Delete>("clean") {
    delete(rootProject.layout.buildDirectory)
}
