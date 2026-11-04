# Round 1: persona N, NetBird at work

## Task 1: is it bonding, over which links

Steps: click the menu bar icon (menu-01-connected.png). "Connected to homebox · 2 links", Wi-Fi (en0) 38 ms and iPhone (USB) (en7) 61 ms, Sam's phone on standby. That answers it in a few seconds.

- **Medium**: the on/off switch is grey in light mode and white in dark (menu-01, -dark), so it looks off or disabled while the header says Connected. Use the system accent tint when it's on.
- **Low**: "Sam's phone (sh5a1b2c)": I don't know what sh5a1b2c is (a peer ID? a hostname?). Add a label or drop it.
- **Low**: links show only download speed. The header has ↑ too, so show upload per link as well.
- **Low**: menu-01 has no "Next to other VPNs" section. When the section is hidden I can't tell "no other VPN found" from "NetBird detection didn't run". Show a single line like "No other VPN detected".

Verdict: done quickly; the switch colour is the only thing that made me hesitate.

## Task 2: keep work traffic direct

Steps: menu-02-netbird.png. "NetBird is on (100.92.1.2/16): its traffic goes Direct (not through tailbond)". "Work VPN traffic: Direct | Bonded", and Direct is already selected. If it said Bonded, I'd click Direct.

- **Medium**: "Direct" only names the overlay, 100.92.0.0/16. It doesn't say whether the routes NetBird pushes (corp subnets) or NetBird's own WireGuard/relay UDP underlay also skip tailbond. List the prefixes that stay direct, and say that the NetBird tunnel packets themselves go out the physical link.
- **Medium**: nothing next to "Bonded" warns that it sends company traffic out through my home server. For me that is exactly "routed somewhere unexpected". Add a line under the control, e.g. "Bonded: work traffic exits via homebox", or ask for confirmation.
- **Medium**: "NetBird's network 10.20.0.0/16 is also your home LAN's: NetBird keeps it". I understand the conflict, but not the consequence (I can't reach home LAN hosts?) or what to do about it. Spell out the effect and link to the fix.
- **Low**: the selected segment of Direct/Bonded is only slightly darker in light mode. Make the selection obvious.
- **Low**: the work-names line (corp.example, netbird.cloud) has no way to test it or see the resolver. A "Test" action or the resolver IP would let me confirm.
- **Low**: the doc's NetBird section ("below") is cut off, and the menu has no "Learn more" link to it.

Verdict: the setting is easy to find and already correct, but it doesn't tell me enough to trust where corp subnets and the NetBird underlay go.

## Task 3: version, pending update, install right away

Steps: the Updates section (menu-02): "tailbond 0.2.0 · up to date · Tell me". Change the Install updates menu to Right away and click Check now.

- **Medium**: menu-03 says "Update available: 0.2.1 (security) (waiting for a quiet moment) · Right away". "Right away" while waiting is a contradiction. I don't know what a quiet moment is, or whether installing drops my tunnel (and NetBird?) in the middle of a call. There is no "Install now" button either. Say what triggers the install, whether it interrupts traffic, and add Install now.
- **Low**: the policy appears twice (status line and menu). Drop it from the status line.
- **Low**: the icon next to "up to date" is the same kind of circle as an update/download icon. Use a checkmark.

Verdict: doable. The update timing would make me nervous on a work day.

## Task 4: report the crash without leaking anything

Steps: menu-03 banner, then Report a problem…, which opens the dashboard (report-01), then the dialog (report-02). I type what happened, read the crash details and logs, tick the box and click Open GitHub issue (report-03).

- **High**: the notice says what gets replaced: keys, pairing links, public IPs, emails, tailnet name. It doesn't mention private/CGNAT addresses (100.92.1.2, 10.20.0.0/16), internal names (*.corp.example) or NetBird peer names. tailbond already knows these from menu-02, so redact them, or at least list them in the notice as things to check.
- **High**: crash details and logs don't wrap ("config=/tmp/", "Report a", "user [email]" are cut off at the edge in report-03). I'm asked to confirm I've read the report when I can't see every line. Wrap the text, or show the full preview before the checkbox.
- **Medium**: the prefilled values are wrong for a Mac: Where "VPS / Linux install.sh", Device and OS "Fedora Linux 44 (WSL), amd64". report-01 is the home server's dashboard ("No devices connected", "Add a device in the Clients tab", but there's no Clients tab). Am I reporting my Mac's crash or the server's? Prefill macOS and state which machine's crash this is.
- **Medium**: the crash time is 2026-11-04T06:51:24Z (raw UTC) in menu-03 but 11/4/2026, 7:14:10 AM in report-01/02, so I can't tell whether it's the same crash. Use one format in local time and make sure it's the same event.
- **Low**: the excerpt says the copied and saved reports "have everything". Does that mean unredacted? Say so in the dialog.
- **Low**: "Report a problem…" appears twice in menu-03, and in the footer even with no crash (menus 01, 02, 04, 05). Its "…" suggests a dialog, but it opens the browser. Use "Report a problem (opens dashboard)".
- **Low** (dark): the grey secondary text ("standby", "not working", the launchctl hint in menu-05-dark) has weak contrast.

Verdict: I'd copy the full report, scrub corp names by hand and post it myself. As it stands I wouldn't tick the box.
