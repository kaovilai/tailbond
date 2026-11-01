# Round 8 — Android persona B (homelab)

**Persona:** I run a homelab with Tailscale subnet routers and use the official Tailscale Android app every day. I care about routes, DNS, ACL identity, which traffic goes where, and how safe sideloading and pairing are. I use the official Tailscale app's behaviour as my baseline.

**Materials:** `task-script.md`, `screens.md`, and the PNGs in `web/` and `android/` (packet `out/ux/final/round-8/packet-android/`). I read every screen, cropped the long ones into sections, and checked the dark variants.

Packet note: `android/03-deeplink-confirm.png` isn't listed in the manifest. It looks identical to `02e-pairing-ask-approved-confirm.png` ("Add this phone to homebox?"), so I treated it as the same moment.

---

## Task 1 — Get the tailbond app onto the phone

**Where I tap:** With Tailscale connected, I type `homebox/` into Chrome and get `web-00a-phone-page-first-screen`: "Get tailbond on this phone", then step 1 "Install the app" with the button **"Download tailbond for Android (27 MB)"**. The page draws the four Android prompts I'll see (`web-00-phone-page`):
① "File might be harmful … Download anyway"
② "For your security, your phone currently isn't allowed to install unknown apps from this source… Settings"
③ "Install unknown apps / Chrome / Allow from this source" toggle
④ "Do you want to install this app? … Install"
Then: "When it says App installed, tap Done and come back to this page (Chrome, this tab) for step 2." If I tap step 2 too early, `web-00b` tells me: "tailbond isn't on this phone yet, so step 2 brought you back here. Install it first, then tap step 2 again." That's good recovery.

**What works:** The page is honest that sideloading is unusual. It says "tailbond isn't in the Play Store: it comes from homebox, so Android asks you to confirm a few times. That's expected." It also has the "Chrome's Open message gone…" fallback, and it mentions the update-signing rule: "Later updates must be signed with the same key, or Android refuses them."

**Where I hesitate (security):**
- **No independent way to trust the APK.** The page says "This page starts with http://, but it comes to you through Tailscale, which encrypts it. To check the file: SHA-256 `54167117c3586c8ff05cc983c6e41381d91562441c4824a998ed8192dae87e9c`." But I reached it by the short name `homebox/` over plain http. That name resolves to Tailscale only if MagicDNS is answering. On a hotel or office LAN whose DNS search domain also answers `homebox`, nothing on the page proves I'm on the real homebox. The hash is served by the same page as the APK, so it only catches a corrupted download, not a spoofed host. The page also doesn't say how to compute a SHA-256 on Android. Finally, it says updates must use "the same key" but never shows that key's certificate fingerprint, which is the thing I could actually pin.
  - Expected: a link to the full MagicDNS name or 100.x address, or an HTTPS `*.ts.net` URL (as with `tailscale serve`), plus the signing-certificate SHA-256 next to the file hash. **Medium.**
- "You can turn Allow from this source off again afterwards; tailbond stays installed." This is a side sentence in a paragraph, not a numbered step, so most people will leave Chrome allowed to install APKs. Expected: a step ⑤ "Turn Allow from this source back off". **Low.**

## Task 2 — Add the phone to homebox (pair), from the phone

**Where I tap:** On the phone page I tap step 2, **"Open tailbond and ask homebox"**.
1. `02a`: "Asking homebox… Looking for homebox on your Tailscale network."
2. `02b`: "Waiting for approval on homebox… Ask whoever runs homebox to approve "pixel-8" on homebox's tailbond page (it's at the top of http://homebox:8077). There it shows as pixel-8 · sam@example.com · 100.101.8.8, so they can tell it's this phone… tailbond keeps waiting for 10 minutes…"
3. `02e`: "Add this phone to homebox?" I tap **Add phone**.

If I open the app from the launcher instead, I go from `01-onboarding` "Get started" to `02-pairing`, type the name, and tap "Ask to connect". The error states are clear:
- `02c`: "Can't find homebox — Check that the Tailscale app says Connected, and that the name matches…"
- `02d`: "homebox said no — Whoever runs homebox declined this phone."
- `02-pairing-paste-error`: "That isn't a complete tailbond link. Copy the whole link starting with tailbond://pair"

**What works (for me especially):** The approval request carries my real Tailscale identity: login `sam@example.com` and node IP `100.101.8.8`. That's the right trust root, the same identity Tailscale ACLs use. The confirm dialog also discloses the transport: "It connects over the internet to home.example.net:443, and your internet traffic goes out from home (you can change that in Settings)." So I learn that tailbond does *not* ride Tailscale and needs a public endpoint at home.

**Where I hesitate:**
- `02e`: "🔒 Secure connection to homebox" plus a collapsed "Technical details". The lock claim is vague: I can't see what is pinned or authenticated, and "Technical details" isn't shown expanded in the packet. This dialog also says ":443" without a protocol. Only the later error (`13`) reveals it's "UDP port 443". As the person who would set up the port forward, I want "UDP 443" here. **Low.**
- `02-pairing`: under "Or use the home server's page" the **"Open home server page"** button is greyed out, with no reason given. The text above it says "Go back to it in your browser and tap step 2". If I came from the launcher, I can't tell why the button is disabled. **Low.**
- Phone page step 2: "You run homebox yourself? Approve it at http://homebox:8077 … and that page works on this phone too." Nothing says *who* can approve on that page. If any tailnet member can open :8077 from their phone and approve themselves, the approval step protects nothing. The computer page (`web-00c`) says "Anyone on your Tailscale account can open it", which is about the install page, not the approval page. I can't tell which tailnet identities can approve. **Low** (can't verify from screens).

## Task 3 — Connect

**Where I tap:** I tap the big switch on the home screen.
- `04-handoff-other-vpn`: "Switch from Tailscale to tailbond? Android runs one VPN at a time. Turning on tailbond turns off Tailscale (or any other VPN)… You still reach your Tailscale machines, like your NAS, through homebox. Next, Android asks you to allow tailbond's connection request. Tap OK." I tap **Turn on tailbond**, then OK in Android's dialog (not shown).
- `11-home-connecting`: "Connecting… Reaching homebox…"
- `10-home-connected`: "Connected — Through homebox · Wi-Fi + mobile data".

**What works:** The handoff dialog matches what I expect from Android: one VPN slot. The "This device" row on `10` gets ACL identity right, which I'm very glad to see: "10.77.0.2 — Not a Tailscale address: to your Tailscale machines this phone's traffic comes from homebox, under its access rules." So my user-based ACLs no longer apply; homebox's do.

**Where I hesitate:**
- **Always-on VPN isn't mentioned where I need it.** Like many Tailscale power users, I have Always-on VPN (often with "Block connections without VPN") set for Tailscale. The handoff dialogs (`04`, `05`) say nothing about it. The warning lives only in Settings (`30-settings`): "Turn off Always-on for Tailscale first: only one app can have it… Leave "Block connections without VPN" off if you want to switch back to Tailscale easily." I'd hit this the first time I tap the switch, not after I've found Settings. Expected: the handoff dialog says "If Always-on VPN is set for Tailscale, turn it off first" and offers "Open VPN settings". **Medium.**
- **Full tunnel by default.** Tailscale on Android only uses an exit node if I pick one. tailbond defaults to "All internet traffic leaves from home, like using an exit node" (`30-settings`). It is disclosed: in grey small print on the phone page, in the `02e` dialog, and at the very bottom of the home screen ("Internet traffic — Through your home server"). But the handoff dialogs I read right before connecting (`04`/`05`) don't repeat it. **Low.**
- **Routes and DNS aren't shown.** The "This device" row says "They can reach the phone at this address through homebox". That only holds if homebox advertises 10.77.0.x as an approved subnet route, which isn't stated. It also means tailnet nodes allowed to reach homebox can now reach my phone. Settings' "Only Tailscale machines — only your tailnet goes through home" doesn't say whether "tailnet" includes routes from my *other* subnet routers (192.168.x), or whether the tailnet's DNS settings (split DNS, global nameserver) apply on the phone. **Low.**
- Header "homebox / via home server · tail1234.ts.net": "via home server" under the home server's own name is redundant. Tailscale shows the tailnet name or account here. **Low.**

## Task 4 — Find and copy the NAS's MagicDNS name or Tailscale IP

**Where I tap:** On the home screen I scroll to "Machines", type `nas` into "Search machines" (`18`), and tap the copy icon. The toast says "Copied 100.101.2.2". Or I tap the row and get `19-machine-detail`: "Tap an address to copy it", with MagicDNS name `nas.tail1234.ts.net`, Short name `nas`, Tailscale IPv4 `100.101.2.2`, and Tailscale IPv6 `fd7a:115c:a1e0::2`, each with its own copy icon. The toast reads "Copied nas.tail1234.ts.net". The footnote says "Reached through homebox. With tailbond on, apps on this phone can use these addresses just like with Tailscale."

**What works:** The detail screen is better than what I'm used to in Tailscale: all four forms, labelled, one tap each.

**Where I hesitate:**
- In the list (`18`), the row's first line is the MagicDNS name `nas.tail1234.ts.net`, but the single copy icon copies the **IP**. I only find out from the toast. **Low.**

## Task 5 — Are Wi-Fi and 5G both in use right now?

**Where I look:** `10-home-connected` shows "Connections — Using both Wi-Fi and mobile data", with Wi-Fi "● In use ↓38.4 ↑4.1 Mbps" and Mobile data "● In use ↓22.7 ↑2.9 Mbps". The status card says "Through homebox · Wi-Fi + mobile data". `20-connections` adds totals (61.1 / 7.0 Mbps), "delay 21 ms" / "delay 48 ms" per link, and "2.05 GB this session". `15-one-path-failed` shows how a failure looks: "Using Wi-Fi only", with Mobile data "● Not working right now".

No problems. This is clear, and the per-link latency is exactly what I'd want.

## Task 6 — Make it save mobile data

**Where I tap:** I flip the home-screen switch **"Save mobile data — Use mobile data only when Wi-Fi stops working"**. In `16-save-data-standby` the switch is on, the screen says "Using Wi-Fi · saving mobile data (backup only)", Mobile data shows "Not in use · saving data", and the mode is "Save mobile data". When Wi-Fi drops (`17`): "Wi-Fi isn't working · using mobile data", Mobile data "In use · taking over".

**Where I hesitate:**
- There are many overlapping data controls on `20`/`21`:
  - the home switch
  - the "How to use your connections" radio (Fastest / Smooth calls and video / Most reliable / Save mobile data)
  - per-link "Use for tailbond" toggles
  - "Advanced: choose which connection is main or backup"
  - "Faster switch to backup (experimental)"
  - "Mobile data limit", which is "Separate from Save mobile data"

  The page does explain how they relate ("The Save mobile data switch on the home screen picks "Save mobile data" here."). Still, I'd like two things answered on screen. First, the limit resets "until the 1st of next month", but my carrier's billing day isn't the 1st. Second, it doesn't say what happens to my traffic when the limit is hit and Wi-Fi is down: is it blocked, or does it leave outside the tunnel? **Low.**

## Task 7 — Hotel Wi-Fi with a sign-in page

**Where I tap:** `16c-home-wifi-needs-sign-in` shows the card "Wi-Fi needs sign-in — This Wi-Fi shows a sign-in page before it lets you online. tailbond keeps you connected over mobile data meanwhile, and adds Wi-Fi by itself once you're signed in." Below it: "Tap the Wi-Fi network (it says "Sign in to network") to open the page. Android's sign-in page uses the Wi-Fi directly, so you don't need to turn tailbond off." I tap **Sign in to Wi-Fi**, sign in on Android's page (not shown), and tailbond adds the Wi-Fi back by itself. Settings backs this up: "tailbond never bonds over a Wi-Fi until Android says it's online."

**What works:** The behaviour is right. I stay connected over mobile data, don't have to disable the VPN, and the Wi-Fi rejoins automatically.

**Where I hesitate:**
- The card gives two instructions for the same action: the small text says to tap the Wi-Fi network ("Sign in to network"), and the button says "Sign in to Wi-Fi". I'd wonder whether they differ. **Low.**

## Task 8 — Turn tailbond off and go back to the Tailscale app

**Where I tap:** I tap **"Switch back to Tailscale"**, the outlined button under the status card on every home state (`10`, `11`, `13`, `15`, `16c`). It's also in Settings: "Turns tailbond off and opens the Tailscale app. Turn Tailscale on there if it says Disconnected". Afterwards, `12b` says "Disconnected — Stopped. Tailscale is in charge on this phone." with a card "Tailscale is in charge on this phone… In the Tailscale app, turn on its switch if it says Disconnected." and an **Open Tailscale** button.

**What works:** It's easy to find, and it matches how the Tailscale app behaves after another VPN takes the slot (it shows Disconnected until I toggle it).

**Where I hesitate:**
- On `12b`, with Tailscale in charge, the rest of the screen still says "This phone reaches your tailnet through homebox" and "Will use Wi-Fi and mobile data together" (both "Ready"). Right now it doesn't reach the tailnet through homebox; Tailscale does that directly. **Low.**

---

## Findings

| # | Task | Finding | Severity |
|---|------|---------|----------|
| 1 | 1 Install | Trust in the APK rests on plain-http `homebox/` short-name resolution. The SHA-256 is served by the same page, there's no instruction for checking it on Android, and no signing-cert fingerprint is shown despite "Later updates must be signed with the same key". | medium |
| 2 | 1 Install | "You can turn Allow from this source off again afterwards" is a side sentence, not a step, so Chrome is likely left able to install APKs. | low |
| 3 | 2 Pair | "🔒 Secure connection to homebox" is vague (details hidden behind "Technical details"). The endpoint "home.example.net:443" doesn't say UDP; only error `13` does. | low |
| 4 | 2 Pair | "Open home server page" is greyed out on `02-pairing` with no reason given. | low |
| 5 | 2 Pair | "Approve it at http://homebox:8077 … that page works on this phone too": the screens don't say which tailnet identities can approve, so self-approval risk is unclear. | low |
| 6 | 3 Connect | Always-on VPN conflict ("Turn off Always-on for Tailscale first") appears only in Settings, not in the handoff dialogs `04`/`05` where I'd hit it. | medium |
| 7 | 3 Connect | Full tunnel ("like using an exit node") is the default, unlike Tailscale. It's disclosed in small print and on the home screen's last row, but not repeated in the handoff dialogs. | low |
| 8 | 3 Connect | Nothing says whether subnet routes from other routers or the tailnet's DNS settings apply, or that "They can reach the phone at this address" depends on an approved route via homebox. | low |
| 9 | 3 Connect | Header "via home server · tail1234.ts.net" under "homebox" is redundant. | low |
| 10 | 4 Copy | In the machine list, the row's first line is the MagicDNS name but the copy icon copies the IP ("Copied 100.101.2.2"). | low |
| 11 | 6 Save data | Six overlapping data controls. The limit resets on "the 1st of next month" (not my billing day), and the screen doesn't say what happens to traffic when the limit is hit and Wi-Fi is down. | low |
| 12 | 7 Hotel Wi-Fi | Two instructions for one action: "Tap the Wi-Fi network (it says "Sign in to network")" vs the "Sign in to Wi-Fi" button. | low |
| 13 | 8 Back to Tailscale | `12b` still says "This phone reaches your tailnet through homebox" / "Will use Wi-Fi and mobile data together" while "Tailscale is in charge". | low |

Totals: 0 high / 2 medium / 11 low
