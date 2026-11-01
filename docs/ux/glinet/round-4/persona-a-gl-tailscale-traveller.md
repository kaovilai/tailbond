# Round 4: persona A, GL.iNet Tailscale traveller (raw report)

Persona: I travel a lot and I'm not a network engineer. I already use GL.iNet travel
routers and know GL's own pages: APPLICATIONS → Tailscale (one Enable toggle) and the
repeater's "Login Mode for Public Hotspots" for hotel sign-in pages. I expect new GL
features to feel like those.

Inputs: task-script.md, screens.md, round-4 PNGs (all phone variants, plus desktop for
01b, 05, 11, 13, 15, 16, 17, 17b, 18).

---

## Task 1: Install tailbond on the Slate 7 without SSH

Where I click:
- I start from the instructions on the home side, "Add a GL.iNet travel router" (11/13/18
  desktop): "On the travel router's admin page: **SYSTEM → Advanced Settings → Go To LuCI**
  (log in with the same password), then **System → Software → Upload Package…**, pick
  the file and click **Install** (leave the other options as they are). The router's
  internet keeps working."
- 01a phone: "SYSTEM / Advanced Settings", "LuCI is OpenWrt's own web interface. Use the
  same admin password to log in." → **Go To LuCI**.
- 01b (desktop shows the full LuCI page): "Software" → "Upload Package…" → dialog "Manually
  install package / Really install the uploaded package?
  tailbond_0.1.0-r42_aarch64_cortex-a53_neon-vfpv4.ipk (6.1 MB)" → **Install**, leaving
  "Allow overwriting conflicting package files" unticked, as told.

What tripped me up / what I expected:
- I've never been in LuCI. GL normally installs extras from APPLICATIONS → Plug-ins, so
  this feels less like GL than I'd like. The step-by-step text matches what's on screen
  exactly, though, and "Use the same admin password" took care of my login worry. I got
  through it. (low)
- No screen shows what happens after **Install**: did it work, and where do I go now?
  Step 3 of the home-side text says "open VPN → tailbond", so I'd look there. In GL,
  Tailscale lives under APPLICATIONS, so VPN isn't where I'd look first. The text tells
  me, so it's only a small thing. (low)
- The Fedora page's version of the steps (15) is shorter. It drops "(log in with the
  same password)" and "leave the other options as they are", and it doesn't say this
  package is the right one for my Slate 7. The file name "aarch64_cortex-a53" means
  nothing to me. The Flint 3 page at least says "It's the same tailbond this router
  runs." (low)

## Task 2: Pair the Slate 7 with the home server

Where I click: 02 "Looking for your home server on your tailnet…" → 03 "Found your home
server: homebox" → **Ask to connect** → 04 "Waiting for approval on homebox… On
homebox, tap **Approve** next to **slate7**. … you can close this page."

- This is smooth. It's not a single toggle like GL's Tailscale page, but it's one obvious
  button, and 04 tells me exactly where to approve ("open http://homebox:8077 … At home
  with a Flint 3, its admin panel → VPN → tailbond works as well.").
- 03b "Turn on Tailscale first" with an "Open Tailscale settings" button is how I'd
  expect GL to point me to its own Tailscale page. 03c "No home server found… Checked 4
  online machines" also makes sense.
- Small thing: the footer link "Setting up the router that stays at home? Make this the
  home server" sits right under the pairing UI. I wouldn't click it, but a rushed user
  might. (low)

## Task 3: Hotel Wi-Fi (repeater) plus phone tethering, both in use

Where I look: 05, top card "Connected to homebox / Using Repeater (Wi-Fi) + Tethering
(phone)". Under Internet connections, both Repeater and Tethering show a green "In use"
badge with their own speeds ("↓ 38 Mbps ↑ 4.2 Mbps · 24 ms", "↓ 21 Mbps ↑ 2.9 Mbps ·
48 ms").

- Clear, easy. 06 also makes sense: "Tethering … Not working right now … It's
  connected, but can't reach homebox. tailbond keeps trying; the other connections carry
  your traffic."
- I still have GL's own Tailscale (and maybe GL's Multi-WAN) turned on. Nothing on the
  page tells me whether they conflict with tailbond or whether I should leave them alone.
  I'd just leave them, so this is only a lingering doubt. (low)

## Task 4: Phone as backup only

Where I click: 05/07, Tethering card, "Use it" → **Backup only**. 07 then shows
"Backup · saving data / Used only if your other connections stop working. Until then it
isn't used at all, so it uses no data." The top card says "Tethering (phone) on standby".

- Easy, and the hint under Speed or reliability ("Saving your phone's data? Set
  Tethering or Cellular to 'Backup only'") points the same way. 07b's limit editor ("5 GB
  a month … 0 = no limit") is clear as an extra. No issues.

## Task 5: Reach the NAS by its MagicDNS name

Where I look: 05, Tailnet machines: "nas / nas.tail1234.ts.net / 100.101.2.2 · Online ·
direct", with **Copy name**. The Home server card says: "Tailnet names: Work on every
device connected to this router while tailbond is on. Use the full name, like
nas.tail1234.ts.net."

- Found it right away. On my Tailscale laptop I'd normally just type "nas". The page
  tells me to use the full name, and Copy name makes that easy. (low: I'd probably try
  the short name first and then read the hint)

## Task 6: Turn tailbond off

Where I click: the blue toggle in the top card (05) → 09 "tailbond is off / Devices on
this router use the internet directly, without your home server. Turn on to bond your
connections again."

- Just like GL's Tailscale Enable toggle. Good. When the server can't be reached (08),
  the "Turn tailbond off" button and the bold text "Meanwhile, devices on this router have
  no internet … turn tailbond off" are very helpful.
- The red "Unpair" sits next to "Ask to connect again" on every connected screen. I might
  think "Unpair" means "off". The 09b confirmation ("This router stops using tailbond and
  forgets the home server … it will need to approve") would stop me, though. (low)
- "Ask to connect again" shows even while I'm connected, and I don't know why I'd ever
  press it. (low)

## Task 7: Approve the travel router at home

Where I click: on the Flint 3 (11), "Waiting for your approval · 1 new / slate7 wants to
connect / GL-BE3600 · linux · 100.101.9.9" → **Approve**. 12 then shows slate7 under
Devices, "Connected just now · 2 connections bonded". On Fedora (14 phone / 15 desktop),
the same card is at the top, with **Approve**.

- Clear on both. The orange "1 new" badge works well.
- 14 phone: right under the approval card there's a big "Add this phone … Get the
  tailbond app / Download tailbond for Android" card. I'm only there to approve my
  router, so for a moment I'd wonder whether I also have to install something on my
  phone. (low)
- "Who can approve: anyone on your Tailscale account who opens http://homebox:8077" is
  good to know. Not a problem for me. (low)

## Task 8: Hotel Wi-Fi has a sign-in page (GL repeater detected it)

Where I click: 16, blue banner "Repeater needs you to sign in / GL's Repeater page
handles this sign-in. Open it, click 'Login Mode for Public Hotspots' (or 'Go to
Login') and accept the hotel's terms. tailbond steps aside while you sign in, and adds
Repeater back to bonding by itself afterwards." → **Open GL's Repeater page**. 16b then
shows "Signing in through GL's Login Mode now: tailbond is letting devices here reach
the sign-in page directly." and Repeater shows "Signing in…".

- This is exactly what I hoped for: it uses GL's own Login Mode, the one I already know,
  in GL's own words.
- No screenshot shows the "done, bonded again" state. I trust "adds Repeater back to
  bonding by itself", but I'd keep refreshing to check. (low)

## Task 9: Hotel Ethernet has a sign-in page

Where I click: 17, orange banner "Ethernet needs you to sign in … Opening the sign-in
page lets devices here reach Ethernet directly (without your home server) for up to 10
minutes, so the page can load." → **Open sign-in page**. 17b: "Signing in to Ethernet…
Until 04:06, devices here use Ethernet directly … tailbond adds it back by itself once
you're signed in." Buttons: "Open the sign-in page again", "I've signed in, check now",
"Stop".

- Clear. GL has no Login Mode for cables, so I'm glad tailbond handles this itself.
- "Until 04:06": is that a clock time (4:06 AM?) or a countdown (4 min 6 s)? The
  previous screen said "up to 10 minutes", so I'd guess countdown, but I'm not sure.
  "for another 4:06" or "until 14:06" would settle it. (low)

## Task 10: Home side set up for best speed

Where I look: the Flint 3 "Setup health" card (11/12: "All good / ✓ Everything is set up
for best speed", with green ticks).

Where I got confused:
- **13 contradicts itself.** The top says "Home server is on · 2 setup steps left" and
  shows two orange "Step 1 / Step 2" warnings. Yet Setup health has the badge "0 to
  improve" and the green line "✓ Everything is set up for best speed", followed by two
  "?" items ("Behind your home router: Forward UDP 443 on that router to 192.168.8.1",
  "UDP 443: not confirmed from outside yet"). Is my setup fine or not? For "make sure
  everything is set up", I'd half believe the green line and might skip the steps.
  (medium)
- **18b contradicts itself.** "Internet and Tailscale" shows a green "Setup done: Travel
  routers and phones can connect from anywhere". Further down, Setup health says "Your
  internet line uses carrier-grade NAT … shared by your provider, so travel routers can't
  reach UDP 443 here. Ask your provider for a public IPv4 address". Those can't both be
  true. The big green "Setup done" box is what I'd notice and trust, so I'd leave home
  thinking it works. (medium)
- **18 is written for network engineers.** "Tailscale: 2 machines direct, 2 via relay
  (slower) … Check with: tailscale ping slate7 … forward UDP 41641 to this machine or turn
  on UPnP/NAT-PMP on your router, and avoid double NAT." and "Hard NAT on this network".
  I'm reading this on my Flint 3, which *is* my router, so "this machine" vs "your router"
  doesn't tell me which box to change. I have no shell for "tailscale ping". I'd click
  "How to fix" and hope. GL's own pages would say something like "Enable UPnP under
  NETWORK → …". (medium)
- 13 Step 2 points at "APPLICATIONS → Dynamic DNS". That's a GL page I know. Good.
- 15 (Fedora, Devices tab): the manifest says it includes setup health, but I can't see
  any health card on that tab, only Add router / Add device / Invite / device list. If
  my home server were the Fedora box, I wouldn't know where to check "best speed". I'm a
  GL person with a Flint 3, so I'd use the Flint page instead. (low)

---

## Findings

| # | task | finding | severity |
|---|---|---|---|
| 1 | 10 | 13: top says "2 setup steps left" with two warnings, but Setup health says "0 to improve" and "✓ Everything is set up for best speed" above two "?" items. I can't tell whether setup is done. | medium |
| 2 | 10 | 18b: green "Setup done: Travel routers and phones can connect from anywhere" contradicts the health item "carrier-grade NAT … travel routers can't reach UDP 443 here". I'd trust the green box and leave with a setup that doesn't work. | medium |
| 3 | 10 | 18: health text is jargon ("Hard NAT", "UPnP/NAT-PMP", "double NAT", "forward UDP 41641", "Check with: tailscale ping slate7"), and on the Flint 3, which is the router, "this machine" vs "your router" doesn't say which box to change. | medium |
| 4 | 1 | Install goes through LuCI ("Go To LuCI" → System → Software → Upload Package…), not GL's APPLICATIONS → Plug-ins. Unfamiliar, but the step text matches the screens. | low |
| 5 | 1 | No post-install screen. tailbond appears under VPN, while GL's Tailscale lives under APPLICATIONS. Only the home-side step 3 tells me where to look. | low |
| 6 | 1 | Fedora page steps (15) drop "log in with the same password" and "leave the other options", and don't reassure me the "aarch64_cortex-a53" package fits the Slate 7. | low |
| 7 | 2 | "Make this the home server" link sits right under the pairing UI and could be misclicked. | low |
| 8 | 3 | Nothing says whether GL's own Tailscale or Multi-WAN should stay on alongside tailbond. | low |
| 9 | 5 | I'd try the short MagicDNS name "nas" first. The "Use the full name" hint and Copy name fix it. | low |
| 10 | 6 | Red "Unpair" next to "Ask to connect again" could be mistaken for "off" (the confirm dialog catches it). "Ask to connect again" shows while already connected, with no reason given. | low |
| 11 | 7 | 14 phone: big "Add this phone / Download tailbond for Android" card under the approval card makes me wonder whether I must also install something. | low |
| 12 | 8 | No screen confirms Repeater is back in bonding after GL Login Mode sign-in. | low |
| 13 | 9 | "Until 04:06" is ambiguous (clock time or countdown?) after "up to 10 minutes". | low |
| 14 | 10 | Fedora Devices tab (15) shows no setup health card despite the manifest. I'd not know where to check best speed on Fedora. | low |

Counts: high 0, medium 3, low 11.
