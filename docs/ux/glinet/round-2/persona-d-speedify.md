# Persona D (long-time Speedify user): GL.iNet round 2, raw report

Background: I've used Speedify for years. I think in Speed / Redundant / Streaming modes,
per-connection priority (Always / Secondary / Backup / Never), monthly caps per connection,
live per-connection graphs, and the "Using 2 connections" line. I'm comparing tailbond on a
Slate 7 against that.

Overall: this is much closer to Speedify than I expected. The connection cards, the
"Use it: Always / Backup only / Off" control, the per-connection data cap and the three modes
map almost one-to-one onto what I know. I finished every task without getting stuck. My
findings are mostly about Speedify features I missed and a few unclear numbers or labels.

---

## Task 1: Install on the Slate 7 without SSH

**Where I'd click:** `01a-install-gl-advanced-settings`: SYSTEM → Advanced Settings → "Go To LuCI"
("Use the same admin password to log in"). Then in `01b-install-luci-upload`: System → Software →
"Upload Package…", pick `tailbond_0.1.0-r42_aarch64_cortex-a53_neon-vfpv4.ipk`, and click "Install" in
"Manually install package / Really install the uploaded package?".

**Friction:**
- I only know which file to download because the home server's page tells me ("the tailbond file
  ending in **_neon-vfpv4.ipk**", screen 11/13, and "Download the router package" on Fedora, screen 15).
  If I went to a releases page first, `aarch64_cortex-a53_neon-vfpv4` means nothing to me. Speedify is
  a single download for each platform. (low)
- The LuCI dialog has a checkbox, "Allow overwriting conflicting package files". Nothing tells me
  whether to tick it. I'd leave it unticked and hope. (low)
- No screen shows how to get from LuCI back to the GL panel after installing. I'd guess I should
  re-open 192.168.8.1. Screen 02 shows that APPLICATIONS → tailbond does appear afterwards. (low)

**Expected (Speedify):** an app-store-style install. This is more steps, but they're spelled out and
there's no SSH. Done.

## Task 2: Pair with the home server

**Where I'd click:** `02-just-installed` shows "Looking for your home server on your tailnet…". Then
`03-found-server` shows "Found your home server: homebox / homebox.tail1234.ts.net · 100.101.1.1", and I click
**"Ask to connect"**. `04-waiting-approval` then shows "Waiting for approval on homebox… On homebox, tap
**Approve** next to **slate7**", plus bullets for Flint 3 and for a computer at home.

**Friction:**
- None really. This is easier than Speedify's account login. The error states are clear too:
  `03b` "Turn on Tailscale first" with "Open Tailscale settings", and `03c` "Checked 4 online machines…".
- The footer link "Setting up the router that stays at home? **Make this the home server**" sits on the
  travel router's setup page. I wouldn't click it, but it's a blue link right under the main flow. (low)

## Task 3: Hotel Wi-Fi (Repeater) + phone tethering, confirm both are used

**Where I'd look:** `05-connected-two-wans`. The header reads "Connected to homebox / Using Repeater
(Wi-Fi) + Tethering (phone)". Both cards show a green "In use" pill with live numbers:
Repeater "↓ 38 Mbps ↑ 4.2 Mbps · 24 ms", Tethering "↓ 21 Mbps ↑ 2.9 Mbps · 48 ms". That's my
"Using 2 connections". The mode box "Speed or reliability" offers **Fastest** / **Smooth calls and
video** / **Most reliable**, which I read as Speed / Streaming / Redundant.

**Friction:**
- There are no live per-connection graphs on the travel router, only a snapshot number per card.
  In Speedify I watch the two graph lines to see that bonding is really happening. The Fedora server
  page has graphs (`14`, "Path 0" / "Path 1"), but the router I actually sit behind doesn't. (low)
- The header "↓ 59 Mbps ↑ 7.1 Mbps" is the sum of the two cards (38+21). I can't tell whether that's
  current traffic or measured capacity. If I'm not downloading anything, does it read 0? (low)
- The mode cards have no Save button, so I assume a click applies right away. Nothing confirms it. (low)
- When the phone link breaks (`06`), the card turns red: "Not working right now / It's connected, but
  can't reach homebox. tailbond keeps trying; the other connections carry your traffic." That's
  better than Speedify's terse status. Good.

## Task 4: Phone as backup only

**Where I'd click:** On the Tethering card, "Use it" → **"Backup only"** (`07-save-data-standby`). The pill
changes to "Backup · saving data" and the card says "Used only if your other connections stop working.
Until then it isn't used at all, so it uses no data." The header says "Tethering (phone) on standby".
The hint under the modes, "Saving your phone's data? Set Tethering or Cellular to 'Backup only'", would
have sent me here even if I'd looked in the modes box first.

Then "Change limit" (`07b`) opens "5 GB a month" with Save / Cancel: "Counts everything tailbond sends and
receives over it. At the limit it pauses; the count restarts on the 1st of each month. 0 = no limit."

**Friction:**
- There's no "Secondary" tier. Speedify has Always / Secondary / Backup / Never. Here it's Always /
  Backup only / Off. It's fine for this task, but I lose "use the phone only when the hotel Wi-Fi is
  slow, not just dead". (low)
- "At the limit **it** pauses": I assume "it" means this connection and not all of tailbond, because
  the text is inside the Tethering card. I'd still want it spelled out ("Tethering stops being used").
  (low)
- The reset is fixed to "the 1st of each month". My carrier's billing cycle starts on the 14th, and
  Speedify lets me pick the day. (low)
- "Counts everything tailbond sends and receives over it" means phone data used while tailbond is off
  isn't counted. So "1.2 GB of 5 GB" won't match my carrier's meter. Speedify has the same caveat. (low)
- It's unclear how "Most reliable – Sends everything over every connection" interacts with a
  "Backup only" phone. Does Most reliable override backup and burn phone data? I'd guess not, but
  nothing says so. (low)

## Task 5: Reach the NAS by MagicDNS name

**Where I'd look:** "Tailnet machines" (`05`): "nas / nas.tail1234.ts.net · 100.101.2.2 · linux · Online" with
a **"Copy name"** button. Above it: "Laptops and phones on this router's Wi-Fi or LAN reach these by their
full name, e.g. nas.tail1234.ts.net, or by their 100.x address." The Home server card repeats: "Tailnet
names: Work on every device connected to this router while tailbond is on. Use the full name…"

**Friction:**
- The bold heading is the short name "nas", and a Tailscale user would type just `nas`. The text says
  twice to use the full name, so I'd get it right, but my first try would probably be the short name.
  (low)
- Phone layout (`05-phone`, `07b-phone`, `11-phone`): the name wraps mid-word, "nas.tail1234.ts.ne" /
  "t · 100.101.2.2". If I typed it by hand off my phone I might drop the "t". "Copy name" saves me. (low)

## Task 6: Turn tailbond off

**Where I'd click:** the blue toggle top-right of the status card. `09-turned-off`: "tailbond is off /
Devices on this router use the internet directly, without your home server. Turn on to bond your
connections again." The cards switch to "Ready / Used when tailbond is on." Clear, and it's like Speedify's
big on/off switch.

**Friction:**
- The red "Unpair" button sits right next to "Ask to connect again". Someone could hit it while looking
  for "off", but `09b` has a proper confirmation ("Unpair from homebox? … it will need to approve"), so
  it's recoverable. (low)
- Note from `08-server-unreachable`: "Meanwhile, devices on this router have no internet … To use this
  internet directly for now, turn tailbond off." That's honest and the "Turn tailbond off" button is
  right there. Good.

## Task 7: Approve the travel router at home

**Flint 3 (`11-server-pending-request`):** "Waiting for your approval · 1 new / slate7 wants to connect /
Travel router on your tailnet · signed in as alex@example.com / GL-BE3600 …" with **Approve**. The name
"slate7" matches what the Slate 7 told me in `04`. After approval (`12`), Devices shows "slate7 / Connected
just now · 2 connections bonded · downloading 48 Mbps · uploading 6.1 Mbps".

**Fedora (`14-fedora-dashboard-request`):** the same orange "Waiting for your approval" banner with **Approve**,
right at the top. Easy.

**Friction:**
- The direction of the numbers flips between views. The Flint header says "↓ 7.4 Mbps ↑ 61 Mbps" while
  the slate7 row says "downloading 48 Mbps". The Fedora header says "1.1 MBPS DOWN / 12.0 MBPS UP", but
  its Devices tab (`15`) shows pixel-8 at "↓ 12.0 Mbps ↑ 1.1 Mbps". One view counts from the server and
  the other from the device, and neither says so. When I'm checking whether my bonding works, I'd
  briefly think upload was huge. (low)
- On the Fedora Connections tab the paths are "Path 0" / "Path 1" with "Pixel-8 · Min 31 Ms". I can't
  tell which path is Wi-Fi and which is cellular. Speedify names each connection, and the router's own
  page does (Repeater / Tethering). "Ms" with a capital M also reads as megaseconds. (low)
- Different addresses for the same server: the Slate 7 says "Connects to home.example.net over the
  internet (UDP port 443)", the Fedora header says "reachable at homebox.tail1234.ts.net:443", and the
  Flint shows both. It's not a problem for the task, but it isn't obvious which one the travel router
  actually uses. (low)

---

## Findings

| # | task | finding | severity |
|---|---|---|---|
| 1 | 1 | Package file name `aarch64_cortex-a53_neon-vfpv4.ipk` is only explained on the home server's page; nothing on a releases page helps you pick it | low |
| 2 | 1 | LuCI's "Allow overwriting conflicting package files" checkbox gets no guidance (tick or not?) | low |
| 3 | 1 | No hint for getting back from LuCI to the GL panel / APPLICATIONS → tailbond after installing | low |
| 4 | 2 | "Make this the home server" link sits under the travel router's setup flow and is easy to mis-click | low |
| 5 | 3 | No live per-connection graphs on the travel router (only snapshot numbers); Speedify's main "is bonding working" signal is missing | low |
| 6 | 3 | Header "↓ 59 Mbps ↑ 7.1 Mbps": unclear whether it's current traffic or capacity | low |
| 7 | 3 | Mode cards (Fastest / Smooth calls and video / Most reliable) have no save or applied confirmation | low |
| 8 | 4 | No "Secondary" priority (only Always / Backup only / Off), so you can't say "use the phone when the hotel Wi-Fi is slow" | low |
| 9 | 4 | "At the limit it pauses": "it" is ambiguous (this connection or all of tailbond?) | low |
| 10 | 4 | Data cap resets on "the 1st of each month" only; carrier billing day can't be set | low |
| 11 | 4 | Interaction between "Most reliable" (every connection) and a "Backup only" phone isn't explained | low |
| 12 | 5 | Bold short name "nas" invites typing `nas`; only the full name works (explained in text) | low |
| 13 | 5 | Phone layout wraps the MagicDNS name mid-word ("nas.tail1234.ts.ne" / "t · 100.101.2.2") | low |
| 14 | 6 | Red "Unpair" sits beside "Ask to connect again"; the confirm dialog makes it recoverable | low |
| 15 | 7 | Up/down direction flips between server header and device rows (Flint ↑61 vs slate7 downloading 48; Fedora 1.1 down/12 up vs pixel-8 ↓12/↑1.1) | low |
| 16 | 7 | Fedora Connections tab names paths "Path 0/1" instead of the link type (Wi-Fi / cellular); "Min 31 Ms" capitalisation | low |
| 17 | 7 | Server address shown differently on each side (home.example.net vs homebox.tail1234.ts.net:443) | low |
