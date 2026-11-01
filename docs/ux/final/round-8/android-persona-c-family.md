# Round 8 — Android persona C (family member)

**Who I am:** a family member who isn't very technical. Someone else set up the home server and put the Tailscale app on my phone. I use the phone for messages, photos and video. I've never installed an app from outside the Play Store.

**What I had:** `task-script.md`, `screens.md` and the PNGs in `web/` and `android/` of `out/ux/final/round-8/packet-android/`. I looked at every screenshot, including the dark versions. There is also an extra screenshot, `03-deeplink-confirm.png`, that the manifest doesn't list. It looks the same as `02e-pairing-ask-approved-confirm.png`, so I treated it as the same dialog.

**What I was told:** "with Tailscale on, type **homebox/** into your phone's browser (with the slash at the end)."

---

## Task 1: Get the tailbond app onto the phone

**Where I'd tap**
- I type `homebox/` in Chrome and get **web-00a**: "Get tailbond on this phone", then "1 Install the app" with a big green **"Download tailbond for Android (27 MB)"** button. That's an easy start.
- I keep scrolling (**web-00**). The page draws the four Android prompts I'll see:
  1. "File might be harmful … Do you want to keep tailbond.apk anyway?", where I tap **Download anyway**, then Open.
  2. "For your security, your phone currently isn't allowed to install unknown apps from this source", where I tap **Settings**.
  3. "Install unknown apps / Chrome / **Allow from this source**", where I turn it on and go back.
  4. "tailbond / Do you want to install this app?", where I tap **Install**. The page adds: "If Play Protect offers to scan the app, let it (Scan app), then Install."
- Then: "When it says App installed, tap **Done** and come back to this page (Chrome, this tab) for step 2."

**Where I'd get stuck or be confused**
- **The security warnings (medium).** I've always been told not to tap anything that says "File might be harmful" and not to "Allow from this source". The page clearly expects this ("Android asks you to confirm a few times. That's expected.") and has a way out ("Stuck, or not comfortable with this? Ask whoever runs homebox to install it for you."). Because the prompts are drawn for me, I'd probably get through. Still, it's four scary prompts in a row, and I'd be nervous the whole time. With apps from the Play Store I just tap "Install" once. The Play Protect step is only one sentence and isn't drawn like the other four. If Play Protect showed me something alarming, I wouldn't know what it looks like.
- **Done or Open (medium).** When Android finishes, it usually offers **Open** and **Done**. "Open" is what I'd tap without thinking. If I do, I land on the app's first screen (**01-onboarding**): "Get started". That leads to **02-pairing**, which asks me to type "Home server's name in Tailscale". There's a box that says "Came here from your home server's page? Go back to it in your browser and tap step 2", but its button, **"Open home server page"**, is greyed out. The one button that seems to take me back doesn't work, and nothing says why. I'd have to work out how to switch back to Chrome myself, or guess that typing "homebox" works. (It does work: **02c** and **02d** show "homebox" typed in.)
- **Jargon in small print (low).** "This page starts with http://, but it comes to you through Tailscale, which encrypts it. To check the file: SHA-256 54167117c3586c8ff05cc983c… Later updates must be signed with the same key, or Android refuses them." It's grey and small, so I'd skip it. It doesn't mean anything to me, and the warning about "refuses them" sounds a bit ominous.
- **Pictures that look like buttons (low).** The drawn dialogs, with orange rings around "Download anyway", "Settings" and "Install", look exactly like real dialogs on a very long page. I might try tapping the picture itself.

**What I expected:** a Play Store-style "Install" and nothing else, or at least one plain sentence saying "tap Done, not Open".

---

## Task 2: Add the phone to homebox (pair), from the phone

**Where I'd tap**
- Back in Chrome, step 2 is "Add this phone to homebox" with the green button **"Open tailbond and ask homebox"**.
- If tailbond isn't installed yet, **web-00b** brings me back with a helpful note: "tailbond isn't on this phone yet, so step 2 brought you back here. Install it first, then tap step 2 again."
- The app opens on **02a**: "Asking homebox… Looking for homebox on your Tailscale network." A few seconds later it shows **02b**: "Waiting for approval on homebox… Ask whoever runs homebox to approve "pixel-8" … tailbond keeps waiting for 10 minutes and carries on by itself once approved."
- I'd text my family member. When they approve, **02e** appears: "Add this phone to homebox?" I tap **Add phone**.
- If things go wrong, the messages are plain: **02c** "Can't find homebox: Check that the Tailscale app says Connected…" and **02d** "homebox said no: Whoever runs homebox declined this phone. Ask them, then ask again."

**Where I'd get stuck or be confused**
- **Waiting text is for the other person (low).** **02b** packs details meant for whoever runs homebox into my screen: "pixel-8 · sam@example.com · 100.101.8.8" and "http://homebox:8077". It's useful to pass on, but there's no button to share or copy that message. I'd end up taking a screenshot to send them.
- **Technical wording in the confirm dialog (low).** **02e** has small print: "It connects over the internet to home.example.net:443, and your internet traffic goes out from home (you can change that in Settings)." I can't follow "home.example.net:443". The fact that my browsing goes out through the home server is real information, but it's in the smallest text on the dialog.
- Apart from that, this went smoothly. "You can switch to another app to message them" was reassuring.

**What I expected:** something like the Tailscale sign-in, where I wait and it just finishes. This is close to that.

---

## Task 3: Connect

**Where I'd tap**
- On the main screen (**12-home-disconnected**), I tap the big switch next to "Disconnected".
- **04-handoff-other-vpn**: "Switch from Tailscale to tailbond? Android runs one VPN at a time. Turning on tailbond turns off Tailscale … You still reach your Tailscale machines, like your NAS, through homebox. Next, Android asks you to allow tailbond's connection request. Tap OK." I tap **Turn on tailbond**, then OK on Android's prompt. **05-handoff-first-permission** shows the same idea when no other VPN is on.
- **11-home-connecting** shows "Connecting… Reaching homebox…". Then **10-home-connected** shows "Connected. Through homebox · Wi-Fi + mobile data".

**Where I'd get stuck or be confused**
- **Android's own prompt isn't shown (low).** The install prompts were drawn for me, but this one isn't. I only get "Tap OK". If Android's wording sounds worrying, I'm going in blind.
- **"Tailnet" (low).** "This phone reaches your tailnet through homebox" (**12**, **04**), and "Still on your tailnet" on onboarding. I don't know what a "tailnet" is. "Your Tailscale machines" is what everything else says.
- Otherwise this was clear, and the warning that Tailscale goes off was honest and upfront.

---

## Task 4: Find and copy the NAS's MagicDNS name or Tailscale IP

**Where I'd tap**
- I scroll down the main screen (**10-home-connected-full**) to **Machines**. I see "nas / nas.tail1234.ts.net / 100.101.2.2 · linux" with a copy icon.
- **18-machines-search-copied**: I type "nas" in "Search machines" and tap the copy icon. It says "Copied 100.101.2.2". Done.
- I can also tap the row to open **19-machine-detail**: "Tap an address to copy it", then "MagicDNS name nas.tail1234.ts.net", "Short name nas", "Tailscale IPv4 100.101.2.2" and IPv6. Tapping copies it: "Copied nas.tail1234.ts.net".

**Where I'd get stuck or be confused**
- **The copy icon copies the number (low).** In the list, the bigger line under "nas" is the name `nas.tail1234.ts.net`, but the copy icon copied the number (**18**: "Copied 100.101.2.2"). Nothing tells me which one it copies until after I tap. The task allowed either, so it's fine, but if I'd wanted the name I'd be surprised.
- **"This device" block (low).** Above the list is **"pixel-8 (this phone) 10.77.0.2: Not a Tailscale address: to your Tailscale machines this phone's traffic comes from homebox, under its access rules…"**. It's the first copy icon I see and it's full of words I don't know. I might copy that number by mistake.

**What I expected:** like the Tailscale app, a list of machines where tapping one shows its addresses. That's what I got.

---

## Task 5: Tell whether Wi-Fi and 5G are both being used right now

**Where I look**
- **10-home-connected**: "Using both Wi-Fi and mobile data", with a **Wi-Fi** card ("In use", green dot) and a **Mobile data** card ("In use", green dot). The top card says "Through homebox · Wi-Fi + mobile data". Clear yes. (It says "mobile data" rather than "5G", which is fine by me.)
- **15-one-path-failed**: "Using Wi-Fi only", with the Mobile data card showing "Not working right now" and a red dot. Also clear.
- **20-connections**, through the Connections area or "More options", shows the same with graphs. I didn't need it.

**Where I'd get stuck or be confused**
- **"Ready" looks like "working" (low).** When off or connecting (**12**, **11**), both cards say "Ready" with grey dots under "Will use Wi-Fi and mobile data together". At a glance "Ready" reads as "it's working". The heading says "Will use", so if I read carefully I get it.
- The speeds ("↓38.4 ↑4.1 Mbps") mean nothing to me, but they don't get in the way.

---

## Task 6: Make it save mobile data

**Where I'd tap**
- Right on the main screen (**10**): **"Save mobile data: Use mobile data only when Wi-Fi stops working"**, with a switch. I flip it.
- **16-save-data-standby**: "Using Wi-Fi · saving mobile data (backup only)", with Mobile data showing "Not in use · saving data" and "Mode: Save mobile data". This is exactly what I wanted.
- **17-save-data-takeover**: when Wi-Fi drops, it says "Wi-Fi isn't working · using mobile data", with Mobile data showing "In use · taking over". That's reassuring.

**Where I'd get stuck or be confused**
- **Too many ways to save data on the Connections page (low).** If I wander into **21-connections-save-data**, there are three different-looking controls:
  - the "Save mobile data" choice among the four modes;
  - a "Use for tailbond" switch on the Mobile data card ("Turning this off doesn't turn off the phone's mobile data");
  - a "Mobile data limit" box: "Separate from Save mobile data: counts everything tailbond sends…".

  It's also not clear whether "Faster switch to backup (experimental) … Uses a little mobile data" undoes the saving. The home-screen switch was enough, so I'd just leave this page alone.

---

## Task 7: Hotel Wi-Fi with a sign-in page

**Where I'd tap**
- **16c-home-wifi-needs-sign-in**: still "Connected · Through homebox · mobile data", with a card: **"Wi-Fi needs sign-in. This Wi-Fi shows a sign-in page before it lets you online. tailbond keeps you connected over mobile data meanwhile, and adds Wi-Fi by itself once you're signed in."** Then a big **"Sign in to Wi-Fi"** button. The Wi-Fi card says "Needs sign-in · not used yet".
- I'd tap **Sign in to Wi-Fi**, fill in the hotel page, and expect Wi-Fi to come back by itself, as the card promises. Settings (**30-settings**) also has "Tell me when Wi-Fi needs a sign-in", which is on.

**Where I'd get stuck or be confused**
- **Two instructions (low).** The small grey line under the main text says "Tap the Wi-Fi network (it says "Sign in to network") to open the page. Android's sign-in page uses the Wi-Fi directly, so you don't need to turn tailbond off." So is it the big button or "the Wi-Fi network" I'm meant to tap, and where would that be? I'd assume the button takes me there, but the most important "how" is in the smallest print. I can't see what the button opens, because Android's screens aren't in the packet.
- I liked "you don't need to turn tailbond off". That's exactly what I would have tried.

---

## Task 8: Turn tailbond off and go back to the Tailscale app

**Where I'd tap**
- On the main screen, the outlined button **"Switch back to Tailscale"** sits right under the status card (**10**, **11**, **15**, **16**). Settings has the same option: "Turns tailbond off and opens the Tailscale app. Turn Tailscale on there if it says Disconnected."
- Afterwards, **12b** shows "Disconnected. Stopped. Tailscale is in charge on this phone." Below that is a card: "**Tailscale is in charge on this phone.** tailbond stays off until you turn it on again. In the Tailscale app, turn on its switch if it says Disconnected." There's an **Open Tailscale** button.

**Where I'd get stuck or be confused**
- **"In charge" sounds like it's working (medium).** The big heading "Tailscale is in charge on this phone" makes me think I'm back to normal and can reach the NAS. Only the smaller text says I may still need to turn Tailscale's own switch on. If I stop reading at the heading, which I would, I might think everything's fine while Tailscale is actually off. I'd find out later when the NAS doesn't open. I expected "switch back" to put Tailscale back on, or else a plain "Tailscale is off. Open it and turn it on" headline.
- **Two ways to turn off (low).** There's both the big switch and "Switch back to Tailscale". If I just flip the big switch off (like turning off Tailscale), I end up with neither on. **12** does say "Stopped. Turn on to reach homebox and your Tailscale machines, or switch back to Tailscale below", which helps if I read it.

---

## Other notes

- Dark theme (**01**, **04**, **10**, **12**, **19**, **20**, **30**, **web-00**): everything was readable and I saw no problems.
- The computer version of the page (**web-00c**) shows a QR code and the same steps. That's useful if my family member shows it to me on a laptop.
- I never needed Settings (**30**) or "Apps that skip tailbond" (**31**) for these tasks.

## Findings

| # | Task | Finding | Severity |
|---|------|---------|----------|
| 1 | 1 | Four security prompts in a row ("File might be harmful", "Download anyway", "Allow from this source", Play Protect). The page warns me well, but it goes against everything I've been taught, and the Play Protect step isn't drawn like the others. | medium |
| 2 | 1→2 | After installing, the natural tap is Android's "Open", but the page says "tap Done". Tapping Open lands on onboarding, then pairing, where "Open home server page" is greyed out with no reason given. I have to find my own way back to Chrome or type "homebox". | medium |
| 3 | 1 | "SHA-256 54167117…", "This page starts with http://", "Later updates must be signed with the same key, or Android refuses them": jargon that sounds ominous. | low |
| 4 | 1 | The drawn Android dialogs, with orange rings around "Download anyway" and "Install", look like real tappable dialogs on a very long page. | low |
| 5 | 2 | The waiting screen ("pixel-8 · sam@example.com · 100.101.8.8", "http://homebox:8077") is for the other person, but there's no share or copy button to send it to them. | low |
| 6 | 2 | The confirm dialog's small print "home.example.net:443" is jargon, and "your internet traffic goes out from home" is buried in the smallest text. | low |
| 7 | 3 | Android's VPN prompt isn't previewed the way the install prompts were. It only says "Tap OK". | low |
| 8 | 3 | "tailnet" ("This phone reaches your tailnet through homebox", "Still on your tailnet") is unexplained jargon. | low |
| 9 | 4 | The list's copy icon copies the IP ("Copied 100.101.2.2") although the bigger line shown is the name. There's no hint before tapping. | low |
| 10 | 4 | The "This device" block ("10.77.0.2 Not a Tailscale address … under its access rules") comes before the machine list and is easy to copy by mistake. | low |
| 11 | 5 | When off or connecting, both cards say "Ready" with grey dots, which at a glance reads as "working". | low |
| 12 | 6 | The Connections page has three overlapping data controls (the Save mobile data mode, "Use for tailbond" on Mobile data, and "Mobile data limit … Separate from Save mobile data"), plus an "experimental" option that "uses a little mobile data". | low |
| 13 | 7 | The sign-in card has the "Sign in to Wi-Fi" button and also small print "Tap the Wi-Fi network (it says "Sign in to network")". It's unclear which to do, and the key how-to is in the smallest text. | low |
| 14 | 8 | After switching back, the heading "Tailscale is in charge on this phone" sounds like everything is working, but Tailscale may still be Disconnected. Only the smaller text says to turn its switch on. | medium |
| 15 | 8 | There are two ways to turn off (the big switch and "Switch back to Tailscale"). Flipping only the switch leaves neither app on, although "or switch back to Tailscale below" helps. | low |

Totals: 0 high / 3 medium / 12 low
