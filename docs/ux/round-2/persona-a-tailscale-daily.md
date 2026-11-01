# Round 2 – Persona (a): daily Tailscale Android user, product designer

Screens: `out/ux/after-2/` (light + dark). Dark variants match light; no extra issues found in them.

## Task 1 – Install tailbond and pair it, from the phone

**Tap path:** browser → home server page (`web-01-add-this-phone`) → "Download tailbond for Android" → install → back to page, keep name "pixel-8" → "+ Add this phone" (`web-02-after-tap`) → "Open in tailbond" → app opens with "Add this phone to homebox?" (`03-deeplink-confirm`) → "Add phone".

Fallback if I open the app first: `01-onboarding` "Get started" → `02-pairing` "On this phone (easiest)" → type name → "Open home server page".

**Stuck points:**
- I have to already know the server page's address. The page lives at `http://<server>.<tailnet>.ts.net:8077`. I can't guess the `:8077` part, and nothing in the Tailscale app shows it. The app's own helper ("Home server's Tailscale name" + "Open home server page") would fill it in, but I can't use that until the app is installed, and the download link is on that same page. If whoever set it up only told me "it's homebox", I'm stuck. **Medium.**
- "Home server's Tailscale name": I can't tell whether to type `homebox` or `homebox.tail1234.ts.net`. Tailscale shows both. **Low.**
- The page header says "Home server · home.example.net:443". The confirm dialog says "It connects over the internet to home.example.net:443." I was told this is a Tailscale machine, and here it has a different, non-tailnet address. It makes me wonder whether I'm on the right server. **Low.**

**Expected (Tailscale-like):** something like Tailscale's login link or QR code that you can reach without knowing a port, or the address shown in a form I recognise (the MagicDNS name).

## Task 2 – Connect

**Tap path:** `12-home-disconnected` big switch → `04-handoff-other-vpn` "Turn off Tailscale's VPN?" → "Turn on tailbond" → Android OK → `11-home-connecting` → `10-home-connected` "Connected".

**Stuck points:** none. The switch and the "Connected" card feel like Tailscale. The dialog explains why Tailscale turns off.

**Side note on the result** (`10-home-connected`, "This device"): it shows "10.77.0.2" with a copy icon, where Tailscale would show my 100.x address. The line "This phone isn't a Tailscale machine itself" helps, but I still don't know what 10.77.0.2 is for. Can the NAS or my laptop reach the phone at that address? Is it worth copying? With Tailscale I copy my own IP to share it. **Medium.**

## Task 3 – Find and copy the NAS's MagicDNS name or IP

**Tap path:** scroll down on `10-home-connected-full` → "Search machines", type "nas" → copy icon (`18-machines-search-copied`, toast "Copied 100.101.2.2"). Or tap the row → `19-machine-detail` → copy "MagicDNS name" (toast "Copied nas.tail1234.ts.net").

**Stuck points:**
- On the list row, the MagicDNS name is shown first, but the single copy icon copied the IP ("Copied 100.101.2.2"). I couldn't tell beforehand which one I'd get. The detail page solves it. **Low.**
- Machines is the main thing I use in Tailscale. Here it's below the fold, under Connections, Save mobile data and Internet traffic, so I have to scroll to find it. **Low.**

**Expected:** Machines high up, like Tailscale. The detail page ("Tap an address to copy it", MagicDNS name / IPv4 / IPv6) matches Tailscale well.

## Task 4 – Are Wi-Fi and 5G both in use right now?

**Tap path:** `10-home-connected`: "Using both Wi-Fi and mobile data", with green "In use" on both tiles. `15-one-path-failed` shows clearly when one connection is broken ("Using Wi-Fi only", "Not working right now").

**Stuck points:** none.

## Task 5 – Save cellular data

**Tap path:** `10-home-connected` "Save mobile data" switch → `16-save-data-standby` ("Using Wi-Fi · saving mobile data (backup only)", mobile data "Not in use · saving data"). `17-save-data-takeover` shows mobile data taking over when Wi-Fi drops.

**Stuck points:**
- After tapping "Change" (`20-connections` / `21-connections-save-data`) I find several other controls that also seem to save data: the "Save mobile data" mode option, mobile data "Use for tailbond" switch, "Role: Extra", "Mobile data limit", and "Faster switch to backup (experimental)" ("Keeps mobile data warmed up… Uses a little mobile data"). I finished the task with one tap. But now I'm unsure whether I set the right thing, and whether turning the home switch off puts me back on "Fastest". **Low.**
- Separately, "Internet traffic: Through your home server" is on by default (Settings: "All internet traffic leaves from home, like using an exit node"). In Tailscale I don't use an exit node. I didn't choose this, and I'd only notice it by reading the row. **Low.**

## Task 6 – Turn tailbond off and go back to Tailscale

**Tap path:** `10-home-connected` "Switch back to Tailscale" → `12b-home-switched-back-to-tailscale`: "Tailscale is in charge on this phone… If Tailscale isn't connected yet, open it and turn on its switch." → "Open Tailscale" → turn Tailscale's switch on.

**Stuck points:**
- In Settings the same row says "Turns tailbond off and opens Tailscale". But `12b` shows me still inside tailbond, with a separate "Open Tailscale" button and a note that I may have to turn Tailscale's switch on myself. So does it open Tailscale or not? And is my tailnet back yet? I'd finish, but with a second guess and extra taps. **Low.**

**Expected:** one tap that turns tailbond off and lands me in Tailscale, or at least a statement of whether Tailscale is connected now.

## Summary

| # | issue | severity | screen |
|---|---|---|---|
| 1 | You need the server page URL (incl. `:8077`) before the app is installed; the app's "Open home server page" helper only works after install, so if all I know is "homebox" I'm stuck | medium | web-01-add-this-phone, 02-pairing |
| 2 | "This device" shows "10.77.0.2" (not 100.x) with a copy icon; unclear what it's for or whether tailnet machines can reach the phone at it | medium | 10-home-connected |
| 3 | "Home server's Tailscale name": unclear whether to type the short name or the full MagicDNS name | low | 02-pairing |
| 4 | Server shown as "home.example.net:443" (web header, "It connects over the internet to…") instead of the tailnet name I know; makes me doubt it's the right server | low | web-01-add-this-phone, 03-deeplink-confirm |
| 5 | Machine row shows MagicDNS name first, but its single copy icon copies the IP ("Copied 100.101.2.2") | low | 18-machines-search-copied |
| 6 | Machines list is below the fold under the connection controls; in Tailscale it's the main content | low | 10-home-connected |
| 7 | Several overlapping data controls (home switch, "Save mobile data" mode, "Use for tailbond", Role "Extra", monthly limit, "Faster switch to backup"); unclear which one is the right one and what turning the switch off restores | low | 20-connections, 21-connections-save-data |
| 8 | "Internet traffic: Through your home server" (exit-node-like) is on by default without the user choosing it | low | 10-home-connected, 30-settings |
| 9 | Settings says "Switch back to Tailscale… opens Tailscale", but the result stays in tailbond with an "Open Tailscale" button and "If Tailscale isn't connected yet… turn on its switch" | low | 12b-home-switched-back-to-tailscale, 30-settings |

Counts: high 0, medium 2, low 7.
