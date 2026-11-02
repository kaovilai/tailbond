plugins {
    id("com.android.application")
    id("org.jetbrains.kotlin.plugin.compose")
    id("org.jetbrains.kotlin.plugin.serialization")
}

/** Gradle property (-PNAME=…) first, then the environment. */
fun setting(name: String): String? =
    (findProperty(name) as String?)?.takeIf { it.isNotBlank() } ?: System.getenv(name)?.takeIf { it.isNotBlank() }

/**
 * versionName: TB_VERSION (a release tag like "v0.2.0" or "0.2.0-beta.1"),
 * else tailbond.version in gradle.properties.
 */
val tbVersion: String = (setting("TB_VERSION") ?: findProperty("tailbond.version") as String).trim().removePrefix("v")

/**
 * versionCode, agreed with scripts/release/make-manifest.sh (same as
 * update/Version.kt's VersionCode, kept in step by VersionCodeTest):
 * release MAJOR*1000000 + MINOR*10000 + PATCH*100; a pre-release ending in
 * a number N is that base - 100 + N (N 1..99, none = 1).
 */
fun versionCodeOf(name: String): Int {
    val m = Regex("""^(\d+)\.(\d+)\.(\d+)(?:-([0-9A-Za-z.-]+))?(?:\+.*)?$""").matchEntire(name)
        ?: throw GradleException("TB_VERSION \"$name\" isn't MAJOR.MINOR.PATCH[-pre]")
    val (ma, mi, pa, pre) = m.destructured
    if (mi.toInt() > 99 || pa.toInt() > 99 || ma.toInt() > 2100) throw GradleException("TB_VERSION \"$name\": MINOR and PATCH must be ≤ 99")
    val base = ma.toInt() * 1_000_000 + mi.toInt() * 10_000 + pa.toInt() * 100
    if (pre.isEmpty()) return base
    val n = Regex("""(\d+)$""").find(pre)?.groupValues?.get(1)?.takeLast(4)?.toInt() ?: 1
    return maxOf(1, base - 100 + n.coerceIn(1, 99))
}

/** TB_UPDATE_PUBKEY: comma-separated base64 raw ed25519 keys (scripts/release/gen-update-key.sh); empty = updates off. */
val updateKeys: String = (setting("TB_UPDATE_PUBKEY") ?: "").trim().also {
    if (!Regex("^[A-Za-z0-9+/=, ]*$").matches(it)) throw GradleException("TB_UPDATE_PUBKEY must be base64 keys separated by commas")
}

/** TB_UPDATE_SOURCE: public base URL of the channel files (https://github.com/OWNER/REPO/releases/download/channels); empty = mirror only. */
val updateSource: String = (setting("TB_UPDATE_SOURCE") ?: "").trim().also {
    if (it.isNotEmpty() && !Regex("^https://[^\"\\\\ ]+$").matches(it)) throw GradleException("TB_UPDATE_SOURCE must be an https:// URL")
}

/** Release signing from the environment only (android/scripts/build-release-apk.sh); never from files in the repo. */
val releaseKeystore: String? = setting("TB_RELEASE_KEYSTORE")

android {
    namespace = "dev.tailbond.app"
    compileSdk = 37

    dependenciesInfo {
        includeInApk = false
        includeInBundle = false
    }

    defaultConfig {
        applicationId = "dev.tailbond.app"
        minSdk = 26
        targetSdk = 36
        versionCode = versionCodeOf(tbVersion)
        versionName = tbVersion
        buildConfigField("String", "UPDATE_KEYS", "\"$updateKeys\"")
        buildConfigField("String", "UPDATE_SOURCE", "\"$updateSource\"")
        // must match sdk-native's abiFilters (only arm64 prebuilts exist)
        ndk {
            abiFilters += listOf("arm64-v8a")
        }
    }

    signingConfigs {
        if (releaseKeystore != null) {
            create("release") {
                storeFile = file(releaseKeystore)
                storePassword = setting("TB_RELEASE_KEYSTORE_PASSWORD")
                keyAlias = setting("TB_RELEASE_KEY_ALIAS")
                keyPassword = setting("TB_RELEASE_KEY_PASSWORD") ?: setting("TB_RELEASE_KEYSTORE_PASSWORD")
            }
        }
    }

    buildTypes {
        debug {
            // debug key from ~/.android/debug.keystore (never committed).
            // Debug and release builds have different keys: neither can update the other.
        }
        release {
            isMinifyEnabled = false
            // unsigned without TB_RELEASE_KEYSTORE; build-release-apk.sh refuses to run without it
            signingConfig = signingConfigs.findByName("release")
        }
    }

    buildFeatures {
        compose = true
        buildConfig = true
    }
    compileOptions {
        sourceCompatibility = JavaVersion.VERSION_11
        targetCompatibility = JavaVersion.VERSION_11
    }
    testOptions {
        unitTests.isReturnDefaultValues = true
        // Robolectric + Roborazzi screen renders (src/test/.../ux); see docs/ux/summary.md
        unitTests.isIncludeAndroidResources = true
        unitTests.all { t ->
            t.maxHeapSize = "1536m"
            t.systemProperty("robolectric.graphicsMode", "NATIVE")
            t.systemProperty("robolectric.pixelCopyRenderMode", "hardware")
            // -Ptailbond.uxOut=<dir> renders every screen to PNGs there; unset = the render tests skip
            project.findProperty("tailbond.uxOut")?.let {
                t.systemProperty("tailbond.uxOut", it.toString())
                t.systemProperty("roborazzi.test.record", "true")
                t.outputs.upToDateWhen { false }
            }
        }
    }
    lint {
        abortOnError = true
        checkDependencies = false
        warningsAsErrors = false
        // version-bump nags are noise for a pinned, reproducible build
        disable += setOf("GradleDependency", "NewerVersionAvailable", "AndroidGradlePluginVersion", "OldTargetApi")
    }
}

dependencies {
    implementation(project(":sdk-core"))
    implementation(project(":sdk-network"))

    implementation("androidx.core:core-ktx:1.19.1")
    implementation("org.jetbrains.kotlinx:kotlinx-coroutines-android:1.11.0")
    implementation("org.jetbrains.kotlinx:kotlinx-serialization-json:1.12.0")

    // Compose
    implementation(platform("androidx.compose:compose-bom:2026.09.00"))
    implementation("androidx.compose.ui:ui")
    implementation("androidx.compose.material3:material3")
    implementation("androidx.compose.material:material-icons-extended")
    implementation("androidx.activity:activity-compose:1.13.0")
    implementation("androidx.lifecycle:lifecycle-viewmodel-compose:2.11.0")
    implementation("androidx.lifecycle:lifecycle-runtime-compose:2.11.0")

    // QR scanning: CameraX preview + ZXing decode (no Google Play services)
    implementation("androidx.camera:camera-camera2:1.6.2")
    implementation("androidx.camera:camera-lifecycle:1.6.2")
    implementation("androidx.camera:camera-view:1.6.2")
    implementation("com.google.zxing:core:3.5.4")

    testImplementation("junit:junit:4.13.2")
    // screen renders: Robolectric (native graphics) + Roborazzi
    testImplementation("org.robolectric:robolectric:4.17")
    testImplementation("io.github.takahirom.roborazzi:roborazzi:1.77.0")
    testImplementation("io.github.takahirom.roborazzi:roborazzi-compose:1.77.0")
    testImplementation(platform("androidx.compose:compose-bom:2026.09.00"))
    testImplementation("androidx.compose.ui:ui-test-junit4")
    testImplementation("androidx.test.ext:junit:1.3.0")
    debugImplementation("androidx.compose.ui:ui-test-manifest")
}
