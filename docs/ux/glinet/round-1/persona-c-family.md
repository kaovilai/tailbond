# Persona C (less-technical family member): GL.iNet round 1

**Who I am:** my nephew set up the home server and Tailscale for me. I can follow clear
steps on a screen, and I've used the GL router's admin page a couple of times to join hotel
Wi-Fi. "Subnet", "tunnel", "MagicDNS", "route" and "port" don't mean much to me. My main
worry is breaking the internet, either at the hotel or at home.

I mostly looked at the phone screenshots, plus the desktop versions of 01 and 05.

---

## Task 1: Install tailbond on the Slate 7 without SSH

**Where I'd click:** going by the manifest, GL admin, then SYSTEM, then Advanced Settings,
then "Go To LuCI", then LuCI, System, Software, "Upload Package…". After that I'd pick the
file, see the "Manually install package" box (`01-not-installed-luci-upload-*`) and press
**Install**.

**Where I'd get stuck or confused:**
- **Getting to LuCI.** None of the screenshots show the way there. It's only in the
  manifest. The desktop screenshot shows a page that looks nothing like the GL admin I know
  (dark blue bar with "Status System Services Network Logout", a "Software" heading, "Free
  space: 61% (153 MiB)", "Update lists…"). From the GL panel alone I would never have
  thought to go into "Advanced Settings" and then "Go To LuCI". It feels like I'm in the
  part of the router I was told not to touch. I'd have to phone my nephew.
- **"Allow overwriting conflicting package files"** is ticked and greyed out. "Overwriting"
  and "conflicting" sound like something could get broken. Nothing on screen says this is
  normal and safe.
- The dialog says the file "ends in **_aarch64_cortex-a53_neon-vfpv4.ipk**". I checked
  letter by letter that my file matched, and it did. It's still gibberish to me, but at
  least I could compare it.
- **After I press Install**, no screen shows whether it worked or what to do next. Nothing
  tells me to go back to the normal GL admin and open APPLICATIONS → tailbond. I only knew
  that from the manifest.
- On the phone screenshot the dialog covers the whole page, so I can't see where I am in
  LuCI. That's fine for this step.

**What I expected:** an install button inside the normal GL admin (for example
APPLICATIONS → Plug-ins → "Upload"), or at least on-screen steps for reaching LuCI and a
message like "Installed! Now go to APPLICATIONS → tailbond."

## Task 2: Pair the Slate 7 with the home server

**Where I'd click:** APPLICATIONS → tailbond (`02-just-installed`). It says "Looking for
your home server on your tailnet…". A few seconds later (`03-found-server`) a big box says
"Found your home server: homebox" with a big blue **Ask to connect** button. I'd press that.

**Where I'd get stuck or confused:**
- This part is really nice. I don't have to type anything. The word "tailnet" ("must be on
  the same tailnet as this router") means nothing to me, but I can still go on.
- After I tap it (`04-waiting-approval`), the screen says "Waiting for approval on
  homebox… Open homebox's tailbond page (on a Flint 3: its admin panel → APPLICATIONS →
  tailbond) and tap **Approve** next to **slate7**."
  - For a Flint 3, I understand that.
  - **If my home server is the Fedora box, I don't know what "homebox's tailbond page"
    is or how to open it.** No address is given. The manifest says it's
    `http://homebox:8077`, but the Slate's screen never tells me that. I'd be stuck
    waiting and have to phone my nephew.
  - It also doesn't say I need to be at home, or on my home network, to open the Flint's
    admin panel. If I'm already at the hotel, can I still do this?
- The footer link "Is this the router at home? **Use it as your home server instead**"
  worries me a bit. It's blue, it's a link, and I could tap it by accident. I'd like to know
  whether that's easy to undo.
- `03b-tailscale-off`: "Turn on Tailscale first … Open Tailscale settings / Search again".
  That's clear and I'd manage it.
- `03c-no-server-found`: "Checked 4 online machines. On the home server, tailbond must be
  installed and running, and Tailscale on." I couldn't check that myself, but at least it's
  clear the problem is at home and not something I broke.

**What I expected:** the waiting screen should give the Fedora page's address (a link or
something I can copy), not only the Flint 3 menu path.

## Task 3: Use hotel Wi-Fi and phone tethering, and confirm both are used

**Where I'd click:** I'd join the hotel Wi-Fi the usual GL way (repeater), plug the phone
in, and open APPLICATIONS → tailbond (`05-connected-two-wans`).

**What I saw:** the top says "Connected to homebox. Using Wi-Fi (repeater) + phone (USB
tethering)". Both **Repeater** and **Tethering** have a green "In use" badge. That's clear,
and it's the best screen in the set for me.

**Small confusions:**
- Ethernet says "Not connected", but its "Use" switch is still blue/on. Should I turn it
  off? (I'd leave it.)
- "↓ 38 Mbps ↑ 4.2 Mbps · 24 ms" doesn't mean much to me, but it doesn't get in the way.
- `06-one-wan-failed`: "phone (USB tethering) not working right now … tailbond keeps
  trying; the other connections carry your traffic." That's reassuring, and the red "Not
  working right now" badge is easy to spot.

## Task 4: Phone as backup only

**Where I'd click:** on the **Tethering** card, I'd tick **Backup only**
(`07-save-data-standby`). The badge changes to "Backup · saving data" with "Used only if
your other connections stop working." That's exactly what I wanted.

**Where I'd get stuck or confused:**
- **There are two ways to do this and I can't tell which is right.** Below the cards,
  "How to use your connections" has a **Save mobile data** option ("Uses Ethernet and
  Wi-Fi; your phone and cellular only when those stop working"). That sounds like the same
  thing. After I ticked "Backup only", the top choice still says **Fastest** ("Adds all
  your connections together for the most speed"). That looks like it contradicts my
  backup choice. Is my phone being saved or added together for speed? I'd probably click
  "Save mobile data" as well, just to be safe, and still not know whether I needed to.
- On each card there's a **Backup only** box and a **Use** switch side by side. I wasn't
  sure at first whether "backup" meant switching "Use" off. If I had switched "Use" off,
  the phone wouldn't be a backup at all. The box worked, but the two controls next to each
  other made me hesitate.
- There's no "Save" button for the Backup only box. I'm trusting that ticking it took
  effect because the badge changed.
- **Change limit** (`07b-set-limit`): "5 GB a month / Save / Cancel … At the limit it
  pauses until next month. 0 = no limit." That's clear.

## Task 5: Reach the NAS from the laptop and find its name

**Where I'd click:** I'd scroll down to **Tailnet machines** (`05`). It says "Laptops and
phones on this router's Wi-Fi or LAN reach these by name, e.g. nas.tail1234.ts.net." The
first row is **nas**, `nas.tail1234.ts.net · 100.101.2.2`, with a **Copy name** button.
I'd press Copy name and paste it into the browser.

**Where I'd get stuck or confused:**
- I found the name easily. I didn't need to know the word "MagicDNS".
- In the **Home server** box there's a line "Names like nas….ts.net: Work on every device
  connected to this router while tailbond is on". The "nas….ts.net" with the dots cut out
  looks like a broken or half-typed name. It's also a strange place for a sentence that
  belongs with the Tailnet machines list.
- On the phone the address wraps badly ("100.101." on one line, "2.2" on the next), so if
  I copied it by hand I'd get it wrong. The Copy name button saves me here.
- It doesn't say whether I can type just "nas", or whether I need "http://" in front.
- When tailbond is off or can't reach home (`08`, `09`), the list says "The list appears
  once tailbond is connected." Fine.

## Task 6: Turn tailbond off

**Where I'd click:** the big blue switch at the top right of the status card. Afterwards
(`09-turned-off`) it says "Off. Devices on this router use the internet directly, without
your home server." That's exactly the reassurance I need: my internet keeps working.

**Where I'd get stuck or confused:**
- The switch has no label. I worked out it was the on/off switch, but the red **Unpair**
  button further down also looks like a way to "turn it off". If I'd tapped Unpair, the
  dialog (`09b-unpair-confirm`) would have stopped me: "This router stops using tailbond
  and forgets the home server … it will need to approve". Good. I'd press Cancel.
- "Pair again" sits next to "Unpair" with no explanation. I might tap it when something
  looks wrong, without knowing what it does.
- `08-server-unreachable`: the top says "Can't reach homebox" and the switch is still blue
  (on). **It doesn't say whether my laptop still has internet right now.** That's exactly
  what I'd worry about at a hotel. Should I switch tailbond off to get online? The text
  "This router is online, but homebox isn't answering … tailbond keeps trying" doesn't
  answer it. "Show this to whoever set up homebox: UDP port 443 must reach…" is a nice
  touch, because I'd forward it to my nephew.
- `10-not-running`: "Starting it is safe: devices here keep their internet while you set
  it up." That's the kind of sentence I like.

## Task 7: Approve the travel router at home

**Flint 3 (`11-server-pending-request`):** at the top it says "Waiting for your approval ·
1 new", "slate7 wants to connect", with **Decline** and a big blue **Approve**. I'd tap
Approve. Afterwards (`12-server-approved`) slate7 is in Devices: "Connected just now".
That's easy.

**Fedora (`14-fedora-dashboard-request`):** the Approve card is at the top. That's fine.
Right under it is a big green "Add this phone … tailbond takes over from Tailscale on this
phone" box with a huge **+ Add this phone** button. I opened the page on my phone to
approve the router, and now it wants to change my phone too. I could easily press the big
green button by mistake. It also says "takes over from Tailscale on this phone", which
sounds like it would change something my nephew set up.

**Flint freshly set up (`13-server-route-pending`):** this one would scare me.
- The top says "Home server is on. Ready for travel routers and phones" in green.
- Further down are two orange warnings: "One step left in the Tailscale admin console:
  Approve the route **10.77.0.0/24** for homebox (Machines → homebox → Edit route
  settings)" and "Set this router's internet address … public IP or a dynamic DNS name".
  I don't know what a route, a public IP or a dynamic DNS name is.
- So is it ready or not? The green top contradicts the warnings below. I would not touch
  "Change" or "Open admin console" because I'd be afraid of breaking the home internet.
  I'd call my nephew. To be fair, this is the setup person's job, not mine.

---

## Findings

| # | task | finding | severity |
|---|---|---|---|
| 1 | 1 | Nothing on screen shows how to get from the GL admin to LuCI (SYSTEM → Advanced Settings → "Go To LuCI"). LuCI looks foreign and "not for me", so I'd need help to install. | medium |
| 2 | 1 | The ticked, greyed-out "Allow overwriting conflicting package files" sounds like it could break something, and nothing says it's safe. | medium |
| 3 | 1 | No screen after Install says it worked or "now open APPLICATIONS → tailbond". | low |
| 4 | 1 | The file-ending hint "_aarch64_cortex-a53_neon-vfpv4.ipk" is gibberish, though I could still compare it with my file. | low |
| 5 | 2 | The waiting screen gives the Flint 3 menu path but no address for a Fedora home server's tailbond page ("Open homebox's tailbond page"), so I'd be stuck waiting. | medium |
| 6 | 2 | The waiting screen doesn't say whether I need to be at home to open the home server's page. | low |
| 7 | 2 | Jargon "tailnet" in the main instructions ("must be on the same tailnet"). | low |
| 8 | 2 | Tempting footer link "Use it as your home server instead" on the travel router, with no hint whether it can be undone. | low |
| 9 | 3 | Ethernet says "Not connected" but its "Use" switch is on, which makes me wonder whether to turn it off. | low |
| 10 | 4 | "Backup only" on Tethering and the "Save mobile data" mode seem to do the same thing, and "Fastest" stays selected after ticking Backup only. I can't tell which one controls my phone data. | medium |
| 11 | 4 | The "Backup only" box and "Use" switch sit side by side, so it's easy to think backup means switching Use off. | low |
| 12 | 5 | The line "Names like nas….ts.net" with the dots cut out looks like a broken name and sits in the Home server box. | low |
| 13 | 5 | On the phone the machine addresses wrap mid-number ("100.101." / "2.2"). It doesn't say whether "nas" alone or "http://" is needed. | low |
| 14 | 6 | The on/off switch has no label. Red "Unpair" and unexplained "Pair again" look like alternatives (the confirm dialog does protect against Unpair). | low |
| 15 | 6 | When the home server can't be reached (08), nothing says whether my devices still have internet or whether I should switch tailbond off. | medium |
| 16 | 7 | Fedora page: a big green "+ Add this phone" ("tailbond takes over from Tailscale on this phone") right under Approve invites a wrong tap. | low |
| 17 | 7 | Flint 13: the green "Home server is on · Ready for travel routers" contradicts the orange warnings below, which are full of jargon (route 10.77.0.0/24, public IP, dynamic DNS). I'd be scared to touch anything. | medium |
