# Round 11 — Android persona A (daily Tailscale user)

**Persona:** I use the official Tailscale Android app every day. I know MagicDNS names (`nas.tail1234.ts.net`), the machine list, the VPN toggle, and roughly what an exit node is. My job has nothing to do with networking.

**Materials used:** `task-script.md`, `screens.md`, all PNGs in `web/` and `android/`. I read the long screenshots in pieces. Android's own screens (VPN permission dialog, captive-portal page, the Tailscale app) aren't in the packet, so I only judge what tailbond and the homebox page show me.

---

## Task 1: Get the tailbond app onto the phone

**Where I tap**
1. With Tailscale on, I type `homebox/` into Chrome and get **web-00a**: "Get tailbond on this phone", then step 1 "Install the app".
2. The teal box says: "Look at these five pictures first (or take a screenshot): once you open the file, Android's own screens cover this page. The download button is below them." I scroll through the five drawings in **web-00**: ① "File might be harmful … Download anyway", ② "For your security … Settings", ③ "Install unknown apps / Chrome / Allow from this source", ④ "Do you want to install this app? Install", ⑤ "App scan recommended … Scan app".
3. I tap **"Download tailbond (27 MB)"**. In Android's prompts I do what the drawings showed: Download anyway → Open → Settings → Allow from this source → back → Install → (Scan app → Install).
4. Then: "When it says App installed, tap Done and come back to this page (Chrome, this tab) for step 2."

**Where I'd stumble**
- On the first screen (**web-00a**), the only thing that looks tappable is the drawn **"Download anyway"** button with a bright orange ring. The text above it does say "the pictures below are drawings, not buttons", but I skim. I could tap the drawing once before I realise the real button is several screens further down. (low)
- I've never installed an app from outside the Play Store. "File might be harmful", then a settings toggle, then Play Protect warning me: that's a lot compared to installing Tailscale from the Play Store. The page handles it well, though. It warns me first, draws each prompt, says "you only go through it once", explains why the `http://` page is still safe ("your phone reached it through Tailscale, which encrypts it"), and offers "Ask whoever runs homebox to install it for you." It's extra effort, but I always know what's coming. (low)
- If I miss the system "Open" message, the page covers that too: "Chrome's menu ⋮ → Downloads, or the Files app → Downloads. Tap tailbond.apk."

**What I expected:** A Play Store link. Since that isn't possible, this is about as good as sideloading gets.

---

## Task 2: Add the phone to homebox (pair), from the phone

**Where I tap**
1. Back in Chrome, step 2: **"Open tailbond and ask homebox"**. The app opens on **02a**: "Asking homebox… Looking for homebox on your Tailscale network."
2. **02b**: "Waiting for approval on homebox… Ask whoever runs homebox to approve "pixel-8" … There it shows as pixel-8 · sam@example.com · 100.101.8.8 … tailbond keeps waiting for 10 minutes and carries on by itself once approved." I text the person who runs homebox. Showing my Tailscale identity (my login and 100.x address) is reassuring; that's how I recognise devices in the Tailscale admin console too.
3. **02e**: "Add this phone to homebox?" I tap **Add phone**.

If I'd tapped Chrome's sign-in step too early (**web-00b**), the page tells me plainly: "tailbond isn't on this phone yet, so step 2 brought you back here. Install it first, then tap step 2 again." Good. The failure cards are clear too: **02c** "Can't find homebox — Check that the Tailscale app says Connected…" with **Ask again**, and **02d** "homebox said no … Ask them, then ask again."

**Where I'd stumble**
- The Android installer has both **Done** and **Open**. Many people tap Open without thinking. Then I land on **01-onboarding** ("Use tailbond instead of Tailscale on this phone") instead of Chrome. The hint I need is small print *under* the big **Get started** button, and on a 1080×2400 screen it runs off the bottom: "Installed tailbond from your home server's page in Chrome? Easier: go back to Chrome and tap step 2 there; it fills everything in." I'd probably tap Get started. **02-pairing** then gives me four ways to pair ("Ask your home server", "Or use the home server's page", "From another screen", "Have a link?"). I'd still finish by typing `homebox` and tapping **Ask to connect**, and the bold "Came here from your home server's page? Go back to it in your browser and tap step 2" helps. It's just a detour. (low)
- The **02e** confirmation says "It connects over the internet to home.example.net:443". I know this server as `homebox` or `homebox.tail1234.ts.net`, never `home.example.net`. For a moment I'd wonder whether I'm adding my phone to a different server. The bigger text ("Add this phone to homebox?", "Secure connection to homebox") reassures me, so I'd go ahead. (low)

**What I expected:** Like a Tailscale device approval: ask, wait, get approved. That's what happened.

---

## Task 3: Connect

**Where I tap**
1. Main screen, big switch in the "Disconnected" card ("Stopped. Turn on to reach homebox and your Tailscale machines, or switch back to Tailscale below.").
2. **04-handoff-other-vpn**: "Switch from Tailscale to tailbond? Android runs one VPN at a time. Turning on tailbond turns off Tailscale (or any other VPN) on this phone. You still reach your Tailscale machines, like your NAS, through homebox. Your internet traffic goes out from homebox too (like an exit node; Settings changes that). Next, Android asks you to allow tailbond's connection request. Tap OK." I tap **Turn on tailbond**, then OK in Android's dialog (not shown).
3. **11-home-connecting** ("Connecting… Reaching homebox…") becomes **10-home-connected**: "Connected — Through homebox · Wi-Fi + mobile data".

The "like an exit node" comparison is exactly right for someone like me. I understand straight away that my internet traffic now goes out from home, and where to change it (**30-settings** → "Only Tailscale machines"). The Always-on VPN note ("If tailbond switches itself off right after this, Tailscale is set as Always-on VPN…") answers a question before I'd have to ask it.

**Where I'd stumble**
- The app title is **"homebox"**, with "via home server · tail1234.ts.net" underneath. The word "tailbond" doesn't appear anywhere on the main screen. In the Tailscale app the header is the app/tailnet. Here I briefly wondered whether "homebox" was the app's name or which machine I'm on. The "Connected — Through homebox" card fixes that. (low)
- "This device — pixel-8 (this phone) 10.77.0.2 — Not a Tailscale address…" Without the explanation I'd have been confused that my phone has no 100.x address. With it, it's fine.

---

## Task 4: Find and copy the NAS's MagicDNS name or Tailscale IP

**Where I tap**
1. Scroll down to **Machines**, type `nas` into "Search machines" (**18-machines-search-copied**). One row: "nas — nas.tail1234.ts.net — 100.101.2.2 · linux".
2. I tap the copy icon on the row. The snackbar says **"Copied 100.101.2.2"**. Task done.
3. Or I tap the row (**19-machine-detail**): "Tap an address to copy it", with MagicDNS name, Short name, Tailscale IPv4 and IPv6, each with its own copy icon. "Copied nas.tail1234.ts.net". This is nicer than in the Tailscale app.

**Where I'd stumble**
- The row's copy icon sits next to the MagicDNS name, which is the more prominent line, but it copies the **IP**. If I wanted the name, I'd only find out from the snackbar and then have to open the detail page. The snackbar makes it obvious what I got, so nothing goes wrong. (low)

---

## Task 5: Tell whether Wi-Fi and 5G are both being used right now

**Where I look:** **10-home-connected**. The card says "Through homebox · Wi-Fi + mobile data", the heading says "Using both Wi-Fi and mobile data", and both tiles show a green dot with "In use" plus speeds (↓38.4 ↑4.1 / ↓22.7 ↑2.9 Mbps). There's an answer three times over. For comparison, **15-one-path-failed** says "Using Wi-Fi only" with a red "Not working right now" on mobile data, and **11** shows grey "Ready" before connecting. No problems. "Mobile data" is labelled as mobile data, not "5G", which is the Android wording I'm used to.

---

## Task 6: Make it save mobile data

**Where I tap:** the **"Save mobile data — Use mobile data only when Wi-Fi stops working"** switch right on the home screen. **16-save-data-standby** confirms it: "Mode: Save mobile data", "Using Wi-Fi · saving mobile data (backup only)", Mobile data "Not in use · saving data". **17-save-data-takeover** shows what happens when Wi-Fi drops: "Wi-Fi isn't working · using mobile data", "In use · taking over". Very clear.

**Where I'd stumble**
- If I open **More options** (**20/21-connections**), I find several more data controls: a "Use for tailbond" switch on the Mobile data card, "Advanced: choose which connection is main or backup", the "How to use your connections" mode list (which repeats Save mobile data), "Faster switch to backup (experimental)", and a separate "Mobile data limit". The page does explain how they relate ("The Save mobile data switch on the home screen picks 'Save mobile data' here", "Separate from Save mobile data: …"), but it's a lot of switches for "use less data". I'd wonder whether I should also turn off "Use for tailbond" on mobile data. The home switch is enough, so I'd leave the rest alone. (low)

---

## Task 7: Hotel Wi-Fi with a sign-in page

**Where I look/tap:** **16c-home-wifi-needs-sign-in**. There's a card right under the status: "Wi-Fi needs sign-in — This Wi-Fi shows a sign-in page before it lets you online. tailbond keeps you connected over mobile data meanwhile, and adds Wi-Fi by itself once you're signed in." Below that: "Android's sign-in page uses the Wi-Fi directly, so you don't need to turn tailbond off." I tap **Sign in to Wi-Fi**, sign in on the hotel page (Android's screen, not shown), and tailbond adds Wi-Fi back by itself. The Wi-Fi tile ("Needs sign-in · not used yet", red dot) tells me why it isn't in the bond yet.

That's better than what I'm used to. With Tailscale I'd usually turn the VPN off before signing in to hotel Wi-Fi.

**Where I'd stumble**
- The card has two instructions. The small print says "Tap the Wi-Fi network (it says "Sign in to network") to open the page", and there's also a big **Sign in to Wi-Fi** button. It doesn't say where "the Wi-Fi network" is (Android's Wi-Fi list? a notification?) or whether the button does that for me. I'd tap the button and follow whatever Android shows. (low)

---

## Task 8: Turn tailbond off and go back to the Tailscale app

**Where I tap:** **Switch back to Tailscale**, the full-width button under the status card. It's on every main-screen state, including the error states **13** and **14**. Afterwards (**12b**): "Disconnected — Stopped. Back to Tailscale: turn it on in its app." plus a card "Last step: turn Tailscale on — tailbond is off. Open the Tailscale app and check that it says Connected; if it says Disconnected, tap its switch. Until then, your Tailscale machines aren't reachable." and **Open Tailscale**. Warning me that my machines aren't reachable until I do that last step is exactly right; on Android, Tailscale doesn't come back by itself.

If I just flip the big switch off instead (**12**), it still points me the right way: "…or switch back to Tailscale below."

**Where I'd stumble**
- **30-settings** describes the same action as "Switch back to Tailscale — Turns tailbond off and opens the Tailscale app." But after I use it, **12b** asks me to tap **Open Tailscale** myself. So did it open Tailscale for me or not? I'd end up in the Tailscale app either way, but the two screens describe different behaviour. (low)

---

## Overall

As a daily Tailscale user I could finish all eight tasks without getting stuck. Comparisons to things I already know ("like an exit node", MagicDNS name / Tailscale IPv4 labels, "check that the Tailscale app says Connected") do a lot of the work. Every screen that hands off to Android (installer, VPN permission, captive portal, Always-on VPN) tells me ahead of time what I'll see and what to tap. What's left is polish: wording that doesn't match between screens, a copy button that copies something other than what I'd guess, and a few small-print hints sitting where they're easy to miss.

## Findings

| # | task | finding | severity |
|---|---|---|---|
| 1 | 1 Install | On web-00a the first tappable-looking element is the drawn "Download anyway" with an orange ring; the real "Download tailbond (27 MB)" button is several screens below. "drawings, not buttons" is easy to skim past. | low |
| 2 | 1 Install | The sideload chain ("File might be harmful" → Settings → "Allow from this source" → Install → Play Protect "Scan app") is a lot of extra effort compared to the Play Store. It's well explained, Android requires it, and the page offers "Ask whoever runs homebox to install it for you." | low |
| 3 | 1→2 Pair | If I tap Android's "Open" instead of "Done", I land on 01-onboarding. The key hint ("Easier: go back to Chrome and tap step 2 there; it fills everything in.") is small print under "Get started" and partly off-screen, so I take the 4-option manual pairing screen instead (still finishable). | low |
| 4 | 2 Pair | The 02e dialog says "It connects over the internet to home.example.net:443", an address I've never seen for homebox (I know homebox / homebox.tail1234.ts.net). A brief "is this the right server?" moment. | low |
| 5 | 3 Connect | The main screen's title is "homebox" ("via home server · tail1234.ts.net") and "tailbond" isn't anywhere on it, so I briefly wondered what the app or the device is called. | low |
| 6 | 4 NAS | The copy icon on the machine row copies the IP ("Copied 100.101.2.2") even though the MagicDNS name is the prominent line. Getting the name means opening the detail page. | low |
| 7 | 6 Save data | Connections page has overlapping data controls (Mobile data "Use for tailbond", "Advanced: choose which connection is main or backup", mode list, "Faster switch to backup", "Mobile data limit"). I'd wonder whether I should also turn off "Use for tailbond". | low |
| 8 | 7 Hotel | The 16c card says "Tap the Wi-Fi network (it says "Sign in to network")" without saying where that is, next to a separate "Sign in to Wi-Fi" button. Two instructions for one action. | low |
| 9 | 8 Back to Tailscale | Settings says "Switch back to Tailscale — Turns tailbond off and opens the Tailscale app", but 12b afterwards asks me to tap "Open Tailscale" myself, so the two screens disagree on what the button does. | low |

Totals: 0 high / 0 medium / 9 low
