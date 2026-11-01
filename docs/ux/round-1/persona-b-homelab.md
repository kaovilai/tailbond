# Round 1: persona (b), homelab power user

Screens judged: `out/ux/after-1/*.png` only. I compare everything against the official Tailscale Android app.

## Task 1: Install tailbond and pair it with the home server

**Tap path:** none of the screens show how to install the app. For pairing, my path is: in Tailscale, find `homebox`, open `http://homebox.<tailnet>.ts.net:8077` in Chrome (web-01), tap "+ Add this phone" (web-02), tap "Open in tailbond", then tap "Add phone" in the "Add this phone to homebox?" dialog (03). The other way in is from the app (01 "Get started", then 02): type the server name and tap "Open home server page".

**Stuck points**
- **Install (high).** web-02 says "Nothing opened? Install the tailbond app first", and 01 says "You'll need a home server with tailbond installed." Neither screen has a store link, an APK link or a QR code to download the app. The script tells me to install from the phone, and no screen shows how. Tailscale links straight to Play Store / F-Droid.
- **The server dashboard toggle (medium).** web-01 has a "Stopped" switch in its header. web-02 shows "Connected" after I tapped "Add this phone". From my phone's browser I can't tell what this switch controls. It might be the server's own tunnel, or it might be my phone. There's also a "Disconnected" pill with "TUNNEL IP 10.77.0.1" and mode tiles. I might flip that switch and break the server for every device. I'd expect the dashboard to label it as the server's state, or hide it on the phone-onboarding view.
- **Two names for my phone (low).** web-01 and web-02 name the phone "pixel-8". The app dialog (03) then says "This phone will be called "pixel" on homebox." and Settings says `this phone is "pixel"`. I can't tell which name the server's Devices list will show.
- **Name format in 02 (low).** The field is labelled "Home server's Tailscale name". It doesn't say whether it wants the short name, the full MagicDNS FQDN or a 100.x address, and it doesn't mention the :8077 port. The hint "Find the name in the Tailscale app's machine list" helps.
- **Paste error (low).** 02-pairing-paste-error says "That isn't a complete tailbond link…" but the field is empty, so I can't see what I pasted or where it got cut off.

## Task 2: Connect

**Tap path:** on the home screen (12) tap the switch. Dialog 04 "Turn off Tailscale's VPN?" (or 05 "Allow tailbond's VPN" the first time) appears. Tap "Turn on tailbond", then the Android VPN prompt "OK". Screen 11 shows "Connecting…" and screen 10 shows "Connected".

**Stuck points**
- **Public endpoint shows up too late (medium).** I paired using a Tailscale name, so I assumed the tunnel rides the tailnet. It doesn't. 13 says "homebox didn't answer at home.example.net:443 … your home router forwards UDP port 443 to it", and Settings says "Reached at home.example.net:443". Neither pairing screen (02, 03) mentions the public hostname or the need for port forwarding. I'd want 03 to show "Connects to home.example.net:443 (UDP)" before I tap "Add phone".
- **Mixed signals while connecting (low).** Screen 11 says "Connecting…", but the This device row reads "No address while disconnected" and both links show "Ready" with grey dots. "Connecting…" or "Waiting" would be clearer.

## Task 3: Find and copy the NAS's MagicDNS name or Tailscale IP

**Tap path:** on 10 or 18, type `nas` in "Search machines" and tap the copy icon. The toast says "Copied 100.101.2.2". Tapping the row opens 19, where "MagicDNS name" can be copied ("Copied nas.tail1234.ts.net"). This task was easy, and it works much like Tailscale.

**Stuck points**
- **Copy icons behave differently (low).** In the list, the icon copies the IPv4 address. On the detail screen, the first icon copies the MagicDNS name. Only the toast tells me what landed on the clipboard. Tailscale shows a menu so I can pick.
- **Subnet routes (medium).** The Machines list has no word on subnet routes. I run subnet routers, so I need to know whether my advertised LAN routes (192.168.x) are reachable through homebox, or only 100.x peers. 19 says "apps on this phone can use these addresses just like with Tailscale", which covers peers only. Tailscale shows subnet routers and exit nodes in the device details.
- **"This device" address (low).** 10 shows "10.77.0.2 · Your Tailscale machines see this phone as homebox". That copy icon copies 10.77.0.2, which is useless to anyone else on the tailnet. The text also doesn't spell out the ACL consequence: the phone uses homebox's identity and permissions, and peers can't connect back to the phone.

## Task 4: Tell whether Wi-Fi and 5G are both being used right now

**Tap path:** 10 answers this directly. It says "Using both Wi-Fi and mobile data", Wi-Fi shows "● In use 42.5 Mbps" and Mobile data shows "● In use 25.6 Mbps". Tapping the Connections area (20) gives per-link throughput, delay and sparklines. I found nothing wrong here. 15 ("Using Wi-Fi only", Mobile data "● Not working") makes a failed link equally clear.

**Stuck points**
- **No reason for "Not working" (low).** 15 says "Not working" without a cause, such as no signal, UDP blocked, or the server unreachable over this link.

## Task 5: Make it save cellular data

**Tap path:** on 10, tap "Mode: Fastest  Change ›". On 20, under "How to use your connections", pick "Save mobile data". 21 and 16 then show "Using Wi-Fi · saving mobile data (backup only)". A monthly limit is also available under "Mobile data limit".

**Stuck points**
- **Two sets of mode names (medium).** The server web page (web-01) offers "Speed / Streaming / Redundant / Failover". The app offers "Fastest / Smooth calls and video / Most reliable / Save mobile data". I can't tell whether these are the same settings, whether the server tile overrides my phone, or whether it's a separate per-server setting.
- **Green dot on a standby link (low).** In 16 and 21, Mobile data shows a green dot next to "Saving data · backup only". Everywhere else, green means "carrying traffic". A standby link should get a neutral or amber dot.
- **"Main" and "Extra" (low).** In 20 and 21, the "Role" values "Main" and "Extra" are blue like links. I can't tell whether they're tappable, and the role seems to follow the mode anyway.
- **Related setting is elsewhere (low).** "Faster switch to backup (experimental)" only applies to "Save mobile data", but it sits in Settings › Advanced (30) instead of next to the mode on the Connections screen.

## Task 6: Turn tailbond off and go back to the Tailscale app

**Tap path:** on 10-full, scroll past the whole Machines list to "Switch back to Tailscale". Alternatively, tap the gear and then "Switch back to Tailscale" in 30. Result: 12b, "Tailscale is in charge on this phone", with an "Open Tailscale" button.

**Stuck points**
- **The home-screen button is hard to reach (medium).** On the home screen, "Switch back to Tailscale" sits below the full machine list (10-full). With a real tailnet of 30+ nodes it's a long scroll. The top switch only turns tailbond off and doesn't hand back to Tailscale. I'd expect the switch-back action near the main toggle or in the top bar.
- **Settings shows something other than what I picked (low).** 30 says "Apps that skip tailbond: None. Every app goes through tailbond", but 31 shows Netflix checked. If 31 was captured after the change, ignore this. Otherwise the summary is stale.

## Summary

| # | issue | severity | screen |
|---|---|---|---|
| 1 | No way to install the app is shown: "Install the tailbond app first" has no store/APK link | high | web-02, 01 |
| 2 | "Stopped/Connected" switch and tunnel stats on the server page are unlabelled; easy to mistake for the phone's own control | medium | web-01, web-02 |
| 3 | Public endpoint (home.example.net:443 UDP, port forward needed) only appears in an error and in Settings, not during pairing | medium | 03, 13, 30 |
| 4 | No info on subnet routes; can't tell whether LAN routes behind my subnet routers are reachable | medium | 10, 19 |
| 5 | Mode names differ between the server page (Speed/Streaming/Redundant/Failover) and the app (Fastest/…/Save mobile data); unclear which wins | medium | web-01, 20 |
| 6 | "Switch back to Tailscale" on the home screen is below the whole machine list | medium | 10-full |
| 7 | Phone named "pixel-8" on the web page but "pixel" in the app and Settings | low | web-01, 03, 30 |
| 8 | Server-name field doesn't say short name vs FQDN, or the port | low | 02 |
| 9 | Paste error clears the field, so I can't see what was wrong | low | 02-pairing-paste-error |
| 10 | Connecting state says "No address while disconnected" and links "Ready" | low | 11 |
| 11 | Copy icon copies IPv4 in the list but MagicDNS on the detail screen; only the toast reveals which | low | 18, 19 |
| 12 | This-device copy gives 10.77.0.2 (useless to peers); ACL consequence of "seen as homebox" not explained | low | 10 |
| 13 | "Not working" link status gives no reason | low | 15 |
| 14 | Green dot on the standby mobile link ("Saving data · backup only") | low | 16, 21 |
| 15 | "Main"/"Extra" role values look tappable, but it's unclear whether they are | low | 20, 21 |
| 16 | "Faster switch to backup" in Settings › Advanced, away from the Save-data mode | low | 30 |
| 17 | Settings says "None" skip tailbond while the apps list has Netflix checked | low | 30, 31 |
