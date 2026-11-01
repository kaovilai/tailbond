# Round 5: Persona C (less-technical family member)

Who I am: a relative set up the home server and Tailscale for me. I can follow
clear steps on screen and I've used the GL router page a couple of times
(joining hotel Wi-Fi, one hotel sign-in page). Words like "subnet", "tunnel",
"MagicDNS", "NAT", "relay" don't mean much to me. My main worry is breaking
the internet. I mostly looked at the phone screenshots and checked a few
desktop ones.

Inputs: task-script.md, screens.md, out/ux/glinet/round-5/*.png.

---

## Task 1: Install tailbond on the Slate 7 without SSH

**Where I'd click.** I'd follow the "Add a GL.iNet travel router" steps that I
saw on the home page when I downloaded the file (11/12/13/18c-phone, and 15 on
Fedora): "On the travel router's admin page: **SYSTEM → Advanced Settings → Go
To LuCI** (log in with the same password), then **System → Software → Upload
Package…**, pick the file and click **Install** (leave the other options as
they are). The router's internet keeps working."

- 01a-phone: "SYSTEM / Advanced Settings" has one big blue "Go To LuCI" button,
  and "LuCI is OpenWrt's own web interface. Use the same admin password to log
  in." Easy to follow.
- 01b-desktop: the menu along the top shows "System", and there's a blue
  "Upload Package…" button. The dialog asks "Really install the uploaded
  package?" and shows "tailbond_0.1.0-r42_aarch64_cortex-a53_neon-vfpv4.ipk
  (6.1 MB)", with an "Allow overwriting conflicting package files" checkbox,
  "Cancel" and "Install". I'd leave the box unticked because the steps said
  so, and click Install.

**Where I hesitated.**
- "LuCI" and "OpenWrt" are new words to me, and going into a different admin
  page feels risky. "The router's internet keeps working" in the home-server
  steps calmed me down. That sentence would also help on the GL "Advanced
  Settings" card itself, because that's the page I'd be looking at. (low)
- The file name "aarch64_cortex-a53_neon-vfpv4" means nothing to me, and I
  can't tell if it's the right file for my Slate 7. "It's the same tailbond
  this router runs" (on the Flint page) hints that it is, but only for the
  Flint. I'd just trust it. (low)
- After I click Install, no screen shows that it worked or how to get back
  from LuCI to the normal GL page. Step 3 says "Back on the travel router's
  admin page", so I'd probably go back to my old tab or type 192.168.8.1
  again. (low)
- The phone version of 01b only shows the dialog. I can't see how LuCI's menu
  works on a phone. Since the file is on my laptop anyway, I'd use the laptop.
  (low)

## Task 2: Pair the Slate 7 with the home server

**Where I'd click.** VPN → tailbond. 02-phone shows a spinner with "Looking
for your home server on your tailnet…". A moment later 03-phone shows "Found
your home server: homebox" and a big blue "Ask to connect" button. I tap it.
04-phone then says "Waiting for approval on homebox… On homebox, tap
**Approve** next to **slate7**. This router connects by itself once approved;
you can close this page."

This was very clear. The "From anywhere: open http://homebox:8077 on any
device signed in to your Tailscale account (a phone works too)" tip tells me
I can approve it from my own phone, which I like.

- I don't know the word "tailnet" ("on your tailnet…"). I can guess it means
  my Tailscale things. (low)
- If I hit 03b ("Turn on Tailscale first", with "Open Tailscale settings" and
  "Search again" buttons), I could fix it myself. If I hit 03c ("No home
  server found on your tailnet… Checked 4 online machines. On the home server,
  tailbond must be installed and running…"), I'd call my relative. That's
  fine and the message is honest. (low)

## Task 3: Hotel Wi-Fi plus phone tethering, both used

**Where I'd look.** 05-phone, top card: "Connected to homebox · Using
Repeater (Wi-Fi) + Tethering (phone)", "↓ 59 Mbps ↑ 7.1 Mbps". Below it,
Repeater and Tethering both have a green "In use" label. That's exactly what I
wanted to confirm.

- 06-phone (phone trouble) is also clear: "Tethering (phone) not working
  right now" in the top card, and under Tethering: "It's connected, but can't
  reach homebox. tailbond keeps trying; the other connections carry your
  traffic." I'd understand that nothing is broken. (no finding)
- "Repeater" is GL's word. I know it from joining hotel Wi-Fi, and "(Wi-Fi)"
  next to it helps. (no finding)

## Task 4: Phone as backup only

**Where I'd click.** On the Tethering card, the "Use it" row has three
choices: "Always | Backup only | Off". I tap "Backup only". 07-phone then
shows the label "Backup · saving data" and the text "Used only if your other
connections stop working. Until then it isn't used at all, so it uses no
data." The top card says "Tethering (phone) on standby". Very reassuring.

- The "Speed or reliability" section offers "Most reliable: Sends everything
  over every connection. Uses the most data." I briefly wondered whether
  "reliable" was the setting I wanted. The hint below it ("Saving your
  phone's data? Set Tethering or Cellular to 'Backup only' under Internet
  connections.") pointed me back to the right place. I still don't know if
  picking "Most reliable" would undo my Backup only. (low)
- 07b "Change limit" ("5 GB a month", "Save", "Cancel", "0 = no limit") is
  clear, and I didn't need it for this task. (no finding)

## Task 5: Reach the NAS by its MagicDNS name

**Where I'd look.** At the very bottom, the "Tailnet machines" card (05-phone)
says "Laptops and phones on this router's Wi-Fi or LAN reach these by their
full name, e.g. nas.tail1234.ts.net". The "nas" row shows
"nas.tail1234.ts.net" and a "Copy name" button. So the name is
**nas.tail1234.ts.net**. I'd tap "Copy name" and send it to my laptop.

- The page never says "MagicDNS", so if my relative used that word I'd have
  to guess that "full name" is what they meant. I'd guess right because the
  example is literally "nas". (low)
- The card is at the very bottom of a long page. On a phone I scrolled past
  four other sections to reach it. (low)
- In the Home server card, the label "Tailnet names" followed by "Work on
  every device connected to this router while tailbond is on…" reads oddly.
  It took me a second to see it as one sentence ("Tailnet names work on…").
  (low)
- It doesn't say what to do with the name (type it into the browser? into
  the photo app?). My relative would tell me. (low)

## Task 6: Turn tailbond off

**Where I'd click.** The blue switch in the top card. 09-phone: "tailbond is
off. Devices on this router use the internet directly, without your home
server. Turn on to bond your connections again." That's exactly the
reassurance I need: the internet still works.

- The red "Unpair" button sits next to "Ask to connect again". I might tap
  it, thinking it means "turn off". The 09b dialog stops me: "Unpair from
  homebox? This router stops using tailbond and forgets the home server…
  (it will need to approve)." I'd press Cancel. (low)
- When tailbond is off, the "Ask to connect again" button is still there. It
  made me wonder whether I'd need my relative's approval again to turn it
  back on. The top card says "Turn on to bond…", so I'd use the switch.
  (low)
- 08 (home server can't be reached): "Meanwhile, devices on this router have
  no internet… To use this internet directly for now, turn tailbond off",
  with a "Turn tailbond off" button. Very helpful for my worry. The "UDP port
  443" part is clearly addressed to "Whoever set up homebox". (no finding)
- 10 ("tailbond isn't running on this router… Starting it is safe: devices
  here keep their internet while you set it up", with "Start tailbond") is
  clear and calming. (no finding)

## Task 7: Approve the new travel router on the home side

**Flint 3 (11-phone).** It's at the top: "Waiting for your approval · 1 new",
"slate7 wants to connect · Travel router on your tailnet · signed in as
alex@example.com · GL-BE3600", then "Decline" and a blue "Approve". I tap
Approve. 12-phone shows slate7 under Devices as "Connected just now". Easy.

**Fedora (14-phone, 15-desktop).** It's also at the top: "slate7 wants to
connect", with "Decline" and "Approve". Easy.

- On Fedora, the "Who can approve" text mentions setting "ui_password" in
  /etc/tailbond/tailbond.json. That's not for me, but it isn't in my way.
  (low)
- On the Fedora phone page (14), a big green-bordered "Add this phone" card
  with "Download tailbond for Android" sits right under the approval. I could
  wander into it by mistake, but the Approve button comes first. (low)
- The Fedora page header is cut off on the phone: "Home server · reachable at
  home.···". (low)

## Task 8: Hotel Wi-Fi with a sign-in page

**Where I'd click.** 16-phone, blue banner: "Repeater needs you to sign in.
GL's Repeater page handles this sign-in. Open it, click 'Login Mode for Public
Hotspots' (or 'Go to Login') and accept the hotel's terms. While you sign in,
devices here use the hotel Wi-Fi directly, not through your home server.
Afterwards tailbond adds Repeater back to bonding by itself." I tap "Open GL's
Repeater page", then GL's Login Mode button, which I've used before.

16b-phone: "GL's Login Mode is on… It ends by itself once you've signed in (or
click 'Quit Login Mode' on GL's Repeater page); tailbond then adds Repeater
back to bonding." The Repeater label shows "Signing in…".

This matched what I already knew from GL, so I was comfortable.

- There's no screenshot of GL's Repeater page, so I'm trusting that the
  button really has one of those two names. (low)
- It doesn't say where the hotel's page will appear (on my laptop? in this
  tab?). From my earlier GL experience I'd expect it on the laptop. (low)
- The phone was still on "Always" during this, so my phone carried
  everything. That's fine, but I'd worry about data use while I fiddle. (low)

## Task 9: Hotel Ethernet with a sign-in page

**Where I'd click.** 17-phone, orange banner: "Ethernet needs you to sign in.
It shows a sign-in page before it lets anyone online. Until then tailbond
keeps Ethernet out of bonding and your other connections carry your
traffic." I tap "Open sign-in page". 17b-phone: "Signing in to Ethernet… Sign
in on the page that opened (accept the terms, enter the room number…). For up
to 9 more minutes (until 04:13), devices here use Ethernet directly… tailbond
adds it back by itself once you're signed in." The buttons are "Open the
sign-in page again", "I've signed in, check now" and "Stop sign-in mode".

That's the clearest screen of the lot. "I've signed in, check now" is exactly
the button I'd want.

- Wi-Fi sign-in goes through GL's page, but Ethernet sign-in goes through
  tailbond's own button. I'd wonder why the two work differently, but each
  banner told me what to do. (low)
- It doesn't say what happens if the 10 minutes run out before I finish
  typing my room number. (low)

## Task 10: At home, make sure everything is set up for best speed

I'd mostly hand this to my relative. I can check the result myself, though:

- Flint, 18c-phone: "Setup health · All good · Everything is set up for best
  speed", with green ticks. I'd stop there, happy.
- Flint, 18-phone: "Setup health · 2 to improve… (tailbond works meanwhile,
  just slower)". It talks about "forwarding UDP port 41641" and "Hard NAT",
  which means nothing to me. But it says "Not sure how? Show this card to
  whoever set up your internet; the links explain each fix." So I'd send a
  screenshot to my relative. That's a good handoff. (no finding)
- Flint, 18b-phone: a red "Step 1: Your internet line uses carrier-grade NAT…
  Ask your provider for a public IPv4 address", with "Not sure? Show this to
  whoever set up your internet." I'd hand it off. The same page also says
  "Port: UDP 443, opened on this router automatically" right below the red
  box that says travel routers "can't reach UDP 443 here". Those two read
  like a contradiction to me. (low)
- Flint, 13-phone: "Home server is on · 2 setup steps left", with Step 1
  about the Tailscale admin console and "Edit route settings", and Step 2
  about Dynamic DNS. I wouldn't attempt either; that's for my relative. The
  steps look precise enough for them. (low; handoff)
- Fedora, 15-desktop: "Setup health: 2 to improve for best speed" includes
  the command "To check one: tailscale ping pixel-8". Unlike the Flint card,
  it has no "Show this card to whoever set up your internet" line, so I'd be
  less sure it's not my job. (low)
- 12/18c: health says "Tailscale: 2 machines direct", but the "Tailnet
  machines" list shows three online machines (nas, apple-tv, pixel-8), all
  "direct". I noticed the numbers don't match, though it didn't stop me.
  (low)

---

## Findings

| # | task | finding | severity |
|---|---|---|---|
| 1 | 1 | GL "Advanced Settings" / LuCI feels risky; the "internet keeps working" reassurance is only on the home-server page, not at the LuCI step | low |
| 2 | 1 | Package file name ("aarch64_cortex-a53_neon-vfpv4") gives no sign that it's the right file for the Slate 7 | low |
| 3 | 1 | No screen confirms the install worked or shows how to get from LuCI back to GL's admin page | low |
| 4 | 1 | Phone 01b shows only the dialog; how LuCI's menu works on a phone isn't shown | low |
| 5 | 2 | "tailnet" is unexplained jargon ("Looking for your home server on your tailnet…") | low |
| 6 | 4 | Unclear whether "Most reliable" overrides Tethering's "Backup only" | low |
| 7 | 5 | The word "MagicDNS" never appears; the "full name" has to be guessed as the same thing | low |
| 8 | 5 | "Tailnet machines" (the names) is at the very bottom of a long phone page | low |
| 9 | 5 | "Tailnet names / Work on every device…" label plus sentence reads awkwardly | low |
| 10 | 5 | No hint of what to do with the copied name (where to type it) | low |
| 11 | 6 | Red "Unpair" sits next to the other buttons and could be mistaken for "turn off" (the confirm dialog catches it) | low |
| 12 | 6 | "Ask to connect again" still shows when tailbond is off, implying re-approval is needed | low |
| 13 | 7 | Fedora approval card mentions /etc/tailbond/tailbond.json "ui_password" | low |
| 14 | 7 | Fedora phone page: the big "Add this phone" card right under the approval invites a wrong tap | low |
| 15 | 7 | Fedora phone header cut off ("reachable at home.···") | low |
| 16 | 8 | GL's Repeater page isn't shown, so the "Login Mode for Public Hotspots" button can't be verified | low |
| 17 | 8 | Doesn't say where the hotel's sign-in page will appear | low |
| 18 | 9 | Wi-Fi sign-in uses GL's page but Ethernet uses tailbond's own button; the two flows differ | low |
| 19 | 9 | No word on what happens if the 10-minute sign-in window runs out | low |
| 20 | 10 | CGNAT screen: "UDP 443, opened on this router automatically" sits beside "can't reach UDP 443 here" | low |
| 21 | 10 | Fedora health card has no "show this to whoever set up your internet" handoff line (Flint has one) | low |
| 22 | 10 | Health says "2 machines direct" while the machines list shows 3 online and direct | low |
| 23 | 10 | Route approval and Dynamic DNS steps (13) are for the person who set things up; handed off | low |

Totals: high 0, medium 0, low 23.
