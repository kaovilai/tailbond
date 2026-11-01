# Round 7 — Fedora persona F2 (home admin)

**Persona:** I run the family's home server, a spare PC with Fedora on it. I can install packages with `dnf`, paste commands from a guide, and I set up Tailscale by following its website. I don't really know firewalls, NAT or port forwarding. I'm the person the family comes to.

**Materials used:** `task-script.md`, `screens.md`, `fedora-server.md`, `glinet/14*`, `glinet/15*`, `web/web-00*`, `web/web-04*`. I read nothing else.

---

## Task 1: Install tailbond on homebox and open its dashboard

**What I'd do:** I'd follow guide §1 "Install" and paste
`sudo dnf install ./tailbond-server-*.rpm` and `sudo systemctl enable --now tailbond`, then check with `systemctl status tailbond`. homebox sits in a closet, so I'd open the dashboard from my laptop at `http://homebox:8077`. The guide says "(or `http://<server>:8077`)". I land on the Devices tab (15c).

**How it went:** Easy. These are commands I can copy, and I never have to run `firewall-cmd` ("You don't need to run any `firewall-cmd` commands yourself"). The table of package paths and the paragraph about `100.64.0.0/10` are more than I need, but I can skip them.

**Stuck / misunderstood:**
- The header has a big **"Connected"** toggle (15c, top right) on a server that has no devices yet. Nothing says what it's connected to, or what happens if I switch it off. Would that cut off the whole family? I'd leave it alone, but I'd be nervous about touching it by mistake. **Low.**

## Task 2: Right after installing (15c), what's left and how?

**What I'd do:** I read the **Setup health** card at the top of 15c:
"Setup health: 1 to improve for best speed / Route 10.77.0.0/24 isn't approved yet / Approve it once in the Tailscale admin console (Machines → this machine → Edit route settings). How to fix". I've used the Tailscale admin console before, so with guide §3 ("find this machine, choose **Edit route settings**, and enable `10.77.0.0/24`") I can do this one.

Then I scroll past four more cards and find the orange banner near the bottom:
"Public address not set. Pairing links will use 192.168.1.2:443, which is probably your LAN address. Enter your public IP or DDNS name (and forward UDP 443 on your router)."

**Stuck / misunderstood:**
- **The most important step is the easiest to miss, and the page understates it.** The header says "Home server · reachable at 192.168.1.2:443". The health card says only "**1** to improve for best speed". The public address and the UDP 443 forward don't appear in Setup health at all. They're in a banner below the health card, the password card, the Android card, the GL.iNet card and the "add with a code" card. Guide §2 says "Bonding needs each of the device's links to reach the server directly over UDP", so without this nothing works away from home. It isn't a speed tweak. If I only read the top card, I'd think I was one click from done. (15d shows that health *can* check "Public internet address" and "UDP 443 reached from the internet", but on a fresh install those lines aren't there.) **Medium.**
- **No help for the "how".** The banner asks for "your public IP or DDNS name" and says "forward UDP 443 on your router". The guide only says "On your home router, forward **UDP port 443** to this server's LAN address. Use a DHCP reservation so that address doesn't change." I don't know my public IP, I don't have a DDNS name, and I don't know whether my IP changes. I don't know what a DHCP reservation is or where my router keeps it. The guide's CGNAT paragraph ("If your ISP puts you behind CGNAT (no public IPv4)…") assumes I can tell whether I'm behind CGNAT. Unlike the health items, the banner has no "How to fix" link. I'd end up searching the web. **Medium.**
- **The guide points to a tab that doesn't exist.** §2 step 2: "Use the banner in the dashboard's **Clients** tab". The tabs are Connections, Devices, Tailscale and Logs (15c). The banner is on **Devices**. I'd click through every tab looking for "Clients" before I scrolled down Devices. **Medium.**
- Should I type the port? The banner's placeholder is `home.example.net:443`, but the guide's config example is `"public_addr": "home.example.net"` with no port. I'd guess and hope. **Low.**
- The route item is framed as "to improve for best speed", but guide §3 says the route is what lets the tailnet reach bonded devices. It has nothing to do with speed, and the guide calls the whole Tailscale section "optional, recommended". So I can't tell how much it matters. **Low.**

## Task 3: Approve the family member's GL.iNet travel router

**What I'd do:** The dashboard (14) shows an orange card at the top: "Waiting for your approval … slate7 wants to connect · Travel router on your tailnet · signed in as alex@example.com · GL-BE3600 · 100.101.9.9". I'd click **Approve**. 14b shows a toast: "slate7 can connect now. It connects by itself in a few seconds."

**How it went:** Very clear. It's at the top, it names the router model, and it shows the Tailscale account, so I know it's ours. This was the easiest task.

**Stuck / misunderstood:**
- After approving, slate7 doesn't show up in Connections (14b), only the toast. In the later screenshot (15d), slate7 is listed as "offline" with a **phone** icon, even though it's a router. Nothing confirms that it actually connected "by itself in a few seconds", and the icon makes me wonder whether I approved the right thing. **Low.**
- When I open the dashboard on my own phone (14-phone), an "Add this phone" card sits right under the approval card: "Two steps, then tailbond takes over from Tailscale on this phone." I only came to approve the router, and for a moment I'd wonder whether approving also changes *my* phone. **Low.**

## Task 4: Get tailbond onto a family member's Android phone and connect it

**What I'd do:** Devices → **Add an Android phone** (15c). I'd click the copy button next to `http://homebox.tail1234.ts.net/` and text it to them, or tell them "type homebox in your browser". On their phone they'd see the page in web-00: **1 Install the app** (with pictures of each Android prompt: "Download anyway", "Settings", "Allow from this source", "Install"), then **2 Add this phone to homebox** → "Open tailbond and ask homebox", then **3 Turn tailbond on**. Their request would appear at the top of my dashboard, and I'd tap Approve, like in Task 3.

**How it went:** The install part is really good. The pictures match the scary prompts ("File might be harmful"), and "That's expected" plus "You can turn Allow from this source off again afterwards" are exactly what my family needs to hear. "Stuck, or not comfortable with this? Ask whoever runs homebox" is fine, because that's me.

**Stuck / misunderstood:**
- **Step 3 says what to do but not how.** "Turn tailbond on: Android runs one VPN at a time, so while tailbond is on, Tailscale's VPN is off on this phone." It names no button and shows nothing of what the phone will ask, unlike step 1, which shows every prompt. The way *back* gets a button name ("tap Switch back to Tailscale in tailbond"), but the way *on* doesn't. The packet has no screenshot of the app itself, and none of a phone's request on my dashboard, so I can't confirm the last step from here. This is where my family member would call me. **Medium.**
- "or just tell them to type the name": if they type `homebox` into Chrome's address bar, Chrome may treat it as a search. Saying "type `homebox/`" or "http://homebox" would be safer. **Low.**

## Task 5: Read setup health and fix it, including the relayed machine

**What I'd do:** Connections shows a red bar, "Setup health: 2 things to improve for best speed. Show" (14). That takes me to the Devices card (15):
- "Tailscale: 2 machines direct, 1 through a relay (slower) / Through a relay: pixel-8. To check one: tailscale ping pixel-8 / Usually fixed by forwarding UDP port 41641 to this machine, or by turning on UPnP on the router in front of it. How to fix"
- "This network makes direct Tailscale connections hard / Hard NAT (the outside port changes per destination); no port mapping on the router. / Forward UDP port 41641 to this machine, or turn on UPnP on the router in front of it. How to fix"

I'd go into my home router, turn on UPnP (that sounds easier than another port forward), and come back. 15d then shows all green: "Network is friendly to direct connections / port mapping: UPnP", "Tailscale: 3 machines direct".

**Stuck / misunderstood:**
- **Port 41641 and UPnP appear nowhere in my guide.** The guide told me to forward UDP **443**. Now the dashboard wants UDP **41641** too. Is that instead of 443, or as well? The guide's only pointer is "For best speed … see [setup tips](setup-tips.md)", which isn't in my materials. I can't see where the "How to fix" links go. I don't know what UPnP is or whether it's safe to turn on, and the card offers it as just as good as a port forward. **Medium.**
- **Which side is broken?** "Through a relay: pixel-8", yet the same page lists pixel-8 as connected through tailbond with "2 Paths", "#0 active · 31 ms", "#1 active · 58 ms". If pixel-8 is out on mobile data, is the relay caused by *my* router or by *its* network? Will forwarding a port at home fix it? "Hard NAT (the outside port changes per destination)" means nothing to me. I'd follow the instruction without knowing whether it targets the right machine. **Medium.**
- "Not sure how? Show this card to whoever set up your internet; the links explain each fix." I *am* whoever set up our internet, so for me this is a dead end. **Low.**
- "To check one: tailscale ping pixel-8": I can run it, but nothing tells me what output means relayed or direct. **Low.**
- Both items have the same fix ("forward UDP port 41641 … or turn on UPnP"), but they're counted as "2 to improve". It feels like twice the work. **Low.**
- After the fix (15d), a stray **"null"** appears on its own line under the green checks. It looks broken, right when I want to trust that "everything is set up". **Low.**

## Task 6: Decide and set who may approve new devices

**What I'd do:** Devices → **Who can approve and add devices** (15c): "Anyone who can open this page (people on your Tailscale account, and anyone you share this machine with). Set a password to require it." Then the field "Password for this page (empty: none)" and **Save**. The family's devices are on my Tailscale account (slate7 is "signed in as alex@example.com"), so I'd set a password.

**Stuck / misunderstood:**
- **The default leaves the dashboard open to the whole family, and their phone page leads them to it.** The phone page says "nothing here changes homebox's settings" (web-00c). Its footer, though, says "homebox's dashboard, for whoever runs it: http://homebox:8077", and step 2 says "You run homebox yourself? Approve it at http://homebox:8077 … that page works on this phone too." So any family member who taps that link reaches the page with Approve, the trash (Revoke) buttons and the Connected toggle. The card doesn't recommend anything, and guide §4 says "check your ACLs", which I don't know how to do. I only worked out that I need a password by connecting those pieces myself. **Medium.**
- The password field has no confirm box, and no screen shows what happens after Save: whether I get a login prompt, or how to reset it if I forget. Guide §1 mentions both the card and `"ui_password"`, but §4 only says "Set `ui_password` in `tailbond.json`". I'd guess that editing the JSON is the way back in. **Low.**

## Other screens

- web-04 (a different server) shows "Get the router package (the file ending in _neon-vfpv4.ipk) from whoever set up tailbond." On my own server, *I'm* the person who set it up, so this would leave me stuck. homebox shows a "Download the router package" button instead (15c), so this doesn't affect my tasks. **Low.**

---

## Findings

| # | task | finding | severity |
|---|---|---|---|
| 1 | 2 | Public address and UDP 443 forward missing from Setup health, buried below 4 cards; header says "reachable at 192.168.1.2:443" and health says "1 to improve for best speed", which understates that nothing works away from home | medium |
| 2 | 2 | No help finding public IP/DDNS, doing the UDP 443 forward or DHCP reservation, or telling whether I'm behind CGNAT; banner has no "How to fix" link | medium |
| 3 | 2 | Guide says "Use the banner in the dashboard's **Clients** tab"; no Clients tab exists (banner is on Devices) | medium |
| 4 | 2 | Banner placeholder `home.example.net:443` vs guide `"public_addr": "home.example.net"`: include the port or not? | low |
| 5 | 2 | Route approval framed "for best speed", but it's about reaching bonded devices; guide calls Tailscale optional | low |
| 6 | 1 | Unlabelled "Connected" toggle on the server header; unclear what switching it off does to the family's devices | low |
| 7 | 3 | After Approve only a toast; slate7 not visible in Connections, later "offline" with a phone icon; no confirmation it connected | low |
| 8 | 3 | Dashboard on phone puts "Add this phone … tailbond takes over from Tailscale on this phone" right under the approval card | low |
| 9 | 4 | Step 3 "Turn tailbond on" names no button and shows no prompt, unlike step 1; no app screen or phone-request screen in the packet to confirm | medium |
| 10 | 4 | "just tell them to type the name": typing `homebox` in Chrome may search instead of opening the page | low |
| 11 | 5 | UDP 41641 and UPnP not in the guide (only an unavailable setup-tips.md link); unclear whether it's instead of or in addition to UDP 443; no word on UPnP safety | medium |
| 12 | 5 | "Through a relay: pixel-8" while pixel-8 is bonded with 2 active paths; unclear whether the fix belongs on my router or the phone's network; "Hard NAT (the outside port changes per destination)" is jargon | medium |
| 13 | 5 | "Show this card to whoever set up your internet": that's me, so it's a dead end | low |
| 14 | 5 | "tailscale ping pixel-8" with no hint how to read the result | low |
| 15 | 5 | Two health items with an identical fix counted as "2 to improve" | low |
| 16 | 5 | Stray "null" line in the all-good health card (15d) | low |
| 17 | 6 | Default no password; phone page tells family "nothing here changes homebox's settings" yet links them to http://homebox:8077; card gives no recommendation; guide says "check your ACLs" | medium |
| 18 | 6 | Password field: no confirm, no view of what happens after Save, no reset path; guide §4 mentions only `ui_password` in JSON | low |
| 19 | other | web-04: "Get the router package … from whoever set up tailbond" when the reader *is* that person (other server only) | low |

Totals: 0 high / 7 medium / 12 low
