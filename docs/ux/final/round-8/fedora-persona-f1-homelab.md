# Round 8 — Fedora persona F1 (homelab admin)

Persona: I run Fedora servers with firewalld and SELinux enforcing, and Tailscale with subnet routers and a hand-written ACL/grants policy. I read install guides for what gets opened, to whom, on which ports, and how I can check it myself. Materials used: `task-script.md`, `screens.md`, `fedora-server.md` and the PNGs in `glinet/` and `web/`. Nothing else.

Overall: this is one of the more honest install guides I've read for a VPN-ish package. It lists every firewalld change, the source ranges and zones, gives me `firewall-cmd` commands to check them, and says plainly that bonded devices inherit the server's tailnet access. The main problem is a contradiction about who can approve devices. Two cards on the same screen disagree, and the decision I'd make in Task 6 depends on which one is true.

---

## Task 1 — Install and open the dashboard

**What I'd do.** I'd follow guide §1 and run `sudo dnf install ./tailbond-server-*.rpm` and `sudo systemctl enable --now tailbond`, then `systemctl status tailbond`. I'd check firewalld with the commands the guide gives (`--get-active-zones`, `--list-services`, `--list-rich-rules`), and also run `ss -tulpn` on my own to confirm 8077 and 80 are bound only to 127.0.0.1 and the Tailscale IPs. Then I'd open `http://homebox:8077` from my workstation on the tailnet. That's screen `15c-fedora-fresh-install-desktop`, Devices tab.

**What worked.** The package table and the "What the package changes in firewalld" list are exactly what I want: UDP 443 in the WAN zone, and "tcp 8077 and 80 from tailnet addresses only (`100.64.0.0/10`, `fd7a:115c:a1e0::/48`)" as rich rules. There's a `tailbond` zone for `tb0` and a `tailbond-out` policy. The guide also warns about the HTTP/3 clash on UDP 443 and says not to point `tailscale serve` at 8077, with the reason. That's good.

**Where I'd hesitate.**
- "SELinux needs no changes: the service runs unconfined as root, like other VPN servers." I appreciate the honesty, but it's a root daemon serving HTTP on two ports to the tailnet, with no SELinux confinement at all. I'd have expected a policy module, or at least a list of what it does as root (it edits firewalld, runs `tailscale set`, writes `/var/lib/tailbond`). **Low**: it's disclosed, I just don't like it.
- "Tailscale drops those source addresses on any other interface, so this doesn't open the ports to your LAN." That's true only while tailscaled manages netfilter. With `--netfilter-mode=off` it isn't. And the fallback "(else the default zone)" puts the rich rules in the same zone as my LAN NIC on a typical one-NIC Fedora box. I'd want that caveat stated, plus a way to check from the LAN. **Low.**
- "For best speed … see [setup tips](setup-tips.md)". That file isn't in what I was given. **Low.**
- Uninstall: "Remove it with `tailscale set --advertise-routes=` (keeping your other routes)." On my subnet router, that command as typed withdraws **all** advertised routes. The parenthetical expects me to re-list mine, but the command doesn't show how (e.g. `--advertise-routes=192.168.1.0/24`). **Low** (not part of this run, but someone will paste it).
- 15c header: a big "Connected" toggle on the **server** dashboard, before anything has connected. I can't tell what turning it off does (stop mqvpn? stop the agent?). I wouldn't touch it, but I'd expect a label. **Low.**

## Task 2 — Right after install (15c): what's left

**What I'd do.** On 15c I'd read the orange card "Setup step: tell devices where home is". It says to forward UDP 443 to 192.168.1.2 and set a DHCP reservation, then "Type it here:" with the public address or DDNS name and **Save**. Below that, "Setup health: 1 thing to improve for best speed" with "Route 10.77.0.0/24 isn't approved yet … Machines → this machine → Edit route settings". I'd approve it in the admin console, or more likely add `10.77.0.0/24` to my `autoApprovers`, which guide §3's health table mentions.

**What worked.** The two remaining steps are clear, both match guide §2 and §3, and the CGNAT hint below the address field ("If the router's own WAN address starts with 100.64–100.127 …") is exactly the check I'd do first.

**Where I'd hesitate.**
- The route item says "Tailscale machines can't reach bonded devices" until the route is approved. With my ACLs, approving the route isn't enough: my policy also needs a rule with `10.77.0.0/24` as a destination. Neither the dashboard nor the guide mentions this. **Low** (no task here needs tailnet→device traffic).

## Task 3 — Approve the family member's GL.iNet router

**What I'd do.** On `14-fedora-dashboard-request-desktop` (and `-phone`), I'd read "Waiting for your approval": "slate7 wants to connect · Travel router on your tailnet · signed in as alex@example.com · GL-BE3600 · 100.101.9.9". I'd check 100.101.9.9 against `tailscale status`, then click **Approve**. `14b` shows the toast "slate7 can connect now. It connects by itself in a few seconds."

**What worked.** The request carries the Tailscale identity, model and IP, so I can verify it against the admin console before approving. That's exactly what I'd want.

**Where I'd get stuck or misread.**
- The same banner says: "Who can approve: **anyone on your Tailscale network who can reach homebox (it's a tagged machine, so it has no single owner)**. Set a password below to limit that." I never tagged homebox. On 15c the "Who can approve" card said "alex@example.com (the Tailscale account homebox belongs to)". On screen `15`, both texts appear **on the same page**: the top banner says tagged and anyone can approve, and the card lower down says owner-only, alex@example.com. The Tailscale tab (15b) shows "TAILNET alex@example.com". I can't tell whether my dashboard is open to the whole tailnet right now. See Task 6. **High** (counted once, under Task 6).
- "Approving lets the device reach your Tailscale machines the way homebox does (they see it as homebox, under its access rules)." That's correct but incomplete. Guide §4 says an approved device reaches "the internet through your home connection, your home LAN, and your tailnet". The banner, which is where I actually decide, leaves out the **home LAN** (router admin, NAS UI, printers, IoT) and the default exit-node behaviour. And on an untagged server, "homebox's access rules" in practice means *my user's* grants. For a family member's device that matters, and the banner doesn't say it. **Medium.**
- "Set a password **below**". I'm on the Connections tab (14), and there's no password field below. It's on the Devices tab. **Low.**
- After the toast in 14b, the Connections list still shows only pixel-8, with no "slate7 connecting…" entry. Later (15d) slate7 appears on Devices as "offline", with a **phone** icon even though it's a router. Nothing confirms the approval actually led to a connection. **Low.**

## Task 4 — Get tailbond onto a family member's Android phone

**What I'd do.** I'd go to Devices → "Add an Android phone" (15c / 15) and copy `http://homebox.tail1234.ts.net/` to the family member, or tell them to type `homebox/`. On their phone (`web-00-phone-page`) they get "Get tailbond on this phone": (1) Download tailbond for Android (27 MB), with pictures of the four Android prompts; (2) "Open tailbond and ask homebox"; (3) Turn tailbond on. On my side, `14c` shows "pixel-7a wants to connect · Phone on your tailnet · signed in as sam@example.com · Pixel 7a · 100.101.7.7", and I'd click **Approve**.

**What worked.** The phone page is good. The APK SHA-256 is printed, "Later updates must be signed with the same key", "Android runs one VPN at a time, so while tailbond is on, Tailscale's VPN is off on this phone", and "by default this phone's internet traffic goes out from homebox, like using it as an exit node". That's the kind of disclosure I want my family to see. The computer view (`web-00c`) explains itself too.

**Where I'd get stuck.**
- Guide §4: "On the phone, with Tailscale on, open the browser and type the server's name (`homebox`) … or just tell them to type the name." Guide §1 and both dashboard cards say the opposite: type `homebox/` "with the slash at the end, so it opens the page instead of searching". If I relay §4 to a family member as written, their browser does a web search. **Medium.**
- I run a restrictive ACL. Sam's phone has to reach homebox on **tcp 80** (phone page) and **tcp 8077** (the pair request). The guide mentions "Keep tcp 80 and 8077 to the server open for the people whose devices should ask to connect" only inside the *tagged-server* example. §4 "An Android phone that runs Tailscale" and the dashboard card don't mention it. With my policy, Sam would get a browser timeout on `homebox/` and nothing would tell either of us why. **Medium.**
- After I approve pixel-7a there's no screen showing the phone connected, and there are no screenshots of the app (step 3: "tap Add phone. Then tap the big switch"). I can't verify the end of this task. **Low.**
- The wording varies between screens: "Anyone on your Tailscale account can open it" (web-00c), "on your tailnet" (banner), and "this dashboard only answers you" (Android card). Sam is on my tailnet but a different account (sam@example.com), so "Tailscale account" reads as if Sam couldn't open the page. **Low.**

## Task 5 — Setup health, including the relayed machine

**What I'd do.** On `15-fedora-devices-tab-desktop`, Setup health shows "2 things to improve":
1. "Tailscale: 2 machines direct, 1 through a relay (slower) — Through a relay: apple-tv … To check: tailscale ping apple-tv ("via DERP" means relayed). On the router in front of this machine, forward UDP port 41641 to it, and allow 41641/udp in this machine's firewall."
2. "This network makes direct Tailscale connections hard … ("hard NAT")", with the same fix.

I'd forward UDP 41641 on my router to 192.168.1.2, then run the guide's §3 command `sudo firewall-cmd --permanent --zone=<wan zone> --add-port=41641/udp && sudo firewall-cmd --reload` (with the zone from `--get-active-zones`), and check with `tailscale ping apple-tv`. `15d` then shows "everything is set up for best speed", with "Tailscale: 3 machines direct" and "UDP 443 reached from the internet — pixel-8 connected from 198.51.100.20, 2 h ago".

**What worked.** It's a concrete fix with a way to verify it (`tailscale ping`, "via DERP" vs "via <ip>:<port>"). It also says bonding itself isn't affected, and it warns about UPnP. That's good.

**Where I'd hesitate.**
- The card footer says "Each fix is on your home router or in the Tailscale admin console". The item right above it asks me to "allow 41641/udp in this machine's firewall", which is neither. The dashboard doesn't give the `firewall-cmd` line; only the guide does, and with a `<wan zone>` placeholder. **Low.**
- apple-tv is relayed, but the dashboard doesn't say the relay might come from apple-tv's side of the network. The guide does: "A relay can also come from the other machine's network, which you may not control". There's also no visible "check again" after I change the router. **Low.**
- After my fix, 15d shows "Network is friendly to direct connections — port mapping: **UPnP**". I set up a manual forward precisely because the guide warned that UPnP "lets any device on your LAN open ports". Seeing UPnP there would make me go check whether something turned UPnP on. **Low.**
- Tailscale tab (15b): a numbered list with one item, "1. All set: bonded devices reach your tailnet through this server." **Low.**

## Task 6 — Decide and set who may approve new devices

**What I'd do.** I'd go to Devices → "Who can approve and add devices" (15 / 15d). My decision: homebox stays untagged and owned by me, so only alex@example.com approves. I'd set no password.

**Where I'd get stuck.**
- The dashboard contradicts itself about the current state, which is what this decision depends on. On screen 15, the banner says "anyone on your Tailscale network who can reach homebox (it's a tagged machine, so it has no single owner)". The card on the same page says "Who can approve: alex@example.com (the Tailscale account homebox belongs to), from any of their devices. People you share homebox with, and others on your Tailscale network, can ask to connect but can't approve." If the banner is right, everyone on my tailnet can approve devices into my LAN right now, and I **must** set a password. If the card is right, I shouldn't. I'd stop and go check the admin console for tags. If I trusted only the banner, I'd set a shared password I don't want (see the next point). **High.**
- The only way to add a second approver (say, my partner) is a shared password: "To let someone else approve too, set a password and give it to them." The guide says a password "protects the whole dashboard API" and that even `curl … http://127.0.0.1:8077` then needs `/api/login`. So setting a password seems to *replace* the identity check rather than add to it. Neither text says whether my own Tailscale identity still gets in without it. I already manage access with Tailscale identities, so I'd expect "approvers: these Tailscale users/groups", not a shared secret. The guide's tagged-server ACL example (which I'd want, to limit what bonded devices reach) also forces me onto the password model: "A tagged server has no owner, so set a dashboard password too". So I can either limit what devices reach or keep identity-based approval, not both. **Medium.**

---

## Findings

| # | Task | Finding | Severity |
|---|---|---|---|
| 1 | 6 (seen in 3) | Same screen says both "it's a tagged machine, so it has no single owner … anyone on your Tailscale network … can approve" (banner) and "alex@example.com (the Tailscale account homebox belongs to)" (card); I can't tell who can approve right now, so I can't decide whether a password is needed | high |
| 2 | 3 | Approval banner "reach your Tailscale machines the way homebox does" leaves out home LAN access and the exit-node default from guide §4, and doesn't say that on an untagged server that means the owner's grants | medium |
| 3 | 4 | Guide §4 "type the server's name (`homebox`) … just tell them to type the name" contradicts §1 and the dashboard ("homebox/ … with the slash"); as written it gives a web search | medium |
| 4 | 4 | Family devices need ACL access to tcp 80/8077 on homebox; only mentioned inside the tagged-server example, not in §4 or on the Android card | medium |
| 5 | 6 | Only a shared password adds approvers, and it seems to replace the identity check (even localhost needs `/api/login`); no Tailscale user/group approver list; tagging (to limit reach) forces the password model | medium |
| 6 | 1 | "runs unconfined as root": no SELinux confinement for a daemon serving 80/8077 | low |
| 7 | 1 | "doesn't open the ports to your LAN" depends on tailscaled netfilter mode; the default-zone fallback is usually the LAN zone; no LAN-side check given | low |
| 8 | 1 | Guide links `setup-tips.md`, which wasn't provided | low |
| 9 | 1 | Uninstall `tailscale set --advertise-routes=` withdraws all subnet routes; doesn't show how to keep them | low |
| 10 | 1 | Unlabelled "Connected" toggle in the server dashboard header | low |
| 11 | 2 | Route approval alone isn't enough under a restrictive ACL (needs a grant to `10.77.0.0/24`); not mentioned | low |
| 12 | 3 | "Set a password below" shown on the Connections tab, where there's no password field | low |
| 13 | 3 | After "slate7 can connect now…" nothing shows it connecting; later listed "offline" with a phone icon | low |
| 14 | 4 | No screen confirms pixel-7a connected; app steps ("tap Add phone", "the big switch") not shown | low |
| 15 | 4 | "Anyone on your Tailscale account can open it" vs "on your tailnet" vs "only answers you": account/tailnet wording varies | low |
| 16 | 5 | Footer "Each fix is on your home router or in the Tailscale admin console" vs "allow 41641/udp in this machine's firewall"; no `firewall-cmd` line on the dashboard | low |
| 17 | 5 | Relay item doesn't say the cause may be on apple-tv's side (only the guide does); no "check again" | low |
| 18 | 5 | After a manual forward, 15d reports "port mapping: UPnP", which the guide warned against | low |
| 19 | 5 | Tailscale tab: one-item numbered list "1. All set: …" | low |

Totals: 1 high / 4 medium / 14 low
