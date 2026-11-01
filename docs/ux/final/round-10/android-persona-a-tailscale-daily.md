# Round 10 — Android persona A (daily Tailscale user)

Persona: uses the official Tailscale Android app every day (MagicDNS names, the machine list, the VPN toggle, exit nodes at a basic level); non-networking job. Today the phone has only the Tailscale app, connected.

Materials used: `task-script.md`, `screens.md`, every PNG in `web/` and `android/` (light and dark). Long screenshots were read in full, top to bottom.

---

## Task 1: Get the tailbond app onto the phone

**Path:** With Tailscale on, I type `homebox/` into Chrome and get **web-00a**: "Get tailbond on this phone", step 1 "Install the app", a teal box "Read the pictures below first (or take a screenshot): once you open the file, Android's own screens cover this page.", then **Download tailbond (27 MB)**. Below that (web-00), five numbered drawings: ① "File might be harmful … Download anyway", ② "Tap Settings", ③ "Turn on Allow from this source, then go back", ④ "Tap Install", ⑤ "Sometimes Google Play Protect asks … Tap Scan app". Then "When it says App installed, tap Done and come back to this page (Chrome, this tab) for step 2."

**What works:** The page sounds like a person talking: "tailbond isn't in the Play Store: it comes from homebox, your own home server, so Android asks you to confirm a few times. That's Android's normal check". The drawings match Android's sideload prompts, and the orange rings show me which button to tap. "Chrome's Open message gone before you tapped it? … Chrome's menu ⋮ → Downloads" covers a slip I really do make. web-00b is a nice touch: if I tap step 2 before installing, the page comes back with "tailbond isn't on this phone yet, so step 2 brought you back here. Install it first, then tap step 2 again."

**Where I'd stumble:**
- The big **Download tailbond** button sits *above* the five drawings I'm told to "Read … first". I'd tap the button straight away (it's the obvious thing), and then "Android's own screens cover this page". I'd probably manage, because Android's dialogs mostly explain themselves, but the instruction and the layout pull in opposite directions, and five steps is a lot to remember. *(low)*
- After installing, the page says "tap Done and come back to this page". Android's "App installed" screen also offers **Open**, and I'd tap Open by reflex. That's recoverable: the app's first screen (01-onboarding) says "Installed tailbond from your home server's page in Chrome? Easier: go back to Chrome and tap step 2 there". Still, it's an extra round trip. Small mismatch too: drawing ① says "Tap Download anyway, then Open" but only shows the Download anyway dialog. *(low)*
- The safety paragraph ("This page starts with http://, but your phone reached it through Tailscale … SHA-256 54167117c3586c8… the same as sha256sum /usr/share/tailbond/tailbond.apk on homebox") is gray small print, so it doesn't get in the way. But the hash and command line mean nothing to me. The first sentence alone would reassure me. *(low)*

**Expected (Tailscale/Android):** Play Store install with one button. Sideloading is new to me, but the page makes it doable.

## Task 2: Add the phone to homebox (pair), from the phone

**Path:** Back in Chrome, step 2: **Open tailbond and ask homebox** ("Keep Tailscale on until then. Whoever runs homebox taps Approve…"). The app opens on **02a** "Asking homebox… Looking for homebox on your Tailscale network.", then **02b** "Waiting for approval on homebox… Ask whoever runs homebox to approve "pixel-8" … There it shows as pixel-8 · sam@example.com · 100.101.8.8 … tailbond keeps waiting for 10 minutes". Once approved, **02e** asks "Add this phone to homebox?" and I tap **Add phone**.

**What works:** No typing. The waiting screen tells me exactly what to tell the owner and that I can switch apps to message them. Failures are plain: 02c "Can't find homebox — Check that the Tailscale app says Connected" and 02d "homebox said no — Whoever runs homebox declined this phone. Ask them, then ask again." The launcher route (01-onboarding → 02-pairing) asks for "Home server's name in Tailscale", which I'd understand from the Tailscale machine list.

**Where I'd stumble:**
- In 02e: "Your Tailscale machines stay reachable through it; **they see this phone as homebox, so homebox's Tailscale access rules apply (not your own account's)**." Then on the home screen my phone's address becomes "10.77.0.2" with "**Not a Tailscale address**: to your Tailscale machines this phone's traffic comes from homebox, under its access rules. They can reach the phone at this address through homebox." But the waiting screen (02b) told the owner my phone is "100.101.8.8". So which one is my phone now? Will my laptop still find "pixel-8" at its 100.x address? Will the NAS still recognise me as me, or as homebox? I don't know what "access rules" are. In the Tailscale app my phone simply is a machine with a 100.x address and my account. Here it's quietly something else, and the explanation uses words I don't have. I'd tap Add phone anyway, but I'd be uneasy, and I'd be surprised later if something that checks "who I am" behaves differently. *(medium)*
- "It connects over the internet to home.example.net:443" in 02e is fine as small print. I'd ignore it.

## Task 3: Connect

**Path:** After Add phone, the home screen is **12-home-disconnected**: "Disconnected — Stopped. Turn on to reach homebox and your Tailscale machines, or switch back to Tailscale below." I tap the big switch and get **04-handoff-other-vpn**: "Switch from Tailscale to tailbond? Android runs one VPN at a time. Turning on tailbond turns off Tailscale…" I tap **Turn on tailbond**, then Android's VPN OK (not shown), then **11** "Connecting… Reaching homebox…", then **10** "Connected — Through homebox · Wi-Fi + mobile data".

**What works:** The switch is right where the Tailscale app's toggle is. The handoff dialog tells me in advance that Tailscale goes off and that Android will ask me to tap OK. The Always-on VPN note ("If tailbond switches itself off right after this…") would save me if it happened.

**Where I'd stumble:**
- On a brand-new setup the card says "Disconnected — **Stopped.**" "Stopped" sounds like something broke. The Tailscale app just says it's off / not connected. *(low)*
- The handoff dialog (04/05) has five paragraphs for one tap. One of them is a real behaviour change I care about: "Your internet traffic goes out from homebox too (like an exit node; Settings changes that)". In Tailscale I never use an exit node by default, so all my browsing going through home is new. It is disclosed (here, in 02e, and on the home screen as "Internet traffic — Through your home server"), but it's buried in a wall of text I'd skim. *(low)*

## Task 4: Find and copy the NAS's MagicDNS name or Tailscale IP

**Path:** On the home screen I scroll to **Machines**, type "nas" into **Search machines** (18) and tap the copy icon. A snackbar says "Copied 100.101.2.2". Or I tap the row and get **19** "nas — Online · linux — Tap an address to copy it", with "MagicDNS name nas.tail1234.ts.net", "Short name nas", "Tailscale IPv4 100.101.2.2", "Tailscale IPv6 fd7a:115c:a1e0::2". I tap one and see "Copied nas.tail1234.ts.net".

**What works:** This is nearly identical to the Tailscale app's machine list and detail: same names, same 100.x addresses, green online dots, "Last seen 45 h ago". Done in seconds. "With tailbond on, apps on this phone can use these addresses just like with Tailscale" is exactly what I wanted to hear.

**Where I'd stumble:**
- The machine list is the fourth section on the main screen, below the status card, Switch back, Connections, Save mobile data and This device. In 10-home-connected it starts at the very bottom of the first screen. In the Tailscale app the machine list *is* the main screen. I had to scroll to find it. *(low)*
- In the list, the row's most prominent line is "nas.tail1234.ts.net", but the copy icon next to it copied "100.101.2.2". If I wanted the name, I'd have pasted the wrong thing. The snackbar does say what was copied, so I'd notice. *(low)*

## Task 5: Are Wi-Fi and 5G both being used right now?

**Path:** Home screen (10): the status card says "Through homebox · Wi-Fi + mobile data". Under Connections it says "Using both Wi-Fi and mobile data", and both tiles show a green dot with "In use" and speeds (↓38.4 ↑4.1 Mbps / ↓22.7 ↑2.9 Mbps). Tapping Connections (20) gives the same headline plus graphs and "delay 21 ms / 48 ms".

**Verdict:** Clear yes, answered three times on the first screen. 15-one-path-failed ("Using Wi-Fi only", red "Not working right now" on Mobile data) and 17 ("Wi-Fi isn't working · using mobile data") are just as clear. No findings.

## Task 6: Make it save mobile data

**Path:** Home screen: the **Save mobile data** switch, "Use mobile data only when Wi-Fi stops working". I flip it and get **16**: "Mode: Save mobile data", "Using Wi-Fi · saving mobile data (backup only)", Mobile data tile "Not in use · saving data", status "Through homebox · Wi-Fi". When Wi-Fi drops (17): "Wi-Fi isn't working · using mobile data", "In use · taking over".

**What works:** One switch, on the main screen, worded the way I'd say it.

**Where I'd stumble:**
- If I go to **More options** / Connections (20/21) to double-check, I find several things that all sound like "use less mobile data": the "Save mobile data" mode radio, a "Use for tailbond" switch on the Mobile data card ("Turning this off doesn't turn off the phone's mobile data"), "Advanced: choose which connection is main or backup", "Faster switch to backup (experimental) … Uses a little mobile data", and a "Mobile data limit" ("Separate from Save mobile data…"). The page does link the home switch to the mode ("The Save mobile data switch on the home screen picks "Save mobile data" here"), but I'd wonder whether I should also turn off "Use for tailbond" on Mobile data. *(low)*

## Task 7: Hotel Wi-Fi with a sign-in page

**Path:** **16c**: status "Connected — Through homebox · mobile data", with a card "Wi-Fi needs sign-in: This Wi-Fi shows a sign-in page before it lets you online. tailbond keeps you connected over mobile data meanwhile, and adds Wi-Fi by itself once you're signed in." Small print: "Tap the Wi-Fi network (it says "Sign in to network") to open the page. Android's sign-in page uses the Wi-Fi directly, so you don't need to turn tailbond off." Button: **Sign in to Wi-Fi**. The Wi-Fi tile says "Needs sign-in · not used yet". Settings (30) also has "Tell me when Wi-Fi needs a sign-in".

**What works:** This is better than what I'm used to. With the Tailscale app, hotel sign-in pages are where I start toggling things. Here it tells me I don't have to turn anything off and that Wi-Fi rejoins by itself.

**Where I'd stumble:**
- There's a big button **Sign in to Wi-Fi**, and right above it the small print says "Tap the Wi-Fi network (it says "Sign in to network")". Do I tap the button, or go find the network in Android's Wi-Fi list or the notification? I'd tap the button and hope. I can't see where it takes me (Android's screens aren't shown), and the small print reads like a separate instruction rather than "the button takes you there; then tap the network". *(low)*
- No screenshot shows the moment after sign-in. I'm trusting "adds Wi-Fi by itself". The Wi-Fi tile would presumably turn green, as in 10. Not a finding, just unverified.

## Task 8: Turn tailbond off and go back to the Tailscale app

**Path:** Every home-screen state has a full-width **Switch back to Tailscale** button directly under the status card. Settings (30) has the same row: "Turns tailbond off and opens the Tailscale app. Turn Tailscale on there if it says Disconnected". Afterwards, **12b**: "Disconnected — Stopped. Back to Tailscale: turn it on in its app." plus a card "Last step: turn Tailscale on — tailbond is off. Open the Tailscale app and check that it says Connected; if it says Disconnected, tap its switch. Until then, your Tailscale machines aren't reachable." and an **Open Tailscale** button.

**What works:** Very easy to find. The "Last step" card covers the thing I'd otherwise miss: Tailscale doesn't come back on by itself.

**Where I'd stumble:**
- Settings says the switch-back option "opens the Tailscale app", but 12b shows me still in tailbond with an **Open Tailscale** button. Does it open Tailscale for me or not? Maybe 12b is what I see when I come back, but I can't tell from the screens. And if I use the big switch to turn tailbond off (my Tailscale habit) instead of the button, 12 only says "…or switch back to Tailscale below". It doesn't say plainly that Tailscale is still off and my NAS is unreachable until I turn it on. 12b says this; 12 doesn't. *(low)*

---

## Findings

| # | task | finding | severity |
|---|---|---|---|
| 1 | 2 (also 3/4) | The phone's identity changes without a clear explanation. The pairing screen shows it as "pixel-8 · sam@example.com · 100.101.8.8", 02e says "they see this phone as homebox, so homebox's Tailscale access rules apply (not your own account's)", and the home screen shows "10.77.0.2 — Not a Tailscale address … under its access rules". A Tailscale user can't tell whether other machines still reach the phone at its 100.x address, or whether the NAS still sees them as themselves. "Access rules" is unexplained jargon. | medium |
| 2 | 1 | The **Download tailbond** button sits above the five drawings that "Read the pictures below first" says to read first. The user taps the button and Android covers the page before they've seen the steps. | low |
| 3 | 1 | After install the page says "tap Done and come back", but Android also offers Open, which users tap by reflex. That's recoverable via the onboarding footer, but it's a detour. Drawing ① says "then Open" but doesn't show it. | low |
| 4 | 1 | The safety note includes a SHA-256 hash and "sha256sum /usr/share/tailbond/tailbond.apk": noise for a non-technical user (it's de-emphasized). | low |
| 5 | 3 | A fresh, correctly set-up app greets the user with "Disconnected — Stopped.", which sounds like a failure. | low |
| 6 | 3 | The handoff dialog has five paragraphs. The real change from Tailscale habits ("Your internet traffic goes out from homebox too (like an exit node…)", on by default) is easy to skim past. | low |
| 7 | 4 | The machine list is the fourth section on the main screen and starts at the bottom of the first screen. In the Tailscale app it's the main content. | low |
| 8 | 4 | The list's copy icon sits next to the prominent MagicDNS name but copies the IP ("Copied 100.101.2.2"). | low |
| 9 | 6 | The Connections page has five overlapping data controls (Save mobile data mode, Mobile data "Use for tailbond", Advanced main/backup, Faster switch to backup, Mobile data limit). The user may wonder whether they also need to turn off "Use for tailbond". | low |
| 10 | 7 | The "Sign in to Wi-Fi" button and the small print "Tap the Wi-Fi network (it says "Sign in to network")" read as two different instructions. It's unclear what the button opens. | low |
| 11 | 8 | Settings says Switch back "opens the Tailscale app", yet 12b shows tailbond with an "Open Tailscale" button. And turning off with the big switch (12) doesn't say plainly that Tailscale is still off and the NAS unreachable. | low |

Totals: 0 high / 1 medium / 10 low
