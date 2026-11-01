# Round 10 — Android persona B (homelab)

Persona: homelab power user. I run Tailscale subnet routers and use the official Tailscale Android app daily. I care about routes, DNS, ACL identity, which traffic goes where, and whether sideloading and pairing are safe. I only had the task script, `screens.md` and the PNGs in `web/` and `android/`.

Overall: this is a careful, honest flow. The things I usually have to dig for are on screen. The install page gives a SHA-256 and the `sha256sum` command. The pair dialog says the phone appears to the tailnet as homebox under homebox's ACLs. The exit-node default is stated three times. The cert-mismatch error refuses to continue. The captive-portal card explains that Android's sign-in page bypasses the tunnel. I could finish all eight tasks. My findings are about what a power user still can't check from the phone. The main one is the identity/ACL consequence, which shows up late and can't be checked.

---

## Task 1: Get the tailbond app onto the phone

**Path:** With Tailscale connected, I type `homebox/` in Chrome. `web/web-00a-phone-page-first-screen` opens: "Get tailbond on this phone". I read the drawn dialogs on `web/web-00-phone-page`, then tap **"Download tailbond (27 MB)"**. The drawn steps are: ① "File might be harmful" → Download anyway, then Open. ② "Tap Settings". ③ "Allow from this source". ④ "Install". ⑤ Play Protect "Scan app". Then: "When it says App installed, tap Done and come back to this page (Chrome, this tab) for step 2."

**What works:** The page warns "Read the pictures below first (or take a screenshot): once you open the file, Android's own screens cover this page." That's exactly the problem with sideload guides. The note "the pictures below are drawings, not buttons" stops people tapping the illustrations. "You can turn Allow from this source off again afterwards" is good hygiene. The fallback for a missed Open toast ("Chrome's menu ⋮ → Downloads … Tap tailbond.apk") is right. `web-00b` covers tapping step 2 too early: "tailbond isn't on this phone yet, so step 2 brought you back here."

**Security read:** "Is this safe? This page starts with http://, but your phone reached it through Tailscale, which encrypts it and makes sure it really comes from homebox". That's accurate for a MagicDNS name over WireGuard. The SHA-256 plus "`sha256sum /usr/share/tailbond/tailbond.apk` on homebox" lets me check it over SSH. What I'd also want is the APK **signing certificate** fingerprint (what `apksigner verify --print-certs` prints). Android pins updates to that, not to the file hash. A file hash only proves this one download.

- **L1 (low):** Only a file hash is offered. No signer-cert fingerprint, so I can't tell whether future updates come from the same key.

## Task 2: Add the phone to homebox (pair), from the phone

**Path:** Back in Chrome I tap **"Open tailbond and ask homebox"** in step 2 (`web-00`). The app opens on `02a`, "Asking homebox… Looking for homebox on your Tailscale network." Then `02b`, "Waiting for approval on homebox…". It says the request shows on homebox's page as **"pixel-8 · sam@example.com · 100.101.8.8"**. That's the phone's real tailnet identity, so the approver can match it in the admin console. I run homebox myself, so I follow the web hint: "Approve it at http://homebox:8077: the request is at the top, and that page works on this phone too." Tailscale is still up, so that works. After approval, `02e` shows "Add this phone to homebox?", where I tap **Add phone**.

**What works:** The failure states are clear. `02c` "Can't find homebox: Check that the Tailscale app says Connected, and that the name matches the one in its list of machines." `02d` "homebox said no". The "keeps waiting for 10 minutes" note means I can switch apps to approve.

**ACL identity:** `02e` says: "Your Tailscale machines stay reachable through it; **they see this phone as homebox, so homebox's Tailscale access rules apply (not your own account's).**" This is the single most important security fact in the product, and it's stated honestly. But:
- it first appears on the last confirmation, after I've already sideloaded and someone has approved. It isn't on the install page (`web-00`, which only says "Your Tailscale machines stay reachable through homebox"). It isn't on onboarding (`01`, "Still on your tailnet") or on the switch dialog (`04`) either.
- homebox is a server. In my tailnet a server like that usually has a tag with broader (or different) grants than my user. Nothing on the phone shows what homebox's rules allow. I also can't tell whether the Machines list (`10-home-connected-full`) is "what homebox can reach" or "everything on the tailnet". To judge the trade-off I'd have to go read my ACL file.

- **M1 (medium):** The phone switching to homebox's ACL identity is disclosed only at the final "Add phone" step. Its consequences (what the phone can now reach) can't be checked anywhere in the app.
- **L2 (low):** `02c`/`02d` hint "Not sure of the name? It's the machine that runs tailbond". "tailbond" is also the app I'm holding, so "the machine that runs tailbond" reads oddly. "The machine that runs the tailbond server" would be clearer.

## Task 3: Connect

**Path:** On the main screen (`12-home-disconnected`) I tap the big switch. Because Tailscale's VPN is up I get `04-handoff-other-vpn`, "Switch from Tailscale to tailbond?" It says: "Turning on tailbond turns off Tailscale (or any other VPN)… Your internet traffic goes out from homebox too (like an exit node; Settings changes that)." It also warns about Always-on VPN: "If tailbond switches itself off right after this, Tailscale is set as Always-on VPN… Android doesn't let tailbond check this setting." I tap **Turn on tailbond**, accept Android's VPN prompt (not pictured), and go through `11-home-connecting` ("Reaching homebox…") to `10-home-connected`.

**What works:** The exit-node default and the Always-on gotcha are both stated before anything changes. I'd have hit that gotcha myself. `13-error-unreachable` gives me something I can act on: "the home router may need to forward UDP port 443 to it". `14-error-cert` ("the server answering isn't the homebox this phone was paired with… Don't pair this phone again unless they ask you to") fails closed and doesn't offer a "continue anyway". That's the right call.

- **L3 (low):** "This device" says "10.77.0.2 · Not a Tailscale address… They can reach the phone at this address through homebox." For a tailnet peer to route to 10.77.0.2, homebox has to advertise that range as a subnet route and it has to be approved. The screen doesn't say whether that's set up. As a subnet-router user I also want to know the range, in case it overlaps one of my LAN or advertised routes.

## Task 4: Find and copy the NAS's MagicDNS name or Tailscale IP

**Path:** On `18-machines-search-copied` I type `nas` in "Search machines". The `nas` row shows `nas.tail1234.ts.net` and `100.101.2.2 · linux`. Tapping the row's copy icon gives the toast **"Copied 100.101.2.2"**. Tapping the row opens `19-machine-detail`: "Tap an address to copy it". It has separate copy buttons for MagicDNS name, Short name, Tailscale IPv4 and IPv6, and tapping the MagicDNS name gives "Copied nas.tail1234.ts.net". That's better than the official app's long-press menu.

"Reached through homebox. With tailbond on, apps on this phone can use these addresses just like with Tailscale." That answers my DNS question for the default mode.

- **L4 (low):** The list row has one copy icon, but the row shows both a name and an IP. I only learn that it copies the IP from the toast. A user who wanted the MagicDNS name has to go into the detail screen.
- **L5 (low):** Settings → "Only Tailscale machines" ("only your tailnet goes through home") doesn't say whether that includes **subnet routes** advertised by my subnet routers, or whether MagicDNS / split DNS still resolve in that mode. "Which traffic goes where" is exactly what I'd want spelled out. No screen mentions subnet routes at all.

## Task 5: Tell whether Wi-Fi and 5G are both being used right now

**Path:** `10-home-connected` has the heading **"Using both Wi-Fi and mobile data"**. Wi-Fi and Mobile data each show "In use" with ↓/↑ Mbps. The status card says "Through homebox · Wi-Fi + mobile data". Tapping the Connections area opens `20-connections`, with the aggregate (61.1 / 7.0 Mbps, which matches 38.4 + 22.7), per-link delay, and "2.05 GB this session". It reads at a glance.

- **L6 (low):** When a link fails (`15-one-path-failed`), Mobile data just says "Not working right now". There's no reason (no signal? blocked UDP? carrier NAT?). A power user wants a hint at the cause.

## Task 6: Make it save mobile data

**Path:** I flip **"Save mobile data — Use mobile data only when Wi-Fi stops working"** on the home screen. `16-save-data-standby` then shows "Using Wi-Fi · saving mobile data (backup only)" and Mobile data "Not in use · saving data". `17-save-data-takeover` shows the failover: "Wi-Fi isn't working · using mobile data", with "In use · taking over". On `21-connections-save-data`, "The Save mobile data switch on the home screen picks 'Save mobile data' here" ties the switch to the mode radio. The "Mobile data limit" ("Separate from Save mobile data") is a hard monthly cap.

- **L7 (low):** There are five overlapping data controls: the home switch, the mode radio, the per-link "Use for tailbond" toggle, "Faster switch to backup (experimental) … Uses a little mobile data", and the monthly limit. Each one is explained, but it's a lot to read to be sure I'm not leaking cellular. The limit also resets "until the 1st of next month", which won't match a carrier billing cycle that starts mid-month.

## Task 7: Hotel Wi-Fi with a sign-in page

**Path:** `16c-home-wifi-needs-sign-in` shows the card **"Wi-Fi needs sign-in"**: "tailbond keeps you connected over mobile data meanwhile, and adds Wi-Fi by itself once you're signed in." The Wi-Fi tile says "Needs sign-in · not used yet". I tap **"Sign in to Wi-Fi"**. The fine print, "Android's sign-in page uses the Wi-Fi directly, so you don't need to turn tailbond off", answers my real worry: a full-tunnel VPN usually breaks captive portals. Settings confirms the protection: "tailbond never bonds over a Wi-Fi until Android says it's online."

- **L8 (low):** The card has two instructions. There's a primary button, "Sign in to Wi-Fi", and fine print saying "Tap the Wi-Fi network (it says 'Sign in to network') to open the page." It's unclear whether the fine print is a fallback for when the button doesn't work, or describes Android's notification. "If the button doesn't open it, tap Android's 'Sign in to network' notification" would settle it.
- **L9 (low):** In Settings, the toggle "Tell me when Wi-Fi needs a sign-in" sits on the same row as "tailbond never bonds over a Wi-Fi until Android says it's online". Read literally, turning the toggle off might also drop that protection. Probably only the notification goes away, but the row doesn't say.

## Task 8: Turn tailbond off and go back to the Tailscale app

**Path:** On the home screen I tap **"Switch back to Tailscale"**. Settings describes it as "Turns tailbond off and opens the Tailscale app. Turn Tailscale on there if it says Disconnected". `12b-home-switched-back-to-tailscale` shows "Disconnected · Stopped. Back to Tailscale: turn it on in its app." plus the card "Last step: turn Tailscale on … Until then, your Tailscale machines aren't reachable." and an **Open Tailscale** button. That's correct and honest: Android doesn't let one VPN app start another's tunnel.

- **L10 (low):** Settings recommends "turn on Always-on VPN for tailbond… Leave 'Block connections without VPN' off if you want to switch back to Tailscale easily". But the switch-back flow (`12b`) never says that I also need to move Always-on back to Tailscale if I'd set it on tailbond. If I don't, the phone could fight me on the next reboot.
- **L11 (low):** Settings says this "opens the Tailscale app". But `12b` is still tailbond's screen, with its own "Open Tailscale" button. It also still shows "Will use Wi-Fi and mobile data together" with both links "Ready", which is noise once I've left tailbond. I can't tell whether Tailscale actually opened or whether I'm expected to tap the button.

---

## Findings

| # | task | finding | severity |
|---|---|---|---|
| M1 | 2 | "they see this phone as homebox, so homebox's Tailscale access rules apply (not your own account's)" first appears on the final Add-phone dialog (not on the install page, onboarding, or switch dialog). Nothing in the app shows what homebox's ACL lets the phone reach, or whether the Machines list reflects that. | medium |
| L1 | 1 | Only a file SHA-256 is offered. No APK signing-cert fingerprint, so I can't verify that future updates come from the same key. | low |
| L2 | 2 | "It's the machine that runs tailbond" is confusing, because tailbond is also the phone app. | low |
| L3 | 3 | "They can reach the phone at this address [10.77.0.2] through homebox" needs an approved subnet route on homebox, which isn't mentioned. The range isn't shown either, so I can't check for overlap with my LAN or advertised routes. | low |
| L4 | 4 | The list row's single copy icon copies the IP (only the "Copied 100.101.2.2" toast tells me). Copying the MagicDNS name means going into the detail screen. | low |
| L5 | 4 / Settings | "Only Tailscale machines — only your tailnet goes through home" doesn't say whether subnet routes or MagicDNS/split DNS are included. | low |
| L6 | 5 | "Not working right now" for a failed link gives no cause. | low |
| L7 | 6 | Five overlapping data controls. The monthly limit resets on "the 1st of next month", not on the carrier billing date. | low |
| L8 | 7 | The "Sign in to Wi-Fi" button and the fine print "Tap the Wi-Fi network (it says 'Sign in to network')" read as two competing instructions. | low |
| L9 | 7 | The "Tell me when Wi-Fi needs a sign-in" toggle shares a row with "never bonds over a Wi-Fi until Android says it's online", so it's unclear whether turning it off removes that protection. | low |
| L10 | 8 | The switch-back flow doesn't remind me to move Always-on VPN back to Tailscale if I'd set it on tailbond, as Settings suggests. | low |
| L11 | 8 | Settings says "opens the Tailscale app", but `12b` shows tailbond with its own "Open Tailscale" button, plus stale "Will use Wi-Fi and mobile data together" / "Ready" tiles. | low |

Totals: 0 high / 1 medium / 11 low
