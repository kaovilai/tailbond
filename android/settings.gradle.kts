pluginManagement {
    repositories {
        google()
        mavenCentral()
        gradlePluginPortal()
    }
}

dependencyResolutionManagement {
    repositoriesMode.set(RepositoriesMode.FAIL_ON_PROJECT_REPOS)
    repositories {
        google()
        mavenCentral()
    }
}

rootProject.name = "tailbond-android"

// mqvpn SDK (third_party/mqvpn, v0.19.0). sdk-runtime and sdk-network are used
// straight from the submodule. sdk-native and sdk-core are forked under
// mqvpn-sdk/ because tailbond needs JNI and config additions (reinjection,
// certificate pin, per-path gating); see mqvpn-sdk/README.md.
val mqvpnAndroid = file("../third_party/mqvpn/android")
include(":sdk-runtime", ":sdk-network", ":sdk-native", ":sdk-core", ":app")
project(":sdk-runtime").projectDir = mqvpnAndroid.resolve("sdk-runtime")
project(":sdk-network").projectDir = mqvpnAndroid.resolve("sdk-network")
project(":sdk-native").projectDir = file("mqvpn-sdk/sdk-native")
project(":sdk-core").projectDir = file("mqvpn-sdk/sdk-core")
