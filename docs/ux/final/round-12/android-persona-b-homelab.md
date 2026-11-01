# Round 12 — Android persona B (homelab)

**Persona:** I run Tailscale subnet routers at home and use the official Tailscale Android app daily. I care about routes, DNS, ACL identity, where each kind of traffic goes, and how safe sideloading and pairing are.

**Materials used:** `task-script.md`, `screens.md` and every PNG in `web/` and `android/`, including the dark variants. I read the long phone page (`web-00-phone-page.png`) in cropped sections so I could read the actual text.

**Overall:** I could finish all 8 tasks from these screens. Nothing blocked me. The app is unusually honest about the things I care about: it says the traffic goes out like an exit node, that tailnet peers see the phone as homebox, that the phone has no 100.x address, and that it connects over the internet to `home.example.net:443`, with a pinned-identity error (14). My one real gap is subnet routes. The rest is polish.

---

## Task 1: Get the tailbond app onto the phone

**Where I tap:** I type `homebox/` in Chrome and get `web-00a` ("Get tailbond on this phone"). I scroll past the five drawn Android dialogs ("File might be harmful" → **Download anyway**, "For your security…" → **Settings**, "Allow from this source" toggle, "Do you want to install this app?" → **Install**, "App scan recommended" → **Scan app**). Then I tap **Download tailbond (27 MB)** and follow the "In short: Download anyway → Open → Settings → Allow from this source → back → Install (→ Scan app, if asked)" line.

**What works for me:**
- "Is this safe? This page starts with http://, but your phone reached it through Tailscale, which encrypts it and makes sure it really comes from homebox." This is the right argument: the transport is WireGuard and the peer is authenticated. They also give me the SHA-256 plus `sha256sum /usr/share/tailbond/tailbond.apk` to check it on homebox.
- "You can turn Allow from this source off again afterwards; tailbond stays installed." Good hygiene for sideloading.
- `web-00b` handles me tapping step 2 too early: "tailbond isn't on this phone yet, so step 2 brought you back here. Install it first, then tap step 2 again."
- `web-00c` on a computer gives a QR code and says "nothing here changes homebox's settings."

**Where I'd stumble:**
- On the phone the hash wraps badly: "SHA-256 `5`" sits alone at the end of one line and `4167117c3586c8ff05cc983c6e41381d9156…` continues on the next. When comparing by eye, I'd likely drop the leading `5`. The only comparison offered is running `sha256sum` on homebox. Nothing on the phone shows the hash of the file I actually downloaded. **Low.**
- The page shows "tailbond 0.1.0-r42 on homebox" but says nothing about updates. For a sideloaded VPN app I want to know whether updates come from the same page and are signed with the same key. Coming from the Play Store, I'd expect updates to just happen. **Low.**

## Task 2: Add the phone to homebox (pair), from the phone

**Where I tap:** Back in Chrome, step 2 → **Open tailbond and ask homebox** → `02a` "Asking homebox… Looking for homebox on your Tailscale network." → `02b` "Waiting for approval on homebox…" → after approval, `02e` "Add this phone to homebox?" → **Add phone**.

**What works for me:**
- `02b` identifies the request the way I'd expect from Tailscale: "There it shows as pixel-8 · sam@example.com · 100.101.8.8, so they can tell it's this phone." That is the Tailscale node, user and IP. Good for ACL-minded approval.
- The web page says only the owner or people they named can approve: "Nobody else on your Tailscale network can approve from there unless you named them."
- `02c` "Can't find homebox — Check that the Tailscale app says Connected, and that the name matches the one in its list of machines" and `02d` "homebox said no" are specific and actionable.
- `02e` spells out the trust model before I commit: "They see this phone as homebox (not as your own phone), so they allow it whatever they allow homebox" and "It connects over the internet to home.example.net:443". There is also a **Technical details** link.

**Where I'd stumble:**
- "home.example.net:443" doesn't say which protocol. A homelab person reads `:443` as TCP/HTTPS. Only the failure screen (`13`) reveals "the home router may need to forward **UDP** port 443". If I were setting up the forward myself, I'd get it wrong first. **Low.**
- The ACL identity trade-off is disclosed well, in `01`, `02e`, `web-00` step 3 and the home screen. But nothing on the phone shows *what* homebox is allowed. If homebox is a tagged server with wide access, every paired phone silently gets that wide access. Each paired phone also shows up in ACLs and logs as homebox, not as a distinct user or device. I can live with that because it's stated, but I'd want a pointer such as "check homebox's ACL in the admin console". **Low.**

## Task 3: Connect

**Where I tap:** On the home screen (`12`), I tap the big switch. `04` says "Switch from Tailscale to tailbond? … Turning on tailbond turns off Tailscale (or any other VPN)… Your internet traffic goes out from homebox too (like an exit node; Settings changes that)… Next, Android asks you to allow tailbond's connection request. Tap OK." I tap **Turn on tailbond**, then OK on Android's dialog, then see `11` "Connecting… Reaching homebox…" and `10` "Connected · Through homebox · Wi-Fi + mobile data".

**What works for me:**
- The Always-on note is exactly the trap I'd hit: "If tailbond switches itself off right after this, Tailscale is set as Always-on VPN on this phone: turn that off in Android Settings → VPN → Tailscale's gear." The same goes for the "Block connections without VPN" advice in Settings.
- Exit-node-style default traffic is stated up front, and I know where to change it.

**Where I'd stumble:**
- `10`, This device: "10.77.0.2 · Not a Tailscale address… They can open connections to the phone at this address only if homebox's tailbond route is approved in the Tailscale admin console". As a subnet-router user I know what to do with this, but I can't act on it from the screen. It doesn't say which route (the CIDR I'd look for under homebox's subnet routes) or whether it is approved right now. **Low.**

## Task 4: Find and copy the NAS's MagicDNS name or Tailscale IP

**Where I tap:** On the home screen, I type `nas` in **Search machines** (`18`) and tap the copy icon, which shows the toast "Copied 100.101.2.2". Or I tap the row to open `19`, which lists MagicDNS name `nas.tail1234.ts.net`, Short name `nas`, Tailscale IPv4 `100.101.2.2` and Tailscale IPv6 `fd7a:115c:a1e0::2`, each with its own copy icon, and shows the toast "Copied nas.tail1234.ts.net". Done.

**Where I'd stumble:**
- In the list (`18`), the row shows the MagicDNS name on top and the IP below, but there is a single copy icon. I only find out from the toast that it copied the IP, not the name. In the Tailscale app I'd long-press and pick which one. **Low.**
- **Subnet routes.** My homelab has LANs behind other Tailscale subnet routers. The only place routes are mentioned is inside the *non-default* option in Settings (`30`), "Only Tailscale machines": "LANs behind other subnet routers aren't reachable through tailbond." With the default "Through your home server" I can't tell whether `192.168.x` behind my other routers works. Nothing on the home screen, onboarding, `04` or the machine list says either way, and the list doesn't show which machines advertise routes. I'd have to test by trial and error, and that's the first thing I'd check before ditching the Tailscale app. **Medium.**
- The "Only Tailscale machines" description says "only your Tailscale machines (100.x addresses) go through home". `19` shows an IPv6 address and says "apps on this phone can use these addresses just like with Tailscale". In split mode, does `fd7a:115c:a1e0::/48` also go through home? The screens don't say. **Low.**

DNS is handled well: "Names are still looked up through home, so MagicDNS names work" answers my split-DNS question for the split mode.

## Task 5: Tell whether Wi-Fi and 5G are both being used right now

**Where I look:** On `10`, the Connections heading says "Using both Wi-Fi and mobile data". Both tiles show a green "In use" with rates (↓38.4 ↑4.1 Mbps and ↓22.7 ↑2.9 Mbps). Tapping into `20` gives totals that add up (61.1 / 7.0 Mbps), per-link delay (21 ms / 48 ms) and "2.05 GB this session". `15` clearly shows the degraded case: "Using Wi-Fi only", with Mobile data "Not working right now" in red.

No issues. This is clearer than anything the Tailscale app gives me.

## Task 6: Make it save mobile data

**Where I tap:** On the home screen, the **Save mobile data** toggle, "Use mobile data only when Wi-Fi stops working". `16` then shows "Mode: Save mobile data", "Using Wi-Fi · saving mobile data (backup only)" and Mobile data "Not in use · saving data". `17` shows it taking over: "Wi-Fi isn't working · using mobile data", "In use · taking over".

**Where I'd stumble:**
- `20`/`21` have several overlapping data controls: the mode radios ("The Save mobile data switch on the home screen picks 'Save mobile data' here"), the per-link **Use for tailbond** toggles, **Mobile data limit** ("Separate from Save mobile data…") and **Faster switch to backup (experimental)** ("Uses a little mobile data"). Each one is explained on its own, but nowhere says what turning the home toggle *off* returns to: Fastest, or whatever mode I had before? Nor does anything say how "Use for tailbond" off on mobile data differs from Save mobile data (no failover at all). **Low.**

## Task 7: Hotel Wi-Fi with a sign-in page; get it back into the bond

**Where I tap:** `16c` shows a card: "Wi-Fi needs sign-in · This Wi-Fi shows a sign-in page before it lets you online. tailbond keeps you connected over mobile data meanwhile, and adds Wi-Fi by itself once you're signed in." I tap **Sign in to Wi-Fi**, sign in on Android's page, and Wi-Fi rejoins on its own. Settings confirms the safety rule: "tailbond never bonds over a Wi-Fi until Android says it's online."

**What works for me:** "Android's sign-in page uses the Wi-Fi directly, so you don't need to turn tailbond off". That's the exact worry I'd have with any full-tunnel VPN.

**Where I'd stumble:**
- The button says **Sign in to Wi-Fi**, but the small print says "Tap the Wi-Fi network (it says "Sign in to network") to open the page." I can't tell whether the button opens the portal directly or drops me in Android's Wi-Fi list where I tap the network again. It's a small two-step ambiguity. **Low.**
- No screenshot shows the moment Wi-Fi rejoins, so I'm trusting "adds Wi-Fi by itself". The tiles on `10` would show it, so this isn't a finding.

## Task 8: Turn tailbond off and go back to the Tailscale app

**Where I tap:** **Switch back to Tailscale** sits right under the status card on every home state (`10`, `11`, `12`, `13`, `14`, `15`, `16`, `16c`, `17`) and in Settings ("Turns tailbond off and opens the Tailscale app. Turn Tailscale on there if it says Disconnected"). Back in tailbond, `12b` shows "Disconnected · Stopped. Back to Tailscale: turn it on in its app" and a **Last step: turn Tailscale on** card with **Open Tailscale**: "Until then, your Tailscale machines aren't reachable."

No issues. It's easy to find and honest that Tailscale doesn't come back on by itself, which matches how Android hands VPN ownership back.

---

## Findings

| # | Task | Finding | Severity |
|---|---|---|---|
| 1 | 4 | Subnet routes: only the non-default "Only Tailscale machines" option says "LANs behind other subnet routers aren't reachable through tailbond"; with the default "Through your home server" no screen says whether LANs behind my other subnet routers work, and the machine list doesn't show advertised routes | medium |
| 2 | 1 | SHA-256 wraps with a lone "5" ("SHA-256 `5` / `4167117c…`"), easy to misread; it can only be checked on homebox, not against the file on the phone | low |
| 3 | 1 | Nothing about how a sideloaded "tailbond 0.1.0-r42" gets updated or whether updates are signed with the same key | low |
| 4 | 2 | `02e` "connects over the internet to home.example.net:443" doesn't say UDP; only error `13` reveals "forward UDP port 443" | low |
| 5 | 2 | "They see this phone as homebox… allow it whatever they allow homebox" is well disclosed, but the phone gives no hint how to see what homebox is allowed; every paired phone gets the server's ACL rights and one shared identity in logs | low |
| 6 | 3 | "only if homebox's tailbond route is approved in the Tailscale admin console" names no CIDR and shows no approved/not-approved status | low |
| 7 | 4 | List row copy icon (`18`) copies the IP ("Copied 100.101.2.2") although the MagicDNS name is shown first; which one it copies is only learned from the toast | low |
| 8 | 4 | Split mode says "(100.x addresses) go through home"; it's unclear whether Tailscale IPv6 (`fd7a:115c:a1e0::2`, shown on `19`) does too | low |
| 9 | 6 | Overlapping data controls (home toggle, mode radios, per-link "Use for tailbond", monthly limit, "Faster switch to backup"); doesn't say what the home toggle returns to when switched off | low |
| 10 | 7 | **Sign in to Wi-Fi** button vs "Tap the Wi-Fi network (it says "Sign in to network")": unclear whether the button opens the portal or the Wi-Fi list | low |

Totals: 0 high / 1 medium / 9 low
