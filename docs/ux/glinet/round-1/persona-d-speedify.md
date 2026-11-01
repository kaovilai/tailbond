# Persona D (long-time Speedify user): GL.iNet round 1

Background: I've used Speedify for years. My mental model is: the app's main screen says "Using 2 connections", each connection has a priority (Always / Secondary / Backup / Never) and an optional monthly cap, there's a mode (Speed / Redundant / Streaming), and every connection gets its own live usage graph. I'm checking whether tailbond on a Slate 7 feels like "Speedify, but to my house".

## Task 1: Install without SSH

**Where I'd click:** `01-not-installed-luci-upload`. LuCI → System → Software → "Upload Package…", pick `tailbond_0.1.0-r42_aarch64_cortex-a53_neon-vfpv4.ipk`, then "Install". The dialog text helps: "for GL.iNet routers the file ends in **_aarch64_cortex-a53_neon-vfpv4.ipk**". It tells me I downloaded the right file.

**Where I'd get stuck:**
- I only know about the LuCI route because the manifest describes it ("GL admin → SYSTEM → Advanced Settings → Go To LuCI"). Nothing in the GL panel screenshots points there. The GL sidebar has "APPLICATIONS → Plug-ins", and that's where I'd look first for "install an app". No screenshot shows Plug-ins accepting an uploaded file. I'd probably find LuCI eventually, but the path isn't on screen. Speedify installs like any normal app, so a hidden second admin UI is a real detour for me.
- "Allow overwriting conflicting package files" is already checked and greyed out. It sounds alarming ("overwriting"?), but I can't change it, so I'd just continue.
- No screenshot shows what happens after "Install". Nothing says "now open GL admin → APPLICATIONS → tailbond". I had to guess that it appears in the GL sidebar, which screen 02 later confirms.

**Expected:** an install through GL's own Plug-ins page, or at least a line in the GL UI telling me where to go next.

## Task 2: Pair with the home server

**Where I'd click:** `02-just-installed` shows "Looking for your home server on your tailnet…". A few seconds later, `03-found-server` shows "Found your home server: homebox" with "homebox.tail1234.ts.net · 100.101.1.1". I'd click the blue "Ask to connect". `04-waiting-approval` then says "Waiting for approval on homebox…", followed by "Open homebox's tailbond page (on a Flint 3: its admin panel → APPLICATIONS → tailbond) and tap **Approve** next to **slate7**." That is very clear and better than I expected. Speedify is just an account login, but this approval flow tells me exactly what to do next.

**Minor notes:**
- `03-found-server` has two "Ask to connect" buttons: the blue one on the found server and a greyed one next to the "Not this one?" field. It's obvious which one to use, so this is polish only.
- The footer link "Is this the router at home? Use it as your home server instead" sits right under the pairing UI. I wouldn't click it, but a hurried user might.
- `03b-tailscale-off` ("Turn on Tailscale first", "Open Tailscale settings") and `03c-no-server-found` ("Checked 4 online machines…") are clear and give me a next step.

## Task 3: Hotel Wi-Fi + phone tethering, confirm both are used

**Where I'd click:** I'd set up Repeater and USB tethering in GL's own INTERNET pages (not shown). Then I'd open APPLICATIONS → tailbond (`05-connected-two-wans`). The header says "Connected to homebox" and "Using Wi-Fi (repeater) + phone (USB tethering)" with "↓ 59 Mbps ↑ 7.1 Mbps Up 2 h 13 min". That's my Speedify "Using 2 connections" moment, and I'd consider the task done. Both cards say "In use", with per-connection numbers: Repeater "↓ 38 Mbps ↑ 4.2 Mbps · 24 ms", Tethering "↓ 21 Mbps ↑ 2.9 Mbps · 48 ms". 38 + 21 = 59, which tells me bonding is actually happening.

**Gaps compared with Speedify:**
- There are no per-connection live graphs on the travel router, only small grey numbers. Speedify shows each connection's graph on the main screen. Funnily, the Fedora home page (`14-fedora-dashboard-request`) does have graphs and per-path cards ("Path 0 / Path 1", DOWN/UP/LATENCY/LOSS), but the device I'm actually holding at the hotel doesn't.
- The Ethernet card shows "Not connected" but still has the "Use" toggle on and a "Backup only" checkbox. That's fine, just a bit of noise.
- The card names don't match the header: the header says "Wi-Fi (repeater)" and "phone (USB tethering)", the cards say "Repeater" and "Tethering". Minor.
- `06-one-wan-failed` is good. Tethering shows "Not working right now" with "It's connected, but can't reach homebox. tailbond keeps trying; the other connections carry your traffic." Speedify gives me about the same.

## Task 4: Phone as backup only

**Where I'd click:** I'd tick "Backup only" on the Tethering card (`07-save-data-standby`). The badge becomes "Backup · saving data", the card says "Used only if your other connections stop working.", and the header says "phone (USB tethering) on standby". This maps to Speedify's "Backup" priority, so I'm happy with the result.

**Where I'm confused:**
- There's a second control that seems to do the same thing: the mode card "Save mobile data — Uses Ethernet and Wi-Fi; your phone and cellular only when those stop working." In Speedify, priority (per connection) and mode (Speed / Redundant / Streaming) are separate settings. Here the mode list mixes them. In `07` I have "Backup only" ticked while the mode is still "Fastest — Adds all your connections together for the most speed." Does "Fastest" override "Backup only"? Do I need both? Would "Save mobile data" mark Tethering as backup without me ticking the box? The screen doesn't say how the two interact. I'd set both "to be safe", and then lose the Fastest mode for the Wi-Fi without knowing whether it mattered.
- The "Use" toggle is effectively Speedify's "Never", and "Backup only" is "Backup". There's no "Secondary" equivalent (use when the primary is slow, not only when it's dead). That's fine for this task, just less granular.
- Data cap (`07b-set-limit`): "Change limit" opens "5 GB a month" with the note "Counts everything tailbond sends and receives over it. At the limit it pauses until next month. 0 = no limit." It's clear and close to Speedify's monthly cap. But "next month" means what: the calendar month? Speedify lets me pick the reset day to match my phone bill. It's also clear the cap counts only tailbond's traffic, not my phone's overall usage, which is fine.

## Task 5: Reach the NAS by MagicDNS name

**Where I'd click:** On `05` I'd scroll to "Tailnet machines" ("Laptops and phones on this router's Wi-Fi or LAN reach these by name, e.g. nas.tail1234.ts.net."). The NAS row shows `nas.tail1234.ts.net · 100.101.2.2` with a "Copy name" button. The name is **nas.tail1234.ts.net**. That's easy. Speedify has nothing like this, so it's a plus.

**Minor notes:**
- In the Home server section, the row label "Names like nas.…ts.net" with the value "Work on every device connected to this router while tailbond is on" reads like a broken sentence spread across two columns.
- It doesn't say whether the short name `nas` also works from the laptop, or only the full name.
- On the phone layout (`05-connected-two-wans-phone`), the IPs wrap mid-number ("100.101. / 2.2", "10 / 0.101.4.4"). That's ugly and could lead to copying a wrong IP.
- When tailbond is off or can't connect, the list is replaced with "The list appears once tailbond is connected." (`08`, `09`). That makes sense.

## Task 6: Turn tailbond off

**Where I'd click:** The big toggle at the top right of the status card. `09-turned-off` shows "Off" and "Off. Devices on this router use the internet directly, without your home server." That's clear and matches Speedify's big on/off switch. "Unpair" is separate and has a confirmation dialog (`09b-unpair-confirm`: "This router stops using tailbond and forgets the home server… (it will need to approve)"). So even if I hit the wrong red button, I'd be warned.

**Minor notes:**
- The title "Off" followed by "Off. Devices on this router…" repeats itself.
- `10-not-running` says "tailbond isn't running on this router — It may have been turned off or stopped after an error" with "Start tailbond". After using the toggle, I'm not sure which of `09` or `10` I'd see, or whether "turned off" means my toggle. Small doubt only.
- While off, the connection cards still show "Ready" and "Used when tailbond is on.", which is fine.

## Task 7: Approve on the home side

**Where I'd click:** On the Flint 3 (`11-server-pending-request`), under "Waiting for your approval", the "1 new" card shows "slate7 wants to connect", "Travel router on your tailnet · signed in as alex@example.com", "GL-BE3600 · linux · 100.101.9.9 · asked just now". I'd click "Approve". After that, `12-server-approved` lists "slate7 Connected just now · 2 connections bonded · ↓ 48 Mbps ↑ 6.1 Mbps". On Fedora (`14-fedora-dashboard-request`), the same "slate7 wants to connect" banner sits at the top with "Approve". Both are easy. The name "slate7" matches what the Slate told me in `04`.

**Where I'm confused:**
- Both home pages also have a mode picker. The Fedora page shows "Fastest / Smooth calls / Most reliable / Save mobile data ('Backup only on failure')", and the Flint's tailbond page from `05` has the same card set. In Speedify, the mode belongs to the client. Here I can't tell whether the home server's mode or the travel router's mode decides how slate7's traffic is handled, or whether the server's "Save mobile data" changes my phone's backup setting. The labels also differ between the two pages ("Smooth calls" with "Calls and video stay up" vs "Smooth calls and video"; "Every packet twice" vs "Sends everything over every connection"). With these two pages open side by side, I'd wonder which one I'm actually configuring.
- `13-server-route-pending` (freshly set up Flint): "One step left in the Tailscale admin console" with "Open admin console", plus "Set this router's internet address". These are clear and actionable, but it's extra setup that a Speedify user never sees. The Approve flow itself isn't blocked by them.

## Findings

| # | task | finding | severity |
|---|---|---|---|
| 1 | 1 | The LuCI install route isn't discoverable from the GL panel; I'd look in "APPLICATIONS → Plug-ins" first and nothing points to SYSTEM → Advanced Settings → Go To LuCI | medium |
| 2 | 1 | No post-install screen or hint telling me tailbond is now under APPLICATIONS → tailbond | low |
| 3 | 1 | Pre-checked, greyed "Allow overwriting conflicting package files" sounds alarming | low |
| 4 | 2 | Two "Ask to connect" buttons on `03-found-server` (one greyed) | low |
| 5 | 2 | "Use it as your home server instead" link right under the pairing flow could tempt a hurried user | low |
| 6 | 3 | No per-connection live graphs on the travel router (only small numbers), although the Fedora page has them | low |
| 7 | 3 | Header says "Wi-Fi (repeater)" / "phone (USB tethering)" but the cards say "Repeater" / "Tethering"; "Not connected" Ethernet still shows an active "Use" toggle | low |
| 8 | 4 | "Backup only" (per connection) and the "Save mobile data" mode overlap; with "Backup only" ticked and the mode on "Fastest" it's unclear which wins or whether both are needed | medium |
| 9 | 4 | No middle priority like Speedify's "Secondary"; only Use on/off and Backup only | low |
| 10 | 4 | Data cap "pauses until next month": reset day unspecified and not configurable to match the phone bill | low |
| 11 | 5 | "Names like nas.…ts.net" / "Work on every device…" row reads as a broken sentence | low |
| 12 | 5 | Unclear whether the short name `nas` works or only `nas.tail1234.ts.net` | low |
| 13 | 5 | Phone layout wraps IPs mid-number ("100.101. 2.2") | low |
| 14 | 6 | "Off" title + "Off. Devices…" is redundant; unclear how toggle-off relates to the "tailbond isn't running" screen | low |
| 15 | 7 | Mode picker exists on both home server and travel router with different wording; unclear which side's mode controls slate7's traffic | medium |
| 16 | 7 | Fresh home server needs extra Tailscale-console route approval and an internet address (clear, but extra steps) | low |
