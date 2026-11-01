# Simulated user testing: bypass rules and multiple servers

Two new features, reviewed with the same method as
[the final loop](final-summary.md):
- **Bypass tailbond** on the router: website, device, address and port rules, plus websites on Android 13+.
- **Several servers with failover**: home + VPS, on the router and the phone.

Fresh reviewer agents each round got only the PNGs, the
[task script](bypass-failover/task-script.md) and the manifests
([GL](bypass-failover/glinet-screens.md), [Android](bypass-failover/android-screens.md)).
Every point was triaged **fix / partial / reject (reason)**. Cap: 3 rounds.

Renders:
- GL: `scripts/ux/glinet/glinet-screens.sh bypass-failover` → `out/ux/bypass-failover/glinet/`
- Android: `android/scripts/gradle.sh :app:testDebugUnitTest --tests '*BypassFailoverScreens*'` → `out/ux/bypass-failover/android/`
- `out/` is not committed.

Personas: A GL traveller, B homelab (Fedora home server + VPS, knows
routing), C family member, D Speedify / SpeedFusion user. Tasks: make the bank
website skip the tunnel; make the work laptop always use tailbond; add a VPS
as a backup; notice the switch to the VPS when home lost power, and the return.

## Findings per round (high / medium / low)

| Round | A traveller | B homelab | C family | D Speedify | Total H / M |
|---|---|---|---|---|---|
| 1 | 2 / 16 / 12 | 2 / 17 / 12 | 3 / 14 / 19 | 2 / 11 / 12 | 9 / 58 |
| 2 | 1 / 13 / 13 | 2 / 14 / 10 | 1 / 15 / 21 | 0 / 11 / 15 | 4 / 53 |
| 3 | 0 / 7 / 16 | 1 / 15 / 11 | 1 / 10 / 14 | 0 / 9 / 15 | 2 / 41 |

Raw reports and triage: [round 1](bypass-failover/round-1/triage.md),
[round 2](bypass-failover/round-2/triage.md), [round 3](bypass-failover/round-3/triage.md).

**Outcome: the target (zero high, mediums only where justified) was not met
within the 3-round cap.**
- Highs fell from 9 to 2, mediums from 58 to 41.
- Round 3's two highs were the kill switch not being shown on the router, and
  "always tailbond" device rules depending on a MAC address that can rotate.
  Both were fixed after round 3: a kill-switch control in the servers card, and
  device rules that follow the device's DHCP name.
- Those fixes and the other round-3 fixes were **not reviewed again**.

Most of the remaining mediums fall into four groups:

| Kind | Examples | Why they stay |
|---|---|---|
| Platform limits | Android VPN apps can't learn sub-names they didn't look up, can't count per-route traffic, can't turn on "Block connections without VPN" themselves; GL's panel has no notification channel | stated on screen and in the docs |
| Ownership boundaries | a subnet clash on a server that already has devices needs that server's admin | both apps give the steps and a message to send |
| Packaging on non-GL firmware | dnsmasq-full missing (GL 4.10 on Flint 3 / Slate 7 ships it) | safest install line given; the Plug-ins page first |
| Depth wanted by the homelab persona | rule order, DNS path and IPv6 handling on screen | in [glinet.md](../glinet.md) and [architecture.md](../architecture.md#bypass-rules-client-role), not on screen |

## What changed because of the loop

- **Clashing servers.** Adding a server that clashes no longer shows an error
  and a success toast at once, and the message is no longer a raw code. It
  says the first server keeps working, who has to act, the exact steps, and
  offers a message to copy.
- **Switches are visible.**
  - The top card says "Using vps since 14:02 (homebox is unreachable)", then
    "Back to homebox in 1:20" with "Stay on vps", then "Back on homebox since 14:21".
  - A **Recent switches** list (agent `failover.history`) and a line on what
    changes on the VPS (public address, connections restart).
  - "How it decides", including what doesn't cause a switch.
  - The Android notification says the same.
- **Bypass is honest about its limits.**
  - "Use the internet directly (now: Repeater)" names the WAN (agent `wans[].default`).
  - Website rules that can't work yet say "Waiting" in amber, not green.
  - The phone's website rules list the names they cover and say the router covers every name.
  - Shared-hosting and secure-DNS limits are one line each.
- **Safety wording.**
  - "Nothing goes out unprotected" says it holds with the kill switch on, and the kill switch is now shown.
  - Android recommends "Block connections without VPN" when there are backup servers, with a button.
  - "Use now" is hidden for unreachable servers.
  - Remove sits behind "More".
  - "Switch back to Tailscale" was renamed and moved away from the failover notes.
- **Product fixes the reviews led to.**
  - Device rules follow the DHCP name.
  - The agent reports the default WAN and keeps the switch history.
  - The Android app resolves the usual sub-names (www, login, secure, …), suggests only tailbond servers when adding one, and keeps its own switch history.
