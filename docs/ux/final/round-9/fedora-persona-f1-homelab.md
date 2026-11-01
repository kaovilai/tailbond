# Round 9 — Fedora persona F1 (homelab admin)

**Who I am:** I run Fedora servers with firewalld and SELinux enforcing. I use Tailscale with subnet routers and a hand-written ACL policy. I read install guides critically. I want to know what gets exposed to whom, on which ports, and how to verify each claim from a shell.

**Materials:** `task-script.md`, `screens.md`, `fedora-server.md`, and the PNGs in `glinet/` and `web/`. Nothing else.

**Overall:** This is a good guide for my kind of reader. It lists every firewalld change with the `firewall-cmd` command to check it. It explains the `tailscale serve` trap on port 8077, explains what an approved device reaches, and gives a `grants` example. The dashboard's request cards show the Tailscale identity, model and IP, which I can cross-check with `tailscale status`. I finished every task. My problems are all about exposure and verification: what stays open after uninstall, what the route approval exposes, how far family devices reach into my LAN, and who exactly counts as an approver.

---

## Task 1: Install and open the dashboard

**What I'd do:**
- Guide §1: `sudo dnf install ./tailbond-server-*.rpm`, then `sudo systemctl enable --now tailbond`, then check with `systemctl status tailbond` and `journalctl -u tailbond -f`.
- Audit firewalld with the guide's commands: `sudo firewall-cmd --get-active-zones`, `--zone=<zone> --list-services`, `--list-rich-rules`, `--info-policy=tailbond-out`.
- Open `http://homebox:8077` from my laptop on the tailnet. That gives me screen 15c.

**What works:** The package table and the "What the package changes in firewalld" list are exactly what I want to read before installing on a server. Each item is scoped: "tcp 8077 and 80 from tailnet addresses only (`100.64.0.0/10`, `fd7a:115c:a1e0::/48`)". The warning not to point `tailscale serve` at 8077 is a real, non-obvious detail, and I appreciated it.

**Where I'd stop and frown:**

1. **Runtime or permanent, and what uninstall leaves open (medium).**
   - The guide says the hook changes firewalld "(the `server-up` hook, every start)". It never says whether those changes are runtime-only (`server-down` reverts them) or written into my permanent zone config.
   - The **Uninstall** section mentions only one leftover: "The tunnel route stays advertised in Tailscale."
   - After `dnf remove`, I can't tell whether UDP 443 and UDP 41641 are still open in my internet-facing zone, or whether the `tailbond` zone and policy remain. I'd have to diff `/etc/firewalld/` before and after to find out.
   - I expected one sentence like "runtime only; `systemctl stop tailbond` removes them", or an uninstall step listing the firewalld leftovers.
2. **SELinux claim (low).** "SELinux needs no changes: the service runs unconfined as root, like other VPN servers."
   - This is a root daemon that serves an HTTP admin API on the tailnet and edits my firewall. "No changes needed" is true, but it isn't reassuring.
   - "Like other VPN servers" isn't accurate: OpenVPN runs confined (`openvpn_t`) on Fedora.
   - I'd prefer an honest "no policy module yet", plus whatever systemd hardening the unit has.
3. **UDP 41641 opened automatically (low).** "UDP 41641 (Tailscale's own port) in the same zone as UDP 443 when Tailscale is installed."
   - The package opens a second port in my WAN-facing zone without asking. It's harmless and it's documented.
   - No opt-out is shown, unlike `"landing_port": -1` for the phone page.
4. **No package verification (low).** §1 goes straight to `dnf install ./tailbond-server-*.rpm`. There is no checksum or signing key for the RPM itself. The phone page, by contrast, gives a SHA-256 for the APK.

## Task 2: Right after install (15c): what's left, and how?

**What I see on 15c:**
- An orange card, "Setup step: tell devices where home is", with the UDP 443 forward and a public-address field with **Save**.
- A "Setup health: 1 thing to improve for best speed" card with "Route 10.77.0.0/24 isn't approved yet".

So two things are left:
- Forward UDP 443 on my router to 192.168.1.2 and save `home.example.net` in the field (guide §2 matches the card step for step).
- Approve the route in the admin console. Since I write ACLs, I'd probably add an `autoApprovers` entry instead; the guide's health table mentions that option.

**Problems:**

5. **The route approval is presented as a speed tweak (medium).**
   - The card heading is "1 thing to improve for best speed (tailbond works meanwhile)", and the item is "Route 10.77.0.0/24 isn't approved yet / Approve it once in the Tailscale admin console".
   - Approving that route is an **exposure** decision, not a speed one. It lets tailnet machines reach bonded devices. The guide's table says so ("Tailscale machines can't reach bonded devices"), but the dashboard doesn't.
   - With a permissive ACL, approving it puts every family phone and travel router in reach of every tailnet member. The card nudges me to approve it with no word about that.
   - The `server-up` hook re-advertises the route "every start". If I decide **not** to approve it, nothing shown lets me turn it off, so I expect the card to nag forever.
   - I expected the item to say what approving opens, and offer a way to say "don't advertise".
6. **No way to test the UDP 443 forward before a device needs it (medium).**
   - The guide's health table says: "UDP 443 not confirmed from outside yet … none: the first device connecting from away confirms it."
   - Troubleshooting says: "check the port forward from outside your network (for example from the phone on 5G)". It never says *how*. Neither the dashboard nor the guide gives a self-test or a command to run from outside (UDP/QUIC can't be checked with `nc` the way TCP can).
   - I'd like to know the forward works before I hand a family member a router. Instead, my first test is that person's device failing to connect.
7. **"reachable at" a LAN address (low).** The 15c header says "Home server · reachable at 192.168.1.2:443" while the card below explains that this address only works inside my home. The header claims something the setup step contradicts. "(home only)" or "not reachable from away yet" would be accurate.
8. **What does the server's toggle do? (low)** The header shows "Connected" with an on toggle, and the stats card has a "Connected" pill, on a server with zero devices. The guide never mentions the toggle. I can't tell what it controls (mqvpn? the whole service? all devices?) or whether switching it off drops every device.

## Task 3: Approve the family member's GL.iNet router

**What I'd do:**
- Screen 14 shows the "Waiting for your approval" card at the top: "slate7 wants to connect · Travel router on your tailnet · signed in as alex@example.com · GL-BE3600 · 100.101.9.9".
- I'd cross-check `100.101.9.9` with `tailscale status`, then click **Approve**.
- On 14b, the toast says "slate7 can connect now. It connects by itself in a few seconds."

The card text explains what approval grants: "Approving lets the device use homebox's internet connection … and your home network, and reach your Tailscale machines as homebox, under homebox's access rules." That's the right warning in the right place.

9. **No confirmation that the router actually connected (low).**
   - After the toast, slate7 never appears on the Connections tab, either on 14b or later on 14d.
   - The only other trace is the Devices tab on 15d: "slate7 · offline".
   - I can't tell whether it connected and dropped, never tried, or is failing the handshake. I'd go to `journalctl -u tailbond` and look for `client authenticated`, which the guide mentions under Troubleshooting.
   - An "approved, waiting for first connection" state on the device row would answer this.

## Task 4: The family member's Android phone

**What I'd do:**
- Devices → **Add an Android phone** (15c): copy `http://homebox.tail1234.ts.net/` with the copy button and send it to them.
- First I'd check that their account can reach homebox on tcp 80 and 8077 in my policy. The guide is explicit: "allowed to reach this machine on tcp 80 and 8077". I'd use its `grants` snippet as a model.
- On their phone (web-00): download, follow the drawn install prompts, then tap "Open tailbond and ask homebox".
- 14c: "pixel-7a wants to connect · Phone on your tailnet · signed in as sam@example.com · Pixel 7a · 100.101.7.7". I click **Approve**.
- 14d: pixel-7a appears with "2 active path(s)". Done.

**What works:**
- The phone page's "Is this safe?" paragraph gives the APK's SHA-256 and the exact `sha256sum /usr/share/tailbond/tailbond.apk` to compare on homebox. That's exactly the verification hook I want.
- The phone page says plainly that the page "changes nothing" on the server.

**Problems:**

10. **Family devices get my whole LAN, with no shown way to limit it (medium).**
    - Guide: "What an approved device reaches: the internet through your home connection, your home LAN, and your tailnet."
    - The only limiting advice is to tag the server, and that only limits the **tailnet** side.
    - For the home LAN (my NAS, IPMI, IoT VLAN), nothing explains how to restrict a family phone.
    - Because the hook re-applies firewalld "every start", I can't tell whether editing the `tailbond-out` policy myself would survive a restart.
    - I expected either a setting ("bonded devices may reach: internet only / + LAN / + tailnet") or a documented supported way to customize `tailbond-out`.
11. **Approving sam's phone means sam acts as my server (low).**
    - The 14c card says the device will "reach your Tailscale machines as homebox, under homebox's access rules". That's accurate.
    - But sam's request shows "signed in as sam@example.com". Once approved, sam's phone has Tailscale *off* (phone page: "while tailbond is on, Tailscale's VPN is off on this phone"). It then reaches the tailnet with **homebox's** (alex's server's) access, not sam's own.
    - Someone who skims will miss that this can widen sam's access. A line saying the request is from a *different* user than the server owner, with this consequence, would help.
12. **Stale toast covers the new device (low).** 14d says "pixel-7a can connect now. It connects by itself in a few seconds." But pixel-7a is already listed as connected for 40s, and the toast covers its second path card.
13. **Path cards don't say which link is which (low).**
    - Connections shows "Path 0 / Pixel-8 · Min 31 Ms" and "Path 1". The Devices row shows "198.51.100.20:51234 · wlb" and "#0 active · 31 ms · 0.0%".
    - I can't tell which path is Wi-Fi and which is 5G.
    - "Min 31 Ms" has odd capitalization, and I'm not sure "Min" means minimum RTT.
    - "wlb" isn't explained anywhere in the guide. My guess is a mode; the guide's modes are named Speed, Streaming, Redundant, Failover.
14. **Two different "add a phone" flows (low).**
    - My own phone opening the dashboard (14-phone) shows "Add this phone … Two steps … No code to scan", with a "Name for this phone" field and "+ Add this phone".
    - The phone page (web-00) shows three steps and "Open tailbond and ask homebox".
    - The guide describes only the second. I'd wonder whether the dashboard version skips approval, or skips the identity check. It's probably fine because I'm already an approver, but it's undocumented.

## Task 5: Setup health, including the machine "through a relay"

**What I'd do** (Devices tab, 15):
- "Tailscale: 2 machines direct, 1 through a relay (slower) / Through a relay: apple-tv … To check: tailscale ping apple-tv ("via DERP" means relayed). On the router in front of this machine, forward UDP port 41641 to it."
- "This network makes direct connections hard … Same fix as above (one forward fixes both)."
- So: one UDP 41641 forward on the router to 192.168.1.2. Then `tailscale ping apple-tv` until it says "via <ip>:<port>".
- I'd confirm with `firewall-cmd --list-services` that 41641 is allowed. The card says "This machine's own firewall is already set up by tailbond", and guide §1 backs that up.
- 15d then shows everything green: "Tailscale: 3 machines direct", "UDP 443 reached from the internet / pixel-8 connected from 198.51.100.20, 2 h ago".

This is the best part of the product for me:
- It tells me the scope (only traffic to *that* machine is slower, and bonding isn't affected).
- It gives the shell command to verify.
- It warns about UPnP ("lets any device on your network open ports").
- The Tailscale tab (15b) agrees with it: "PEERS ONLINE 3 / 4", "10.77.0.0/24 · approved".

15. **"no port mapping on the router" after I added a forward (low).** On 15d, after I forwarded UDP 41641, the green item says "Network is friendly to direct connections / no port mapping on the router". I'd briefly think my forward wasn't detected. I assume "port mapping" means UPnP/NAT-PMP/PCP, but a reader who just made a *port forward* will read it as "no forward". Something like "no UPnP/NAT-PMP" would be clearer.
16. **Broken reference in the packet (low).** The guide says "For best speed … see [setup tips](setup-tips.md)". That file isn't part of what I was given. The health table carries the content, so I didn't need it.

## Task 6: Decide and set who may approve

**What I'd read:**
- The "Who can approve and add devices" card: "Who can approve: alex@example.com (the Tailscale account homebox belongs to), from any of their devices, and anyone using homebox itself. Others on your Tailscale network, and people you share homebox with, can ask to connect but can't approve."
- Below it are "Also allowed to approve (Tailscale logins, comma separated)" and "Or a password for this page (replaces the above; empty: none)".
- The guide's "Who can approve" section matches.

**My decision:** Keep identity-based approval (no password), leave the approvers list empty, and don't add sam. If I later tag the server, add only `alex@example.com` as an approver at the same moment.

17. **"anyone using homebox itself" is ambiguous where it matters (medium).**
    - The same card stack says "Approving lets the device **use** homebox's internet connection". So does a bonded device "use homebox"?
    - The guide resolves the wording ("anyone logged in on this machine (`http://127.0.0.1:8077`)"). That means **any local user**, including unprivileged accounts, can approve devices and change the config of a root service. The guide never calls that out.
    - The guide also says "bonded devices may use this machine's services on its tunnel address (`10.77.0.1` …)". It doesn't say whether that includes 8077, or which identity such a request would have.
    - I'd want an explicit statement, e.g. "bonded devices can't reach the dashboard" and "local = anyone who can open a socket to 127.0.0.1".
18. **"from any of their devices" covers a device I gave away (medium).**
    - In 14, the family member's travel router is "signed in as alex@example.com". By the card's own rule, a request that Tailscale attributes to that router counts as an approver.
    - The card doesn't list *which* devices that is. The guide doesn't suggest tagging or re-logging devices you hand to other people.
    - I can't decide with confidence whether this is acceptable. I'd probably tag slate7 or set a password to be safe, but the UI didn't lead me there.
19. **The guide's own hardening step opens the dashboard to the whole tailnet (medium).**
    - The guide recommends tagging the server and gives a grant of `autogroup:member → tag:tailbond` on `tcp:8077`.
    - Once the server is tagged, the card (web-04) changes to "Who can approve: anyone on your Tailscale network who can reach homebox (it's a tagged machine, so it has no single owner)."
    - Follow the example literally and every member can approve devices and change settings until approvers are named. The dashboard shows this only in small grey text. It isn't a Setup health item and it isn't in orange.
    - The guide does add "name its approvers (or set a password)", but *after* the grant example.
    - I expected a red health item for "tagged and no approvers/password", and the guide to say "name approvers *before* tagging".
20. **Self-referencing pointer (low).** The tagged-state text (web-04) says "Name approvers or set a password under Devices → Who can approve", while I'm already reading that card on the Devices tab. "Below" would do.

---

## Findings

| # | Task | Finding | Severity |
|---|---|---|---|
| 1 | 1 | Firewalld changes: runtime vs permanent never stated; Uninstall lists only "The tunnel route stays advertised", so I can't tell whether UDP 443/41641 stay open after `dnf remove` | medium |
| 2 | 1 | "SELinux needs no changes: the service runs unconfined as root, like other VPN servers": root HTTP admin API, unconfined; comparison isn't accurate for Fedora's confined OpenVPN | low |
| 3 | 1 | UDP 41641 opened in the WAN-facing zone automatically "when Tailscale is installed"; no opt-out shown | low |
| 4 | 1 | No checksum or signature for the RPM (the APK gets one) | low |
| 5 | 2 | "Route 10.77.0.0/24 isn't approved yet" listed under "to improve for best speed", but it's an exposure decision (tailnet reaches bonded devices); no opt-out from the hook re-advertising it "every start" | medium |
| 6 | 2 | No way to verify the UDP 443 forward before a device connects: "the first device connecting from away confirms it"; Troubleshooting "check the port forward from outside" gives no method | medium |
| 7 | 2 | Header "reachable at 192.168.1.2:443" contradicts the setup step ("they only work at home") | low |
| 8 | 2 | Server header "Connected" toggle unexplained (what it stops, for whom) | low |
| 9 | 3 | After Approve, only a toast; slate7 never shows on Connections (14b/14d) and later just "offline" (15d), so I can't tell if it connected | low |
| 10 | 4 | Approved devices reach "your home LAN" with no shown way to limit that; tagging only limits the tailnet; unclear whether `tailbond-out` edits survive the hook re-applying "every start" | medium |
| 11 | 4 | sam's phone approved "as homebox, under homebox's access rules": doesn't flag that this can exceed sam's own ACL access | low |
| 12 | 4 | Stale toast "pixel-7a can connect now…" while already connected 40s, covering its Path 1 card (14d) | low |
| 13 | 4 | "Path 0 / Path 1", "Min 31 Ms", "wlb": no Wi-Fi vs 5G label; "wlb" unexplained | low |
| 14 | 4 | Dashboard-on-phone "Add this phone … No code to scan" (14-phone) is a different flow from the phone page's "Open tailbond and ask homebox"; approval behavior undocumented | low |
| 15 | 5 | After forwarding 41641, 15d says "no port mapping on the router": reads as "forward not detected" | low |
| 16 | 5 | Guide links "[setup tips](setup-tips.md)", which isn't in the packet | low |
| 17 | 6 | "anyone using homebox itself" ambiguous next to "use homebox's internet connection"; guide's "anyone logged in on this machine" means any local user can approve on a root service; unclear whether bonded devices reach 8077 via 10.77.0.1 | medium |
| 18 | 6 | "from any of their devices" includes the family's slate7 "signed in as alex@example.com"; the card doesn't list which devices count, and there's no guidance for devices handed to others | medium |
| 19 | 6 | Tagging (the guide's hardening) makes "anyone on your Tailscale network who can reach homebox" an approver, and the grant example opens tcp:8077 to `autogroup:member`; flagged only in grey text, not in Setup health | medium |
| 20 | 6 | Tagged-state text says "under Devices → Who can approve" while I'm already in that card | low |

Totals: 0 high / 7 medium / 13 low
