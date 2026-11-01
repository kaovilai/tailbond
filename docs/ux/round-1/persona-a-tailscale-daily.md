# Round 1: persona (a), daily Tailscale Android user, product designer

Screens: `out/ux/after-1/`. I judged only from what the screenshots show.

## Task 1: Install tailbond and pair it with the home server

**Tap path:** I already have Tailscale on, so I open the home server page in Chrome (`web-01-add-this-phone`) and tap **"+ Add this phone"**. Then `web-02-after-tap` appears and I tap **"Open in tailbond"**. The app opens on `03-deeplink-confirm` and I tap **"Add phone"**. Another route: open the app first (`01-onboarding`), tap **"Get started"**, which leads to `02-pairing`, then **"On this phone (easiest)"**, type the server name, and tap **"Open home server page"**.

**Stuck points:**
- **I can't find anywhere to install the app.** No screen tells me where to get tailbond. The web page only says "Nothing opened? Install the tailbond app first, then tap Open in tailbond." It gives no Play Store button, no download link, and no QR code for the APK. The onboarding screen starts with the app already installed. I'd be stuck at the very first step. (high)
- **The server page's own on/off switch.** The server page header has a toggle with the label "Stopped" (`web-01`). After I tapped Add this phone, it showed "Connected" (`web-02`), even though I hadn't opened the app yet. Below it are "0 BPS DOWN", "TUNNEL IP 10.77.0.1" and tiles labelled "Speed / Streaming / Redundant / Failover". On my phone this looks like the control for *my* connection. I can't tell whether flipping it would stop the server for the whole family. It also makes me think I'm already connected when I'm not. (medium)
- **The phone's name changes.** I typed or accepted "pixel-8" on the web page (`web-01`). The app's confirm dialog then says "This phone will be called "pixel" on homebox." (`03`). That makes me wonder whether I'm confirming a different or stale pairing. (medium)
- **The server's name changes.** The same box is called "tailbond · Home server · home.example.net:443" on the web page and "homebox" in the app. I have to trust that they're the same machine. (low)
- **The server name field.** In the app, `02-pairing` asks for "Home server's Tailscale name" with the hint "Find the name in the Tailscale app's machine list (the home server)." My Tailscale machine list doesn't label any machine as the "home server", so I'd have to guess which one it is. (low)
- **Paste error jargon.** The paste error (`02-pairing-paste-error`) shows "Details: pairing data is not base64url". That line means nothing to me. The first line of the error is fine. (low)

**What I expected (Tailscale):** Tailscale installs from the Play Store and signs in from one obvious button. Any page that sends me to an app links straight to its store listing. Names stay the same everywhere.

## Task 2: Connect

**Tap path:** I tap the big switch on the home screen (`12-home-disconnected`). The dialog `04-handoff-other-vpn` ("Turn off Tailscale's VPN?") or `05-handoff-first-permission` opens, and I tap **"Turn on tailbond"**. `11-home-connecting` shows "Connecting… Reaching homebox…", then `10-home-connected` shows "Connected".

**Stuck points:** None that matter. The big switch and the "Connected" card match what I know from Tailscale. The dialog explains that Tailscale turns off and that I can still reach the NAS.

The error screens use one line of jargon: "your home router forwards UDP port 443 to it" (`13-error-unreachable`). It tells me what to forward, but I don't know what that means. I'd just send it to whoever set this up. (low)

## Task 3: Find and copy the NAS's MagicDNS name or Tailscale IP

**Tap path:** On `10-home-connected-full`, I scroll to **Machines** and type "nas" in **"Search machines"** (`18`). Then I tap the copy icon and get the toast "Copied 100.101.2.2". Or I tap the row, which opens `19-machine-detail`, and tap the MagicDNS row ("Copied nas.tail1234.ts.net").

**Stuck points:**
- **The row's copy icon copies something I can't predict.** The row shows `nas.tail1234.ts.net` in the larger text, but the icon copies the IP ("Copied 100.101.2.2"). I expected to get the name. The toast tells me what was copied, so I recover, but I'd need a second try. (low)
- **"This device" doesn't look like Tailscale's "This device".** It shows "pixel 10.77.0.2" and says "Your Tailscale machines see this phone as homebox". In Tailscale, This device is my phone's own 100.x address and MagicDNS name, and I copy it to share with others. Here the copy icon gives a 10.x address that other machines can't use. My phone also has the same name as the server. If my wife asks "what's your phone's Tailscale address?", I have no answer. (medium)

**What I expected (Tailscale):** The rest of this task works exactly as I expected. The machine detail page with MagicDNS name, IPv4 and IPv6 copy rows is the same as in Tailscale.

## Task 4: Tell whether Wi-Fi and 5G are both used right now

**Tap path:** Home screen (`10-home-connected`) shows "Using both Wi-Fi and mobile data", with both cards on "In use" and green dots. Tapping Connections opens `20-connections`.

**Stuck points:** None. This is clear at a glance and the clearest part of the app.

In `16-save-data-standby`, the mobile data card has a **green** dot next to "Saving data · backup only". Everywhere else, green means "In use", so for a second I think mobile data is being used. (low)

## Task 5: Make it save cellular data

**Tap path:** On the home screen, I tap **"Mode: Fastest · Change >"**. That opens `20-connections`. I scroll past the graphs to **"How to use your connections"** and pick **"Save mobile data"** (`21`). The home screen then shows "Mode: Save mobile data" (`16`).

**Stuck points:**
- **Mode names don't match between the web page and the app.** The web page calls the modes "Speed / Streaming / Redundant / Failover", and none of them sounds like saving data. The app calls them "Fastest / Smooth calls and video / Most reliable / Save mobile data". If I had looked on the web page first, I wouldn't have found the setting. (low)
- **Blue labels that look like buttons.** "Role: **Main**" and "Role: **Extra**" are blue, link-like labels, but tapping them doesn't seem to do anything. I'd tap them, expecting to change something. (low)

## Task 6: Turn tailbond off and go back to Tailscale

**Tap path:** I scroll to the bottom of the home screen and tap **"Switch back to Tailscale"** (`10-home-connected-full`). It's also in Settings (`30-settings`). After that, `12b` shows "Tailscale is in charge on this phone" with **"Open Tailscale"**.

**Stuck points:**
- **"Switch back to Tailscale" sits under the whole machine list.** I only see it after scrolling past every machine (`10-home-connected-full`). With a real tailnet of 20+ devices, it's far away. My first move would be to flip the big switch off. That only turns tailbond off and doesn't bring Tailscale back. (low)

**What I expected (Tailscale):** It works. The wording "tailbond stays off until you turn it on again" is clear.

## Summary

| # | issue | severity | screen |
|---|---|---|---|
| 1 | No way to install the app is shown. The web page says "Install the tailbond app first" but has no store or download link. | high | web-02-after-tap, 01-onboarding |
| 2 | The server page's header switch ("Stopped" / "Connected") and speed tiles look like this phone's controls. The switch shows "Connected" before the app has even opened, and it's unclear whether tapping it stops the server for everyone. | medium | web-01-add-this-phone, web-02-after-tap |
| 3 | The phone name is "pixel-8" on the web page but "pixel" in the confirm dialog, so I can't tell whether this pairing is mine. | medium | web-01, 03-deeplink-confirm |
| 4 | "This device" shows 10.77.0.2 and "Tailscale machines see this phone as homebox". There's no 100.x address or MagicDNS name of its own, and the copy icon gives an address other machines can't use. | medium | 10-home-connected |
| 5 | The server is called "home.example.net:443" on the web page and "homebox" in the app. | low | web-01, 03, 30-settings |
| 6 | "Home server's Tailscale name" assumes I know which machine in my Tailscale list is the server. | low | 02-pairing |
| 7 | The paste error shows "Details: pairing data is not base64url". | low | 02-pairing-paste-error |
| 8 | The unreachable error asks me to check "forwards UDP port 443". | low | 13-error-unreachable |
| 9 | The row copy icon copies the IP, while the most prominent text in the row is the MagicDNS name. | low | 18-machines-search-copied |
| 10 | A green dot next to "Saving data · backup only" reads as "in use". | low | 16-save-data-standby, 21-connections-save-data |
| 11 | Mode names differ between the web page (Speed/Streaming/Redundant/Failover) and the app (Fastest/…/Save mobile data). | low | web-01, 20-connections |
| 12 | The blue "Main" / "Extra" role labels look tappable. | low | 20-connections |
| 13 | "Switch back to Tailscale" is only reachable after scrolling past the whole machine list. | low | 10-home-connected-full |
