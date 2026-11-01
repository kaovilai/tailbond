# Round 7 — GL persona A (traveller)

Persona: frequent traveller, owns GL.iNet travel routers, used to GL's built-in
Tailscale page (one toggle, it just works). Not a network engineer.
Materials: `out/ux/final/round-7/packet-gl/` (task-script.md, screens.md, PNGs) only.
Desktop screens checked for every task; phone versions checked for 01a, 01b, 05,
11, 14, 17b.

Overall: I could finish all ten tasks from the screens. Nothing blocked me. The
GL panel feels like part of GL's own UI (VPN → tailbond, toggle top right, same
cards and buttons). The two places I got confused are the "signing in" states
at the hotel (task 8) and the Flint's carrier-grade NAT health card (task 10).

---

## Task 1 — Install on the Slate 7 without SSH

**Where I click.** On the home server's page (15-fedora-devices-tab, card "Add a
GL.iNet travel router (recommended)") I click **"Download the router package"**.
On the Slate 7: SYSTEM → Advanced Settings → **"Go To LuCI"**
(01a; "LuCI is OpenWrt's own web interface. Use the same admin password to log
in."). In LuCI: System → Software → **"Upload Package…"**, pick
`tailbond_0.1.0-r42_aarch64_cortex-a53_neon-vfpv4.ipk (6.1 MB)`, then
**"Install"** in the "Manually install package / Really install the uploaded
package?" dialog (01b). No SSH needed. The Flint version of the steps (11–13) is
even clearer.

**Where I hesitate.**
- The Fedora page's step 2 is shorter than the Flint's. Fedora: "pick the file,
  Install." Flint: "(log in with the same password) … click **Install** (leave
  the other options as they are). The router's internet keeps working." The
  LuCI dialog shows an unticked **"Allow overwriting conflicting package
  files"** box. On the Fedora path nothing tells me whether to tick it. 01a does
  tell me the password, so that part is covered. (low)
- After I click Install, LuCI's top bar only has "Status System Services Network
  Logout". Nothing links back to the GL admin. Step 3 says "Back in the router's
  admin page", so I have to remember the GL address myself (the LuCI URL shown
  is `192.168.8.1:8080`). The packet has no screen for after Install, so I can't
  see how I'd know it worked. (low)
- The file name says `aarch64_cortex-a53_neon-vfpv4`, and I only see one
  download. Neither page says it fits my Slate 7. The Flint says "It's the same
  tailbond this router runs", which tells me about the Flint, not my Slate. On
  GL's site, packages differ by model, so I'd worry I picked the wrong one. (low)

## Task 2 — Pair with the home server

**Where I click.** VPN → tailbond. First I see "Looking for your home server on
your tailnet…" (02). A few seconds later: "Found your home server: homebox,
homebox.tail1234.ts.net · 100.101.1.1" → **"Ask to connect"** (03). Then 04:
"Waiting for approval on homebox… On homebox, tap **Approve** next to
**slate7**. This router connects by itself once approved; you can close this
page." This is close to GL's "one toggle" feel. The error states read well: 03b
"Turn on Tailscale first … Open Tailscale settings", and 03c "No home server
found on your tailnet. Checked 4 online machines…".

**Where I hesitate.**
- In 04, the top status card still says "Not connected to a home server / Not
  set up yet. Connect this router to your home server below." That's while the
  card below says it's waiting for approval. It's a small mixed message. (low)

## Task 3 — Hotel Wi-Fi (Repeater) + phone USB tethering, both in use

**Where I look.** 05: the top card says "Connected to homebox — Using Repeater
(Wi-Fi) + Tethering (phone)". Under "Internet connections", the Repeater card
says "In use ↓38 Mbps ↑4.2 Mbps · 24 ms" and the Tethering card says "In use
↓21 Mbps ↑2.9 Mbps · 48 ms". That confirms both are used. 06 shows a problem
clearly: "Tethering (phone) not working right now", and "It's connected, but
can't reach homebox. tailbond keeps trying; the other connections carry your
traffic."

**Where I hesitate.**
- Top card, desktop: "↓ 59 Mbps  ↑ 7.1 Mbps  Up 2 h 13 min". "Up" sits right
  after the ↑ upload number, so I first read it as upload, not uptime. (low)

## Task 4 — Phone as backup only

**Where I click.** The Tethering card → "Use it" → **"Backup only"** (07). The
badge changes to "Backup · saving data" and the text says "Used only if your
other connections stop working. Until then it isn't used at all, so it uses no
data." The top card says "Tethering (phone) on standby". The hint under "Speed
or reliability" also points me there: "Saving your phone's data? Set Tethering
or Cellular to 'Backup only' under Internet connections." Clear.

**Where I hesitate.**
- I clicked "Change limit" (07b): "5 GB a month … Counts everything tailbond
  sends and receives over it. At the limit, tailbond stops using this
  connection…". Two questions remain. If I hit 5 GB and the hotel Wi-Fi then
  dies, am I just offline? And this count won't match my carrier's count. Also,
  "Most reliable — Sends everything over every connection. Uses the most data."
  doesn't say whether it overrides "Backup only" on the phone. (low)

## Task 5 — Reach the NAS by its MagicDNS name

**Where I look.** "Tailnet machines" card (05): **nas**, `nas.tail1234.ts.net`,
`100.101.2.2`, "linux · Online · direct", with a **"Copy name"** button. The
Home server card says: "Tailnet names — Work on every device connected to this
router while tailbond is on. Use the full name, like nas.tail1234.ts.net." The
answer is `nas.tail1234.ts.net`.

**Where I hesitate.**
- On my Tailscale devices, just typing `nas` works. The page says I need the
  full name. The page is clear and Copy name helps, so this only catches you if
  you skip the text. (low)

## Task 6 — Turn tailbond off

**Where I click.** The toggle at the top right of the status card (05). 09:
"tailbond is off — Devices on this router use the internet directly, without
your home server. Turn on to bond your connections again." That's exactly the
GL one-toggle feel. "Unpair" is separate, and its confirm dialog (09b) explains
the difference: "forgets the home server … it will need to approve".

**Where I hesitate.**
- 10-not-running says "tailbond isn't running on this router — It may have been
  turned off or stopped after an error." Turning it off with the toggle gives
  me 09, not 10. If I ever see 10, "turned off" makes me think the toggle did
  it. (low)

## Task 7 — Approve the travel router at home

**Flint 3 (11).** Under "Waiting for your approval" (badge "1 new"): "slate7
wants to connect — Travel router on your tailnet · signed in as
alex@example.com · GL-BE3600 · 100.101.9.9 · asked just now" → **"Approve"**.
12 then shows slate7 under Devices: "Connected just now · 2 connections bonded".
Easy.

**Fedora (14).** The orange "Waiting for your approval" card at the top with
**"Approve"**. Easy. 04 even told me I can approve from a phone: "open
http://homebox:8077 on any device signed in to your Tailscale account (a phone
works too)".

**Where I hesitate.**
- I approved from my phone (14-phone). Right under Approve is a big "Add this
  phone" card with a large green "Add this phone" button: "Two steps, then
  tailbond takes over from Tailscale on this phone." I only came to approve the
  router. On a small screen I could tap the wrong big button, and "takes over
  from Tailscale" sounds like a big change. (low)

## Task 8 — Hotel Wi-Fi with a sign-in page (GL Login Mode)

**Where I click.** 16 shows a blue banner: "Repeater needs you to sign in — GL's
Repeater page handles this sign-in. Open it, click 'Login Mode for Public
Hotspots' (or 'Go to Login') and accept the hotel's terms." → **"Open GL's
Repeater page"**. It also says "Afterwards tailbond adds Repeater back to
bonding by itself." I know GL's Login Mode, so that part is familiar. Then 16b:
"Signing in to Repeater through GL's Login Mode … It ends by itself once you've
signed in (or click 'Quit Login Mode' on GL's Repeater page)". I can finish.

**Where I get confused.**
- On 16b the top card says **"Router connected to homebox · devices signing
  in"**. It still shows "↓ 21 Mbps ↑ 2.9 Mbps", and the Tethering card says
  "In use". But the banner says "all traffic from devices here goes out the
  hotel Wi-Fi directly (not just the sign-in page), not through your home
  server". It then adds "The router itself stays connected to your home server
  over your other connections (that's what the top card shows), so they keep
  being used." I had to read it three times. "They keep being used" means my
  phone data is being spent while my laptop isn't going through homebox at
  all. And the green "Connected to homebox"-style card tells me things are
  normal. I'd worry about my phone data here and can't tell what's using it.
  (medium)

## Task 9 — Hotel Ethernet with a sign-in page

**Where I click.** 17 shows an orange banner: "Ethernet needs you to sign in —
GL's Login Mode covers the repeater Wi-Fi only; tailbond handles sign-in pages
on other connections like this one." → **"Open sign-in page"**. 17b: "Sign in
on the page that opened (accept the terms, enter the room number…)" → after
signing in, **"I've signed in, check now"**. There's also "Stop sign-in mode".
Clear, and it tells me why GL's Login Mode doesn't apply here.

**Where I hesitate.**
- "For up to 9 more minutes (until 04:40 on this router's clock)". I'm a
  traveller, so the router's clock may not be in my time zone. "9 more minutes"
  is enough; the clock time just made me check my watch for nothing. (low)

## Task 10 — Home side set up for best speed

**Flint 3.** "Setup health" card: 18c says "All good — Everything is set up for
best speed", with checks for "Tailscale: 3 machines direct", "Network is
friendly to direct connections", "Route 10.77.0.0/24 approved", "Public internet
address", and "UDP 443 reached from the internet". 13 (fresh setup) has a
"Finish setting up" card with "Step 1: allow tailbond devices on your Tailscale
network … Open admin console" and "Step 2: tell travel routers where home is …
turn on APPLICATIONS → Dynamic DNS". 18 (relayed): "On this router: NETWORK →
UPnP, turn it on." These are GL-specific, which I like.

**Fedora.** 15 Devices tab: "Setup health: 2 to improve for best speed", with
"Tailscale: 2 machines direct, 1 through a relay (slower)" and "This network
makes direct Tailscale connections hard", each with "How to fix" links.

**Where I get confused.**
- 18b (carrier-grade NAT). The top card says "Home server is on · 1 setup step
  left", and the red Step 1 says "travel routers can't reach UDP 443 here … from
  away they can't until this is fixed". But the Setup health card on the same
  page says **"1 thing to improve for best speed (tailbond works meanwhile, just
  slower)"**. One says it won't work away from home, the other says it works,
  just slower. If I only read the health card ("for best speed" is my task),
  I'd go on my trip thinking it's fine. (medium)
- 13 Step 1 ends with "Devices can already reach your machines without it."
  That makes the Tailscale admin-console step sound optional, and I'd likely
  skip it. Leaving GL to tick a route in Tailscale's console is already the
  least "one toggle" part. (low)
- 18 (Flint) and 15 (Fedora) say "To check one: tailscale ping pixel-8". That's
  a command, and I have nowhere to type it. (low)
- Fedora 15: both health items give the same fix in full ("forwarding UDP port
  41641 to this machine, or by turning on UPnP on the router in front of it").
  It doesn't say where UPnP is if the router in front is a GL router (the Flint
  page does: "NETWORK → UPnP"). On 14, the summary bar is red ("Setup health: 2
  things to improve for best speed"), which looks like an error for something
  that only makes things slower. (low)

---

## Findings

| # | Task | Finding | Severity |
|---|---|---|---|
| 1 | 8 | 16b: top card "Router connected to homebox · devices signing in" still shows speeds, and Tethering is "In use", while the banner says device traffic "goes out the hotel Wi-Fi directly … not through your home server"; "(that's what the top card shows), so they keep being used" is hard to parse. I can't tell what's using my phone data during Login Mode. | medium |
| 2 | 10 | 18b: Setup health says "tailbond works meanwhile, just slower", but on the same page Step 1 says "travel routers can't reach UDP 443 here … from away they can't until this is fixed". Contradictory; I could leave on a trip thinking it works. | medium |
| 3 | 1 | Fedora page step 2 lacks the Flint's "(leave the other options as they are)". The LuCI dialog's unticked "Allow overwriting conflicting package files" is unexplained on the Fedora path. | low |
| 4 | 1 | LuCI (01b) has no link back to GL admin, and the packet has no screen after Install, yet step 3 says "Back in the router's admin page". | low |
| 5 | 1 | One download, `…aarch64_cortex-a53_neon-vfpv4.ipk`; nothing confirms it's right for my Slate 7 ("It's the same tailbond this router runs" is about the Flint). | low |
| 6 | 2 | 04: the status card still says "Not connected to a home server / Not set up yet" while it's waiting for approval. | low |
| 7 | 3 | "↑ 7.1 Mbps  Up 2 h 13 min": "Up" (uptime) next to the upload arrow reads as upload. | low |
| 8 | 4 | 07b limit: unclear what happens at "the limit" if the hotel Wi-Fi then fails, and whether "Most reliable — Sends everything over every connection" overrides "Backup only". | low |
| 9 | 5 | The short name `nas` won't work ("Use the full name, like nas.tail1234.ts.net"); a Tailscale user expects short names. It's explained, and Copy name helps. | low |
| 10 | 6 | 10: "It may have been turned off or stopped after an error" mixes up the toggle-off state (09) with a stopped service. | low |
| 11 | 7 | 14-phone: a big "Add this phone" button ("tailbond takes over from Tailscale on this phone") sits right under Approve when approving from a phone. | low |
| 12 | 9 | 17b: "until 04:40 on this router's clock" is meaningless to a traveller in another time zone. | low |
| 13 | 10 | 13 Step 1: "Devices can already reach your machines without it" makes the admin-console route step sound optional. | low |
| 14 | 10 | 15/18: "To check one: tailscale ping pixel-8" is a command a non-engineer can't run from these pages. | low |
| 15 | 10 | Fedora 15: the same fix is repeated in full for both items, with no GL location for UPnP. On 14 the summary bar is red for a slower-not-broken issue. | low |

Totals: 0 high / 2 medium / 13 low
