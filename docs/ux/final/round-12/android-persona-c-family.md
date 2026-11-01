# Round 12 — Android persona C (family member)

**Who I am:** a family member who isn't very technical. Someone else set up the home server and put
the Tailscale app on my phone. I use the phone for messages, photos and video. I've never installed
an app from outside the Play Store. All I was told is the setup person's note in the task script:
"type **homebox/** into your phone's browser".

**Materials:** `task-script.md`, `screens.md`, every PNG in `web/` and `android/`, including the
dark-theme copies. The long phone page (`web-00-phone-page.png`) was too small to read in one piece,
so I read it in five slices.

---

## Task 1: Get the tailbond app onto the phone

**Where I tap**
1. In Chrome, with Tailscale on, I type `homebox/` and get **web-00a**: "Get tailbond on this phone".
   The first box says "Look at these five pictures first (or take a screenshot) ... The download
   button is below them. They're drawings: the dashed ring marks what to tap on the real screen."
2. I scroll past the five drawings (**web-00**): ① "Download anyway" → ② "Settings" → ③ "Allow from
   this source" toggle, then back → ④ "Install" → ⑤ "Scan app". Then I tap the green
   **"Download tailbond (27 MB)"** button.
3. On the real phone I follow the "In short" line: "Download anyway → Open → Settings → Allow from
   this source → back → Install (→ Scan app, if asked)".
4. The page says "When it says App installed, tap Done and come back to this page (Chrome, this
   tab) for step 2."

**What went well:** The page expects me to be nervous. It says "That's Android's normal check for
every app from outside the Play Store, and you only go through it once". It also draws the scary
"File might be harmful" box before I see the real one. Without that I would have tapped Cancel.
"Stuck, or not comfortable with this? Ask whoever runs homebox to install it for you" gives me a
way out. If I tap step 2 too early, **web-00b** brings me back with "tailbond isn't on this phone
yet, so step 2 brought you back here. Install it first, then tap step 2 again." That's clear.

**Where I'd get stuck or misunderstand**
- When an app finishes installing, my habit is to tap **Open**, not Done. The page asks for "tap
  Done". If I tap Open, I land on the app's own start screen (**01-onboarding**). The hint I'd need
  is at the very bottom, under the big "Get started" button, and it runs off the screen: "Installed
  tailbond from your home server's page in Chrome? Easier: go back to Chrome and tap step 2 there:
  it fills everything in". I'd just tap "Get started". That still works (see Task 2), so it's only
  a detour. **Low.**
- The "Is this safe?" paragraph has a long code ("SHA-256 54167117c3586c8ff05cc...") and
  "`sha256sum /usr/share/tailbond/tailbond.apk`". It's marked "For the curious", so I skip it, but a
  wall of gibberish in a safety paragraph makes me a little less sure. **Low.**

**What I expected:** In the Play Store I tap Install once and the app opens. This takes about six
taps and a trip into Settings. The page warns me about that, so it's acceptable.

## Task 2: Add the phone to homebox (pair), from the phone

**Where I tap**
1. Back in Chrome, step 2: **"Open tailbond and ask homebox"**.
2. The app opens on **02a**: "Asking homebox… Looking for homebox on your Tailscale network."
3. **02b**: "Waiting for approval on homebox… Ask whoever runs homebox to approve "pixel-8" on
   homebox's tailbond page ... There it shows as pixel-8 · sam@example.com · 100.101.8.8, so they
   can tell it's this phone. You can switch to another app to message them; tailbond keeps waiting
   for 10 minutes and carries on by itself once approved." I go to Messages and text my relative.
4. **02e**: "Add this phone to homebox?" I tap **Add phone**.

**What went well:** It tells me exactly who has to do what. It's fine for me to leave and message
them, and the app tells me how long it will wait. "homebox said no" (**02d**) is plain and kind:
"Whoever runs homebox declined this phone. Ask them, then ask again."

**Where I'd get stuck or misunderstand**
- The confirm box in **02e** says: "They see this phone as homebox (not as your own phone), so they
  allow it whatever they allow homebox." It also says "It connects over the internet to
  home.example.net:443". I don't know what the first sentence means for me. Is my phone pretending
  to be the server? Is that allowed? I'd tap Add phone because my relative said to, but with a
  little worry. **Low.**
- In "Can't find homebox" (**02c**), the first hint is useful: "Check that the Tailscale app says
  Connected". The next card says "Came here from your home server's page? Go back to it in your
  browser and tap step 2, Open tailbond and ask." That's exactly what I just did. Doing it again
  sends me round in a circle. **Low.**
- "Not sure of the name? It's the machine that runs tailbond, in the Tailscale app's list of
  machines." I think tailbond is the app on *my* phone, so "the machine that runs tailbond" doesn't
  mean anything to me. "the home server" would. **Low.**
- The same button is called "Open homebox page" in 02c/02d and "Open home server page" in 02 and
  behind the 02e dialog. **Low.**
- The phone page asks "Is homebox yours (it belongs to your Tailscale account)? Then you approve it
  yourself at http://homebox:8077". It's "our" home server, so I might think it's mine, open that
  page, and find I'm not allowed. The page does say "Nobody else on your Tailscale network can
  approve from there unless you named them", so I'd figure it out. **Low.**

**What I expected:** In Tailscale, someone approves a new device and it just appears. This works
the same way and keeps me informed, which is better than Tailscale's silence.

## Task 3: Connect

**Where I tap**
1. After Add phone, the main screen (**12-home-disconnected**) says "Disconnected. Stopped. Turn on
   to reach homebox and your Tailscale machines, or switch back to Tailscale below." I tap the big
   switch.
2. **04-handoff-other-vpn**: "Switch from Tailscale to tailbond?" I tap **Turn on tailbond**, then OK
   on Android's own box ("Next, Android asks you to allow tailbond's connection request. Tap OK.").
3. **11-home-connecting** ("Connecting… Reaching homebox…"), then **10-home-connected**:
   "Connected. Through homebox · Wi-Fi + mobile data".

**What went well:** The switch is where the Tailscale app's switch is. The pop-up warns me that
Tailscale will go off, and says how to go back: "To go back later, tap Switch back to Tailscale on
tailbond's main screen".

**Where I'd get stuck or misunderstand**
- On the connected main screen (**10**, and also in **15, 16, 17, 16c**), "This device" shows
  "10.77.0.2". Under it: "Not a Tailscale address: your Tailscale machines see this phone as
  homebox and allow it whatever they allow homebox. They can open connections to the phone at this
  address only if homebox's tailbond route is approved in the Tailscale admin console (your access
  rules apply)." To me, "Not a Tailscale address" plus "only if ... is approved in the Tailscale
  admin console" sounds like something isn't finished or approved. I'd message my relative to ask
  whether it's working, even though the big blue card says Connected. The copy icon beside it makes
  me think I'm supposed to do something with that number. The Tailscale app just shows my 100.x
  address with no warning-sounding text. Because this block sits on every connected screen, it also
  pushes the Machines list below the first screen. **Medium.**
- The pop-up has five paragraphs. The small print says "If tailbond switches itself off right after
  this, Tailscale is set as Always-on VPN on this phone: turn that off in Android Settings → VPN →
  Tailscale's gear". "Like an exit node" is also there. I don't know these words. I'd ignore them and
  tap Turn on tailbond, so this is only clutter. **Low.**
- If the server can't be reached (**13-error-unreachable**), the message includes "forward UDP port
  443", but it also says "if someone else set up your home server, show them this message", and
  there's a "Try again" button. I know what to do. **14-error-cert** says "Don't pair this phone
  again unless they ask you to", which is good advice. No finding.

## Task 4: Find and copy the NAS's MagicDNS name or Tailscale IP

**Where I tap**
1. On the main screen I scroll to "Machines" and type `nas` in "Search machines" (**18**). The row
   shows "nas · nas.tail1234.ts.net · 100.101.2.2 · linux" with a copy icon.
2. Tapping the copy icon shows "Copied 100.101.2.2".
3. Or I tap the row and get **19-machine-detail**: "Tap an address to copy it" with "MagicDNS name
   nas.tail1234.ts.net", "Short name nas", "Tailscale IPv4 100.101.2.2" and "Tailscale IPv6 …". Each
   has its own copy icon, and the message confirms "Copied nas.tail1234.ts.net".

**What went well:** This is just like the Tailscale app. The detail page labels each address,
including "MagicDNS name". The note "With tailbond on, apps on this phone can use these addresses
just like with Tailscale" reassures me.

**Where I'd get stuck or misunderstand**
- The single copy icon on the list row sits next to two values, a name and a number. I can't tell
  which one it will copy until the message tells me ("Copied 100.101.2.2"). If I wanted the name,
  I'd need a second try through the detail page. **Low.**

## Task 5: Tell whether Wi-Fi and 5G are both being used right now

**Where I look:** At the top of the main screen (**10**): "Connected. Through homebox · Wi-Fi +
mobile data". Under Connections: "Using both Wi-Fi and mobile data", with two tiles both saying
"In use" with green dots. When one breaks (**15**), it says "Using Wi-Fi only", and the Mobile data
tile shows "Not working right now" with a red dot. The Connections page (**20**) says the same at
the top.

**Verdict:** I could answer at a glance. It says "mobile data" rather than "5G", but that's the word
my phone uses anyway. No problems.

## Task 6: Make it save mobile data

**Where I tap:** On the main screen, the **"Save mobile data — Use mobile data only when Wi-Fi stops
working"** switch. After that (**16**) the screen says "Using Wi-Fi · saving mobile data (backup
only)", the Mobile data tile says "Not in use · saving data", and the line reads "Mode: Save mobile
data". When Wi-Fi drops (**17**), it says "Wi-Fi isn't working · using mobile data" and "In use ·
taking over", which is what I'd expect.

**What went well:** It's one obvious switch on the first screen, and its label is in my own words.

**Where I'd get stuck or misunderstand**
- If I tap "More options" (**21**), I also find "Mobile data limit: 1.23 GB of 5.00 GB used this
  month" and "Faster switch to backup (experimental) ... Uses a little mobile data." Now there are
  three data controls, and I'm not sure whether I'm supposed to set the limit as well. The note
  "Separate from Save mobile data: counts everything tailbond sends ..." helps, but I had to read it
  twice. **Low.**

## Task 7: Hotel Wi-Fi with a sign-in page; get Wi-Fi working and back into the bond

**Where I tap:** On the main screen (**16c**), a card at the top: "Wi-Fi needs sign-in. This Wi-Fi
shows a sign-in page before it lets you online. tailbond keeps you connected over mobile data
meanwhile, and adds Wi-Fi by itself once you're signed in." I tap **Sign in to Wi-Fi**. Android's
sign-in page isn't in the screenshots, so I'm trusting "adds Wi-Fi by itself" for the last part.

**What went well:** I'm told not to turn tailbond off: "you don't need to turn tailbond off". In the
past I would have turned the VPN off to make a hotel page load, so this saves me from a mistake. The
Wi-Fi tile says "Needs sign-in · not used yet", which matches.

**Where I'd get stuck or misunderstand**
- The small print says "Tap the Wi-Fi network (it says "Sign in to network") to open the page." Right
  under it is the "Sign in to Wi-Fi" button. Tap the network where? In the Wi-Fi list? In the
  notification? Or is the button enough? I'd tap the button first. If that opens Android's Wi-Fi
  list, the small print would make sense, but from this screen I can't tell whether it's one step or
  two. **Low.**

## Task 8: Turn tailbond off and go back to the Tailscale app

**Where I tap:** On the main screen, **"Switch back to Tailscale"** (it's on every main-screen
state). Then **12b**: "Disconnected. Stopped. Back to Tailscale: turn it on in its app." Below that,
a card: "Last step: turn Tailscale on. tailbond is off. Open the Tailscale app and check that it says
Connected; if it says Disconnected, tap its switch. Until then, your Tailscale machines aren't
reachable." I tap **Open Tailscale** and switch it on there.

**What went well:** The "last step" card is exactly what I need, because I'd otherwise assume
Tailscale comes back on by itself. If I just flip the big switch off instead, the Disconnected screen
(**12**) still points me to "switch back to Tailscale below".

**Where I'd get stuck or misunderstand**
- Settings (**30**) says this option "Turns tailbond off and opens the Tailscale app". After using it
  from the main screen, I'm still in tailbond (12b) with an "Open Tailscale" button. Either it didn't
  open Tailscale, or I came back to this screen later; I can't tell which. It's a small mismatch, and
  the card covers both cases. **Low.**

---

## Dark theme

I checked 01, 04, 10, 12, 19, 20 and 30 in dark, plus the dark phone page. The text is readable, the
dashed rings on the phone page still stand out, and the red "Remove from this phone" is still clearly
red. No findings.

## Findings

| # | Task | Finding | Severity |
|---|---|---|---|
| 1 | 3 (also 4–7) | "This device" on the connected main screen says "10.77.0.2 — Not a Tailscale address ... only if homebox's tailbond route is approved in the Tailscale admin console (your access rules apply)". It reads like something is unfinished or unapproved, so I'd message my relative even though it says Connected. The copy icon suggests I need that number, and the block pushes Machines below the fold. | medium |
| 2 | 1 | The page says "tap Done", but my habit is to tap Open. The app's start screen then hides "go back to Chrome and tap step 2" at the very bottom, cut off below "Get started". It's still recoverable by typing homebox. | low |
| 3 | 1 | The "Is this safe?" paragraph has a SHA-256 code and a `sha256sum /usr/share/...` command. It's labelled "For the curious", but it's noise in a safety message. | low |
| 4 | 2 | The confirm box says "They see this phone as homebox (not as your own phone), so they allow it whatever they allow homebox" and "home.example.net:443". It's unclear and slightly worrying for a non-technical person. | low |
| 5 | 2 | "Can't find homebox" tells me to "Go back to it in your browser and tap step 2", which is what I just did, so the advice goes round in a circle. | low |
| 6 | 2 | "Not sure of the name? It's the machine that runs tailbond". To me tailbond is the app on my phone; "the home server" would be clearer. | low |
| 7 | 2 | The same button is labelled "Open homebox page" (02c/02d) and "Open home server page" (02/02e). | low |
| 8 | 2 | "Is homebox yours (it belongs to your Tailscale account)?" A family member may think the shared home server is "theirs" and try http://homebox:8077 to approve themselves, which wastes a step. | low |
| 9 | 3 | The "Switch from Tailscale to tailbond?" pop-up is long. Its small print about "Always-on VPN ... Tailscale's gear" and "like an exit node" is jargon. | low |
| 10 | 4 | The list row's single copy icon sits beside both "nas.tail1234.ts.net" and "100.101.2.2", so I can't tell which one gets copied until the "Copied 100.101.2.2" message. | low |
| 11 | 6 | "More options" adds "Mobile data limit" and "Faster switch to backup (experimental)" next to Save mobile data, so I'm unsure whether I need to set the limit too. | low |
| 12 | 7 | "Tap the Wi-Fi network (it says "Sign in to network")" sits under a "Sign in to Wi-Fi" button, so I can't tell whether the button is enough or there's a second step, and where. | low |
| 13 | 8 | Settings says Switch back "opens the Tailscale app", but after using it I'm still in tailbond with an "Open Tailscale" button. The two don't match. | low |

Totals: 0 high / 1 medium / 12 low
