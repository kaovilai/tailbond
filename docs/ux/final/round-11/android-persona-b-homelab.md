# Round 11 — Android persona B (homelab)

Persona: homelab power user. I run Tailscale subnet routers and use the official Tailscale Android app every day. I care about what traffic goes where (routes, exit node, DNS), which ACL identity my packets carry, and whether sideloading and pairing are safe.

Materials: `task-script.md`, `screens.md`, and the PNGs in `web/` and `android/` (round-11 packet). I looked at every screenshot, including the dark variants (10, 04, 20, 30). I found no contrast or legibility problems in dark mode.

---

## Task 1: Get the tailbond app onto the phone

**Path:** With Tailscale on, I type `homebox/` in Chrome. That opens **web-00a** ("Get tailbond on this phone"). I scroll past the five drawings of Android's dialogs (web-00) to **"Download tailbond (27 MB)"**. Then I go through Chrome's "File might be harmful" → "Download anyway" → Open, then "Settings" → "Allow from this source" → back, then "Install", then "Scan app" if Play Protect asks. The page also covers the case where Chrome's Open message disappears: "Chrome's menu ⋮ → Downloads, or the Files app → Downloads. Tap tailbond.apk." If I tap step 2 too early, **web-00b** tells me "tailbond isn't on this phone yet, so step 2 brought you back here." That is a good recovery.

**What worked:** I finished this task. The drawings match what Android really shows, and "Look at these five pictures first (or take a screenshot): once you open the file, Android's own screens cover this page" is good advice. The page tells me I can turn "Allow from this source off again afterwards; tailbond stays installed", and that's the right security reminder.

**Where I hesitated (as a security-minded user):**
- "Is this safe? This page starts with http://, but your phone reached it through Tailscale, which encrypts it and makes sure it really comes from homebox". The page claims this but can't know it. If I typed `homebox/` with Tailscale off, or on a network whose DNS search domain answers for `homebox`, I'd get some other host and the same reassuring text. In the Tailscale world I'd check that the name is the MagicDNS FQDN or a 100.x address. The page doesn't show me `homebox.tail1234.ts.net` or homebox's 100.x address to compare against the address bar. **Low.**
- The SHA-256 is served from the same page as the APK, and the only way to check it is "`sha256sum /usr/share/tailbond/tailbond.apk` on homebox", so I need a shell on homebox. Fine for me. But nothing shows the APK **signing-certificate** fingerprint, which is what Android actually enforces on future updates. I also see nothing about how updates arrive for a sideloaded app. **Low.**

---

## Task 2: Add the phone to homebox (pair), from the phone

**Path:** On web-00 step 2, I tap **"Open tailbond and ask homebox"**. That leads to **02a** ("Asking homebox… Looking for homebox on your Tailscale network."), then **02b** ("Waiting for approval on homebox… There it shows as pixel-8 · sam@example.com · 100.101.8.8, so they can tell it's this phone… tailbond keeps waiting for 10 minutes"), then **02e** ("Add this phone to homebox?") → **"Add phone"**.

**What worked:** I finished this task. Using my Tailscale identity (node name, login, 100.x) so the approver can recognise the request is exactly what I'd want. The failure screens are clear: 02c "Can't find homebox — Check that the Tailscale app says Connected", 02d "homebox said no". 02e spells out the identity consequence: "They see this phone as homebox (not as your own phone), so they allow it whatever they allow homebox." It also gives the public endpoint, "It connects over the internet to home.example.net:443". The pinning behaviour in 14 ("the server answering isn't the homebox this phone was paired with… Don't pair this phone again unless they ask you to") tells me the pairing is bound to a server identity. Good.

**Where I got stuck or doubtful:**
- **Who is allowed to approve?** Web-00 step 2 says "You run homebox yourself? Approve it at http://homebox:8077: the request is at the top, and that page works on this phone too." 02b repeats "approve "pixel-8" on homebox's tailbond page". Nothing tells me what stops any other tailnet user, or the requesting phone itself, from opening :8077 and approving. Is it gated on homebox's owner identity, tailnet admin, or nothing? The page invites self-approval from the same phone, so I can't tell whether approval means anything. I'd go read homebox's config before trusting it. **Medium.**
- 02e gives "home.example.net:443" with no protocol, and Settings (30) says "Reached at home.example.net:443". Only the error screen 13 says "forward UDP port 443". A homelab user setting up port forwarding from 02e or Settings would assume TCP/HTTPS. **Low.**
- 02e covers what the phone *gains* ("allow it whatever they allow homebox"). It doesn't say what it *loses*: grants written for `sam@example.com` or for pixel-8's own node no longer apply. On the tailnet side, this phone and any other paired phone become indistinguishable from homebox in ACLs and logs. **Low.**
- 02 / 02c helper text: "Not sure of the name? It's the machine that runs tailbond". The phone also runs tailbond now. They mean "the server side of tailbond". **Low.**

---

## Task 3: Connect

**Path:** On the main screen (12, "Disconnected"), I tap the big switch. Because Tailscale's VPN is on, I get **04** "Switch from Tailscale to tailbond?" → **"Turn on tailbond"**, then Android's connection request → OK. Then **11** "Connecting… Reaching homebox…" and **10** "Connected · Through homebox · Wi-Fi + mobile data".

**What worked:** I finished this task. 04 is honest and matches how Android works: "Android runs one VPN at a time. Turning on tailbond turns off Tailscale (or any other VPN)". It covers exit-node behaviour: "Your internet traffic goes out from homebox too (like an exit node; Settings changes that)". It also covers the Always-on trap: "If tailbond switches itself off right after this, Tailscale is set as Always-on VPN… Android doesn't let tailbond check this setting." That last one would have caught me. 13 and 14 are good failure screens.

**Where I got stuck or doubtful:**
- **Inbound exposure of the phone.** Under "This device" (10): "10.77.0.2 — Not a Tailscale address: your Tailscale machines see this phone as homebox and allow it whatever they allow homebox. They can reach the phone at this address through homebox." Who exactly can open connections *to* my phone? Every tailnet node that can reach homebox? Only homebox? Is 10.77.0.0/x advertised as a subnet route that I'd have to approve in the admin console? Is there any inbound filter on the phone? In the Tailscale app my phone is a normal node and ACLs govern inbound. Here I can't tell, and I'd hesitate to leave it on. **Medium.**
- **Routes and DNS aren't visible.** Nothing tells me whether subnet routes on my tailnet (my 192.168.x LAN behind a subnet router, or ones advertised by homebox itself) are reachable through tailbond. It doesn't say which DNS resolver the phone uses either (MagicDNS 100.100.100.100 through homebox? split DNS? the carrier's?), in "Through your home server" or in "Only Tailscale machines… only your tailnet goes through home" (30). In the Tailscale app I can see and toggle subnet routes, the exit node and DNS. I'd have to test by hand to find out what "tailnet" covers here. **Medium.**

---

## Task 4: Find and copy the NAS's MagicDNS name or Tailscale IP

**Path:** On the main screen I type `nas` in **"Search machines"**, then tap the copy icon on the nas row. **18** shows the toast "Copied 100.101.2.2". Alternatively I tap the row, which opens **19**: "Tap an address to copy it", with MagicDNS name `nas.tail1234.ts.net`, Short name `nas`, Tailscale IPv4 `100.101.2.2` and IPv6 `fd7a:115c:a1e0::2`. Copying gives "Copied nas.tail1234.ts.net".

**What worked:** I finished this task quickly. Labelling the detail page with "MagicDNS name" and "Tailscale IPv4/IPv6" is more precise than the Tailscale app. "With tailbond on, apps on this phone can use these addresses just like with Tailscale" answers my question about short names and 100.x.

**Minor:** The list row puts the MagicDNS name first (`nas.tail1234.ts.net`), but its copy icon copies the IP. I only learn which one from the toast. **Low.**

---

## Task 5: Tell whether Wi-Fi and 5G are both in use

**Path:** Main screen **10**, Connections: "Using both Wi-Fi and mobile data". Wi-Fi shows "In use ↓38.4 ↑4.1 Mbps" and Mobile data shows "In use ↓22.7 ↑2.9 Mbps". The status card also says "Wi-Fi + mobile data". Tapping into **20** gives per-link throughput, "delay 21 ms" / "delay 48 ms", and "2.05 GB this session". **15** shows the broken case clearly: "Using Wi-Fi only… Mobile data · Not working right now".

No problems. Clearer than anything the Tailscale app shows.

---

## Task 6: Make it save mobile data

**Path:** Main screen: **"Save mobile data"** toggle ("Use mobile data only when Wi-Fi stops working"). That leads to **16**: "Using Wi-Fi · saving mobile data (backup only)", "Mobile data · Not in use · saving data", "Mode: Save mobile data". **17** shows the failover: "Wi-Fi isn't working · using mobile data… In use · taking over". **21** has the same choice as a radio button, plus "Faster switch to backup (experimental)… Uses a little mobile data" and a separate "Mobile data limit".

No problems. 20/21 explain that the home switch and the radio list are the same setting ("The Save mobile data switch on the home screen picks "Save mobile data" here"), and that the monthly limit is separate.

---

## Task 7: Hotel Wi-Fi with a sign-in page

**Path:** **16c** shows the "Wi-Fi needs sign-in" card ("tailbond keeps you connected over mobile data meanwhile, and adds Wi-Fi by itself once you're signed in") → **"Sign in to Wi-Fi"**. After I sign in on Android's page, Wi-Fi rejoins by itself. Settings (30) has "Tell me when Wi-Fi needs a sign-in… tailbond never bonds over a Wi-Fi until Android says it's online."

**What worked:** I finished this task. "Android's sign-in page uses the Wi-Fi directly, so you don't need to turn tailbond off" answers exactly what I'd worry about with any VPN on a captive portal.

**Minor:** The card has a big "Sign in to Wi-Fi" button, and under it the small print says "Tap the Wi-Fi network (it says "Sign in to network") to open the page". Do I tap the button, or find the network in Android's notification or Wi-Fi list? The small text reads like a second, different instruction, not a fallback. **Low.**

---

## Task 8: Turn tailbond off and go back to the Tailscale app

**Path:** Main screen **"Switch back to Tailscale"** → **12b**: "Disconnected · Stopped. Back to Tailscale: turn it on in its app." Below that is the card "Last step: turn Tailscale on… Until then, your Tailscale machines aren't reachable" with **"Open Tailscale"**. In the Tailscale app I tap Connect (not shown).

**What worked:** I finished this task. The button is on the main screen in every state (10, 11, 13, 14, 15, 16c), and 12b clearly says the tailnet is unreachable until Tailscale is back on.

**Minor:**
- Settings' row says "Switch back to Tailscale — Turns tailbond off and opens the Tailscale app". The main-screen result (12b) instead lands on tailbond's own "Last step" card with an "Open Tailscale" button. Does it open Tailscale for me or not? **Low.**
- Web-00 step 3 says "Android asks once to allow its connection". From switching between VPN apps, I know Android shows the "Connection request" dialog again whenever the other app was the last one allowed. Going back to Tailscale, and later back to tailbond, will probably prompt again. "Once" is a promise Android won't keep. **Low.**

---

## Findings

| # | Task | Finding | Severity |
|---|---|---|---|
| 1 | 2 | Nothing says who can approve a pairing. "Approve it at http://homebox:8077… that page works on this phone too" invites self-approval from the requesting phone, and I can't tell if any tailnet user could approve. | medium |
| 2 | 3 | Inbound exposure is unclear: "They can reach the phone at this address [10.77.0.2] through homebox". Who can connect to the phone? Is it a subnet route? Is there any inbound filter? | medium |
| 3 | 3/4 | No visibility of subnet routes or DNS. "Only Tailscale machines… only your tailnet goes through home" doesn't say whether other subnet routers' LANs are included or which resolver handles MagicDNS/split DNS. The Tailscale app shows both. | medium |
| 4 | 1 | "Is this safe?… your phone reached it through Tailscale" is asserted, not shown. The page shows no ts.net name or 100.x address to check against the address bar. | low |
| 5 | 1 | The SHA-256 is on the same page as the APK and needs a shell on homebox to check. No APK signing-cert fingerprint and no update story. | low |
| 6 | 2 | 02e / Settings say "home.example.net:443" with no protocol. Only error 13 says "UDP port 443", so I'd forward TCP by mistake. | low |
| 7 | 2 | "allow it whatever they allow homebox" doesn't say that per-user (sam@) grants stop applying, or that paired phones look identical to homebox in ACLs and logs. | low |
| 8 | 2 | "Not sure of the name? It's the machine that runs tailbond". The phone runs tailbond too. | low |
| 9 | 4 | The list row shows the MagicDNS name first but its copy icon copies the IP ("Copied 100.101.2.2"). | low |
| 10 | 7 | The "Sign in to Wi-Fi" button plus "Tap the Wi-Fi network (it says "Sign in to network")" read as two different instructions. | low |
| 11 | 8 | Settings says Switch back "opens the Tailscale app", but 12b shows tailbond's own "Last step… Open Tailscale" card. Inconsistent. | low |
| 12 | 8 | Web page "Android asks once to allow its connection". Android re-asks after switching VPN apps back and forth. | low |

Totals: 0 high / 3 medium / 9 low
