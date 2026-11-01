# Round 10 — Fedora persona F1 (homelab admin)

**Persona:** homelab admin. Runs Fedora servers with firewalld and SELinux enforcing, Tailscale with subnet routers and a hand-written ACL policy. Reads install guides critically and wants to know what gets exposed to whom, on which ports, and how to check it.

**Materials used:** `task-script.md`, `screens.md`, `fedora-server.md`, and the screenshots in `glinet/` and `web/`. Nothing else.

**Overall impression:** this is one of the better install guides I've read for a VPN-ish daemon. It lists every port and firewalld object the package touches, gives the `firewall-cmd` commands to check them, explains the source-address scoping for 8077 and 80, and warns about `tailscale serve` on 8077. The dashboard's setup health finds the relayed machine by name and gives the exact forward. What still bothers me is mostly about **exposure and identity**: what an approved family device can reach, a router signed in to *my* account, and the switch from owned to tagged. There is also one ordering problem in the port-forward steps.

---

## Task 1: Install tailbond on homebox and open the dashboard

**What I'd do**
- Guide §1: `sudo dnf install ./tailbond-server-*.rpm`, `sudo systemctl enable --now tailbond`, then `systemctl status tailbond`.
- Check firewalld using the guide's list: `sudo firewall-cmd --get-active-zones`, `--zone=FedoraServer --list-services` (expect `tailbond`, plus UDP 41641), and `--list-rich-rules` (expect tcp 8077/80 from `100.64.0.0/10` and `fd7a:115c:a1e0::/48`), and `sudo firewall-cmd --info-policy=tailbond-out`.
- homebox is headless, so I'd open the dashboard from my laptop on the tailnet: `http://homebox:8077`. This gives screen `glinet/15c-fedora-fresh-install-desktop`.

**What works for me**
- The "What the package changes in firewalld" list, with the commands to check it and the note that `dnf remove` takes it back out. That is exactly what I want.
- "tcp 8077 and 80 from tailnet addresses only … Tailscale drops those source addresses on any other interface, so this doesn't open the ports to your LAN." The reasoning is right, and I can check it myself.
- The warning not to point `tailscale serve` at 8077, with the reason (identity lost), shows the authors understand `whois`-based auth.

**Where I'd hesitate**
- **SELinux / hardening (low).** "SELinux needs no changes: the service runs unconfined as root, like other VPN servers." A root daemon that serves HTTP on two ports to the tailnet, with no SELinux policy and no mention of systemd sandboxing (`ProtectSystem=`, `CapabilityBoundingSet=` etc.), is the first thing I'd look at with `systemctl cat tailbond`. It doesn't stop the install, but I'd want one sentence on why it needs root, or a statement that it's hardened.
- **Firewall writes I can't opt out of (low).** "the `server-up` hook, every start, both the running and the saved configuration". It also opens "UDP 41641 (Tailscale's own port) in the same zone as UDP 443 when Tailscale is installed". If I deliberately remove 41641 from my WAN-facing zone, it comes back on the next restart. The guide documents an opt-out for the phone page (`"landing_port": -1`) but none for the 41641 opening or for firewalld management as a whole.
- **Home LAN reachable by default (low).** "bonded devices … reach the internet, your home LAN and your tailnet through it. To keep them off your home LAN, add a rule of your own". The example hard-codes `192.168.1.0/24` with `family=ipv4`. That's fine for me, but it's a hand-written rich rule, not a setting, and family devices get the whole LAN (printers, NAS admin UIs, IPMI) unless I remember to add it. I would have expected the default to be off, or a config key for it.
- **Header toggle on a server (low).** 15c, top right: "Connected" with a toggle, while "No devices yet". Connected to what? On a server I'd guess it stops mqvpn, but nothing says so, and I wouldn't click it to find out.
- **Header vs setup step (low).** The 15c header says "Home server · reachable at 192.168.1.2:443". The card below says "Right now devices would connect to 192.168.1.2:443, an address inside your home, so they only work at home." "Reachable" in the header is generous.

## Task 2: Right after installing (15c): what's left, and how?

**What I'd do:** read the orange **"Setup step: tell devices where home is"** card. It tells me to forward UDP 443 to 192.168.1.2, set a DHCP reservation, and type the public address or DDNS name, then **Save**. The setup health card shows one optional item: approving the route `10.77.0.0/24`. Below that are the approver card and the add-device cards.

The setup-step card is good. It says "The port (443) is added for you" and points at CGNAT detection, and the field accepts a DDNS name.

**Where I'd get confused**
- **"All set" right under "only work at home" (medium).** Directly below the orange card, 15c says **"Setup health: all set (1 optional item below)"**. The card above says devices "only work at home" until I forward a port and enter an address. So which is it? The health card doesn't count the missing public address or the unconfirmed UDP 443, although the guide's health table has a "UDP 443 not confirmed from outside yet (?)" row. Someone who only glances at the health card would think they're done. I'd expect the health header to count the setup step, e.g. "1 thing to do (above), 1 optional".
- **Guide says the route is required; the dashboard says it's optional (medium).** Guide §3: "**Approve the route once:** open the admin console … and enable `10.77.0.0/24`." The guide's own health table, and 15c, say "Route 10.77.0.0/24 isn't approved yet (optional)" and "a choice, not a speed problem: approving exposes bonded devices to your tailnet (under your ACLs)". For me this is an exposure decision. With my ACLs, approving the route makes family phones addressable from my tailnet. §3 tells me to do it as a setup step without mentioning that trade-off. I'd expect §3 to say "optional; approve only if you want tailnet machines to reach bonded devices", matching the dashboard.
- **The guide's test step can't work in the guide's order (medium).** Guide §2, step 3: "Test the forward from outside right away: on a phone with tailbond, turn Wi-Fi off; once it connects over mobile data, the dashboard's *UDP 443* health item turns green." Step 4 is "Tell tailbond: type that address into the setup step". Until step 4 is done, devices are told `192.168.1.2:443` (15c says so), so a phone on mobile data can't connect and the test fails. Right after install I also don't have a phone with tailbond yet. I'd swap steps 3 and 4 and say the test needs a device added (§4). Even better would be a way to verify the forward from the dashboard. As it is, there's no way to verify a UDP forward before handing devices to family.
- **Missing linked doc (low).** "For best speed … see [setup tips](setup-tips.md)". That file isn't in my packet. I also can't tell where the dashboard's "How to fix" links lead. The guide's health table covers most of it, so this is minor.

## Task 3: Approve a family member's GL.iNet travel router

**What I'd do:** on `glinet/14-fedora-dashboard-request-desktop` the request is at the top: "Waiting for your approval", "slate7 wants to connect", "Travel router on your tailnet · signed in as alex@example.com · GL-BE3600 · 100.101.9.9". I'd click **Approve**. In 14b a toast says "slate7 can connect now. It connects by itself in a few seconds." The phone layout (14-…-phone) works too, with full-width Decline/Approve buttons.

The approval card does explain what approval grants: "Approving lets the device use homebox's internet connection (by default all its traffic goes out from here) and your home network, and reach your Tailscale machines as homebox, under homebox's access rules." I like that.

**Where I'd get stuck or do something wrong**
- **A family member's router signed in as me (medium).** The request says "signed in as alex@example.com", but the task says this is a *family member's* router. The guide (Who can approve) warns that such a device can use the dashboard: "including a router or phone you lent to someone while it's signed in to your account: sign those in with the borrower's account, or tag them". The "Who can approve" card says the same: "A device you lend to someone while it's signed in to an approver's account can approve too." The request card, where I actually make the decision, shows the account as a neutral fact and doesn't warn. I'd expect a flag on the request, something like "This device is signed in to your own account, so whoever holds it can approve devices and has your tailnet access. Sign it in with their account or tag it first." It's easy to click Approve here and leave a router in a relative's house that can approve devices on my dashboard. (Its tailnet access as alex existed before tailbond, but tailbond adds dashboard rights.)
- **Little feedback after Approve (low).** In 14b the only feedback is the toast. slate7 isn't on the Connections tab yet, and the request card is gone. In 15d it finally shows under devices as "slate7 · offline" with the **same phone icon** as pixel-8 and ipad. A travel router with a phone glyph makes the list harder to scan. I'd want a router icon, and a "waiting for first connection" state rather than "offline" right after approval.

## Task 4: Get tailbond onto a family member's Android phone and connect it

**What I'd do**
- Make sure the relative is on my tailnet (invite) or share homebox with their account (guide §4, "An Android phone that runs Tailscale"). Check that my ACL lets them reach homebox on tcp 80 and 8077. The guide says so explicitly, which I appreciate.
- From the Devices tab "Add an Android phone" card (15c, `web/web-04`), copy `http://homebox.tail1234.ts.net/` with the copy button and send it to them, or tell them to type `homebox/`.
- On their phone (`web/web-00-phone-page`) they go through: Download tailbond (27 MB), the illustrated install prompts, **Open tailbond and ask homebox**, then step 3, turning tailbond on.
- On homebox (`glinet/14c`): "pixel-7a wants to connect · Phone on your tailnet · signed in as sam@example.com · Pixel 7a · 100.101.7.7". Approve. In `14d` pixel-7a shows under Connections with two active paths.

**What works for me:** the SHA-256 of the APK on the phone page, with "the same as `sha256sum /usr/share/tailbond/tailbond.apk` on homebox". I can verify it. The page also says plainly that it reaches homebox over Tailscale and can change nothing on the server. The request shows the requesting Tailscale login (sam@example.com), so I know whose phone it is.

**Where I'd get confused**
- **Approval bypasses the ACLs I wrote for that person (medium).** The card says the phone will "reach your Tailscale machines as homebox, under homebox's access rules". The guide is sharper: "**your Tailscale access rules for this server apply to every bonded device**." For an ACL user this is the important part. If my policy only lets sam reach `tag:media`, sam's phone *through tailbond* reaches everything homebox reaches. The phrase "under homebox's access rules" sounds like a limit, when it's actually an escalation for anyone with fewer rights than homebox. The only way to narrow it is to tag the server (guide §4), and that changes who can approve (task 6). There is nothing per device. On the request card I'd expect it said directly, e.g. "sam's phone will reach what homebox reaches, not what sam's own ACLs allow".
- **Inconsistent "who can open the phone page" (low).** `web-00c` says "Anyone on your Tailscale account can open it". The guide and the dashboard say anyone on the tailnet *and people you share homebox with* can open it. "Account" is the wrong word for a tailnet with multiple users.
- **Shared-in relatives and `homebox/` (low).** The card says "Then send them this link, or tell them what to type". From experience, I'd expect someone who has homebox *shared* into their own tailnet to need the full `homebox.tail1234.ts.net` name, not the short `homebox`. Neither the guide nor the card says which name to use for sharees. I can't confirm this from the packet; it's what I'd test first.

## Task 5: Read the setup health and fix it, including the relayed machine

**What I'd do:** Devices tab, `glinet/15-fedora-devices-tab-desktop`. The other tabs show the banner "Setup health: 2 things to improve (tailbond works meanwhile). Show".
- "Tailscale: 2 machines direct, 1 through a relay (slower)". It names the machine: "Through a relay: apple-tv". The fix: "forward UDP port 41641 to this machine (192.168.1.2)". It also warns "Turning on UPnP on that router also works, but lets any device on your network open ports."
- "This network makes direct Tailscale connections hard" ("hard NAT"), with "Same fix as above (one forward fixes both)".
- So I'd add one forward on my router, UDP 41641 to 192.168.1.2. The guide says the host firewall already allows it. Then I'd check with `tailscale ping apple-tv`, looking for "via <ip>:<port>" instead of "via DERP(…)", as the guide's table describes. Result: `15d`, "Setup health: everything is set up for best speed", "Tailscale: 3 machines direct", "Network is friendly to direct connections", "UDP 443 reached from the internet · pixel-8 connected from 198.51.100.20, 2 h ago".

**Assessment:** this task went smoothly. It names the relayed machine, scopes the impact correctly ("Only traffic between this machine and that machine is slower … bonding itself doesn't go through Tailscale"), says the cause might be on the other machine's network, gives one fix for both items, and points to a CLI check. No medium or high issues.

- **Polish (low):** the 15d health card shows "Public internet address 203.0.113.7" while the header says "reachable at home.example.net:443". It's fine (the DDNS name resolved), but saying "home.example.net → 203.0.113.7" would show the DDNS name is the thing being checked.

## Task 6: Decide and set who may approve new devices

**My decision:** I want to limit what bonded devices reach, so I'll follow the guide: tag the server (`tag:tailbond`) and grant the tag only `tag:nas`. A tagged server has no owner, so I must name approvers. I'd put `alex@example.com` (and my partner's login) in **Devices → Who can approve and add devices → "Also allowed to approve (Tailscale logins, comma separated)" → Save**. I would not use the password, because per-login identity is better than a shared secret.

**Where I'd get confused or do something wrong**
- **"Also allowed" is wrong once the server is tagged, and the guide's ordering leaves a gap (medium).** On an owned server (15, 15d) the field is "Also allowed to approve", i.e. in addition to alex. The guide says that on a tagged server "the approvers are the only ones". `web-04` shows a tagged server, and the field there is *still* labelled "Also allowed to approve". After tagging I could easily enter only my partner's login, thinking I'm "also" covered, and lose approve rights everywhere except `http://127.0.0.1:8077` on the box. The guide also says to tag and "name its approvers (or set a password) in the same step". Its own grant example gives `autogroup:member` tcp:8077 to the tagged server, so between saving the ACL in the admin console and saving approvers in the dashboard, anyone on my tailnet can use the dashboard. I'd expect: (a) the guide says "add your own login as an approver *first*, then tag"; (b) on a tagged server the field is labelled "Allowed to approve (only these)".
- **ACL snippet only limits if you remove allow-all (low).** "To limit it, tag the server and give the tag only what bonded devices should reach" shows a `grants` block. If the policy still has the default `{"action":"accept","src":["*"],"dst":["*:*"]}`, adding this grant limits nothing. I know that; many readers won't. One sentence would fix it.
- **The card points at itself (low).** On `web-04` (tagged), inside the "Who can approve and add devices" card: "Name approvers or set a password in the "Who can approve and add devices" card on the Devices tab." I'm already in that card.
- **Health footer points the wrong way (low).** On `web-04` the health card lists "Anyone on your Tailscale network can approve devices", then says "Fixes are on your home router or in the Tailscale admin console". That fix is on this page.

## General polish

- **(low)** The data stat reads "DATA 7.5 GB to · 858 MB from devices". "7.5 GB to" with nothing after it looks cut off. "7.5 GB to devices · 858 MB from devices" or two labelled values would read better.
- **(low)** Tailscale tab (`15b`): a numbered list with a single item, "1. All set: bonded devices reach your tailnet through this server.". The TLS certificate value also wraps mid-hostname: "Let's Encrypt via Tailscale (home / box.tail1234.ts.net)".

---

## Findings

| # | task | finding | severity |
|---|---|---|---|
| 1 | 2 | 15c "Setup health: all set (1 optional item below)" sits directly under a setup step saying devices "only work at home"; the health card doesn't count the missing public address or the unconfirmed UDP 443 | medium |
| 2 | 2 | Guide §3 says "**Approve the route once**" as a setup step; the dashboard and the guide's health table call it "(optional)" and an exposure choice. Conflicting advice on an ACL/exposure decision | medium |
| 3 | 2 | Guide §2 step 3 ("Test the forward … on a phone with tailbond, turn Wi-Fi off") comes before step 4 ("Tell tailbond: type that address"). Devices still dial 192.168.1.2:443, so the test can't pass in that order, and no device exists yet | medium |
| 4 | 3 | The family router's request says "signed in as alex@example.com" with no warning on the request card that whoever holds it can approve devices / use the dashboard (only mentioned in the separate approver card and the guide) | medium |
| 5 | 4 | "reach your Tailscale machines as homebox, under homebox's access rules" reads like a limit but is an escalation: a family phone gets homebox's tailnet reach, not its owner's ACLs. There's no per-device limit; narrowing it means tagging the server | medium |
| 6 | 6 | On a tagged server the field is still "Also allowed to approve" (web-04), although approvers are then "the only ones". The guide's "in the same step" ordering leaves a window where anyone with tcp 8077 can use the dashboard. Easy to lock yourself out or leave it open | medium |
| 7 | 1 | "the service runs unconfined as root": network-facing root daemon, no SELinux policy and no systemd hardening mentioned | low |
| 8 | 1 | Hook rewrites permanent firewalld config "every start", incl. opening UDP 41641; no opt-out documented (only `landing_port: -1` for the phone page) | low |
| 9 | 1 | Bonded devices reach the home LAN by default; opt-out is a hand-written IPv4 rich rule with a hard-coded subnet, not a setting | low |
| 10 | 1 | Server header has a "Connected" toggle with no explanation of what turning it off does | low |
| 11 | 1 | 15c header "reachable at 192.168.1.2:443" while the card says devices "only work at home" | low |
| 12 | 2 | Guide links `setup-tips.md`, which isn't available; can't tell where "How to fix" leads | low |
| 13 | 3 | After Approve, only a toast; slate7 later appears as "offline" with a phone icon, which doesn't distinguish a router or a "never connected yet" state | low |
| 14 | 4 | Phone page says "Anyone on your Tailscale account can open it"; guide/dashboard say tailnet members and people you share homebox with | low |
| 15 | 4 | Doesn't say which name shared-in relatives should use (short `homebox/` vs the full `homebox.tail1234.ts.net`) | low |
| 16 | 5 | 15d shows "Public internet address 203.0.113.7" without tying it to the configured `home.example.net` | low |
| 17 | 6 | ACL `grants` example only limits anything if the default allow-all rule is removed; the guide doesn't say so | low |
| 18 | 6 | Tagged-server approver card tells you to go to "the "Who can approve and add devices" card", which is the card you're in | low |
| 19 | 6 | Tagged-server health footer says "Fixes are on your home router or in the Tailscale admin console"; the approver fix is on this page | low |
| 20 | all | "DATA 7.5 GB to · 858 MB from devices" reads as truncated | low |
| 21 | all | Tailscale tab: single-item numbered list "1. All set…"; TLS cert value wraps mid-hostname ("home / box.tail1234.ts.net") | low |

Totals: 0 high / 6 medium / 15 low
