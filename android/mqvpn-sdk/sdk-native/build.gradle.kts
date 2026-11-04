// Fork of third_party/mqvpn/android/sdk-native/build.gradle.kts (v0.19.0).
// tailbond: sources and prebuilt libs come from the patched copy of the
// submodule that android/scripts/build-native.sh makes (out/mqvpn-android/src:
// patches/mqvpn/*, e.g. 0004's relay paths) via -DMQVPN_ROOT; instrumented
// tests and the debug-only JNI test seams are left out.
plugins {
    id("com.android.library")
}

val mqvpnRoot = rootProject.file("../out/mqvpn-android/src").canonicalFile.takeIf {
    it.resolve("src/platform/android/reactor.c").isFile
} ?: rootProject.file("../third_party/mqvpn").canonicalFile.also {
    // unpatched: Kotlin-only tasks work, the JNI build doesn't (no relay paths)
    logger.warn("tailbond: out/mqvpn-android/src missing, using the unpatched submodule; run android/scripts/build-native.sh")
}

// tailbond: same ABI list as :app (-Ptailbond.abis / TB_ANDROID_ABIS, default arm64-v8a)
val abis: List<String> = ((findProperty("tailbond.abis") as String?)?.takeIf { it.isNotBlank() }
    ?: System.getenv("TB_ANDROID_ABIS")?.takeIf { it.isNotBlank() } ?: "arm64-v8a")
    .split(',', ' ').map { it.trim() }.filter { it.isNotEmpty() }

android {
    namespace = "com.mqvpn.sdk.native_"
    compileSdk = 37
    ndkVersion = "28.2.13676358"

    defaultConfig {
        minSdk = 26
        consumerProguardFiles("consumer-rules.pro")

        externalNativeBuild {
            cmake {
                cFlags("-ffile-prefix-map=$mqvpnRoot=/mqvpn")
                arguments(
                    "-DCMAKE_SHARED_LINKER_FLAGS=-Wl,--build-id=none",
                    "-DMQVPN_ROOT=$mqvpnRoot",
                )
            }
        }

        ndk {
            // Only ABIs with prebuilt .a files (scripts/build-native.sh output).
            abiFilters += abis
        }
    }

    externalNativeBuild {
        cmake {
            path = file("src/main/jni/CMakeLists.txt")
            version = "3.22.1"
        }
    }

    compileOptions {
        sourceCompatibility = JavaVersion.VERSION_11
        targetCompatibility = JavaVersion.VERSION_11
    }
}

dependencies {
    implementation("androidx.annotation:annotation:1.11.0")
}
