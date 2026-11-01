# Round 7 — GL persona D (Speedify user)

**Who I am:** I've run Speedify for years, including on a GL.iNet router through GL's VPN menu. I know Speedify's bonding modes (Speed / Redundant / Streaming), the per-connection Always / Secondary (backup) / Never priority, monthly or daily data limits, and the live per-connection graphs showing latency and loss. I also know how Speedify behaves when a link drops. I'm judging tailbond against all of that. Everything below comes from the GL packet screenshots and the manifest. I didn't open any other file.

**Overall:** This is the closest thing to "Speedify's GL integration, but to my house" that I've seen. The per-connection **Always / Backup only / Off** control maps one-to-one onto Speedify's connection priorities. The three modes **Fastest / Smooth calls and video / Most reliable** map onto Speed / Streaming / Redundant. Link-failure and sign-in-page handling are explained better than in Speedify. I found nothing that stops a task. What I do miss is the observability Speedify gives me on the router: live per-link graphs, loss figures, and the health of the standby link. The biggest real difference is that my "server" is my own home line. That brings port and CGNAT problems Speedify never has, and one screen is misleading about how bad CGNAT is.

---

## Task 1: Install tailbond on the Slate 7 without SSH

**Where I click:**
- On the Fedora page (`15-fedora-devices-tab`), the card **"Add a GL.iNet travel router (recommended)"** → **"Download the router package"**.
- On the Slate 7 (`01a`), **SYSTEM → Advanced Settings → "Go To LuCI"**. The hint says "Use the same admin password to log in."
- In LuCI (`01b`), **System → Software → "Upload Package…"**. The dialog reads "Really install the uploaded package?" and shows `tailbond_0.1.0-r42_aarch64_cortex-a53_neon-vfpv4.ipk (6.1 MB)`. I click **Install**.

**What works:** The three numbered steps on the Fedora and Flint pages match the screens exactly. The Flint version (`11`) also says "(leave the other options as they are)", which tells me to ignore the "Allow overwriting conflicting package files" checkbox. It also reassures me: "The router's internet keeps working."

**Friction:**
- Speedify on GL is a menu item in GL's own UI. Here I have to drop into LuCI. It's a bit more hands-on, but the steps are spelled out, so I can finish.
- There is one download button and no model or architecture choice. The file name ends in `aarch64_cortex-a53_neon-vfpv4`. The Flint page says "It's the same tailbond this router runs", but on the Fedora page (an x86 box) nothing says which GL models this one package fits. I'd hope it suits a Slate 7, but nothing on screen confirms that. (low)

## Task 2: Pair the Slate 7 with the home server

**Where I click:** VPN → tailbond (`02`) shows "Looking for your home server on your tailnet…". A few seconds later (`03`) it shows "Found your home server: homebox — homebox.tail1234.ts.net · 100.101.1.1" and I click **"Ask to connect"**. On `04` it says "Waiting for approval on homebox… On homebox, tap **Approve** next to **slate7**. This router connects by itself once approved; you can close this page."

**What works:** This is easier than Speedify. I don't need an account login or a server pick. The error states are concrete. `03b` "Turn on Tailscale first" has an **"Open Tailscale settings"** button. `03c` "No home server found on your tailnet — Checked 4 online machines…" gives a manual name field. No finding.

## Task 3: At the hotel, use hotel Wi-Fi (Repeater) and phone tethering, and confirm both are used

**Where I look:** `05-connected-two-wans`. The top card says "Connected to homebox — Using Repeater (Wi-Fi) + Tethering (phone)", with "↓ 59 Mbps ↑ 7.1 Mbps". Both the **Repeater** and **Tethering** cards show a green **"In use"** badge. The cards show per-link rates and latency: "↓ 38 Mbps ↑ 4.2 Mbps · 24 ms" and "↓ 21 Mbps ↑ 2.9 Mbps · 48 ms". The per-link numbers add up to the total, which I like.

**When a link drops** (`06`): the top card says "Tethering (phone) not working right now". The Tethering card has a red "Not working right now" badge and the line "It's connected, but can't reach homebox. tailbond keeps trying; the other connections carry your traffic." That's clearer than Speedify's own wording.

**Compared with Speedify:**
- The travel router panel has **no live per-connection graph, no packet-loss figure, and no history**. It shows a single snapshot of numbers per card. I still finish the task, because "In use" on both cards is unambiguous. But I can't see a link getting flaky before it fails. In `06` I also can't tell how long the phone has been down. (low)
- Oddly, the Fedora home server page (`14`) *does* have Speedify-style per-path graphs with LATENCY and LOSS. Those paths are labeled "Path 0 · Pixel-8 · Min 31 Ms" and "Path 1". They're numbered, not named "Wi-Fi" or "Cellular", so I can't match them to my links. "Min 31 Ms" is also odd capitalization. (low)

## Task 4: Limit the phone to backup only

**Where I click:** On the Tethering card, **Use it → "Backup only"** (`07`). The badge becomes "Backup · saving data" with the text "Used only if your other connections stop working. Until then it isn't used at all, so it uses no data." The top card says "Tethering (phone) on standby". The tip under the modes even points here: "Saving your phone's data? Set Tethering or Cellular to "Backup only" under Internet connections."

That's exactly Speedify's "Secondary/Backup" priority. Done.

**Data limit** (`07b`, "Change limit"): a "5 GB a month" field and the text "Counts everything tailbond sends and receives over it. At the limit, tailbond stops using this connection until the count restarts on the 1st of each month. 0 = no limit."
- Speedify lets me set a daily limit or a monthly limit **with my own reset day**. Here the count always restarts on **the 1st**, and my phone plan's cycle doesn't start on the 1st. The counter won't match my carrier's meter, so I'd have to set the limit lower to stay safe. (low)
- On standby, the Tethering card shows only "192.168.42.129". It shows no latency and no "last checked OK". I can't tell if my backup link is actually healthy until I need it. Speedify keeps showing whether a backup link is usable. (low)
- None of the mode descriptions say whether **"Most reliable"** ("Sends everything over every connection") overrides "Backup only". In Speedify, Redundant mode still respects backup links. I'd guess the same here, but the screen doesn't say. (low)

## Task 5: From the laptop, reach the NAS by its MagicDNS name

**Where I look:** On `05`, the **Tailnet machines** card shows "nas — nas.tail1234.ts.net — 100.101.2.2 — linux · Online · direct" with a **"Copy name"** button. The Home server card adds "Tailnet names: Work on every device connected to this router while tailbond is on. Use the full name, like nas.tail1234.ts.net."

That's clear. I'd copy `nas.tail1234.ts.net`. Only nitpick: "Last seen 1 days ago" on work-laptop. (low)

## Task 6: Turn tailbond off

**Where I click:** The toggle at the top right of the status card (`05`). The result (`09`) reads "tailbond is off — Devices on this router use the internet directly, without your home server. Turn on to bond your connections again." **Unpair** is a separate action with a proper confirmation (`09b`): "This router stops using tailbond and forgets the home server…". Good split. When the server is down (`08`), a **"Turn tailbond off"** button sits right in the error: "Meanwhile, devices on this router have no internet…". That's honest kill-switch behavior.

**Friction (data, from a Speedify-on-GL point of view):** While tailbond is off, the Tethering card still shows **"Backup only"** selected and "1.2 GB of 5 GB this month". But it also says "Used when tailbond is on." So my backup-only setting and my 5 GB cap **don't apply** while tailbond is off. The screen never tells me which connection(s) "the internet directly" now uses. Could GL's own multi-WAN start pulling from my phone with no cap? I'd have to go into GL's network settings to find out. With Speedify on GL I learned to check this. Here, after turning it off, I don't know if my phone data is protected. (medium)

## Task 7: Approve the travel router on the home side

**Flint 3** (`11`): The **"Waiting for your approval · 1 new"** card shows "slate7 wants to connect — Travel router on your tailnet · signed in as alex@example.com · GL-BE3600 · linux · 100.101.9.9 · asked just now". I click **Approve**. In `12`, Devices shows "slate7 Connected just now · 2 connections bonded · downloading 48 Mbps · uploading 6.1 Mbps".

**Fedora** (`14`): The orange card at the top says "slate7 wants to connect" and I click **Approve**.

Both are easy. The model number and Tailscale identity give me enough to trust the request.

- On the Flint, the header says "Sending to devices **61 Mbps** · Receiving **7.4 Mbps**" in `11`, `18` and `18b`, where only pixel-8 is connected at "downloading 12 Mbps · uploading 1.1 Mbps". The header doesn't add up to the device list. I compare those numbers the way I'd read Speedify's stats, so this undermines trust. (low)
- On the Fedora page (`14`), the big numbers read "1.1 MBPS DOWN / 12.0 MBPS UP", but the only device is downloading at 12 Mbps. The numbers are from the server's point of view. That's not wrong, but it's flipped compared with how every other screen (and Speedify) describes my traffic. (low)
- The Flint panel's subtitle still reads "Bond this router's internet connections into one tunnel to your home server." But this router *is* the home server. (low)

## Task 8: Hotel Wi-Fi with a sign-in page (GL's repeater detected it)

**Where I click:**
- `16`: the top card says "Using Tethering (phone) · Repeater (Wi-Fi) needs sign-in". A blue box explains: "Repeater needs you to sign in — GL's Repeater page handles this sign-in. Open it, click "Login Mode for Public Hotspots" (or "Go to Login") and accept the hotel's terms." I click **"Open GL's Repeater page"**.
- `16b`, after starting Login Mode: "Router connected to homebox · devices signing in". The text: "GL's Login Mode is on: all traffic from devices here goes out the hotel Wi-Fi directly (not just the sign-in page), not through your home server… It ends by itself once you've signed in (or click "Quit Login Mode" on GL's Repeater page); tailbond then adds Repeater back to bonding."

**What works:** It's honest about the privacy cost, which Speedify is not, and it says recovery is automatic. I can finish.

**Friction:** During Login Mode the top card still shows "↓ 21 Mbps ↑ 2.9 Mbps", and Tethering is "In use" with "Always". The box says "The router itself stays connected to your home server over your other connections… so they keep being used." Used for what, if all device traffic goes out the hotel Wi-Fi? As a data-conscious phone user, I can't tell whether my phone is burning data during sign-in, or whose traffic those 21 Mbps are. (low)

## Task 9: Hotel Ethernet with a sign-in page

**Where I click:**
- `17`: "Ethernet needs you to sign in". Small print: "GL's Login Mode covers the repeater Wi-Fi only; tailbond handles sign-in pages on other connections like this one." Then: "Opening the sign-in page lets devices here reach Ethernet directly (without your home server) for up to 10 minutes, so the page can load." I click **"Open sign-in page"**.
- `17b`: "Signing in to Ethernet… For up to 9 more minutes (until 04:40 on this router's clock)…" with **"I've signed in, check now"** and **"Stop sign-in mode"**. Meanwhile the Repeater stays "In use" and Tethering stays "Backup · saving data".

This is clear and better than Speedify, which just sees a dead link. My only quibbles:
- There are two different sign-in mechanisms, GL's Login Mode for Wi-Fi and tailbond's own for everything else, and the distinction lives in one line of small print. (low)
- "until 04:40 on this router's clock" is meaningless to me if the travel router's clock or time zone is off. A countdown is enough. (low)

## Task 10: At home, set everything up for best speed

**Flint 3:**
- `13`: two setup steps, both clear. Step 1 is the Tailscale route (**"Open admin console"**, tick 10.77.0.0/24). Step 2 is Dynamic DNS via "APPLICATIONS → Dynamic DNS".
- `18`: Setup health "2 to improve" gives the exact fix: "On this router: NETWORK → UPnP, turn it on. If another router sits in front of this one … forward UDP port 41641 to this router there."
- `18c`: "All good — Everything is set up for best speed", and each check is listed. Good.

**Fedora** (`15`): "Setup health: 2 to improve for best speed", with the same UDP 41641 / UPnP advice and "How to fix" links.

**Friction:**
- **CGNAT (`18b`) contradicts itself.** The red card says "Step 1: Your internet line uses carrier-grade NAT … travel routers can't reach UDP 443 here … from away they can't until this is fixed." So bonding from a hotel **doesn't work at all**. Yet the Setup health card files the same item under "1 thing to improve for best speed **(tailbond works meanwhile, just slower)**". If I only read the health card, the same kind of card that says "All good" in `18c`, I'd leave on my trip thinking it works, just slower. This is also the sharpest contrast with Speedify, whose cloud servers never care about my home line. The screen needs to say plainly that this blocks remote use. (medium)
- To a Speedify user, it's strange that there are **two different ports** to care about: "UDP 443" for tailbond (`11`: "that box must forward UDP 443 to this router") and "UDP 41641" for Tailscale (`18`). Each is explained where it appears, but nothing ties them together. "Same fix as above" under the hard-NAT item does help. (low)

---

## Findings

| # | Task | Finding | Severity |
|---|---|---|---|
| 1 | 10 | CGNAT screen (`18b`) contradicts itself. The red Step 1 says "travel routers can't reach UDP 443 here… from away they can't until this is fixed", but Setup health lists it as "1 thing to improve for best speed (tailbond works meanwhile, just slower)". A reader of the health card would think remote bonding works. | medium |
| 2 | 6 | When tailbond is off (`09`), "Backup only" and the 5 GB phone limit stop applying ("Used when tailbond is on."). "Devices on this router use the internet directly" doesn't say which connection, so I can't tell if GL will now use my phone data uncapped. | medium |
| 3 | 1 | One "Download the router package" button (`…aarch64_cortex-a53_neon-vfpv4.ipk`), no model or architecture choice. The Fedora page never says which GL models it fits. | low |
| 4 | 3 | Travel router panel has no live per-connection graph, loss %, or history. There's only a snapshot "↓ 38 Mbps ↑ 4.2 Mbps · 24 ms" per card, and no "down since" when a link fails (`06`). | low |
| 5 | 3 | Fedora page (`14`) does have per-path graphs with latency and loss, but they're labeled "Path 0 / Path 1 · Pixel-8 · Min 31 Ms", not by link type (Wi-Fi or Cellular). | low |
| 6 | 4 | Data limit always resets "on the 1st of each month". There's no reset day and no daily limit, so it won't match my phone plan's cycle (`07b`). | low |
| 7 | 4 | A standby backup link shows only its IP ("192.168.42.129"), with no latency or health. I can't tell if the backup will actually work. | low |
| 8 | 4 | Nothing says whether "Most reliable – Sends everything over every connection" also uses a "Backup only" link. | low |
| 9 | 5 | "Last seen 1 days ago" grammar. | low |
| 10 | 7 | Flint header "Sending to devices 61 Mbps · Receiving 7.4 Mbps" doesn't match a device list of only pixel-8 at "downloading 12 Mbps · uploading 1.1 Mbps" (`11`, `18`, `18b`). | low |
| 11 | 7 | Fedora page shows "1.1 MBPS DOWN / 12.0 MBPS UP" while the device is downloading 12 Mbps. The direction is from the server's view, flipped from every other screen. | low |
| 12 | 7 | Flint (home server) panel subtitle still says "Bond this router's internet connections into one tunnel to your home server." | low |
| 13 | 8 | During GL Login Mode (`16b`), the top card still shows "↓ 21 Mbps ↑ 2.9 Mbps" and Tethering "In use", and the text says "so they keep being used". It doesn't say what traffic is using my phone data while devices go out the hotel Wi-Fi directly. | low |
| 14 | 9 | Two sign-in mechanisms (GL Login Mode for Wi-Fi, tailbond's own for Ethernet), distinguished only in small print: "GL's Login Mode covers the repeater Wi-Fi only…" | low |
| 15 | 9 | "until 04:40 on this router's clock" depends on the travel router's clock and time zone. A countdown alone would be clearer. | low |
| 16 | 10 | Two unrelated ports to forward (UDP 443 for tailbond, UDP 41641 for Tailscale), each explained separately. Speedify users never forward anything. | low |

Totals: 0 high / 2 medium / 14 low
