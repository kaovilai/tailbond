# Round 11 — Fedora persona F1 (homelab admin)

**Persona.** I run Fedora servers with firewalld and SELinux enforcing, and Tailscale with subnet routers and a hand-written ACL policy. I read install guides critically. What I care about most is what gets exposed to whom, on which ports, and how I can check it myself.

**Materials used.** Only the packet: `task-script.md`, `screens.md`, `fedora-server.md` and the PNGs in `glinet/` and `web/`.

**Overall.** I could finish all six tasks. The guide is unusually honest about what the package does to firewalld and what an approved device reaches, and I liked several things: the `firewall-cmd` check commands, the SHA-256 on the phone page, the `tailscale ping` "via DERP" hint, and the 15d health card that shows *how* UDP 443 was confirmed ("pixel-8 connected from 198.51.100.20, 2 h ago"). What's left is mostly about exposure: a route that's advertised for me, a sharing option that sounds more limited than it is, and the "anyone logged in on homebox" rule, which in practice covers every local process. Nothing stopped me outright.

---

## Task 1: Install and open the dashboard

**Where I click / type.** I follow guide §1: `sudo dnf install ./tailbond-server-*.rpm`, then `sudo systemctl enable --now tailbond`, then `systemctl status tailbond`. homebox is headless, so I open `http://homebox:8077` from my laptop on the tailnet, as the manifest and §1 say ("the agent also listens on the server's Tailscale IPs").

Then I check what changed, using the guide's own commands:
- `sudo firewall-cmd --get-active-zones`
- `--list-services` (looking for UDP 443 in the WAN-facing zone)
- `--list-rich-rules` (tcp 8077/80 from `100.64.0.0/10` / `fd7a:115c:a1e0::/48` only)

That list in §1 ("There's nothing to do here yourself; this list is for checking") is exactly what I want.

**Where I hesitate.**
- **SELinux.** "SELinux needs no changes: the service runs unconfined as root, like other VPN servers." That doesn't hold up for me. OpenVPN has its own confined domain, and WireGuard is in the kernel. A root daemon that serves a web UI and runs unconfined is something I'd want to know is temporary. It's disclosed, so this is **low**.
- **firewalld ownership.** The hook rewrites "both the running and the saved configuration" on every start. The uninstall section says "`dnf remove` closes UDP 443 and 41641". I may well have opened 41641 myself before installing tailbond, for my own Tailscale. The guide doesn't say whether remove only takes out what tailbond added, or kills my rule too. **Low.**
- **No listening-socket inventory.** I learn about the listeners piecemeal:
  - 127.0.0.1:8077
  - Tailscale IPs :8077 and :80
  - 10.77.0.1, where "the agent answers them there only with the machine list, never the dashboard". Which port, and what is "the machine list"?
  - DNS on 10.77.0.1
  - UDP 443

  I'd like one table I can compare against `ss -tulpn`. **Low.**
- **Package provenance.** The phone page gives the APK's SHA-256 and `sha256sum /usr/share/tailbond/tailbond.apk`. The RPM I'm installing as root gets no checksum and no signature/GPG note. **Low.**

---

## Task 2: Right after installing (15c): what's left?

**Where I click.** The orange card "Setup step: tell devices where home is" says what's left:
1. "Forward UDP port 443 to this machine (192.168.1.2) … Give this machine a fixed address there too (DHCP reservation)".
2. Type the public address or DDNS name into the field, then **Save**.

Below it, the health card says "nothing to fix here (1 optional item below) · finish the setup step above first", with "Route 10.77.0.0/24 isn't approved yet (optional)". That's clear. It matches guide §2 steps 1–3.

**Where I get stuck / misread.**
- **The route is advertised for me, with no way to say no (medium).** The guide states, almost in passing, "The `server-up` hook runs `tailscale set --advertise-routes=10.77.0.0/24`". 15c only says the route "isn't approved yet (optional)". Neither the card nor the guide tells me:
  - that tailbond is already advertising it;
  - how to stop it from doing that (no config key is given).

  On my tailnet, `autoApprovers` covers my RFC1918 ranges for the subnet routers. If homebox's owner matches, 10.77.0.0/24 would be approved without me ever deciding, and that exposes bonded devices to my tailnet. I expected an opt-in, or at least an opt-out like `"advertise_route": false`, and wording like "tailbond advertises 10.77.0.0/24; approve it only if…".
- **The header contradicts the setup step (low).** The header says "Home server · reachable at 192.168.1.2:443". Directly below, the setup step says devices "only work at home". "reachable at" a private address reads like a success state. I'd expect something like "only reachable inside your home until you set a public address". The green "Connected" toggle in the header is also unclear for a server with zero devices: what does turning it off do?
- **No way to check the forward from the server side (low).** Guide §2.4 and the health table say "none: the first device connecting from away confirms it". I can't test my UDP 443 forward until a real device goes to mobile data. A self-test, or a suggested `nc -u` from a VPS, would help.

---

## Task 3: Approve the family member's GL.iNet router (14, 14b)

**Where I click.** The top card "Waiting for your approval" shows "slate7 wants to connect · Travel router on your tailnet · signed in as alex@example.com · GL-BE3600 · 100.101.9.9", with **Approve** and **Decline**. The phone view (14-phone) has the same card with full-width buttons. After I click Approve, 14b shows the toast "slate7 can connect now. It connects by itself in a few seconds."

The general approval text is good, and it's exactly the warning I'd want: "reach your Tailscale machines as homebox: they allow it whatever they allow homebox, which may be more than its owner's own account gets."

**Where I hesitate.**
- **The router is on my account (medium).** The orange warning reads: "It's signed in to alex@example.com, an account that may approve here: whoever holds this router can also open this dashboard and approve devices. If it's for someone else, sign it in to their own Tailscale account first, then let it ask again."

  This router is for family, so as a careful admin I would **not** click Approve. The router is on my account, which means it already has my full ACL rights and could approve devices here. But the fix isn't shown anywhere I can see:
  - no screen of re-signing the GL.iNet's Tailscale;
  - nothing on screen about the alternative the guide gives in "Who can approve" ("sign those in with the borrower's account, **or tag them**"), which is what I'd actually do (`tag:travel`);
  - no hint whether **Decline** blocks the device or just dismisses the request, which matters because "then let it ask again" implies asking again works.

  The task said "approve it". Following the on-screen advice means extra steps outside what I have.
- **I can't confirm the router connected (low).** After approval, 14b's Connections tab shows only pixel-8, which is fine "in a few seconds". The later screen 15d still lists "slate7 · not connected right now (approved devices connect by themselves when they're on and online)". No screen shows slate7 actually connected, and there's no "last seen" or last error I could use to tell "router off" from "router failing to reach 443".

---

## Task 4: Get tailbond onto a family member's Android phone (Devices card, web-00, web-00c, 14c, 14d)

**Where I click.**
1. Devices → **Add an Android phone** (15c/15/web-04). I copy the link `http://homebox.tail1234.ts.net/` and send it to the family member. Or they type `homebox/`.
2. They open web-00 on the phone: install steps with drawings, then **Open tailbond and ask homebox**, then turn tailbond on.
3. On my dashboard, 14c shows "pixel-7a wants to connect · Phone on your tailnet · signed in as sam@example.com · Pixel 7a · 100.101.7.7". I click **Approve**.
4. 14d shows pixel-7a under Connections from 198.51.100.44 with 2 active paths. The task is clearly done.

The phone page's "Is this safe?" paragraph and the SHA-256 are good. The note "the pictures below are drawings, not buttons" is a nice touch.

**Where I get stuck / misread.**
- **"Share just this machine" sounds more limited than it is (medium).** The card offers two ways to get the family member onto the tailnet: invite them as users, "or share just this machine with their own account (Machines → this machine → Share; they then see only this machine …)".

  As a Tailscale admin, I'd pick sharing *precisely* because it limits them to one machine. But after approval, the approval card says the phone "reach[es] your Tailscale machines as homebox", and the phone page (web-00, step 3) says "Your Tailscale machines stay reachable through homebox. They see this phone as homebox". So with tailbond on, a shared-in user reaches whatever homebox reaches. "they then see only this machine" is true for Tailscale alone and misleading here. It's the sentence right where I choose the method.

  I expected: "sharing limits what Tailscale itself lets them reach; once approved here, their phone reaches what homebox reaches".
- **Custom ACLs need their own example (low).** The guide says the phone must be "allowed to reach this machine on tcp 80 and 8077 (Tailscale's default policy allows it)". My policy isn't the default. The only grant example assumes a *tagged* server (`autogroup:member → tag:tailbond`). For my untagged homebox I have to write the rule myself, and the guide gives no example for that case.
- **Path cards (low).**
  - Connections shows "Path 0 · Pixel-7a · Min 28 Ms" and "Path 1". I can't tell which path is Wi-Fi and which is 5G, which is the whole point of bonding.
  - "Ms" should be "ms".
  - The pixel-8 device row ends with an unexplained "· wlb".

---

## Task 5: Read setup health and fix it, including the relay (15, 15b, 15d)

**Where I click.** Devices tab (15), card "Setup health: 2 things to improve (tailbond works meanwhile)":
- "Tailscale: 2 machines direct, 1 through a relay (slower). Through a relay: apple-tv. … On your home router's admin page (Port forwarding …), forward UDP port 41641 to this machine (192.168.1.2)."
- "This network makes direct Tailscale connections hard … Same fix as above (one forward fixes both)."

So I add a UDP 41641 forward to 192.168.1.2 on my home router. The firewall side is already done (guide §1: "UDP 41641 … in the same zone as UDP 443"). I check with `tailscale ping apple-tv`, looking for "via <ip>:<port>" (guide health table). 15d then shows "everything is set up for best speed", with "Tailscale: 3 machines direct" and "Network is friendly to direct connections". The Tailscale tab (15b) shows "Tunnel route 10.77.0.0/24 · approved" and "All set". That's a good, verifiable loop.

**Where I hesitate.**
- **41641 conflicts with my subnet router (low).** I already have a Tailscale subnet router on this LAN, and it may already own the 41641 forward. Only one LAN host can get UDP 41641. Neither the card nor the guide mentions moving homebox's tailscaled to another port (e.g. `--port` in `/etc/default/tailscaled`) and forwarding that instead.
- **No check on screen (low).** The `tailscale ping` verification is only in the guide, not on the card. The card has a "How to fix" link and "The cause can also be on that machine's own network", but no CLI check.
- **Arrows flip between views (low).** The header shows a blue ↓ with "1.1 MBPS FROM DEVICES" and a purple ↑ with "12.0 MBPS TO DEVICES". The pixel-8 row on the same page shows "↓ 12.0 Mbps ↑ 1.1 Mbps". Same numbers, opposite arrows. I paused on that while reading the page.

---

## Task 6: Decide and set who may approve (15c / 15 / 15d, web-04)

**Where I click.** Devices → **Who can approve and add devices**:
- "Also allowed to approve (Tailscale logins, comma separated)" + **Save**
- "Or a password for this page (replaces the above; empty: none)" + **Save**

The current rule is spelled out: "alex@example.com (the Tailscale account homebox belongs to), from any device signed in to that Tailscale account, and anyone logged in on homebox itself (http://127.0.0.1:8077)".

**My decision.** No extra approvers. I'd set a password only if I accept losing the identity check (see below). I'd also take my own devices that I lend out (slate7) off my account, or tag them.

**Where I get stuck / misread.**
- **"Anyone logged in on homebox" really means any local process (medium).** "anyone logged in on homebox itself (http://127.0.0.1:8077)" sounds like interactive users. Reading guide §4, though, the no-password `curl … http://127.0.0.1:8077/api/clients` call creates a device and returns its `tailbond://pair` link with no authentication at all.

  homebox runs other services: host-network containers, a web app with possible SSRF. Any local process that can make an HTTP POST to localhost can mint a pair link. That device then gets my home LAN and my tailnet "as homebox". I expected this to be called out as "any program on this machine", or the localhost exemption to be something I can switch off independently.
- **Password replaces identity instead of adding to it (low).** The only mitigation is the password, and it "replaces the rules above". I can't require *both* my Tailscale identity *and* a password. Also:
  - the field doesn't show whether a password is currently set;
  - there's no confirm field;
  - no screen shows what happens after Save (a login prompt?);
  - the guide's `"ui_password"` in `tailbond.json` suggests it's stored in plaintext.
- **Open approval on a tagged server reads like a minor item (medium).** The guide recommends tagging the server to limit what bonded devices reach. web-04 shows what a tagged server without approvers looks like: "Setup health: 1 thing to improve (tailbond works meanwhile) (and 1 optional)" with "! Anyone on your Tailscale network can approve devices".

  It uses the same orange "!" as the optional route item and the same reassuring "tailbond works meanwhile". The card's boilerplate also says "Fixes are on your home router or in the Tailscale admin console", but this fix is on the dashboard itself. Anyone on the tailnet being able to mint devices with homebox's access is a security exposure. I expected red, and a heading that says so.

---

## Findings

| # | Task | Finding | Severity |
|---|---|---|---|
| 1 | 1 | "SELinux needs no changes: the service runs unconfined as root, like other VPN servers": a root daemon with a web UI runs unconfined, and the comparison doesn't hold (OpenVPN is confined, WireGuard is in the kernel) | low |
| 2 | 1 | The hook rewrites saved firewalld config on every start, and "`dnf remove` closes UDP 443 and 41641" doesn't say whether it also removes a 41641 opening I made myself | low |
| 3 | 1 | No single list of listeners (8077, 80, 10.77.0.1 "machine list", DNS, UDP 443) to check with `ss -tulpn`; "the machine list" is unexplained | low |
| 4 | 1 | The APK gets a SHA-256 on the phone page; the root-installed RPM gets no checksum or signature note | low |
| 5 | 2 | The hook advertises 10.77.0.0/24 by itself; 15c only says "isn't approved yet (optional)"; no opt-out, and existing `autoApprovers` could approve it without me deciding | medium |
| 6 | 2 | Header "reachable at 192.168.1.2:443" reads as success while the setup step says devices "only work at home"; the "Connected" toggle on a server is unexplained | low |
| 7 | 2 | The UDP 443 forward can only be confirmed by a real device away from home ("none: the first device connecting from away confirms it"); no self-test | low |
| 8 | 3 | slate7 "signed in as alex@example.com" for a family member: the warning says re-sign it, but the on-screen text doesn't mention tagging and doesn't say whether Decline blocks the device | medium |
| 9 | 3 | After "It connects by itself in a few seconds", no screen shows slate7 connected; 15d says "not connected right now" with no last-seen or error | low |
| 10 | 4 | "share just this machine … they then see only this machine" is misleading: once approved, the phone reaches the tailnet "as homebox" | medium |
| 11 | 4 | "Tailscale's default policy allows it"; for custom ACLs there's only a tagged-server grant example, none for an untagged homebox | low |
| 12 | 4 | "Path 0 / Path 1 · Min 28 Ms": can't tell Wi-Fi from 5G; "Ms" typo; unexplained "· wlb" on the device row | low |
| 13 | 5 | One UDP 41641 forward per LAN; no advice for when a subnet router already uses it (another tailscaled port) | low |
| 14 | 5 | The `tailscale ping` check is only in the guide, not on the health card | low |
| 15 | 5 | Header ↓ = "1.1 from devices", but the device row shows "↓ 12.0 Mbps ↑ 1.1 Mbps" (arrows flip) | low |
| 16 | 6 | "anyone logged in on homebox itself (http://127.0.0.1:8077)" really means any local process can POST /api/clients unauthenticated and mint a pair link with LAN and tailnet access | medium |
| 17 | 6 | Password "replaces" identity instead of adding to it; no "is set" indicator or confirm field; `ui_password` likely stored in plaintext in the JSON | low |
| 18 | 6 | Tagged server with open approval shows as "1 thing to improve (tailbond works meanwhile)", with the same orange "!" as the optional route and the boilerplate "Fixes are on your home router or in the Tailscale admin console" | medium |

Totals: 0 high / 5 medium / 13 low
