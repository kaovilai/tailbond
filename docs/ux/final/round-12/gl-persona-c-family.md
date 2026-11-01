# Round 12 — GL persona C (family member)

**Who I am:** a family member who doesn't do tech. Someone else set up the home server. I was handed a GL.iNet Slate 7 travel router and asked to get it working. I know what Wi-Fi and a phone hotspot are. I don't know routes, ports or NAT.

**What I had:** only `task-script.md`, `screens.md` and the PNGs in `packet-gl/`. I looked at the desktop version of every screen and at the phone versions of 01b, 04, 09 and 11. The phone layouts stack cleanly and the buttons are big enough to tap. I found no phone-only problems.

---

## Task 1 — Install tailbond on the Slate 7 without SSH

**Where I'd click:**
1. On the home server page (15, "Add a GL.iNet travel router (recommended)"), I click **Download the router package**. On the Flint (11–13) the same button is blue.
2. On the Slate 7 admin (01a): **SYSTEM → Advanced Settings → Go To LuCI**. The note says "Use the same admin password to log in."
3. In LuCI (01b), I click **Upload Package…** and pick the file. The dialog asks "Really install the uploaded package?" and shows `tailbond_0.1.0-r42_aarch64_cortex-a53_neon-vfpv4.ipk (6.1 MB)`. I click **Install**.
4. Back in the GL admin, I open **VPN → tailbond** (02).

**What went well:** the three numbered steps on 15 and 11–13 match the screens exactly: "SYSTEM → Advanced Settings → Go To LuCI, then System → Software → Upload Package…, pick the file, Install." I could follow them word for word.

**Where I'd hesitate:**
- LuCI looks like a different, more technical product. The only explanation is "LuCI is OpenWrt's own web interface", and I don't know what OpenWrt is. The filename `…aarch64_cortex-a53_neon-vfpv4.ipk` means nothing to me, so I can't tell it's the right file for a Slate 7. No screen shows that the install worked. Step 3 just says "Back in the router's admin page", without saying that LuCI is a different tab or address (01a shows LuCI at `http://192.168.8.1:8080/cgi-bin/luci`). **Low.**
- The LuCI dialog has a checkbox "Allow overwriting conflicting package files". The Flint panel says "(leave the other options as they are)", but the Fedora page's step 2 only says "pick the file, Install". Coming from the Fedora page, I'd wonder whether to tick it. **Low.**

## Task 2 — Pair with the home server

**Where I'd click:** on 02 I see "Looking for your home server on your tailnet…". A few seconds later (03) it shows "Found your home server: homebox", and I click **Ask to connect**. On 04, "Waiting for approval on homebox…" tells me: "On homebox, tap **Approve** next to **slate7**… Not you? Ask whoever runs your home server to approve it." It also says "you can close this page." That is exactly what I need. I'd text the person who set it up.

**Where I'd hesitate:**
- The word "tailnet" ("on your tailnet", 03c "No home server found on your tailnet") is unfamiliar. The longer sentence above it ("must be on the same Tailscale network as this router") helps. **Low.**
- 03b (Tailscale off) and 03c (nothing found) both say what to do, with **Open Tailscale settings** and **Search again** buttons. Those were fine.

## Task 3 — Hotel Wi-Fi + phone tethering, confirm both are used

**Where I'd look:** on 05 the top card says "Connected to homebox · Using Repeater (Wi-Fi) + Tethering (phone)". Under Internet connections, both the Repeater and Tethering cards show a green "In use". The total speed (↓ 59 Mbps) is about the two cards added together (38 + 21). Done, with no confusion.

On 06, with the phone having trouble, the card says "Not working right now — It's connected, but can't reach homebox. tailbond keeps trying; the other connections carry your traffic." That reassured me. No findings.

## Task 4 — Phone as backup only

**Where I'd click:** on the **Tethering** card, under "Use it", I click **Backup only**. The hint under Speed or reliability points me there too: "Saving your phone's data? Set Tethering or Cellular to "Backup only" under Internet connections." After that (07) the badge says "Backup · saving data" and the card says "Used only if your other connections stop working. Until then tailbond sends nothing over it." The top line changes to "Tethering (phone) on standby". **Change limit** (07b) shows "5 GB a month… 0 = no limit". All clear.

**Where I'd hesitate:**
- Under "Speed or reliability", the "Most reliable" option says "Sends everything over every connection. Uses the most data." I'd worry that picking it would use my phone's data again even with Backup only set. The page doesn't say which setting wins. "Fastest" is already selected, so I'd just leave it alone. **Low.**

## Task 5 — Reach the NAS by name from the laptop

**Where I'd look:** on 05, **Tailnet machines** lists "nas / nas.tail1234.ts.net / 100.101.2.2 · linux · Online · direct" with a **Copy name** button. The Home server card also says: "Tailnet names: Work on every device connected to this router while tailbond is on. Use the full name, like nas.tail1234.ts.net." So the name is **nas.tail1234.ts.net**, and the page tells me not to use just "nas". Found it fast.

**Where I'd hesitate:**
- I was told to use the NAS's "MagicDNS name", but that word appears nowhere. The page says "full name" and "Tailnet names". I'd guess they mean the same thing. The note "(not this router's own Tailscale)" also lost me. **Low.**

## Task 6 — Turn tailbond off

**Where I'd click:** the blue switch at the top right of the "Connected to homebox" card (05). 09 then says "tailbond is off — Devices on this router use the internet directly, without your home server". The yellow warning is very good: "While tailbond is off, GL may use Tethering (your phone's data)… To be sure the phone's data isn't used, unplug it or turn tethering off on it, or turn tailbond back on." I'm glad it says this, because I'd have assumed Backup only still protected me.

If I clicked the red **Unpair** by mistake, the dialog on 09b ("This router stops using tailbond and forgets the home server… it will need to approve") would stop me. If tailbond isn't running (10), "Start tailbond" is clear.

**Where I'd hesitate:**
- The main switch has no label (no "On/Off" text). On the same page, the "Check for sign-in pages" switch looks identical. While tailbond is off (09), the Home server card still shows **Ask to connect again**. I might click that to turn it back on, not knowing whether it starts a new approval. **Low.**

## Task 7 — Approve the travel router on the home side

**Where I'd click:**
- **Flint (11):** at the top, "Waiting for your approval · 1 new — slate7 wants to connect", then **Approve**. After that (12), slate7 shows up under Devices as "Connected just now · 2 connections bonded". Clear.
- **Fedora (14):** an orange box "Waiting for your approval — slate7 wants to connect" with a green **Approve** button. Also clear.

**Where I'd get confused:**
- In the orange note right next to Approve (11 and 14): "Signed in to alex@example.com, an account that may approve here. If this device stays with you, just approve. **If you'll give it to someone: approve, then before handing it over sign its Tailscale in to their account (or tag it)**; tailbond keeps working, and until then whoever holds it could also approve devices here." *I am* the person it was handed to. So I can't tell whether the router is now "staying with me" or "given to someone". The note says that I, the holder, could approve other devices on the family server, which sounds like a risk. But it doesn't tell me (or whoever is approving) how to "sign its Tailscale in to their account" or what "tag it" means. I'd still press Approve, but I'd be unsure whether I'd just done something unsafe. I'd call the person who set it up. **Medium.**
- The "Who can approve" paragraph above it is long and technical: on Fedora (14) it reads "any user or program on homebox itself (http://127.0.0.1:8077; set a password if homebox runs things you don't trust)…"; the Flint version ends "…which may be more than its owner's own account gets." I'd skip it. It's only information, but it sits right on top of the one button I need. **Low.**
- Fedora's Connections tab (14) shows "Path 0 · Pixel-8 · Min 31 Ms" and "2 active path(s)". "Ms" with a capital M, "Pixel-8" spelled differently from "pixel-8" elsewhere, and "path(s)" all read as unfinished. **Low.**

## Task 8 — Hotel Wi-Fi with a sign-in page (GL's repeater detected it)

**Where I'd click:** on 16 the top line says "Using Tethering (phone) · Repeater (Wi-Fi) needs sign-in". The blue box "Repeater needs you to sign in" tells me: "GL's Repeater page handles this sign-in. Open it and click "Login Mode for Public Hotspots" (or "Go to Login"). Then open any website on your laptop or phone: the hotel's sign-in page appears there." I click **Open GL's Repeater page** and follow it. On 16b, "Signing in to Repeater through GL's Login Mode" explains that sign-in ends by itself and "tailbond then adds Repeater back to bonding." It also tells me my phone's data isn't used while I sign in. I could do this.

**Where I'd hesitate:**
- The Ethernet sign-in (17b) has an **I've signed in, check now** button, but the Wi-Fi sign-in (16b) only has **Back to GL's Repeater page**. Once I'd accepted the hotel terms, I wouldn't know how to confirm it worked except by waiting. 16 says "(until it's done, or 10 minutes)" but not what happens if I'm not done in 10 minutes. **Low.**

## Task 9 — Hotel Ethernet cable with a sign-in page

**Where I'd click:** on 17, "Ethernet needs you to sign in", I click **Open sign-in page**. The small note "GL's Login Mode covers the repeater Wi-Fi only; tailbond handles sign-in pages on other connections like this one" explains why this works differently from Task 8. That helped. On 17b, "Sign in on the page that opened (accept the terms, enter the room number…)", then **I've signed in, check now**. "tailbond adds Ethernet back by itself once you're signed in." I could do this.

**Where I'd hesitate:**
- "For up to 9 more minutes (until 05:34 on this router's clock)". Why would the router's clock be different from mine? "9 more minutes" alone would be enough. The header "Router connected to homebox · devices signing in" made me wonder whether each of my devices has to sign in separately. **Low.**

## Task 10 — Best speed at home

**Where I'd look:**
- **Flint:** the **Setup health** card. 18c says "All good — Everything is set up". On 13 (fresh setup), "Step 1: tell travel routers where home is… Easiest: turn on APPLICATIONS → Dynamic DNS and paste the name it shows here", with a link, which I could follow. It marks the other item "Optional… Skip this unless…", so I'd skip it.
- **Fedora (15):** Devices tab, "Setup health: 2 optional speed-ups (tailbond works fine without them)".

**Where I'd get stuck:**
- On 18 (Flint, "2 to improve"), the fix reads: "On this router: NETWORK → Firewall → Open Ports on Router, add UDP 41641. If another router sits in front of this one (for example your internet provider's box), also forward UDP port 41641 to this router there." On Fedora (15): "On your home router's admin page (Port forwarding, also called Virtual server or NAT), forward UDP port 41641 to this machine (192.168.1.2). (Turning on UPnP on that router also works, but lets any device on your network open ports.)" I don't know what UDP, forwarding or UPnP are, or whether there's "another router in front". The page is honest ("Not sure how? Show this card to whoever set up your internet") and says it's optional, so I'd stop and pass it on. I can't finish "make sure everything is set up for best speed" myself. **Medium.**
- On 18 the green "Setup done — Travel routers and phones can connect from anywhere" sits right above "Setup health · 2 to improve". Done or not done? The word "Optional" explains it, but only after some reading. **Low.**
- Fedora 15 says "Router fixes are on your home router's admin page, route ones in the Tailscale admin console", but there's no "route" item in the list, so I went looking for something that wasn't there. **Low.**
- 18b (CGNAT) is headed "Your internet line uses carrier-grade NAT", which is jargon. But the action is something I can actually do: "Ask your provider for a public IPv4 address (often free or a small monthly fee; say it's for reaching your home from outside)." I could read that out on the phone to the provider. **Low** (heading only).

---

## Findings

| # | Task | Finding | Severity |
|---|---|---|---|
| 1 | 7 | Approval note "If you'll give it to someone: approve, then before handing it over sign its Tailscale in to their account (or tag it)… whoever holds it could also approve devices here". I'm the person it was given to. It suggests a risk, but I can't tell if it applies or how to "tag it". | medium |
| 2 | 10 | Speed-up fixes need "add UDP 41641" / "forward UDP port 41641… (Port forwarding, also called Virtual server or NAT)" / "UPnP". I can't do them myself and can only hand off ("Show this card to whoever set up your internet"). | medium |
| 3 | 1 | LuCI step: "LuCI is OpenWrt's own web interface", cryptic `…aarch64_cortex-a53_neon-vfpv4.ipk` filename, no success screen, and "Back in the router's admin page" doesn't say it's a different tab/address | low |
| 4 | 1 | Fedora step 2 lacks "(leave the other options as they are)", but LuCI shows "Allow overwriting conflicting package files" | low |
| 5 | 2 | "tailnet" jargon ("Looking for your home server on your tailnet…") | low |
| 6 | 4 | "Most reliable — Sends everything over every connection. Uses the most data." Unclear whether it overrides "Backup only" | low |
| 7 | 5 | "MagicDNS" never appears (the page says "full name" / "Tailnet names"); "(not this router's own Tailscale)" is confusing | low |
| 8 | 6 | Main on/off switch has no label; "Ask to connect again" still offered while off | low |
| 9 | 7 | Dense "Who can approve…" paragraph ("any user or program on homebox itself (http://127.0.0.1:8077…)") right above Approve | low |
| 10 | 7 | Fedora path cards: "Min 31 Ms", "Pixel-8" vs "pixel-8", "2 active path(s)" | low |
| 11 | 8 | Wi-Fi (Login Mode) flow has no "I've signed in, check now" unlike Ethernet; "(until it's done, or 10 minutes)" doesn't say what happens after 10 minutes | low |
| 12 | 9 | "until 05:34 on this router's clock"; header "devices signing in" suggests every device must sign in | low |
| 13 | 10 | Flint 18: green "Setup done" next to "Setup health · 2 to improve" | low |
| 14 | 10 | Fedora 15: "route ones in the Tailscale admin console" but no route item is listed | low |
| 15 | 10 | 18b heading "Your internet line uses carrier-grade NAT" is jargon (the action under it is clear) | low |

Totals: 0 high / 2 medium / 13 low
