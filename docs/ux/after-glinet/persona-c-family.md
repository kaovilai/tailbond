# Persona C (less-technical family member): raw report, after-glinet screens

Who I am: someone else set up the home server and Tailscale for me. All I was
told is that the home server's Tailscale name is "homebox". I use the Tailscale
app now and it works. I don't know words like "port", "exit node" or "Flint".

Screens used: every PNG in `out/ux/after-glinet/` (light versions), with
`docs/ux/screens.md` for what each one shows.

## Task 1: Install tailbond and pair it with the home server, from the phone

### Install

- **Where I'd tap:** I'd open the Play Store and search "tailbond". No
  screenshot shows a Play Store listing. The only install path shown is the
  home server's web page (`web-01-add-this-phone`): "1 Get the tailbond app ...
  Download tailbond for Android", with "Android warns about apps from outside
  the Play Store and asks you to allow installs from your browser". So the app
  isn't on the Play Store, and it only comes from that page.
- **Where I'd get stuck:** that page is at `http://<server>.<tailnet>.ts.net:8077`
  (screens.md). Nobody told me that address, and the phone never shows it to me
  before I have the app. The address first appears inside tailbond, on the
  waiting screen (`02b`: "http://homebox:8077"). That's too late. Knowing only
  "homebox", I wouldn't think to type `homebox:8077` into Chrome. I'd be stuck
  and have to text the person who set it up.
- **Expected:** the Tailscale app came from the Play Store. I'd expect the same,
  or at least one thing I can find myself, like the address on the Tailscale
  machine's page.
- **Severity: high.** I can't do the install part of the task on my own.

If the person had sent me the page link: `web-01` is clear. It shows two
numbered steps, the phone name is filled in ("pixel-8"), and there's a big
"+ Add this phone" button. "(skip if it's installed)" is helpful. The warning
about installs from outside the Play Store is honest, and "Ask whoever set up
the server to install it for you" gives me a way out. After the tap
(`web-02-after-tap`): "Opening tailbond… In the app, tap Add phone, then turn
tailbond on." Then the app shows "Add this phone to homebox?" (`03`) with
"Add phone". That's easy. One thing bothered me: the red bar "Setup health: 1
thing to improve for best speed. Show" is the scariest thing on the page. It
isn't meant for me, but the red makes me think something is wrong. (low)

### Pair (the app is installed, I only know "homebox")

1. `01-onboarding`: I read "Use tailbond instead of Tailscale on this phone".
   The small print "Next you'll type your home server's name (the one in the
   Tailscale app) and ask it to connect. If someone else set it up, they
   approve it." describes my situation exactly, which is reassuring. I tap
   **Get started**.
2. `02-pairing`: The first card, "Ask your home server", is the one for me:
   "Keep Tailscale on for now. Type your home server's name as it appears in
   the Tailscale app, and tap Ask to connect." I type `homebox` in the box that
   says "Home server's name in Tailscale" and tap **Ask to connect**. I can
   ignore the other three cards ("Or use the home server's page", "From another
   screen", "Have a link?").
   - Small doubt: under "Or use the home server's page" the **Open home server
     page** button is greyed out, and it says "Came here from your home
     server's page? Go back to it". I didn't come from there, and I can't tap
     it, so I'm not sure why it's shown. (low)
3. `02b-pairing-ask-waiting`: "Waiting for approval on homebox… Ask whoever runs
   homebox to open its tailbond page (http://homebox:8077 on a computer, or a
   Flint 3's admin panel → VPN → tailbond) and tap Approve next to "pixel-8".
   Keep this screen open; tailbond continues by itself once approved."
   - I'd screenshot this and text it to whoever set it up. I don't understand
     "Flint 3's admin panel → VPN → tailbond" and don't know if we have one. It
     works fine as a message to pass on, though. There's no "send this to
     them" button, so screenshotting is my only way to pass it on. (low)
   - "Keep this screen open; tailbond continues by itself" is clear about what
     I should do while I wait.
4. `02e-pairing-ask-approved-confirm`: "Add this phone to homebox?" I tap **Add
   phone**. The line "It connects over the internet to home.example.net:443"
   names a different address than homebox, and for a second I'd wonder if
   that's right. The padlock line "Secure connection to homebox" calms me down.
   (low)

When it goes wrong:
- `02c-pairing-ask-not-found`: "Can't find homebox. Check that the Tailscale app
  says Connected, and that the name matches the one in its list of machines."
  The advice is good. But the name box now looks empty (it shows the grey
  placeholder "Home server's name in Tailscale"), and **Ask again** is greyed
  out. I'd tap "Ask again", nothing would happen, and I couldn't see what I had
  typed to check the spelling, even though the message tells me to check it. I
  only worked out that I had to retype the name after some fiddling.
  **Severity: medium.**
- `02d-pairing-ask-declined`: "homebox said no. Whoever runs homebox declined
  this phone. Ask them, then ask again." This is plain and clear. The same
  greyed "Ask again" and empty-looking box appear here too. (covered above)
- `02-pairing-paste-error`: "That isn't a complete tailbond link. Copy the whole
  link starting with tailbond://pair". This is clear, and I wouldn't use that
  path anyway.

**Answer to "could I find and pair with the home server from the phone
alone?"** Pairing: yes. Once the app is open, typing "homebox" and tapping "Ask
to connect" is the obvious path, and the only outside step is the approval I
was told to expect. Installing: no. Nothing I can reach on the phone tells me
where to get the app or what the home server's page address is.

## Task 2: Connect

- **Where I'd tap:** On the main screen (`12-home-disconnected`), the card says
  "Disconnected. Stopped. Turn on to reach homebox and your Tailscale machines".
  I tap the switch on the right.
- `04-handoff-other-vpn`: "Switch from Tailscale to tailbond?" with "You still
  reach your Tailscale machines, like your NAS, through homebox." and "Next,
  Android asks you to allow tailbond's connection request. Tap OK." I tap
  **Turn on tailbond**, then OK on Android's prompt. It's clear, and it tells
  me how to go back ("tap Switch back to Tailscale on the home screen"). I see
  the same pattern the first time in `05-handoff-first-permission` ("Allow
  tailbond's VPN").
- `11-home-connecting`: "Connecting… Reaching homebox…", then
  `10-home-connected`: "Connected. Through homebox · Wi-Fi + mobile data".
- **Expected vs Tailscale:** it's the same big-switch idea as Tailscale, so it
  feels familiar. No problems.
- Errors: `13-error-unreachable` says "the home router may need to forward UDP
  port 443 to it: if someone else set up your home server, show them this
  message." I don't know what UDP is, but "show them this message" tells me
  what to do. `14-error-cert` says "Don't pair this phone again unless they ask
  you to." That's a clear do/don't. (low)

## Task 3: Find and copy the NAS's MagicDNS name or Tailscale IP

- **Where I'd tap:** On the main screen I scroll to "Machines" and type `nas` in
  "Search machines" (`18-machines-search-copied`). The row shows "nas /
  nas.tail1234.ts.net / 100.101.2.2 · linux". I tap the copy icon and see
  "Copied 100.101.2.2". Done.
- If I want the name instead, I tap the row and get `19-machine-detail`, which
  says "Tap an address to copy it", with labels "MagicDNS name
  nas.tail1234.ts.net", "Short name", "Tailscale IPv4", "Tailscale IPv6". I tap
  the first one and see "Copied nas.tail1234.ts.net".
- **Expected vs Tailscale:** this looks like Tailscale's machine list, which is
  good. The copy icon on the list copies the number, not the name, and nothing
  tells me that until the "Copied …" message appears. It's fine for this task.
  (low)
- Small oddity: "This device: pixel-8 (this phone) 10.77.0.2". In Tailscale my
  phone had a 100.x number. Here it's a different kind of number, but the line
  "Your Tailscale machines can reach this phone at this address, through
  homebox" explains it well enough. (low)

## Task 4: Are Wi-Fi and 5G both being used right now?

- **Where I'd look:** The main screen (`10-home-connected`) says "Connected.
  Through homebox · Wi-Fi + mobile data", and under Connections it says "Using
  both Wi-Fi and mobile data". The Wi-Fi tile says "In use ↓38.4 ↑4.1 Mbps" and
  the Mobile data tile says "In use ↓22.7 ↑2.9 Mbps". The answer is yes, and
  it's obvious.
- It also says "5G" nowhere, only "Mobile data". That's fine, it's what my
  phone calls it.
- Other states are just as clear: `15-one-path-failed` says "Using Wi-Fi only"
  with "Mobile data: Not working right now" in red, and `16c` says "Wi-Fi needs
  sign-in · using mobile data" with a "Sign in to Wi-Fi" button. No problems.

## Task 5: Make it save cellular data

- **Where I'd tap:** The main screen has a switch "Save mobile data. Use mobile
  data only when Wi-Fi stops working". I flip it. `16-save-data-standby` then
  shows "Using Wi-Fi · saving mobile data (backup only)" and "Mobile data: Not
  in use · saving data". That's exactly what I wanted.
- `17-save-data-takeover` shows what happens if the Wi-Fi drops: "Wi-Fi isn't
  working · using mobile data", "In use · taking over". That's reassuring.
- If I tap "More options" (`20`/`21-connections`), there are more things that
  sound like data saving: a "Use for tailbond" switch on Mobile data, a "Mobile
  data limit 1.23 GB of 5.00 GB", and "Faster switch to backup (experimental)
  ... Uses a little mobile data". The page says "The Save mobile data switch on
  the home screen picks "Save mobile data" here." and the limit is "Separate
  from Save mobile data", so I'd leave it alone. It's a bit much, but it didn't
  confuse me about the main switch. (low)
- **Expected vs Tailscale:** Tailscale has no such setting. The home-screen
  switch is easier than I expected.

## Task 6: Turn tailbond off and go back to the Tailscale app

- **Where I'd tap:** The **Switch back to Tailscale** button sits right under
  the big card on every main screen. It's also in Settings (`30-settings`):
  "Switch back to Tailscale. Turns tailbond off and opens the Tailscale app."
- After tapping (`12b-home-switched-back-to-tailscale`): "Disconnected. Stopped.
  Tailscale is in charge on this phone." and "tailbond stays off until you turn
  it on again. In the Tailscale app, turn on its switch if it says
  Disconnected." I tap **Open Tailscale** and turn its switch on if needed.
  This is clear.
- **Expected vs Tailscale:** in Tailscale you just flip its switch. Here the one
  extra step (turning Tailscale back on) is spelled out for me. No problems.

## Findings

| # | task | finding | severity |
|---|---|---|---|
| 1 | 1 (install) | With only the name "homebox", I can't find where to get the app. It isn't on the Play Store (`web-01`: "apps from outside the Play Store"), and the home server page address (`homebox:8077`) first appears inside the app (`02b`), so I'd have to ask the person who set it up. | high |
| 2 | 1 (pair) | After "Can't find homebox" / "homebox said no" (`02c`, `02d`), the name box looks empty and "Ask again" is greyed out. Tapping it does nothing, and I can't check the spelling the message asks me to check. | medium |
| 3 | 1 (pair) | The waiting text "a Flint 3's admin panel → VPN → tailbond" means nothing to me. It works as a message to forward, but there's no share/send button, so I have to take a screenshot. | low |
| 4 | 1 (pair) | "Open home server page" is greyed out with "Came here from your home server's page? Go back to it". It's a visible option I can't use and didn't ask about. | low |
| 5 | 1 (pair) | The confirm dialog names "home.example.net:443", not homebox. It made me hesitate for a moment, until "Secure connection to homebox" reassured me. | low |
| 6 | 1 (web) | The red "Setup health: 1 thing to improve for best speed" banner on the add-phone page looks alarming to someone who can't act on it. | low |
| 7 | 2 | The error texts use "UDP port 443", but they say "show them this message", so I know what to do. | low |
| 8 | 3 | The list copy icon copies the IP, not the name, and nothing says which until the "Copied …" message appears. | low |
| 9 | 3 | This phone shows "10.77.0.2", not a 100.x address like in Tailscale. The note below the address explains it. | low |
| 10 | 5 | The Connections page has several data-related controls (mode, Use for tailbond, monthly limit, experimental faster switch). The home-screen switch is enough, so the extras are clutter. | low |
