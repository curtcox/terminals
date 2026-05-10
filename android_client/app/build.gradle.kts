import java.io.File
import java.time.Instant
import java.util.Properties
import com.google.protobuf.gradle.id
import com.google.protobuf.gradle.proto
import org.gradle.api.tasks.testing.logging.TestLogEvent
import org.jetbrains.kotlin.gradle.dsl.JvmTarget
import org.jetbrains.kotlin.gradle.tasks.KotlinCompile

// Maven Central `protoc-gen-grpc-java` artifacts for macOS are x86_64 binaries (often labeled
// osx-aarch_64). On Apple Silicon without Rosetta they fail at codegen time; prefer a host
// `protoc-gen-grpc-java` when available (Homebrew formula `protoc-gen-grpc-java`, or any path
// via local.properties / env).
val localProperties = Properties().apply {
    val f = rootProject.file("local.properties")
    if (f.exists()) {
        f.inputStream().use { load(it) }
    }
}

fun resolveGrpcJavaProtocPluginPath(): String? {
    localProperties.getProperty("grpc.java.plugin")?.trim()?.takeIf { it.isNotEmpty() }?.let { return it }
    System.getenv("GRPC_JAVA_PLUGIN")?.trim()?.takeIf { it.isNotEmpty() }?.let { return it }
    val osName = System.getProperty("os.name", "")
    val osArch = System.getProperty("os.arch", "")
    if (osName.contains("Mac", ignoreCase = true) && osArch == "aarch64") {
        listOf(
            File("/opt/homebrew/bin/protoc-gen-grpc-java"),
            File("/usr/local/bin/protoc-gen-grpc-java"),
        ).firstOrNull { it.isFile }?.let { return it.absolutePath }
    }
    return null
}

val grpcJavaProtocPluginPath = resolveGrpcJavaProtocPluginPath()

plugins {
    id("com.android.application")
    id("org.jetbrains.kotlin.android")
    id("org.jetbrains.kotlin.plugin.compose")
    id("com.google.protobuf")
    id("io.gitlab.arturbosch.detekt")
    id("org.owasp.dependencycheck")
}

android {
    namespace = "com.curtcox.terminals.android"
    compileSdk = 36

    defaultConfig {
        applicationId = "com.curtcox.terminals.android"
        minSdk = 25
        targetSdk = 36
        versionCode = 1
        versionName = "0.1.0"

        val buildSha = providers.gradleProperty("TERMINALS_BUILD_SHA").orElse("unknown")
        val buildDate = providers.gradleProperty("TERMINALS_BUILD_DATE").orElse(Instant.now().toString())
        buildConfigField("String", "TERMINALS_BUILD_SHA", "\"${buildSha.get()}\"")
        buildConfigField("String", "TERMINALS_BUILD_DATE", "\"${buildDate.get()}\"")

        testInstrumentationRunner = "androidx.test.runner.AndroidJUnitRunner"
    }

    buildFeatures {
        compose = true
        buildConfig = true
    }

    compileOptions {
        sourceCompatibility = JavaVersion.VERSION_17
        targetCompatibility = JavaVersion.VERSION_17
    }

    packaging {
        resources.excludes += "/META-INF/{AL2.0,LGPL2.1}"
    }

    sourceSets {
        getByName("main") {
            proto {
                srcDir("../../api")
            }
        }
    }

    testOptions {
        unitTests {
            all {
                it.testLogging {
                    events(
                        TestLogEvent.PASSED,
                        TestLogEvent.SKIPPED,
                        TestLogEvent.FAILED,
                    )
                }
            }
        }
    }

    lint {
        baseline = file("lint-baseline.xml")
        abortOnError = true
        checkDependencies = true
        warningsAsErrors = false
        enable += setOf(
            "GradleDependency",
            "RtlHardcoded",
        )
        // Promote high-signal issues to errors; matches already listed in lint-baseline.xml stay suppressed.
        error += setOf(
            "NewApi",
            "InlinedApi",
            "ObsoleteSdkInt",
            "UnusedResources",
            "UnusedIds",
            "VectorPath",
            "Autofill",
            "UseKtx",
        )
    }
}

protobuf {
    protoc {
        artifact = "com.google.protobuf:protoc:4.29.3"
    }
    plugins {
        create("grpc") {
            val hostPlugin = grpcJavaProtocPluginPath
            if (hostPlugin != null) {
                path = hostPlugin
            } else {
                // Keep in sync with `io.grpc:grpc-bom` below so generated stubs match runtime.
                artifact = "io.grpc:protoc-gen-grpc-java:1.81.0"
            }
        }
    }
    generateProtoTasks {
        all().configureEach {
            builtins {
                id("java") {
                    option("lite")
                }
            }
            plugins {
                create("grpc") {
                    option("lite")
                }
            }
        }
    }
}

dependencies {
    val composeBom = platform("androidx.compose:compose-bom:2025.10.00")
    implementation(composeBom)
    androidTestImplementation(composeBom)

    implementation("androidx.activity:activity-compose:1.11.0")
    implementation("androidx.compose.material3:material3")
    implementation("androidx.compose.ui:ui")
    implementation("androidx.compose.ui:ui-tooling-preview")
    implementation("androidx.core:core-ktx:1.17.0")
    implementation("androidx.lifecycle:lifecycle-runtime-compose:2.9.4")
    implementation("androidx.lifecycle:lifecycle-viewmodel-compose:2.9.4")
    implementation("com.google.protobuf:protobuf-javalite:4.29.3")
    implementation(platform("io.grpc:grpc-bom:1.81.0"))
    implementation("io.grpc:grpc-okhttp")
    implementation("io.grpc:grpc-protobuf-lite")
    implementation("io.grpc:grpc-stub")
    implementation("org.jetbrains.kotlinx:kotlinx-coroutines-android:1.10.2")
    compileOnly("javax.annotation:javax.annotation-api:1.3.2")

    debugImplementation("androidx.compose.ui:ui-tooling")
    debugImplementation("androidx.compose.ui:ui-test-manifest")

    testImplementation("junit:junit:4.13.2")
    testImplementation("com.lemonappdev:konsist:0.17.3")
    testImplementation("org.jetbrains.kotlinx:kotlinx-coroutines-test:1.10.2")
    androidTestImplementation("androidx.test.ext:junit:1.3.0")
    androidTestImplementation("androidx.test.espresso:espresso-core:3.7.0")
    androidTestImplementation("androidx.compose.ui:ui-test-junit4")

    detektPlugins("io.gitlab.arturbosch.detekt:detekt-formatting:1.23.8")
    detektPlugins("ru.kode:detekt-rules-compose:1.4.0")
}

kotlin {
    compilerOptions {
        jvmTarget.set(JvmTarget.JVM_17)
    }
}

detekt {
    buildUponDefaultConfig = true
    allRules = false
    parallel = true
    config.setFrom("$rootDir/config/detekt.yml")
    baseline = file("detekt-baseline.xml")
}

dependencyCheck {
    outputDirectory.set(layout.buildDirectory.dir("reports/dependency-check"))
    formats.set(listOf("HTML", "JSON"))
    // Fail the build on high+ CVEs (7–10). Tune suppressions in config/dependency-check-suppressions.xml.
    failBuildOnCVSS.set(7f)
    // Allow NVD API update failures (e.g. rate-limiting without an API key) without failing the build.
    // CVE findings at CVSS 7+ still fail via failBuildOnCVSS above.
    failOnError.set(false)
    suppressionFile.set(rootProject.file("config/dependency-check-suppressions.xml").absolutePath)
    scanConfigurations.set(
        listOf(
            "releaseRuntimeClasspath",
            "debugRuntimeClasspath",
        ),
    )

    nvd {
        apiKey.set(providers.environmentVariable("NVD_API_KEY").orElse(""))
        // Reuse cached NVD data between runs (CI caches ~/.gradle/dependency-check-data).
        validForHours.set(48)
    }

    analyzers {
        assemblyEnabled.set(false)
        nuspecEnabled.set(false)
    }
}

tasks.named("detekt") {
    mustRunAfter(tasks.named("detektBaseline"))
}

afterEvaluate {
    tasks.named("check").configure {
        dependsOn("detektMain")
    }
    tasks.named("detektMain").configure {
        mustRunAfter(tasks.named("detektBaselineMain"))
    }
}

tasks.withType<KotlinCompile>().configureEach {
    compilerOptions {
        allWarningsAsErrors.set(true)
    }
}
