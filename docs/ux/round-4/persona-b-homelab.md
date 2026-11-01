# Round 4 — Persona B (homelab power user)

Judged only from screenshots in `out/ux/after-4/` plus the task script and manifest.

## Task 1 — Install tailbond and pair it with the home server

**Tap path:** browser, Tailscale still on, open homebox's page (web-01). Tap "Download tailbond for Android", sideload it, come back. Tap "+ Add this phone" with name "pixel-8". web-02: "Opening tailbond…", then the app opens on 03-deeplink-confirm "Add this phone to homebox?". Tap "Add phone". (Fallback if I open the app first: 01 "Get started", then 02 "Home server's Tailscale name", then "Open home server page".)

**Stuck points:**
- 03-deeplink-confirm: "your internet traffic goes out from home (you can change that in Settings)". So the default is full tunnel, like having an exit node on. The Tailscale app's default is no exit node. It is disclosed, so I'd notice, but it's an opinionated default and I can't pick it here. **Low.**
- 03-deeplink-confirm: "This phone will be called "pixel-8" on homebox". It doesn't say whether pixel-8 becomes a tailnet node (it doesn't, see Task 2) or how it shows up to ACLs. **Covered under issue 1.**

Otherwise the flow is clear: the web page and app use the same wording ("Add this phone", "Add phone"), and the dialog shows the public endpoint `home.example.net:443`.

## Task 2 — Connect

**Tap path:** home, then toggle. 04-handoff-other-vpn "Switch from Tailscale to tailbond?" then "Turn on tailbond", then Android's OK. 11 "Connecting… Reaching homebox…", then 10 "Connected · Through homebox · Wi-Fi + mobile data".

**Stuck points:**
- 10-home-connected "This device": "pixel-8 (this phone) 10.77.0.2 — Your Tailscale machines can reach this phone at this address, through homebox". In the Tailscale app, This device shows a 100.x address and a MagicDNS name. Here the phone isn't a tailnet node: it's a 10.77.x address behind homebox. Nothing tells me whether:
  - homebox SNATs my traffic (so the NAS sees homebox, and ACLs written for my user or devices won't match), or
  - 10.77.0.0/x is an advertised subnet route I have to approve in the admin console and allow in ACLs.

  I'd have to guess why `tag:`- or user-scoped ACL rules suddenly behave differently. **Medium.**
- 14-error-cert: "Try again later, and tell whoever set it up. If homebox was just reinstalled, they can add this phone again (Settings → Pair again)." "Settings → Pair again" is on this phone (30-settings), but the sentence says "they" do it. It's unclear whether I tap it or the admin does. **Low.**

13-error-unreachable ("forward UDP port 443") is clear and actionable for me.

## Task 3 — Find and copy the NAS's MagicDNS name or Tailscale IP

**Tap path:** home, scroll to "Machines", type "nas" in "Search machines" (18), tap the copy icon. Toast "Copied 100.101.2.2". Or tap the row (19) and tap "nas.tail1234.ts.net" for "Copied nas.tail1234.ts.net".

**Stuck points:**
- 18-machines-search-copied: the row shows both `nas.tail1234.ts.net` and `100.101.2.2`, but there is one copy icon. Only the toast tells me it copied the IP, not the MagicDNS name. If I wanted the name I'd have to paste it to find out, or know to open the detail page. **Low.**

The detail page (19) is good: it matches Tailscale's tap-to-copy list (MagicDNS, short name, IPv4, IPv6) with an online dot.

## Task 4 — Are Wi-Fi and 5G both being used right now?

**Tap path:** home (10). "Using both Wi-Fi and mobile data", both tiles "In use" with ↓/↑ Mbps. More detail in 20-connections (per-link throughput, "delay 21 ms" / "48 ms", 60 s graph).

**Stuck points:**
- 15-one-path-failed: "Mobile data — Not working right now" (red) with no reason: no signal, server unreachable over that path, or blocked UDP? There's also no tap target to find out. **Low.**

Answering the task itself is immediate.

## Task 5 — Make it save cellular data

**Tap path:** home, turn on the "Save mobile data" toggle (16). Result: "Mode: Save mobile data", mobile tile "Not in use · saving data", Wi-Fi drop then shows "In use · taking over" (17). 21-connections-save-data confirms the radio follows the switch.

**Stuck points:**
- 20/21-connections "Mobile data limit": "At the limit, tailbond stops using mobile data until the 1st of next month." It doesn't say what happens when I'm on cellular only and hit the limit. Does tailbond disconnect (no NAS access at all)? Or does traffic leave outside the tunnel over the phone's own connection? I need to know which link carries what. **Low.**

The links between the home switch, the mode radio, and the limit are spelled out ("The Save mobile data switch on the home screen picks…", "Separate from Save mobile data"). No real confusion.

## Task 6 — Turn tailbond off and go back to the Tailscale app

**Tap path:** home, "Switch back to Tailscale". 12b "Tailscale is in charge on this phone… In the Tailscale app, turn on its switch if it says Disconnected", then "Open Tailscale".

**Stuck points:**
- 30-settings "Switch back to Tailscale" says it "Turns tailbond off and opens the Tailscale app". But 12b (after the same action) shows me still in tailbond with an "Open Tailscale" button. Either it opens Tailscale or it doesn't. The two descriptions disagree. **Low.**
- 12b-home-switched-back-to-tailscale: below "Tailscale is in charge", the screen still shows "Will use Wi-Fi and mobile data together", both links "Ready", and "Mode: Fastest". tailbond is inactive, so this reads like it's still about to bond. **Low.**

## Summary

| # | issue | severity | screen |
|---|---|---|---|
| 1 | This device shows "10.77.0.2 … reach this phone … through homebox" instead of a 100.x / MagicDNS identity. It doesn't explain whether traffic is SNATed as homebox or needs an approved subnet route, so ACL behaviour for the phone is unknown. | medium | 10-home-connected |
| 2 | Default is full tunnel ("your internet traffic goes out from home"), unlike Tailscale's no-exit-node default; can only be changed later in Settings | low | 03-deeplink-confirm |
| 3 | One copy icon per machine row; only the toast reveals it copied the IP, not the MagicDNS name | low | 18-machines-search-copied |
| 4 | Cert error: "they can add this phone again (Settings → Pair again)" is unclear on who acts (phone user or admin) | low | 14-error-cert |
| 5 | Failed link shows "Not working right now" with no reason and no way to get details | low | 15-one-path-failed |
| 6 | Mobile data limit: doesn't say whether hitting it on cellular only drops the tunnel (and NAS access) or sends traffic outside it | low | 20-connections / 21-connections-save-data |
| 7 | Settings says "Switch back to Tailscale… opens the Tailscale app", but the after-state stays in tailbond with an "Open Tailscale" button | low | 30-settings / 12b-home-switched-back-to-tailscale |
| 8 | After switching back, "Will use Wi-Fi and mobile data together" / "Ready" / "Mode: Fastest" still show while tailbond is inactive | low | 12b-home-switched-back-to-tailscale |

Counts: high 0, medium 1, low 7.
