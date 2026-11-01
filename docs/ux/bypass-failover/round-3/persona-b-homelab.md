# Round 3: Persona B, "homelab"

Fedora home server, a rented VPS, uses Tailscale daily, knows routing,
subnets, nftables and DNS. Wants to know exactly what goes where, how failover
decides, and what happens to subnets and the kill switch. Judged only from
the PNGs in `out/ux/bypass-failover/{glinet,android}/` and the two screen
manifests.

## Task 1: Make `mybank.com` skip the tunnel (router, then phone)

**Done?** Yes on both. Router: bypass-empty, bypass-add-website-presets,
bypass-rules (+ phone), bypass-website-rules-wait. Phone: 46c-websites-none-yet,
47-add-website-dialog, 47b-add-website-preset-banking, 46-websites-skip-tailbond,
46b-websites-needs-android-13, 48-settings-skip-rows-android-12.

The router row shows where the traffic goes ("Uses the internet directly (now:
Repeater) · 37 connections"), and the "Waiting" state is safe: the bank stays
on the tunnel instead of half-working.

Findings:

- **medium**, bypass-add-website-presets-desktop.png: The router says a website rule "matches the addresses the router looks up for that name". It never says which resolver does that lookup: homebox/vps through the tunnel, or the hotel's DNS. If the server resolves it, a geo-DNS bank or CDN can return home-region addresses that I then reach from the hotel. If the hotel resolves it, the query leaks. The phone page does say "looked up through your server". **Suggestion:** add one line to the router dialog: "Looked up through homebox (or vps while in use); the device's own DNS query also goes through tailbond."
- **medium**, bypass-add-website-presets-desktop.png / bypass-add-wifi-calling-desktop.png: The "It should" dropdown is closed on every screen, so I never see the promised "use only one of the router's connections" option. "The router's main internet connection (now Repeater)" doesn't say what makes Repeater "main" when both Repeater and Tethering are "Always", or whether the bank moves to Tethering when Repeater drops. **Suggestion:** show the options under the dropdown, and say "main = the first connection that's online, in this order: Ethernet, Repeater, Tethering; switches by itself" (or whatever the rule really is).
- **medium**, bypass-website-rules-wait-desktop.png: The SSH fallback `opkg remove dnsmasq && opkg install ./dnsmasq-full*.ipk || opkg install dnsmasq` removes the running resolver before the replacement is proven to install. It gives no backup of `/etc/config/dhcp`, no verify step, and doesn't say whether LAN clients leak or lose DNS while tailbond is "off" for the reinstall. **Suggestion:** use `opkg install --force-overwrite` / a download-then-swap order, add `cp /etc/config/dhcp /tmp/dhcp.bak`, and finish with a check command (`dnsmasq --version | grep -q nftset`). Say what happens to traffic during the off/on.
- **medium**, 46-websites-skip-tailbond.png / 47c-add-website-preset-streaming.png: On the phone each row says "Covers nflxvideo.net and its usual names (www., login., secure., online. …)". For a CDN domain those names mean nothing: the real hosts are arbitrary sub-names, so the preset looks complete but video doesn't match. The phone also never shows what a rule actually matched (the router shows connection counts). **Suggestion:** for CDN entries in presets, say "only the names listed are covered; video servers aren't". Show "N addresses · M connections" per row, and let me add extra names inside one rule.
- **low**, 46-websites-skip-tailbond.png: When the phone is on a tailbond router's Wi-Fi, the phone's bank rule only gets the bank as far as the router, and the router tunnels it again unless it has its own rule. Every other packet is double-tunnelled (phone→homebox inside router→homebox). This is only in the small print at the bottom. **Suggestion:** detect the tailbond router's Wi-Fi and show a banner: "On a tailbond router: add mybank.com there too, or turn tailbond off on this phone."
- **low**, 46-websites-skip-tailbond.png: The off rule (tax.example.gov) is only a grey toggle, with no "Off" text, and the subtitle still claims it "Covers …". **Suggestion:** prefix "Off ·" to the subtitle.

## Task 2: Make the work laptop always use tailbond

**Done?** Partly. I can create the rule (bypass-add-device-picker desktop + phone, quick pick "Work laptop", row in bypass-rules), but I cannot verify that it can never leave through the hotel.

Findings:

- **high**, bypass-add-device-picker-desktop.png / servers-home-active-vps-standby-desktop.png: The guarantee rests on "With the kill switch on (the default), it has no internet while tailbond can't connect" and "with the kill switch on (the default), nothing goes out unprotected". No screen shows a kill switch control or its current state. If someone turned it off, the laptop would silently leave through the hotel during a switch or an outage, and the panel would still say "Always uses tailbond". **Suggestion:** show the kill switch state on the "Always uses tailbond" row and in the Servers card ("Kill switch: on — devices here have no internet while tailbond is down"), with a link to where it's changed. Show a warning on the row if it's off.
- **medium**, bypass-add-device-picker-desktop.png: The rule is keyed on the MAC address. The dialog admits "If its address changes anyway, it goes through tailbond like everything else, but website rules then apply to it". In other words, a Windows laptop that rotates its MAC quietly falls under the bank/streaming bypass, and the list shows no warning. **Suggestion:** on the rule row show "not seen with this address since …". Flag a new MAC that reports the same DHCP hostname ("work-laptop may be back with a new address: update the rule?"), and optionally match by DHCP hostname or a fixed lease too.
- **medium**, bypass-rules-desktop.png: Precedence ("even websites or ports set to skip tailbond") is only stated inside the add-device dialog. The list shows four flat rows with no order. Rule scope is also unstated: does "UDP ports 4500 and 500" mean destination ports, and for every device? It catches every IPsec/IKE flow on the LAN, not just phones' Wi-Fi calling. **Suggestion:** on the "Always uses tailbond" row add "overrides the rules below". On port rules show "to UDP 500, 4500 · all devices" and offer to limit them to chosen devices.
- **medium**, bypass-add-device-picker-desktop.png: Nothing says how IPv6 or DNS is handled for an "Always" device. If the hotel hands out IPv6 via RA, or the laptop uses its own DoH, I can't tell whether that leaves outside the tunnel. **Suggestion:** one line: "Its DNS and IPv6 also go through tailbond (or are blocked)". Say whatever is true.
- **low**, bypass-rules-desktop.png: The same page shows "work-laptop · connected now" (LAN rule) and, in Tailnet machines, "work-laptop · Last seen 1 day ago". Same name, opposite states. **Suggestion:** label the tailnet list's entries as "on Tailscale", or note that the LAN device is a separate identity.

## Task 3: Add the VPS as a backup, and understand a clash

**Done?** Yes on both (the clash path ends at "send a message"). Router: servers-single-home, add-server-open, add-server-waiting-approval, add-server-subnet-clash, add-server-paste-clash, servers-home-active-vps-standby. Phone: 44b-settings-one-server-add-backup, 45-add-another-server, 45b-add-another-server-confirm, 45c-add-another-server-same-addresses, 44-settings-servers.

The clash screens say clearly that nothing broke, name the range (10.77.0.0/24), and explain why the automatic fix didn't happen.

Findings:

- **medium**, add-server-subnet-clash-desktop.png / 45c-add-another-server-same-addresses.png: I *am* "whoever runs vps", yet the actual fix is behind a collapsed "How they change it" on the router and missing on the phone ("Copy a message to send them"). I don't know which setting or command to run, or which range is free. **Suggestion:** show the steps inline, with the exact setting/command on vps and a suggested free range (e.g. 10.83.142.0/24). Keep "Copy a message" for the case where someone else runs it.
- **medium**, servers-single-home-desktop.png vs 40-home-on-homebox-vps-on-standby.png: The router's tunnel address on homebox is 10.77.0.2, and the phone's tunnel address on homebox is also 10.77.0.2 (both are 10.83.142.2 on vps). Right after being told that an overlapping address is fatal, two clients sharing one address looks like a bug or a routing collision. **Suggestion:** give each client its own address, or say why sharing is fine ("each device's address is private to its own tunnel; Tailscale sees it as homebox").
- **medium**, servers-home-active-vps-standby-desktop.png / failover-using-vps-home-unreachable-desktop.png: "Optional: to reach devices behind this router from your Tailscale machines, approve each server's route in Tailscale." It doesn't say which prefix (192.168.8.0/24?). If homebox and vps both advertise it, Tailscale's own subnet-router failover picks a primary independently of tailbond, so return traffic can go to the standby server. **Suggestion:** show the exact prefix with a copy button, and state the behaviour, e.g. "only the server in use advertises 192.168.8.0/24", or "both do; Tailscale may send replies to the other one".
- **medium**, add-server-waiting-approval-desktop.png / 45-add-another-server.png: Approval happens on `http://vps.tail1234.ts.net:8077` "from any device on your Tailscale". On a VPS with a public IP I need to know that :8077 isn't bound to the public interface, and who on the tailnet can click Approve (any user, or only the owner?). **Suggestion:** add "the page only listens on Tailscale; anyone who can reach vps over Tailscale can approve, so limit tcp/8077 in your ACLs" (or whatever the real policy is).
- **low**, 45-add-another-server.png: The phone offers nas, vps, apple-tv and pixel-tablet as machines to "ask", even though only vps runs tailbond. The router detects vps directly ("Found on your tailnet: vps"). **Suggestion:** put machines that answer tailbond first, mark them, and grey out the rest.
- **low**, add-server-open-desktop.png: vps is "Found on your tailnet" but isn't listed under Tailnet machines on the same page (nor is homebox). **Suggestion:** list the servers in Tailnet machines with a "tailbond server" tag, as the phone does for homebox.
- **low**, servers-home-active-vps-standby-desktop.png: The router's vps row shows "203.0.113.9" with no port or protocol, while the phone shows 203.0.113.9:8443 and homebox shows "UDP port 443". I need the port for the VPS firewall. **Suggestion:** show "203.0.113.9 · UDP 8443" on the row.
- **low**, add-server-paste-clash-desktop.png: The text says to try "Its name in Tailscale" above, but the field is labelled "Or type its name in Tailscale", and the "Found on your tailnet: vps · Ask to add" button is right there. **Suggestion:** "click Ask to add next to vps above".

## Task 4: Notice the switch, and that it came back

**Done?** Yes on the screens themselves. Router: failover-using-vps-home-unreachable (+ phone), failback-countdown, failover-you-picked-vps, back-on-homebox-since, servers-recent-switches-all. Phone: 41/41b, 41c, 41d notification, 42, 43, 43b, 44, 44c, 44d, 49.

The router header states the switch time, the reason, and the side effects (new public IP, connections restart, tailnet sees devices as vps). The "How it decides" text gives real numbers (10 s probe, 15 s down, 2 min up). This is what I wanted.

Findings:

- **medium**, 41d-notification-using-vps.png: The switch only changes the text of the ongoing notification. Nothing suggests an alerting (heads-up/sound) notification, so I only "notice" it if I pull the shade. The router has no alert channel at all. **Suggestion:** one-shot alerting notifications for "switched to vps" and "back on homebox" (separate channel, user can silence). On the router, write switches to syslog and offer GL's push/email.
- **medium**, 49-home-error-no-server-reachable.png: With both servers down, the phone "stopped" and fails open to normal internet. The only way back is a manual "Try again", so when homebox returns the phone stays untunnelled indefinitely. No router screen shows the both-down state, so I can't tell what the kill switch does there. **Suggestion:** keep retrying in the background with backoff and say so ("retrying every 30 s"). Add the router both-down state, showing whether LAN devices are blocked.
- **medium**, failback-countdown-desktop.png / 42-home-back-to-homebox-in-1-20.png: "Stay on vps for now" / "Stay on vps" doesn't say for how long or what it changes: until I click again, until the next outage, or until a reboot? The countdown is relative ("in 1:20"), with no clock time and no "answering again since HH:MM". **Suggestion:** "Back to homebox at 14:21 (answering since 14:19)" and "Stay on vps: pauses switching until you pick 'Go back to automatic'".
- **medium**, failover-you-picked-vps-desktop.png / back-on-homebox-since-desktop.png: There are three controls: "Use now" on homebox, "Go back to homebox", and "Go back to automatic". The recent switch at 14:38 reads "you picked homebox". If picking homebox also pauses automatic switching, the next outage won't fail over, and nothing on back-on-homebox-since shows whether automatic is back on. **Suggestion:** always show the mode next to the in-use server ("automatic" / "paused: you picked X"). Make "Use now" on the first-choice server simply mean "go back to automatic".
- **low**, servers-home-active-vps-standby-desktop.png: The collapsed "Recent switches" hides the one row that explains the history (14:00 "homebox stopped answering"). **Suggestion:** always show the most recent automatic switch, even when collapsed.
- **low**, failover-using-vps-home-unreachable-desktop.png: "homebox is unreachable" doesn't separate "home is down" from "this hotel can't reach home" (e.g. UDP 443 to the home IP is filtered). **Suggestion:** add "vps answers over the same connections, so the problem is on homebox's side".
- **low**, failover-using-vps-home-unreachable-desktop.png vs 41-home-using-vps-homebox-unreachable.png: The router says 14:00 and the phone says 14:02, with no time zone. When travelling, the router's clock zone may be home time. **Suggestion:** show the zone when it differs from the browser's ("14:00 router time, UTC−5").
- **low**, bypass-wifi-calling-added-desktop.png: The toast reads "(now: repeater)" in lowercase, unlike the rows' "Repeater", and covers the homebox card. **Suggestion:** match the capitalisation and anchor the toast to the bypass card.

## Count

- **high:** 1
- **medium:** 15
- **low:** 11
