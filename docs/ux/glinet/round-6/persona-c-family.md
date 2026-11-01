# Round 6, persona C (less-technical family member): raw report

Who I am: a relative set up the home server and Tailscale for me. I can follow
clear steps on screen and I've used the GL router's admin page a few times (hotel
Wi-Fi, once a hotel sign-in page). "Subnet", "tunnel", "MagicDNS", "NAT" and
"relay" don't mean much to me, and I'm scared of breaking the internet. I
mostly looked at the phone screenshots and checked a few desktop ones.

## Task 1: Install tailbond on the Slate 7 without SSH

Where I click: the home server page told me how. On the Flint 3 panel
(11/12/13/18c, "Add a GL.iNet travel router") or the Fedora page (15, Devices
tab) I see: "1. Download the router package to this computer. 2. On the travel
router's admin page: SYSTEM → Advanced Settings → Go To LuCI (log in with the
same password), then System → Software → Upload Package…, pick the file and
click Install (leave the other options as they are). The router's internet
keeps working. 3. Back on the travel router's admin page, open VPN → tailbond
and click Ask to connect."
- 01a: SYSTEM / Advanced Settings, then the blue "Go To LuCI" button. The text
  "Use the same admin password to log in" helps.
- 01b: LuCI's "Manually install package / Really install the uploaded
  package?" dialog, then "Install". I leave "Allow overwriting conflicting
  package files" unticked because the steps said to leave the other options.

Where I hesitate:
- There's no screenshot of LuCI's own System → Software page, where "Upload
  Package…" is. The only explanation of LuCI is "LuCI is OpenWrt's own web
  interface", and I don't know what OpenWrt is. I'd go slowly, reading the menu
  names against the steps. I'd manage, but this is the scariest part of the
  whole thing. (low)
- "Use the same admin password" doesn't say what to put in a username box if
  LuCI's login shows one. (low)
- The file is "tailbond_0.1.0-r42_aarch64_cortex-a53_neon-vfpv4.ipk". All that
  gibberish makes me wonder if I picked the right file. The line "The router's
  internet keeps working" reassured me before I clicked Install. (low)

## Task 2: Pair with the home server

- 02 shows "Looking for your home server on your tailnet…", and a few seconds
  later 03 shows "Found your home server: homebox" with a big blue "Ask to
  connect" button. Easy.
- 04: "Waiting for approval on homebox… On homebox, tap Approve next to slate7.
  This router connects by itself once approved; you can close this page." Very
  clear.
- 03b ("Turn on Tailscale first" plus "Open Tailscale settings") and 03c ("No
  home server found on your tailnet … Search again") tell me what to do. 03c is
  really something for my relative ("tailbond must be installed and running").

Where I hesitate:
- "tailnet" ("on your tailnet…", "No home server found on your tailnet"). The
  paragraph above says "the same Tailscale account", which I understand. I
  guess they mean the same thing. (low)

## Task 3: Hotel Wi-Fi plus phone tethering, both used

- 05: the top says "Connected to homebox, Using Repeater (Wi-Fi) + Tethering
  (phone)", and both the Repeater and Tethering cards have a green "In use".
  Done. "Repeater" is the word GL uses, so I know it.
- 06: when the phone acts up, Tethering turns red, "Not working right now", and
  explains "It's connected, but can't reach homebox. tailbond keeps trying; the
  other connections carry your traffic." That reassures me.

No problems.

## Task 4: Phone as backup only

- 07: on the Tethering card, the "Use it" buttons "Always / Backup only / Off".
  I click "Backup only". The badge becomes "Backup · saving data", and the
  card says "Used only if your other connections stop working. Until then it
  isn't used at all, so it uses no data." That's exactly what I wanted.
- The hint under "Speed or reliability" ("Saving your phone's data? Set
  Tethering or Cellular to “Backup only” under Internet connections.") would
  have pointed me there if I'd started in the wrong section.
- 07b: "Change limit" (5 GB a month, "0 = no limit") is an extra I'd leave
  alone.

Where I hesitate:
- "Most reliable: Sends everything over every connection. Uses the most data."
  If I picked that, would it also use my phone's data even though I set Backup
  only? The screen doesn't say. I'd leave "Fastest" alone. (low)

## Task 5: Reach the NAS by name from the laptop

- On the Slate 7's tailbond page, near the bottom, "Tailnet machines" lists
  "nas, nas.tail1234.ts.net, 100.101.2.2" with a "Copy name" button. Under Home
  server, "Tailnet names: Work on every device connected to this router while
  tailbond is on. Use the full name, like nas.tail1234.ts.net." So the name is
  **nas.tail1234.ts.net**. I'd press "Copy name".

Where I hesitate:
- The word "MagicDNS" isn't on screen, so I'd just trust that this is the
  name. The section intro ("not this router's own Tailscale", "“Direct” / “via
  relay” is how homebox reaches each machine") is over my head, but I don't
  need it. It also doesn't say where to type the name. I'd guess the browser's
  address bar. (low)

## Task 6: Turn tailbond off

- 05: the blue switch next to "Connected to homebox". I flip it.
- 09: "tailbond is off. Devices on this router use the internet directly,
  without your home server. Turn on to bond your connections again." Very
  reassuring: the internet still works.
- I would not press the red "Unpair". 09b's dialog ("This router stops using
  tailbond and forgets the home server … it will need to approve") makes it
  clear that's a bigger step, and I'd press Cancel.

Where I hesitate:
- The switch has no "On/Off" label; I know it from GL's other toggles. After
  turning it off (09), the Home server card still shows "Ask to connect again"
  and "Unpair" next to an "Off" badge. To turn it back on I might press "Ask
  to connect again" instead of the switch at the top. (low)

## Task 7: Approve the travel router at home

- Flint 3 (11): at the top, "Waiting for your approval · 1 new", "slate7 wants
  to connect … GL-BE3600", and a blue "Approve". One tap. In 12, slate7 is
  under Devices, "Connected just now". Done.
- Fedora (14, on my phone): "Waiting for your approval … slate7 wants to
  connect", then "Approve". Easy.

Where I hesitate:
- Fedora page: above the button it says 'To require a password, set
  "ui_password" in /etc/tailbond/tailbond.json (see the Fedora server guide).'
  That's computer talk and makes me nervous, but I can ignore it. On the
  desktop (14) a pink bar says "Setup health: 2 things to improve for best
  speed. Show". Pink looks like something is broken, although the words say
  it's only about speed. (low)
- Fedora on the phone: right under the approval there's a big "Add this phone"
  card with "Download tailbond for Android". I could think I need to do that
  too in order to approve the router. I wouldn't, because Approve is first.
  (low)

## Task 8: Hotel Wi-Fi with a sign-in page (GL's repeater detected it)

- 16: a blue box, "Repeater needs you to sign in. GL's Repeater page handles
  this sign-in. Open it, click “Login Mode for Public Hotspots” (or “Go to
  Login”) and accept the hotel's terms." Button "Open GL's Repeater page". I've
  done GL's hotel login before, so this matches what I know. "Your other
  connections keep working meanwhile" is reassuring.
- 16b: "Signing in to Repeater through GL's Login Mode … It ends by itself once
  you've signed in (or click “Quit Login Mode” on GL's Repeater page); tailbond
  then adds Repeater back to bonding." I'd accept the hotel's terms in GL's
  page and come back.

Where I hesitate:
- The top card stays green, "Connected to homebox", while the text under it
  says "all traffic from devices here goes out Repeater (Wi-Fi) directly, not
  through homebox". Is it connected or not? The blue box explains it, but the
  two messages fight each other. (low)
- No screenshot shows the moment it's done. I'd assume I'm back to bonding
  when Repeater shows the green "In use" again, as in 05. (low)

## Task 9: Hotel Ethernet cable with a sign-in page

- 17: an orange box, "Ethernet needs you to sign in … It shows a sign-in page
  before it lets anyone online. Until then tailbond keeps Ethernet out of
  bonding and your other connections carry your traffic." Button "Open sign-in
  page". I click it.
- 17b: "Signing in to Ethernet… Sign in on the page that opened (accept the
  terms, enter the room number…). For up to 9 more minutes … tailbond adds
  Ethernet back by itself once you're signed in." Then "I've signed in, check
  now". Clear. "Open the sign-in page again" is there if the tab got lost.

Where I hesitate:
- "(until 04:18 on this router's clock)": why the router's clock and not
  mine? Small. The line "GL's Login Mode covers the repeater Wi-Fi only;
  tailbond handles sign-in pages on other connections like this one" is in
  small print, but it does tell me not to go to GL's page this time. (low)

## Task 10: Best speed at home (I'd mostly hand this to my relative)

- Flint 3, "Setup health". 18c: "All good, Everything is set up for best
  speed". I'd stop there, happy.
- 18 (relayed): "2 things to improve for best speed (tailbond works meanwhile,
  just slower)". The fix: "On this router: NETWORK → UPnP, turn it on. If
  another router sits in front of this one (for example your internet
  provider's box), forward UDP port 41641 to this router there." I can find
  NETWORK → UPnP in GL's menu, but nothing tells me turning it on is safe, so
  I'd ask first. Forwarding a port I'd hand off. "Not sure how? Show this card
  to whoever set up your internet" is exactly what I'd do. (low)
- 18b (CGNAT): a red box, "Step 1: Your internet line uses carrier-grade NAT …
  Ask your provider for a public IPv4 address (or use public IPv6)." I wouldn't
  know what to ask the provider. It's red and called a "setup step left",
  which feels like something's broken even though devices are connected. "Not
  sure? Show this to whoever set up your internet" means I'd hand it off. (low)
- 13 (fresh Flint): "Step 1: … open homebox, choose Edit route settings, tick
  the one route listed (10.77.0.0/24)" and "Step 2: … APPLICATIONS → Dynamic
  DNS". This is for my relative, so I'd hand it off. The Dynamic DNS step looks
  doable with the "(it looks like abc1234.glddns.com)" hint. (low)
- Fedora (15): the same kind of card, "Forward UDP port 41641 to this machine,
  or turn on UPnP on the router in front of it." I don't know which router is
  "in front of it". I'd hand it off. (low)

## Overall

I could do tasks 1 to 9 myself. Task 1 is the nervous one because of LuCI,
but the steps on the home page are precise. For task 10 I'd only check "All
good", and the screens themselves tell me to hand the fixes to whoever set up
my internet. Nothing made me think I'd break the internet: off, unpair and
sign-in all say plainly that the internet keeps working.

## Findings

| # | task | finding | severity |
|---|---|---|---|
| 1 | 1 | LuCI's System → Software page, where "Upload Package…" is, isn't pictured, and LuCI is only explained as "OpenWrt's own web interface". The most intimidating step. | low |
| 2 | 1 | "Use the same admin password" doesn't say what goes in a username field if LuCI asks for one. | low |
| 3 | 1 | The package name "…aarch64_cortex-a53_neon-vfpv4.ipk" makes me unsure I picked the right file. | low |
| 4 | 2 | "tailnet" ("Looking for your home server on your tailnet…") where the rest says "Tailscale account". | low |
| 5 | 4 | Unclear whether "Most reliable … Uses the most data" would override Tethering "Backup only". | low |
| 6 | 5 | "MagicDNS" isn't on screen, the Tailnet machines intro uses jargon ("via relay"), and it doesn't say where to type the name. | low |
| 7 | 6 | The on/off switch has no label. When off, "Ask to connect again" looks like the way to turn it back on. | low |
| 8 | 7 | The Fedora approval card mentions "ui_password" in "/etc/tailbond/tailbond.json", and the pink "Setup health: 2 things to improve" bar looks like an error. | low |
| 9 | 7 | The Fedora phone page puts a big "Add this phone / Download tailbond for Android" card right under the router approval. | low |
| 10 | 8 | The top card stays green "Connected to homebox" while saying traffic goes "not through homebox" during sign-in. | low |
| 11 | 8 | No screenshot confirms that Repeater went back to bonding after the hotel sign-in. | low |
| 12 | 9 | "until 04:18 on this router's clock" is odd, and the "GL's Login Mode covers the repeater Wi-Fi only" note is in small print. | low |
| 13 | 10 | Relayed fix "NETWORK → UPnP, turn it on" with no word on whether it's safe. Port forwarding is a hand-off, though "Show this card to whoever set up your internet" helps. | low |
| 14 | 10 | The CGNAT box is red and called a "setup step left" ("Ask your provider for a public IPv4 address"), which feels broken. Hand-off. | low |
| 15 | 10 | The fresh-Flint steps (route approval in the Tailscale admin console, Dynamic DNS) are for the person who set things up. Hand-off. | low |
| 16 | 10 | Fedora: "the router in front of it" means nothing to me. Hand-off. | low |
