# Persona A: GL.iNet Tailscale traveller (round 2)

Who I am: I travel a lot and already use GL.iNet travel routers. I'm not a network engineer. I know GL's own Tailscale page (APPLICATIONS → Tailscale: one Enable toggle, Apply, a bind-account link). I expect new GL features to work the same way: one switch, not much to set up.

Screens viewed: phone versions of 01a–14, plus desktop versions of 05, 11, 13 and 15.

---

## Task 1: Install tailbond on the Slate 7 without SSH

**Where I'd click**
- `01a-install-gl-advanced-settings`: SYSTEM → Advanced Settings → **"Go To LuCI"**. The card says "LuCI is OpenWrt's own web interface. Use the same admin password to log in." That helps, because I've never logged in to LuCI and I'd have wondered which password to use.
- `01b-install-luci-upload`: The dialog "Manually install package / Really install the uploaded package?" shows `tailbond_0.1.0-r42_aarch64_cortex-a53_neon-vfpv4.ipk (6.1 MB)`. I click **Install**.

**Where I'd get stuck**
- Nothing on the Slate 7's screens tells me what to do *inside* LuCI. The 01a card only gets me to the LuCI door. The path "System → Software → Upload Package…" appears only on the home server's pages (`11`/`13` "Add a GL.iNet travel router" step 2, `15` step 2). If I set up the home side first and kept that page open, I'd manage. If I only had the Slate 7 in front of me, I'd be poking around a menu I've never seen. As a GL user I'd first look in **APPLICATIONS → Plug-ins** for an "upload" button, because that's where GL add-ons live.
- In `01b`, the checkbox "Allow overwriting conflicting package files" makes me hesitate. Should I tick it? Nothing says. I'd leave it unticked and hope.
- The file name `..._aarch64_cortex-a53_neon-vfpv4.ipk` means nothing to me. The home server page does say "the tailbond file ending in **_neon-vfpv4.ipk**", which helps me pick the right download, but only if I've seen that page.

**What I expected:** Upload under Plug-ins, or the same path printed on the 01a card ("then System → Software → Upload Package…").

## Task 2: Pair the Slate 7 with the home server

**Where I'd click**
- `02-just-installed`: "Looking for your home server on your tailnet…" with a spinner. The "Ask to connect" button is greyed out while it searches. Fine.
- `03-found-server`: "Found your home server: homebox / homebox.tail1234.ts.net · 100.101.1.1" and one big blue **"Ask to connect"**. This feels like GL's Tailscale page: one button. Very good.
- `04-waiting-approval`: "Waiting for approval on homebox… On homebox, tap **Approve** next to **slate7**. This router connects by itself once approved; you can close this page." Clear.
- `03b-tailscale-off`: "Turn on Tailscale first" plus an **"Open Tailscale settings"** button. That's exactly where I'd go. Good.
- `03c-no-server-found`: "Checked 4 online machines…" plus "Search again". Clear enough, and I'd type the name in the box.

**Where I'd get stuck**
- `04` tells me to approve on the "Flint 3 at home: its admin panel → APPLICATIONS → tailbond." If I'm doing this **at the hotel**, I don't know how to open my Flint's admin panel from here. The Fedora bullet gives me a link ("open http://homebox:8077 from any device on your Tailscale account (a phone works too)"). The Flint bullet gives me no link and no "from anywhere on your tailnet" hint. I'd be stuck until I got home, or I'd guess the Flint's 100.x address.
- The footer link "Setting up the router that stays at home? **Make this the home server**" is on the travel router too. "(you can switch back here)" calms me down, but I could see myself tapping it by mistake when I'm confused. Minor.

## Task 3: At the hotel, use hotel Wi-Fi (Repeater) and phone tethering, and confirm both are used

**Where I'd click:** Nowhere. `05-connected-two-wans` header: "Connected to homebox / Using Repeater (Wi-Fi) + Tethering (phone)", with a toggle at the right like GL's. Both cards show a green "**In use**" with live speeds ("↓ 38 Mbps ↑ 4.2 Mbps · 24 ms", "↓ 21 Mbps ↑ 2.9 Mbps · 48 ms"). I can confirm both are used at a glance. Nice.

**Small things**
- Ethernet shows "Not connected" but "Always" is selected. The text "It's used automatically once connected" explains it. OK.
- `06-one-wan-failed`: "Tethering (phone) not working right now" in the header, and on the card "It's connected, but can't reach homebox. tailbond keeps trying; the other connections carry your traffic." Reassuring, no action needed.
- On the phone, the three "Speed or reliability" cards are squeezed into narrow columns ("Smooth / calls / and / video"). Readable, but cramped.

## Task 4: Phone as backup only

**Where I'd click:** `05`, Tethering card → "Use it" → **"Backup only"**. The hint under Speed or reliability also points there: "Saving your phone's data? Set Tethering or Cellular to 'Backup only' under Internet connections." `07-save-data-standby` confirms it: the badge says "Backup · saving data" and the card says "Used only if your other connections stop working. Until then it isn't used at all, so it uses no data." The header says "Tethering (phone) on standby". Perfect, and it reads like plain English.

`07b-set-limit`: "Change limit" opens "5 GB a month / Save / Cancel" with "At the limit it pauses; the count restarts on the 1st of each month. 0 = no limit." Clear. I'd like to pick the reset day (my phone plan doesn't reset on the 1st), but that's polish.

## Task 5: Reach the NAS by its MagicDNS name from the laptop

**Where I'd look:** `05`, "Tailnet machines" → `nas` → `nas.tail1234.ts.net · 100.101.2.2` → **"Copy name"**. The Home server card also says "Tailnet names: Work on every device connected to this router while tailbond is on. Use the full name, like nas.tail1234.ts.net." The answer is `nas.tail1234.ts.net`. Easy.

**Small things**
- On the phone layout the full name wraps in the middle of the word: "nas.tail1234.ts.ne" / "t · 100.101.2.2" (same for apple-tv, pixel-8, work-laptop). If I typed it by hand from the phone screen I could easily type "ts.ne". "Copy name" saves me, and desktop shows it on one line.
- "Use the full name" tells me plain `nas` won't work. On my own Tailscale devices the short name usually works, so it's good that this is spelled out.

## Task 6: Turn tailbond off

**Where I'd click:** The toggle in the top card (`05`). `09-turned-off`: "tailbond is off. Devices on this router use the internet directly, without your home server. Turn on to bond your connections again." That's exactly GL-style. Unpair is a separate red button with a clear confirm (`09b`: "This router stops using tailbond and forgets the home server… (it will need to approve)"), so I won't mix the two up.

**Small things**
- When off, "Tailnet names: Work … while tailbond is on" and "The list appears once tailbond is connected." I still have GL's own Tailscale turned on, so I'm not sure whether the laptop can still reach the NAS through GL's Tailscale when tailbond is off. A one-liner would help.
- `08-server-unreachable` surprised me: "Meanwhile, devices on this router have no internet, so nothing goes around your home server. To use this internet directly for now, turn tailbond off." I assumed it would fall back to plain hotel internet. The screen is honest and gives me a **"Turn tailbond off"** button right there, so I wouldn't be stuck. It's still worth knowing before a trip.
- `10-not-running`: "Starting it is safe: devices here keep their internet while you set it up." with a **"Start tailbond"** button. Clear.

## Task 7: Approve the travel router on the home side

**Flint 3 (`11-server-pending-request`, phone + desktop):** At the top, "Waiting for your approval · 1 new / slate7 wants to connect / Travel router on your tailnet · signed in as alex@example.com / GL-BE3600 · linux · 100.101.9.9", with **Approve** / Decline. The model number and my own account email make me confident it's mine. One tap. `12-server-approved` lists slate7 under Devices as "Connected just now · 2 connections bonded". Header now says "2 connected now". Great.

**Fedora (`14`, phone; `15`, desktop):** The approval card is at the very top with **Approve**. Good. On the phone, though, the rest of the page is a huge "Add this phone" block ("tailbond takes over from Tailscale on this phone", "Download tailbond for Android", a big green "+ Add this phone"). I only opened this page on my phone to approve the router, and that block looks like something I'm supposed to do next. I wouldn't tap it, but it pulls my attention away and "takes over from Tailscale" sounds scary.

**Flint first-time setup (`13-server-route-pending`, phone + desktop):** "Home server is on · 2 setup steps left".
- Step 1: "Machines → homebox → Edit route settings → turn on **10.77.0.0/24**" plus **"Open admin console"**. That's more technical than GL's Tailscale page, but the exact clicks are there and I've done similar approvals for subnet routes. OK.
- Step 2: "Enter your home's public IP address or a Dynamic DNS name (APPLICATIONS → Dynamic DNS gives you one)". As a GL user I know GL DDNS, so fine.
- The **Port** line confuses me: "UDP 443, opened on this router automatically. If another router sits in front of this one (for example your internet provider's box), forward UDP 443 to this router there." My Flint is behind my ISP's modem/router, and I honestly don't know whether that counts as "another router in front". I don't know how to forward a port on the ISP box either. Nothing on the page tells me whether it's actually reachable from outside. `11` later shows "All set", but I can't tell whether that was tested or just means "you filled in the address". I'd likely find out the hard way, at the hotel, with `08` "Can't reach homebox".
- "Secure connection: This router's own certificate, checked by each device" (13) vs "Certificate from Tailscale for homebox.tail1234.ts.net" (11). It's jargon, I don't know if I should care, and nothing tells me to act. Ignored.

**Small inconsistencies:** The Slate 7 tells me "open http://homebox:8077" (04). The Fedora page's invite link is `http://homebox.tail1234.ts.net:80/#add-phone`, and its header says "reachable at homebox.tail1234.ts.net:443". Three addresses for the same box. The Fedora footer reads "tailbondd 0.1.0-r42" (double d). The Flint's "asked 1 min ago" (phone) vs "asked just now" (desktop) is just fixture timing.

---

## Overall

This round feels much closer to GL's Tailscale page than I feared. Pairing is one button, turning off is one switch, and "Backup only" is plain English. My real worries are getting into and around LuCI from the Slate 7 itself, approving on a Flint while I'm away from home, and not knowing whether my home's port setup actually works before I leave.

## Findings

| # | task | finding | severity |
|---|---|---|---|
| 1 | 1 | On the Slate 7, `01a` only says "Go To LuCI". The steps inside LuCI ("System → Software → Upload Package…") appear only on the home server's pages, so a GL user who has never seen LuCI is left exploring an unfamiliar menu. Expected an upload under APPLICATIONS → Plug-ins, or the full path on the 01a card. | medium |
| 2 | 2 | `04` waiting screen: the Flint bullet ("its admin panel → APPLICATIONS → tailbond") gives no way to reach the Flint's panel when I'm not home. The Fedora bullet has a tailnet link; the Flint one doesn't. Pairing from the hotel stalls. | medium |
| 3 | 7 | `13` Port line: "If another router sits in front of this one (for example your internet provider's box), forward UDP 443…". I can't tell whether that applies to me, and nothing checks that the port is reachable from outside before I travel. "All set" in `11` doesn't say whether it was tested. | medium |
| 4 | 1 | `01b` "Allow overwriting conflicting package files" checkbox has no guidance (tick or not?). | low |
| 5 | 5 | Phone layout breaks MagicDNS names mid-word ("nas.tail1234.ts.ne" / "t · 100.101.2.2"), which is easy to mistype if read by eye. "Copy name" mitigates it. | low |
| 6 | 6 | When tailbond is off, it's unclear whether the laptop can still reach the NAS through GL's own Tailscale ("Tailnet names: Work … while tailbond is on"). | low |
| 7 | 6 | `08`: home server down means "devices on this router have no internet". This surprised me (I expected fallback), but it's clearly explained and there's a "Turn tailbond off" button. | low |
| 8 | 7 | Fedora phone page (`14`): after the Approve card, a large "Add this phone" block ("tailbond takes over from Tailscale on this phone") draws attention when I only came to approve. | low |
| 9 | 7 | Three different addresses for the Fedora page: `http://homebox:8077` (04), `homebox.tail1234.ts.net:80` (15 invite), `…:443` (15 header). | low |
| 10 | 7 | "Secure connection" wording differs between `13` ("This router's own certificate, checked by each device") and `11` ("Certificate from Tailscale…"). Jargon with no action for me. | low |
| 11 | 2 | "Make this the home server" link on the travel router's page could be tapped by mistake. "(you can switch back here)" softens it. | low |
| 12 | 3 | Phone layout: "Speed or reliability" cards are squeezed into narrow columns ("Smooth / calls / and / video"). | low |
| 13 | 4 | Data limit always resets "on the 1st of each month". My phone plan resets on a different day. | low |
| 14 | 7 | Fedora footer typo "tailbondd 0.1.0-r42". | low |
