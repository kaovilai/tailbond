# Round 9 — Android persona A (daily Tailscale user)

Tester: simulated persona A. I use the official Tailscale Android app every
day: I know the machine list, MagicDNS names (`nas.tail1234.ts.net`), the big
VPN toggle, and roughly what an exit node is. My job has nothing to do with
networking.

Materials: `task-script.md`, `screens.md` and every PNG in `web/` and
`android/` (light and dark). Nothing else.

Starting point: the phone has only the Tailscale app, connected. I was told to
type **homebox/** into the phone's browser.

---

## Task 1: Get the tailbond app onto the phone

**Where I tap**

1. Chrome, type `homebox/`. The page opens (`web-00a-phone-page-first-screen`):
   "Get tailbond on this phone", then "tailbond joins your phone's Wi-Fi and
   mobile data into one faster, steadier connection to homebox… Your
   Tailscale machines (like the NAS) stay reachable through it." That matches
   what I was told.
2. Step 1 "Install the app": I tap the green **Download tailbond for Android
   (27 MB)** button.
3. Then I follow the drawn steps (`web-00-phone-page`): ① "File might be
   harmful", tap **Download anyway**, then Open. ② "For your security…", tap
   **Settings**. ③ turn on "Allow from this source", go back. ④ **Install**.
   ⑤ maybe "App scan recommended", tap **Scan app**, then Install.
4. "When it says App installed, tap Done and come back to this page (Chrome,
   this tab) for step 2."

**How it went**

- This was easier than I expected for an app that isn't in the Play Store. The
  page says what Android's warnings will look like before I see them ("the
  pictures below are drawings, not buttons"), so "File might be harmful" didn't
  scare me off. The orange rings show which button to press.
- It covers the part I'd normally get stuck on: "Chrome's Open message gone
  before you tapped it? … Chrome's menu ⋮ → Downloads, or the Files app →
  Downloads. Tap tailbond.apk."
- "Is this safe? This page starts with http://, but your phone reached it
  through Tailscale…" answers the worry I'd have about the http:// address.
  I'd never check the SHA-256, but it doesn't get in the way.
- If I tap step 2 before installing, the page brings me back with "tailbond
  isn't on this phone yet, so step 2 brought you back here. Install it first,
  then tap step 2 again." (`web-00b`). Good.
- On a computer the same address shows a QR code and "type homebox/ into the
  phone's browser (with the slash at the end, so it opens this page instead of
  searching)" (`web-00c`). Good.

**Small things**

- The main button wraps as "Download tailbond for Android (27" with "MB)"
  alone on the second line (`web-00a`). Cosmetic. **Low.**
- When Android finishes installing it shows **Open** and **Done**. I'm used to
  tapping Open. The page says to tap Done. If I tap Open, the app's first
  screen covers it ("Installed tailbond from your home server's page in Chrome?
  Easier: go back to Chrome and tap step 2 there; it fills everything in.",
  `01-onboarding`), but that costs me an extra switch back to Chrome. **Low.**

## Task 2: Add the phone to homebox (pair), from the phone

**Where I tap**

1. Back in Chrome, step 2: **Open tailbond and ask homebox**. The page says
   "Keep Tailscale on until then. Whoever runs homebox taps Approve on
   homebox's page; you'll see it in the app right away."
2. tailbond opens on "Asking homebox… Looking for homebox on your Tailscale
   network." (`02a`), then "Waiting for approval on homebox…" (`02b`): "Ask
   whoever runs homebox to approve "pixel-8"… There it shows as pixel-8 ·
   sam@example.com · 100.101.8.8, so they can tell it's this phone… tailbond
   keeps waiting for 10 minutes and carries on by itself once approved."
3. When it's approved: "Add this phone to homebox?" (`02e`), and I tap **Add
   phone**.

**How it went**

- Straightforward. I never typed anything. The waiting message tells me what
  to tell the person who runs homebox and that I can go to another app to
  message them. Showing my email and 100.x address is how Tailscale itself
  identifies devices, so that made sense to me.
- The web page warned me there would be a second confirmation ("Once approved,
  tailbond shows what it will do: tap Add phone"), so "Add this phone to
  homebox?" after it was already approved didn't surprise me.

**Where I got confused**

- **Not-found and declined screens (`02c`, `02d`).** "Can't find homebox:
  Check that the Tailscale app says Connected, and that the name matches the
  one in its list of machines." That's clear, and **Ask again** is right there.
  Further down, under "Or use the home server's page", the **Open home server
  page** button is greyed out, and the hint under it says "Type the home
  server's name in the box above to open its page here." But the box above
  already says `homebox`. So I've done what it asks and the button still
  doesn't work. When I'm already stuck, a disabled button that contradicts its
  own hint makes me think the app is broken. I'd still get through with Ask
  again or by asking the person who set it up. **Medium.**
- **`02e` "It connects over the internet to home.example.net:443".** I was
  told homebox is on our Tailscale network. Now an address I've never seen,
  not a `.ts.net` one, shows up in the confirmation. "Secure connection to
  homebox" with a lock helps, but I'd still stop for a second and wonder
  whether I'm connecting to the right thing. Nothing tells me this address is
  homebox's own internet address. **Low.**

**Couldn't assess**

- `02-pairing-paste-error`: the screenshot ends at the "Pairing link" field,
  which has a red outline, so I can't see the error text. I'm not counting it
  as a finding.
- `03-deeplink-confirm` is described as "shown on top of the main screen",
  but behind the dialog I see the same "Add this phone" pairing screen as in
  `02e`. As a user I wouldn't notice, so it's not a finding.

## Task 3: Connect

**Where I tap**

1. Main screen, "Disconnected · Stopped. Turn on to reach homebox and your
   Tailscale machines, or switch back to Tailscale below." (`12`). I tap the
   big switch at the top, same place as in Tailscale.
2. "Switch from Tailscale to tailbond?" (`04`): "Android runs one VPN at a
   time. Turning on tailbond turns off Tailscale (or any other VPN) on this
   phone. You still reach your Tailscale machines, like your NAS, through
   homebox." I tap **Turn on tailbond**.
3. Android's own VPN request (not shown). The dialog already said "Tap OK", so
   I tap OK.
4. "Connecting… Reaching homebox…" (`11`), then "Connected · Through homebox ·
   Wi-Fi + mobile data" (`10`).

**How it went**

- It feels like Tailscale: one big switch, then Android's VPN prompt. The
  "Set Tailscale as Always-on VPN? Turn that off first (Android Settings → VPN
  → Tailscale's gear)" line covers the case where tailbond would keep getting
  kicked off. It also tells me how to undo it ("tap Switch back to Tailscale on
  tailbond's main screen") before I've even started. Reassuring.
- "This phone 10.77.0.2: Not a Tailscale address…" told me why my phone no
  longer shows a 100.x address. Without it I'd have thought something was
  wrong.

**Small things**

- **Internet traffic goes through homebox by default.** "Your internet traffic
  goes out from homebox too (like an exit node; Settings changes that)." It's
  said clearly in three places (web step 3, `02e`, `04`), so I wasn't misled.
  But in my Tailscale app exit nodes are something I switch on. Here it's on
  unless I go to Settings → Internet traffic → "Only Tailscale machines". At a
  hotel that means all my browsing goes through home. **Low**, because it's
  clearly disclosed.
- **Header "homebox / via home server · tail1234.ts.net".** The page title is
  the name of a machine, and "homebox via home server" reads like it's
  repeating itself. In Tailscale the top shows my account or tailnet. The first
  time, I briefly thought I was on homebox's detail page. **Low.**

## Task 4: Find and copy the NAS's MagicDNS name or Tailscale IP

**Where I tap**

1. On the main screen I scroll down past Connections, Save mobile data and
   This device to **Machines** (`10-home-connected-full`).
2. I type `nas` into **Search machines**. One row is left: "nas ·
   nas.tail1234.ts.net · 100.101.2.2 · linux". I tap the copy icon and get
   "Copied 100.101.2.2" (`18`).
3. Or I tap the row and get the `nas` page (`19`): "Tap an address to copy it",
   with MagicDNS name, Short name, Tailscale IPv4 and IPv6, each with a copy
   icon. I tap the MagicDNS one and get "Copied nas.tail1234.ts.net".

**How it went**

- This is the part that feels most like Tailscale. Same names, same 100.x
  addresses, green online dots, "Last seen 44 h ago" on the laptop. The detail
  page is labelled even better than Tailscale's ("MagicDNS name", "Short
  name"). "With tailbond on, apps on this phone can use these addresses just
  like with Tailscale." That's what I needed to hear.

**Small things**

- **The machine list is below the fold.** In Tailscale the machine list *is*
  the main screen. Here, on the first screen (`10`), only the "Machines"
  heading peeks in at the bottom. Connection tiles, the data switch and a
  paragraph about "This device" come first. I found it by scrolling. **Low.**
- **The row's copy icon copies the IP.** The row shows the MagicDNS name as
  the larger line and the IP under it. I expected the icon to copy the name.
  The toast "Copied 100.101.2.2" tells me what I got, and both work for the
  task. **Low.**

## Task 5: Are Wi-Fi and 5G both being used right now?

**Where I look**

- Main screen (`10`), no tapping needed: the status card says "Connected ·
  Through homebox · Wi-Fi + mobile data". Under **Connections** it says "Using
  both Wi-Fi and mobile data", and both tiles show a green dot, "In use" and
  live speeds (↓38.4 ↑4.1 Mbps / ↓22.7 ↑2.9 Mbps).
- Tapping into Connections (`20`) shows the combined speed graph and a card per
  connection with delay.
- When one is broken (`15`) the heading changes to "Using Wi-Fi only" and the
  tile shows a red dot with "Not working right now".

**How it went:** answered in a glance, plain words, no networking terms. No
problems.

## Task 6: Make it save mobile data

**Where I tap**

- Main screen: the **Save mobile data** switch, "Use mobile data only when
  Wi-Fi stops working". After that (`16`): "Using Wi-Fi · saving mobile data
  (backup only)", the Mobile data tile says "Not in use · saving data" and the
  mode line says "Mode: Save mobile data".
- If Wi-Fi drops (`17`): "Wi-Fi isn't working · using mobile data", "In use ·
  taking over". So I can see it still keeps me online.

**How it went:** one switch on the home screen, labelled with exactly what I'd
search for. Done.

**Small things**

- **Several data controls on the Connections page (`20`, `21`).** Besides the
  mode list there's a "Use for tailbond" switch on the mobile data card, a
  "Faster switch to backup (experimental)… Uses a little mobile data" switch,
  and a "Mobile data limit" box. The line "The Save mobile data switch on the
  home screen picks 'Save mobile data' here" helps, but I'd wonder whether I
  should also turn off "Use for tailbond" on mobile data to save more, and
  whether that would cost me the backup. **Low.**
- **Numbers that look like they disagree.** The Connections page says "2.05 GB
  this session" at the top and "1.23 GB of 5.00 GB used this month" further
  down. At first I read it as using more this session than all month. Then I
  saw the second one is mobile data only. **Low.**

## Task 7: Hotel Wi-Fi with a sign-in page

**Where I tap**

- The main screen (`16c`) shows a card: "Wi-Fi needs sign-in. This Wi-Fi shows
  a sign-in page before it lets you online. tailbond keeps you connected over
  mobile data meanwhile, and adds Wi-Fi by itself once you're signed in." I tap
  **Sign in to Wi-Fi**, sign in on the hotel page, and wait for Wi-Fi to come
  back into the bond.
- Settings has "Tell me when Wi-Fi needs a sign-in" switched on, with "tailbond
  never bonds over a Wi-Fi until Android says it's online".

**How it went:** better than Tailscale. With Tailscale on, hotel sign-in pages
are where I usually end up turning the VPN off and on. "Android's sign-in page
uses the Wi-Fi directly, so you don't need to turn tailbond off" is the line I
needed. I stay online over mobile data meanwhile, and the tiles show "Needs
sign-in · not used yet" in red.

**Small thing**

- **Two instructions that don't line up.** The small text says "Tap the Wi-Fi
  network (it says "Sign in to network") to open the page", and right under it
  is a button labelled **Sign in to Wi-Fi**. I can't tell whether the button
  opens the hotel page itself, or opens Android's Wi-Fi list where I then tap
  the network. I'd tap the button and work it out. **Low.**

## Task 8: Turn tailbond off and go back to the Tailscale app

**Where I tap**

1. Main screen: the **Switch back to Tailscale** button right under the status
   card (also in Settings: "Turns tailbond off and opens the Tailscale app.
   Turn Tailscale on there if it says Disconnected").
2. In Tailscale I flip its switch if it says Disconnected.
3. If I come back to tailbond (`12b`): "Disconnected · Stopped. Back to
   Tailscale: turn it on in its app." plus a card "Last step: turn Tailscale
   on… Until then, your Tailscale machines aren't reachable." and an **Open
   Tailscale** button.

**How it went:** easy to find. The button has been on every main-screen state
I saw (connected, connecting, off, errors), and the reminder that Tailscale
itself still needs turning on is exactly what I'd forget.

**Small thing**

- **Always-on isn't mentioned on the way back.** Settings has "Keep tailbond
  on all the time (optional)… turn on Always-on VPN for tailbond in Android's
  VPN settings" and "Leave "Block connections without VPN" off if you want to
  switch back to Tailscale easily". But neither "Switch back to Tailscale" nor
  `12b` says to turn Always-on off for tailbond first. The switch-on dialog
  did warn about this for Tailscale (`04`). If I had followed the optional
  advice, I expect Android would keep bringing tailbond back. I can't tell
  from the screenshots whether the app catches this. **Low.**

---

## Other notes

- Dark theme (web and app) reads fine. Contrast is good on the dialogs, the
  connected card and the settings.
- The error screens I saw (`13` "Can't reach your home server", `14` "Can't
  confirm this is your home server") tell me what to do and who to show. "Don't
  pair this phone again unless they ask you to" is a good safety line. Neither
  is part of a task.

## Findings

| # | Task | Finding | Severity |
|---|---|---|---|
| 1 | 1 | Download button wraps as "Download tailbond for Android (27" / "MB)" | low |
| 2 | 1 | Android's install-done dialog offers Open; the page says tap Done. Tapping Open (the natural choice) lands on onboarding, which sends me back to Chrome: an extra switch | low |
| 3 | 2 | `02c`/`02d`: "Open home server page" is greyed out while its hint says "Type the home server's name in the box above to open its page here", but the box already contains `homebox` | medium |
| 4 | 2 | `02e` "It connects over the internet to home.example.net:443": an unfamiliar non-Tailscale address appears without saying it's homebox's own internet address | low |
| 5 | 3 | Internet traffic goes out from homebox "like an exit node" by default. Disclosed three times, but it's opt-out where Tailscale's exit node is opt-in | low |
| 6 | 3 | Header "homebox / via home server · tail1234.ts.net" repeats itself and reads like a machine's detail page, not the app's home | low |
| 7 | 4 | The Machines list is below the fold, under connections, the data switch and This device. Tailscale leads with machines | low |
| 8 | 4 | The row copy icon copies the IP ("Copied 100.101.2.2") although the MagicDNS name is the larger line | low |
| 9 | 6 | Overlapping data controls on Connections ("Use for tailbond" on mobile data, "Faster switch to backup (experimental)", Mobile data limit) make me wonder whether the home switch is enough | low |
| 10 | 6 | "2.05 GB this session" vs "1.23 GB of 5.00 GB used this month" look like they disagree until I notice the second is mobile only | low |
| 11 | 7 | "Tap the Wi-Fi network (it says "Sign in to network")" vs the **Sign in to Wi-Fi** button: unclear which to use, or whether the button opens the page or the Wi-Fi list | low |
| 12 | 8 | "Switch back to Tailscale" and `12b` don't mention turning off Always-on for tailbond, which Settings suggests as optional | low |

Totals: 0 high / 1 medium / 11 low
