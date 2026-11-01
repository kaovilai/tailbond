# Persona D (long-time Speedify user): GL.iNet round 3

Background: I've used Speedify for years. I think in terms of Speed / Redundant / Streaming modes, per-connection priority (Always / Secondary / Backup / Never), monthly caps per connection, live per-connection graphs and the "Using 2 connections" status line. I'm comparing tailbond on a Slate 7 against that.

Overall: it maps closely onto how I already think. "Fastest / Smooth calls and video / Most reliable" read as Speed / Streaming / Redundant, and "Always / Backup only / Off" read as Always / Backup / Never. "Using Repeater (Wi-Fi) + Tethering (phone)" works as my "Using 2 connections" line. The main thing I miss is live graphs on the router. The only real snag is getting the package file if my home server is a Flint 3.

---

## Task 1: Install on the Slate 7 without SSH

**Clicks:**
- `01a-install-gl-advanced-settings`: SYSTEM → Advanced Settings → **Go To LuCI**. The line "Use the same admin password to log in." is reassuring.
- `01b-install-luci-upload`: System → Software → **Upload Package…**, pick `tailbond_0.1.0-r42_aarch64_cortex-a53_neon-vfpv4.ipk`, then **Install** in "Manually install package / Really install the uploaded package?". I'd leave "Allow overwriting conflicting package files" unticked.
- `02-just-installed`: tailbond now appears under APPLICATIONS.

**Where I'd stumble:**
- **Fedora home (screen 15):** no problem. "Add a GL.iNet travel router" has a **Download the router package** button and three steps that match screens 01a/01b exactly.
- **Flint 3 home (screens 11–13):** the Flint panel has no download button. It only says: "Get the router package on your computer: the tailbond file ending in _neon-vfpv4.ipk (from the tailbond releases page, or from a Fedora home server's tailbond page)." "the tailbond releases page" is not a link, and I have no Fedora box. I'd have to search the web for it and then pick the right file from a list of architectures. The hint "ending in _neon-vfpv4.ipk" helps me choose, but Speedify's "download for your platform" was one click. **Medium.**
- No screenshot shows LuCI's result after Install, or whether I need to reload the GL panel before the APPLICATIONS → tailbond entry appears. I'd guess a page refresh. **Low.**

## Task 2: Pair with the home server

**Clicks:** `02-just-installed` shows "Looking for your home server on your tailnet…". Then `03-found-server` shows "Found your home server: homebox / homebox.tail1234.ts.net · 100.101.1.1" with **Ask to connect**. Then `04-waiting-approval` shows "Waiting for approval on homebox… On homebox, tap **Approve** next to **slate7**."

**Where I'd stumble:** nowhere. This is easier than Speedify's account login. The other states also look good:
- `03b` "Turn on Tailscale first" gives me a button.
- `03c` "No home server found on your tailnet. Checked 4 online machines…" tells me what to check.

**Low only:** in `04` the hint "From anywhere: open http://homebox:8077 …" points me at the Fedora page, while I may have a Flint. It does go on to say "At home with a Flint 3, its admin panel → APPLICATIONS → tailbond works as well", so it's fine.

## Task 3: Hotel Wi-Fi + USB tethering, confirm both used

**Clicks:** none. I'd look at `05-connected-two-wans`:
- Header: "Connected to homebox / Using Repeater (Wi-Fi) + Tethering (phone)", "↓ 59 Mbps ↑ 7.1 Mbps Up 2 h 13 min".
- Both cards show green "In use" badges with live numbers: Repeater "↓ 38 Mbps ↑ 4.2 Mbps · 24 ms", Tethering "↓ 21 Mbps ↑ 2.9 Mbps · 48 ms". 38 + 21 = 59 adds up, so I believe it's actually bonding.
- On the Flint side (`12`), slate7 shows "2 connections bonded". That is the Speedify "Using 2 connections" moment.

**Where it falls short of Speedify:**
- The router page has no live per-connection graph, only numbers. In Speedify I watch the two lines move to see the phone pick up the slack. **Low.**
- The Repeater card doesn't show which hotel network (SSID) it's on. Speedify shows the Wi-Fi name. **Low.**
- `06-one-wan-failed` handles failure well: "Tethering (phone) not working right now" and "It's connected, but can't reach homebox. tailbond keeps trying; the other connections carry your traffic." That's clearer than Speedify's grey-out.
- On the Fedora page (`14`), the two path cards are labelled "Path 0 · Pixel-8 · Min 31 Ms" and "Path 1", not Wi-Fi / Cellular. I can't tell which connection is which. Also "Min 31 Ms" is odd capitalisation. **Low.**

## Task 4: Phone = backup only

**Clicks:** in `07-save-data-standby`, on the Tethering card, **Use it → Backup only**. The badge changes to "Backup · saving data", with the text "Used only if your other connections stop working. Until then it isn't used at all, so it uses no data." The header changes to "Tethering (phone) on standby". The hint under the modes, "Saving your phone's data? Set Tethering or Cellular to "Backup only" under Internet connections.", points the right way.

For a cap (`07b-set-limit`): **Change limit**, then "5 GB a month", then **Save**. The caption reads: "Counts everything tailbond sends and receives over it. At the limit, tailbond stops using this connection until the count restarts on the 1st of each month. 0 = no limit."

**Where I'd stumble:**
- There's no "Secondary" level (use only when the primary is slow). Speedify has four levels and this has three. For this task, "Backup only" is exactly what I want. **Low.**
- The cap always resets "on the 1st of each month". My phone plan resets mid-month, and Speedify lets me pick the day. **Low.**
- The cap counts only what tailbond sends ("Counts everything tailbond sends and receives over it"). When tailbond is off (`09`), I can't tell whether the router still uses the phone without any cap. **Low.**
- It's unclear whether picking **Most reliable** ("Sends everything over every connection. Uses the most data.") would override Backup only and burn phone data. In Speedify, Redundant still respects Backup priority. I'd assume the same here, but nothing says so. **Low.**
- Small wording difference: the button is "Set limit" on Ethernet/Repeater and "Change limit" on Tethering once a cap exists. It makes sense. **Low / no issue.**

## Task 5: Reach the NAS by MagicDNS name

**Clicks:** in `05`, the **Tailnet machines** section lists "nas / nas.tail1234.ts.net / 100.101.2.2 / linux · Online", with **Copy name**. The help text says "Laptops and phones on this router's Wi-Fi or LAN reach these by their full name, e.g. nas.tail1234.ts.net". The Home server row adds "Use the full name, like nas.tail1234.ts.net." The name is `nas.tail1234.ts.net`.

**Where I'd stumble:** nowhere that matters. Speedify has nothing like this, so it's a bonus. **Low:** the short name "nas" is the boldest text, but the page twice tells me to use the full name. In `08`/`09` the hint reads "like nas.your-tailnet.ts.net", which is a placeholder rather than my real tailnet, and the list says "The list appears once tailbond is connected." That's fine.

## Task 6: Turn tailbond off

**Clicks:** the toggle at the top right of the status card (`05`). That gives `09-turned-off`: "tailbond is off / Devices on this router use the internet directly, without your home server. Turn on to bond your connections again." Same as Speedify's main switch. `08-server-unreachable` also offers a **Turn tailbond off** button, plus a clear warning: "Meanwhile, devices on this router have no internet…"

**Where I'd stumble:**
- The toggle has no "On/Off" label. The headline says it, so this is **Low.**
- In `09`, the Home server box still shows **Ask to connect again** and **Unpair**. I might wonder whether I need "Ask to connect again" to turn it back on, when the toggle is the right control. **Low.** Unpair is clearly separate, and its confirmation in `09b` explains it ("forgets the home server… it will need to approve"), so I wouldn't click it by mistake.
- In `09`, Tethering shows "Always" again, though I had set Backup only. Maybe it's just the test fixture, but as a user I'd wonder whether my setting survived turning tailbond off and on. **Low.**

## Task 7: Approve on the home side

**Flint 3:**
- `11-server-pending-request`: "Waiting for your approval · 1 new", then "slate7 wants to connect / Travel router on your tailnet · signed in as alex@example.com / GL-BE3600 · linux · 100.101.9.9 · asked just now". Click **Approve**.
- `12-server-approved`: slate7 appears under Devices with "Connected just now · 2 connections bonded · downloading 48 Mbps · uploading 6.1 Mbps".

**Fedora:**
- `14`: the same orange "Waiting for your approval" card at the top. Click **Approve**.

**Where I'd stumble:** not in approving. **Low:** the Fedora page (`14`/`15`) contradicts itself on direction. The big header says "1.1 MBPS DOWN / 12.0 MBPS UP", while the pixel-8 row says "↓ 12.0 Mbps ↑ 1.1 Mbps", and Data shows "858 MB ↓ · 7.5 GB ↑". One is the server's view and the other is the device's, but nothing says so. The Flint wording is clearer: "Sending to devices 61 Mbps / Receiving 7.4 Mbps". The Fedora device row also shows the jargon "wlb".

---

## Findings

| # | task | finding | severity |
|---|---|---|---|
| 1 | 1 | Flint 3 home: no download button. "from the tailbond releases page" isn't a link, so a Flint-only user must hunt for the .ipk and pick an architecture | medium |
| 2 | 1 | No screen shows LuCI's install result or whether to refresh the GL panel before APPLICATIONS → tailbond appears | low |
| 3 | 2 | `04` hint leads with "http://homebox:8077" (Fedora); the Flint path is second | low |
| 4 | 3 | No live per-connection graph on the router page (Speedify's main "is it bonding?" visual); numbers only | low |
| 5 | 3 | Repeater card doesn't show the hotel SSID | low |
| 6 | 3 | Fedora path cards say "Path 0 / Path 1 · Min 31 Ms", not which connection (Wi-Fi/cellular) | low |
| 7 | 4 | Only Always / Backup only / Off; no "Secondary"-style level | low |
| 8 | 4 | Monthly cap always resets "on the 1st of each month"; can't match the phone plan's billing day | low |
| 9 | 4 | Cap "Counts everything tailbond sends and receives". Unclear whether the phone is used uncapped while tailbond is off | low |
| 10 | 4 | Unclear whether "Most reliable" ("Uses the most data") overrides a Backup only connection | low |
| 11 | 5 | `08`/`09` hint uses placeholder "nas.your-tailnet.ts.net" instead of the real tailnet | low |
| 12 | 6 | Main toggle has no On/Off label; "Ask to connect again" next to it when off may look like the way to turn back on | low |
| 13 | 6 | `09` shows Tethering back on "Always" after I'd set Backup only, so I'm unsure the setting persists (may be a fixture artifact) | low |
| 14 | 7 | Fedora page: header "1.1 MBPS DOWN / 12.0 MBPS UP" vs device row "↓ 12.0 ↑ 1.1": direction flips with no label; "wlb" jargon | low |
