# Round 9 — Android persona B (homelab)

Persona: I run a homelab with Tailscale subnet routers and use the official
Tailscale Android app every day. When I look at a new app I care about routes,
DNS, which ACL identity my traffic carries, where my internet traffic leaves
from, and whether sideloading and pairing are safe. My only materials were
`task-script.md`, `screens.md` and the PNGs in `web/` and `android/`.

Packet note (not counted as a finding): `03-deeplink-confirm` is
pixel-identical to `02e-pairing-ask-approved-confirm`. The background is the
"Add this phone" pairing screen, not the main screen that the manifest
describes. So I couldn't judge the "app already open" variant.

---

## Task 1: Get the tailbond app onto the phone

**Path:** With Tailscale on, I type `homebox/` into Chrome and land on
`web-00a` ("Get tailbond on this phone"). In step 1 I tap **Download tailbond
for Android (27 MB)**. Then I follow the five drawn Android prompts on
`web-00`: ① "Download anyway", then Open; ② "Settings"; ③ "Allow from this
source", then back; ④ "Install"; ⑤ "Scan app", then Install. Then, as the page
says, "When it says App installed, tap Done and come back to this page (Chrome,
this tab) for step 2."

**What works:** This is the clearest sideload page I've seen for a self-hosted
APK. It labels the drawings ("the pictures below are drawings, not buttons"),
it says I can revoke the permission afterwards ("You can turn Allow from this
source off again afterwards; tailbond stays installed"), and it covers the case
where the Chrome download message disappears ("Chrome's menu ⋮ → Downloads").
`web-00b` catches the "tapped step 2 too early" case well: "tailbond isn't on
this phone yet, so step 2 brought you back here." The "Is this safe?" note
gives the correct reason that plain `http://` is acceptable here: the transport
is Tailscale.

**Where I hesitate (low):** The integrity check doesn't help me on the phone.
"SHA-256 5 4167117c… the same as `sha256sum /usr/share/tailbond/tailbond.apk`
on homebox" can only be compared by someone with a shell on homebox. Android
gives me no built-in way to hash the download. On the phone layout the hash
also wraps so that the leading `5` sits alone at the end of the previous line,
which makes it easy to misread. I'd prefer an APK signing-certificate
fingerprint, which I can check with Android tools and which also covers future
updates. Severity: **low**.

## Task 2: Add the phone to homebox (pair), from the phone

**Path:** Back in Chrome I tap step 2, **Open tailbond and ask homebox**. The
app opens on `02a` ("Asking homebox… Looking for homebox on your Tailscale
network."). Then `02b` shows "Waiting for approval on homebox…". The approver
sees "pixel-8 · sam@example.com · 100.101.8.8". Once approved, `02e` asks
"Add this phone to homebox?" and I tap **Add phone**.

**What works:** `02b` is the right amount of identity information. It shows the
node name, the Tailscale login, and the 100.x address, so the approver can tell
it's really my phone. That is the Tailscale identity I'd expect. "tailbond
keeps waiting for 10 minutes and carries on by itself once approved" tells me I
can leave the app. `02e` is honest about the data path: "It connects over the
internet to home.example.net:443, and your internet traffic goes out from home
(you can change that in Settings)". `02c`, "Can't find homebox — Check that the
Tailscale app says Connected", is the correct first check. `02d`, "homebox said
no", is clear.

**Where I hesitate:**

- **Who is allowed to approve?** (low) Step 2 on the phone page says
  "Whoever runs homebox taps Approve on homebox's page". Below that it says
  "You run homebox yourself? Approve it at http://homebox:8077: … and that page
  works on this phone too." From what I can see, I can't tell whether *anyone*
  on the tailnet who can open `homebox:8077` can approve, or only the homebox
  admin. That includes the person on the phone asking to join. Once connected,
  a paired phone acts with homebox's access rules (see Task 3), so this
  matters. I'd want one line such as "only <who> can approve".
- **Disabled button contradicts its hint** (low). In `02c` and `02d` the name
  box already contains "homebox", but **Open home server page** is still greyed
  out. Its hint still says "Type the home server's name in the box above to
  open its page here." Either enable the button or change the hint.
- **Paste error not visible** (low). In `02-pairing-paste-error` I can only see
  a red outline on the "Pairing link" field at the very bottom of the
  screenshot, and the field is cut off. I can't see any error text. If the
  error message also sits below the fold on a real phone, someone pasting a
  wrong link sees only a colour change.
- **Full tunnel is the default and the dialog offers no choice** (low). `02e`
  states the default ("your internet traffic goes out from home"), but the
  only way to change it is "in Settings" after pairing. In the Tailscale app,
  using an exit node is an explicit opt-in. Here it's opt-out, which means
  hairpinning all phone traffic through my home uplink. The disclosure is
  there, so this is polish, not a trap.

## Task 3: Connect

**Path:** On the main screen (`12-home-disconnected`) I tap the big switch.
With Tailscale's VPN on, `04-handoff-other-vpn` appears: "Switch from Tailscale
to tailbond?". I tap **Turn on tailbond**, then OK on Android's VPN dialog
(not shown). I see `11-home-connecting` ("Connecting… Reaching homebox…"), then
`10-home-connected` ("Connected — Through homebox · Wi-Fi + mobile data").

**What works:** The handoff dialog covers what I'd otherwise trip over:
"Android runs one VPN at a time. Turning on tailbond turns off Tailscale (or
any other VPN)". It also covers the Always-on gotcha: "Set Tailscale as
Always-on VPN? Turn that off first (Android Settings → VPN → Tailscale's gear),
or Android keeps switching back to it." `05` (first time, no other VPN) is
consistent with it. The error screens are useful to a power user. `13` says
"homebox didn't answer at home.example.net:443 … the home router may need to
forward UDP port 443". `14` says "the server answering isn't the homebox this
phone was paired with … Don't pair this phone again unless they ask you to",
which is the right advice for a pinning failure.

**Where I get it wrong (medium): the ACL identity changes, and I'm only told
after the switch.** Once connected, the "This device" row shows `10.77.0.2`
and, in small text: "Not a Tailscale address: to your Tailscale machines this
phone's traffic comes from homebox, under its access rules." That explanation
is good, and it's exactly what I want to know. But the screens where I decide
(`04`/`05` and the `02e` pairing confirm) only say "You still reach your
Tailscale machines, like your NAS, through homebox", which implies nothing
about my access changes. In my tailnet, ACLs and Tailscale SSH/whois-based auth
are keyed on `sam@example.com` / `pixel-8`. After the switch the NAS sees
homebox's node and tags instead. So rules written for my user stop applying,
and anything homebox is allowed to reach becomes reachable from the phone. I'd
expect one sentence in the switch dialog, such as "Your Tailscale machines will
see this phone as homebox and apply homebox's access rules, not yours." Today I
learn it only if I scroll to "This device" after connecting. Severity:
**medium**.

## Task 4: Find and copy the NAS's MagicDNS name or Tailscale IP

**Path:** On the main screen I scroll to **Machines**, type `nas` into "Search
machines" and tap the copy icon (`18`). The toast says "Copied 100.101.2.2". For
the name I tap the row and get `19-machine-detail`, which lists "MagicDNS name
nas.tail1234.ts.net", "Short name nas", "Tailscale IPv4 100.101.2.2" and
"Tailscale IPv6 fd7a:115c:a1e0::2", each with its own copy icon. The toast says
"Copied nas.tail1234.ts.net".

**What works:** The detail page is better than what I'm used to. It has every
address with a label, including IPv6 and the short name, plus "With tailbond
on, apps on this phone can use these addresses just like with Tailscale."
Search works, and online/offline dots plus "Last seen 44 h ago" match what the
Tailscale app shows.

**Where I hesitate:**

- **Which address does the list's copy icon copy?** (low) The row puts
  `nas.tail1234.ts.net` above `100.101.2.2`, and the name is the more prominent
  line. A single unlabeled copy icon could copy either. Only the toast tells me
  it copied the IP.
- **No routes or DNS information** (low). As someone who runs subnet routers, I
  look for advertised routes (e.g. a 192.168.x/24 behind a router node) and
  for which resolver handles DNS. The Machines list and detail show only
  tailnet nodes. Neither Settings option ("Through your home server" / "Only
  Tailscale machines") says where DNS queries go. So I can't tell whether my
  subnet-routed LAN is reachable through homebox, or whether split mode still
  resolves MagicDNS names. These are questions the screens leave open, not
  something I saw break.

## Task 5: Tell whether Wi-Fi and 5G are both being used right now

**Path:** I read the main screen (`10-home-connected`). It says "Using both
Wi-Fi and mobile data", both tiles show "In use" with their rates (↓38.4 ↑4.1
and ↓22.7 ↑2.9 Mbps), and the status card says "Wi-Fi + mobile data". For
detail I tap **More options ›** to open `20-connections`, which shows per-path
"delay 21 ms" / "delay 48 ms". The totals add up (61.1 = 38.4 + 22.7, 7.0 =
4.1 + 2.9). `15-one-path-failed` shows the degraded state just as clearly:
"Using Wi-Fi only", with Mobile data marked by a red dot and "Not working right
now".

**Where I hesitate (low):** The top chart on `20-connections` has two lines,
teal and purple, with no legend. I assume they're download and upload, but
nothing on screen says so.

## Task 6: Make it save mobile data

**Path:** I flip the **Save mobile data** switch on the main screen ("Use mobile
data only when Wi-Fi stops working"). `16` confirms it: "Using Wi-Fi · saving
mobile data (backup only)", with Mobile data shown as "Not in use · saving
data". `17` shows the failover: "Wi-Fi isn't working · using mobile data",
"In use · taking over". `21` adds a monthly cap ("1.23 GB of 5.00 GB used this
month") and an "experimental" warm-standby option.

**What works:** One switch does it. The mobile-data limit is clearly
"Separate from Save mobile data", which is what I'd want.

**Where I hesitate (low):** `21` says "The Save mobile data switch on the home
screen picks 'Save mobile data' here", but it doesn't say what turning the
switch *off* goes back to. Is it Fastest, or whatever mode I had before (for
example "Smooth calls and video")?

## Task 7: Hotel Wi-Fi with a sign-in page

**Path:** `16c` shows a card: "Wi-Fi needs sign-in — This Wi-Fi shows a sign-in
page before it lets you online. tailbond keeps you connected over mobile data
meanwhile, and adds Wi-Fi by itself once you're signed in." I tap **Sign in to
Wi-Fi**, complete the portal in Android's sign-in page (not shown), and
tailbond adds Wi-Fi back by itself, as the card promises.

**What works:** This is correct and reassuring for a power user: "Android's
sign-in page uses the Wi-Fi directly, so you don't need to turn tailbond off."
Settings also confirms the behaviour I'd want: "tailbond never bonds over a
Wi-Fi until Android says it's online."

**Where I hesitate (low):** The card gives two instructions. One is the
button **Sign in to Wi-Fi**. The other is the small print "Tap the Wi-Fi
network (it says "Sign in to network") to open the page." I can't tell whether
the button opens the portal directly, or opens Android's Wi-Fi list where I
then have to tap the network. If it's the second, the small print should say
"…then tap the network".

## Task 8: Turn tailbond off and go back to the Tailscale app

**Path:** On the main screen I tap **Switch back to Tailscale** (also in
Settings: "Turns tailbond off and opens the Tailscale app. Turn Tailscale on
there if it says Disconnected"). `12b` shows "Disconnected — Stopped. Back to
Tailscale: turn it on in its app." plus a "Last step: turn Tailscale on" card
with an **Open Tailscale** button. In the Tailscale app I flip its switch.

**What works:** This matches how Android actually behaves: when tailbond took
over, Tailscale's VPN was revoked, and it needs a manual re-enable. The
"Until then, your Tailscale machines aren't reachable" line is the right
warning. "Keep tailbond on all the time (optional)" in Settings correctly says
to leave "Block connections without VPN" off if I want to switch back easily.

**Where I hesitate (low):** On `12b`, after I've gone back to Tailscale, "This
device" still says "This phone reaches your tailnet through homebox". At that
moment it doesn't: it reaches the tailnet through the Tailscale app, as its
own node. The line is stale in this state.

---

## Findings

| # | Task | Finding | Severity |
|---|---|---|---|
| 1 | 3 | The ACL identity change is disclosed only after connecting ("to your Tailscale machines this phone's traffic comes from homebox, under its access rules"). The switch and pairing dialogs only say "You still reach your Tailscale machines … through homebox", so user-keyed ACLs and SSH silently stop applying and homebox's access rules apply instead. | medium |
| 2 | 1 | The SHA-256 check needs a shell on homebox ("the same as `sha256sum /usr/share/tailbond/tailbond.apk` on homebox"), so I can't check it from the phone. On the phone layout the hash wraps with the leading "5" alone on the previous line. No signing-cert fingerprint is offered. | low |
| 3 | 2 | It's unclear who may approve: "Approve it at http://homebox:8077 … that page works on this phone too". Can any tailnet user, including the requester, approve? | low |
| 4 | 2 | In `02c` and `02d`, "Open home server page" stays disabled with "homebox" already typed, while its hint says "Type the home server's name in the box above". | low |
| 5 | 2 | In `02-pairing-paste-error`, only a red outline on "Pairing link" is visible, at the cut-off bottom. No error text is visible. | low |
| 6 | 2 | The pairing confirm sets full tunnel by default ("your internet traffic goes out from home (you can change that in Settings)") and offers no choice in the dialog. | low |
| 7 | 4 | The list's copy icon is unlabeled and copies the IP ("Copied 100.101.2.2"), even though the MagicDNS name is the more prominent line. | low |
| 8 | 4 | No subnet-route or DNS-resolver information anywhere, and the Internet traffic options don't say where DNS goes or whether MagicDNS works in "Only Tailscale machines" mode. | low |
| 9 | 5 | The `20-connections` chart has teal and purple lines with no legend. | low |
| 10 | 6 | It's unclear what turning the "Save mobile data" switch off goes back to (Fastest, or the previous mode). | low |
| 11 | 7 | The "Sign in to Wi-Fi" button and the small print "Tap the Wi-Fi network (it says "Sign in to network")" read as two different actions. | low |
| 12 | 8 | After switching back (`12b`), "This device" still says "This phone reaches your tailnet through homebox". | low |

Totals: 0 high / 1 medium / 11 low
