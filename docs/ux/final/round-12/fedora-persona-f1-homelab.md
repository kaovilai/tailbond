# Round 12 — Fedora persona F1 (homelab admin)

**Persona:** homelab admin. Fedora servers with firewalld and SELinux enforcing, Tailscale with subnet routers and a hand-written ACL policy. Reads install guides line by line. Wants to know what gets exposed to whom, on which ports, and how to verify each claim.

**Materials used:** `task-script.md`, `screens.md`, `fedora-server.md`, `glinet/14*`, `glinet/15*`, `web/web-00*`, `web/web-04*`. Nothing else.

**Overall:** I finished every task. The guide is unusually honest about exposure. It lists every firewalld change with a `firewall-cmd` command to check it, admits that SELinux isn't confining the service, and the approval card says plainly that a device gets "whatever they allow homebox". My remaining problems are about control and cleanup:

- what the package changes on every start;
- what it leaves behind after uninstall;
- one config key whose location the UI doesn't give;
- how far a family phone reaches into a tailnet that has subnet routers;
- an approval model where any local process counts as an approver.

---

## Task 1 — Install on homebox and open the dashboard

**What I'd do:** guide §1: `sudo dnf install ./tailbond-server-*.rpm`, `sudo systemctl enable --now tailbond`, then `systemctl status tailbond` / `journalctl -u tailbond -f`. Before trusting it, I'd run the guide's checks: `firewall-cmd --get-active-zones`, `--list-services`, `--list-rich-rules`, `--info-policy=tailbond-out`. Then I'd open `http://homebox:8077` from my laptop on the tailnet, which lands on the screen in 15c.

**What works for me:**
- The package-contents table and the firewalld list ("this list is for checking") are exactly what I want before installing anything that touches the firewall.
- The commands to verify each change are included.
- The warning not to point `tailscale serve` at 8077 is a good catch.
- "SELinux needs no changes: the service runs unconfined as root" is honest, even if I don't like it.

**Where I stumble:**

1. **The firewall is rewritten on every start, the permanent config too, and there's no opt-out.** The guide says "What the package changes in firewalld (the `server-up` hook, every start, both the running and the saved configuration …). There's nothing to do here yourself." This includes opening "UDP 41641 (Tailscale's own port) in the same zone as UDP 443 when Tailscale is installed", which is my WAN-facing zone, at install time, before I've decided to forward anything. If I remove one of those services by hand, the next restart puts it back. I manage firewalld with my own tooling. I'd expect either a documented switch that turns off firewall management (as `"no_advertise_route"` does for the route), or for the hook to touch only the running config. The only opt-out I can find is for the case where firewalld isn't installed at all ("**No firewalld:** the hooks fall back…"). **Medium.**

2. **Uninstall leaves the Tailscale route behind, and the cleanup command is a trap if you run subnet routers.** The guide says "The tunnel route stays advertised in Tailscale: `tailscale set --advertise-routes=<your other routes, comma separated>` keeps the others (an empty value withdraws all of them)." `dnf remove` cleans up firewalld, but I have to hand-retype every subnet route this box advertises. One typo, or an empty value, and my LAN subnet route is gone. Since the hook "keeps any routes you already advertise" on the way in, I'd expect the `server-down`/remove step to take only `10.77.0.0/24` back out the same way. Failing that, the guide should give a command that reads the current list first (e.g. from `tailscale debug prefs`). **Medium.**

3. **SELinux.** The guide says "the service runs unconfined as root, like other VPN servers". It's disclosed, but on Fedora I'd hope for at least a plan for a policy module. A root, unconfined daemon serves HTTP on 80 and 8077. **Low.**

4. **One firewall safety claim depends on how tailscaled is configured.** The guide says "Tailscale drops those source addresses on any other interface, so this doesn't open the ports to your LAN." That's only true while tailscaled manages netfilter. Some of us run `--netfilter-mode=off` so that firewalld owns the ruleset. I'd expect a caveat, or rich rules bound to the `tailscale0` interface/zone rather than to source ranges in the default zone. **Low.**

5. **"Machine list" is undefined.** Section 1 says that on `10.77.0.1` "the agent answers them there only with the machine list, never the dashboard". Which machine list? My tailnet's hostnames and IPs? Every bonded device gets it, including code-paired ones that aren't on Tailscale at all. I'd want to know exactly what that endpoint returns. **Low.**

6. **Config keys without their location.** The guide says to turn off the phone page "with `"landing_port": -1`", but doesn't say where in `tailbond.json` it goes (top level? under `"server"`?) or whether a restart is needed. **Low.**

7. **The header's "reachable" contradicts the setup step (15c).** The header says "Home server · reachable at 192.168.1.2:443". The setup step underneath says "an address inside your home, so they only work at home". It isn't reachable in the sense I care about. **Low.**

8. **The header toggle is unexplained.** On the server's own dashboard there's a "Connected" switch in the header on every screen. What does turning it off do: stop mqvpn, stop the whole agent, block new devices? Neither the guide nor the screen says, and I'm not going to flip it on a production box to find out. **Low.**

## Task 2 — Right after installing (15c): what's left, and how?

**What I'd do:** 15c makes this clear. The orange card is "Setup step: tell devices where home is":
1. "Forward UDP port 443 to this machine (192.168.1.2)… Give this machine a fixed address there too (DHCP reservation)".
2. Find the public address or DDNS name and type it into the box ("e.g. home.example.net or 203.0.113.7"), then **Save**.

The health card says "nothing to fix here (1 optional item below) · finish the setup step above first". The one item is "Route 10.77.0.0/24 isn't approved yet (optional)". This matches guide §2 and §3. I like that it says "The port (443) is added for you" and mentions CGNAT up front.

On the route: I don't want my tailnet reaching bonded family devices, so I'd leave it unapproved. I'd also stop advertising it, so it doesn't sit pending in my admin console.

**Where I stumble:**

9. **The route item gives an incomplete config instruction.** 15c says: "If you do not want it: leave it, or set "no_advertise_route": true in tailbond.json to stop advertising it." The guide (§3) says it belongs "under `"server"` in `tailbond.json`", followed by `sudo systemctl restart tailbond`. The on-screen text leaves out both the `"server"` key and the restart. Following the screen, I'd add it at the top level, reload the page, see the route still advertised, and not know why. I'd expect the screen text to read `"server": {"no_advertise_route": true}` and say to restart. **Medium.**

10. **I can't verify the forward on its own terms.** Guide §2.4: "Once a phone is paired… turn Wi-Fi off; once tailbond connects over mobile data, the dashboard's *UDP 443* health item turns green." So the first proof that my firewall and router forward work only comes after I've paired a phone and taken it off Wi-Fi. There's no packet screen showing the state after **Save** either. I can use `tcpdump -ni any udp port 443` myself, but the guide could mention it as the check. **Low.**

## Task 3 — Approve the family's GL.iNet travel router

**What I'd do:** 14 (or 15) shows the "Waiting for your approval" card at the top of every tab: "slate7 wants to connect · Travel router on your tailnet · signed in as alex@example.com · GL-BE3600 · 100.101.9.9". I'd check the Tailscale IP against `tailscale status`, read the orange warning, then click **Approve**. 14b shows a toast: "slate7 can connect now. It connects by itself in a few seconds."

**What works:**
- The identity line (login, model, Tailscale IP) is what I'd want in order to verify the request.
- The text "reach your Tailscale machines as homebox: they allow it whatever they allow homebox, which may be more than its owner's own account gets" is the most important sentence on the page, and it's there.
- The warning that the router is signed in as me ("If you'll give it to someone: approve, then before handing it over sign its Tailscale in to their account (or tag it)") is correct and useful. I'd tag it.

**Where I stumble:**

11. **I can't confirm the approval worked.** The toast promises a connection "in a few seconds", but 14b's Connections list still shows only pixel-8. No "slate7 · connecting" row appears. The next time I see slate7 (15d) it's "Not Connected", with the same phone icon as the iPad and pixel-8, even though it's a router. I'd expect a "waiting for slate7…" row until it connects, or an explanation of what to check if it doesn't. **Low.**

12. **Decline isn't a block.** The card says "Decline only refuses this request; it can ask again." The guide adds that everyone on the tailnet "can still **ask to connect**". I can't see any way to stop a device or user from asking repeatedly, apart from my ACL on tcp 80/8077. **Low.**

## Task 4 — Get tailbond onto a family member's Android phone and connect it

**What I'd do:** in the Devices tab, the "Add an Android phone" card (15c/15/web-04) gives a QR code and `http://homebox.tail1234.ts.net/`. The family member is already on Tailscale, so for my tailnet they're either invited or have homebox shared with them.

Because my ACL isn't allow-all, I'd add the guide's grant from `autogroup:member` to the server on `tcp:80`/`tcp:8077`. Then I'd send them the link. On their phone (web-00) they see "Get tailbond on this phone":
- step 1 installs the APK, with drawings of each Android prompt;
- step 2 is the **Open tailbond and ask homebox** button;
- step 3 turns tailbond on.

On my dashboard, 14c shows "pixel-7a wants to connect · Phone on your tailnet · signed in as sam@example.com · Pixel 7a · 100.101.7.7". I approve it, and 14d shows pixel-7a with "2 active path(s) · 40s". Done.

**What works:** the APK's SHA-256 is printed on the page, along with "the same as `sha256sum /usr/share/tailbond/tailbond.apk` on homebox". That's exactly how I'd want to verify a sideloaded APK. The page is upfront that the phone's traffic exits from my home ("like using it as an exit node") and that Tailscale's VPN is off while tailbond runs.

**Where I stumble:**

13. **Every family phone gets homebox's tailnet identity, and nothing narrows it per device.** The phone page (web-00) says: "They see this phone as homebox (it has no 100.x address of its own meanwhile), so they allow it whatever they allow homebox." The dashboard card says: "Once you approve their phone here, it reaches what this server reaches." homebox is a server, and in my ACLs it reaches a lot (backups, NAS, monitoring). sam's phone does not need any of that.

    The only remedy in the guide is to "tag the server and give the tag only what bonded devices should reach". That changes homebox's own access as well, removes its owner (which changes who can approve), and "only narrow[s] anything once your policy's default allow-all rule is removed". The 14c card for sam's phone repeats only the generic paragraph.

    I'd expect a way to limit what bonded devices reach, in tailbond itself (a per-device or all-devices allow-list in `tailbond-out`) or as a documented firewalld recipe for tailnet destinations. The guide does give one for the home LAN (`--policy=tailbond-out --add-rich-rule=… reject`), but none for the tailnet. **Medium.**

14. **It's unclear whether bonded devices reach my other subnet routers' LANs.** I run subnet routers. The pitch says "Your Tailscale machines stay reachable through it". §3 says only that traffic to "`100.64.0.0/10` and `fd7a:115c:a1e0::/48` leaves through `tailscale0` with masquerade". It doesn't say what happens to traffic for, e.g., `192.168.50.0/24` advertised by another subnet router that homebox accepts with `--accept-routes`. Is it reachable? Is it masqueraded? Does it leak out the WAN instead? For exposure planning I need to know if family phones can reach my parents' LAN. **Medium.**

15. **I can't tell whether my RPM includes the APK.** The guide says the app is offered "(when the package was built with it)" and "the RPM ships it when it was built with one". Before sending family a link, I'd check with `rpm -qlp tailbond-server-0.1.0-1.fc44.x86_64.rpm | grep apk`. The guide could just say so, or say what the phone page shows when there's no APK. **Low.**

## Task 5 — Read setup health and fix it, including the relay

**What I'd do:** the banner on every tab (14/14b/14c) reads "Setup health: 2 optional speed-ups (tailbond works fine without them). Show". That opens the Devices tab (15), where:
- "Tailscale: 2 machines direct, 1 through a relay (slower)… Through a relay: apple-tv… forward UDP port 41641 to this machine (192.168.1.2)."
- "This network makes direct Tailscale connections hard… Same fix as above (one forward fixes both)."

I'd add the UDP 41641 forward on my router and verify with `tailscale ping apple-tv`, which the guide says to use: "via DERP(…)" means relayed, "via <ip>:<port>" means direct. 15d then shows "Setup health: everything is set up": "Tailscale: 3 machines direct", "Network is friendly to direct connections", "UDP 443 reached from the internet · pixel-8 connected from 198.51.100.20, 2 h ago". 15b agrees (Peers online 3 / 4, route approved). This is a good, verifiable loop.

The health wording is good: it says that only traffic to apple-tv is slower, that bonding is unaffected, that the cause may be on the other machine's network, and that UPnP "lets any device on your network open ports". All accurate.

**Where I stumble:**

16. **The fix assumes tailscaled listens on 41641.** Neither the screen nor the guide says that the forward only helps if homebox's tailscaled is on 41641. On Fedora that's `PORT=` in `/etc/default/tailscaled`. Some of us change it, or set it to 0. One line saying "check that `PORT` in `/etc/default/tailscaled` is 41641" would do. **Low.**

17. **I can't follow the links.** "How to fix" (15) and the guide's "[setup tips](setup-tips.md)" point at a document I don't have. The guide's own table was enough for me to fix things, but I couldn't check what the links add. **Low.**

## Task 6 — Decide and set who may approve new devices

**What I'd do:** Devices tab → "Who can approve and add devices" (15c/15/15d). The current state reads: "alex@example.com (the Tailscale account homebox belongs to), from any device signed in to that Tailscale account, and any user or program on homebox itself (http://127.0.0.1:8077…)". Options: "Also allowed to approve (Tailscale logins, comma separated)", or "Or a password for this page (replaces the above; empty: none)".

My decision: only me. I wouldn't add sam. homebox runs containers and other services, so the local-access rule bothers me, and I'd set a password. web-04 shows how a tagged server is flagged ("Anyone on your Tailscale network can approve devices"), and the guide's advice to add approvers "**before** tagging it" is right.

**Where I stumble:**

18. **Any local process is an approver, and fixing that removes identity-based auth.** The card says "any user or program on homebox itself (http://127.0.0.1:8077; set a password if homebox runs things you don't trust)". On a homelab box, that includes every container with host networking and every unprivileged account. Any of them can approve a device, and that device then reaches my tailnet as homebox. That's a local-to-tailnet privilege escalation path.

    The fix offered is a password, which "replaces the above": I lose Tailscale-identity checking and must use the password everywhere, including the `curl -c jar … /api/login` dance in the guide. I'd expect to be able to keep identity-based approval and require a password (or disable access) only on 127.0.0.1, or to bind the local listener to a socket with permissions. As it stands I finish the task, but I have to give up the identity check to do it. **Medium.**

19. **Approvers must be individual logins.** The field takes "Tailscale logins, comma separated". My policy already has `group:admins` / `autogroup:admin`. I'd expect to be able to reuse them instead of keeping a second list in sync. **Low.**

20. **Where is the password stored?** The guide says `"ui_password"` in `tailbond.json`, which suggests plaintext in `/etc/tailbond/tailbond.json`. Neither the screen nor the guide states the file's permissions, or whether the dashboard's **Save** hashes it. **Low.**

---

## Findings

| # | Task | Finding | Severity |
|---|---|---|---|
| 1 | 1 | firewalld changes re-applied "every start, both the running and the saved configuration", including UDP 41641 in the WAN zone at install; "nothing to do here yourself" with no opt-out | medium |
| 2 | 1 | Uninstall leaves `10.77.0.0/24` advertised; manual `--advertise-routes=<your other routes…>` ("an empty value withdraws all of them") is risky for subnet routers | medium |
| 3 | 1 | "runs unconfined as root" on SELinux Fedora | low |
| 4 | 1 | "Tailscale drops those source addresses on any other interface" doesn't hold with `--netfilter-mode=off` | low |
| 5 | 1 | "answers them there only with the machine list": contents undefined, given to all bonded devices | low |
| 6 | 1 | `"landing_port": -1`: where it goes in the JSON and whether a restart is needed are not stated | low |
| 7 | 1/2 | Header "reachable at 192.168.1.2:443" contradicts "they only work at home" | low |
| 8 | all | Header "Connected" toggle on the server is unexplained | low |
| 9 | 2 | 15c "set "no_advertise_route": true in tailbond.json" leaves out the `"server"` key and the restart that the guide gives | medium |
| 10 | 2 | UDP 443 forward can only be verified after pairing a phone and going to mobile data | low |
| 11 | 3 | After "slate7 can connect now… in a few seconds" nothing appears; later shown "Not Connected" with a phone icon | low |
| 12 | 3 | "Decline only refuses this request; it can ask again": no way to block repeat requests | low |
| 13 | 4 | Phones get homebox's identity ("they allow it whatever they allow homebox"); no per-device or tailnet-destination limit, only re-tagging the server | medium |
| 14 | 4 | Unclear whether bonded devices reach other subnet routers' LANs; guide only covers 100.64.0.0/10 and fd7a | medium |
| 15 | 4 | "when the package was built with it": no way given to check the RPM for the APK | low |
| 16 | 5 | UDP 41641 fix assumes tailscaled's default port; `/etc/default/tailscaled` not mentioned | low |
| 17 | 5 | "How to fix" and `setup-tips.md` not available to check | low |
| 18 | 6 | "any user or program on homebox itself" can approve; the only fix (password) "replaces the above" identity rules | medium |
| 19 | 6 | Approvers are individual logins only; can't reuse ACL groups | low |
| 20 | 6 | `"ui_password"` storage and file permissions not stated | low |

Totals: 0 high / 6 medium / 14 low
