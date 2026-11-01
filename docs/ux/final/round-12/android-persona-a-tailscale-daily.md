# Round 12 — Android persona A (daily Tailscale user)

**Who I am:** I use the official Tailscale Android app every day. I know MagicDNS names, the machine list, the VPN toggle, and roughly what an exit node is. My job has nothing to do with networking. I have only ever installed apps from the Play Store.

**What I had:** the task script, the screen manifest, and the PNGs in `web/` and `android/` (packet-android, round 12). I looked at every screen. Long pages were cropped into sections so I could read them.

---

## Task 1: Get the tailbond app onto the phone

**Where I tap:**
1. With Tailscale on, I type `homebox/` in Chrome and get **web-00a**: "Get tailbond on this phone". Step 1 "Install the app" starts with a green box: "Look at these five pictures first (or take a screenshot) … The download button is below them."
2. I scroll past the "In short: Download anyway → Open → Settings → Allow from this source → back → Install (→ Scan app, if asked)" line and the five drawings (①–⑤) to **"Download tailbond (27 MB)"** (web-00).
3. Android's own screens appear in the order the drawings showed: "Download anyway", then "Open", then "Settings", then "Allow from this source", then back, then "Install", then "Scan app" if asked.
4. "When it says App installed, tap Done and come back to this page (Chrome, this tab) for step 2."

**How it went:** It's clear. I've never sideloaded anything, and "File might be harmful" would normally scare me off. The page warns about it ahead of time, and the dashed rings show what to tap. "That's Android's normal check for every app from outside the Play Store, and you only go through it once" is reassuring. The "Is this safe? … your phone reached it through Tailscale, which encrypts it" note makes sense to me as a Tailscale user. I skip the SHA-256 line, but it's marked "For the curious" and is grey, so it doesn't get in the way. The fallback "Chrome's Open message gone before you tapped it? … Chrome's menu ⋮ → Downloads" covers the case I'd worry about most. If I tap step 2 too early, web-00b says plainly: "tailbond isn't on this phone yet, so step 2 brought you back here. Install it first, then tap step 2 again."

**Small snags:**
- The download button isn't on the first screen. I have to scroll about two screen heights of drawings to reach it. The page does warn me ("The download button is below them"), so I didn't get lost. (low)
- "Then open the downloaded file." sits above the drawings and the download button, before I've downloaded anything. On first read I looked for a file I didn't have yet. (low)
- Android's "App installed" screen usually offers **Open** as well as **Done**. Out of habit I'd tap **Open**. That puts me on 01-onboarding instead of back in Chrome. The hint "Installed tailbond from your home server's page in Chrome? Easier: go back to Chrome and tap step 2 there: it fills everything in" is small text under "Get started", and it runs off the bottom of the first screen. I can still finish by tapping Get started and typing `homebox` (02-pairing), so it costs a bit more effort but doesn't block me. (low)

## Task 2: Add the phone to homebox (pair), from the phone

**Where I tap:** I go back to Chrome and tap step 2, **"Open tailbond and ask homebox"** (web-00). tailbond opens on **02a**: "Asking homebox… Looking for homebox on your Tailscale network." Then **02b**: "Waiting for approval on homebox… Ask whoever runs homebox to approve "pixel-8" … There it shows as pixel-8 · sam@example.com · 100.101.8.8 … tailbond keeps waiting for 10 minutes and carries on by itself once approved." I message the person who set it up. Once they approve, **02e** appears: "Add this phone to homebox? … This phone will be called "pixel-8" on homebox." I tap **Add phone**.

**How it went:** Very smooth. The phone name, my Tailscale login and the 100.x address on 02b are exactly how the other person would recognise my phone in Tailscale. The 10-minute note means I'm not afraid to switch to WhatsApp. Both failure screens are plain: 02c "Can't find homebox — Check that the Tailscale app says Connected, and that the name matches the one in its list of machines" and 02d "homebox said no — Whoever runs homebox declined this phone. Ask them, then ask again." Each has an **Ask again** button.

**Small snag:**
- 02e says "It connects over the internet to home.example.net:443". I only know this server as `homebox`. Tailscale never showed me an address like this, so for a moment I wondered whether this is still my homebox. The lock line "Secure connection to homebox" and the step 3 note on the phone page ("tailbond connects straight to homebox over the internet (not through Tailscale)") explain it, so I'd go on. (low)

## Task 3: Connect

**Where I tap:** On the main screen (**12-home-disconnected**), the card reads "Disconnected — Stopped. Turn on to reach homebox and your Tailscale machines". I tap the big switch. **04-handoff-other-vpn** opens: "Switch from Tailscale to tailbond? Android runs one VPN at a time. Turning on tailbond turns off Tailscale … Next, Android asks you to allow tailbond's connection request. Tap OK." I tap **Turn on tailbond**, then OK in Android's dialog. **11** shows "Connecting… Reaching homebox…" and **10** shows "Connected — Through homebox · Wi-Fi + mobile data".

**How it went:** It works just like the Tailscale switch. Before I'm asked, I'm told that Tailscale will go off and that the NAS stays reachable. "Your internet traffic goes out from homebox too (like an exit node; Settings changes that)" makes sense to me because I know exit nodes. The Always-on note is in small print, which is fine, since it only matters if something goes wrong. The first-time dialog (05, "Allow tailbond's VPN") matches.

**Small snag:**
- Under "This device", where Tailscale would show my phone's 100.x address, I see "pixel-8 (this phone) 10.77.0.2 — Not a Tailscale address: your Tailscale machines see this phone as homebox … They can open connections to the phone at this address only if homebox's tailbond route is approved in the Tailscale admin console (your access rules apply)." "Not a Tailscale address" reads like a warning at first. The admin-console sentence is beyond me, and I can't tell whether I'm supposed to act on it. It doesn't stop anything I need to do, and the same long text repeats on every connected screen (10, 15, 16, 16c, 17, 18). (low)

## Task 4: Find and copy the NAS's MagicDNS name or Tailscale IP

**Where I tap:** I scroll down the main screen to "Machines" and type `nas` in "Search machines" (**18**). One row is left: "nas · nas.tail1234.ts.net · 100.101.2.2 · linux". Tapping the copy icon shows "Copied 100.101.2.2". If I tap the row instead, **19-machine-detail** opens with "Tap an address to copy it": MagicDNS name `nas.tail1234.ts.net`, Short name `nas`, Tailscale IPv4 `100.101.2.2`, Tailscale IPv6. I tap the MagicDNS row and see "Copied nas.tail1234.ts.net".

**How it went:** It feels like the Tailscale app, with the same labels ("MagicDNS name", "Tailscale IPv4"). "With tailbond on, apps on this phone can use these addresses just like with Tailscale" answers the question I'd have.

**Small snag:**
- The copy icon on the list row copies the **IP** without saying so beforehand. I only find out from the "Copied 100.101.2.2" message. If I wanted the name, I have to open the row. (low)

## Task 5: Are Wi-Fi and 5G both in use right now?

**Where I look:** The main screen (**10**) says "Using both Wi-Fi and mobile data". Both tiles show a green "In use" with speeds (↓38.4 ↑4.1 Mbps and ↓22.7 ↑2.9 Mbps), and the status card says "Wi-Fi + mobile data". Tapping Connections opens **20**, which has the same headline, a combined 61.1 / 7.0 Mbps, and a card per connection.

**How it went:** I could answer at a glance without tapping anything. The one-connection-broken screen (**15**: "Using Wi-Fi only", mobile data "Not working right now" with a red dot) is just as clear.

**Small snag:**
- On 20, the top chart has a teal line and a purple line with no legend, so I'm guessing that purple means upload. The caption "2.05 GB this session" sits above "1.23 GB of 5.00 GB used this month" further down. At first glance the session total looks bigger than the month total, until I work out that the second number is mobile data only. (low)

## Task 6: Make it save mobile data

**Where I tap:** On the main screen I turn on the **"Save mobile data — Use mobile data only when Wi-Fi stops working"** switch. **16** then shows "Using Wi-Fi · saving mobile data (backup only)", mobile data "Not in use · saving data", and "Mode: Save mobile data". When Wi-Fi drops, **17** shows "Wi-Fi isn't working · using mobile data" and "In use · taking over".

**How it went:** It's a single switch on the first screen, worded the way Android's own Data Saver is. Connections (**21**) explains the link between the two controls: "The Save mobile data switch on the home screen picks "Save mobile data" here". The "Mobile data limit" section says it's "Separate from Save mobile data". No problems.

## Task 7: Hotel Wi-Fi with a sign-in page

**Where I tap:** The main screen (**16c**) keeps "Connected — Through homebox · mobile data" and adds a card: "Wi-Fi needs sign-in. This Wi-Fi shows a sign-in page before it lets you online. tailbond keeps you connected over mobile data meanwhile, and adds Wi-Fi by itself once you're signed in." I tap **Sign in to Wi-Fi**, sign in on Android's page, and Wi-Fi should rejoin on its own. Settings has "Tell me when Wi-Fi needs a sign-in" turned on, so I'd expect a notification as well.

**How it went:** It's reassuring that I don't need to turn tailbond off. "Android's sign-in page uses the Wi-Fi directly, so you don't need to turn tailbond off" answers exactly what I'd worry about. The Wi-Fi tile says "Needs sign-in · not used yet".

**Small snag:**
- The card gives two different instructions. The small text says "Tap the Wi-Fi network (it says "Sign in to network") to open the page", which seems to mean something in Android's Wi-Fi list or notification shade, not on this screen. Right under it is a **Sign in to Wi-Fi** button. I'd tap the button, but for a moment I wasn't sure whether the button and "tap the Wi-Fi network" are the same thing. (low)

## Task 8: Turn tailbond off and go back to Tailscale

**Where I tap:** The **"Switch back to Tailscale"** button sits right under the status card on every main-screen state (10, 11, 12, 13, 14, 15, 16, 16c, 17), and there's a row with the same name in Settings (**30**). After using it, **12b** shows "Disconnected — Stopped. Back to Tailscale: turn it on in its app" and a card "Last step: turn Tailscale on. tailbond is off. Open the Tailscale app and check that it says Connected; if it says Disconnected, tap its switch. Until then, your Tailscale machines aren't reachable." I tap **Open Tailscale** and switch Tailscale on.

**How it went:** Clear. Out of habit I might just flip the big switch off instead. Then 12 tells me "Stopped. … or switch back to Tailscale below", so I'd still understand that Tailscale isn't back on yet.

**Small snag:**
- The Settings row says "Switch back to Tailscale — Turns tailbond off and opens the Tailscale app", but 12b shows tailbond's own screen with an **Open Tailscale** button. I can't tell whether Tailscale opens automatically or whether I'm meant to tap that button. Either way I get there. (low)

---

## Findings

| # | Task | Finding | Severity |
|---|---|---|---|
| 1 | 1 | The "Download tailbond (27 MB)" button is about two screens below the first view, after five drawings. The page warns "The download button is below them", so I wasn't lost. | low |
| 2 | 1 | "Then open the downloaded file." appears above the download button, before there's a file to open. | low |
| 3 | 1/2 | If I tap Android's "Open" (not "Done") after installing, I land on 01-onboarding. The "go back to Chrome and tap step 2 there" hint is small text under "Get started" and runs off the first screen. Typing `homebox` on 02-pairing still works. | low |
| 4 | 2 | 02e "It connects over the internet to home.example.net:443": an unfamiliar address for a server I only know as `homebox`. | low |
| 5 | 3/4 | "This device" shows "10.77.0.2 — Not a Tailscale address … only if homebox's tailbond route is approved in the Tailscale admin console". This reads like a warning, has admin jargon, and repeats on every connected screen. | low |
| 6 | 4 | The list row's copy icon copies the IP ("Copied 100.101.2.2") without saying which value it copies. The MagicDNS name needs the detail screen. | low |
| 7 | 5 | The Connections chart has two unlabeled lines (teal and purple). "2.05 GB this session" next to "1.23 GB of 5.00 GB used this month" looks contradictory at first. | low |
| 8 | 7 | The "Wi-Fi needs sign-in" card says "Tap the Wi-Fi network (it says "Sign in to network")" and also shows a "Sign in to Wi-Fi" button. Two instructions for one action. | low |
| 9 | 8 | The Settings row says "Turns tailbond off and opens the Tailscale app", but 12b shows tailbond with an "Open Tailscale" button, so it's unclear whether Tailscale opens by itself. | low |

Totals: 0 high / 0 medium / 9 low
