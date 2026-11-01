# Round 11 — Android persona C (family member)

**Who I am:** someone else set up our home server and put Tailscale on my phone. I use the phone for messages, photos and video. I've never installed an app from outside the Play Store. I was told: "with Tailscale on, type **homebox/** into your phone's browser."

**What I looked at:** `task-script.md`, `screens.md`, every PNG in `web/` and `android/` (I cut the long ones into pieces so I could read them).

Overall: I could finish all eight tasks with what's on screen. The app and the phone page talk to me in plain words most of the time. The hard part is still installing an app from outside the Play Store. A handful of spots use words I don't know ("exit node", ":443", "Always-on VPN").

---

## Task 1 — Get the tailbond app onto the phone

**Where I tap**
- `web-00a`: typing `homebox/` gets me "Get tailbond on this phone". The first box says: "Look at these five pictures first (or take a screenshot): once you open the file, Android's own screens cover this page. The download button is below them."
- `web-00` (whole page): I scroll past five drawings:
  - ① "File might be harmful … Download anyway"
  - ② "For your security … Settings"
  - ③ "Allow from this source" switch
  - ④ "Do you want to install this app? … Install"
  - ⑤ "App scan recommended … Scan app"

  Then I tap the green **"Download tailbond (27 MB)"** and follow the Android prompts.
- `web-00b`: if I tap step 2 too early, the page comes back with "tailbond isn't on this phone yet, so step 2 brought you back here. Install it first, then tap step 2 again." That's clear.

**Where I'd struggle**
- I have to remember five steps that will be hidden while I do them ("once you open the file, Android's own screens cover this page"). The page suggests taking a screenshot. Then I'd be flipping between Photos and the install screens, which I've never done. The scary bits are "File might be harmful" and "Play Protect hasn't seen this app before. Scan it for harmful behavior?" The page warns me ("That's Android's normal check for every app from outside the Play Store"). I'd still go slowly, and I'd probably phone whoever set it up partway through. The page also says "Stuck, or not comfortable with this? Ask whoever runs homebox to install it for you."
- The drawings have orange rings around "Download anyway", "Settings", "Install" and "Scan app", so they look like real buttons. The paragraph above says "the pictures below are drawings, not buttons". The drawings themselves don't say so, and I'd likely tap one once.
- In ⑤ ("Tap Scan app; … then tap Install") I can't tell whether Play Protect comes before or after the Install in ④.
- The answer to "is this safe?" sits below the download button in small grey text, mixed with "http://", "SHA-256 5416…" and "sha256sum /usr/share/tailbond/tailbond.apk". The first sentence is reassuring, but the rest is gibberish to me.
- When the installer finishes, the page tells me to "tap Done and come back to this page (Chrome, this tab)". Most people tap **Open** instead. That lands on `01-onboarding`, where the big button is "Get started". The helpful line ("Installed tailbond from your home server's page in Chrome? Easier: go back to Chrome and tap step 2 there; it fills everything in.") is under that button and runs off the bottom of the screen. I'd tap Get started and type the name by hand (see Task 2). That works, but it's the longer way.

**What I expected:** the Play Store, where one "Install" tap does everything. I know that's impossible here. The page does about as well as a page can.

**Severity:** medium (real effort and nerves, but I finish). The drawings, the step order, the safety text and the onboarding hint are each low.

---

## Task 2 — Add the phone to homebox (pair), from the phone

**Where I tap**
- I go back to Chrome and tap step 2's **"Open tailbond and ask homebox"** (`web-00`).
- `02a`: "Asking homebox… Looking for homebox on your Tailscale network."
- `02b`: "Waiting for approval on homebox… Ask whoever runs homebox to approve "pixel-8" … There it shows as pixel-8 · sam@example.com · 100.101.8.8, so they can tell it's this phone. You can switch to another app to message them; tailbond keeps waiting for 10 minutes and carries on by itself once approved." I'd message the person who runs it, and that's easy.
- `02e`: after they approve, I see "Add this phone to homebox?". I tap **Add phone**.
- Things that can go wrong are explained in plain words:
  - `02c`: "Can't find homebox: Check that the Tailscale app says Connected …" with **Ask again**.
  - `02d`: "homebox said no: Whoever runs homebox declined this phone. Ask them, then ask again."
- From the launcher instead (`01-onboarding` → `02-pairing`): I type "homebox" into "Home server's name in Tailscale" and tap **Ask to connect**. That works because I was told the name.

**Where I'd struggle**
- The `02e` dialog says "They see this phone as homebox (not as your own phone), so they allow it whatever they allow homebox." I don't know who "they" are (the other machines?). It also says "It connects over the internet to home.example.net:443". The ":443" means nothing to me. I'd tap Add phone anyway, but I'd wonder whether I just did something odd.
- `02-pairing` also offers "Or use the home server's page", "From another screen" (scan a code, "Not the code on the 'Get tailbond' page"), and "Have a link?". That's a lot of choices, but the top one is the obvious one.

**What I expected:** something like the Tailscale app's sign-in, where I tap a button and someone approves. That's what happens here.

**Severity:** low (the dialog wording).

---

## Task 3 — Connect

**Where I tap**
- The home screen is "Disconnected … Stopped. Turn on to reach homebox and your Tailscale machines, or switch back to Tailscale below." with a big switch (`12-home-disconnected`). I tap the switch.
- `04-handoff-other-vpn`: "Switch from Tailscale to tailbond? Android runs one VPN at a time. Turning on tailbond turns off Tailscale …". I tap **Turn on tailbond**.
- Android's own "connection request" appears, and the dialog told me to "Tap OK".
- `11-home-connecting`: "Connecting… Reaching homebox…".
- `10-home-connected`: "Connected. Through homebox · Wi-Fi + mobile data".

**Where I'd struggle**
- The dialog has a small-print paragraph: "If tailbond switches itself off right after this, Tailscale is set as Always-on VPN on this phone: turn that off in Android Settings → VPN → Tailscale's gear (or ask whoever set up the phone)." I don't know what Always-on VPN is, and I'd skip it. If it happened, I'd hope I remembered "ask whoever set up the phone".
- "Your internet traffic goes out from homebox too (like an exit node; Settings changes that)." I don't know what an exit node is. This comes up on the phone page, in this dialog and in Settings.

**What I expected:** the Tailscale app's single switch and a single Android OK. That's basically what I got, with one extra explanation dialog first.

**Severity:** low.

**Error screens (seen in passing):**
- `13-error-unreachable`: "Can't reach your home server … the home router may need to forward UDP port 443 … if someone else set up your home server, show them this message." I wouldn't understand the port part, but "show them this message" tells me what to do.
- `14-error-cert`: "Don't pair this phone again unless they ask you to." Clear.

---

## Task 4 — Find and copy the NAS's MagicDNS name or Tailscale IP

**Where I tap**
- I scroll the connected home screen to **Machines**. The **Search machines** box takes "nas", and one row is left: "nas · nas.tail1234.ts.net · 100.101.2.2 · linux" (`18-machines-search-copied`).
- I tap its copy icon and get the toast "Copied 100.101.2.2".
- Or I tap the row (`19-machine-detail`): "Tap an address to copy it", with "MagicDNS name nas.tail1234.ts.net", "Short name nas", "Tailscale IPv4 100.101.2.2" and "Tailscale IPv6 …". Tapping the first gives "Copied nas.tail1234.ts.net".

**Where I'd struggle**
- The row's copy icon copies the IP number, but the bigger text on that row is the name. I only found out which one I got from the toast. The detail page is clearer.

**What I expected:** the Tailscale app's machine list. This is very close to it, and the "MagicDNS name" label matches the words I was given.

**Severity:** low.

---

## Task 5 — Are Wi-Fi and 5G both being used right now?

**Where I look:** the main screen (`10-home-connected`):
- The status card says "Through homebox · Wi-Fi + mobile data".
- Under it: **"Using both Wi-Fi and mobile data"**, with a Wi-Fi tile ("● In use ↓38.4 ↑4.1 Mbps") and a Mobile data tile ("● In use ↓22.7 ↑2.9 Mbps"). Both dots are green.
- `20-connections` repeats this with graphs.

The answer is **yes**, and I get it right away. The app says "Mobile data" rather than 5G, which is fine.

**Where I'd struggle**
- When one connection breaks (`15-one-path-failed`), the screen says "Using Wi-Fi only" and the Mobile data tile says "Not working right now" with a red dot. There's no hint why or whether I should do anything. I'd guess it's fine because the top still says "Connected".

**Severity:** low.

---

## Task 6 — Make it save mobile data

**Where I tap**
- On the home screen I flip the **"Save mobile data — Use mobile data only when Wi-Fi stops working"** switch.
- `16-save-data-standby` then shows "Using Wi-Fi · saving mobile data (backup only)", the Mobile data tile says "Not in use · saving data", and "Mode: Save mobile data".
- `17-save-data-takeover`: when Wi-Fi drops, it says "Wi-Fi isn't working · using mobile data" and "In use · taking over". Very clear.
- `20/21-connections` ("More options") shows the same choice as a list and says "The Save mobile data switch on the home screen picks 'Save mobile data' here."

**Where I'd struggle:** nowhere. There's also a "Mobile data limit" with "Separate from Save mobile data" and "Faster switch to backup (experimental)", but I don't need them and they're out of the way.

**Severity:** no findings.

---

## Task 7 — Hotel Wi-Fi with a sign-in page

**Where I tap**
- `16c-home-wifi-needs-sign-in` stays "Connected · Through homebox · mobile data" and shows a card: **"Wi-Fi needs sign-in.** This Wi-Fi shows a sign-in page before it lets you online. tailbond keeps you connected over mobile data meanwhile, and adds Wi-Fi by itself once you're signed in."
- I tap **Sign in to Wi-Fi** and fill in the hotel's page.
- The Wi-Fi tile says "Needs sign-in · not used yet". Settings has "Tell me when Wi-Fi needs a sign-in" turned on, so I'd expect the app to point this out to me.

**Where I'd struggle**
- Just above the button, the small print says "Tap the Wi-Fi network (it says 'Sign in to network') to open the page." I'm not sure whether I'm meant to tap the button or go find the Wi-Fi network somewhere in Android. I'd tap the button and hope it takes me there.
- No screenshot shows what happens after I sign in. I can't confirm Wi-Fi goes back into the bond; I only have the card's promise that it "adds Wi-Fi by itself". (Not counted as a finding. That screen just isn't in the packet.)

**What I expected:** Android's own "Sign in to network" notification. The app's card doesn't fight it, and it tells me I don't need to turn tailbond off, which is reassuring.

**Severity:** low.

---

## Task 8 — Turn tailbond off and go back to Tailscale

**Where I tap**
- The outlined **"Switch back to Tailscale"** button on the main screen (also in Settings).
- `12b-home-switched-back-to-tailscale`: the screen says "Disconnected. Stopped. Back to Tailscale: turn it on in its app." A card reads "**Last step: turn Tailscale on.** tailbond is off. Open the Tailscale app and check that it says Connected; if it says Disconnected, tap its switch. Until then, your Tailscale machines aren't reachable." with an **Open Tailscale** button. I tap it and flip Tailscale's switch.
- If I just turned the big switch off instead, the disconnected screen still tells me to "switch back to Tailscale below".

**Where I'd struggle**
- Settings describes the button as "Turns tailbond off and opens the Tailscale app". After using it I'm looking at tailbond's screen with an "Open Tailscale" button, so I can't tell whether Tailscale already opened by itself or I still need to open it. Either way, the "Last step" card tells me what to do.

**What I expected:** in the Tailscale app, turning things off is one switch. Here it's two steps (switch back, then turn Tailscale on), but the screen spells out both.

**Severity:** low.

---

## General

- The main screen's title is "homebox" with "via home server · tail1234.ts.net" underneath. The word "tailbond" isn't at the top. Next to the Tailscale app, which also shows machine names and "ts.net", I might briefly not know which app I'm in. "homebox via home server" also reads oddly, since homebox *is* the home server.
- Dark theme screens (`*-dark`) read fine. I didn't spot contrast problems.

---

## Findings

| # | Task | Finding | Severity |
|---|------|---------|----------|
| 1 | 1 Install | I have to memorise or screenshot five steps ("Look at these five pictures first (or take a screenshot): once you open the file, Android's own screens cover this page"). Then I go through "File might be harmful" and "Play Protect hasn't seen this app before" on my own. I'd finish, but nervously and probably with a phone call. | medium |
| 2 | 1 Install | The drawn dialogs have orange rings around "Download anyway", "Settings", "Install" and "Scan app", so they look like real buttons. The "drawings, not buttons" note is only in the paragraph above them. | low |
| 3 | 1 Install | ⑤ "Tap Scan app; … then tap Install" comes after ④ "Tap Install", so I can't tell which comes first. | low |
| 4 | 1 Install | The "Is this safe?" answer is small grey text below the download button, mixed with "SHA-256 5416…" and "sha256sum /usr/share/tailbond/tailbond.apk". | low |
| 5 | 1→2 | Tapping Android's **Open** after install (instead of "Done") lands on onboarding. The "go back to Chrome and tap step 2 there" hint sits under "Get started" and runs off the screen, so I'd end up typing the name by hand. | low |
| 6 | 2 Pair | The approve dialog has words I don't follow: "home.example.net:443" and "They see this phone as homebox (not as your own phone), so they allow it whatever they allow homebox" (who is "they"?). | low |
| 7 | 3 Connect | The switch dialog's small print about "Always-on VPN … Android Settings → VPN → Tailscale's gear" and "like an exit node" is jargon. I'd skip it and wouldn't remember it if tailbond switched itself off. | low |
| 8 | 4 NAS | The row's copy icon copies the IP ("Copied 100.101.2.2") although the bigger text is the name. I only learn which one I got from the toast. | low |
| 9 | 5 Paths | "Mobile data: Not working right now" (red dot) gives no reason and no next step. | low |
| 10 | 7 Hotel | The "Sign in to Wi-Fi" button sits under the small print "Tap the Wi-Fi network (it says 'Sign in to network')", so I'm unsure whether to tap the button or look for the network in Android. | low |
| 11 | 8 Back | Settings says "Switch back to Tailscale … opens the Tailscale app", but afterwards I'm on tailbond's screen with an "Open Tailscale" button, so I can't tell whether Tailscale opened. | low |
| 12 | General | The main screen title is "homebox / via home server · tail1234.ts.net" and the app name "tailbond" isn't there. It's easy to confuse with the Tailscale app, and "homebox via home server" reads oddly. | low |

Totals: 0 high / 1 medium / 11 low
