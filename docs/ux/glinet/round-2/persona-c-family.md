# Round 2: Persona C (less-technical family member)

Who I am: my nephew set up the home server and Tailscale for me. I have used
the GL router's admin page a couple of times to join hotel Wi-Fi. I can follow
clear steps, but "subnet", "tunnel", "MagicDNS" don't mean much to me. My big
fear is breaking the internet for everyone. I mostly used the phone-sized
screenshots.

## Task 1: Install tailbond on the Slate 7 without SSH

**Where I click:** `01a-install-gl-advanced-settings-phone`: SYSTEM → Advanced
Settings, blue **Go To LuCI** button. The note "Use the same admin password to
log in." is reassuring. Then `01b-install-luci-upload-*`: a dialog
"Manually install package / Really install the uploaded package?" with
`tailbond_0.1.0-r42_aarch64_cortex-a53_neon-vfpv4.ipk (6.1 MB)`, then **Install**.

**Where I'd get stuck / confused:**
- The Slate 7 itself never tells me these steps. The only step-by-step I can
  find is on the *home* router's page (`11`/`12`/`13`, card "Add a GL.iNet
  travel router": "SYSTEM → Advanced Settings → Go To LuCI ... then System →
  Software → Upload Package…"). If I'm at the hotel with only the Slate 7 in
  front of me, I'd have to remember that, or have a screenshot. It worked out
  because the steps are written in GL's own menu words, which I recognised.
- LuCI looks like a different, older website (dark blue bar, "Status System
  Services Network Logout"). On the phone shot I only see the dialog and the
  grey background; I can't see the "System → Software" menu or the "Upload
  Package…" button behind it, so on my phone I'd be hunting for the menu.
- The checkbox **"Allow overwriting conflicting package files"**: should I tick
  it? "Overwriting" and "conflicting" sound like I could break something.
  Nothing says "leave this unticked". I'd probably leave it, but I'd hesitate.
- The file name `..._aarch64_cortex-a53_neon-vfpv4.ipk` is gibberish to me.
  The home page's hint "the tailbond file ending in _neon-vfpv4.ipk" helps me
  pick the right one if there are several downloads.
- No screen shows what happens after Install (success message, or where to go
  next). The "Back on the travel router's admin page, open APPLICATIONS →
  tailbond" step is only on the home router's page.

**Expected:** a short "how to install" note I can see from the Slate 7 itself,
and a word on the checkbox ("leave unticked").

## Task 2: Pair the Slate 7 with the home server

**Where I click:** `02-just-installed`: "Looking for your home server on your
tailnet…" (spinner). A few seconds later, `03-found-server`: **"Found your home
server: homebox"** and a big blue **Ask to connect**. I tap it. `04-waiting-
approval`: "Waiting for approval on homebox… On homebox, tap **Approve** next to
**slate7**. This router connects by itself once approved; you can close this
page." with two bullets for Flint 3 vs computer at home.

**What went well:** this is the easiest part. "homebox" is the name my nephew
uses, so I know it's right. "you can close this page" is exactly what I want to
hear.

**Small confusions:**
- "tailnet" ("Looking for your home server on your tailnet…", "No home server
  found on your tailnet") — I don't know that word. From context I guess it
  means "my Tailscale stuff".
- `03b-tailscale-off`: "Turn on Tailscale first" with **Open Tailscale
  settings** is clear; I'd manage.
- `03c-no-server-found`: "Checked 4 online machines. On the home server,
  tailbond must be installed and running, and Tailscale on." I can't check that
  myself; I'd phone my nephew. That's fine, it's honest.
- "Other ways to connect / Paste a link" and "Make this the home server" at the
  bottom: I'd ignore them, but "Make this the home server" is a little
  worrying, like a button I could press by mistake. It's small and has
  "(you can switch back here)", so OK.

## Task 3: Hotel Wi-Fi + phone tethering, confirm both used

**Where I look:** `05-connected-two-wans` top card: **"Connected to homebox /
Using Repeater (Wi-Fi) + Tethering (phone)"** with "↓ 59 Mbps ↑ 7.1 Mbps".
Below, under "Internet connections", **Repeater** and **Tethering** both have a
green **"In use"** tag. Ethernet says "Not connected" with "Plug a cable from the
hotel or modem into the WAN port."

**What went well:** very clear; the top sentence answers the question in plain
words. Green "In use" twice = both used.

**Small confusions:**
- "Repeater" — I know it from GL's own page when joining hotel Wi-Fi, so fine.
  The "(Wi-Fi)" in the top card helps.
- `06-one-wan-failed`: "Tethering (phone) not working right now" and the red tag
  "Not working right now" with "It's connected, but can't reach homebox.
  tailbond keeps trying; the other connections carry your traffic." Clear and
  calming: I know the internet still works.
- The "Speed or reliability" box (Fastest / Smooth calls and video / Most
  reliable) — I'd leave it alone. "Most reliable ... Uses the most data" is a
  fair warning.

## Task 4: Phone as backup only

**Where I click:** under **Tethering**, "Use it" row: **Backup only**
(`07-save-data-standby`). The hint under Speed or reliability also tells me:
"Saving your phone's data? Set Tethering or Cellular to "Backup only" under
Internet connections." Result: top card "Tethering (phone) on standby", tag
"Backup · saving data", and "Used only if your other connections stop working.
Until then it isn't used at all, so it uses no data."

**What went well:** exactly what I wanted, and it says "uses no data" in plain
words.

**Small confusions:**
- There's also **Change limit** right above, showing "1.2 GB of 5 GB this
  month". When I first read "limit the phone", I'd be tempted to press "Change
  limit" instead. `07b-set-limit` explains "At the limit it pauses; the count
  restarts on the 1st of each month. 0 = no limit." That's understandable, but
  it's a second way of "limiting" next to the first one. I picked Backup only
  thanks to the hint text.
- "Off" sits right next to "Backup only". I'd be careful not to tap Off by
  mistake, but it's labelled.

## Task 5: Reach the NAS by its name from the laptop

**Where I look:** `05-connected-two-wans`, card **Tailnet machines**: "Laptops
and phones on this router's Wi-Fi or LAN reach these by their full name, e.g.
nas.tail1234.ts.net, or by their 100.x address." Row **nas** with a **Copy
name** button. Also in "Home server": "Tailnet names — Work on every device
connected to this router while tailbond is on. Use the full name, like
nas.tail1234.ts.net."

**Where I'd stumble:**
- On the phone the long name breaks in the middle of a word:
  `nas.tail1234.ts.ne` on one line and `t · 100.101.2.2` on the next (same for
  apple-tv, pixel-8, work-laptop, on `05`, `06`, `07`, `11`, `12`, `13`). If I
  copied it by eye I'd type `nas.tail1234.ts.ne` and it wouldn't work, and I'd
  have no idea why. On the desktop shot it reads correctly
  (`nas.tail1234.ts.net · 100.101.2.2`). "Copy name" saves me if I use it, but
  I'm looking at this on my phone and typing on the laptop.
- I never see the word "MagicDNS" anywhere, which honestly is fine for me; I
  wouldn't know it. The page just says "full name", which I understand.
- I'm not sure whether "Copy name" copies "nas" or the long name. I'd assume
  the long one because the page keeps saying "full name".

## Task 6: Turn tailbond off

**Where I click:** the blue switch in the top card. `09-turned-off`:
**"tailbond is off / Devices on this router use the internet directly, without
your home server. Turn on to bond your connections again."** Connections show
"Ready", "Used when tailbond is on."

**What went well:** very reassuring: it says my devices still have internet.
`08-server-unreachable` even tells me what to do when home is down: "Meanwhile,
devices on this router have no internet... To use this internet directly for
now, turn tailbond off." with a **Turn tailbond off** button. That's exactly
the safety net I want. `10-not-running`: "Starting it is safe: devices here
keep their internet while you set it up." Also calming.

**Small confusions:**
- The red **Unpair** button sits next to "Ask to connect again" in the Home
  server card. Someone wanting "off" might press Unpair. The confirm dialog
  (`09b-unpair-confirm`) catches me: "This router stops using tailbond and
  forgets the home server... (it will need to approve)." Good, though the
  difference between "off" and "unpair" only becomes clear after reading it.
- "Can't reach homebox" in `08` says "Whoever set up homebox should check that
  it's on and that UDP port 443 reaches home.example.net:443". I don't know
  what that means, but it tells me to ask my nephew, which is fine.

## Task 7: Approve the travel router at home

**Flint 3 (`11-server-pending-request`):** right at the top, "Waiting for your
approval · 1 new", "slate7 wants to connect / Travel router on your tailnet ·
signed in as alex@example.com / GL-BE3600", buttons **Decline** / **Approve**.
I tap Approve. `12-server-approved`: slate7 appears in Devices: "Connected just
now · 2 connections bonded". Clear; I'd know it worked.

**Fedora page on my phone (`14-fedora-dashboard-request`):** same approval card
at the top with **Approve**. Easy. Below it, though, there's a huge card
**"Add this phone ... then tailbond takes over from Tailscale on this phone"**
with "Download tailbond for Android" and a big **+ Add this phone** button. I
only came to approve the router; that big green button is tempting and "takes
over from Tailscale on this phone" sounds like it might change my phone. I'd
leave it, but it pulls attention.
The header also cuts off: "Home server · reachable at homeb···".

**Flint 3 freshly set up (`13-server-route-pending`):** if I were the one
setting up the Flint, this is where I'd stop and phone my nephew. "Step 1:
allow tailbond devices on your Tailscale network ... Machines → homebox → Edit
route settings → turn on 10.77.0.0/24" and "Step 2: ... Enter your home's
public IP address or a Dynamic DNS name (APPLICATIONS → Dynamic DNS gives you
one)". I don't know what my "public IP address" is or what Dynamic DNS is, and
"10.77.0.0/24" is a code to me. The steps are honest and there's an **Open admin
console** button, but I couldn't finish this alone. (It's not part of
approving, and the persona says a relative set it up, so it's a smaller
problem for me.) The "Approve" step itself still works here.

## Findings

| # | task | finding | severity |
|---|---|---|---|
| 1 | 5 | On phone, tailnet names wrap mid-word ("nas.tail1234.ts.ne" / "t · 100.101.2.2"); copying by eye gives a wrong name. Seen on 05/06/07/11/12/13 phone. | medium |
| 2 | 1 | The install steps only live on the home router's page; nothing on the Slate 7 (where I'm installing) tells me the LuCI → System → Software → Upload Package… path or where to go after. | medium |
| 3 | 7 (setup) | `13`: "turn on 10.77.0.0/24", "public IP address or a Dynamic DNS name" are beyond me; I'd be stuck setting up the Flint without help. | medium |
| 4 | 1 | LuCI dialog checkbox "Allow overwriting conflicting package files" with no hint whether to tick it; sounds risky. | low |
| 5 | 1 | On the phone LuCI screen, the menu / "Upload Package…" button isn't visible behind the dialog; LuCI looks like a different website. | low |
| 6 | 4 | "Change limit" sits next to "Backup only"; "limit the phone" could send me to the data cap instead of standby. The hint text saved me. | low |
| 7 | 6 | Red "Unpair" sits next to the Home server buttons; the difference from the on/off switch is only clear in the confirm dialog. | low |
| 8 | 7 | Fedora phone page: big "+ Add this phone" card ("tailbond takes over from Tailscale on this phone") under the approval card pulls attention and sounds like it changes my phone. | low |
| 9 | 7 | Fedora phone header truncated: "reachable at homeb···". | low |
| 10 | 2 | "tailnet" is used without explanation ("Looking for your home server on your tailnet…"). | low |
| 11 | 5 | Unclear whether "Copy name" copies short "nas" or the full name. | low |
| 12 | 6 | `08` "check that UDP port 443 reaches home.example.net:443" is jargon, but it does say to ask whoever set it up. | low |
| 13 | 2 | "Make this the home server" link on the travel router's first screen feels like something I could press by mistake. | low |
