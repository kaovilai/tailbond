# Persona A: frequent traveller, GL.iNet + GL's built-in Tailscale page (round 1)

Who I am: I travel a lot and I'm not a network engineer. I already use GL.iNet travel routers, and GL's Tailscale page (APPLICATIONS → Tailscale) is my reference: one Enable toggle, Apply, and a link to bind my account. I want new GL features to feel the same: one switch, very little to configure.

Screens viewed: all phone variants 01–14, plus the desktop versions of 01, 05 and 11.

---

## Task 1: Install tailbond on the Slate 7 without SSH

**Where I'd click (01-not-installed-luci-upload):** I'd follow the manifest: GL admin → SYSTEM → Advanced Settings → Go To LuCI, then LuCI → System → Software → "Upload Package…". The dialog says "Manually install package", then "Upload the tailbond package you downloaded (for GL.iNet routers the file ends in **_aarch64_cortex-a53_neon-vfpv4.ipk**)". My file `tailbond_0.1.0-r42_aarch64_cortex-a53_neon-vfpv4.ipk (6.1 MB)` matches that ending, so I'd click **Install**.

**Where I'd hesitate:**
- The route through LuCI is unfamiliar. In the GL panel I add things from APPLICATIONS → Plug-ins (desktop 05 and 11 show that "Plug-ins" entry in the sidebar), and nothing there sends me to LuCI. "Advanced Settings → Go To LuCI" feels like the part of the router I'm warned not to touch. I could only find this path because the manifest spelled it out. Once I was there, the dialog itself was clear.
- "Allow overwriting conflicting package files" is ticked and greyed out. "Overwriting" and "conflicting" sound scary, and I can't untick it. Nothing says this is harmless.
- No screen shows what happens after Install: no success message, and no "now go to APPLICATIONS → tailbond". I'd guess the GL panel has a new menu entry, and 02 confirms it does, but I had to guess.

**Expected:** Upload the file on the GL Plug-ins page, or at least a one-line note on the download page saying "GL's Plug-ins page can't upload files; use LuCI → System → Software". After installing: "Installed. Open APPLICATIONS → tailbond in your GL admin panel."

## Task 2: Pair the Slate 7 with the home server

**Where I'd click:** On 02 I see "Looking for your home server on your tailnet…". A few seconds later (03) it shows "Found your home server: homebox", `homebox.tail1234.ts.net · 100.101.1.1`, and one big blue **Ask to connect** button, which I'd tap. That then shows (04) "Waiting for approval on homebox…" with clear directions: "Open homebox's tailbond page (on a Flint 3: its admin panel → APPLICATIONS → tailbond) and tap **Approve** next to **slate7**. This router connects by itself once approved."

This is the closest thing in the product to GL's Tailscale page: one button, and it tells me where the other half happens. I liked it.

**Small snags:**
- 03 has two "Ask to connect" buttons: the big blue one and a greyed one next to the "Not this one? Type your home server's Tailscale name:" box. It's clear enough which one I want, but it's a bit busy.
- The placeholder "e.g. homebox or homebox.ta" is cut off on phone.
- "tailnet" is Tailscale jargon. I know it from the Tailscale app, but a less technical travel companion wouldn't.
- The error states read well. 03b says "Turn on Tailscale first" and has an "Open Tailscale settings" button, which is exactly where I'd go. 03c says "No home server found on your tailnet · Checked 4 online machines…" and offers "Search again".

## Task 3: Hotel Wi-Fi (repeater) + phone tethering, confirm both are used

**Where I'd look (05):** The top card says "Connected to homebox · Using Wi-Fi (repeater) + phone (USB tethering)" with "↓ 59 Mbps ↑ 7.1 Mbps". Under "Internet connections", both **Repeater** ("↓ 38 Mbps ↑ 4.2 Mbps · 24 ms") and **Tethering** ("↓ 21 Mbps ↑ 2.9 Mbps · 48 ms") show a green "In use". Ethernet shows "Not connected". So both are in use, and I didn't have to configure anything. "Repeater" and "Tethering" are the same words GL uses on its INTERNET page, which helps.

When the phone has trouble (06), the top card reads "phone (USB tethering) not working right now", and Tethering shows a red "Not working right now" plus "It's connected, but can't reach homebox. tailbond keeps trying; the other connections carry your traffic." That's reassuring.

**Small snag:** Each connection card has a "Use" toggle, a "Backup only" checkbox and a "Set limit" link, and the top card has its own toggle. That's more switches than I expected from "one switch". None of them are needed for this task, though.

## Task 4: Phone as backup only

**Where I'd click:** On the Tethering card I'd tick **Backup only**. 07 then shows the chip "Backup · saving data", the line "Used only if your other connections stop working", and the top card reads "phone (USB tethering) on standby". That confirms it worked.

**Where I got confused:** Further down, "How to use your connections" has a mode called **Save mobile data**: "Uses Ethernet and Wi-Fi; your phone and cellular only when those stop working." That's word for word the goal of my task. On 07, after I ticked Backup only, the selected mode is still **Fastest**. Now I don't know:
- whether I should have picked "Save mobile data" instead of, or as well as, the checkbox;
- whether "Fastest" ("Adds all your connections together") overrides my Backup only tick. The chip says "saving data", but the selected mode says "adds all".

I'd probably click both to be safe, and I still wouldn't know which one is actually in charge. Two controls for the same idea, with no link between them.

**Minor:**
- "Change limit" (07b) is a different thing: a monthly data cap ("At the limit it pauses until next month. 0 = no limit."). It's explained well, and I'd actually use it alongside backup.
- Changes seem to apply instantly with no Apply button. That's different from GL's usual Apply flow, but the status chip changing was enough feedback.

**Expected:** One place to do this. Either the "Save mobile data" mode ticks Backup only on Tethering automatically, or ticking Backup only shows a note like "Mode: Save mobile data" or "Backup connections stay unused in every mode".

## Task 5: Reach the NAS by its MagicDNS name

**Where I'd look (05):** At the bottom, the "Tailnet machines" list (account `alex@example.com`) says "Laptops and phones on this router's Wi-Fi or LAN reach these by name, e.g. nas.tail1234.ts.net." The first row is **nas**, `nas.tail1234.ts.net · 100.101.2.2`, "linux · Online", with a **Copy name** button. That answers the task: the name is `nas.tail1234.ts.net`, and I'd tap Copy name.

**Small snags:**
- In the Home server card, the row label is "Names like nas….ts.net" and its value is "Work on every device connected to this router while tailbond is on". The "nas….ts.net" with an ellipsis looks like a rendering glitch, and the label/value split reads oddly as a sentence.
- Does the short name `nas` work too, or do I always need the full `.ts.net` name? It only says "e.g. nas.tail1234.ts.net", so I'd use the full one.
- When tailbond isn't connected (08, 09), the list is replaced by "The list appears once tailbond is connected." That's fine, but it doesn't say whether my Slate's own GL Tailscale still reaches the NAS in that case.

## Task 6: Turn tailbond off

**Where I'd click:** The toggle in the top card. 09 shows "Off · Off. Devices on this router use the internet directly, without your home server." That's clear, and it's the one-switch experience I wanted.

**Small snags:**
- "Off" in the title followed by "Off." at the start of the text is redundant.
- The red **Unpair** button in the Home server card is close by and looks more "final". A nervous user might tap it to turn things off. The confirmation (09b) explains it well: "This router stops using tailbond and forgets the home server… To connect again, ask the home server again (it will need to approve)." So no harm is done.
- Nothing says whether turning tailbond off also affects GL's own Tailscale on the Slate. I'd assume it doesn't.

## Task 7: Approve the travel router at home

**Flint 3 (11):** The card "Waiting for your approval · 1 new" shows "slate7 wants to connect · Travel router on your tailnet · signed in as alex@example.com · GL-BE3600 · linux · 100.101.9.9", and I'd tap **Approve**. 12 then lists slate7 under Devices as "Connected just now · 2 connections bonded". Easy, and it matches the words in 04 ("tap Approve next to slate7").

**Fedora (14):** "Waiting for your approval · slate7 wants to connect", then **Approve**. Easy. But the large "Add this phone" panel underneath ("This is your home server's page. Two steps, then tailbond takes over from Tailscale on this phone…") assumes I'm on a phone. I opened it on my laptop to approve a router, so for a second I wondered whether I was on the wrong page.

**Home-side setup, where I'd get into trouble (13 + 08):** On a freshly set-up Flint, 13 asks for two things I don't really understand:
- "One step left in the Tailscale admin console · Approve the route **10.77.0.0/24** for homebox (Machines → homebox → Edit route settings)." There's an "Open admin console" button, so I'd manage, but "route" and "10.77.0.0/24" are the kind of things I don't want to touch.
- "Set this router's internet address · Travel routers connect to 192.168.8.1:443, which looks like a home-network address. Enter your public IP or a dynamic DNS name (Applications → Dynamic DNS)…" I'd turn on GL's Dynamic DNS and paste its name in via "Change". Nothing here mentions opening or forwarding a port. Then at the hotel, 08 says: "Still not working? Show this to whoever set up homebox: UDP port 443 must reach home.example.net:443 from the internet." I set up homebox, and I'm now in a hotel, so I can't fix it from there. The home page should have warned me about the port, or checked it, while I was still at home.

---

## Findings

| # | task | finding | severity |
|---|---|---|---|
| 1 | 1 | Install goes through LuCI ("SYSTEM → Advanced Settings → Go To LuCI" → "Upload Package…"). Nothing in GL's own panel (APPLICATIONS → Plug-ins) leads there, so I only found it from written steps, and LuCI feels like "don't touch" territory. | medium |
| 2 | 1 | "Allow overwriting conflicting package files" is pre-ticked and greyed out, with no explanation. It sounds risky. | low |
| 3 | 1 | No post-install screen or message saying "open APPLICATIONS → tailbond". | low |
| 4 | 2 | Two "Ask to connect" buttons on 03 (found server + manual name box). Slightly busy, but the big one is clearly the right one. | low |
| 5 | 2 | The placeholder "e.g. homebox or homebox.ta" is cut off on phone. "tailnet" is jargon for non-Tailscale users. | low |
| 6 | 3 | Many controls per connection ("Use" toggle, "Backup only", "Set limit") plus the main toggle. More than the "one switch" I expect, though none are needed. | low |
| 7 | 4 | Two controls do the same job: the "Backup only" checkbox on Tethering and the "Save mobile data" mode. After ticking Backup only, the mode still shows "Fastest · Adds all your connections together", so I can't tell which one wins and would probably set both. | medium |
| 8 | 4 | Changes apply instantly with no Apply button, unlike GL's usual flow. The status chip "Backup · saving data" makes up for it. | low |
| 9 | 5 | The row "Names like nas….ts.net" / "Work on every device connected to this router while tailbond is on" looks garbled (ellipsis in a hostname, label/value reads oddly). | low |
| 10 | 5 | Unclear whether the short name `nas` works or only `nas.tail1234.ts.net`. | low |
| 11 | 6 | "Off" title + "Off." text is redundant. The red "Unpair" sits next to the off switch, but 09b's confirmation explains it well. | low |
| 12 | 6 | Doesn't say whether turning tailbond off affects GL's own Tailscale on the Slate (can I still reach the NAS?). | low |
| 13 | 7 | Fedora page (14) puts a big "Add this phone" panel under the approval card even when opened from a laptop. A moment of "am I on the right page?". | low |
| 14 | 7 | Home setup (13) asks a non-engineer to "Approve the route 10.77.0.0/24" in the Tailscale admin console. Doable via "Open admin console", but intimidating. | low |
| 15 | 7 / 3 | Home setup (13) never mentions that UDP port 443 must be reachable from the internet. I only learn it at the hotel from 08 ("Show this to whoever set up homebox: UDP port 443 must reach…"), when I'm the one who set it up and I can't fix it remotely. | medium |
