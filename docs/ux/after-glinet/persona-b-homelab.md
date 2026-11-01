# Persona B (homelab power user): raw report, after-glinet screenshots

Who I am: I run Tailscale subnet routers and exit nodes at home and use the
official Tailscale Android app every day. I know MagicDNS names, 100.x
addresses, exit nodes and subnet routes. I only had the screenshots in
`out/ux/after-glinet/` and the manifest in `docs/ux/screens.md`.

## Task 1: Install tailbond and pair it with the home server, from the phone

### Install

- The only install path in the screenshots is the server's web page
  (`web-01-add-this-phone`): step 1 "Get the tailbond app (skip if it's
  installed)", button "Download tailbond for Android", then a sideload note:
  "Android warns about apps from outside the Play Store and asks you to allow
  installs from your browser: this app comes from your own home server." That
  is clear and honest, and for me sideloading is fine.
- Where I get stuck: how do I reach that page in the first place, from the
  phone alone? Nothing I was given says the URL. The manifest says it's
  `http://<server>.<tailnet>.ts.net:8077`, but on screen the port 8077 only
  appears *inside the tailbond app* (`02b`: "open its tailbond page
  (http://homebox:8077 on a computer, ...)"), which I can't see before
  installing. I can find `homebox` in the Tailscale app's machine list, but
  nothing tells me to try port 8077 on it. With no Play Store listing shown,
  I'd have to ask whoever set it up for the link. A power user can guess a
  port scan, but a normal person can't, and I shouldn't have to.
- Expected (Tailscale): you install from the Play Store and sign in. Here I
  at least expect the person who set it up to hand me one link, or the server
  name in the Tailscale app to lead somewhere (e.g. port 80 redirecting to the
  page).
- Severity: **medium** (I'd finish only after asking someone for the URL).

### Pair (once installed)

- `01-onboarding`: clear. "Use tailbond instead of Tailscale on this phone",
  "Still on your tailnet ... reachable, by name or 100.x address, through it",
  "One VPN at a time". Footer tells me exactly what's next: "Next you'll type
  your home server's name (the one in the Tailscale app) and ask it to
  connect." I tap **Get started**.
- `02-pairing`: first card "Ask your home server": "Keep Tailscale on for
  now. Type your home server's name as it appears in the Tailscale app, and
  tap Ask to connect." I'd type `homebox` and tap **Ask to connect**. This is
  the from-the-phone path and it's good: no QR from another screen needed.
  - Small doubt: does the field take `homebox`, `homebox.tail1234.ts.net`, or
    `100.101.1.1`? Hint text "Home server's name in Tailscale" suggests the
    short name; I'd try that. Low.
  - "Open home server page" in the second card is greyed out with no reason
    shown ("Came here from your home server's page? Go back to it in your
    browser..."). I'd wonder why I can't tap it. Low.
- `02b-pairing-ask-waiting`: "Waiting for approval on homebox... Ask whoever
  runs homebox to open its tailbond page (http://homebox:8077 on a computer,
  or a Flint 3's admin panel → VPN → tailbond) and tap Approve next to
  "pixel-8"."
  - If I am the one who runs homebox, I'd want to approve from the same
    phone: switch to Chrome, open http://homebox:8077 (Tailscale is still on,
    so it resolves). But the text says "on a computer", and no screenshot
    shows the Approve list on the web page, so I can't confirm the phone
    browser view shows it. Not blocking because of the other path below. Low.
  - "a Flint 3's admin panel" is jargon for me: I don't have a GL.iNet
    router; I'd ignore it, but it reads like an ad for hardware. Low.
- Alternative path, fully on the phone: `web-01` → tap **+ Add this phone**
  → `web-02` "Opening tailbond... In the app, tap Add phone, then turn
  tailbond on" with **Open in tailbond** / **Copy link** → app shows
  `03-deeplink-confirm` "Add this phone to homebox?" → **Add phone**. This
  works without anyone approving and without a computer. Good.
- `02e`/`03` confirm dialog: "It connects over the internet to
  home.example.net:443, and your internet traffic goes out from home (you can
  change that in Settings)." Two power-user notes:
  - The bond runs over the public internet to a forwarded port, not over the
    tailnet. Fine, that's the point, and it's stated. Low.
  - Full-tunnel by default ("internet traffic goes out from home") differs from
    Tailscale, where no exit node is the default. It's disclosed and Settings
    has "Only Tailscale machines". Low.
- Error states are understandable: `02c` "Can't find homebox — Check that the
  Tailscale app says Connected, and that the name matches the one in its list
  of machines." `02d` "homebox said no". `02-pairing-paste-error` "That isn't
  a complete tailbond link. Copy the whole link starting with tailbond://pair".
  All fine.

**Could I find and pair with the home server from the phone alone?** Pairing:
yes. I know the server name from the Tailscale app, type it, and either get
approved or use the web page's "Add this phone", which needs no approver.
Install: only if someone gives me the web page URL. Nothing on the phone shows
port 8077 until tailbond is already installed.

## Task 2: Connect

- Home screen (`12-home-disconnected`): "Disconnected — Stopped. Turn on to
  reach homebox and your Tailscale machines, or switch back to Tailscale
  below." Tap the switch.
- `04-handoff-other-vpn`: "Switch from Tailscale to tailbond? Android runs one
  VPN at a time. Turning on tailbond turns off Tailscale ... You still reach
  your Tailscale machines, like your NAS, through homebox. Next, Android asks
  you to allow tailbond's connection request. Tap OK." → **Turn on tailbond**
  → Android OK. (`05` is the same for the first-time case.) Clear.
- `11-home-connecting` → `10-home-connected`: "Connected — Through homebox ·
  Wi-Fi + mobile data". Good, matches Tailscale's big switch.
- Power-user note: "This device: pixel-8 (this phone) 10.77.0.2 — Your
  Tailscale machines can reach this phone at this address, through homebox."
  So the phone isn't a tailnet node with its own 100.x; it sits behind
  homebox like a subnet route. Makes sense to me, and it tells me ACLs and
  `tailscale ping pixel-8` won't work as before. Nothing on screen says homebox
  must advertise (and I must approve) that route in the admin console, but
  the setup person would handle that. Low.
- Errors: `13-error-unreachable` names the cause ("the home router may need to
  forward UDP port 443 to it"), which I appreciate. `14-error-cert` "Can't
  confirm this is your home server ... Don't pair this phone again unless they
  ask you to" is the right security message.
- Severity: none / low.

## Task 3: Find and copy the NAS's MagicDNS name or Tailscale IP

- `10-home-connected-full`: Machines list mirrors Tailscale's: "nas ·
  nas.tail1234.ts.net · 100.101.2.2 · linux" with a copy icon. Search box
  "Search machines".
- `18-machines-search-copied`: type `nas`, tap the copy icon → toast "Copied
  100.101.2.2". Done.
- `19-machine-detail`: tap the row → "Tap an address to copy it" with
  MagicDNS name / Short name / Tailscale IPv4 / Tailscale IPv6, each with a
  copy icon; toast "Copied nas.tail1234.ts.net". This is the same as the
  Tailscale app's machine detail. Footer: "Reached through homebox. With
  tailbond on, apps on this phone can use these addresses just like with
  Tailscale."
- Small thing: on the list, the row's copy icon copies the IP, not the
  MagicDNS name, and you only learn which from the toast. Low.
- Severity: low.

## Task 4: Are Wi-Fi and 5G both being used right now?

- `10-home-connected`: "Connections — Using both Wi-Fi and mobile data", two
  tiles "Wi-Fi ● In use ↓38.4 ↑4.1 Mbps" and "Mobile data ● In use ↓22.7
  ↑2.9 Mbps". Status line also says "Wi-Fi + mobile data". Immediately clear.
- `15-one-path-failed`: "Using Wi-Fi only", mobile tile red "Not working right
  now". `16c`: "Wi-Fi needs sign-in · using mobile data" with a **Sign in to
  Wi-Fi** button. Both clear.
- `20-connections` (tap Connections / "More options"): per-link delay
  (21 ms / 48 ms), graph, "2.05 GB this session". Nice for me.
- Severity: none.

## Task 5: Make it save cellular data

- `10-home-connected`: "Save mobile data — Use mobile data only when Wi-Fi
  stops working" switch, right on the home screen. Flip it.
- `16-save-data-standby`: "Using Wi-Fi · saving mobile data (backup only)",
  mobile tile "Not in use · saving data", "Mode: Save mobile data". `17`
  shows mobile taking over when Wi-Fi drops. Clear.
- `21-connections-save-data`: the same choice as a radio, with the note "The
  Save mobile data switch on the home screen picks "Save mobile data" here",
  plus a separate "Mobile data limit" ("1.23 GB of 5.00 GB used this month",
  "Separate from Save mobile data"). Also the per-link "Use for tailbond"
  switches. Lots of options, but the relationship is explained.
- Settings "Apps that skip tailbond" (`31`) could also save data, but it's a
  different thing (and "They can't reach your Tailscale machines" warns
  correctly).
- Severity: none / low.

## Task 6: Turn tailbond off and go back to the Tailscale app

- "Switch back to Tailscale" is a full-width button right under the status card
  on every home state (`10`, `11`, `13`, `14`, `16`, `17`) and in Settings
  ("Turns tailbond off and opens the Tailscale app. Turn Tailscale on there if
  it says Disconnected"). The web page and the turn-on dialog also say so.
- `12b-home-switched-back-to-tailscale`: "Disconnected — Stopped. Tailscale is
  in charge on this phone." Card: "tailbond stays off until you turn it on
  again. In the Tailscale app, turn on its switch if it says Disconnected."
  **Open Tailscale**. Clear; I know Tailscale won't auto-reconnect when
  another VPN took over, so that instruction is correct.
- Settings "Keep tailbond on all the time (optional)": "Turn off Always-on for
  Tailscale first: only one app can have it. Leave "Block connections without
  VPN" off if you want to switch back to Tailscale easily." Good warning for
  someone like me who has Always-on set for Tailscale.
- Severity: none.

## Other observations

- "via home server · tail1234.ts.net" under the title is a nice nod to the
  tailnet name.
- "Connecting... This is the list from 20 min ago." / "Not connected. This is
  the list from 20 min ago; connect to reach these machines." Honest about
  stale data, unlike Tailscale which just greys out.
- Dark variants not needed; light screens were readable.

## Findings

| # | task | finding | severity |
|---|---|---|---|
| 1 | 1 | Installing from the phone alone needs the server's web page URL (`http://homebox:8077`), which isn't shown anywhere I can see before installing; port 8077 appears only inside the app (`02b`). I'd have to ask whoever set it up. | medium |
| 2 | 1 | "Open home server page" (`02-pairing`) is greyed out with no reason given. | low |
| 3 | 1 | Waiting text says approve "on a computer, or a Flint 3's admin panel"; the Flint 3 reference is unexplained jargon for non-GL.iNet users, and it's unclear whether I can approve from this phone's browser. | low |
| 4 | 1 | Unclear whether the name field accepts short name, MagicDNS FQDN, or 100.x IP. | low |
| 5 | 1 | Full-tunnel ("your internet traffic goes out from home") is the default, unlike Tailscale's no-exit-node default; it's disclosed and changeable in Settings. | low |
| 6 | 2 | Phone gets 10.77.0.2 (behind homebox), not a 100.x tailnet identity; doesn't mention that this relies on a subnet route on homebox (ACLs, `tailscale ping pixel-8` won't work as before). | low |
| 7 | 3 | List-row copy icon copies the IP, not the MagicDNS name; you only find out from the toast. | low |
