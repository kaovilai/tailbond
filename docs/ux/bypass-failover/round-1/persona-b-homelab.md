# Round 1: Persona B, "homelab"

Runs a Fedora home server plus a rented VPS, uses Tailscale every day, and
knows routing, subnets, nftables and DNS. Wants to see exactly what goes
where, how failover decides, and what happens to subnets and the kill switch.
Based only on the PNGs in `out/ux/bypass-failover/{glinet,android}/` and the
two screen manifests.

## Task 1: Make `mybank.com` skip the tunnel (router, then phone)

**Done?** Partly. Router: yes, but only once dnsmasq-full is installed. Phone:
the rule saves, but it probably doesn't cover the bank's login host.
Screens: glinet `bypass-empty`, `bypass-add-website-presets`, `bypass-rules`,
`bypass-website-rules-wait`, `bypass-add-streaming-preset`,
`bypass-add-wifi-calling`; android `48-settings-skip-rows-android-12`,
`46c-websites-none-yet`, `47-add-website-dialog`, `47b/c/d-add-website-preset-*`,
`46-websites-skip-tailbond` (+dark), `46b-websites-needs-android-13`.

Findings:

1. **medium**, `47-add-website-dialog.png` / `46-websites-skip-tailbond.png` vs glinet `bypass-add-website-presets-desktop.png`.
   The same entry means different things on each platform. The router matches
   the whole domain ("like www.mybank.com and login.mybank.com"). The phone
   matches only the name plus `www.`, and its footnote says "parts of a site
   loaded from other names still use tailbond". So on the phone,
   `login.mybank.com` / `secure.mybank.com` still exit through the tunnel, and
   the bank blocks the login, which is the exact thing I was trying to avoid.
   *Suggestion:* match subdomains on Android too (re-resolve `*.` names you see
   in DNS answers), or say plainly in the dialog "only mybank.com and
   www.mybank.com; add login.mybank.com etc. separately", and let me add
   several names to one rule.

2. **medium**, `bypass-website-rules-wait-desktop.png`.
   "Install dnsmasq-full" has no button, no command, and no warning that
   swapping dnsmasq on GL firmware can break GL's own DNS features. It also
   doesn't say what `mybank.com` does while the rule shows "Waiting". It
   presumably still goes through the tunnel and gets blocked, but I have to
   guess.
   *Suggestion:* add "Until then this website still goes through tailbond"
   to the notice, plus a copyable `opkg update && opkg remove dnsmasq && opkg
   install dnsmasq-full` (or a one-click install that tells me what it changes),
   and say whether GL features are affected.

3. **medium**, `bypass-add-website-presets-desktop.png`.
   "Uses this router's internet, whichever connection GL picks" is vague. If GL
   picks Tethering, bank traffic burns phone data, and the source IP can change
   between Repeater and Tethering mid-session, which banks often treat as
   session hijacking. The dropdown is closed in every screenshot, so I can't
   see the "only one connection" option the feature description promises.
   *Suggestion:* show the options under "It should" (for example "Internet
   directly, Repeater only / Tethering only") and recommend pinning banks to
   one connection.

4. **medium**, `46-websites-skip-tailbond.png`.
   "New addresses for a website reconnect tailbond for a moment", and the
   addresses are re-resolved every 10 minutes. With CDN-hosted sites whose IPs
   rotate, that could mean a tunnel blip every 10 minutes that drops my SSH
   and other long-lived sessions. Lookups also go "through your server", so the
   answers are geo-DNS for home or the VPS, not for the hotel. That undercuts
   the "site that should see where you really are" use case.
   *Suggestion:* say how long "a moment" is and whether open connections
   survive it. Resolve bypass names over the local network (not the tunnel) so
   the CDN answers match the path the traffic actually takes.

5. **medium**, `bypass-add-wifi-calling-desktop.png`.
   The manifest says this shows the Wi-Fi calling quick pick with UDP 4500/500,
   but the screenshot is just the empty card with no dialog. Either the chip
   does nothing visible, or the dialog opened off-screen. I can't check the
   port-rule form at all, and "More: an address or port" isn't shown expanded
   anywhere.
   *Suggestion:* fix the capture or the click handler, and add a screenshot of
   the address/port form showing protocol, port range and CIDR input.

6. **low**, glinet `bypass-add-streaming-preset-desktop.png` vs android `47c-add-website-preset-streaming.png`.
   The streaming quick pick adds five sites on the router but only
   netflix.com and nflxvideo.net on the phone.
   *Suggestion:* use the same list on both, or label the phone version
   "Netflix" instead of "Streaming".

7. **low**, `47d-add-website-preset-wifi-calling.png`.
   The Wi-Fi calling chip is a dead end ("Nothing to add") inside an add
   dialog.
   *Suggestion:* drop the chip on Android, or turn it into a link to "Apps that
   skip tailbond".

## Task 2: Work laptop always uses tailbond

**Done?** Yes on the router: a device rule set to "Always use tailbond", and
the help text says it wins over website and port rules and blocks rather than
leaking. Open questions remain about failover and default traffic.
Screens: glinet `bypass-add-device-picker` (desktop+phone), `bypass-rules`
(desktop+phone), `failover-using-vps-home-unreachable-desktop`.

Findings:

8. **medium**, `bypass-add-device-picker-desktop.png`.
   "Everything from this device goes through **homebox** ... If tailbond is
   down, it waits." It doesn't say what happens when tailbond fails over to
   vps. Does work traffic then exit from a rented VPS, a third-party datacenter
   my employer may not allow, or is it held because homebox is down? The
   wording ties the promise to homebox.
   *Suggestion:* say "through tailbond (homebox, or vps while homebox is
   down)", and offer "homebox only" as a separate choice for traffic that must
   never touch the VPS.

9. **medium**, glinet `bypass-rules-desktop.png` / `failover-using-vps-home-unreachable-desktop.png` and android `49-home-error-no-server-reachable.png`.
   No screen says what happens to traffic that has no rule when the tunnel is
   down or switching servers (the 15 s detection window, or both servers
   unreachable). The router only says rules that skip tailbond "keep working
   while it's down". The phone shows "Not connected / Stopped" with the toggle
   off, which suggests apps fall back to the hotel network. The only kill-switch
   hint is the advice to leave Android's "Block connections without VPN" *off*.
   *Suggestion:* add one explicit line on each platform, e.g. "While tailbond
   is reconnecting or switching servers, other traffic is held / goes direct",
   plus a router "Block when tailbond is down" toggle. On Android, explain how
   "Block connections without VPN" interacts with websites that skip tailbond.

10. **low**, `bypass-rules-desktop.png`.
    The work-laptop row shows only a MAC, with no connection count (the other
    rules show one) and no current IP or online state. With per-network private
    MACs I can't tell whether the rule is matching anything right now.
    *Suggestion:* show "matching: 192.168.8.23, N connections" or "not seen
    since …", and warn when the device's MAC hasn't been seen for a while.

11. **low**, `bypass-add-device-picker-desktop.png`.
    Picking an arbitrary device (alex-iphone) defaults to "Always use tailbond".
    The card is called "Bypass tailbond", so I'd expect the default to be the
    bypass.
    *Suggestion:* only default to "Always use tailbond" when the "Work laptop"
    quick pick was used. Otherwise make me choose.

12. **low**, `bypass-rules-desktop.png`.
    The list doesn't show precedence (device > website/port). That rule is only
    in the add dialog's help text.
    *Suggestion:* add a one-line note under the list, or sort "always tailbond"
    device rules first and label them "overrides website rules".

## Task 3: Add the VPS as a backup server and handle a clash

**Done?** Partly. Router via the tailnet: yes. Router via a pasted link: the
outcome is contradictory. Phone: yes, with clear clash guidance.
Screens: glinet `servers-single-home`, `add-server-open`,
`add-server-waiting-approval`, `add-server-subnet-clash`, `add-server-paste-clash`,
`servers-home-active-vps-standby`; android `44b-settings-one-server-add-backup`,
`45-add-another-server`, `45b-add-another-server-confirm`,
`45c-add-another-server-same-addresses`.

Findings:

13. **high**, `add-server-paste-clash-desktop.png`.
    Two messages appear at the same time: a red banner "That didn't work:
    subnet_in_use" (a raw error code) and a green toast "The server added as a
    backup". The server list still shows only homebox. A user who trusts the
    toast thinks failover is configured when it isn't, and finds out only when
    home goes down. This path also gives no fix instructions, unlike the
    tailnet path.
    *Suggestion:* suppress the success toast when the add fails, and reuse the
    full clash explanation (which subnet, which file, which value) from
    `add-server-subnet-clash`. Fix the toast grammar too ("vps added as a
    backup").

14. **medium**, glinet `add-server-subnet-clash-desktop.png` vs android `45c-add-another-server-same-addresses.png`.
    The router says to change "the setting `server.subnet`" with no file path
    and without naming the clashing range. The phone says `/etc/tailbond/tailbond.json`,
    `10.77.0.0/24`, and that "asking over Tailscale does this by itself while vps
    has no other devices". But the router's clash came from asking over
    Tailscale, so the two screens contradict each other on whether the
    tailnet path auto-fixes it.
    *Suggestion:* use one message on both platforms with the file path, both
    ranges, a ready-to-paste `jq`/sed one-liner, and the exact conditions for
    auto-renumbering. Keep the typed name in the field so I can retry after
    fixing.

15. **medium**, `45-add-another-server.png`.
    The "add a backup" page reuses the first-pairing text: "Came here from your
    **home server's** page", "open the **home server's** dashboard
    (http://homebox:8077)", "Scan code from your **home server**". Following it
    literally pairs with homebox again instead of vps.
    *Suggestion:* use "the other server" / "vps" throughout (for example
    `http://vps:8077`), or hide the cards that only apply to first pairing.

16. **medium**, `add-server-waiting-approval-desktop.png`.
    "Click Approve … at http://vps:8077 from any device on your Tailscale"
    suggests that anyone on my tailnet, shared nodes included, can approve a
    device onto my VPS through an unauthenticated HTTP page.
    *Suggestion:* say who can approve (for example "signed-in tailnet admin /
    devices of alex@example.com only") and whether the page needs a login.
    Show the router's key fingerprint so I can check it on the vps side.

17. **low**, `servers-home-active-vps-standby-desktop.png`.
    The router's server rows show `203.0.113.9` with no port or protocol, and
    homebox's "UDP port 443" disappeared once there were two servers. The phone
    shows `203.0.113.9:8443`. I need the port and protocol to open the VPS
    firewall.
    *Suggestion:* show `host:port/udp` for every server in the router list.

18. **low**, glinet `servers-single-home-desktop.png` / android `40-home-on-homebox-vps-on-standby.png`.
    The router and pixel-8 both show tunnel address `10.77.0.2` on homebox,
    and both show `10.83.142.2` on vps. If both are clients of the same server,
    that looks like an address collision.
    *Suggestion:* if addresses are per client, make the fixture show distinct
    ones. If they really are shared, explain why that's safe.

19. **low**, `servers-home-active-vps-standby-desktop.png`.
    "Remove" sits on the server that's currently in use, and "Forget all
    servers" sits next to "Add another server". No confirmation dialog is
    shown for either.
    *Suggestion:* confirm both, and warn when removing the active server
    ("traffic will move to vps").

## Task 4: Notice the switch, and the switch back

**Done?** Partly. I can tell I'm on vps and why on both platforms, and see the
1:20 countdown. I can't tell when the switch happened, where my internet
traffic exits now, or what changed for tailnet peers (router).
Screens: glinet `failover-using-vps-home-unreachable` (desktop+phone),
`failback-countdown`, `failover-you-picked-vps`; android `41`, `41b`, `42`, `43`,
`43b`, `49`, `44-settings-servers` (+dark), `40` (+dark).

Findings:

20. **high**, android `45b-add-another-server-confirm.png` and `44-settings-servers.png`.
    The confirm dialog says that through vps "your internet traffic goes out
    from home", and Settings → Internet traffic says "All internet traffic
    leaves from home". When homebox is unreachable, that can't be true. Traffic
    either exits from the VPS's datacenter IP or nowhere. For someone choosing
    between privacy and geo (bank, streaming), this misstates where traffic
    goes. The router never says where internet traffic exits at all.
    *Suggestion:* show the current exit in the status card, e.g. "Internet
    traffic exits from vps (203.0.113.9)". Reword the setting to "through your
    server (homebox, or vps while homebox is down)", and offer "Tailscale only
    while on a backup".

21. **medium**, glinet `failover-using-vps-home-unreachable-desktop.png` / `servers-home-active-vps-standby-desktop.png`.
    The Bypass tailbond card disappears from every multi-server screen (and from
    `servers-single-home`). On vps I can't confirm that the bank and work-laptop
    rules still apply. The card's own text says "Everything else goes through
    homebox", which is wrong in this state.
    *Suggestion:* always show the card, and change its text to "through
    tailbond (now: vps)".

22. **medium**, glinet `servers-home-active-vps-standby-desktop.png`, android `44-settings-servers.png`.
    How failover decides isn't explained. "15 s without an answer" (router) vs
    "15 seconds without a working connection" (phone): an answer to what probe,
    sent over which link? Does losing only hotel Wi-Fi count? Is it per-link or
    end to end? And what happens to open TCP sessions on a switch?
    *Suggestion:* add a "How switching works" disclosure covering the probe
    (keepalive over the bond, any link counts), the interval, the 15 s
    threshold, the 2-minute stable window, and "open connections are reset when
    the server changes".

23. **medium**, `failover-using-vps-home-unreachable-desktop.png`.
    The router's tunnel address quietly changes from `10.77.0.2` to
    `10.83.142.2`, and tailnet peers now see LAN devices as vps. The router
    never mentions this or the vps subnet-route approval in the admin console.
    The phone does (android `41`: "they see this phone as vps … only if vps's
    tailbond route is approved"). Without that, ACL rules and inbound
    connections to router LAN devices silently break on failover.
    *Suggestion:* show the phone's "this device" explanation on the router as
    well, and show whether each server's route is approved, e.g. "vps route
    10.83.142.0/24: not approved → link to admin console".

24. **medium**, all four Task 4 router screens and android `43-home-back-on-homebox.png`.
    There's no timestamp or event history. "Up 1 min" / "Up 9 min" and "last
    answered 2 min ago" are relative and ambiguous (last answered 2 min ago but
    connected to vps for 1 min, so was the switch 15 s or 60 s?). "Back on
    homebox" has no time. When I come back to the panel I can't answer "when
    did it switch, and when did it come back?"
    *Suggestion:* add a short event log ("14:02:11 homebox stopped answering →
    vps; 15:40:03 back to homebox"), and on Android post a notification for each
    switch.

25. **medium**, android `41-home-using-vps-homebox-unreachable.png`, `42-home-back-to-homebox-in-1-20.png`.
    The Machines list shows homebox with a green "online" dot while the Servers
    section says homebox is "Unreachable". Either Tailscale on homebox is up and
    only tailbond is down, or the list is stale, and the UI doesn't say which.
    For a "home lost power" diagnosis, that matters.
    *Suggestion:* add a reason to "Unreachable" ("Tailscale says online; tailbond
    on homebox:443 not answering" vs "offline in Tailscale too"), and the last
    answered time the router already shows.

26. **medium**, glinet `failover-you-picked-vps-desktop.png`, android `44-settings-servers.png`.
    After "Use now" on vps, "When to switch" still shows "Prefer homebox". It's
    unclear whether the manual pick holds or automatic switching takes it back
    after 2 minutes. "Use now" / "Use this server now" is also offered on a
    server that is "Can't reach" / "Unreachable".
    *Suggestion:* add "(until you switch back)" or "(for now; switches back
    automatically in 2 min)" to "you picked it", with a "Back to automatic"
    button. Disable "Use now" on unreachable servers or say what it does.

27. **medium**, android `44-settings-servers.png` and glinet `failover-using-vps-home-unreachable-desktop.png`.
    Subnet routes aren't covered. Only the "Only Tailscale machines" mode
    mentions that "LANs behind other subnet routers aren't reachable". In the
    default mode, on either server, can I reach other advertised subnets on my
    tailnet (office LAN, homelab VLANs)? Does vps accept routes?
    *Suggestion:* add a "Reaches" line per server, e.g. "tailnet 100.64/10,
    subnet routes: 192.168.1.0/24 via homebox (unreachable now)".

28. **low**, android `43b-home-switching-to-vps.png`.
    The status says "Using vps (homebox is unreachable)" while vps is still
    "Connecting…", and nothing says whether traffic is held or goes direct
    during those seconds.
    *Suggestion:* use "Switching to vps…" until it's connected. See finding 9
    for the traffic note.

29. **low**, glinet `failover-using-vps-home-unreachable-desktop.png`.
    The page subtitle still says "one tunnel to your home server" while on vps.
    *Suggestion:* use "to your servers" or "to vps (backup)".

30. **low**, android `49-home-error-no-server-reachable.png`.
    In the "can't reach any server" error, the server rows lose their status
    and show neutral "Main server" / "Backup". The header still says "vps via
    backup server" even though nothing is connected.
    *Suggestion:* show the failure reason per server, e.g. "timeout at
    home.example.net:443" or "timeout at 203.0.113.9:8443".

31. **low**, android `44-settings-servers-dark.png`.
    The "Unreachable" dot is pale pink in the dark theme and hard to tell
    apart from neutral.
    *Suggestion:* use a saturated red, and add an icon or shape so color isn't
    the only signal.

## Overall count

| Severity | Count |
|---|---|
| high | 2 |
| medium | 17 |
| low | 12 |
