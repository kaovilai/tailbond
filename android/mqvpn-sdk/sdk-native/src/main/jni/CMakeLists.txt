cmake_minimum_required(VERSION 3.22)
project(mqvpn_jni C CXX)

# tailbond: this module is a fork that lives outside the mqvpn tree; Gradle
# passes the submodule root as MQVPN_ROOT. The default keeps the upstream
# in-tree layout working.
set(MQVPN_ROOT ${CMAKE_SOURCE_DIR}/../../../../../ CACHE PATH "mqvpn source root")

# Prebuilt libmqvpn.a, xquic, BoringSSL per ABI (scripts/build_android.sh output)
set(PREBUILT_DIR ${MQVPN_ROOT}/android/sdk-native/prebuilt/${ANDROID_ABI})

# libmqvpn (static)
add_library(mqvpn STATIC IMPORTED)
set_target_properties(mqvpn PROPERTIES
    IMPORTED_LOCATION ${PREBUILT_DIR}/libmqvpn.a)

# lwIP core (hybrid TCP lane; static, split out of libmqvpn.a)
add_library(lwip_core STATIC IMPORTED)
set_target_properties(lwip_core PROPERTIES
    IMPORTED_LOCATION ${PREBUILT_DIR}/liblwip_core.a)

# xquic (static)
add_library(xquic-static STATIC IMPORTED)
set_target_properties(xquic-static PROPERTIES
    IMPORTED_LOCATION ${PREBUILT_DIR}/libxquic.a)

# BoringSSL
add_library(ssl STATIC IMPORTED)
set_target_properties(ssl PROPERTIES
    IMPORTED_LOCATION ${PREBUILT_DIR}/libssl.a)
add_library(crypto STATIC IMPORTED)
set_target_properties(crypto PROPERTIES
    IMPORTED_LOCATION ${PREBUILT_DIR}/libcrypto.a)

# Include paths
# tailbond: BoringSSL headers for SHA-256 (certificate pin in mqvpn_jni.c)
include_directories(
    ${MQVPN_ROOT}/include
    ${MQVPN_ROOT}/src
    ${MQVPN_ROOT}/third_party/xquic/include
    ${MQVPN_ROOT}/third_party/xquic/third_party/boringssl/include
)

# JNI shared library
# LINKER_LANGUAGE CXX: BoringSSL (libssl.a) is C++ and needs clang++ for linking
# to resolve C++ ABI symbols (__cxa_begin_catch, __gxx_personality_v0, etc.)
# The reactor is platform code shared with the Linux host test
# (tests/test_android_reactor); it is never part of libmqvpn.a.
add_library(mqvpn_jni SHARED
    mqvpn_jni.c
    ${MQVPN_ROOT}/src/platform/android/reactor.c)
set_target_properties(mqvpn_jni PROPERTIES LINKER_LANGUAGE CXX)

target_link_libraries(mqvpn_jni
    mqvpn
    lwip_core
    xquic-static
    ssl
    crypto
    log           # Android __android_log_print
    c++_static)   # BoringSSL is C++ — needs libc++ for vtable/RTTI/exceptions

# Test-only JNI entry points (NativeBridgeTestSeams, debug source set). The
# debug build type passes -DMQVPN_JNI_TEST_SEAMS=ON from build.gradle.kts;
# release compiles none of them.
option(MQVPN_JNI_TEST_SEAMS "Compile test-only JNI entry points" OFF)
if(MQVPN_JNI_TEST_SEAMS)
    target_compile_definitions(mqvpn_jni PRIVATE MQVPN_JNI_TEST_SEAMS=1)
endif()
