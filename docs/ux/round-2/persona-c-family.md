# Round 2: persona (c), less-technical family member

Who I am: I'm in my 60s and a retired teacher. My son set up Tailscale on my phone. I open it to flip the big
switch on, and sometimes I look up "nas" and copy its address for the photo app.
Red error text makes me nervous, and I won't touch settings I can't explain.

Screens reviewed: every PNG in `out/ux/after-2/` (light and dark).

---

## Task 1: Install tailbond and pair it with the home server

**Tap path**
1. Open the home server's page in the browser (web-01-add-this-phone). Tap **"Download tailbond for Android"**, then
   open the downloaded file to install it.
2. Back on the same page, keep **"pixel-8"** as the name and tap **"+ Add this phone"** (web-02-after-tap).
   The page says "Opening tailbond··· In the app, tap Add phone, then turn tailbond on."
3. tailbond opens with "Add this phone to homebox?" (03-deeplink-confirm). Tap **"Add phone"**.

**Stuck points**
- **I can't get to the starting page by myself.** Everything starts on the server's web page, but nothing I can see
  tells me its address. If I open the app first, I see 01-onboarding ("You'll need a home server with tailbond
  installed.") and then 02-pairing, which asks for "Home server's Tailscale name". The hint says "It's the machine that
  runs tailbond, listed in the Tailscale app's Machines. Not sure which one? Ask whoever set it up". I don't know
  which machine "runs tailbond", so I would have to phone my son. **High**: I can't do this task alone.
- **The install step scares me.** "Open the downloaded file to install it. Android may ask you to allow installs from
  your browser." Android's own warning about unknown apps would make me stop. My son taught me apps come from the
  Play Store. **Medium.**
- **Going back and forth between the browser and the app.** The web page never says "after installing, come back to
  this page". When the install finishes, Android offers "Open". That takes me to the app's 01-onboarding and then
  02-pairing, where I'm asked for a name I don't know. I'd forget the browser tab is where step 2 lives.
  **Medium.**
- **The pasted link is wrong.** In 02-pairing-paste-error, the red text says "That isn't a complete tailbond link. Copy the whole
  link starting with tailbond://pair", but the box looks empty, so I can't see what I pasted. The red makes me
  think I broke something. **Low.**
- In the confirm dialog, "It connects over the internet to home.example.net:443." and "Technical details" mean
  nothing to me. "Secure connection to homebox" with the lock reassures me. **Low.**
- Below the "Copy link" button, web-02 says "Keep this link to yourself: it lets a phone use your home server."
  It sounds like a warning, and I'm not sure if I've done something risky. **Low.**

**Expected (Tailscale-like):** My son installed Tailscale from the Play Store, and I signed in by tapping one button.
I expected to get the app from the Play Store and then sign in or "join" with one button, without needing to know
a server's name or web address.

---

## Task 2: Connect

**Tap path:** On the home screen (12-home-disconnected), tap the big switch. The "Turn off Tailscale's VPN?" dialog
(04) or "Allow tailbond's VPN" (05) appears. Tap **"Turn on tailbond"**, then tap OK on Android's dialog.
The screen shows "Connected" (10-home-connected).

**Stuck points**
- The title "Turn off Tailscale's VPN?" stops me, because my son said Tailscale is what reaches the NAS. The body
  explains "You still reach your Tailscale machines, like your NAS, through homebox", and "Tap OK" tells me what's
  coming next, so I'd go ahead. **Low.**
- The "This device" block says "10.77.0.2 … This phone isn't a Tailscale machine itself. Its traffic reaches your
  tailnet from homebox (100.101.1.1)." It's numbers and a word I don't know ("tailnet"), right under my status, and
  "isn't a Tailscale machine" sounds like something is wrong. **Low.**
- If it fails, 13-error-unreachable shows a red box: "…the home router may need to forward UDP port 443 to it: if someone else set
  up your home server, show them this message." I'm glad it says to show someone. But there is no button to send or
  copy the message to my son, so I'd have to take a screenshot. **Low.**
- If it fails, 14-error-cert says "Can't confirm this is your home server … tell whoever set it up". **"Pair again"** sits
  next to "Try again" as an equal choice. I'd tap one of them without knowing whether that's safe. **Medium.**

**Expected:** Tailscale has one switch and one word ("Connected"), without numbers on the main view.

---

## Task 3: Find and copy the NAS's name or address

**Tap path:** Scroll down the home screen to "Machines", type "nas" in **"Search machines"**, and tap the copy icon on the nas row.
The message "Copied 100.101.2.2" appears (18-machines-search-copied). Or tap the row to open the nas page (19-machine-detail): "Tap an address to copy it", then
"MagicDNS name nas.tail1234.ts.net" with a copy icon.

**Stuck points**
- **The machines are below the fold.** On the first screen (10-home-connected) I only see "Machines" at the very
  bottom. To reach it I have to scroll past "Connections", "Mode: Fastest / Change", "Save mobile data", "Internet
  traffic" and "This device", all things I don't understand and don't want to touch by accident. In Tailscale the
  machine list is right under the switch. **Medium.**
- The row shows the name "nas.tail1234.ts.net" in large text and the number below it. But the copy icon copied the
  number ("Copied 100.101.2.2"). I'd only know from the message, which goes away. **Low.**

**Expected:** Like Tailscale: the list is right there, I tap nas, and I copy the address.

---

## Task 4: Are Wi-Fi and 5G both being used?

**Tap path:** Read the home screen: "Using both Wi-Fi and mobile data", with Wi-Fi "In use" and Mobile data "In use", both with green dots
(10-home-connected).

This is clear, and I could answer without tapping. When one connection breaks (15-one-path-failed), I see a red dot and "Not
working right now" on Mobile data while the top still says "Connected". The red worries me a little, but I can see that things
still work. **Low.**

---

## Task 5: Make it save cellular data

**Tap path:** On the home screen, flip **"Save mobile data: Use mobile data only when Wi-Fi stops working"**. The line changes to "Using Wi-Fi ·
saving mobile data (backup only)" and "Mode: Save mobile data" (16-save-data-standby).

**Stuck points**
- The home screen has two controls for the same thing: the "Mode: Fastest / Change >" link and the "Save mobile
  data" switch. I'd wonder whether I need to do both. After flipping the switch the Mode line changes too, so it
  works out. **Low.**
- If I tap "Change" instead, the Connections page (20/21) has four modes, "Role: Main / Extra" drop-downs, "Faster
  switch to backup (experimental)", a GB limit box and "Reset this month's count". That's far more than I can
  explain, so I'd back out without touching anything. It isn't a blocker, because the home switch is enough. **Low.**

---

## Task 6: Turn tailbond off and go back to the Tailscale app

**Tap path:** On the home screen, tap **"Switch back to Tailscale"**. The screen shows "Tailscale is in charge on this phone … If Tailscale isn't
connected yet, open it and turn on its switch." Tap **"Open Tailscale"** and flip its switch (12b-home-switched-back-to-tailscale).

**Stuck points**
- **The big switch is what I'd actually use.** I'm used to turning things off with the big switch. If I flip it off,
  12-home-disconnected only says "Stopped. Turn on to reach homebox and your Tailscale machines." It doesn't say
  that Tailscale is still off too. I'd think I was "back to normal", but my photo app couldn't reach the NAS. Only
  the small blue "Switch back to Tailscale" text gets me back properly. **Medium.**
- "Switch back to Tailscale" looks like a link, not a button, so it's easy to miss. **Low.**

**Expected:** Like Tailscale, with one switch. Turning tailbond off would either bring Tailscale back or plainly say "Tailscale is
still off, open it to turn it back on".

---

## Other things I noticed

- Settings (30-settings) has "Remove from this phone" as a plain link right next to "Pair again", and no screen shows
  whether it asks before removing. I'd be afraid of tapping it by accident. **Low.**
- Settings has an "Always-on VPN" paragraph ("Leave "Block connections without VPN" off…"). I don't understand it and
  wouldn't touch it, but it adds to my worry. **Low.**

---

## Summary

| # | issue | severity | screen |
|---|---|---|---|
| 1 | Can't start alone: the home server's web page address is shown nowhere, and the app asks for a "Home server's Tailscale name" I don't know | high | 01-onboarding, 02-pairing |
| 2 | Installing from the browser ("allow installs from your browser") triggers Android's unknown-apps warning; it feels unsafe compared with the Play Store | medium | web-01-add-this-phone |
| 3 | Nothing tells me to return to the browser after installing; opening the app lands on a pairing screen asking for a name I don't know | medium | web-01-add-this-phone, 02-pairing |
| 4 | The cert error offers "Pair again" next to "Try again" with no guidance on which is safe for me | medium | 14-error-cert |
| 5 | The machines list is below the fold, under settings I don't understand; in Tailscale it's right under the switch | medium | 10-home-connected |
| 6 | Flipping the big switch off leaves both VPNs off, and the screen doesn't say Tailscale is still off, so the NAS is silently unreachable | medium | 12-home-disconnected |
| 7 | The paste-error box looks empty, so I can't see what I pasted, and the red text alarms me | low | 02-pairing-paste-error |
| 8 | Jargon in the confirm dialog ("home.example.net:443", "Technical details") | low | 03-deeplink-confirm |
| 9 | "Keep this link to yourself…" reads like a security warning | low | web-02-after-tap |
| 10 | "This device" shows IPs, "tailnet" and "isn't a Tailscale machine itself", which sounds like an error | low | 10-home-connected |
| 11 | The unreachable error says "show them this message" but has no share/copy button | low | 13-error-unreachable |
| 12 | The row shows the name, but its copy icon copies the IP; only the passing message says so | low | 18-machines-search-copied |
| 13 | Two controls for the same setting ("Mode / Change" and the "Save mobile data" switch); the Change page is overwhelming | low | 10-home-connected, 20-connections |
| 14 | "Switch back to Tailscale" is a small text link, easy to miss | low | 10-home-connected |
| 15 | "Remove from this phone" is a plain link next to "Pair again"; no screen shows whether it asks for confirmation | low | 30-settings |
| 16 | The Always-on VPN paragraph is unexplained jargon on the Settings page | low | 30-settings |
