# Round 6, Persona A: GL.iNet traveller who already uses GL's Tailscale page

Background: I travel a lot and use GL.iNet travel routers. I'm not a network
engineer. I know GL's APPLICATIONS → Tailscale page (one Enable toggle) and the
Repeater page's "Login Mode for Public Hotspots" for hotel sign-in pages. I
expect a new GL feature to feel like those: a toggle, a status line, and
plain words.

Inputs used: task-script.md, screens.md, and the round-6 PNGs (phone for every
screen; desktop for 01b, 05, 11, 13, 15, 16, 16b, 17, 17b, 18, 18b, 18c).

---

## Task 1: Install tailbond on the Slate 7 without SSH

**Where I click:** On the Flint 3 panel (11/13/18c, "Add a GL.iNet travel
router") I click **"Download the router package"**. Step 2 tells me exactly
where to go: "On the travel router's admin page: SYSTEM → Advanced Settings →
Go To LuCI (log in with the same password), then System → Software → Upload
Package…, pick the file and click Install (leave the other options as they
are)." On the Slate 7, 01a shows SYSTEM / Advanced Settings with "Go To LuCI"
and the hint "Use the same admin password to log in." 01b (desktop) shows
LuCI's Software page with "Upload Package…" and the dialog "Really install the
uploaded package?" showing `tailbond_0.1.0-r42_aarch64_cortex-a53_neon-vfpv4.ipk (6.1 MB)`.
I leave "Allow overwriting conflicting package files" unchecked, as the
instructions say, and click **Install**.

**Friction:**
- I've only ever added things from GL's own pages (APPLICATIONS → Plug-ins).
  LuCI looks like a different product (dark-blue bar, "Status / System /
  Services / Network"). The step-by-step text makes it doable, but it feels
  like a back door. (low)
- The Fedora page (15) says "Download the router package" without saying
  which routers it fits. The Flint 3 page says "It's the same tailbond this
  router runs", which reassures me there, but from the Fedora box I'd wonder
  whether a file from a PC fits my Slate 7. The filename (`aarch64_cortex-a53`)
  means nothing to me. (low)
- On the phone, 01b only shows the dialog; I couldn't see the "Upload
  Package…" button itself on that screen. The desktop shot shows it. (low)
- After installing, I'd first look under APPLICATIONS (where GL's Tailscale
  lives), but step 3 tells me "VPN → tailbond", and the left menu shows it
  there under VPN. Fine.

## Task 2: Pair the Slate 7 with the home server

**Where I click:** VPN → tailbond (02). It says "Looking for your home server
on your tailnet…", then 03 shows "Found your home server: homebox" with a big
blue **"Ask to connect"**. One tap. 04 shows "Waiting for approval on
homebox… On homebox, tap Approve next to slate7. This router connects by
itself once approved; you can close this page."

**Friction:** Almost none. This is as easy as GL's Tailscale page. 03b
("Turn on Tailscale first" + "Open Tailscale settings") and 03c ("No home
server found on your tailnet… Checked 4 online machines") would tell me what
to do if something was off.
- "Other ways to connect: No Tailscale on this router? Paste a link…" Not for
  me, and it's clearly marked as the alternative. OK.

## Task 3: Hotel Wi-Fi (Repeater) + phone USB tethering, confirm both used

**Where I look:** 05, the top card: "Connected to homebox · Using Repeater
(Wi-Fi) + Tethering (phone)", "↓ 59 Mbps ↑ 7.1 Mbps". Under Internet
connections, Repeater and Tethering both say green **"In use"**, each with
its own speed (38 Mbps and 21 Mbps). "Repeater" and "Tethering" are the same
names GL uses on its INTERNET page, so I recognise them.

**Friction:** None that stops me. 06 also shows clearly when the phone has
trouble: "Tethering (phone) not working right now" and "It's connected, but
can't reach homebox. tailbond keeps trying; the other connections carry your
traffic."

## Task 4: Phone as backup only

**Where I click:** On the Tethering card, "Use it": **Always | Backup only |
Off**. I tap **Backup only** (07). Badge becomes "Backup · saving data" with
"Used only if your other connections stop working. Until then it isn't used
at all, so it uses no data." The top card says "Tethering (phone) on standby".
The hint under Speed or reliability ("Saving your phone's data? Set Tethering
or Cellular to 'Backup only'…") pointed me straight there.

**Friction:**
- "Change limit" (07b) is a nice extra: "5 GB a month… At the limit, tailbond
  stops using this connection until the count restarts on the 1st of each
  month." Clear. (no issue)
- The three "Speed or reliability" choices sit right below and also sound
  like a way to save data ("Most reliable… Uses the most data"). The hint
  sentence stops me picking one of those by mistake. (no issue)

## Task 5: Reach the NAS by its MagicDNS name

**Where I look:** "Tailnet machines" on 05: `nas`, `nas.tail1234.ts.net`,
`100.101.2.2`, "linux · Online · direct", with **"Copy name"**. The Home
server card says "Tailnet names: Work on every device connected to this router
while tailbond is on. Use the full name, like nas.tail1234.ts.net."

**Friction:**
- With GL's Tailscale I'm used to typing just `nas`. Here it says to use the
  full name. Clear enough, just a habit change. (low)
- "Laptops and phones on this router's Wi-Fi or LAN reach these through the
  bond to homebox (not this router's own Tailscale)". I turned on GL's
  Tailscale for this, so it's a bit surprising that it isn't the one carrying
  my traffic, but it doesn't change what I do. (low)

## Task 6: Turn tailbond off

**Where I click:** The blue toggle at the top right of the status card, just
like GL's Tailscale "Enable" toggle. 09: "tailbond is off. Devices on this
router use the internet directly, without your home server. Turn on to bond
your connections again." Exactly what I expected.

**Friction:**
- Under Home server there's also a red "Unpair" next to "Ask to connect
  again". The confirm dialog (09b) explains it well ("This router stops using
  tailbond and forgets the home server… it will need to approve"), so I
  wouldn't unpair by accident. "Ask to connect again" while already connected
  (05) puzzles me a bit; I'd leave it alone. (low)

## Task 7: Approve the new travel router on the home side

**Flint 3 (11):** The card at the top, "Waiting for your approval · 1 new":
"slate7 wants to connect · Travel router on your tailnet · signed in as
alex@example.com · GL-BE3600" with **Approve**. I click it; 12 shows slate7 in
Devices, "Connected just now · 2 connections bonded". Easy.

**Fedora (14 phone / 15 desktop):** Same "Waiting for your approval" card at
the top with **Approve**.

**Friction:**
- Fedora: "To require a password, set "ui_password" in
  /etc/tailbond/tailbond.json (see the Fedora server guide)." That's a
  file-editing job I couldn't do; the Flint 3 just has a "Set" link. Doesn't
  block approval. (low)
- Fedora on my phone (14): right under the approval card is a big "Add this
  phone… Download tailbond for Android" card. I came to approve the router,
  not install an app; slightly distracting. (low)

## Task 8: Hotel Wi-Fi with a sign-in page (GL's repeater detected it)

**Where I click:** 16 shows a blue banner: "Repeater needs you to sign in.
GL's Repeater page handles this sign-in. Open it, click "Login Mode for Public
Hotspots" (or "Go to Login") and accept the hotel's terms." I click **"Open
GL's Repeater page"** and use Login Mode as I always do. Meanwhile the phone
keeps me online ("Using Tethering (phone) · Repeater (Wi-Fi) needs sign-in").
16b: "Signing in to Repeater through GL's Login Mode… It ends by itself once
you've signed in (or click "Quit Login Mode" on GL's Repeater page); tailbond
then adds Repeater back to bonding."

This is exactly how I hoped it would work: it points me to the GL feature I
already know, and it says it adds Repeater back on its own.

**Friction:**
- 16b's headline still says "Connected to homebox" with a green icon, and
  Tethering still shows "In use · 21 Mbps", while the line under it says "all
  traffic from devices here goes out Repeater (Wi-Fi) directly, not through
  homebox". The blue banner explains it, so I get it, but the green
  "Connected" plus "not through homebox" made me read it twice. (low)

## Task 9: Hotel Ethernet cable with a sign-in page

**Where I click:** 17, orange banner "Ethernet needs you to sign in. GL's
Login Mode covers the repeater Wi-Fi only; tailbond handles sign-in pages on
other connections like this one." I click **"Open sign-in page"**. 17b:
"Signing in to Ethernet… Sign in on the page that opened (accept the terms,
enter the room number…)." Buttons: "Open the sign-in page again", "I've
signed in, check now", "Stop sign-in mode". Meanwhile the hotel Wi-Fi keeps
me online.

**Friction:**
- The first line of 17 answers my obvious question (why not GL's Login
  Mode?). Good. Still, two different sign-in tools for Wi-Fi and cable is a
  bit odd, but I wouldn't get stuck. (low)
- "For up to 9 more minutes (until 04:18 on this router's clock)". The
  router's clock might not match my laptop's time zone; "9 more minutes" is
  what I'd go by. (low)

## Task 10: At home, set everything up for best speed

**Flint 3:** "Setup health" card. 18c: "All good · Everything is set up for
best speed" with ticks. 18 (relayed): "2 to improve… Tailscale: 2 machines
direct, 1 through a relay (slower)… On this router: NETWORK → UPnP, turn it
on." That's a GL menu I know, so I can do it myself. Second item "Same fix as
above." Good. 13 (fresh setup) gives two numbered steps ("Step 1: allow
tailbond devices on your Tailscale network… Open admin console" and "Step 2:
tell travel routers where home is… turn on APPLICATIONS → Dynamic DNS").
Clear.

**Fedora (15):** "Setup health: 2 to improve for best speed… Forward UDP port
41641 to this machine, or turn on UPnP on the router in front of it." Plus
"Not sure how? Show this card to whoever set up your internet". Fine.

**Friction:**
- 18b (CGNAT): headline "Home server is on · 1 setup step left", red "Step 1:
  Your internet line uses carrier-grade NAT… travel routers can't reach UDP
  443 here. Ask your provider for a public IPv4 address". But the same page
  shows pixel-8 "Connected 3 h 0 min · 2 connections bonded", and the
  Tailscale route/health checks I saw on 18c are gone (no Setup health card
  at all). So is my setup broken for travel or not? I'd be worried my Slate 7
  won't work at the next hotel, and I can't tell whether something still
  works (pixel-8 says it does). Calling "phone your ISP" a "setup step" also
  makes me feel I did something wrong. (medium)
- 13 Step 1 sends me to the Tailscale admin console ("choose Edit route
  settings, tick the one route listed (10.77.0.0/24)"). That's outside GL and
  more technical than anything on GL's Tailscale page, but the instructions
  are exact. (low)
- "Show 3 more checks" (18) hides the checks that passed; fine. (no issue)

## Other screens

- 08 (home server unreachable): "Meanwhile, devices on this router have no
  internet, so nothing goes around your home server. To use this internet
  directly for now, turn tailbond off." with a "Turn tailbond off" button.
  Very clear about the consequence and the way out. Like GL's VPN kill
  switch.
- 10 (not running): "Starting it is safe: devices here keep their internet
  while you set it up." One button. Clear.

---

## Findings

| # | task | finding | severity |
|---|---|---|---|
| 1 | 1 | Install goes through LuCI ("Go To LuCI" → System → Software → Upload Package…) instead of a GL page like APPLICATIONS → Plug-ins; feels like a back door, but the steps are exact | low |
| 2 | 1 | Fedora page's "Download the router package" doesn't say which routers it fits; filename `aarch64_cortex-a53_neon-vfpv4` means nothing to me (Flint page's "same tailbond this router runs" reassures) | low |
| 3 | 1 | 01b phone shows only LuCI's install dialog; the "Upload Package…" button is only visible on desktop | low |
| 4 | 5 | "Use the full name, like nas.tail1234.ts.net": I'm used to short MagicDNS names (`nas`) from GL's Tailscale | low |
| 5 | 5 | "(not this router's own Tailscale)": surprising since I turned GL's Tailscale on for this; doesn't change what I do | low |
| 6 | 6 | "Ask to connect again" shown while already connected (05) is puzzling; next to red "Unpair" | low |
| 7 | 7 | Fedora password needs editing `/etc/tailbond/tailbond.json`; the Flint has a "Set" link | low |
| 8 | 7 | Fedora page on phone (14): big "Add this phone / Download tailbond for Android" card right under the approval I came for | low |
| 9 | 8 | 16b headline "Connected to homebox" (green) and Tethering "In use" while text says all traffic goes out Repeater directly, "not through homebox"; I had to read it twice | low |
| 10 | 9 | Wi-Fi sign-in uses GL's Login Mode, cable sign-in uses tailbond's own: explained ("GL's Login Mode covers the repeater Wi-Fi only"), still two tools | low |
| 11 | 9 | "until 04:18 on this router's clock": the router's clock may differ from mine | low |
| 12 | 10 | 18b CGNAT: "1 setup step left / travel routers can't reach UDP 443 here" while pixel-8 shows "Connected… 2 connections bonded" and the Setup health card disappears; unclear whether travel will work, and "ask your provider" framed as a setup step I failed | medium |
| 13 | 10 | 13 Step 1 needs the Tailscale admin console ("Edit route settings", tick 10.77.0.0/24): outside GL and more technical, but exact | low |

Counts: high 0, medium 1, low 12.
