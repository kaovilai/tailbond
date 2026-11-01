# Round 4, persona C: less-technical family member (raw report)

Who I am: a relative set up our home server and Tailscale for me. I can follow
clear steps on a screen. I've used the GL router's admin page a couple of times
(joining hotel Wi-Fi, once a hotel sign-in page). Words like "subnet", "tunnel",
"MagicDNS", "NAT", "relay" don't mean much to me. My main worry is breaking the
internet.

I mostly looked at the phone screenshots, plus a few desktop ones (13, 15, 18, 18b).

---

## Task 1: Install tailbond on the Slate 7 without SSH

**Where I'd click.** On the home server's page (15-fedora-devices-tab-desktop, "Add a
GL.iNet travel router (recommended)") or the Flint 3's panel (11/12/13 "Add a GL.iNet
travel router"), I'd click "Download the router package". Then on the Slate 7:
SYSTEM → Advanced Settings → "Go To LuCI" (01a). The page says "LuCI is OpenWrt's own
web interface. Use the same admin password to log in.", so I know which password to
use. Then System → Software → Upload Package…, pick the file, and the dialog (01b)
"Manually install package / Really install the uploaded package?" → "Install".

**Where I'd hesitate.**
- I've never been in LuCI. 01b only shows the confirmation dialog, not the Software page
  behind it, so I'd have to hunt for the "Upload Package…" button myself. The written
  path helps, though.
- The dialog has a checkbox, "Allow overwriting conflicting package files". The Flint
  panel says "(leave the other options as they are)", which reassures me. The Fedora
  page (15) doesn't say that. Coming from there, I'd wonder whether I should tick it.
- The file is called "tailbond_0.1.0-r42_aarch64_cortex-a53_neon-vfpv4.ipk". I can't tell
  whether that's the right file for *my* router. I'd trust it only because the page made it.
- The Flint text "The router's internet keeps working." is exactly the reassurance I
  need. The Fedora page doesn't have it.

**Expected.** Basically what's there. One sentence on the Fedora page like the Flint's
"leave the other options as they are… internet keeps working" would cover it.
Severity: low.

## Task 2: Pair the Slate 7 with the home server

**Where I'd click.** VPN → tailbond (02). It shows "Looking for your home server on your
tailnet…" and a few seconds later (03) "Found your home server: homebox" with a big
"Ask to connect" button. I click it. 04 says "Waiting for approval on homebox… On
homebox, tap **Approve** next to **slate7**. This router connects by itself once
approved; you can close this page." Very clear.

**Small snags.**
- "tailnet" (02, 03c "No home server found on your tailnet") is a word I don't know.
  From context I'd guess it means "my Tailscale stuff".
- At the bottom of every screen there's a blue link, "Setting up the router that stays at
  home? Make this the home server". It's on the travel router. I'd worry about tapping it
  by accident, or wonder whether I'm *supposed* to. "(you can switch back here)" helps a bit.
- 03b "Turn on Tailscale first" with "Open Tailscale settings" is clear.
- 03c "Checked 4 online machines. On the home server, tailbond must be installed and
  running…" tells me it's not my fault, and I'd call my relative.

Severity: low.

## Task 3: Hotel Wi-Fi + phone tethering, both used

**Where I'd look.** 05-connected-two-wans: the top card says "Connected to homebox · Using
Repeater (Wi-Fi) + Tethering (phone)". Under "Internet connections", both Repeater and
Tethering show a green "In use" badge with speeds. Ethernet says "Not connected" and
"Plug a cable from the hotel or modem into the WAN port." Easy to confirm.

**Small snag.** In 06, the Tethering card turns red: "Not working right now. It's
connected, but can't reach homebox. tailbond keeps trying; the other connections carry
your traffic." The second sentence calms me down. "Repeater" is GL's word, and I know it
from joining hotel Wi-Fi. Fine.

Severity: none worth noting.

## Task 4: Phone as backup only

**Where I'd click.** In the Tethering card, "Use it: Always | Backup only | Off" → "Backup
only" (07). The badge becomes "Backup · saving data", with the text "Used only if your
other connections stop working. Until then it isn't used at all, so it uses no data."
That's exactly what I wanted to hear. The hint under "Speed or reliability" also points
me there: "Saving your phone's data? Set Tethering or Cellular to "Backup only" under
Internet connections."

**Small snag.** "Change limit" (07b) opens "5 GB a month" with the text "Counts everything
tailbond sends and receives over it… 0 = no limit." I'd wonder whether this matches what
my phone company counts. It says tailbond only, so probably not. Not needed for the task.

Severity: low (limit wording only).

## Task 5: Reach the NAS by its name from the laptop

**Where I'd look.** On 05, scroll to "Tailnet machines": "nas, nas.tail1234.ts.net,
100.101.2.2, linux · Online · direct", with a "Copy name" button. The text above says
"Laptops and phones on this router's Wi-Fi or LAN reach these by their full name, e.g.
nas.tail1234.ts.net". The Home server card also says "Tailnet names: Work on every
device connected to this router while tailbond is on. Use the full name, like
nas.tail1234.ts.net." So the name is **nas.tail1234.ts.net**, and I'd press "Copy name".

**Small snags.**
- The task says "MagicDNS name". That word doesn't appear anywhere, which is fine by me
  because "full name" is clearer. Someone told to "find the MagicDNS name" might hesitate
  for a second, though.
- "or by their 100.x address" and "via relay (slower)" next to pixel-8 are jargon. The
  orange "(slower)" makes me wonder whether something is wrong, but it's not about the NAS.

Severity: low.

## Task 6: Turn tailbond off

**Where I'd click.** The toggle in the top card (05). The result (09): "tailbond is off.
Devices on this router use the internet directly, without your home server. Turn on to
bond your connections again." That's reassuring: my internet still works. Connections
show "Ready · Used when tailbond is on."

**Small snags.**
- With tailbond off, the Home server card still shows "Ask to connect again" and a red
  "Unpair". I'd wonder whether turning it back on means pressing "Ask to connect again"
  rather than the toggle. The top card answers that ("Turn on to…").
- Red "Unpair" sits right there. If I tapped it by mistake, 09b asks first: "Unpair from
  homebox? … To connect again, ask the home server again (it will need to approve)." with
  "Cancel". It's safe because of the confirmation.
- 08 (server unreachable) has a good, plain button: "Turn tailbond off", with "Meanwhile,
  devices on this router have no internet… To use this internet directly for now, turn
  tailbond off." It's scary but it tells me what to do.

Severity: low.

## Task 7: Approve the travel router at home

**Flint 3 (11).** Near the top, "Waiting for your approval · 1 new" shows "slate7 wants to
connect · Travel router on your tailnet · signed in as alex@example.com · GL-BE3600" with
"Decline" / "Approve". I'd press Approve. 12 shows slate7 under Devices as "Connected just
now". Easy.

**Fedora page (14 phone, 15 desktop).** Same card: "slate7 wants to connect" with "Approve".
Easy.

**Small snags.**
- The Fedora card text "To require a password, set "ui_password" in
  /etc/tailbond/tailbond.json (see the Fedora server guide)" is meaningless to me. I'd skip it.
- On my phone (14), right under the approval there's a big "Add this phone" box with
  "Download tailbond for Android" and Android install warnings. I came here to approve the
  router, and for a moment I'd think I also have to install an app. It does say "Unsure?
  Ask whoever set up the server…", which I like.
- The phone header "Home server · reachable at home.…" is cut off. That doesn't matter to me.

Severity: low.

## Task 8: Hotel Wi-Fi with a sign-in page (GL repeater)

**Where I'd click.** 16: a blue box "Repeater needs you to sign in. GL's Repeater page
handles this sign-in. Open it, click "Login Mode for Public Hotspots" (or "Go to Login")
and accept the hotel's terms. tailbond steps aside while you sign in, and adds Repeater
back to bonding by itself afterwards. Your other connections keep working meanwhile."
Button: "Open GL's Repeater page". I've done GL's hotel login once, so this matches what I
know. 16b then says "Signing in through GL's Login Mode now: tailbond is letting devices
here reach the sign-in page directly." The Repeater badge reads "Signing in…".

**Small snags.**
- Two names for the GL button ("Login Mode for Public Hotspots" or "Go to Login"). I'd
  look for both. OK.
- No screen shows the "after" state. I have to trust "adds Repeater back… by itself".
  I'd come back and look for "In use" on Repeater.

Severity: low.

## Task 9: Hotel Ethernet cable with a sign-in page

**Where I'd click.** 17: an orange box "Ethernet needs you to sign in. It shows a sign-in
page before it lets anyone online. Until then tailbond keeps Ethernet out of bonding and
your other connections carry your traffic." Button: "Open sign-in page". 17b: "Signing in
to Ethernet… Sign in on the page that opened (accept the terms, enter the room
number…). Until 04:06, devices here use Ethernet directly, without your home server.
tailbond adds it back by itself once you're signed in." Buttons: "Open the sign-in page
again", "I've signed in, check now", "Stop".

**Where I'd hesitate.**
- "Until 04:06" reads like a clock time (4:06 in the morning?) rather than "4 minutes 6
  seconds left". The orange box before said "for up to 10 minutes", so I'd eventually work
  it out.
- "Stop": stop what? Stop tailbond? Stop the internet? Given my fear of breaking things,
  I wouldn't touch it. "Stop signing in" would be clearer.
- After I sign in, "I've signed in, check now" is a nice button. I'd press it.

Severity: low.

## Task 10: At home, make sure everything is set up for best speed

Honestly, I'd **hand this off to whoever set things up**. Still, I looked at the "Setup
health" card because its title sounds like the answer.

**Flint 3, all good (11/12).** A green "All good" badge with "✓ Everything is set up for
best speed" and a list of ticks. I'd be happy and stop there. Great.

**Flint 3, freshly set up (13).** This one confused me. The top says "Home server is on ·
2 setup steps left", and "Finish setting up" lists "Step 1: allow tailbond devices on your
Tailscale network" and "Step 2: tell travel routers where home is". But the Setup health
card says **"0 to improve"** (in an orange badge) and right under it **"✓ Everything is set
up for best speed"**, followed by two "?" items: "Behind your home router: Forward UDP
443 on that router to 192.168.8.1." and "UDP 443: not confirmed from outside yet". So is
it fine or not? If I only read the health card, which is the thing that claims to answer
"best speed", I'd tell my relative "it says everything's set up", and that would be wrong.

The steps themselves ("In the Tailscale admin console, open homebox, choose Edit route
settings, tick the one route listed (10.77.0.0/24)…", "turn on APPLICATIONS → Dynamic DNS
and paste the name…") are beyond me. Unlike the Fedora phone page, nothing here says
"ask whoever set this up". I'd hand it off.

**Flint 3, relayed (18).** "2 things to improve for best speed: Tailscale: 2 machines direct,
2 via relay (slower)… Let Tailscale connect directly: forward UDP 41641 to this machine or
turn on UPnP/NAT-PMP on your router, and avoid double NAT." Plus "Hard NAT on this
network…" and "Check with: tailscale ping slate7". None of it means anything to me, and I
would hand it off. It does say clearly *that* something should be improved, so I'd know to
call. One oddity: it says "Via relay: slate7, work-laptop", but the Tailnet machines list
right below marks only pixel-8 as "via relay (slower)".

**Flint 3, carrier NAT (18b).** The green box under "Internet and Tailscale" says "Setup done.
Travel routers and phones can connect from anywhere…". The health card says "Your internet
line uses carrier-grade NAT… so travel routers can't reach UDP 443 here." Those two
contradict each other: green says it works from anywhere, orange says travel routers
can't reach it. "Ask your provider for a public IPv4 address" is at least something I could
read out to the internet company.

**Fedora page (15).** The task notes say this screen includes setup health, but I couldn't
find a "Setup health" card on the Devices tab. I see approval, the speed chart, the tabs
"Connections / Devices / Tailscale / Logs", "Add a GL.iNet travel router", the device code,
the invite link, and devices. Maybe it's on another tab, but I wouldn't know to look. On
Fedora, I'd have no way to check "best speed" myself.

---

## Findings

| # | task | finding | severity |
|---|---|---|---|
| 1 | 10 | Flint 13: Setup health shows "0 to improve" + "✓ Everything is set up for best speed" while the same page says "2 setup steps left" and the card itself lists two "?" items (forward UDP 443, not confirmed). I'd wrongly report "all set" | high |
| 2 | 10 | Flint 18b: green "Setup done… can connect from anywhere" contradicts health's "travel routers can't reach UDP 443 here" (carrier-grade NAT). I can't tell which to believe | medium |
| 3 | 10 | Fedora 15: no visible "Setup health" card on the Devices tab, so I can't check "best speed" on the Fedora page | medium |
| 4 | 10 | Flint 13/18: fix steps are all jargon ("Edit route settings… 10.77.0.0/24", "forward UDP 41641… UPnP/NAT-PMP… avoid double NAT", "tailscale ping slate7") with no "ask whoever set this up" line like the Fedora phone page has. I'd hand off anyway | medium |
| 5 | 10 | Flint 18: "Via relay: slate7, work-laptop" doesn't match the Tailnet machines list, where only pixel-8 is "via relay (slower)" | low |
| 6 | 1 | 01b: "Allow overwriting conflicting package files" checkbox. The Fedora page doesn't say "leave the other options as they are" or "router's internet keeps working" (the Flint page does) | low |
| 7 | 1 | 01b only shows the confirmation dialog, not where "Upload Package…" sits in LuCI's Software page. A first-time LuCI user has to hunt for it | low |
| 8 | 2 | "Make this the home server" link on every travel-router screen. A worried user may tap it or think they should | low |
| 9 | 2 | "tailnet" ("Looking for your home server on your tailnet…") is unexplained jargon | low |
| 10 | 4 | 07b limit: unclear whether "Counts everything tailbond sends and receives" matches the phone carrier's count | low |
| 11 | 5 | Task says "MagicDNS name". The screen says "full name", which is clearer, but the word mismatch could make a told-what-to-find user hesitate. "via relay (slower)" and "100.x address" are jargon | low |
| 12 | 6 | With tailbond off (09), the "Ask to connect again" and red "Unpair" buttons still show, so it's unclear whether to use them or the toggle to turn back on (confirmation dialog 09b keeps it safe) | low |
| 13 | 7 | Fedora approval text "set "ui_password" in /etc/tailbond/tailbond.json" is meaningless to me. On phone (14), the big "Add this phone / Download tailbond for Android" box under the approval suggests I must install an app | low |
| 14 | 8 | Two GL button names ("Login Mode for Public Hotspots" or "Go to Login"), and no screen showing Repeater back "In use" afterwards | low |
| 15 | 9 | 17b: "Until 04:06" reads as a clock time, not a countdown. "Stop" is ambiguous (stop what?) and scary to a cautious user | low |
