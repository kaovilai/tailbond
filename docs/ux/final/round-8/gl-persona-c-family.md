# Round 8 — GL persona C (family member)

**Who I am:** a family member who didn't set up the home server. I was handed a GL.iNet Slate 7 and asked to get it working. I know what Wi-Fi and a phone hotspot are. I don't know what routes, ports or NAT are, and I've never heard the words "tailnet" or "MagicDNS".

**What I had:** only `task-script.md`, `screens.md` and the PNG screenshots in `out/ux/final/round-8/packet-gl/`. I looked at the desktop versions of every screen and at phone versions where it mattered (01b, 07). Quotes are copied from the screens.

---

## Task 1: Install tailbond on the Slate 7 without SSH

**Where I'd click**
- On the home server's page, I click **"Download the router package"**:
  - Fedora: `15-fedora-devices-tab`, card "Add a GL.iNet travel router (recommended)".
  - Flint 3: `11`/`12`/`13`, card "Add a GL.iNet travel router".
- On the Slate 7 (`01a-install-gl-advanced-settings`) I go to SYSTEM → Advanced Settings and click **"Go To LuCI"**. The page reassures me: "LuCI is OpenWrt's own web interface. Use the same admin password to log in." I don't know what OpenWrt is, but "same admin password" is all I need.
- In LuCI (`01b-install-luci-upload`) I go to Software → **"Upload Package…"**, pick the file, and click **"Install"** in "Manually install package — Really install the uploaded package?". The Flint's instructions say "leave the other options as they are", so I leave "Allow overwriting conflicting package files" unticked. The phone version of the dialog is readable and the buttons are big enough.

**Where I hesitated**
- **File name.** The file is called `tailbond_0.1.0-r42_aarch64_cortex-a53_neon-vfpv4.ipk`. That means nothing to me, and nothing says "this is the right file for your Slate 7". The Flint panel says "It's the same tailbond this router runs", but the Flint is a different router from mine, so that doesn't reassure me. (low)
- **Fedora page is harder to follow.**
  - The download is on the **Devices** tab. The page opens on **Connections** (`14`), so I'd have to know to switch tabs.
  - The Fedora steps are shorter than the Flint's. Step 2 ends "pick the file, Install." It leaves out "(log in with the same password)" and "leave the other options as they are", which the Flint version has. (low)

After installing, `02-just-installed` shows the new VPN → tailbond page, so I know it worked.

## Task 2: Pair the Slate 7 with the home server

**Where I'd click**
- `02`: a spinner says "Looking for your home server on your tailnet…".
- `03`: "Found your home server: homebox". I click **"Ask to connect"**.
- `04`: "Waiting for approval on homebox… On homebox, tap Approve next to slate7. This router connects by itself once approved; you can close this page." Very clear: I know what to do next and that I can walk away.
- The problem screens are clear too:
  - `03b` (Tailscale off) gives me "Turn on Tailscale first" with an "Open Tailscale settings" button.
  - `03c` (no server found) says "No home server found on your tailnet — Checked 4 online machines…" and offers "Search again".

**Where I hesitated**
- **"tailnet".** "on your tailnet" is a word I don't know. I worked out it means "your Tailscale stuff". (low)
- **"Make this the home server" link.** Every travel-router screen ends with "Setting up the router that stays at home? Make this the home server (you can switch back here)". I'm confused enough about which box is "home" that I might click it by mistake. It does say I can switch back. (low)

## Task 3: Hotel Wi-Fi (Repeater) and phone over USB, confirm both are used

**Where I'd look**
- `05-connected-two-wans`: the top card says "Connected to homebox — Using Repeater (Wi-Fi) + Tethering (phone)". Both the Repeater and Tethering cards show a green **"In use"** with their own speeds. That's exactly the confirmation I wanted.
- `06-one-wan-failed` is also clear:
  - Top card: "Tethering (phone) not working right now".
  - Tethering card: "It's connected, but can't reach homebox. tailbond keeps trying; the other connections carry your traffic."

**Where I hesitated:** nothing important. "Repeater" for hotel Wi-Fi is GL's own word and I've seen it before.

## Task 4: Phone as backup only

**Where I'd click**
- A hint under "Speed or reliability" says: "Saving your phone's data? Set Tethering or Cellular to "Backup only" under Internet connections."
- On the Tethering card I click **"Backup only"**.
- `07-save-data-standby` confirms it:
  - Badge: "Backup · saving data".
  - Card: "Used only if your other connections stop working. Until then it isn't used at all, so it uses no data."
  - Top card: "Tethering (phone) on standby".

That's exactly what I wanted.

**Where I hesitated**
- **Data limit** (`07b-set-limit`, after clicking "Change limit"): "Counts everything tailbond sends and receives over it… until the count restarts on the 1st of each month."
  - My phone plan doesn't reset on the 1st, and I can't choose the day.
  - I could easily think "5 GB" means my whole phone plan. It only counts tailbond's traffic, not what the phone itself uses. (low)

## Task 5: Find the NAS's name to use from the laptop

**Where I'd look**
- The **"Tailnet machines"** card on `05`/`07` lists:

  ```
  nas
  nas.tail1234.ts.net
  100.101.2.2
  ```

  with a **"Copy name"** button.
- The text above the list says to use "their full name, e.g. nas.tail1234.ts.net". The Home server card says the same thing: "Tailnet names — Work on every device connected to this router while tailbond is on. Use the full name, like nas.tail1234.ts.net."

**Where I hesitated**
- **The word "MagicDNS" never appears.** The task asked for the "MagicDNS name". I found it anyway because the page calls it the "full name".
- **The short name stands out more.** The bold heading is just "nas", which is the first thing my eye lands on, but the text says to use the long one. I'd hope "Copy name" copies the long one; it doesn't say. (low)

## Task 6: Turn tailbond off

**Where I'd click**
- The blue switch at the top right of the status card. `09-turned-off` then says "tailbond is off".

**Where I got confused**
- **My phone data may be used again (medium).** The off screen says: "Devices on this router use the internet directly, without your home server, over whichever connection GL's own settings pick (INTERNET page). Backup only and data limits here apply only while tailbond is on."
  - I set the phone to Backup only to protect my data. Now the page tells me that doesn't count any more, and "whichever connection GL's own settings pick" could be my phone.
  - It doesn't tell me what to do: unplug the phone? Change something on the INTERNET page? I'd be worried I'm now burning phone data.
  - The Tethering card still shows "Backup only" highlighted, which makes it look like the setting still applies.
- **The switch has no label (low).**
  - The switch has no "On/Off" text. The obvious red button is **"Unpair"**. As someone looking for an "off" button, I might click Unpair first.
  - The confirmation `09b-unpair-confirm` would stop me: "This router stops using tailbond and forgets the home server… (it will need to approve)". So it's recoverable.
  - The only clearly labelled "Turn tailbond off" button appears on the error screen `08`.
- **Wrong text on `08`, where tailbond is still on (low).**
  - Fact: on `08-server-unreachable` tailbond is **still on**. The switch is on and the page says "Can't reach homebox".
  - What the cards say: the Repeater and Tethering cards read "This choice applies when tailbond is on. While it's off, GL's own settings (INTERNET page) decide…".
  - Why that's confusing: that is the "off" text, so the page contradicts itself about whether tailbond is on.

## Task 7: Approve the travel router at home

**Flint 3** (`11-server-pending-request`)
- A "Waiting for your approval — 1 new" card says "slate7 wants to connect — Travel router on your tailnet · signed in as alex@example.com · GL-BE3600…". I click **"Approve"**.
- `12-server-approved` shows "slate7 — Connected just now · 2 connections bonded". Easy.

**Fedora** (`14`/`15`): the same card is at the top of the page, with the same **"Approve"** button. Easy to find.

**Where I got confused**
- **The Fedora page contradicts itself about who can approve (medium).** On `15`:
  - The top card says: "Who can approve: anyone on your Tailscale network who can reach homebox (it's a tagged machine, so it has no single owner). Set a password below to limit that."
  - Lower on the **same page**, "Who can approve and add devices" says: "Who can approve: alex@example.com (the Tailscale account homebox belongs to)… others on your Tailscale network, can ask to connect but can't approve."
  - I'm not Alex, so whether I'm allowed to press Approve is exactly what I need to know, and the page gives two different answers.
- **Jargon (low).** "it's a tagged machine" and "they see it as homebox, under its access rules" mean nothing to me.

## Task 8: Hotel Wi-Fi with a sign-in page

**Where I'd click**
- `16-captive-gl-repeater`: a blue box says "Repeater needs you to sign in — GL's Repeater page handles this sign-in. Open it and click "Login Mode for Public Hotspots" (or "Go to Login"). Then open any website on your laptop or phone: the hotel's sign-in page appears there."
- I click **"Open GL's Repeater page"**. That page isn't in my screenshots, so I follow the instructions on trust.
- `16b-captive-gl-login-mode`:
  - Top card: "Router connected to homebox · devices signing in".
  - Box: "Login Mode ends by itself once you've signed in… tailbond then adds Repeater back to bonding."
  - I sign in and wait. Repeater should go back to "In use" as on `05`.
- I'm reassured that "your phone's data isn't used for them" during sign-in. The low speed (12 Kbps) on the top card is explained: "it carries almost nothing meanwhile".

**Where I hesitated**
- **Which button name?** I'm told to click "Login Mode for Public Hotspots" **or** "Go to Login". I don't know which one my router will show, and I can't see GL's page.
- **"Laptop or phone".** It says "open any website on your laptop or phone", but my phone is plugged into the router by USB as the backup connection. Does it count as a "phone on the router"? (low)

## Task 9: Hotel Ethernet cable with a sign-in page

**Where I'd click**
- `17-captive-own`: an orange box says "Ethernet needs you to sign in". I click **"Open sign-in page"**.
- `17b-captive-own-signing-in`:
  - The box tells me what to do: "Sign in on the page that opened (accept the terms, enter the room number…)".
  - It gives clear buttons: **"I've signed in, check now"**, "Open the sign-in page again" and "Stop sign-in mode".

Very good. I'd finish this.

**Where I hesitated**
- **Time limit.** "For up to 9 more minutes (until 04:57 on this router's clock)". I don't know whether the router's clock matches my watch. It doesn't say what happens if I need longer, though I'd guess "Open the sign-in page again" restarts the timer. (low)
- **Reading order on `17`.** The first, small line is "GL's Login Mode covers the repeater Wi-Fi only; tailbond handles sign-in pages on other connections like this one." That's about something I'm not doing. The sentence that explains my problem comes second, in a bigger font. I had to read it twice. (low)

## Task 10: At home, make sure everything is set for best speed

**Where I'd look**
- The **"Setup health"** card on the Flint (`18`, `18b`, `18c`), or the Devices tab on the Fedora page (`15`).
- When it's fine, it's very clear: `18c` shows "All good — Everything is set up for best speed" with green ticks.

**Where I'd get stuck**
- **Port-forwarding instructions I can't follow alone (medium).** `18-server-health-relayed` says:
  - "On this router: NETWORK → Firewall → Open Ports on Router, add UDP 41641." I could probably follow these clicks.
  - "If another router sits in front of this one (for example your internet provider's box), also forward UDP port 41641 to this router there." I don't know whether there's another box, or how to "forward a port".
  - The Fedora version (`15`) is worse: "forward UDP port 41641 to it, and allow 41641/udp in this machine's firewall. (Turning on UPnP…)".
  - To its credit, the Flint says "Not sure how? Show this card to whoever set up your internet". That's what I'd do. But I can't finish this task myself.
- **Carrier-grade NAT (medium).** `18b-server-health-cgnat` says: "Step 1: Your internet line uses carrier-grade NAT… Ask your provider for a public IPv4 address (or use public IPv6)."
  - I'd have to phone the internet company and repeat words I don't understand. I have no idea how I would "use public IPv6".
  - It clearly tells me this blocks travel use: "from away they can't until this is fixed". slate7 also shows "Not connected right now".
  - So I understand how serious it is, but I can't fix it myself.
- **Tailscale admin console (medium).** `13-server-route-pending`, Step 1: "In the Tailscale admin console, open homebox, choose Edit route settings, tick the one route listed (10.77.0.0/24) and save. That lets your Tailscale machines reach devices that connect through tailbond. Devices can already reach your machines without it."
  - I don't have the Tailscale account; someone else set it up.
  - The last two sentences sound like the same thing said both ways, so I can't tell whether I need this step or not.
  - Step 2 (Dynamic DNS) is fine: it names the GL menu and shows what the name looks like.
- **Things I can't do (low).** "To check: tailscale ping apple-tv ("via DERP" means relayed)" is a command I can't run. The Fedora page also says "the guide fedora-server.md has a section per item", a file I don't have.
- **Is the relay worth fixing? (low)** "Through a relay: apple-tv. Only traffic between this machine and that machine is slower (including bonded devices reaching it); bonding itself doesn't go through Tailscale." If only the Apple TV is affected, I can't tell whether this matters to me.

---

## Findings

| # | Task | Finding | Severity |
|---|---|---|---|
| 1 | 6 | `09`: "Backup only and data limits here apply only while tailbond is on" plus "whichever connection GL's own settings pick". Turning off may silently use phone data. There's no advice on what to do, and the Tethering card still shows "Backup only" highlighted. | medium |
| 2 | 7 | Fedora `15` contradicts itself. Top card: "Who can approve: anyone on your Tailscale network… (it's a tagged machine…)". Lower card: "Who can approve: alex@example.com… others… can't approve." | medium |
| 3 | 10 | `18`/`15`: "forward UDP port 41641 to this router there", "allow 41641/udp in this machine's firewall". I can't do this alone; I can only hand it off ("Show this card to whoever set up your internet"). | medium |
| 4 | 10 | `18b`: "Your internet line uses carrier-grade NAT… Ask your provider for a public IPv4 address (or use public IPv6)". Travel use is blocked, and the fix is jargon I'd have to repeat to the internet provider. | medium |
| 5 | 10 | `13` Step 1 needs the Tailscale admin console, which I don't have. "That lets your Tailscale machines reach devices… Devices can already reach your machines without it" reads as contradictory, so I can't tell if it's needed. | medium |
| 6 | 1 | The file name `tailbond_0.1.0-r42_aarch64_cortex-a53_neon-vfpv4.ipk` gives no sign it's right for the Slate 7. "It's the same tailbond this router runs" refers to a different router. | low |
| 7 | 1 | Fedora: the download is on the Devices tab, but the page opens on Connections (`14`). The steps leave out "log in with the same password" and "leave the other options as they are". | low |
| 8 | 2 | "Looking for your home server on your tailnet…": "tailnet" is unexplained jargon. | low |
| 9 | 2 | "Make this the home server (you can switch back here)" is on every travel-router screen and is easy to click by mistake. | low |
| 10 | 4 | `07b`: the limit "restarts on the 1st of each month" (not my plan's day). "Counts everything tailbond sends" could be mistaken for the whole phone plan. | low |
| 11 | 5 | "MagicDNS" isn't used anywhere. The bold short name "nas" stands out more than the full name the page says to use, and "Copy name" doesn't say which name it copies. | low |
| 12 | 6 | The switch has no On/Off label, while the red "Unpair" stands out more. A clearly labelled "Turn tailbond off" appears only on the error screen `08`. | low |
| 13 | 6 | `08`: with tailbond on, the cards say "This choice applies when tailbond is on. While it's off, GL's own settings… decide", and the status badges read "Ready". | low |
| 14 | 7 | Approval text: "it's a tagged machine", "under its access rules" is jargon. | low |
| 15 | 8 | "click "Login Mode for Public Hotspots" (or "Go to Login")": I don't know which one I'll see. "open any website on your laptop or phone" is unclear when the phone is the USB tether. | low |
| 16 | 9 | `17b`: "until 04:57 on this router's clock". The router's clock may not match mine, and it doesn't say what happens after 10 minutes. | low |
| 17 | 9 | `17`: the first small line about "GL's Login Mode covers the repeater Wi-Fi only" comes before the explanation of my actual problem, with mixed font sizes. | low |
| 18 | 10 | "To check: tailscale ping apple-tv ("via DERP" means relayed)" and "the guide fedora-server.md" point to commands and files a family member can't use. | low |
| 19 | 10 | "Through a relay: apple-tv. Only traffic between this machine and that machine is slower…": unclear whether it's worth fixing. | low |

Totals: 0 high / 5 medium / 14 low
