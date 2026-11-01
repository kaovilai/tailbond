# tailbond vs OpenMPTCProuter

Snapshot 2026-11-01: OpenMPTCProuter (OMR) v0.64.2, GPL-3.0, ~2.5k★, very active ([repo](https://github.com/Ysurac/openmptcprouter), [site](https://www.openmptcprouter.com/), [downloads](https://www.openmptcprouter.com/download)).

## In one line

- **OMR**: a full OpenWrt-based **router firmware** plus a **VPS server** script. It aggregates up to 8 WANs with kernel Multipath TCP (and optionally Glorytun, MLVPN, DSVPN, OpenVPN, **MQVPN**) so a whole site gets bonded internet and the VPS's public IP.
- **tailbond**: an **add-on** for stock firmware (GL.iNet), Fedora and Android. It bonds to **your own home server** using MQVPN, with **Tailscale** as the identity, pairing and tailnet layer.

Both now share a data-plane option: OMR ships MQVPN 0.16.2 as one of its tunnels. tailbond pins MQVPN 0.19.0 plus a client-stats patch that upstream MQVPN 0.16.2+ also partly added ("statistics and status available in client mode").

## Feature comparison

| Area | OpenMPTCProuter | tailbond |
|---|---|---|
| Deployment model | Replaces router firmware (flash image) | Package on top of stock firmware: GL.iNet `.ipk`, Fedora RPM, Android APK |
| GL.iNet support | Images for MediaTek GL models (MT2500, MT3000, MT5000, MT6000, X3000, XE3000), which replace GL firmware and the GL UI. **No Flint 3 (BE9300) or Slate 7 (BE3600)**: their IPQ53xx isn't in OMR's build list | Built for Flint 3 / Slate 7 on GL's stock 4.x firmware. Native panel under VPN in GL's admin; GL features (repeater, Login Mode, GL Tailscale) kept |
| Endpoint | Rented VPS (Debian/Ubuntu install script, VPS admin API); gives a dedicated public IP | Your home server (Fedora box or Flint 3); exits via your home IP. A VPS works too if it runs the RPM |
| Aggregation tech | Kernel MPTCP + proxy (Shadowsocks, V2Ray/XRay) for TCP; Glorytun TCP/UDP, MLVPN, DSVPN, OpenVPN, MQVPN for UDP/everything | MQVPN only (MASQUE CONNECT-IP over Multipath QUIC): WLB/MinRTT schedulers, hybrid TCP lane, reorder buffer, reinjection, redundant UDP |
| WAN count | Up to 8 | Any number of interfaces mqvpn binds (router); Wi-Fi + cellular (+ ethernet) on Android |
| Modes | Aggregation vs failover per interface (master/backup), MPTCP schedulers and path managers | Plain-language modes (Fastest / Smooth calls and video / Most reliable / Save mobile data), per-WAN Always / Backup only / Off, data caps |
| Phone client | None (router only) | Android app: bonds the phone's Wi-Fi + 5G, Tailscale-style UI, machines list, quick-settings tile |
| Desktop client | None | Not built (Fedora is server role only). mqvpn's own Linux/macOS/Windows CLIs exist |
| Tailscale | Not integrated (generic OpenWrt package possible) | Core: tailnet reachable through the home server, pairing via `tailscale whois` with one-tap approval, MagicDNS forwarding, `tailscale cert` TLS, peers list, direct-vs-relay health |
| Pairing / onboarding | Wizard: enter VPS IP + key | Discover server over tailnet → ask → approve; QR / deep link fallback; phone landing page at `http://homebox/` |
| Captive portals | Not a feature | Per-WAN detection, scoped sign-in bypass, cooperates with GL's repeater Login Mode; Android sign-in handling |
| Connection monitoring | OMR-Tracker (per-WAN health checks, route changes), status page, graphs | mqvpn per-path stats (RTT, loss, rates) via control API; dashboard sparklines; setup health card |
| Bypass / split routing | OMR-ByPass: per domain/IP/protocol/device to a chosen WAN, bypass via VPS | Android per-app exclude and "tailnet + home LAN only" split; router routes all LAN via tunnel (kill switch optional). **No per-domain/device bypass** |
| Port forwarding / public IP | Yes, ports opened on the VPS forward to LAN hosts | No (your home IP is the exit; inbound goes through your home router as usual) |
| QoS | SQM / QoS packages | None beyond mqvpn's congestion control |
| Server failover | Multiple VPS with failover / load balancing | Single server per client |
| Obfuscation | Encrypted by default; can be obfuscated (V2Ray/XRay, etc.) | QUIC on UDP 443 (looks like HTTP/3); no further obfuscation |
| IPv6 | Yes | Yes inside the tunnel (mqvpn dual-stack) |
| Router extras | Full OpenWrt: any package, VPN over OMR, Pi-hole guides, Wi-Fi config | Whatever GL firmware already offers |
| Maturity | Years of releases, large user base, hardware tested | New; netns e2e + emulated OpenWrt + simulated UX tests; **untested on real hardware** |
| Licence | GPL-3.0 | Apache-2.0 (mqvpn's licence; repo-level LICENSE still to add) |

## Where OMR is ahead

1. **Battle-tested aggregation and monitoring**: MPTCP + several tunnel types, OMR-Tracker, years of field fixes.
2. **Up to 8 WANs with per-WAN master/backup** and server failover across several VPSes.
3. **OMR-ByPass and port forwarding via the VPS** (dedicated public IP, reachable services).
4. **SQM/QoS** and the whole OpenWrt package ecosystem.
5. **Hardware coverage** for x86, Raspberry Pi, Banana Pi and MediaTek GL models.

## Where tailbond is ahead

1. **Stock GL.iNet firmware on Flint 3 / Slate 7**: no reflash, GL UI and features kept. OMR can't run there today.
2. **Phone bonding**: the Android app bonds Wi-Fi + 5G on the phone itself.
3. **Tailscale-native**: tailnet access, pairing, identity, MagicDNS and setup health. No VPS needed.
4. **Travel UX**: captive-portal handling that works with GL Login Mode, plain-language modes, data caps, one-tap pairing.

## Gaps worth closing (ideas taken from OMR)

| Gap | Effort | Note |
|---|---|---|
| Per-domain / per-device bypass (like OMR-ByPass) | Medium | nftables sets + dnsmasq `nftset` on the router; per-app on Android exists |
| Multiple servers with failover (home + VPS) | Medium | Pairing list in the agent; mqvpn client reconnect to the next server |
| Port forwarding when the exit is a VPS | Medium | Only matters if a VPS endpoint is used |
| WAN health tracker independent of mqvpn | Low | The captive-portal probe already covers part of it; extend it to latency/loss probes per WAN |
| SQM on the tunnel | Low | GL firmware has SQM/QoS; document how to apply it to `tb0` |
| Desktop client (Linux/macOS) | Low–Medium | mqvpn CLIs exist; a client-role tailbondd on Fedora is mostly packaging |

## Using both

The two can coexist. OMR at a fixed site with a VPS, and tailbond on travel gear and phones to your home server. They could even share an endpoint: OMR's MQVPN option and tailbond both speak MQVPN, though versions and our extensions (pairing, client stats) would need checking.
