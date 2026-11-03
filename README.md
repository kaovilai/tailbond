<img src="assets/icon.svg" width="64" height="64" alt="">

# tailbond

Speedify-style WAN bonding to **your own** server, with your [Tailscale](https://tailscale.com) tailnet reachable through it.

tailbond combines Wi-Fi, 5G/cellular, ethernet and phone tethering into one tunnel to a server you run: a box at home, a VPS, or both with failover. The data plane is [mqvpn](https://github.com/mp0rta/mqvpn) (MASQUE CONNECT-IP over Multipath QUIC). tailbond adds the agent, web UIs, Tailscale integration, pairing, packaging and an Android app.

> **Status: early.** Everything is covered by unit tests, network-namespace end-to-end tests, an emulated OpenWrt install test and simulated user testing. It has **not yet been run on real GL.iNet routers, phones or VPSes**. Expect rough edges and please file issues.

## What it does

- **Bonding modes:** Fastest (aggregate bandwidth), Smooth calls and video, Most reliable (duplicate packets), Save mobile data (backup-only links, data caps).
- **Tailnet through your server:** your Tailscale machines stay reachable, with MagicDNS names. Devices pair with one tap: they ask the server over Tailscale and you approve.
- **Home + VPS failover:** an ordered server list. Switches after 15 s of outage and fails back after 2 min healthy.
- **Bypass rules (router):** send chosen websites, devices, addresses or ports direct, or through a chosen WAN.
- **Pair & Share:** your phones lend their mobile data to the travel router over Wi-Fi (Android app, iPhone app with a free Apple ID), and it bonds them with its own links. USB-tethered phones are bonded too. [Guide](docs/pair-and-share.md)
- **Hotel Wi-Fi:** captive-portal detection and sign-in that cooperates with GL.iNet's own repeater Login Mode.
- **Setup health:** direct-vs-relayed Tailscale peers, CGNAT detection, port reachability, with links to fix each.
- **Updates and backups:** GitHub releases (COPR / dnf on Fedora, one click on VPS and GL routers, Obtainium on Android); encrypted config backup and restore.

## Pieces

| Where | What | Guide |
|---|---|---|
| Home server | Fedora RPM, or a GL.iNet Flint 3 | [Fedora](docs/fedora-server.md) · [GL.iNet](docs/glinet.md) |
| VPS | `install.sh` for Debian, Ubuntu, AlmaLinux/RHEL/Rocky, Fedora; Ansible role | [VPS](docs/vps.md) |
| Travel router | GL.iNet Slate 7 (stock firmware, native admin panel) and other OpenWrt 23.05+ aarch64 routers | [GL.iNet](docs/glinet.md) |
| Phone | Android app (Wi-Fi + mobile data bonding; shares its mobile data with the travel router) | [Android](docs/android.md) |
| iPhone | tailbond Share: lends its mobile data to the travel router (no VPN, builds with a free Apple ID) | [iOS](docs/ios.md) · [Pair & Share](docs/pair-and-share.md) |

More: [setup tips for best speed](docs/setup-tips.md) · [updates](docs/updates.md) · [backup and restore](docs/backup-restore.md) · [architecture and API contract](docs/architecture.md) · [comparison with OpenMPTCProuter](docs/comparison-openmptcprouter.md)

## Building

```sh
git clone --recurse-submodules https://github.com/kaovilai/tailbond
cd tailbond
agent/build.sh                          # tailbondd (static, amd64 + arm64)
scripts/build-mqvpn-native.sh           # patched mqvpn for this machine
packaging/fedora/build-rpm.sh           # Fedora RPM → out/rpm/
scripts/build-mqvpn-aarch64-musl.sh && packaging/openwrt/build-ipk.sh   # GL.iNet/OpenWrt → out/ipk/
android/scripts/setup-sdk.sh && android/scripts/build-apk.sh            # Android → out/android/
(cd ios && xcodegen) && open ios/TailbondShare.xcodeproj               # iPhone (macOS, Xcode), see docs/ios.md
scripts/e2e-netns.sh                    # end-to-end test (needs sudo)
scripts/e2e-share.sh                    # Pair & Share end-to-end (needs sudo)
```

CI (`.github/workflows/ci.yml`) builds and tests all of it and publishes releases on `v*` tags.

## License

[Apache-2.0](LICENSE). See [NOTICE](NOTICE) for third-party software. Tailscale, GL.iNet, Speedify, Peplink and SpeedFusion are trademarks of their respective owners; tailbond is not affiliated with them.
