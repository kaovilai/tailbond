# Round 7 — Fedora persona F1 (homelab admin)

Persona: I run Fedora servers with firewalld and SELinux enforcing. Tailscale with subnet routers and ACLs, more than one user on the tailnet, and a couple of machines shared out to family. I read install guides line by line, and for anything new I want to know three things: what gets opened, to whom, and how I check it.

Materials: `task-script.md`, `screens.md`, `fedora-server.md`, and the screenshots in `glinet/` and `web/`. Nothing else.

---

## Task 1: Install on homebox and open the dashboard

**What I'd do.** Guide section 1: `sudo dnf install ./tailbond-server-*.rpm`, then `sudo systemctl enable --now tailbond`, then `systemctl status tailbond` and `journalctl -u tailbond -f`. The guide says Fedora 43 or newer, and the RPM is `fc44`, so that's fine. I'd open `http://homebox:8077` from my laptop on the tailnet, which the guide covers: "the agent also listens on the server's Tailscale IPs". I land on the Devices tab (15c).

**What's good.** The package table is the kind of thing I want: binaries, config path, state dir, hooks dir, unit, and firewalld objects. The note on 8077/80 is precise: "lets tailnet addresses (`100.64.0.0/10`, `fd7a:115c:a1e0::/48`) reach tcp 8077 and 80 … Tailscale drops those source addresses on any other interface, so this doesn't open the ports to your LAN". The uninstall section admits that the advertised route stays behind.

**Where I'd stop and worry.**

- **UDP 443 goes into "your default zone".** Section 2.3: "the `server-up` hook opens the `tailbond` service (UDP 443) in your default zone. You don't need to run any `firewall-cmd` commands yourself." On my boxes the default zone is `FedoraServer`, but the LAN NIC isn't always in the default zone; some of mine sit in `internal` or `home`. If the NIC that receives the port-forwarded traffic isn't in the default zone, this silently doesn't work, and the guide just told me not to touch `firewall-cmd`. I'd expect "the zone of the interface holding the default route" or a config key, plus a one-line check such as `firewall-cmd --zone=<zone> --list-services`. **Medium.**
- **What the tunnel can reach on my side isn't stated.** The package table says only "firewalld `tailbond` service, zone and policies | UDP 443, `tb0`, forwarding and NAT". Section 3 explains the tunnel → tailnet path. Nothing says whether a bonded device can reach homebox's own services (SSH, Cockpit on 9090, the 8077 dashboard via `10.77.0.1`) or other LAN hosts (`192.168.1.0/24`). Is the `tailbond` zone `target=ACCEPT`? Does the policy forward to `ANY`? The only inspection command (`sudo firewall-cmd --info-policy=tailbond-out`) is buried in Troubleshooting. I'd expect a short "what's exposed" table: from tb0 to host, LAN, tailnet and internet, each with the rule and a `firewall-cmd --info-zone=tailbond` / `--info-policy=…` check. **Medium.**
- **UDP 443 might already be taken.** Plenty of homelab boxes run Caddy, nginx or Traefik with HTTP/3, which listens on UDP 443. The guide never says whether the port can be changed or what happens on a conflict. Section 1 handles a port 80 conflict ("If something else already uses port 80 … the page is at `http://<server>:8077/get`"); UDP 443 gets no such note. **Low.**
- **SELinux isn't mentioned at all.** I run enforcing. The guide doesn't say whether the agent is confined or runs `unconfined_service_t`, what user it runs as, or whether I should expect AVCs when it binds port 80 or runs `tailscale cert` and firewalld hooks. I'd want a line like "no AVCs expected; check with `ausearch -m avc -ts recent`". **Low.**
- **The dashboard is plain HTTP.** The guide says the agent gets a real cert via `tailscale cert` (the Tailscale tab shows "Let's Encrypt via Tailscale (homebox.tail1234.ts.net)"), but the dashboard and its password field are still served on `http://…:8077`. WireGuard covers the transport, but Chrome will show "Not secure" on a page with a password field. **Low.**

---

## Task 2: Right after install (15c): what's left, and how?

**What I see.** The Devices tab:
- "Setup health: 1 to improve for best speed" with "Route 10.77.0.0/24 isn't approved yet — Approve it once in the Tailscale admin console (Machines → this machine → Edit route settings). How to fix".
- Then the cards "Who can approve and add devices", "Add an Android phone", "Add a GL.iNet travel router (recommended)" and "Add a device with a code (no Tailscale on it)".
- Then, near the bottom, a peach banner: "Public address not set. Pairing links will use 192.168.1.2:443, which is probably your LAN address. Enter your public IP or DDNS name (and forward UDP 443 on your router)", with a field (placeholder `home.example.net:443`) and Save.

**My answer to "what's left".** Two things. (1) Approve `10.77.0.0/24` in the admin console. (2) Forward UDP 443 on the router to homebox (with a DHCP reservation) and enter the DDNS name in the bottom banner. The guide also points me to the password setting in "Who can approve…". I'd do (2) first, because without it nothing works away from home.

**Problems.**

- **The guide sends me to a tab that doesn't exist.** Section 2.2: "Use the banner in the dashboard's **Clients** tab". The tabs are Connections, Devices, Tailscale and Logs. The banner is on Devices, at the bottom. I'd find it, but only after looking for "Clients". **Medium.**
- **Setup health undercounts, and the blocking item is last.** The health card says "1 to improve for best speed" and lists only the route. The item that actually breaks pairing away from home (public address and UDP 443 forward) isn't in the health card. It sits in a separate banner below three "Add a…" cards, so the natural top-to-bottom flow is to add devices first, with LAN-only links. The header also says "Home server · reachable at 192.168.1.2:443", and "reachable" is the wrong word for a private address the banner itself calls "probably your LAN address". I'd expect public address / port forward to be the first health item, and the header to say "not reachable from outside yet". **Medium.**
- **Is the port part of the public address or not?** The field placeholder is `home.example.net:443`. The guide's JSON is `"public_addr": "home.example.net"` without a port, and its IPv6 example is `[2001:db8::10]:443` with one. I'd guess the port is optional, but I shouldn't have to guess. **Low.**
- **Odd state on a fresh install.** Right after `systemctl enable --now`, the header shows "UPTIME 5d 0h". That may be a fixture artifact, but it's what's on screen. There's also a "Connected" toggle at top right on a server with "No devices yet". What does turning it off do: stop mqvpn, or drop all clients? It isn't labelled anywhere. **Low.**
- **No mention of `autoApprovers`.** I'd approve routes through `autoApprovers` in my ACL policy, not by clicking in the console. The guide and the health text only say "Edit route settings". One line would help. **Low.**
- **Advice aimed at the wrong person.** Under the route item: "Not sure how? Show this card to whoever set up your internet; the links explain each fix." Route approval is a Tailscale admin-console action, not an ISP or router one. Whoever set up the internet may not be a tailnet admin. **Low.**

---

## Task 3: Approve the family's GL.iNet router (14, 14b)

**What I'd do.** On 14 there's an orange card at the top: "Waiting for your approval … slate7 wants to connect — Travel router on your tailnet · signed in as alex@example.com · GL-BE3600 · 100.101.9.9", with Decline and Approve. Before clicking I'd cross-check with `tailscale whois 100.101.9.9`. The card gives me the IP to do that, which is exactly right. Then I'd click **Approve**. 14b shows a toast: "slate7 can connect now. It connects by itself in a few seconds." The phone layout (14-phone) puts the same buttons at a reachable size.

**Problems.**

- **Identity disappears after approval.** After approval, the Devices list (15d) shows "slate7 — offline" with a phone icon, although it's a router. The Tailscale identity from the request ("signed in as alex@example.com · GL-BE3600 · 100.101.9.9") is gone. pixel-8 shows only "198.51.100.20:51234 · wlb". Six months from now, deciding which key to revoke, I'd want to see whose device each key belongs to. **Low.**
- **The health banner shouts.** Right under the approval card, the "Setup health: 2 things to improve for best speed. Show" banner is red, the same colour as an error, for what are speed tips. **Low.**

The task itself was straightforward.

---

## Task 4: Get tailbond onto a family member's Android phone

**What I'd do.** Devices → **Add an Android phone** (15c): "On the phone, with Tailscale on: point the camera at this code, or type homebox into its browser … For family: send them this link, or just tell them to type the name." I'd copy `http://homebox.tail1234.ts.net/` with the copy button and send it. On the phone (web-00): "Download tailbond for Android (27 MB)", the four illustrated install prompts, then "Open tailbond and ask homebox", then I approve on the dashboard, then step 3, "Turn tailbond on". There's no screenshot of a phone's request on the dashboard, so I assume it looks like slate7's card. The guide says it shows "the phone's Tailscale identity".

**What's good.** The install-prompt illustrations ("File might be harmful … Download anyway", "Allow from this source") and "You can turn Allow from this source off again afterwards" are exactly what I'd otherwise explain by phone. Step 3 is honest: "Android runs one VPN at a time, so while tailbond is on, Tailscale's VPN is off on this phone."

**Problems.**

- **"Just tell them to type the name" won't work in Chrome.** On Android, Chrome treats a single word like `homebox` as a search query and sends it to the search engine; it doesn't go to `http://homebox/`. My family member would get Google results. The dashboard card, the 14-phone card ("No code to scan") and guide section 4 ("or just tell them to type the name") all lean on this. I'd tell them to type `homebox/` or `http://homebox`, or just send the link. **Medium.**
- **"Nothing here changes the server" is one tap away from being false.** The dashboard says "The page only offers the app; nothing on it can change this server", and the phone page says "nothing here changes homebox's settings". But the phone page links straight to the admin dashboard twice: "You run homebox yourself? Approve it at http://homebox:8077 … that page works on this phone too", and the footer "homebox's dashboard, for whoever runs it: http://homebox:8077". By default that dashboard has no password, so a family member who taps the link can approve, revoke and add devices. I'd expect the family page not to advertise the admin URL, or at least not until a password is set. **Medium.**
- **Two steps or three?** The dashboard card says "That page has the app and two steps", the guide says "the Android app and two steps", and 14-phone says "Two steps". The phone page has three numbered steps (1 Install the app, 2 Add this phone, 3 Turn tailbond on). **Low.**
- **No way to verify the APK.** The page tells my family to tap "Download anyway" on "File might be harmful" and allow unknown sources. Neither the dashboard nor the phone page shows a SHA-256 or signing-cert fingerprint I could check first. For a side-loaded VPN app that sees all of the phone's traffic, I'd want one. **Low.**

The task can be finished: send link → install → ask → Approve.

---

## Task 5: Read setup health and fix it, including the relayed machine (15, 15d)

**What I see (15).** "Setup health: 2 to improve for best speed":
1. "Tailscale: 2 machines direct, 1 through a relay (slower) — Through a relay: pixel-8. To check one: tailscale ping pixel-8 — Usually fixed by forwarding UDP port 41641 to this machine, or by turning on UPnP on the router in front of it. How to fix"
2. "This network makes direct Tailscale connections hard — Hard NAT (the outside port changes per destination); no port mapping on the router. Forward UDP port 41641 to this machine, or turn on UPnP on the router in front of it. How to fix"

**What I'd do.** Run `tailscale ping pixel-8` and `tailscale netcheck` on homebox. I like that it names the command. Then add a router port forward for UDP 41641 → homebox (no UPnP on my network). 15d then shows all green: "Tailscale: 3 machines direct", "Network is friendly to direct connections (port mapping: UPnP)", route approved, "Public internet address 203.0.113.7", "UDP 443 reached from the internet — pixel-8 connected from 198.51.100.20, 2 h ago".

**Problems.**

- **firewalld on homebox also has to allow UDP 41641, and nothing says so.** Both fixes say "Forward UDP port 41641 to this machine". On Fedora, the router forward alone doesn't help if the `FedoraServer` zone drops unsolicited UDP 41641, which by default it does. The guide stresses that the hook handles UDP 443 ("You don't need to run any `firewall-cmd` commands yourself"), which suggests the hook handles 41641 too. Nothing says it does. I'd expect "forward UDP 41641 on the router **and** `firewall-cmd --permanent --add-port=41641/udp`" (or a statement that the hook does it), plus a note that 41641 is tailscaled's `PORT` in `/etc/default/tailscaled`. **Medium.**
- **The relayed machine is pixel-8, the bonded phone.** Connections shows pixel-8 bonded with "2 active path(s)". The phone page says that while tailbond is on, "Tailscale's VPN is off on this phone", and guide section 2 says bonding "does not go through Tailscale's relays". So what does fixing pixel-8's Tailscale relay buy me? The card says "for best speed" without saying which traffic gets faster. Is it homebox's own tailnet hops to the NAS, or the phone when it isn't bonded? I'd go fix a relay that may make no difference to the bonded phone's speed. I expected the card to say which path is slow (for example "homebox ↔ pixel-8 Tailscale traffic, used when tailbond is off") or to leave out peers that are currently bonded. **Medium.**
- **One fix, counted twice.** "2 to improve" are the same fix: forward UDP 41641 or turn on UPnP. UPnP is offered as an equal option, which as a firewall-minded admin I'd rather it didn't encourage. **Low.**
- **A stray "null" on the all-good screen.** 15d shows a bare "null" under the last check, above "Direct connections are faster than relayed ones." It looks like a check that failed to render, and it made me wonder whether something wasn't verified. **Low.**

---

## Task 6: Decide and set who may approve new devices

**What I see.** Devices → "Who can approve and add devices": "Anyone who can open this page (people on your Tailscale account, and anyone you share this machine with). Set a password to require it." There's a field "Password for this page (empty: none)" and Save. The approval card repeats it: "To require a password: Devices tab → Who can approve." Guide section 4: "Who can approve: anyone who can open this page … everyone on your tailnet who can reach this machine (check your ACLs). Set `ui_password` in `tailbond.json` to require a password."

**My decision.** Set a password now, and restrict `tcp:8077` on homebox to my admin group in the tailnet ACL. Then I'd hit the questions below.

**Problems.**

- **High: a bonded device inherits homebox's tailnet access, and by default anyone who can reach homebox can grant that.** Guide section 3: "Traffic from the tunnel (`10.77.0.0/24`) to `100.64.0.0/10` … leaves through `tailscale0` with masquerade. To the tailnet it looks like it comes from the server, so bonded devices don't show up as extra machines." That is presented as a convenience, but for me it's an ACL bypass. Every bonded device gets homebox's ACL permissions, not its owner's. The phone page confirms it ("Your Tailscale machines (like the NAS) stay reachable through homebox"), and a bonded phone drops its own Tailscale identity because "Tailscale's VPN is off on this phone".

  Now combine that with the default. The dashboard says approval is open to "anyone you share this machine with", and the API in section 4 (`POST /api/clients`) mints keys. A sharee, who by Tailscale's design should only reach homebox, can approve their own device or add a key and then reach my whole tailnet and egress from my home IP. Nothing in the guide or on screen warns about this, there's no per-device "internet only / no tailnet" option, and the open default is the one the product ships. I'd expect a clear warning in section 3 and on the "Who can approve" card, and a password, or approval limited to the tailnet owner, by default. **High.**
- **Does the password cover the API?** The setting is called "Password for this page" / `ui_password`. The guide's shell examples (`curl … http://127.0.0.1:8077/api/clients`) show no auth. With a password set, can another tailnet node still `curl -X POST http://homebox:8077/api/clients` and get a pairing URI? I can't tell from anything I have. **Medium.**
- **"Check your ACLs", but which ports?** If I restrict `tcp:8077` to admins, will a family router's "Ask to connect" or the phone's "Open tailbond and ask homebox" still reach homebox? Do those requests use 8077, port 80, or something else? The guide doesn't say, so the ACL route it recommends could break the family flow. I'd expect an ACL snippet: `tcp:80` (and whatever the request uses) for family, `tcp:8077` for admins, `udp:443` from anywhere. **Medium.**
- **Only a shared password, no identity list.** The approval card already shows "signed in as alex@example.com", so the server knows Tailscale identities, yet the only control is one shared password. I expected "approvers: tailnet users X, Y" (via `tailscale whois` on the caller). **Low.**
- **The UI understates who has access.** It says "people on your Tailscale account". On a multi-user tailnet, a reader takes "your account" to mean just them, while the guide says "everyone on your tailnet who can reach this machine". The UI should say "everyone on your tailnet" so the decision is made with the right picture. **Low.**

---

## Findings

| # | Task | Finding | Severity |
|---|---|---|---|
| 1 | 1 | "the `server-up` hook opens the `tailbond` service (UDP 443) in your default zone. You don't need to run any `firewall-cmd` commands": assumes the WAN-facing NIC is in the default zone; no zone override or check command | medium |
| 2 | 1 | What tb0 can reach (homebox's own services, LAN `192.168.1.0/24`, tailnet, internet) isn't stated; "forwarding and NAT" is the only description, and the inspection command is buried in Troubleshooting | medium |
| 3 | 1 | UDP 443 conflicts with HTTP/3 reverse proxies on the same box; no note on changing the port (port 80 gets a fallback note, 443 doesn't) | low |
| 4 | 1 | No SELinux mention (confinement, service user, expected AVCs) | low |
| 5 | 1 | Dashboard and password field served on plain `http://…:8077` though the server has a Tailscale TLS cert | low |
| 6 | 2 | Guide: "Use the banner in the dashboard's **Clients** tab"; there is no Clients tab (banner is at the bottom of Devices) | medium |
| 7 | 2 | "Setup health: 1 to improve" leaves out the blocking public-address / UDP 443 item, which sits below the "Add a…" cards; header says "reachable at 192.168.1.2:443" | medium |
| 8 | 2 | Public address format: placeholder `home.example.net:443` vs guide `"public_addr": "home.example.net"` | low |
| 9 | 2 | Fresh install shows "UPTIME 5d 0h"; unlabelled "Connected" toggle on a server with no devices | low |
| 10 | 2 | Route approval only via "Edit route settings"; no mention of ACL `autoApprovers` | low |
| 11 | 2 | "Show this card to whoever set up your internet" shown for a Tailscale admin-console task | low |
| 12 | 3 | After approval the device list drops the Tailscale identity ("signed in as …", 100.101.9.9); router slate7 shown with a phone icon | low |
| 13 | 3 | Speed tips shown in an error-red banner ("Setup health: 2 things to improve…") | low |
| 14 | 4 | "or just tell them to type the name": Chrome on Android sends a single word like `homebox` to the search engine | medium |
| 15 | 4 | Phone page says "nothing here changes homebox's settings" yet links twice to `http://homebox:8077`, which has no password by default | medium |
| 16 | 4 | "two steps" (dashboard, guide, 14-phone) vs three numbered steps on the phone page | low |
| 17 | 4 | No APK SHA-256 or signing fingerprint shown while users are told to tap "Download anyway" | low |
| 18 | 5 | "Forward UDP port 41641 to this machine": no word that firewalld on homebox must also allow 41641/udp (the hook only handles 443) | medium |
| 19 | 5 | Relayed machine is "pixel-8", the bonded phone whose Tailscale is off and whose bonding "does not go through Tailscale's relays"; the card doesn't say which traffic the fix speeds up | medium |
| 20 | 5 | "2 to improve" is one fix (41641 / UPnP) counted twice; UPnP offered as an equal option | low |
| 21 | 5 | Stray "null" on the all-good health card (15d) | low |
| 22 | 6 | Masqueraded tunnel gives every bonded device homebox's tailnet ACL rights ("To the tailnet it looks like it comes from the server"), and by default approval and key creation are open to "anyone you share this machine with": a sharee can get full tailnet access and home egress; no warning, no per-device restriction | high |
| 23 | 6 | Unclear whether "Password for this page" / `ui_password` also protects `/api/clients` (guide's curl shows no auth) | medium |
| 24 | 6 | "check your ACLs" with no snippet or port list; restricting tcp:8077 might break "Ask to connect" requests from family devices | medium |
| 25 | 6 | Only a shared password; no approver list by Tailscale identity even though identities are shown | low |
| 26 | 6 | UI says "people on your Tailscale account"; guide says "everyone on your tailnet who can reach this machine" (UI understates) | low |

Totals: 1 high / 10 medium / 15 low
