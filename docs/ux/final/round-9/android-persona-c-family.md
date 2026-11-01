# Round 9 — Android persona C (family member)

**Who I am:** someone else set up our home server (homebox) and put the Tailscale app on my phone. I use my phone for messages, photos and video. I have never installed an app from outside the Play Store. All I was told was: "with Tailscale on, type **homebox/** into your phone's browser."

**What I had:** only `task-script.md`, `screens.md` and the screenshots in `web/` and `android/` (round-9 packet). I looked at every screenshot, including the dark-theme ones. Android's own screens (install prompts, VPN permission, Wi-Fi sign-in page, the Tailscale app) are not in the packet, so I don't judge them.

**Overall:** I could finish all eight tasks from what's on screen. Nothing would have stopped me. The two places I'd hesitate are installing an app from outside the Play Store (the guide is good, but I have to remember it while Android's screens are in front of it) and one question in the "switch from Tailscale" dialog that I can't answer. Everything else is wording or small extra effort.

---

## Task 1: Get the tailbond app onto the phone

**Where I tap:** I type `homebox/` in Chrome and get `web-00a-phone-page-first-screen`: "Get tailbond on this phone". Then I tap the big green **"Download tailbond for Android (27 MB)"** button. The page then walks me through what Android will show: ① "File might be harmful … Download anyway", ② "Tap Settings", ③ "Turn on Allow from this source, then go back (← or swipe from the edge of the screen)", ④ "Tap Install", ⑤ "Sometimes Google Play Protect asks to check the app first. Tap Scan app…". Each step has a drawing with the button to press circled in orange. Then: "When it says App installed, tap Done and come back to this page (Chrome, this tab) for step 2."

**What works for me:** this is the page I needed. It tells me before I see it that Android will warn me, and that "That's Android's normal check for every app from outside the Play Store, and you only go through it once." Without that I'd have tapped Cancel at "File might be harmful". It also says "the pictures below are drawings, not buttons", so I won't tap the drawings. It covers the case where I miss the Open message: "Chrome's Open message gone before you tapped it? … Files app → Downloads. Tap tailbond.apk". It also gives me a way out: "Stuck, or not comfortable with this? Ask whoever runs homebox to install it for you." If I tap step 2 too early, `web-00b` says plainly: "tailbond isn't on this phone yet, so step 2 brought you back here. Install it first, then tap step 2 again."

**Where I'd get stuck:**
- Steps ② to ⑤ happen on screens that cover the page: a Settings screen and the installer. While I'm there I can't see the guide. I have to remember five steps from one read, or go back and forth. Steps ③ to ⑤ aren't even on the first screen (`web-00a` stops partway through ②). Nothing at the top says "read all the steps before you tap Download". I expected something like the Play Store, where one tap installs. Instead it's five prompts, one of them in Settings. **Medium.**
- The grey "Is this safe?" paragraph has "This page starts with http://" and "SHA-256 5416 7117c3586c8ff05cc983c6e41381d9156…", plus "sha256sum /usr/share/tailbond/tailbond.apk". The first sentence reassures me. The code and the hash make me wonder if I'm meant to check something. **Low.**
- "You can turn Allow from this source off again afterwards" doesn't say how or where. I wouldn't do it. **Low.**
- The page says "tap Done and come back", but Android's own "App installed" message also has an Open button, and I'd likely tap Open. That puts me on the app's first screen (`01-onboarding`). The hint to go back is there ("Installed tailbond from your home server's page in Chrome? Easier: go back to Chrome and tap step 2 there; it fills everything in."), but it's under the big **Get started** button, which I'd tap first. **Low.**

## Task 2: Add the phone to homebox (pair)

**Where I tap:** back in Chrome, step 2's **"Open tailbond and ask homebox"**. The app opens on "Asking homebox… Looking for homebox on your Tailscale network." (`02a`). Then it shows "Waiting for approval on homebox…" (`02b`). It says I can "switch to another app to message them; tailbond keeps waiting for 10 minutes and carries on by itself once approved". So I'd message whoever runs homebox. After they approve I get "Add this phone to homebox?" (`02e`/`03`). I tap **Add phone**, which is what the web page told me to do ("tailbond shows what it will do: tap Add phone").

**What works for me:** I don't type anything. "Keep Tailscale on until then" is on both the web page and the app. The two failure screens are plain: "Can't find homebox — Check that the Tailscale app says Connected…" (`02c`) and "homebox said no — Whoever runs homebox declined this phone. Ask them, then ask again." (`02d`). If I opened the app from the launcher instead, `02-pairing` repeats "Came here from your home server's page? Go back to it in your browser and tap step 2", and I can also just type `homebox` in the box.

**Where I'd get stuck:**
- The waiting text is long and partly for the other person: "approve "pixel-8" on homebox's tailbond page (it's at the top of http://homebox:8077). There it shows as pixel-8 · sam@example.com · 100.101.8.8". I'd copy it all into a message and hope it's right. It also doesn't say what happens after "10 minutes". Do I start again? **Low.**
- The confirmation says "It connects over the internet to home.example.net:443, and your internet traffic goes out from home (you can change that in Settings)." I don't know what ":443" is. "Your internet traffic goes out from home" makes me wonder if someone at home can see my browsing, and the dialog doesn't tell me whether that matters. **Low.**
- On the first app screen (`01-onboarding`), "Still on your tailnet" and "by name or 100.x address" are words I don't use. **Low.**

(Note: `02-pairing-paste-error` ends just below the red "Pairing link" box, so I couldn't read the error. I wouldn't use that path anyway.)

## Task 3: Connect

**Where I tap:** the switch next to "Disconnected" on the main screen (`12-home-disconnected`). Because Tailscale is on, I get "Switch from Tailscale to tailbond?" (`04`). I tap **Turn on tailbond**, then OK on Android's permission prompt (not shown; the dialog says "Next, Android asks you to allow tailbond's connection request. Tap OK."). I see "Connecting… Reaching homebox…" (`11`), then "Connected — Through homebox · Wi-Fi + mobile data" (`10`).

**What works for me:** the dialog tells me before it happens that Tailscale will go off, that I still reach the NAS, and how to go back ("tap Switch back to Tailscale on tailbond's main screen"). The web page said the same thing.

**Where I'd get stuck:**
- "Set Tailscale as Always-on VPN? Turn that off first (Android Settings → VPN → Tailscale's gear), or Android keeps switching back to it." I didn't set the phone up, so I have no idea whether someone did that. Nothing in the app tells me whether it's on. If it is, I'd apparently see tailbond keep turning off and think it's broken. The dialog asks a question I can't answer and gives me a Settings path to hunt for. **Medium.**
- "Your internet traffic goes out from homebox too (like an exit node; Settings changes that)." "Exit node" means nothing to me. **Low.**
- The main screen's big title is **"homebox"** with "via home server · tail1234.ts.net" under it. The word "tailbond" is nowhere on it. With two similar-looking apps (Tailscale and this one), I'd sometimes wonder which one I'm in. **Low.**

## Task 4: Find and copy the NAS's MagicDNS name or Tailscale IP

**Where I tap:** I scroll the main screen down to **Machines**, type `nas` into "Search machines" and tap the copy icon on the nas row. The message "Copied 100.101.2.2" appears (`18`). Or I tap the nas row and get the "nas" page (`19`), which says "Tap an address to copy it". I tap the copy icon next to "MagicDNS name nas.tail1234.ts.net" and see "Copied nas.tail1234.ts.net".

**What works for me:** search, a green dot for "Online", and the "Copied …" message that tells me exactly what I got. The detail page labels each address.

**Where I'd get stuck:**
- The Machines list starts below the first screen. Above it are the connection cards, Save mobile data, and a "This device" block. In the Tailscale app the machines are the first thing I see, so at first I'd think the list isn't there. **Low.**
- The copy icon on a list row copies the number (100.101.2.2), but the big bold text on that row is the name "nas". I'd expect to get the name. The "Copied 100.101.2.2" message saves me. **Low.**
- The "This device" block reads: "10.77.0.2 — Not a Tailscale address: to your Tailscale machines this phone's traffic comes from homebox, under its access rules…" It sits right above Machines on every main screen and makes me worry I copied the wrong kind of address. **Low.**

## Task 5: Are Wi-Fi and 5G both being used right now?

**Where I look:** the main screen (`10-home-connected`). The title says "Through homebox · Wi-Fi + mobile data". Under **Connections** it says **"Using both Wi-Fi and mobile data"**, and both the Wi-Fi and Mobile data cards show a green dot and "In use". Tapping "Connections" opens `20-connections`, which also says "Using both Wi-Fi and mobile data". Answer: yes, both.

**What works for me:** I get the answer in one sentence, with green dots.

**Where I'd get stuck:**
- When one connection is broken (`15-one-path-failed`), the screen says "Using Wi-Fi only" and the Mobile data card says "Not working right now" with a red dot. It doesn't say why or whether I should do anything, for example whether my mobile data is switched off. **Low.**

## Task 6: Make it save mobile data

**Where I tap:** the **"Save mobile data — Use mobile data only when Wi-Fi stops working"** switch on the main screen. Afterwards (`16`) the screen says "Using Wi-Fi · saving mobile data (backup only)", the Mobile data card says "Not in use · saving data", and the line under the cards says "Mode: Save mobile data". When Wi-Fi drops (`17`) it says "Wi-Fi isn't working · using mobile data" and "In use · taking over".

**What works for me:** one switch on the main screen, and it says what it does in my words.

**Where I'd get stuck:**
- If I tap "More options" instead, the Connections page (`20`/`21`) has three things that all sound like saving data. There's a "Use for tailbond" switch on the Mobile data card ("Turning this off doesn't turn off the phone's mobile data"), the "Save mobile data" choice in a list of four, and a "Mobile data limit" box that already says 5. After I pick Save mobile data, a fourth appears: "Faster switch to backup (experimental) … Uses a little mobile data". The page does say "The Save mobile data switch on the home screen picks 'Save mobile data' here", which helps, but I'd still worry I need to change all of them. **Low.**

## Task 7: Hotel Wi-Fi with a sign-in page

**Where I tap:** the main screen (`16c`) shows a card at the top, "Wi-Fi needs sign-in — This Wi-Fi shows a sign-in page before it lets you online. tailbond keeps you connected over mobile data meanwhile, and adds Wi-Fi by itself once you're signed in." It has a button, **Sign in to Wi-Fi**, which I tap. The Wi-Fi card says "Needs sign-in · not used yet", and the title says "Through homebox · mobile data", so I know I'm still online.

**What works for me:** it tells me what's going on, that I'm not cut off, that I don't need to turn tailbond off, and that Wi-Fi will join "by itself" afterwards. That last part covers "back into the bond".

**Where I'd get stuck:**
- The small text and the button don't quite agree. The small text says "Tap the Wi-Fi network (it says "Sign in to network") to open the page." The button says "Sign in to Wi-Fi". Does the button open the hotel page, or a Wi-Fi list where I then tap the network? The instruction comes before the button, so it reads like a second thing I have to do somewhere else. I'd tap the button and probably manage. **Low.**

## Task 8: Turn tailbond off and go back to the Tailscale app

**Where I tap:** **Switch back to Tailscale**, which is on the main screen in every state. I get `12b`: "Disconnected — Stopped. Back to Tailscale: turn it on in its app." There's a card, "Last step: turn Tailscale on — Open the Tailscale app and check that it says Connected; if it says Disconnected, tap its switch. Until then, your Tailscale machines aren't reachable.", with an **Open Tailscale** button. I tap it and turn Tailscale on.

**What works for me:** the button is always in the same place, its name says exactly what I want, and the next screen tells me the one thing Android won't do for me (turn Tailscale back on). If I just flip the big switch off instead, the screen says "or switch back to Tailscale below", which points me back to the button.

**Where I'd get stuck:**
- Settings describes the same option as "Turns tailbond off and opens the Tailscale app." But `12b` shows me still in tailbond, with its own "Open Tailscale" button. I'm not sure whether Tailscale should have opened by itself, or whether something went wrong. **Low.**
- In Settings (`30`), if I go there looking for "off", a red **"Remove from this phone"** sits right next to "Pair again", with no explanation of what either does. I could tap the red one by mistake and lose the setup someone else did for me. **Low.**

---

## Findings

| # | Task | Finding | Severity |
|---|---|---|---|
| 1 | 1 Install | Steps ②–⑤ ("Tap Settings", "Turn on Allow from this source, then go back", "Tap Install", "Tap Scan app") happen on screens that cover the guide. The first screen shows only ① and part of ②, and nothing says to read all the steps before tapping Download. I have to memorize a five-prompt flow I've never done. | medium |
| 2 | 1 Install | "Is this safe?" paragraph shows "http://", a SHA-256 hash and "sha256sum /usr/share/tailbond/tailbond.apk". This is technical noise that makes me think I need to check something. | low |
| 3 | 1 Install | "You can turn Allow from this source off again afterwards" doesn't say how. | low |
| 4 | 1 Install | The page says "tap Done and come back", but I'd likely tap Android's Open. The app's hint "go back to Chrome and tap step 2 there" is below the big "Get started" button. | low |
| 5 | 2 Pair | The waiting text is dense ("http://homebox:8077", "pixel-8 · sam@example.com · 100.101.8.8"), and after "keeps waiting for 10 minutes" it doesn't say what then. | low |
| 6 | 2 Pair | Confirmation: "connects over the internet to home.example.net:443" and "your internet traffic goes out from home". Jargon, plus an unexplained privacy worry. | low |
| 7 | 2 Pair | Onboarding: "Still on your tailnet", "by name or 100.x address". Jargon. | low |
| 8 | 3 Connect | "Set Tailscale as Always-on VPN? Turn that off first (Android Settings → VPN → Tailscale's gear)" asks something only the person who set up the phone knows. Nothing shows whether it's on, and failure would look like tailbond breaking. | medium |
| 9 | 3 Connect | "(like an exit node; Settings changes that)". "Exit node" is jargon. | low |
| 10 | 3 Connect | The main screen title is "homebox"; the name "tailbond" isn't on it, so it's easy to mix up with the Tailscale app. | low |
| 11 | 4 Copy NAS | The Machines list is below the first screen, under the connection cards and "This device". In Tailscale the machines come first. | low |
| 12 | 4 Copy NAS | The list-row copy icon copies "100.101.2.2" although the row's bold text is the name "nas". The toast clarifies. | low |
| 13 | 4 Copy NAS | "Not a Tailscale address: … under its access rules" above Machines suggests I might copy the wrong kind of address. | low |
| 14 | 5 Both paths | "Mobile data · Not working right now" (red) gives no reason and no next step. | low |
| 15 | 6 Save data | Connections page has "Use for tailbond" switch, mode list, "Mobile data limit" and "Faster switch to backup (experimental)". It's unclear which ones I need. | low |
| 16 | 7 Hotel Wi-Fi | "Tap the Wi-Fi network (it says "Sign in to network")" vs the "Sign in to Wi-Fi" button. It's unclear whether these are one step or two. | low |
| 17 | 8 Back to Tailscale | Settings says Switch back "opens the Tailscale app", but 12b shows tailbond with its own "Open Tailscale" button. I can't tell if it should have opened by itself. | low |
| 18 | 8 Back to Tailscale | A red "Remove from this phone" sits next to "Pair again" in Settings with no explanation. Easy to tap by mistake. | low |

Totals: 0 high / 2 medium / 16 low
