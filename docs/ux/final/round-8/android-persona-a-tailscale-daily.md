# Round 8 — Android persona A (daily Tailscale user)

**Persona:** I use the official Tailscale Android app every day. I know MagicDNS names
(`nas.tail1234.ts.net`), the machine list, the VPN toggle, and exit nodes at a basic level.
I don't work in networking. Today only the Tailscale app is on the phone, and it's connected.

**Materials:** `task-script.md`, `screens.md`, and every PNG in `web/` and `android/`
(light and dark). I didn't open any other file.

**Overall:** I could do all eight tasks from what's on screen. Nothing stopped me and
nothing led me into a wrong action. The words match what I know from Tailscale: "MagicDNS
name", "Tailscale IPv4", "like using an exit node", "Switch back to Tailscale". What
confused me was a few places where two things share a name, or where a small line of
help text doesn't quite match the button next to it. Dark theme reads fine on every
screen I checked (web-00-dark, 01, 04, 10, 12, 19, 20, 30).

---

## Task 1 — Get the tailbond app onto the phone

**Where I tap:** With Tailscale on, I type `homebox/` in Chrome. That opens
**web-00a** ("Get tailbond on this phone"). I tap **"Download tailbond for Android (27 MB)"**.
The page draws Android's prompts ahead of time (web-00), so I know what's coming:
"File might be harmful" → **Download anyway**, "For your security, your phone currently
isn't allowed to install unknown apps…" → **Settings**, then **Allow from this source** →
back, then "Do you want to install this app?" → **Install**.

**What went well:**
- "That's expected" ahead of the scary "File might be harmful" dialog is exactly what I
  needed. Without it I would have stopped at Chrome's warning.
- The drawn copies of the dialogs, with the right button circled, match the real ones
  well enough to follow.
- "Chrome's Open message gone before you tapped it? … Chrome's menu ⋮ → Downloads" covers
  the thing I always lose.
- I scrolled and tapped step 2 before installing (it's a big green button right there).
  web-00b caught that: "tailbond isn't on this phone yet, so step 2 brought you back
  here. Install it first, then tap step 2 again."

**Where I hesitate:**
- The page says "When it says App installed, tap **Done** and come back to this page".
  Android's "App installed" screen also has **Open**, and that's what I'd normally tap.
  Tapping Open puts me on 01-onboarding ("Use tailbond instead of Tailscale on this
  phone" → "Get started"), which goes a different way: "Next you'll type your home
  server's name". It still works, because I know the name is `homebox` and 02-pairing
  has a box for it. But for a moment I'm off the path the page described. **Low.**
- The SHA-256 line means nothing to me. It's grey and small, so I skip it. Not a problem.

## Task 2 — Add the phone to homebox (pair), from the phone

**Where I tap:** Back in Chrome I tap **"Open tailbond and ask homebox"** (web-00, step 2).
tailbond opens on **02a** ("Asking homebox… Looking for homebox on your Tailscale
network."), then **02b** ("Waiting for approval on homebox…"). I message the person who
runs homebox. Once they approve, **02e** asks "Add this phone to homebox?" and I tap
**Add phone**.

**What went well:**
- 02b tells me exactly what to tell the admin: "There it shows as pixel-8 ·
  sam@example.com · 100.101.8.8". It also says "You can switch to another app to message
  them; tailbond keeps waiting for 10 minutes". That's what I'd want to know.
- The two failure screens are plain: 02c "Can't find homebox — Check that the Tailscale
  app says Connected, and that the name matches the one in its list of machines" and 02d
  "homebox said no — Whoever runs homebox declined this phone." Both have **Ask again**.
- 02-pairing-paste-error ("That isn't a complete tailbond link. Copy the whole link
  starting with tailbond://pair") is clear.

**Where I get confused:**
- **"Home server's page" means two different pages.** On 02-pairing, the card "Or use
  the home server's page" says "Came here from your home server's page? Go back to it in
  your browser and tap step 2". That's the `homebox/` install page. The next card, "From
  another screen", says "If the home server's page is open on a computer, tap **Add a
  device** there and scan the code it shows." I open `homebox/` on my computer, since
  that's the address I was given, and get web-00c. It has no "Add a device" button. It
  shows a QR code labelled "point the phone's camera at this code", which only opens the
  install page. So I'd likely tap **"Scan code from your home server"** in the app and
  scan the wrong QR code. The page that does have "Add a device" seems to be the one at
  `http://homebox:8077`, but the app never says which address it means. In the
  Tailscale app, "add a device" lives in one place. **Medium.**
- On 02-pairing, the **"Open home server page"** button is greyed out and nothing says
  why. The text above it tells me to go back to the browser, but the button that looks
  like it would take me there doesn't work. **Low.**
- 02e says "It connects over the internet to **home.example.net:443**". I only know my
  server as `homebox` / `homebox.tail1234.ts.net`. Seeing a different name right on the
  "Add this phone?" screen makes me wonder for a second whether this is the right server.
  The lock line "Secure connection to homebox" reassures me, and the phone page did say
  "connects straight to homebox over the internet (not through Tailscale)". **Low.**

## Task 3 — Connect

**Where I tap:** On the home screen (12-home-disconnected) I tap the big switch next to
"Disconnected". Because Tailscale's VPN is on, I get **04-handoff-other-vpn**: "Switch
from Tailscale to tailbond?" → **Turn on tailbond**. Then Android's VPN prompt (not shown),
where I tap OK. I see **11-home-connecting** ("Connecting… Reaching homebox…") and then
**10-home-connected** ("Connected — Through homebox · Wi-Fi + mobile data").

**What went well:**
- It's the same idea as Tailscale's toggle. The handoff dialog explains the one-VPN rule
  before Android's own prompt shows up, and "You still reach your Tailscale machines, like
  your NAS, through homebox" answers my main worry.
- The first-time version (05, "Allow tailbond's VPN") says the same thing in the same
  words.
- The failure screens (13 "Can't reach your home server", 14 "Can't confirm this is your
  home server") each say what to do and who to show. The "forward UDP port 443" line is
  over my head, but it says "show them this message", so that's fine.

**Where I hesitate:**
- "To go back later, tap Switch back to Tailscale on the **home screen**" (04, 05). On
  Android, "home screen" means the launcher. Once I've seen the app it's obvious they mean
  tailbond's main screen, but the wording could trip someone up. **Low.**

## Task 4 — Find and copy the NAS's MagicDNS name or Tailscale IP

**Where I tap:** On the home screen I scroll to **Machines**, type `nas` in **"Search
machines"**, and tap the copy icon on the row (**18**). A toast says "Copied 100.101.2.2".
Or I tap the row and get **19-machine-detail**, with labelled fields "MagicDNS name
nas.tail1234.ts.net", "Short name", "Tailscale IPv4", "Tailscale IPv6" and "Tap an
address to copy it". The toast says "Copied nas.tail1234.ts.net".

**What went well:** This is the best screen in the app for me. It looks and reads like
Tailscale's machine detail, and the labels are the words I already use. Green dot plus
"Online · linux" is familiar.

**Where I hesitate:**
- On the list row the big text is the MagicDNS name (`nas.tail1234.ts.net`), but the copy
  icon next to it copies the **IP** ("Copied 100.101.2.2"). I expected it to copy the name
  I was looking at. The toast tells me what I got, and the detail screen lets me pick, so
  I can fix it. **Low.**
- "This device: pixel-8 (this phone) **10.77.0.2** — Not a Tailscale address: to your
  Tailscale machines this phone's traffic comes from homebox, under its access rules…"
  During pairing the phone was shown as 100.101.8.8, and now it has a 10.x address. The
  explanation is right there, but it's dense. I still can't tell if my other devices can
  reach this phone by `pixel-8`. Not part of a task. **Low.**

## Task 5 — Tell whether Wi-Fi and 5G are both being used right now

**Where I look:** **10-home-connected**: the header says "Through homebox · Wi-Fi + mobile
data". Under Connections: "**Using both Wi-Fi and mobile data**", with two tiles, each
showing a green dot, "In use" and its speed (↓38.4 ↑4.1 Mbps / ↓22.7 ↑2.9 Mbps). Tapping
the Connections area gives **20-connections** with live graphs and "delay 21 ms / 48 ms".

**What went well:** I get the answer in a second, and it's said three ways (header,
sentence, tiles). The one-broken case (15, "Using Wi-Fi only", red dot "Not working right
now") is just as clear. I noticed the app says "mobile data" and never "5G". That's fine,
it's what Android calls it.

No findings.

## Task 6 — Make it save mobile data

**Where I tap:** On the home screen, the **"Save mobile data — Use mobile data only when
Wi-Fi stops working"** switch (10). After flipping it, 16 shows "Mode: Save mobile data",
"Using Wi-Fi · saving mobile data (backup only)", and the Mobile data tile reads "Not in
use · saving data". When Wi-Fi drops (17), it says "Wi-Fi isn't working · using mobile
data" and the mobile tile reads "In use · taking over".

**What went well:** It's one switch on the main screen, the label says what it does, and
the state shows right away. Easier than I expected.

**Where I hesitate:**
- If I go to **More options** (20/21) first, I see four ways to cut mobile data: the
  "How to use your connections" radio list (whose "Save mobile data" option is the same
  as the home switch, as it says), a **"Use for tailbond"** switch on the Mobile data
  card, a **"Mobile data limit"** box (already set to "5"), and on 21 "Faster switch to
  backup (experimental)". Each has a line of explanation ("Separate from Save mobile
  data…", "Turning this off doesn't turn off the phone's mobile data"), so I wouldn't do
  anything wrong. It's still a lot of overlapping options for "use less data". **Low.**

## Task 7 — Hotel Wi-Fi with a sign-in page: get it working and back into the bond

**Where I tap:** **16c-home-wifi-needs-sign-in** shows a card, "Wi-Fi needs sign-in — This
Wi-Fi shows a sign-in page before it lets you online. tailbond keeps you connected over
mobile data meanwhile, and adds Wi-Fi by itself once you're signed in." I tap **"Sign in
to Wi-Fi"**, sign in on Android's page (not shown), and come back. The Wi-Fi tile says
"Needs sign-in · not used yet" until then.

**What went well:** My habit at hotels is to turn the VPN off. This screen tells me not
to: "Android's sign-in page uses the Wi-Fi directly, so you don't need to turn tailbond
off." And it promises Wi-Fi gets added back on its own, so I don't need to do anything
after signing in.

**Where I hesitate:**
- The small print says "**Tap the Wi-Fi network** (it says "Sign in to network") to open
  the page." But right below it is a button, **"Sign in to Wi-Fi"**. I can't tell if
  "tap the Wi-Fi network" means the button, the Android notification, or the network in
  Wi-Fi settings. I'd tap the button, but the sentence makes me unsure whether the
  button is enough. **Low.**
- I can't see a screen that confirms Wi-Fi rejoined after sign-in. I'd expect to end up
  on something like 10, with both tiles "In use". The text promises it, so I'm not
  counting this as a problem.

## Task 8 — Turn tailbond off and go back to the Tailscale app

**Where I tap:** **"Switch back to Tailscale"**, the outlined button under the status card
on every home screen (10, 11, 12, 13, 14, 15, 16, 17, 16c). It's also in Settings (30):
"Turns tailbond off and opens the Tailscale app. Turn Tailscale on there if it says
Disconnected". Result (**12b**): "Disconnected — Stopped. Tailscale is in charge on this
phone", with a card "Tailscale is in charge on this phone… In the Tailscale app, turn on
its switch if it says Disconnected" and **Open Tailscale**.

**What went well:** It's always visible, it's named after what I want to do, and it tells
me the one extra step (turn Tailscale's switch back on). That matches how the Tailscale
app behaves when another VPN has taken over.

**Where I hesitate:**
- On 12b, the "This device" row still says "This phone reaches your tailnet through
  homebox" right under "Tailscale is in charge on this phone". The two lines disagree.
  Right now Tailscale is how I reach the tailnet. **Low.**

---

## Findings

| # | Task | Finding | Severity |
|---|---|---|---|
| 1 | 2 | "Home server's page" names two different pages. "From another screen" says "tap **Add a device** there and scan the code it shows", but the computer page at the address I was given (`homebox/`, web-00c) has no "Add a device" and shows a different QR code (install only). That invites scanning the wrong code. | medium |
| 2 | 1 | Page says "tap Done and come back to this page". Tapping Android's usual **Open** instead lands on onboarding ("Next you'll type your home server's name"), a different pairing path from the one the page described. | low |
| 3 | 2 | "Open home server page" button on 02-pairing is greyed out with no explanation, right under text telling me to go back to that page. | low |
| 4 | 2 | 02e "It connects over the internet to home.example.net:443": an unfamiliar name on the confirm screen when I only know `homebox`. Causes a moment of doubt. | low |
| 5 | 3 | "tap Switch back to Tailscale on the **home screen**" (04/05): "home screen" can be read as Android's launcher. | low |
| 6 | 4 | The machine-list copy icon sits next to the MagicDNS name but copies the IP ("Copied 100.101.2.2"). | low |
| 7 | 4 | "10.77.0.2 — Not a Tailscale address…" is dense, and it differs from the 100.101.8.8 shown during pairing. Unclear whether other devices can still reach the phone by name. | low |
| 8 | 6 | Connections page has four overlapping data controls (mode list, "Use for tailbond" switch, "Mobile data limit", experimental backup switch). All explained, but busy. | low |
| 9 | 7 | "Tap the Wi-Fi network (it says "Sign in to network")" vs the "Sign in to Wi-Fi" button: unclear which one to tap. | low |
| 10 | 8 | 12b "This phone reaches your tailnet through homebox" contradicts "Tailscale is in charge on this phone". | low |

Totals: 0 high / 1 medium / 9 low
