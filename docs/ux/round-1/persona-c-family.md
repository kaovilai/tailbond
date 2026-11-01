# Round 1: persona (c), less-technical family member

Persona: retired teacher in her 60s. Her son set up Tailscale. She uses it to flip the big switch on and to copy "nas" for the photo app. She doesn't know IP, DNS, TLS, tunnel or pairing, beyond "scan the square code". Red text and settings she can't explain make her anxious.

Screens reviewed: all PNGs in `out/ux/after-1/`, light and dark.

---

## Task 1: Install tailbond and pair it with the home server

**Tap path:** No screen shows how to install. I would wait for my son, or look in the Play Store myself. Then:
- `01-onboarding`: tap "Get started".
- `02-pairing`: the card marked "On this phone (easiest)" asks for "Home server's Tailscale name".
- Or I open the server page in the browser (`web-01`): tap "+ Add this phone", then "Open in tailbond" (`web-02`).
- `03-deeplink-confirm`: tap "Add phone".

**Stuck points:**

1. **I don't know the server's name.** (`02-pairing`) The "easiest" option needs a box filled in: "Home server's Tailscale name". The hint says "Find the name in the Tailscale app's machine list (the home server)." My Tailscale list shows several machines (homebox, nas, apple-tv…), and none of them is labelled "home server" in the Tailscale app. I couldn't tell which one to type. The "Open home server page" button stays grey until I do. The other two options need a computer ("If the home server's page is open on a computer…") or a link I don't have. I'd stop here and phone my son.
   - *Expected:* In Tailscale, signing in is one button and I pick from a list. Here I'd expect the app to show my Tailscale machines so I can tap one, or to guess the home server for me.
   - **Severity: high.**

2. **The server page is full of switches and numbers I shouldn't touch.** (`web-01`, `web-02`) The page I'm sent to has a big switch at the top labelled "Stopped". There's also "BPS DOWN", "TUNNEL IP 10.77.0.1", four buttons ("Speed", "Streaming", "Redundant", "Failover") and tabs ("Connections / Devices / Tailscale / Logs"). For me, the big switch is *the* thing I flip in Tailscale, so I'd likely flip this one too. That would change the home server for the whole family. In `web-02` that switch reads "Connected" after I tapped Add this phone, so I'd think I had done something to it.
   - *Expected:* A page that is just "Add this phone", with the server controls hidden or clearly marked "for the person who set this up".
   - **Severity: high.**

3. **Nowhere to get the app.** (`web-02`) The page says "Nothing opened? Install the tailbond app first, then tap Open in tailbond." There's no button or link to the Play Store.
   - *Expected:* A "Get it on Google Play" button.
   - **Severity: medium.**

4. **The phone's name changes.** (`web-01` → `03-deeplink-confirm`) On the web page I named the phone "pixel-8". The app then says: "This phone will be called "pixel" on homebox." It looks like it ignored me, or that I'm adding a different phone.
   - **Severity: medium.**

5. **Red error text with computer words.** (`02-pairing-paste-error`) It says: "That isn't a complete tailbond link. Copy the whole link starting with tailbond://pair". Under that is "Details: pairing data is not base64url". The first sentence is fine. The "Details" line is gibberish to me and makes me think I broke something.
   - *Expected:* Only the plain sentence. Hide the details behind a tap.
   - **Severity: medium.**

6. **Warnings that worry me.** (`03-deeplink-confirm`, `web-02`) "Secure: homebox's certificate is checked" and "The link holds this phone's key: don't share it." They're meant to reassure, but "key" and "certificate" make me nervous.
   - **Severity: low.**

## Task 2: Connect

**Tap path:** I flip the switch on the main screen (`12-home-disconnected`). The "Turn off Tailscale's VPN?" box (`04`) or "Allow tailbond's VPN" box (`05`) appears, and I tap "Turn on tailbond". I tap OK on Android's request. The screen then shows "Connected" (`10`).

That part is fine. The big switch and the "Connected" panel feel like Tailscale, and the box tells me what to do next ("Tap OK").

**Stuck points:**

7. **"Pair again" is the big button on a scary message.** (`14-error-cert`) A red box says "Your home server's identity changed". It goes on: "homebox's security certificate doesn't match… If you reinstalled or renamed it, pair this phone again. Otherwise, don't connect on this network." I don't know whether anyone reinstalled anything. The dark, filled button is "Pair again", so I'd press it, even though the message says that might be the wrong thing.
   - *Expected:* Tell me to ask whoever set it up. Don't make the risky choice the obvious button.
   - **Severity: high.**

8. **A fix I can't do.** (`13-error-unreachable`) "Check that it's turned on and that your home router forwards UDP port 443 to it, then try again." I can press "Try again", but the rest means nothing to me. Nothing says "ask the person who set this up".
   - *Expected:* In plain words, something like "Your home server isn't answering. Check it's plugged in, or ask [son]." Put the router details behind a tap.
   - **Severity: medium.**

9. **The phone seems to have two names.** (`10-home-connected`, and on every main screen) Under "This device", it says "pixel", then "Your Tailscale machines see this phone as homebox". So is my phone "pixel" or "homebox"? The big title at the top of the screen is also "homebox". That makes me think I'm looking at the server and not my phone.
   - *Expected:* In Tailscale the top shows my account and my phone has one name.
   - **Severity: medium.**

## Task 3: Find and copy the NAS address

**Tap path:** On `10-home-connected` I scroll to "Machines", type "nas" in "Search machines" and tap the copy icon. The message "Copied 100.101.2.2" appears (`18`). Or I tap the "nas" row, which opens `19-machine-detail`. There, "Tap an address to copy it" lists each address with its own copy button.

This works, and it's close to Tailscale. The nas detail page is clear.

**Stuck points:**

10. **My own phone has a copy button above the machine list.** (`10`, `18`) The "This device" row ("pixel 10.77.0.2") sits above the machine list with the same copy icon. In a hurry I might copy that number instead. It also doesn't start with "100." like the addresses my son taught me to look for.
    - **Severity: low.**

## Task 4: Are Wi-Fi and 5G both being used?

**Tap path:** I read the main screen (`10`). It says "Using both Wi-Fi and mobile data", and both tiles say "In use" with a green dot. `15` clearly says "Using Wi-Fi only", with "Mobile data: Not working" next to a red dot.

No problems. This is the clearest part of the app.

## Task 5: Make it save mobile data

**Tap path:** On the main screen I tap "Change >" next to "Mode: Fastest". That opens `20-connections`. I scroll past the graphs and tap "Save mobile data" under "How to use your connections". The main screen then reads "Mode: Save mobile data" (`16`).

**Stuck points:**

11. **Too many ways to "save data" on one long page.** (`20-connections`) Before I reach the right choice, I pass two graphs, "delay 21 ms", and a "Use for tailbond" switch on the Mobile data card. I'd probably turn that switch off, thinking it's "the save data switch". Then there's "Role Main / Extra" and a "Mobile data limit" box already set to "5". Settings also has "Faster switch to backup (experimental)… Uses a little mobile data" (`30-settings`). I'm not sure which one is the right one, and I worry about pressing the wrong one.
    - *Expected:* One clear "Save mobile data" switch or choice at the top, with the technical parts lower down.
    - **Severity: medium.**

## Task 6: Turn tailbond off and go back to Tailscale

**Tap path:** I scroll to the bottom of the main screen and tap "Switch back to Tailscale" (`10-home-connected-full`). The same option is in Settings (`30`). Afterwards I see "Tailscale is in charge on this phone" and an "Open Tailscale" button (`12b`).

**Stuck points:**

12. **I can't tell whether Tailscale is really back on.** (`12b`, `10-home-connected-full`) The button's note says "Turns tailbond off and opens the Tailscale app, so Tailscale's VPN *can* connect again." After that, the screen only says "Tailscale is in charge on this phone" and offers "Open Tailscale". Do I still need to flip Tailscale's switch myself? Nothing says so. If I just flip tailbond's switch off instead (the thing I'd naturally do), nothing reminds me to turn Tailscale back on. Then my photo app can't reach the NAS, and I won't know why.
    - *Expected:* Say plainly "Now turn on the switch in Tailscale", or confirm that Tailscale is connected.
    - **Severity: medium.**

13. **The button is far down.** (`10-home-connected-full`) "Switch back to Tailscale" sits below the whole machine list, so you only see it if you scroll.
    - **Severity: low.**

---

## Summary

| # | issue | severity | screen |
|---|---|---|---|
| 1 | "Easiest" pairing needs "Home server's Tailscale name" typed in; the Tailscale app never says which machine is the home server, so a non-technical user is stuck (other options need a computer or a link) | high | 02-pairing |
| 2 | Server web page shows the home server's own "Stopped/Connected" switch, mode buttons and stats; the user would flip the big switch and change the server for everyone | high | web-01, web-02 |
| 3 | "Install the tailbond app first" but no Play Store link/button | medium | web-02 |
| 4 | Phone named "pixel-8" on web page, app says "This phone will be called "pixel"" | medium | web-01, 03-deeplink-confirm |
| 5 | Red paste error includes "Details: pairing data is not base64url" | medium | 02-pairing-paste-error |
| 6 | "certificate is checked" / "holds this phone's key" wording unsettles non-technical users | low | 03-deeplink-confirm, web-02 |
| 7 | Certificate-changed error makes "Pair again" the primary button while the text says it may be unsafe; user can't judge "if you reinstalled or renamed it" | high | 14-error-cert |
| 8 | Unreachable error asks user to check router "forwards UDP port 443"; no plain "ask whoever set it up" | medium | 13-error-unreachable |
| 9 | Phone shown as "pixel" but "Your Tailscale machines see this phone as homebox"; screen title is the server name | medium | 10-home-connected (all home screens) |
| 10 | Own-device row with copy icon sits above Machines; easy to copy 10.77.0.2 instead of the NAS | low | 10-home-connected, 18-machines-search-copied |
| 11 | Save-data choice buried on a long page among graphs, "Use for tailbond" switches, Role, data limit; user likely flips the wrong control | medium | 20-connections, 30-settings |
| 12 | After "Switch back to Tailscale" it's unclear whether Tailscale is actually on or the user must flip it; plain switch-off gives no reminder | medium | 12b-home-switched-back-to-tailscale, 10-home-connected-full |
| 13 | "Switch back to Tailscale" only below the full machine list | low | 10-home-connected-full |

Counts: high 3, medium 7, low 3.
