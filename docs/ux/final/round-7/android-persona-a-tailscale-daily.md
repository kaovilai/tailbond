# Round 7 — Android persona A (daily Tailscale user)

**Persona:** I use the official Tailscale Android app every day. I know MagicDNS
names (`nas.tailXXXX.ts.net`), the machine list, the big VPN toggle, and that an
exit node sends all my browsing through another machine. I don't work in
networking.

**Materials:** `task-script.md`, `screens.md`, and the PNGs in `web/` and
`android/` (packet-android, round 7). I looked at every screen listed, including
the dark variants. Android's own dialogs, the Wi-Fi sign-in page and the
Tailscale app aren't in the packet, so I don't judge them.

Overall the journey works, and it feels close to the Tailscale app: same toggle
at the top, a machine list with MagicDNS names and 100.x addresses, copy icons,
and a detail page labelled "MagicDNS name" / "Tailscale IPv4". I finished all 8
tasks. I hit one real moment of confusion and some polish issues.

---

## Task 1: Get the tailbond app onto the phone

**What I'd do:** With Tailscale on, type `homebox` in Chrome. I land on
*web-00a*, "Get tailbond on this phone". I tap **"Download tailbond for Android
(27 MB)"**. Then I follow the drawn prompts in *web-00*: ① "File might be
harmful" → **Download anyway**, then Open; ② "For your security… unknown apps"
→ **Settings**; ③ **Allow from this source**, then back; ④ **Install**. Then
"When it says App installed, tap Done and come back to this page for step 2."

**Good:** The page draws Android's scary warnings ahead of time and says
"Android asks you to confirm a few times. That's expected." That's exactly what
I'd need not to panic at "File might be harmful". The short top paragraph
("Your Tailscale machines (like the NAS) stay reachable through it") matches
what I was told.

**Where I'd stumble:**
- If I tap step 2's big button first (it looks just as important as step 1's),
  *web-00b* shows Chrome scrolled back to **"1 Install the app"** with an orange
  outline around the card. No text says "tailbond isn't installed yet". I'd
  probably work it out from the outline, but a short sentence would remove the
  guess. **Low.**

The computer version (*web-00c*) with the QR code is clear but isn't part of my
task.

## Task 2: Add the phone to homebox (pair), from the phone

**What I'd do:** Back on the phone page, step 2: **"Open tailbond and ask
homebox"**. The app opens to *02a*: "Asking homebox… Looking for homebox on
your Tailscale network." Then *02b*: "Waiting for approval on homebox…". I'd
message the person who set it up. Once they approve, *02e* asks "Add this phone
to homebox?" and I tap **Add phone**.

**Good:** The phone's name ("pixel-8") is shown, so I can tell the approver what
to look for. The errors are clear and match the Tailscale app's wording: *02c*
"Can't find homebox — Check that the Tailscale app says Connected, and that the
name matches the one in its list of machines" and *02d* "homebox said no —
Whoever runs homebox declined this phone." From the launcher, *01-onboarding*
("Use tailbond instead of Tailscale on this phone", "Still on your tailnet",
"by name or 100.x address") speaks my language.

**Where I'd stumble:**
- **Mixed instructions while waiting (02a/02b).** The top card says "Keep this
  screen open; tailbond continues by itself once approved." The card right
  below says "**Came here from your home server's page? Go back to it in your
  browser and tap step 2, Open tailbond and ask.**" I *did* come from the home
  server's page and I *did* tap step 2. Reading that, I'd think I did something
  wrong and should go back to Chrome, which contradicts "Keep this screen open".
  Its button, **"Open home server page"**, is greyed out with no reason given,
  which adds to the confusion. While a request is in progress, this card
  shouldn't say this. **Medium.**
- The waiting text says the approver can use "http://homebox:8077 on a
  computer, **or a Flint 3's admin panel → VPN → tailbond**". I don't know what
  a Flint 3 is, or if we have one. I'd just forward the message, but it reads
  like it's meant for someone else. **Low.**
- *02e*: "your internet traffic goes out from home (you can change that in
  Settings)". In Tailscale, that's like an exit node, and I normally keep exit
  nodes off. Here it's on by default and only mentioned in small grey text next
  to "home.example.net:443", which means nothing to me. It is disclosed, and
  Settings (*30*) explains it well ("like using an exit node"). Still, I'd
  probably tap Add phone without noticing that all my browsing now goes through
  home. **Low.**

## Task 3: Connect

**What I'd do:** On the home screen (*12-home-disconnected*), flip the toggle
next to "Disconnected", just like in Tailscale. Because Tailscale's VPN is on, I
get *04*: "Switch from Tailscale to tailbond? Android runs one VPN at a time.
Turning on tailbond turns off Tailscale… You still reach your Tailscale
machines, like your NAS, through homebox. Next, Android asks you to allow
tailbond's connection request. Tap OK." → **Turn on tailbond** → Android's
dialog (not shown) → *11* "Connecting… Reaching homebox…" → *10* "Connected —
Through homebox · Wi-Fi + mobile data".

**Good:** This is the most familiar part. Same toggle, same "Connecting…" →
"Connected" flow. The dialog explains the one-VPN rule before Android's dialog
does, and tells me how to get back ("tap Switch back to Tailscale on the home
screen").

**Where I'd stumble:**
- Under "This device", my phone shows **"10.77.0.2"** with "Your Tailscale
  machines can reach this phone at this address, through homebox". In Tailscale
  my phone has a 100.x address. A 10.x address makes me wonder whether my phone
  is still "on" Tailscale and which address to give someone. The sentence helps,
  but the different number is surprising. **Low.**

## Task 4: Find and copy the NAS's MagicDNS name or Tailscale IP

**What I'd do:** On the home screen, scroll to **Machines**, type `nas` in
**"Search machines"** (*18*). The row shows "nas", "nas.tail1234.ts.net",
"100.101.2.2 · linux" with a copy icon. Tapping the copy icon shows the toast
"Copied 100.101.2.2". Or I tap the row and get *19*, "Tap an address to copy
it", with **MagicDNS name** `nas.tail1234.ts.net`, **Short name**, **Tailscale
IPv4** `100.101.2.2` and **Tailscale IPv6**, each with a copy icon. The toast
says "Copied nas.tail1234.ts.net".

**Good:** It works like the Tailscale app, with the same labels I already know.
The note "With tailbond on, apps on this phone can use these addresses just like
with Tailscale" answers my main worry.

**Where I'd stumble:**
- The copy icon in the list sits next to the MagicDNS name (the bigger line),
  but it copies the **IP**. I only find out from the toast. It doesn't break
  anything, since the detail page lets me pick. **Low.**

## Task 5: Tell whether Wi-Fi and 5G are both being used right now

**What I'd do:** Look at the home screen (*10*): "Connections — **Using both
Wi-Fi and mobile data**", with a Wi-Fi card ("In use", ↓38.4 ↑4.1 Mbps) and a
Mobile data card ("In use", ↓22.7 ↑2.9 Mbps), and "Through homebox · Wi-Fi +
mobile data" in the status card. Tapping Connections (*20*) adds totals and
graphs.

**Good:** Immediate and clear. Other states also read well: *15* "Using Wi-Fi
only" with "Not working right now" on Mobile data, and *17* "Wi-Fi isn't working
· using mobile data". It says "mobile data", not "5G", but that's fine.

No problems found.

## Task 6: Make it save mobile data

**What I'd do:** On the home screen, flip **"Save mobile data — Use mobile data
only when Wi-Fi stops working"**. *16* then shows "Using Wi-Fi · saving mobile
data (backup only)", Mobile data "Not in use · saving data", "Mode: Save mobile
data". When Wi-Fi drops (*17*), mobile data shows "In use · taking over".

**Good:** One obvious switch on the main screen, plain description, and the
status updates.

**Where I'd stumble:**
- If I go into **More options** first (*20/21*), there are four data-related
  controls: the "Save mobile data" mode radio, a **"Use for tailbond"** switch
  on Mobile data, **"Mobile data limit"** ("Separate from Save mobile data…"),
  and in *21* **"Faster switch to backup (experimental) … Uses a little mobile
  data."** The line "The Save mobile data switch on the home screen picks 'Save
  mobile data' here" helps, but I'd hesitate over whether turning off "Use for
  tailbond" is the "real" way to save data. **Low.**

## Task 7: Hotel Wi-Fi with a sign-in page

**What I'd do:** The home screen (*16c*) still says "Connected — Through homebox
· mobile data" and shows a card **"Wi-Fi needs sign-in — This Wi-Fi shows a
sign-in page before it lets you online. tailbond keeps you connected over mobile
data meanwhile, and adds Wi-Fi by itself once you're signed in."** I tap
**"Sign in to Wi-Fi"**, sign in on Android's page (not shown), and wait for
Wi-Fi to come back into the bond. The Wi-Fi card says "Needs sign-in · not used
yet".

**Good:** This is better than what I'm used to. With Tailscale, hotel sign-in
pages sometimes mean turning the VPN off. Here the card says "Android's sign-in
page uses the Wi-Fi directly, so you don't need to turn tailbond off." I stay
connected meanwhile.

**Where I'd stumble:**
- The small print says "**Tap the Wi-Fi network (it says "Sign in to network")
  to open the page.**" Right under it is a big **"Sign in to Wi-Fi"** button.
  Do I tap the button, or go find the network in Android's Wi-Fi list? I'd
  guess the button takes me to that list, but it's not clear which comes first.
  **Low.**
- There's no screen showing Wi-Fi back in the bond after sign-in, so I can't say
  how that looks. Going by the other screens, the cards would turn back to "In
  use".

## Task 8: Turn tailbond off and go back to the Tailscale app

**What I'd do:** On the home screen, tap the outlined **"Switch back to
Tailscale"** button under the status card. It's on every home state, including
the error screens *13/14*. *12b* shows "Disconnected — Stopped. Tailscale is in
charge on this phone." with a card "tailbond stays off until you turn it on
again. In the Tailscale app, turn on its switch if it says Disconnected." and
**Open Tailscale**. I open Tailscale and flip its toggle.

**Good:** Easy to find, and it says plainly what to do in the Tailscale app.
Settings (*30*) has the same option and mentions Always-on VPN in case I'd set
it for Tailscale.

**Where I'd stumble:**
- Settings describes Switch back as "**Turns tailbond off and opens the
  Tailscale app**", but *12b* shows me still in tailbond with an "Open
  Tailscale" button. Did it open Tailscale (and I came back), or do I still need
  to tap it? Small doubt. **Low.**
- On *12b*, below "Tailscale is in charge on this phone", the screen still says
  "**Will use Wi-Fi and mobile data together**" and "This phone **reaches your
  tailnet through homebox**". Once Tailscale is in charge, that's not true right
  now and contradicts the card above. **Low.**

---

## Findings

| # | Task | Finding | Severity |
|---|---|---|---|
| 1 | 2 | While asking/waiting (02a/02b), the card "Came here from your home server's page? Go back to it in your browser and tap step 2" contradicts "Keep this screen open", and its "Open home server page" button is greyed out with no reason; looks like I did something wrong | medium |
| 2 | 1 | Tapping step 2 before installing (web-00b) jumps back to step 1 with an orange outline but no words explaining the app isn't installed yet | low |
| 3 | 2 | Waiting text mentions "a Flint 3's admin panel → VPN → tailbond"; jargon for me, unclear if it applies | low |
| 4 | 2 | 02e: "your internet traffic goes out from home" (exit-node-like, on by default) is only in small print next to "home.example.net:443"; easy to accept without noticing | low |
| 5 | 3 | This phone shows "10.77.0.2", not a 100.x Tailscale address; surprising to a Tailscale user | low |
| 6 | 4 | List copy icon sits by the MagicDNS name but copies the IP ("Copied 100.101.2.2"); only the toast tells you | low |
| 7 | 6 | More options has overlapping data controls (mode radio, "Use for tailbond" on mobile data, "Mobile data limit", "Faster switch to backup … Uses a little mobile data"); unclear which is the one to use | low |
| 8 | 7 | "Tap the Wi-Fi network (it says "Sign in to network")" vs the big "Sign in to Wi-Fi" button: unclear which to do / what the button opens | low |
| 9 | 8 | Settings says Switch back "opens the Tailscale app", but 12b shows tailbond with an "Open Tailscale" button; unclear whether Tailscale already opened | low |
| 10 | 8 | 12b still says "Will use Wi-Fi and mobile data together" and "reaches your tailnet through homebox" while "Tailscale is in charge" | low |

Totals: 0 high / 1 medium / 9 low
