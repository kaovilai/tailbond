# Round 10 — Fedora persona F2 (home admin)

**Persona.** I look after the family's tech. Our "server" is a spare PC running Fedora. I can install things with `dnf` and paste commands from a guide. I set up Tailscale by following its website. I don't really understand firewalls, NAT or port forwarding. I know my router has an admin page and I've clicked around in it before.

**Materials used.** `task-script.md`, `screens.md`, `fedora-server.md` and the PNGs in `glinet/` and `web/`. Nothing else.

---

## Task 1: Install tailbond on homebox and open its dashboard

**What I'd do.** Guide §1 "Install": I'd paste `sudo dnf install ./tailbond-server-*.rpm` and then `sudo systemctl enable --now tailbond`. The file I was given is `...fc44...` and the guide says "Fedora 43 or newer", so that fits. I'd check it with `systemctl status tailbond`. Then I'd open <http://127.0.0.1:8077/> on homebox itself, or `http://homebox:8077` from my laptop. The guide says the dashboard "answers only devices of the Tailscale account this server belongs to", which is me, so I expect it to work. I'd land on the dashboard shown in 15c.

**Where I hesitated.**
- §1 says about firewalld: "There's nothing to do here yourself; this list is for checking". Good, I'd stop worrying. But three bullets later there's a copy-paste block: `sudo firewall-cmd --permanent --policy=tailbond-out --add-rich-rule='rule family=ipv4 destination address=192.168.1.0/24 reject'`. I paste whatever command blocks a guide gives me. I'd have to stop and work out whether this one is a step. It isn't, it's optional ("To keep them off your home LAN, add a rule of your own"), and the subnet is an example that may not match mine. I'd expect an optional, home-specific command to sit under a clearly marked "Optional" heading, not inside a list I was told needs no action. **Low.**
- The HTTPS / `tailscale serve` paragraph is above my level, but it says "It isn't needed", so I'd skip it. Fine.

No blocker. The install part is short and clear.

---

## Task 2: Right after installing (15c): what's left, and how?

**What I see (15c, Devices tab).**
- At the top, an orange card: "Setup step: tell devices where home is … Right now devices would connect to 192.168.1.2:443, an address inside your home, so they only work at home. Two things, on your home router's admin page: 1. Forward UDP port 443 to this machine (192.168.1.2) … Give this machine a fixed address there too (DHCP reservation) … 2. Find your home's public address … Type it here:" followed by a box and **Save**.
- Under it: "Setup health: all set (1 optional item below)", with "Route 10.77.0.0/24 isn't approved yet (optional)".
- Then the "Who can approve", "Add an Android phone", "Add a GL.iNet travel router" and "Add a device with a code" cards, and "No devices yet".

**What's left, as I understand it.**
1. In my router's admin page: set up port forwarding for UDP 443 to 192.168.1.2, and a DHCP reservation for homebox. The card names the menu ("Port forwarding, Virtual server or NAT"), which helps a lot.
2. Read the WAN IP off the router's status page (or a DDNS name), type it into the box and click **Save**. "The port (443) is added for you" tells me not to type `:443`. Good.
3. Optionally approve the route in the Tailscale admin console. The text says "Devices connected through tailbond reach your Tailscale machines without it", so I'd skip it for now.

**Where I'd get confused.**
- **"Setup health: all set"** sits right under an orange box telling me devices "only work at home". To me "all set" means I'm done. Someone skimming (me, on a busy evening) could skip the port forward because the health card is green. I'd expect the health card to say something like "1 thing to do (see above)", or at least not "all set" while the main setup step is still open. **Medium.**
- **Testing the forward.** Guide §2 step 3: "Test the forward from outside right away: on a phone with tailbond, turn Wi-Fi off; once it connects over mobile data, the dashboard's *UDP 443* health item turns green." At this point I have no phone with tailbond (that's Task 4). And 15c has no "UDP 443" health item at all. The guide's table lists "UDP 443 not confirmed from outside yet (?)", but the card shows only the route item. I don't know any other way to test a port forward, so I'd be stuck wondering whether I got it right until much later. I expected either a test I can run now, or the guide to say "you'll confirm this in §4 when you add the first phone". **Medium.**
- The header says "Home server · reachable at 192.168.1.2:443", but the card below says it's *not* reachable from outside. "Reachable" isn't the right word for an inside-only address. **Low.**

---

## Task 3: A family member's GL.iNet travel router is asking to connect. Approve it.

**What I'd do.** 14 (desktop) / 14-phone: the "Waiting for your approval" card is at the very top: "slate7 wants to connect · Travel router on your tailnet · signed in as alex@example.com · GL-BE3600 · 100.101.9.9", with **Decline** and **Approve**. I'd click **Approve**. On the phone view the buttons are large and easy to tap. 14b shows the toast "slate7 can connect now. It connects by itself in a few seconds."

**Where I'd get confused.**
- **Did it actually connect?** The toast promises "It connects by itself in a few seconds". But in 14b and in the later 14d, the Connections tab lists only pixel-8 (and later pixel-7a). slate7 never shows up there. In 15d's Devices list, slate7 is shown as "offline", with no reason given. I can't tell whether it's broken, the router is switched off, or my port forward is wrong. As the family's admin I'd have to phone them and ask. I expected the router to appear under Connections, or the "offline" row to say why (for example "hasn't connected since approval"). **Medium.**
- The approval card puts a long grey paragraph before the decision. It ends "reach your Tailscale machines as homebox, under homebox's access rules." I don't know what "homebox's access rules" are in practice (the guide calls them ACLs), so I'd click Approve without understanding what I'm granting. A short plain line such as "This device will be able to reach everything homebox can reach" would land better. **Low.**
- In 15d, slate7 (a travel router) has the same phone icon as pixel-8 and ipad. **Low.**

---

## Task 4: Get tailbond onto a family member's Android phone and get it connected

**What I'd do.**
1. Devices tab → "Add an Android phone" card (15c/15). I'd click the copy button next to `http://homebox.tail1234.ts.net/` and text it to them, or tell them to type `homebox/`.
2. On their phone (web-00): "Download tailbond (27 MB)", then go through the drawn Android prompts ("Download anyway", "Settings", "Allow from this source", "Install", and maybe "Scan app"). The drawings of each prompt are excellent. This is exactly where family members normally phone me in a panic. "Stuck, or not comfortable with this? Ask whoever runs homebox to install it for you." is also nice.
3. They tap **Open tailbond and ask homebox**. On my dashboard (14c), "pixel-7a wants to connect · Phone on your tailnet · signed in as sam@example.com · Pixel 7a", and I click **Approve**.
4. They follow phone page step 3: "tap Add phone. Then tap the big switch … Android asks once to allow its connection: tap OK." In 14d, pixel-7a shows in Connections with 2 active paths. Done.

**Where I'd get stuck or confused.**
- **The prerequisite: invite or share?** The card says "their phone needs Tailscale signed in to your Tailscale network (invite them in the Tailscale admin console, or share this machine with their account)". The task says they "have Tailscale on it", probably on their own account. Nothing tells me which of the two to pick for family, how to do either, or what the difference means: does inviting them let them see all my machines? Does sharing let them reach only homebox? I'd have to go back to Tailscale's website and guess. The "Add a device with a code" card adds to the doubt: "Routers and phones on your **Tailscale account** don't need this". Sam's phone is on Sam's account, so do they need a code after all? I expected one plain recommendation ("For family, share homebox with their Tailscale account: admin console → Machines → homebox → Share"), or a link to how to do it. **Medium.**
- **How do I know a request is waiting?** The phone page tells the family member "Whoever runs homebox taps Approve on homebox's page; you'll see it in the app right away." Nothing tells *me*. I'd have to be looking at the dashboard at that moment, so it needs a phone call to coordinate ("tap it now… OK, now I'm approving"). Workable for family, but extra effort. **Low.**
- In 15d's Devices list (after the health fixes), pixel-7a isn't listed, though it was connected in 14d. Only pixel-8, ipad and slate7 appear. This may just be a different moment in time, but if I saw it I'd wonder whether the approval stuck. **Low.**
- The phone page on my computer (web-00c) says "Anyone on your Tailscale account can open it", while the dashboard card says people you *share* homebox with can use it too. The word is "account" in one place and "network" in the other. Small, but it feeds the confusion in the first point. **Low.**

---

## Task 5: Read setup health and fix it, including the machine "through a relay"

**What I'd do.** On 14/14b/15b there's a banner "Setup health: 2 things to improve (tailbond works meanwhile). Show". The Devices tab (15) shows:

- "Tailscale: 2 machines direct, 1 through a relay (slower)". "Through a relay: apple-tv. Only traffic between this machine and that machine is slower (including bonded devices reaching it); bonding itself doesn't go through Tailscale. … On your home router's admin page (Port forwarding, also called Virtual server or NAT), forward UDP port 41641 to this machine (192.168.1.2). (Turning on UPnP on that router also works, but lets any device on your network open ports.)"
- "This network makes direct Tailscale connections hard". "The router in front of this machine picks a new outside port for every destination ("hard NAT"), so other machines can't connect straight back; no port mapping on the router. Same fix as above (one forward fixes both)."

So I'd go back to the same router page as in Task 2, add a second forward (UDP 41641 to 192.168.1.2) and skip UPnP. "tailbond works meanwhile" and "only traffic to that machine is slower" tell me this is not urgent, which I appreciate. The card gives the exact port and address, which is what someone like me needs. Afterwards, 15d shows "Setup health: everything is set up for best speed": "Tailscale: 3 machines direct", "Network is friendly to direct connections", "UDP 443 reached from the internet — pixel-8 connected from 198.51.100.20, 2 h ago". That's a clear finish line, and it also answers my Task 2 question about testing UDP 443.

**Where I'd hesitate.**
- "no port mapping on the router" in the hard-NAT item. I don't know that "port mapping" and "port forward" are related, so on first read it sounds like an instruction ("no port mapping!") right before "Same fix as above (one forward fixes both)". The last line rescues it. I'd expect "…because nothing on the router is forwarded to this machine yet". **Low.**
- "The cause can also be on that machine's own network." I don't know whether apple-tv is at home or at grandma's, so I can't tell whether my forward will fix it. There's also no "check again" button or guidance on how long to wait. The guide's `tailscale ping <machine>` tip isn't on the card. 15d shows that it eventually resolved, so I finish. **Low.**

---

## Task 6: Decide and set who may approve new devices

**What I see (15c/15/15d).** The card "Who can approve and add devices" says: "Who can approve: alex@example.com (the Tailscale account homebox belongs to), from any device signed in to that Tailscale account, and anyone logged in on homebox itself (http://127.0.0.1:8077). Others on your Tailscale network, and people you share homebox with, can ask to connect but can't approve. A device you lend to someone while it's signed in to an approver's account can approve too. To let someone else approve, add their Tailscale login (shown in the Tailscale admin console under Users) below. A password instead replaces all this …". Below it are "Also allowed to approve (Tailscale logins, comma separated)" with **Save**, and "Or a password for this page (replaces the above; empty: none)" with **Save**.

**My decision.** I'd add my partner, `sam@example.com`, under "Also allowed to approve" and click **Save**. I wouldn't use a password, because then I'd have to type it every time too, and the family wouldn't remember it.

**Where I'd get confused.**
- **The router I set up is signed in as me.** In Task 3, slate7 shows "signed in as alex@example.com". I set up the family member's router with my own Tailscale login, as most home admins would. The card's generic line "A device you lend to someone while it's signed in to an approver's account can approve too" tells me that this family member can now approve devices and reach my dashboard. Nothing on screen points out that slate7 is such a device. Nothing on screen says how to fix it either. The guide does ("sign those in with the borrower's account, or tag them"), but "tag them" means nothing to me. So my decision about who may approve isn't really mine: anyone holding a device logged in as me is in. I expected the approval card or the device row to flag "this device is signed in as you, so it can approve too", with a plain fix. **Medium.**
- "(replaces the above; empty: none)" is terse. I'd guess "save it empty to remove the password", but I'm not sure. There's also no screenshot of the card after saving an approver (does `sam@example.com` then appear in the "Who can approve:" sentence?), so I can't confirm my change took. **Low.**
- "anyone logged in on homebox itself": the spare PC has a kids' login. The guide says "set a password if this machine has users you don't trust with that", but the card doesn't, and that is exactly when I'd want the password option. **Low.**
- web-04 (a tagged server) shows, inside the "Who can approve and add devices" card: "Name approvers or set a password in the "Who can approve and add devices" card on the Devices tab", which points to the very card I'm reading. On the same screen, the health card says "Fixes are on your home router or in the Tailscale admin console", but this fix is right here on the dashboard. **Low.**

---

## General polish

- The data line "7.5 GB to · 858 MB from devices" reads as if words are missing. "7.5 GB to devices · 858 MB from devices" would be clearer. **Low.**
- Phone view (14-phone): the header is cut off ("reachable at home.…") and the device name wraps as "pixel- / 8". **Low.**

---

## Findings

| # | Task | Finding | Severity |
|---|---|---|---|
| 1 | 2 | "Setup health: all set (1 optional item below)" shows right under the orange "Setup step" that says devices "only work at home"; "all set" invites skipping the port forward | medium |
| 2 | 2 | Guide §2.3 "Test the forward from outside right away: on a phone with tailbond…": no phone exists yet, and 15c has no "UDP 443" health item to watch, so there's no way to check the forward now | medium |
| 3 | 3 | Toast "slate7 can connect now. It connects by itself in a few seconds." but slate7 never appears under Connections (14b/14d), and 15d lists it as "offline" with no reason | medium |
| 4 | 4 | "invite them in the Tailscale admin console, or share this machine with their account": no advice on which to choose for family, how to do it, or what each exposes; the code card's "on your Tailscale account" adds doubt | medium |
| 5 | 6 | The family's router is "signed in as alex@example.com", so it can approve too; the card only says "A device you lend to someone … can approve too", doesn't flag slate7, and gives no on-screen fix | medium |
| 6 | 1 | Guide §1 says "There's nothing to do here yourself", then shows a copy-paste `firewall-cmd … 192.168.1.0/24 reject` block (optional, example subnet) | low |
| 7 | 2 | Header "reachable at 192.168.1.2:443" while the setup step says it's only reachable at home | low |
| 8 | 3 | Approval text "reach your Tailscale machines as homebox, under homebox's access rules" is jargon; it doesn't say plainly what is granted | low |
| 9 | 3 | slate7 (travel router) shows a phone icon in the Devices list (15d) | low |
| 10 | 4 | No notification of a pending request; the phone page says "Whoever runs homebox taps Approve", so the admin must be watching or be phoned | low |
| 11 | 4 | pixel-7a is connected in 14d but missing from the 15d Devices list | low |
| 12 | 4 | Phone page "Anyone on your Tailscale account can open it" vs. the dashboard's "Tailscale network … or share this machine": inconsistent wording | low |
| 13 | 5 | "no port mapping on the router" reads like an instruction next to "forward UDP port 41641" | low |
| 14 | 5 | "The cause can also be on that machine's own network": can't tell if my fix applies to apple-tv; no re-check button or wait guidance, and the `tailscale ping` tip isn't on screen | low |
| 15 | 6 | "(replaces the above; empty: none)" is cryptic; no screen confirms a saved approver | low |
| 16 | 6 | "anyone logged in on homebox itself": the card doesn't say when a password is the better choice (shared PC logins), though the guide does | low |
| 17 | 6 | web-04: the "Who can approve…" card tells you to go to the "Who can approve and add devices" card (itself); the health card says fixes are on the router or in the admin console, but this one is on the dashboard | low |
| 18 | all | "7.5 GB to · 858 MB from devices" reads as if words are missing | low |
| 19 | all | Phone view: header truncated "reachable at home.…", device name wraps "pixel- / 8" | low |

Totals: 0 high / 5 medium / 14 low
