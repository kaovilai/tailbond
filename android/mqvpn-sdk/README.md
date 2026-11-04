# mqvpn SDK fork (sdk-native, sdk-core)

Copied from `third_party/mqvpn/android/{sdk-native,sdk-core}` at mqvpn v0.19.0
(Apache-2.0, Copyright (c) 2026 mp0rta and mqvpn contributors; see `LICENSE`
and `../NOTICE`). `sdk-runtime` and `sdk-network` are not copied: the Gradle
build uses them straight from the submodule.

The first commit that adds this directory is the verbatim copy, so
`git log -p -- android/mqvpn-sdk` shows every tailbond change. All of them are
marked `tailbond:` in the source.

## Why a fork

tailbond must not modify the submodule, and the stock SDK lacks five things
the tailbond contract needs:

| Change | Files | Why |
|---|---|---|
| `NativeBridge.configSetReinjection` | `mqvpn_jni.c`, `NativeBridge.kt`, `MqvpnConfig.reinjection`, `MqvpnTunnel.create` | the mode table sets `Reinjection = deadline/idle/off`; libmqvpn has `mqvpn_config_set_reinjection` but the JNI never exposed it |
| Leaf-certificate SHA-256 pin | `mqvpn_jni.c` (`setCertPin`, `jni_cert_verify`), `MqvpnConfig.certSha256`, `MqvpnTunnel.create` | the JNI always installs `PlatformTrust` (system CAs) as the verifier, and `insecure=1` disables verification. A self-signed server with `cert_sha256` in the pairing needs a verifier that checks the pin; it uses libmqvpn's real verifier hook, so the pin is enforced in the TLS handshake |
| `NativeBridge.configSetTunMtu` | `mqvpn_jni.c`, `NativeBridge.kt`, `MqvpnConfig.tunMtu`, `MqvpnTunnel.create` | the app caps the TUN MTU at 1280 like tailbondd's client; libmqvpn has `mqvpn_config_set_tun_mtu` but the JNI never exposed it, so the phone got the auto 1382 |
| Path gate | `PathManager` (`wanted`, `reconcile`), `MqvpnVpnService` (`pathPolicy`, `isPathWanted`, `requestPathReconcile`, `availableNetworks`, `onMetrics`) | per-path enable/priority (primary/secondary/backup), failover standby and the cellular data cap attach and detach paths at run time; upstream only fixes the transport set when the tunnel starts |
| TUN rebuild | `MqvpnVpnService.rebuildTun` | per-website bypass (Android 13+) changes the TUN's excluded routes while connected; this re-runs `onCreateTun` and swaps the fd the way a reconnect does, without a new QUIC connection |

Build changes: the CMake project takes `MQVPN_ROOT` (passed by Gradle) for the
sources and the prebuilt archives in the submodule, and adds BoringSSL's
include dir for `SHA256()`. Instrumented tests and the debug-only JNI test
seams are not copied.

## Updating

Bump the submodule, re-copy `src/main` of both modules over this directory,
and re-apply the `tailbond:` hunks (`git diff <import-commit> -- android/mqvpn-sdk`).
