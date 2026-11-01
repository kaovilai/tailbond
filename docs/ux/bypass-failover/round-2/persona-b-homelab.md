# Round 2: persona B, "homelab"

Fedora home server + rented VPS, uses Tailscale daily, knows routing, subnets,
nftables and DNS. Wants to know exactly what goes where, how failover
decides, and what happens to subnets and the kill switch.

Screens used: every PNG in `out/ux/bypass-failover/glinet/` and
`out/ux/bypass-failover/android/` (the file names below leave off the
`-desktop`/`-phone` suffix and `.png` unless the variant matters).

---

## Task 1: make `mybank.com` skip the tunnel (router, then phone)

**Done?** Yes on both. Router: `bypass-empty`, `bypass-add-website-presets`,
`bypass-rules`, `bypass-website-rules-wait`. Phone: `46c-websites-none-yet`,
`47-add-website-dialog`, `47b-add-website-preset-banking`,
`46-websites-skip-tailbond`.

The flow is quick. I still couldn't tell which uplink the bank traffic uses,
which names it covers, or how the matching works.

### Findings

1. **High**: `bypass-website-rules-wait-desktop.png`. The SSH fallback runs
   `opkg remove dnsmasq && opkg install /tmp/dnsmasq-full*.ipk`. On
   OpenWrt the router resolves names through its own dnsmasq
   (127.0.0.1). Once dnsmasq is removed, `opkg install` can't fetch any
   missing dependencies (libnettle, libnetfilter-conntrack, ...) from the
   feed. The install then fails and leaves the router with no DNS and no
   DHCP for the LAN, on a hotel network where you probably have no cable
   or console to recover. **Suggestion:** install the dependencies first,
   while dnsmasq still runs (e.g. `opkg install --download-only` the full
   dependency set, or install dnsmasq-full's dependencies explicitly
   before removing dnsmasq). Add a one-line recovery command
   (`opkg install dnsmasq`). Say that `/etc/config/dhcp` may come back as
   `dhcp-opkg`.
2. **Medium**: `bypass-website-rules-wait-desktop.png`. "Meanwhile these
   websites keep going through tailbond as before; nothing breaks." For
   `mybank.com` that's false. The whole reason for the rule is that the
   bank blocks the tunnel, so the bank stays broken until dnsmasq-full is
   in. **Suggestion:** say "until then these websites still go through
   tailbond, so a bank that blocks VPNs will still block you; use a
   device rule or an address rule meanwhile".
3. **Medium**: `bypass-add-website-presets-desktop.png`, `bypass-rules-desktop.png`.
   "Use the internet directly" = "the router's main internet connection".
   Repeater and Tethering are both set to "Always", so which one is
   "main"? It matters: Fire-TV-Stick has 212 connections on a "direct"
   rule, and Tethering has a 5 GB cap. Nothing says what happens to
   bypassed traffic if that uplink drops, either. **Suggestion:** name the
   uplink in the hint and in each rule row ("direct via Repeater"). Say
   whether bypassed traffic fails over to Tethering. Show the "only one
   connection" option next to it.
4. **Medium**: `bypass-add-website-presets-desktop.png` vs
   `47-add-website-dialog.png` / `46-websites-skip-tailbond.png`. The same
   rule covers different names on each platform. The router says
   `mybank.com` "includes all its pages and parts, like www.mybank.com and
   login.mybank.com". The phone covers only "the name and its www. name;
   add login.mybank.com separately". If I set up the router first, I'll
   assume the phone covers `login.` too, and the bank login will still go
   through the tunnel and get blocked. **Suggestion:** make both platforms
   cover the same names. If they can't, label each row with what it
   actually covers ("*.mybank.com" vs "mybank.com, www.mybank.com").
5. **Medium**: `bypass-rules-desktop.png`, `bypass-add-website-presets-desktop.png`.
   The router doesn't explain how website matching works. The phone
   (`46`) does: matching is by the addresses the names resolve to, so
   other sites on the same CDN IPs skip too, and lookups get refreshed.
   The router only gives the secure-DNS caveat. Its "dnsmasq" notice
   implies DNS-learned IP sets, so the same side effects apply, plus
   misses for clients with cached DNS answers from before the rule
   existed. **Suggestion:** use the phone's explanation on the router
   too. On each website row, show "N addresses learned, last updated
   HH:MM".

## Task 2: the work laptop always uses tailbond

**Done?** Partly. I can create the rule (`bypass-add-device-picker`
desktop + phone, `bypass-rules`), and the text says device rules beat
website and port rules and that it "waits rather than going around it".
The screens don't let me check "never leaves through the hotel".

### Findings

6. **Medium**: `bypass-add-device-picker-desktop.png`. "If tailbond is
   down, it waits." Nothing says what happens when I flip the main
   tailbond toggle off, press "Forget all servers"/"Unpair", or during the
   window after a reboot before tailbond starts. For a "never" guarantee
   I need each of those spelled out. The rule is also keyed on a MAC. If
   the laptop comes back with a randomized MAC, the rule stops matching
   silently: the row says "not connected" while the laptop is online
   under a new address and gets the website/port bypasses.
   **Suggestion:** state the behaviour for toggle-off, unpair and boot. Show
   "blocked N times while tailbond was down" on the row. Warn when a
   device with the rule's hostname shows up under a different MAC.
7. **Medium**: whole router page (`bypass-rules-desktop.png`,
   `servers-home-active-vps-standby-desktop.png`). No screen says where
   LAN clients' DNS goes (through the tunnel to homebox/vps, or the hotel
   resolver), or what happens to IPv6. If the hotel hands out v6 and
   that isn't tunnelled or blocked, "always uses tailbond" is not true.
   **Suggestion:** add a short "What goes where" block: DNS path for
   normal, bypassed and always-tailbond devices, and the IPv6 policy
   (tunnelled / blocked / not supported).
8. **Low**: `bypass-add-wifi-calling-desktop.png`, `bypass-rules-desktop.png`.
   The Wi-Fi calling rule is "UDP 4500 and 500" for every device. That's
   all IKE/IPsec NAT-T, not just carrier Wi-Fi calling: any other guest
   device's IPsec VPN goes out direct too. It doesn't say whether these
   are destination ports. **Suggestion:** say "destination ports, any
   device; this also covers IPsec VPNs" and offer to limit it to chosen
   devices.
9. **Low**: `bypass-wifi-calling-added-desktop.png`. The confirmation toast
   sits on top of the homebox "Connects to" line, so it covers the
   details I was reading. **Suggestion:** anchor the toast to the bypass
   card or the page corner.
10. **Low**: `servers-home-active-vps-standby-desktop.png` vs the bypass
    card. "While it switches, devices here wait a few seconds; nothing
    goes out unprotected" sits right above "Rules that skip tailbond keep
    working while it's down." Bypassed traffic does go out unprotected,
    because I chose that. **Suggestion:** "nothing that normally uses
    tailbond goes out another way".

## Task 3: add the VPS as a backup to `homebox`

**Done?** Yes on both, including the clash. Router: `servers-single-home`,
`add-server-open`, `add-server-waiting-approval`, `add-server-subnet-clash`,
`add-server-paste-clash`, `servers-home-active-vps-standby`. Phone:
`44b-settings-one-server-add-backup`, `45-add-another-server`,
`45b-add-another-server-confirm`, `45c-add-another-server-same-addresses`,
`44-settings-servers`.

### Findings

11. **Medium**: `add-server-subnet-clash-desktop.png` vs
    `45c-add-another-server-same-addresses.png`. The phone says "Asking
    over Tailscale does this by itself while vps has no other devices".
    But on the router I used "Ask to add" over the tailnet, vps approved
    it, and only then did I get the clash. Either the router doesn't do
    the automatic re-pick, or the phone's promise is wrong. The clash
    also shows up only after someone has approved on vps, so that
    approval is wasted. **Suggestion:** check the tunnel range before
    asking (the router can see vps's range when it finds it on the
    tailnet). Make the automatic re-pick behave the same on router and
    phone, or drop the claim.
12. **Medium**: `add-server-subnet-clash-desktop.png`, `45c-...png`. The
    suggested fix "10.78.0.0/24" is a fixed string. Nothing checks it
    against the router LAN (192.168.8.0/24 here), the hotel's subnet, or
    subnet routes already advertised on my tailnet. The phone also gives
    only the `tailbond.json` method, while the router gives three
    (installer flag, uci, json). **Suggestion:** suggest a range that has
    been checked against local and advertised routes, and show the same
    three methods on both.
13. **Medium**: `servers-home-active-vps-standby-desktop.png`. "For your
    Tailscale machines to reach devices here through each server, approve
    each server's route in Tailscale." Which prefix? If homebox and vps
    both advertise this router's LAN, Tailscale's HA subnet routing
    picks a primary on its own. While the router is on vps but homebox is
    back online (the 2-minute hold), return traffic from the tailnet can
    go to homebox, which no longer has the tunnel. **Suggestion:** show
    the exact CIDR each server advertises. Say whether only the active
    server advertises it, or how overlap is handled.
14. **Low**: `servers-home-active-vps-standby-desktop.png` vs
    `44-settings-servers.png`. The router shows homebox as
    `home.example.net` (UDP 443 elsewhere) but vps only as `203.0.113.9`,
    with no port or protocol. The phone shows `:8443`, also with no
    protocol. I need both to open the VPS firewall. **Suggestion:** show
    `host:port/udp` for every server on both platforms.

## Task 4: notice the switch to vps, and the return

**Done?** Partly. That it switched and that it came back: yes, on both
(`failover-using-vps-home-unreachable` desktop + phone, `failback-countdown`,
`failover-you-picked-vps`, `back-on-homebox-since`; phone `41`, `41b`, `41c`,
`42`, `43`, `43b`, `49`, `44`, `44c`). Why it switched: only "unreachable".

### Findings

15. **High**: `43b-home-switching-to-vps.png`, `49-home-error-no-server-reachable.png`,
    `44-settings-servers.png` / `44b-...png`. On the phone, every switch
    sends all apps over the phone's own connection "for these few
    seconds", and if both servers are down tailbond stops and the phone
    "is using its normal internet now". That's a fail-open on hotel
    Wi-Fi, DNS included. The only protection is Android's "Block
    connections without VPN", and Settings tells me to leave it off "if
    you want to switch back to Tailscale easily". The router promises
    "nothing goes out unprotected", so the phone behaving the opposite
    way is a trap. **Suggestion:** keep the VPN interface up and hold or
    drop traffic during the switch, like the router does. When all
    servers are down, stay up and block by default, with an explicit
    "use the phone's own internet" button. In Settings, say plainly that
    without "Block connections without VPN" traffic leaks during switches
    and outages.
16. **Medium**: `failover-using-vps-home-unreachable-desktop.png`, `41-...png`.
    The only reason given is "homebox is unreachable". I can't tell
    whether `home.example.net` stopped resolving, no handshake came back
    over either uplink, or the hotel started blocking UDP 443. Each one
    needs a different fix. **Suggestion:** add the last error and per-uplink
    result to the homebox row ("no reply over Repeater or Tethering since
    14:11:45; DNS OK, resolves to x.x.x.x").
17. **Medium**: `back-on-homebox-since-desktop.png`, `43-home-back-on-homebox.png`.
    After the fact there's only "Back on homebox since 14:11". No history
    of when it left, how long it was on vps, or why. If I wasn't watching,
    I can't reconstruct the outage. **Suggestion:** add a short event list
    on both platforms (time, from → to, reason, duration).
18. **Medium**: `failover-using-vps-home-unreachable-desktop.png`, `servers-home-active-vps-standby-desktop.png`.
    The router never says what changes for my LAN devices when it moves
    to vps. The phone says it all: internet traffic now exits from the
    VPS (datacenter IP, may look like a VPN to some websites), open
    connections start over, and tailnet ACLs now treat these devices as
    vps (so a nas ACL that allows homebox but not vps will refuse me).
    **Suggestion:** use the phone's sentences in the router's Servers card
    and in the failover banner.
19. **Medium**: `servers-home-active-vps-standby-desktop.png`. The router
    says "switches when homebox hasn't worked for 15 s", without the
    phone's definition ("tunnel worked over none of your connections"
    and "checks outside the tunnel"). Neither says what it does *not*
    detect: homebox up but its home ISP down, or tailscaled down on
    homebox. Then the tunnel works, tailnet names and internet fail, and
    nothing fails over. **Suggestion:** use the phone's definition on the
    router and add one line on what doesn't trigger a switch.
20. **Medium**: `failover-using-vps-home-unreachable-desktop.png` / `-phone.png`.
    Home has lost power, yet the router's Tailnet machines list still shows
    `nas` and `apple-tv` as "Online · direct". The phone has the caveat
    "Tailscale may still show it online"; the router has nothing, so I'll
    go try the nas and wonder why it times out. **Suggestion:** add the same
    caveat, or flag machines that sit behind the unreachable server.
21. **Low**: `failover-you-picked-vps-desktop.png`. With a manual "Use now"
    pin in place, "When to switch" still shows "Prefer homebox, use vps if
    homebox is down" selected. Nothing explains how a temporary pin
    differs from "Always use vps", and from `failback-countdown` it's
    unclear whether "Stay on vps" lasts. **Suggestion:** show the pin as
    its own state above the radio buttons ("Pinned to vps until you go
    back"), and say what "Stay on vps" lasts for.
22. **Low**: `failover-using-vps-home-unreachable-desktop.png` ("since
    14:12") vs `back-on-homebox-since-desktop.png` ("since 14:11"); the
    phone has 14:02 → 14:20. On the router, the return happens before
    the switch. **Suggestion:** make the fixture timeline consistent.
    Reviewers will distrust a timeline that runs backwards.
23. **Low**: `servers-home-active-vps-standby-desktop.png` (router
    10.77.0.2 on homebox, 10.83.142.2 on vps) vs `40-...png` / `41-...png`
    (phone pixel-8 also 10.77.0.2 / 10.83.142.2). Same tunnel address
    for two devices on the same server looks like an address collision.
    **Suggestion:** give the phone a different host address in the
    fixtures. If it's real, explain it.
24. **Low**: `40`, `41`, `42`, `43`, `49`. "Switch back to Tailscale" is a
    big button on every main screen, right next to "Go back to homebox",
    "Stay on vps" and "Back on homebox". It's the only "switch" that turns
    tailbond off. **Suggestion:** make it less prominent or rename it
    ("Turn off tailbond and use Tailscale").
25. **Low**: `47d-add-website-preset-wifi-calling.png`. "Wi-Fi calling" is a
    choice in the Add a website dialog that adds nothing. **Suggestion:**
    move the explanation to the page's help text and drop the chip here.
26. **Low**: `44-settings-servers-dark.png`. The "Unreachable" dot for
    homebox is pale pink in dark mode and barely reads as an error next
    to the green "In use". **Suggestion:** use a saturated error colour,
    or an icon as well as the colour.

---

## Overall count

| Severity | Count |
|---|---|
| High | 2 |
| Medium | 14 |
| Low | 10 |
