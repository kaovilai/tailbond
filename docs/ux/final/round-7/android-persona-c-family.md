# Round 7 — Android persona C (family member)

**Who I am:** a family member who isn't technical. Someone else set up the home server and put Tailscale on my phone. I use the phone for messages, photos and video. I have never installed an app from outside the Play Store.

**What I had:** `task-script.md`, `screens.md`, and every PNG in `web/` and `android/` (light and dark). I did not open anything else.

Overall: I could finish every task. The phone page and the app explain things in plain words much more often than I expected. The places where I'd slow down are installing (not knowing where the file went), not knowing why the page jumped back, and being told to "keep this screen open" when I need to leave the app to text the person who runs homebox.

---

## Task 1: Get the tailbond app onto the phone

**Where I tap:** With Tailscale on, I type `homebox` in Chrome and get **web-00a** "Get tailbond on this phone". I tap **"Download tailbond for Android (27 MB)"**. Then I follow the four drawn boxes (web-00): ① "File might be harmful… **Download anyway**", then Open; ② "For your security… **Settings**"; ③ turn on "**Allow from this source**", then tap back (←); ④ "**Install**". After that I tap Done and come back to the page.

**What went well:** The page shows me the scary Android messages *before* I see them, and it says "That's expected." Without that I would have stopped at "File might be harmful". The line "Stuck, or not comfortable with this? Ask whoever runs homebox to install it for you." gives me a way out without feeling stupid.

**Where I'd get stuck:**
- Step ① says "Tap Download anyway, then Open." and the top says "Then open the downloaded file." On my phone the Open message from Chrome disappears after a few seconds. If I miss it, nothing on the page tells me where "the downloaded file" is (the Downloads or Files app). I don't know where downloads go, because Play Store apps just install themselves. **Medium.**
- Step ③ says "then tap back (←)". My phone uses swipe gestures and has no back button at the bottom. I'd look for an arrow at the top of Settings, and I might not find one. **Low.**
- At the end, Android shows **Done** and **Open**. The page says "tap Done and come back to this page for step 2", but "Open" is the natural tap. If I tap Open, I get the app's welcome screen (01-onboarding) and then 02-pairing, which asks me to type the server name. I can recover, because I know it's "homebox" and the "Or use the home server's page" box tells me to go back to the browser. Still, it's a detour. **Low.**

**Expected (Play Store / Android):** "Install" and it's simply on my home screen. I knew this would be harder, and the page does most of what it can.

## Task 2: Add the phone to homebox (pair), from the phone

**Where I tap:** Back on the page, step 2: **"Open tailbond and ask homebox"**. The app opens straight to 02a: "Asking homebox… Looking for homebox on your Tailscale network." A few seconds later (02b) it says "Waiting for approval on homebox…". Once it's approved I get 02e "Add this phone to homebox?" and tap **Add phone**.

**What went well:** I never had to type or paste anything. 02b names my phone ("tap Approve next to "pixel-8"") so I can tell the person exactly what to look for. The two failure screens are plain: 02c "Can't find homebox — Check that the Tailscale app says Connected…" and 02d "homebox said no — Whoever runs homebox declined this phone. Ask them, then ask again."

**Where I'd get stuck:**
- If I tap step 2 before the app is really installed (for example, I downloaded the file but never finished installing), web-00b just puts me back at "1 Install the app" with an orange outline. Nothing I can see says *why*, for example "tailbond isn't installed yet". I'd think the button was broken or that I'd done something wrong. **Medium.**
- 02b tells me to "Ask whoever runs homebox…" and then "**Keep this screen open**; tailbond continues by itself once approved." To ask them I have to leave and open Messages or WhatsApp. Does leaving break it? Do I have to start over? The screen asks me to do two things that seem to conflict. I'd probably go back and forth nervously, or ask again for no reason. **Medium.**
- 02b's instruction for the other person ("http://homebox:8077 on a computer, or a Flint 3's admin panel → VPN → tailbond") means nothing to me. I don't know what a "Flint 3" is. I can only forward the text. **Low.**
- On 02a/02b, the second box says "Came here from your home server's page? Go back to it in your browser and tap step 2, Open tailbond and ask." I just did that, and the spinner is already running. That makes me think it didn't work. Its button "Open home server page" is greyed out and nothing says why. **Low.**
- 02e "Add this phone to homebox?" says "It connects over the internet to home.example.net:443, and your internet traffic goes out from home (you can change that in Settings)." I don't know if "traffic goes out from home" is good or bad, or whether I should change it. The lock line "Secure connection to homebox" reassures me, so I'd tap Add phone. **Low.**

**Expected (Tailscale app):** In Tailscale I signed in once and I was in. Waiting for someone else to approve me is new, but the app explains it.

## Task 3: Connect

**Where I tap:** On the main screen (12-home-disconnected), "Disconnected — Stopped. Turn on to reach homebox and your Tailscale machines…", I tap the **switch**. I get 04 "Switch from Tailscale to tailbond?", which says "You still reach your Tailscale machines, like your NAS, through homebox." and "Next, Android asks you to allow tailbond's connection request. Tap OK." I tap **Turn on tailbond**, then OK on Android's prompt. Then 11 "Connecting… Reaching homebox…", then 10 "**Connected** — Through homebox · Wi-Fi + mobile data".

**What went well:** The warning that Tailscale goes off, and that my NAS still works, comes *before* Android's scary VPN prompt. It also tells me which button to press. 05 says the same for the first time, when no other VPN is on.

**Where I'd get stuck:**
- Words like "Still on your tailnet… by name or 100.x address" (01-onboarding) and "This phone reaches your tailnet through homebox" (home screen) use words I don't know ("tailnet", "100.x"). I can ignore them, but they make the app feel like it's for someone else. **Low.**
- If it fails with 13, the message reads "…the home router may need to forward UDP port 443 to it: if someone else set up your home server, show them this message." I can't act on "UDP port 443" myself, but at least it tells me to show it to someone. The same screen also says "Will use Wi-Fi and mobile data together" with both "Ready" right under the red error, which looks like things are fine. **Low.**

**Expected (Tailscale app):** One switch and Android's OK. That's what happens here, with one extra explanation box.

## Task 4: Find and copy the NAS's MagicDNS name or Tailscale IP

**Where I tap:** On the home screen, I type `nas` in **"Search machines"**. One row is left: "nas / nas.tail1234.ts.net / 100.101.2.2 · linux". I tap the copy icon and get the toast "**Copied 100.101.2.2**" (18). If I tap the row instead, I get 19 "nas", which says "Tap an address to copy it" and has labelled rows "MagicDNS name nas.tail1234.ts.net", "Tailscale IPv4 100.101.2.2", and so on. Tapping one gives "Copied nas.tail1234.ts.net".

**What went well:** It works like the Tailscale app's machine list, and the detail page uses the same labels ("MagicDNS name", "Tailscale IPv4").

**Where I'd get stuck:**
- The list row shows the name on top, but its one copy icon copies the **IP number**, not the name. Only the toast tells me which one I got. If someone asked me for "the name", I'd paste a number. **Low.**
- My own phone under "This device" is "10.77.0.2". Every other machine is "100.something", and the welcome screen talked about "100.x" addresses. If someone asked "what's your phone's Tailscale address?", I wouldn't know which number is right. **Low.**

## Task 5: Tell whether Wi-Fi and 5G are both being used right now

**Where I look:** On the home screen (10), under Connections: "**Using both Wi-Fi and mobile data**". The Wi-Fi tile says "● In use ↓38.4 ↑4.1 Mbps" and the Mobile data tile says "● In use ↓22.7 ↑2.9 Mbps". The top banner says "Through homebox · Wi-Fi + mobile data". Tapping Connections (20) shows the same with graphs.

**What went well:** It's clear without tapping anything. When one link breaks (15), it plainly says "Using Wi-Fi only" and "Mobile data — Not working right now" with a red dot. No problems found.

## Task 6: Make it save mobile data

**Where I tap:** On the home screen, the **"Save mobile data — Use mobile data only when Wi-Fi stops working"** switch. Afterwards (16) it says "Using Wi-Fi · saving mobile data (backup only)" and "Mobile data — Not in use · saving data". When Wi-Fi drops (17), it says "Wi-Fi isn't working · using mobile data" and "In use · taking over", which is exactly what I'd want to see.

**Where I'd get stuck:**
- If I tap "More options" first (20/21), there are several things that all sound like saving data: a "Use for tailbond" switch on the Mobile data card, the "Save mobile data" choice in a list of four, "Faster switch to backup (experimental)… Uses a little mobile data", and "Mobile data limit 1.23 GB of 5.00 GB". Turning off "Use for tailbond" under Mobile data looks like the most direct "stop using my data", and I suspect it would also take away the backup. The home-screen switch is easy enough that I'd most likely just use that. **Low.**

## Task 7: Hotel Wi-Fi with a sign-in page

**Where I tap:** The home screen (16c) shows a card "**Wi-Fi needs sign-in** — This Wi-Fi shows a sign-in page before it lets you online. tailbond keeps you connected over mobile data meanwhile, and adds Wi-Fi by itself once you're signed in." I tap **"Sign in to Wi-Fi"**, sign in on the hotel page, and come back. The Wi-Fi tile says "Needs sign-in · not used yet". I'd expect it to change back to "In use" like on 10, but there's no screenshot of that moment.

**What went well:** It tells me not to turn tailbond off ("so you don't need to turn tailbond off"). That is exactly what I would have done otherwise, because with other VPNs the hotel page never shows up.

**Where I'd get stuck:**
- The card gives two instructions. The small text says "Tap the Wi-Fi network (it says "Sign in to network") to open the page." Right below it is a big button "Sign in to Wi-Fi". Tap the network where, in Android's Wi-Fi list or here? I'd just press the button, but the small text made me wonder whether the button was enough. **Low.**

## Task 8: Turn tailbond off and go back to the Tailscale app

**Where I tap:** On the home screen (10), **"Switch back to Tailscale"**. According to Settings (30), this "Turns tailbond off and opens the Tailscale app. Turn Tailscale on there if it says Disconnected". Back in tailbond (12b), it says "Disconnected — Stopped. Tailscale is in charge on this phone." with an **Open Tailscale** button.

**What went well:** The button is right under the main switch on every home-screen state, including the error screens 13 and 14. The help text says plainly that I may need to turn on Tailscale's own switch.

**Where I'd get stuck:**
- My first instinct for "turn tailbond off" is the big switch. If I flip only that, I get 12: "Stopped. Turn on to reach homebox and your Tailscale machines, or switch back to Tailscale below." Neither app is now connecting me. That's only obvious if I read the sentence to the end. **Low.**
- 12b's heading "**Tailscale is in charge on this phone**" sounds like Tailscale is already working. The real instruction ("turn on its switch if it says Disconnected") is in smaller text below it. Someone who reads only the heading may think they're done and later wonder why the NAS doesn't open. **Low.**

**Expected (Android):** Turning one VPN off doesn't turn the other back on; you have to open Tailscale and tap Connect. The app tells me this, so the extra step is explained.

---

## Findings

| # | Task | Finding | Severity |
|---|---|---|---|
| 1 | 1 Install | "Tap Download anyway, then Open" / "Then open the downloaded file": if Chrome's Open message disappears, nothing says where the downloaded file is (Downloads/Files) | medium |
| 2 | 1 Install | Step ③ "then tap back (←)": phones that use swipe gestures have no back button | low |
| 3 | 1 Install | Android offers Done and Open after install; tapping Open (the natural tap) goes to the welcome screen and "type the name" pairing instead of the automatic ask. Recoverable | low |
| 4 | 2 Pair | web-00b: tapping step 2 before the app is installed silently jumps back to "1 Install the app" with an orange outline and no visible reason ("not installed yet") | medium |
| 5 | 2 Pair | 02b "Ask whoever runs homebox…" but also "Keep this screen open": I have to leave the app to text them, and it doesn't say whether leaving is safe | medium |
| 6 | 2 Pair | 02b "http://homebox:8077 on a computer, or a Flint 3's admin panel → VPN → tailbond": jargon I can only forward | low |
| 7 | 2 Pair | While already asking, the second box still says "Came here from your home server's page? Go back to it… and tap step 2", and its "Open home server page" button is greyed out with no reason | low |
| 8 | 2 Pair | 02e "home.example.net:443… your internet traffic goes out from home (you can change that in Settings)": not explained whether this is fine or something to change | low |
| 9 | 3 Connect | "Still on your tailnet… by name or 100.x address", "reaches your tailnet through homebox": unfamiliar words | low |
| 10 | 3 Connect | 13 error "forward UDP port 443" is beyond me (it does say to show someone), and "Will use Wi-Fi and mobile data together / Ready" under the red error sends mixed signals | low |
| 11 | 4 NAS | List row shows the name on top, but its copy icon copies the IP ("Copied 100.101.2.2"); only the toast says which | low |
| 12 | 4 NAS | My phone shows "10.77.0.2" while every other machine and the welcome screen say "100.x" | low |
| 13 | 6 Save data | Connections page has overlapping data controls ("Use for tailbond" on Mobile data, mode list, "Mobile data limit", "Faster switch to backup"); the wrong one is easy to pick | low |
| 14 | 7 Hotel Wi-Fi | Two instructions: small "Tap the Wi-Fi network (it says "Sign in to network")" vs the big "Sign in to Wi-Fi" button | low |
| 15 | 8 Back to Tailscale | Flipping the main switch off leaves neither app connecting; only the end of "…or switch back to Tailscale below" says so | low |
| 16 | 8 Back to Tailscale | 12b heading "Tailscale is in charge on this phone" reads as done, though Tailscale may still say Disconnected (the real step is in smaller text) | low |

Totals: 0 high / 3 medium / 13 low
