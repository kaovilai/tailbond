# Round 3: Persona C (less-technical family member)

Who I am: my nephew set up the home server and Tailscale for me. I've used the
GL router's admin page a couple of times to join hotel Wi-Fi ("Repeater"). I
can follow clear steps. "Subnet", "tunnel", "MagicDNS", "UDP" mean nothing to
me. My big worry is breaking the internet in the hotel room or at home.

I mostly looked at the phone screenshots, plus 15 (desktop).

---

## Task 1: Install tailbond on the Slate 7 without SSH

**Getting the file.** The task says I already downloaded it. On the Fedora page (15-desktop)
there's a clear button, "Download the router package", in the "Add a GL.iNet travel router"
box. That one is easy.

If our home server were the Flint 3, though, its panel (11/12/13-phone, "Add a GL.iNet
travel router") only says *"Get the router package on your computer: the tailbond file
ending in _neon-vfpv4.ipk (from the tailbond releases page, or from a Fedora home server's
tailbond page)."* That's not a button or a link. I don't know what "the tailbond releases
page" is or where to find it, and we don't have a Fedora server. I'd have to call my
nephew. The Fedora page gives a download button, so the Flint panel feels like it's
missing one. **Medium.**

Good: the "ending in _neon-vfpv4.ipk" hint lets me check that I picked the right file
in my Downloads folder.

**Step 1 (01a-phone).** SYSTEM → Advanced Settings shows "LuCI is OpenWrt's own web
interface. Use the same admin password to log in." and a big "Go To LuCI" button.
I can do this because the steps on the home server page name the same words. "LuCI"
and "OpenWrt" sound scary, but the line about the same password helps.

**Step 2 (01b-phone).** The LuCI page looks completely different (dark blue bar, only
"GL-BE3600" at the top). For a moment I'd wonder if I'd left the router's page. The
screenshot only shows the final dialog, so I can't judge the System → Software → Upload
Package… path on the phone. The dialog "Manually install package / Really install the
uploaded package?" with my file name and "Install" is clear.

- *"Allow overwriting conflicting package files"* has a checkbox and no explanation.
  "Overwriting" and "conflicting" make me nervous. I'd leave it unticked and hope that's
  right. Nothing says whether it matters. **Low.**
- Nothing tells me whether installing will drop the internet while it runs, or how to
  get back from LuCI to the normal GL page afterwards. (Step 3 on the home page just says
  "Back on the travel router's admin page".) I'd probably retype 192.168.8.1. **Low.**

## Task 2: Pair the Slate 7 with the home server

02-phone: APPLICATIONS → tailbond shows "Not connected to a home server. Not set up yet."
and a spinner, "Looking for your home server on your tailnet…". A few seconds later
(03-phone) a big blue box says "Found your home server: homebox" with a big "Ask to
connect" button. I recognise "homebox", so I'd tap it. Easy.

04-phone: "Waiting for approval on homebox… On homebox, tap **Approve** next to
**slate7**. This router connects by itself once approved; you can close this page." Very
clear about what happens next, and the reassurance that I can close the page is nice.

- "tailnet" (02, 03c) is a word I don't know, but the sentence still makes sense.
  **Low.**
- The bottom link "Setting up the router that stays at home? Make this the home server"
  appears on every setup screen. I wouldn't click it, because "the router that stays
  at home" is clear. "(you can switch back here)" reassures me. Fine.
- 03b (Tailscale off): "Turn on Tailscale first" with an "Open Tailscale settings" button
  is clear. 03c (none found): "Checked 4 online machines. On the home server, tailbond
  must be installed and running…" tells me it's not my fault and to ask whoever set
  up the server. Good.

## Task 3: Hotel Wi-Fi + phone tethering, confirm both used

05-phone: the top card says "Connected to homebox · Using Repeater (Wi-Fi) + Tethering
(phone)" with speeds. Under "Internet connections", both **Repeater** and **Tethering** show
a green "In use". I know "Repeater" from joining hotel Wi-Fi on the GL page, so this
matches my experience. Done, no confusion.

- 06-phone (phone trouble): the red "Not working right now" made me flinch, but the
  text "It's connected, but can't reach homebox. tailbond keeps trying; the other
  connections carry your traffic." calms me down. Good.
- The "Home server" box shows "Connects to home.example.net over the internet (UDP port
  443)" and "This router's tunnel address 10.77.0.2". I don't understand these and
  don't need them. **Low** (noise for me, but harmless).
- In that same box, "Ask to connect again" sits next to "Unpair" while I'm already
  connected. I'd wonder whether I'm supposed to press it. **Low.**

## Task 4: Phone as backup only

Under Tethering, "Use it: Always | Backup only | Off". I'd tap "Backup only". The
"Speed or reliability" box also has a hint: "Saving your phone's data? Set Tethering or
Cellular to "Backup only" under Internet connections." That confirmed it.

07-phone: Tethering shows "Backup · saving data" and "Used only if your other connections
stop working. Until then it isn't used at all, so it uses no data." The top says
"Tethering (phone) on standby". This is exactly what I wanted to hear.

07b-phone ("Change limit"): a box with "5 GB a month", Save/Cancel, and "At the limit,
tailbond stops using this connection until the count restarts on the 1st of each month.
0 = no limit." Understandable. Nothing to flag.

- "Speed or reliability" has three choices ("Fastest", "Smooth calls and video", "Most
  reliable"). I'd leave it alone. "Most reliable … Uses the most data" is a fair
  warning. Fine.

## Task 5: Reach the NAS by name from the laptop

At the bottom of 05-phone, "Tailnet machines" lists "nas" with "nas.tail1234.ts.net",
"100.101.2.2", "linux · Online" and a "Copy name" button. The text above says "Laptops and
phones on this router's Wi-Fi or LAN reach these by their full name, e.g.
nas.tail1234.ts.net". I'd tap "Copy name" and paste it into the laptop's browser. Easy.

- The task says "MagicDNS name", but the page never uses that word. It says "full name"
  instead. For me that's *better*. I only mention it in case my nephew tells me "use the
  MagicDNS name". **Low.**
- When tailbond is off or can't reach home (08, 09), the list says "The list appears once
  tailbond is connected." Fair enough.

## Task 6: Turn tailbond off

The top card has an on/off switch next to "Connected to homebox". I'd tap it. 09-phone:
"tailbond is off. Devices on this router use the internet directly, without your home
server. Turn on to bond your connections again." That reassures me the internet still
works. Each connection says "Used when tailbond is on." Clear.

- I'd steer clear of the red "Unpair". 09b-phone's confirmation ("Unpair from homebox? …
  To connect again, ask the home server again (it will need to approve)") explains the
  difference well, so a mis-tap is recoverable.
- 08-phone (home unreachable): "Meanwhile, devices on this router have no internet … To
  use this internet directly for now, turn tailbond off." plus a "Turn tailbond off"
  button. Scary but honest, and it tells me what to do. Good for someone who worries
  about breaking the internet.
- 10-phone (not running): "Starting it is safe: devices here keep their internet while you
  set it up." Exactly the reassurance I need.

## Task 7: Approve the travel router at home

**Flint 3 (11-phone).** Near the top, under "Waiting for your approval" with an orange
"1 new" badge: "slate7 wants to connect … GL-BE3600 … asked 1 min ago" with
"Decline" / "Approve". I'd tap Approve. 12-phone: slate7 now shows under "Devices" as
"Connected just now", and the approval box is gone. Easy.

**Fedora (14-phone / 15-desktop).** The same "Waiting for your approval / slate7 wants to
connect / Approve" box is the first thing on the page. Easy.

- 15-desktop: below the tabs, the "Add a device" box says "Each phone or travel router
  gets its own key…" and has a name field and "+ Add". Right under it is another box,
  "Add a GL.iNet travel router". With two "add" boxes that both mention travel routers,
  I could easily type "slate7" into "Add a device" and press Add instead of (or as well
  as) pressing Approve. I wouldn't know whether that breaks something or creates a
  duplicate. **Medium.**
- 14-phone: under the approval box there's a big "Add this phone" card that says
  "tailbond takes over from Tailscale on this phone". I'm only here to approve the router,
  and "takes over from Tailscale" sounds like it could break my phone's setup. I'd
  ignore it, but it competes for attention. **Low.**
- The Fedora header has a "Connected" switch (15-desktop). I'd be afraid to touch it,
  since it's not clear whether it turns off the whole home server for everyone. **Low.**
- 13-phone (Flint freshly set up, "2 setup steps left") is full of words like "Edit route
  settings, tick the one route listed (10.77.0.0/24)" and "Dynamic DNS". I couldn't do
  that myself. It isn't one of my tasks, though, and it looks like a job for whoever set
  up the server. **Low.**

---

## Findings

| # | task | finding | severity |
|---|---|---|---|
| 1 | 1 | Flint 3 panel has no download button. It says "from the tailbond releases page, or from a Fedora home server's tailbond page", which is not a link, and a Flint-only household has no Fedora page. I'd be stuck without help. | medium |
| 2 | 1 | LuCI install dialog "Allow overwriting conflicting package files" checkbox has no guidance. It sounds dangerous, and I'd guess to leave it unticked. | low |
| 3 | 1 | No word on whether installing interrupts the internet, or how to get from LuCI back to the GL admin page (LuCI looks like a different site). | low |
| 4 | 2 | "tailnet" ("Looking for your home server on your tailnet…") is unexplained jargon. The sentence still works. | low |
| 5 | 3 | Home server box shows "UDP port 443" and "tunnel address 10.77.0.2". Meaningless noise for me. | low |
| 6 | 3/6 | "Ask to connect again" button shown while already connected. I'd wonder if I'm supposed to press it. | low |
| 7 | 5 | The page says "full name", never "MagicDNS". That's better for me, but the words won't match if someone tells me "MagicDNS name". | low |
| 8 | 7 | Fedora page: "Add a device" box ("Each phone or travel router gets its own key…", name + Add) sits right next to "Add a GL.iNet travel router". I might type slate7 and press Add instead of Approve. | medium |
| 9 | 7 | Fedora phone page: big "Add this phone" card ("tailbond takes over from Tailscale on this phone") right under the approval. It distracts and sounds risky. | low |
| 10 | 7 | Fedora header "Connected" switch: not clear whether it shuts off the home server for everyone. I'd be afraid to touch it. | low |
| 11 | 7 | Flint "Finish setting up" steps (route 10.77.0.0/24, Dynamic DNS) are beyond me. OK only because the person who set up the server does them. | low |

Totals: high 0, medium 2, low 9.
