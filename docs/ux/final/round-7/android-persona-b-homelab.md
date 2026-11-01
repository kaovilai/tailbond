# Round 7 — Android persona B (homelab)

**Persona:** homelab power user. I run Tailscale subnet routers and the Tailscale Android app, and I care about routes, DNS, ACL identity, which traffic goes where, and how safe sideloading and pairing are.
**Materials:** `task-script.md`, `screens.md` and every PNG in `web/` and `android/` (light and dark). I didn't read anything else.

Overall: this is a well-written flow. Every task can be finished from what's on screen, and I didn't find anything blocking. What bothers me as a homelab person is what the screens don't tell me: how to verify the sideloaded APK, how approval proves it's really my phone asking, which identity my traffic has on the tailnet, and the fact that the bond runs over the public internet rather than over Tailscale.

---

## Task 1: Get the tailbond app onto the phone

**Where I tap:** Tailscale is on, so I type `homebox` in Chrome and land on `web-00a`, "Get tailbond on this phone". In step 1 I tap **"Download tailbond for Android (27 MB)"**. Then I follow the drawn prompts in `web-00`: ① "File might be harmful" → **Download anyway**, then Open; ② "For your security, your phone currently isn't allowed to install unknown apps from this source" → **Settings**; ③ "Install unknown apps / Chrome / Allow from this source" → toggle on, back; ④ "Do you want to install this app?" → **Install**, and let Play Protect scan if it offers.

**What works:** the mock-ups of Android's own dialogs are accurate and help a lot. I'm glad the page tells me "You can turn Allow from this source off again afterwards; tailbond stays installed". That's exactly what I'd do. "Stuck, or not comfortable with this? Ask whoever runs homebox to install it for you" is a fair escape hatch. The computer view (`web-00c`) with a QR code and "nothing here changes homebox's settings" is reassuring.

**Where I hesitate:**
- **The page asks me to click through "File might be harmful" but gives me nothing to check the file against.** There's no SHA-256, no signing-certificate fingerprint and no "signed by" line next to the download button. The only version info is the footer "tailbond 0.1.0-r42". The page is plain `http://homebox` (the dashboard link is `http://homebox:8077`). I know the transport is inside my tailnet's WireGuard tunnel, but the page never says so, and it does tell me to override Chrome's warning. With the Tailscale app I install from Play, so the signer is vouched for. Here I'd want a fingerprint I can compare with `apksigner verify --print-certs`, or with the admin's dashboard, before I flip "Allow from this source". The page also doesn't say how updates arrive or whether they'll be signed by the same key. **Medium.**
- `web-00b`: if I tap step 2's button before installing, Chrome returns to the page and step 1 just gets an orange outline. No text in the visible area says "tailbond isn't installed yet, do step 1 first". I'd work it out, but a one-line explanation would help. **Low.**

## Task 2: Add the phone to homebox (pair), from the phone

**Where I tap:** back on the phone page (`web-00`), step 2 → **"Open tailbond and ask homebox"**. The app opens on `02a` ("Asking homebox… Looking for homebox on your Tailscale network."), then `02b` ("Waiting for approval on homebox…"). When the admin approves, I get the `02e` dialog "Add this phone to homebox?" and tap **Add phone**. If I open the app from the launcher instead, I see `01` and tap **Get started**, then on `02-pairing` I type `homebox` and tap **Ask to connect**.

**What works:** the error states are clear: `02c` "Can't find homebox — Check that the Tailscale app says Connected, and that the name matches the one in its list of machines", and `02d` "homebox said no". The `02-pairing-paste-error` message "That isn't a complete tailbond link. Copy the whole link starting with tailbond://pair" is good. The `02e` dialog tells me the phone's name ("This phone will be called "pixel-8" on homebox").

**Where I hesitate:**
- **The only thing that ties this request to my phone is the name "pixel-8".** `02b` says "tap Approve next to "pixel-8"". There's no short code or fingerprint shown on the phone for the approver to match on homebox's page. Anyone on the tailnet with a Pixel 8, or anyone who chooses that name, produces an identical-looking request. The approver can't tell which "pixel-8" they're approving, and nothing on screen says whether homebox records which Tailscale user or node made the request. Tailscale's own device approval shows the owner's identity. I'd expect a "make sure homebox shows code 4F-2K" line on both sides. **Medium.**
- **The tunnel's route and the default for internet traffic only show up in small print, at the last step.** `02e`: "It connects over the internet to home.example.net:443, and your internet traffic goes out from home (you can change that in Settings)." That's two big facts for me:
  1. The bond is **not** carried over Tailscale. It's a separate tunnel to a public endpoint on my WAN. `13-error-unreachable` confirms this: "the home router may need to forward UDP port 443". So there's a port forward and an internet-facing listener I didn't know about.
  2. **The default is full tunnel through home**, which works like an exit node. In the Tailscale app, exit node is off unless I pick one. Neither the phone page ("Your Tailscale machines … stay reachable through it") nor onboarding `01` mentions either fact, so the first I hear of them is the small print on the confirm dialog.

  The text is honest, but it should be up front, before I've decided to install. **Medium.**
- `02b` names a product I don't have: "open its tailbond page (http://homebox:8077 on a computer, or a Flint 3's admin panel → VPN → tailbond)". homebox is a server, not a Flint 3, so this clause doesn't apply to my setup. **Low.**
- The "Or use the home server's page" card stays on screen in `02a`, `02b`, `02c`, `02d` and `02-pairing`. Its **"Open home server page"** button is always greyed out and nothing explains why. And while I'm already asking (02a/02b), "Go back to it in your browser and tap step 2" is redundant. **Low.**
- `02b` says "Keep this screen open; tailbond continues by itself once approved", but the phone page also says the approve page "works on this phone too". If the admin is me, on this same phone, I have to leave the screen to approve. I assume it resumes, but the two instructions point in different directions. **Low.**

## Task 3: Connect

**Where I tap:** on the home screen I tap the main switch. Because Tailscale's VPN is on, `04-handoff-other-vpn` appears: "Switch from Tailscale to tailbond? Android runs one VPN at a time. Turning on tailbond turns off Tailscale (or any other VPN) on this phone." I tap **Turn on tailbond**, then OK on Android's VPN prompt (not shown). `11-home-connecting` ("Connecting… Reaching homebox…") then becomes `10-home-connected` ("Connected — Through homebox · Wi-Fi + mobile data").

**What works:** the handoff dialog is accurate and tells me how to get back ("tap Switch back to Tailscale on the home screen"). The `14-error-cert` screen ("Can't confirm this is your home server … the server answering isn't the homebox this phone was paired with … Don't pair this phone again unless they ask you to") shows the server identity is pinned. That's exactly the behaviour I want, and the warning not to re-pair is the right call.

**Where I hesitate:**
- **What identity does my traffic have on the tailnet?** "This device — pixel-8 (this phone) 10.77.0.2 — Your Tailscale machines can reach this phone at this address, through homebox." So the phone isn't a tailnet node. It sits behind homebox, which looks like a subnet router or NAT. My ACLs are written per user and tag. When I open the NAS, does it see traffic from homebox (with homebox's tags and permissions) or from 10.77.0.2 (which only works if homebox advertises that subnet route and I've approved it)? The screen says the reverse path works, but my admin console would need a route approval for that, and no screen mentions one. If the traffic is NATed to homebox, my phone quietly gets homebox's ACL rights, which could be broader or narrower than mine. A homelab user needs one line here: "Your tailnet sees this phone as homebox" or "as 10.77.0.2 via homebox's subnet route". **Medium.**

## Task 4: Find and copy the NAS's MagicDNS name or Tailscale IP

**Where I tap:** in Machines on `10-home-connected-full`, I type `nas` in **Search machines** (`18`). Tapping the copy icon on the row shows the toast "Copied 100.101.2.2". Or I tap the row and get `19-machine-detail`, "Tap an address to copy it", with MagicDNS name `nas.tail1234.ts.net`, Short name `nas`, Tailscale IPv4 `100.101.2.2` and Tailscale IPv6 `fd7a:115c:a1e0::2`. Tapping the first one shows "Copied nas.tail1234.ts.net".

**What works:** this beats the Tailscale app. All four forms are there, each with a labelled copy button, and "With tailbond on, apps on this phone can use these addresses just like with Tailscale" answers my MagicDNS question.

**Where I hesitate:**
- The copy icon on the list row in `18` doesn't show which value it copies. The row's top line is the MagicDNS name, but the icon copies the IPv4, which I only find out from the toast. **Low.**
- I only see machines. No screen says whether **subnet routes advertised by my other routers** (e.g. my LAN /24s) or my tailnet's **DNS settings** (split DNS, custom nameservers) also work through homebox. The Tailscale app has a "Use Tailscale DNS" toggle. Here I can't tell what DNS the phone uses. That's not needed for this task, but it's the next thing I'd want to know. **Low.**

## Task 5: Tell whether Wi-Fi and 5G are both being used right now

**Where I look:** on `10-home-connected`, under Connections, the heading reads "Using both Wi-Fi and mobile data", with Wi-Fi "In use ↓38.4 ↑4.1 Mbps" and Mobile data "In use ↓22.7 ↑2.9 Mbps". The status card also says "Wi-Fi + mobile data". Tapping **More options** opens `20-connections`, which has per-link rates, "delay 21 ms" / "delay 48 ms" and a 60-second graph.

**Findings:** none. It's unambiguous. `15-one-path-failed` ("Using Wi-Fi only", Mobile data "Not working right now") is just as clear.

## Task 6: Make it save mobile data

**Where I tap:** the **Save mobile data** switch on the home screen ("Use mobile data only when Wi-Fi stops working"). The result is `16-save-data-standby`: "Using Wi-Fi · saving mobile data (backup only)", Mobile data "Not in use · saving data", "Mode: Save mobile data". `17-save-data-takeover` shows it failing over: "Wi-Fi isn't working · using mobile data", "In use · taking over".

**What works:** one tap, and the status text proves it worked.

**Where I hesitate:**
- `20`/`21-connections` hold several data-related controls: the mode radio "Save mobile data", the per-link "Use for tailbond" toggles, "Advanced: choose which connection is main or backup", "Faster switch to backup (experimental) … Uses a little mobile data", and "Mobile data limit … (empty = no limit)". The page explains how they relate ("The Save mobile data switch on the home screen picks "Save mobile data" here", "Separate from Save mobile data: counts everything tailbond sends and receives over mobile data"), so I'm not lost, but that's five knobs for one goal. **Low.**

## Task 7: Hotel Wi-Fi with a sign-in page

**Where I tap:** `16c-home-wifi-needs-sign-in` shows the card "Wi-Fi needs sign-in — This Wi-Fi shows a sign-in page before it lets you online. tailbond keeps you connected over mobile data meanwhile, and adds Wi-Fi by itself once you're signed in." I tap **Sign in to Wi-Fi**. The Wi-Fi tile reads "Needs sign-in · not used yet".

**What works:** this is better than what I get with Tailscale today. "Android's sign-in page uses the Wi-Fi directly, so you don't need to turn tailbond off" tells me exactly where the portal traffic goes. In Settings (`30`), "tailbond never bonds over a Wi-Fi until Android says it's online" is the right security stance: nothing gets bonded over an unauthenticated portal network.

**Where I hesitate:**
- The card gives two different instructions: the button **Sign in to Wi-Fi**, and the small print "Tap the Wi-Fi network (it says "Sign in to network") to open the page". Do I use the button, or go to Android's Wi-Fi list or notification? I'd try the button first. It's just unclear whether the small print is a fallback or a separate step. **Low.**

## Task 8: Turn tailbond off and go back to the Tailscale app

**Where I tap:** **Switch back to Tailscale** on the home screen (`10`). It's also in Settings (`30`): "Turns tailbond off and opens the Tailscale app. Turn Tailscale on there if it says Disconnected". The result is `12b`: "Disconnected — Stopped. Tailscale is in charge on this phone." with the card "Tailscale is in charge on this phone … In the Tailscale app, turn on its switch if it says Disconnected" and an **Open Tailscale** button.

**What works:** it's clear, and the button is in a prominent place on every home-screen state. The Settings note on Always-on VPN ("Turn off Always-on for Tailscale first: only one app can have it", "Leave "Block connections without VPN" off if you want to switch back to Tailscale easily") is correct Android advice that I'd otherwise have learned the hard way.

**Findings:** none beyond Android's own constraint that Tailscale has to be switched back on by hand, and the screen states that honestly.

---

## Findings

| # | Task | Finding | Severity |
|---|---|---|---|
| 1 | 1 Install | Page asks me to tap "Download anyway" past "File might be harmful" and allow unknown sources, but shows no APK hash or signing-cert fingerprint to verify against, and nothing about update signing. Page is plain `http://homebox`; it never says the transport is protected by Tailscale. | medium |
| 2 | 1 Install | `web-00b`: after tapping step 2 with no app installed, step 1 is only outlined in orange; no visible text explains "install first". | low |
| 3 | 2 Pair | Approval identifies the request only by name ("tap Approve next to "pixel-8""); no matching code/fingerprint on phone and homebox, and nothing says which Tailscale user/node requested it. Easy to approve the wrong or spoofed request. | medium |
| 4 | 2 Pair | Only the `02e` small print says "It connects over the internet to home.example.net:443, and your internet traffic goes out from home". The bond isn't over Tailscale (needs UDP 443 forwarded, per `13`) and the default is full tunnel, unlike Tailscale's exit-node-off default. Not mentioned on the phone page or onboarding. | medium |
| 5 | 2 Pair | `02b` mentions "or a Flint 3's admin panel → VPN → tailbond", which doesn't apply to my homebox server. | low |
| 6 | 2 Pair | "Or use the home server's page" card with a permanently greyed-out "Open home server page" button, with no reason given; redundant while already asking. | low |
| 7 | 2 Pair | "Keep this screen open" vs "that page works on this phone too": a self-hosting admin on the same phone must leave the screen to approve. | low |
| 8 | 3 Connect | ACL identity unclear: "10.77.0.2 — Your Tailscale machines can reach this phone at this address, through homebox". Doesn't say whether tailnet peers see my traffic as homebox (its tags/ACLs) or as a subnet-routed 10.77.0.2 (which needs route approval in the admin console). | medium |
| 9 | 4 Copy | Row copy icon in Machines list copies the IPv4 ("Copied 100.101.2.2") while the row's headline is the MagicDNS name; which value gets copied isn't indicated before tapping. | low |
| 10 | 4 Copy | No screen says whether other subnet routers' routes or tailnet DNS settings (split DNS / nameservers) apply through homebox. | low |
| 11 | 6 Save data | Five overlapping data controls (home switch, mode radio, per-link "Use for tailbond", "Advanced: main or backup", "Mobile data limit", plus "Faster switch to backup"); explained, but heavy. | low |
| 12 | 7 Hotel Wi-Fi | Two instructions on one card: "Sign in to Wi-Fi" button vs "Tap the Wi-Fi network (it says "Sign in to network")"; unclear which to use. | low |

Totals: 0 high / 4 medium / 8 low
