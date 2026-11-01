# Round 9 — Fedora persona F2 (home admin)

**Persona:** home admin, not a networking person. Fedora 44 spare PC (`homebox`) is the family's home server. I can install packages with dnf and copy commands from a guide, and I set up Tailscale by following its website. I don't really understand firewalls, NAT or port forwarding. I set things up for the family.

**Materials used:** `task-script.md`, `screens.md`, `fedora-server.md` and the PNGs in `glinet/` and `web/`. Nothing else.

---

## Task 1: Install tailbond on homebox and open its dashboard

**What I'd do**

- Guide section 1: `cd` to where the RPM is, then `sudo dnf install ./tailbond-server-*.rpm` and `sudo systemctl enable --now tailbond`. Then `systemctl status tailbond` to check it.
- Open the dashboard. homebox is a spare PC and may not have a screen, so I'd use the guide's other option, `http://homebox:8077` from my laptop on Tailscale. That gives screen 15c.

**Where I'd hesitate**

- The rest of section 1 is mostly firewall detail: "firewalld `tailbond` service, zone and policies", "`--list-rich-rules`", "`tb0` in the `tailbond` zone, and the `tailbond-out` policy". I can't tell which of it I need to act on. I worked out that it describes what the package does for me, so I skipped it. One sentence like "nothing to do here, the package sets the firewall up" would have saved me reading it. The line "the one your router's port forward arrives on" refers to a port forward I haven't made yet, which made me think I'd missed a step. **Low.**
- 15c, header: "Home server · reachable at 192.168.1.2:443" with a green **Connected** switch in the top-right. The orange card below says devices "only work at home", so "reachable" looks wrong to me. Nothing is connected yet, so I don't know what "Connected" means here. Nothing says what turning the switch off would do (cut off the family?). I'd leave it alone. **Low.**

Task completed with no real trouble.

---

## Task 2: Right after installing (15c), what's left to set up, and how?

**What I'd do**

The orange card at the top of the Devices tab, "Setup step: tell devices where home is", is clear about what to do:

1. "Forward UDP port 443 to this machine (192.168.1.2) … Port forwarding, Virtual server or NAT. Give this machine a fixed address there too (DHCP reservation)". I'd find that page on my router by following those words. Giving me the LAN address helps a lot.
2. "Find your home's public address: the router's status page shows it (WAN or Internet IP) … Type it here". I'd type it in and click **Save**.

Then the "Setup health" card: "Route 10.77.0.0/24 isn't approved yet. Approve it once in the Tailscale admin console (Machines → this machine → Edit route settings)." I've used the Tailscale admin console before, so I'd find homebox and tick the route.

**Where I'd get stuck or misunderstand**

- **Carrier-grade NAT check.** Step 2 says to copy the WAN IP from the router's status page into the box. The small print under it says: "If the router's own WAN address starts with 100.64–100.127, or is 10.x / 192.168.x **while your public address is different**, your provider shares one address…". The only place I've been told to find my "public address" is that same router status page, so I have nothing to compare it with. If my provider uses CGNAT, I'd type the 100.x address into the box and click Save without noticing anything wrong. Neither the card nor the guide says how to find the real public address (for example, a "what is my IP" site). The guide says "The dashboard's setup health detects this", but the card doesn't tell me that, so I wouldn't know to look there. **Medium.**
- The health header says "1 thing to improve **for best speed**", but the only item is the route. Guide section 3 says the route is about "Tailscale machines can't reach bonded devices". That isn't about speed, so I couldn't tell how important it was. **Low.**
- I'll be in the router's admin page anyway to forward UDP 443, but nothing here tells me to forward UDP 41641 at the same time. I only find out later from the relay item (Task 5), which means a second trip into the router. The guide mentions it only in passing in section 1 ("so a forward on your router is all direct Tailscale connections need"). **Low.**

---

## Task 3: A family member's GL.iNet travel router is asking to connect. Approve it.

**What I'd do**

- 14 (desktop): the orange "Waiting for your approval" card is at the very top: "slate7 wants to connect · Travel router on your tailnet · signed in as alex@example.com · GL-BE3600 · 100.101.9.9". I recognise slate7 and my own login, so I'd click **Approve**. The sentence "Approving lets the device use homebox's internet connection … and your home network, and reach your Tailscale machines as homebox" tells me plainly what I'm agreeing to. That's good.
- 14b: toast "slate7 can connect now. It connects by itself in a few seconds."

**Where I'd get stuck or misunderstand**

- **On my phone (14-phone).** The Approve card is at the top and the buttons are large, so approving works fine. Directly under it, though, is a big framed card, "Add this phone": "This is your home server's page. Two steps, then tailbond takes over from Tailscale on this phone. No code to scan." It has a name box already filled with "phone" and a large teal "**+ Add this phone**" button. I opened the dashboard to approve the router, not to add my own phone. This card is also different from what I'd read: the guide and the desktop "Add an Android phone" card both describe "three short steps" and an "**Open tailbond and ask homebox**" button. I'd wonder whether I'm supposed to do this as well, or I'd tap it by accident and add my phone under the name "phone". "Show the home server's own dashboard (for whoever runs it)" made it worse, because I am whoever runs it and I thought I was already on the dashboard. **Medium.**
- After I approve, slate7 doesn't appear anywhere in 14b, 14c or 14d (the Connections tab only lists pixel-8 and later pixel-7a). The only other place I see it is in 15d, listed as "slate7 · offline" with a **phone** icon. Nothing says "approved, waiting for slate7 to come online", so I can't tell from the dashboard whether the family's router worked. **Low.**

Task completed.

---

## Task 4: Get tailbond onto a family member's Android phone and get it connected

**What I'd do**

1. Devices tab, card "Add an Android phone" (15c / 15 / web-04). The small print "For family: their phone needs Tailscale signed in to your Tailscale network (invite them in the Tailscale admin console, or share this machine with their account)" tells me what to check first. Then I'd click the copy button next to `http://homebox.tail1234.ts.net/` and text it to them, or tell them to type `homebox/`.
2. On their phone (web-00): "Get tailbond on this phone". Step 1, "Download tailbond for Android (27 MB)". The drawings of each Android warning ("File might be harmful → Download anyway", "Settings", "Allow from this source", "Install", "Scan app") are exactly what a nervous relative needs. "Is this safe? … nobody on the Wi-Fi or mobile network can change the file" reassures me too.
3. Step 2: they tap "**Open tailbond and ask homebox**".
4. 14c: "pixel-7a wants to connect · Phone on your tailnet · signed in as sam@example.com · Pixel 7a". I'd click **Approve**.
5. Step 3 on their phone: "tap Add phone. Then tap the big switch … tap OK."
6. 14d: pixel-7a appears in Connections with 2 paths, "Active".

**Where I'd get stuck or misunderstand**

- 14d: the toast says "pixel-7a can connect now. It connects by itself in a few seconds." But pixel-7a is already shown connected for "40s", so the message is out of date. It also covers the top of pixel-7a's "Path 1" card, hiding its name and status. **Low.**
- Phone page intro: "Your Tailscale machines (like the NAS) stay reachable through it." Not every family has a NAS, and my relative would ask me "what NAS?". **Low.**

Task completed. This was the smoothest task.

---

## Task 5: Read the setup health and fix it, including the machine "through a relay"

**What I'd do**

- 15, Devices tab, "Setup health: 2 things to improve for best speed (tailbond works meanwhile)":
  - "Tailscale: 2 machines direct, 1 through a relay (slower). Through a relay: apple-tv. … On the router in front of this machine, forward UDP port 41641 to it."
  - "This network makes direct Tailscale connections hard … Same fix as above (one forward fixes both)."
- I'd go back to the router's Port forwarding page and add UDP 41641 → 192.168.1.2, the same way as 443. Then I'd check with `tailscale ping apple-tv`, as the card suggests. I'd skip UPnP, because the card says it "lets any device on your network open ports".
- 15d: "Setup health: everything is set up for best speed", with "Tailscale: 3 machines direct" ticked. Done.

**Where I'd get stuck or misunderstand**

- **15d after the fix:** "✓ Network is friendly to direct connections — *no port mapping on the router*". I've just added a port forward on the router, and in 15 those same words, "no port mapping on the router", were part of the problem description. To me "port mapping" and "port forward" mean the same thing, so I'd think my forward hadn't been detected even though the item has a green tick. I'd probably go back into the router to check it. **Medium.**
- The relay item says only the fix on my side. The guide adds "A relay can also come from the other machine's network, which you may not control", but the dashboard doesn't. If the Apple TV is at grandma's house and still shows as relayed after my forward, the dashboard gives me no hint why. I'd keep changing settings on my router. **Low.**
- The relay item doesn't repeat the LAN address (192.168.1.2) the way the 443 step did. "Forward UDP port 41641 to it": I had to remember what "it" was. **Low.**
- Devices list (15 / 15d), pixel-8 row: "↓ 12.0 Mbps ↑ 1.1 Mbps". The header uses ↓ for "1.1 MBPS FROM DEVICES" and ↑ for "12.0 MBPS TO DEVICES", so the arrows mean opposite things on the same page. "198.51.100.20:51234 · wlb": I don't know what "wlb" is. It isn't one of the guide's modes (Speed / Streaming / Redundant / Failover). **Low.**
- Tailscale tab (15b): a numbered list with a single item, "1. All set: bonded devices reach your tailnet through this server.", while the banner above still says "2 things to improve". The two don't conflict, but the lone "1." looks unfinished. **Low.**
- Guide section 3: "For best speed … see [setup tips](setup-tips.md)". I don't have that file. The table in the guide and the card were enough, though. **Low.**

Task completed.

---

## Task 6: Decide and set who may approve new devices

**What I'd do**

- Devices tab, card "Who can approve and add devices" (15c / 15 / 15d): "Who can approve: alex@example.com (the Tailscale account homebox belongs to), from any of their devices, and anyone using homebox itself. Others on your Tailscale network, and people you share homebox with, can ask to connect but can't approve."
- My choice: keep it on my own account and add my partner as a second approver, so either of us can approve the kids' devices. I'd type `sam@example.com` into "Also allowed to approve (Tailscale logins, comma separated)" and click **Save**. No password.

**Where I'd get stuck or misunderstand**

- **The password option:** "Or a password for this page (replaces the above; empty: none)". The card's own text says "A password instead replaces all this: then everyone, you included, needs it." Neither the card nor the guide says whether family members can still "ask to connect" once a password is set. The guide says the password "protects the whole dashboard API", and ask-to-connect seems to go to this server, so I'd worry that setting a password breaks the phone and router flow from Tasks 3 and 4. That's the main thing I need to know to decide, and I can't find it. **Medium.**
- "Tailscale logins": I'm not sure what my partner's login is. They sign into Tailscale with Google, so it's probably their Gmail, but the card doesn't say where to look it up (for example, Users in the admin console). It also doesn't say what an added approver can do beyond approving. The title says "and add devices", and the guide says they can also remove devices. Could they also change the approvers list or set a password? **Low.**
- web-04 (a tagged server, not mine): "Name approvers or set a password under Devices → Who can approve." That sentence is inside the Who can approve card. It sent me looking for another place that doesn't exist. **Low.**

Task completed (no password, one extra approver).

---

## Findings

| # | task | finding | severity |
|---|---|---|---|
| 1 | 1 | Guide section 1 firewall details ("zone and policies", "rich-rules", "the one your router's port forward arrives on") with no line saying nothing is needed from me; it refers to a port forward I haven't made yet | low |
| 2 | 1 | Fresh-install header "reachable at 192.168.1.2:443" and a "Connected" switch with no devices; unclear what the switch turns off | low |
| 3 | 2 | CGNAT check needs "your public address" when it is "different" from the router's WAN IP, but the only source given is that same WAN IP; under CGNAT I'd save the 100.x address. Card doesn't say setup health detects it | medium |
| 4 | 2 | "1 thing to improve for best speed" labels the route item, which is about reaching bonded devices, not speed | low |
| 5 | 2 | UDP 41641 not suggested alongside the UDP 443 forward; second trip into the router later | low |
| 6 | 3 | Dashboard on my phone (14-phone) puts a large "Add this phone" card ("Two steps … No code to scan", name "phone", "+ Add this phone") under Approve; differs from the guide's "three short steps"/"Open tailbond and ask homebox"; easy to add my own phone by mistake; "Show the home server's own dashboard (for whoever runs it)" confusing for the admin | medium |
| 7 | 3 | After "slate7 can connect now…", slate7 never appears in Connections (14b–14d); only later as "offline" with a phone icon (15d); no "approved, waiting" state | low |
| 8 | 4 | 14d toast "pixel-7a … connects by itself in a few seconds" while it's already connected 40s, and it covers pixel-7a's Path 1 card | low |
| 9 | 4 | Phone page "Your Tailscale machines (like the NAS)" assumes a NAS | low |
| 10 | 5 | 15d green item still says "no port mapping on the router" right after I added a forward, the same words as the problem text in 15; I'd think the forward failed | medium |
| 11 | 5 | Relay item doesn't mention that the relay can come from the other machine's network (only the guide does) | low |
| 12 | 5 | 41641 fix says "forward … to it" without repeating the LAN address 192.168.1.2 | low |
| 13 | 5 | Device-row arrows (↓12.0 / ↑1.1) are the reverse of the header's (↓ from / ↑ to devices); "wlb" unexplained | low |
| 14 | 5 | Tailscale tab "1. All set" lone numbered item next to a "2 things to improve" banner | low |
| 15 | 5 | Guide links "setup tips (setup-tips.md)", which isn't available | low |
| 16 | 6 | Password option doesn't say whether family devices can still "ask to connect" once it's set, which is the key fact for the decision | medium |
| 17 | 6 | No hint where to find a family member's "Tailscale login", or whether approvers can also change approvers/password | low |
| 18 | 6 | Tagged-server text "under Devices → Who can approve" points to the card it's already in (web-04) | low |

Totals: 0 high / 4 medium / 14 low
