# Round 1: persona A, "GL traveller"

Persona: a consultant who travels every month with a GL.iNet Slate 7. Knows GL's
admin panel (Repeater, VPN Dashboard) but isn't a network engineer. Wants
hotels to just work, needs banking and video calls to work, and must not leak
work traffic.

Screens reviewed: every PNG in `out/ux/bypass-failover/glinet/` and
`out/ux/bypass-failover/android/`, using the two manifests.

---

## Task 1: make `mybank.com` skip the tunnel (router, then phone)

**Done?** Partly. On the router, the "Add a rule" dialog and the rules list are
clear enough to add the rule (bypass-empty, bypass-add-website-presets,
bypass-rules). In the "Waiting" state I can't finish (bypass-website-rules-wait).
On the phone I can add it (46c, 47, 47b, 46), but I'm not sure my bank's login
would actually skip the tunnel.

Findings:

1. **high**, `bypass-website-rules-wait-desktop.png`: the notice says "dnsmasq can't fill nftables sets … Install dnsmasq-full". I don't know what that means, and nothing on the screen tells me how to install it: no button, and no link to Applications → Plug-ins. The bank rule stays "Waiting", so the bank stays blocked, and I can't get past this step.
   *Suggestion:* add an "Install now" button, or at least a link to GL's Plug-ins page with the package name, and say what happens to the bank traffic until it's installed ("still goes through tailbond, so your bank may block it").
2. **medium**, `46-websites-skip-tailbond.png` (vs `bypass-add-website-presets-desktop.png`): the phone says "mybank.com · also www.mybank.com", and the footnote says parts loaded from other names still use tailbond. The router says it "includes all its pages and parts, like www.mybank.com and login.mybank.com". On the phone, `login.mybank.com` probably still goes through the tunnel, which is exactly the part the bank blocks.
   *Suggestion:* match the router (include all subdomains), or say clearly in the row "only mybank.com and www.mybank.com; add login.mybank.com separately".
3. **medium**, `47b-add-website-preset-banking.png`: the Banking pick only talks about the website. I bank in the bank's app, and the phone says nothing about that. The Streaming pick, by contrast, sends me to "Apps that skip tailbond". The text also says "Type your bank's website below", but the box is above it.
   *Suggestion:* add "Use your bank's app? Add it under Apps that skip tailbond" with a button, and change "below" to "above".
4. **medium**, `46-websites-skip-tailbond.png`: one dense footnote hides two things I care about. "Other sites sharing those addresses … skip it too" means other traffic, maybe work traffic, could leave through the hotel. "New addresses for a website reconnect tailbond for a moment" means a video call could drop when the bank's addresses change, which the footnote says are looked up every 10 minutes.
   *Suggestion:* split these into two plain, separate warnings: "Other sites on the same servers may also skip tailbond" and "Adding a site briefly reconnects tailbond (calls may stutter)". Say whether the 10-minute refresh also causes a reconnect.
5. **medium**, `46b-websites-needs-android-13.png`: on Android 12 it suggests putting "the browser you use for it" in Apps that skip tailbond. It doesn't warn that everything I open in that browser, work sites included, would then go out over hotel Wi-Fi.
   *Suggestion:* add "Everything in that browser will skip tailbond. Use a separate browser just for banking."
6. **low**, `bypass-website-rules-wait-desktop.png` / `bypass-add-website-presets-desktop.png`: the Add dialog doesn't warn that website rules can't work yet on this router. I only find out after saving.
   *Suggestion:* show the same notice inside the dialog when "A website" is chosen.
7. **low**, `bypass-add-website-presets-desktop.png`: "uses this router's internet, whichever connection GL picks" is unclear. Does "GL" mean the router or the company? I can't tell whether the bank goes over hotel Wi-Fi or my capped phone tethering (1.2 of 5 GB used).
   *Suggestion:* "whichever of this router's connections is working (Repeater or Tethering)", and point to the "only one connection" option if it exists. That option isn't visible in the dropdown.
8. **low**, `46-websites-skip-tailbond.png`: the phone gives no sign that a rule is doing anything. The router shows "37 connections". The disabled tax.example.gov row has no "Off" label, just a grey toggle.
   *Suggestion:* show a connection count or "last used" time like the router does.
9. **low**, `bypass-add-wifi-calling-desktop.png`: the manifest says it shows the dialog after the Wi-Fi calling quick pick, but the PNG only shows the empty card with no dialog. I couldn't review it.
   *Suggestion:* re-render the screenshot.
10. **low**, `47d-add-website-preset-wifi-calling.png` / `47c-add-website-preset-streaming.png` vs router: on the phone, Wi-Fi calling adds nothing, while the router adds a port rule. Streaming adds 2 sites on the phone and 5 on the router. The same quick picks behave differently on each device.
    *Suggestion:* use the same preset contents, or say why they differ ("the phone's own Wi-Fi calling already skips tailbond").

## Task 2: make the work laptop always use tailbond

**Done?** Yes, using bypass-add-device-picker (desktop and phone) and
bypass-rules. The dialog text "Everything from this device goes through homebox,
even websites or ports set to skip tailbond. If tailbond is down, it waits rather
than going around it" answers my question. But I only trust it while the rule
actually matches, and the screens don't let me check that.

Findings:

11. **medium**, `bypass-add-device-picker-desktop.png`, `bypass-rules-desktop.png`: the rule is tied to a MAC address. The dialog itself warns that laptops use a private MAC per network. My Mac can also rotate it, or use a different one over Ethernet. If the MAC changes, the rule quietly stops matching, and the laptop's bank and Netflix traffic leaves through the hotel. The work-laptop row is the only row with no connection count or "last seen", so I can't tell whether the rule is matching anything.
    *Suggestion:* show "Matched N connections / last seen …" on device rules, and show "Not seen for X days, its address may have changed" when it goes quiet. Tell macOS users to set Private Wi-Fi Address to "Fixed" for this network.
12. **medium**, `bypass-rules-desktop.png` / `bypass-add-device-picker-desktop.png` / `failover-using-vps-home-unreachable-desktop.png`: the important promise, "waits rather than going around it", appears only in the Add dialog. The row just says "Always uses tailbond". The dialog also says "goes through homebox", but when tailbond fails over, the work traffic goes through a rented VPS. Nothing says what happens if I turn the main tailbond switch off, for example to deal with a hotel sign-in page.
    *Suggestion:* put "Blocked when tailbond is down or off" under the row, say "goes through your tailbond server (homebox or vps)", and state clearly what the main switch does to this rule.
13. **low**, `bypass-add-device-picker-desktop.png` / Tailnet machines card: there are two "work-laptop" entries. The router's device list shows it online at 192.168.8.23, but Tailnet machines shows "work-laptop · Last seen 1 day ago". It isn't obvious that they're the same machine seen two different ways.
    *Suggestion:* label the Tailnet list "as Tailscale sees it", or link the two entries.

## Task 3: add the VPS as a backup (router and phone), understand a clash

**Done?** Partly. The router flow is clear up to approval (servers-single-home,
add-server-open, add-server-waiting-approval), and the two-server card makes
sense (servers-home-active-vps-standby). The clash and the pasted-link path
mislead me, and the phone's add page points me at the wrong server.

Findings:

14. **high**, `add-server-paste-clash-desktop.png`: a red banner says "That didn't work: subnet_in_use", and at the same time a green toast says "The server added as a backup". No vps appears under Home server. I'd believe the toast, leave, and have no backup when home goes down. Also, "subnet_in_use" is a raw error code.
    *Suggestion:* never show the success toast on failure. Replace the banner with the same plain clash explanation and fix steps as add-server-subnet-clash.
15. **medium**, `add-server-subnet-clash-desktop.png`: "Fix: on vps, change the setting `server.subnet` to another range … restart tailbond there". It doesn't say where that setting lives. The phone version (45c) at least names `/etc/tailbond/tailbond.json` and adds "or ask whoever runs vps". "Tunnel addresses" means nothing to me. It also doesn't say whether changing vps breaks anything that already uses it.
    *Suggestion:* give the file path and the exact command, or better, a "Fix it on vps" button that asks vps to pick a free range. Use the same wording on router and phone.
16. **medium**, `add-server-waiting-approval-desktop.png`: it tells me to open `http://vps:8077`, but the same page says tailnet names must be the full name, like nas.tail1234.ts.net. The short link may not open from my laptop behind the router. It also doesn't tell me that I'm the "someone" who has to approve.
    *Suggestion:* link `http://vps.tail1234.ts.net:8077`, and say "Open this on any of your devices and click Approve".
17. **medium**, `45-add-another-server.png`: on the "Add a backup server" page, the second and third cards still talk about the home server: "Open home server page", "open the home server's dashboard (http://homebox:8077)", "Scan code from your home server". Following them, I'd scan homebox's code again instead of adding vps.
    *Suggestion:* replace "home server" with "the backup server", and use `http://vps:8077` / "Scan code from the backup server" here.
18. **medium**, `servers-home-active-vps-standby-desktop.png` (and every two-server screen): the "Bypass tailbond" card isn't on the page once there are two servers. My bank, work-laptop and Fire TV rules seem to have vanished, and I can't check them.
    *Suggestion:* keep the Bypass card on the page in every server state. If it's only missing from the fixture, fix the fixture so reviewers can see it.
19. **medium**, `44-settings-servers.png`, `45b-add-another-server-confirm.png`: "Through your home server: all internet traffic leaves from home" and "your internet traffic goes out from home" are wrong while I'm on vps. Traffic then leaves from a data-center address, which banks and streaming block even more. That matters for Task 1.
    *Suggestion:* say "leaves from your server (homebox, or vps when it takes over)", and add a note that websites may treat the VPS address as a VPN.
20. **low**, `add-server-open-desktop.png` / `servers-home-active-vps-standby-desktop.png` / `44-settings-servers.png`: the same thing has several names: "Add a backup server", "Add another server", "Backup" vs "Backup 1", "Use now" vs "Use this server now".
    *Suggestion:* pick one term for each and use it on both router and phone.
21. **low**, `servers-home-active-vps-standby-desktop.png`: order is set two ways, with ↑/↓ arrows and with "When to switch" options that name the servers. It's unclear what the arrows do when "Always use homebox" is chosen. "Forget all servers" (red) sits right next to "Add another server".
    *Suggestion:* explain the arrows ("first = preferred"), and move "Forget all servers" away from Add or put it behind a confirmation.

## Task 4: notice the switch, and that it came back

**Done?** Partly. "Using vps (homebox is unreachable)" and "Back to homebox in
1:20" are clear on both router and phone
(failover-using-vps-home-unreachable, failback-countdown, 41, 41b, 42, 43). But
I can't tell when the switch happened or when it came back, and some status
signals contradict each other.

Findings:

22. **medium**, `failover-using-vps-home-unreachable-desktop.png`, `failback-countdown-desktop.png`, `43-home-back-on-homebox.png`: there's no event history or timestamps. "When did it switch?" is only guessable from "Up 1 min". The router has no "back on homebox" state at all, so after it switches back, the panel looks the same as if nothing had happened.
    *Suggestion:* add a short history ("14:02 switched to vps, homebox stopped answering; 16:40 back on homebox"), and show "Back on homebox since 16:40" for a while afterwards.
23. **medium**, `failback-countdown-desktop.png`, `42-home-back-to-homebox-in-1-20.png`: it will switch back by itself in 1:20 even if I'm on a video call, and it doesn't say whether calls drop during the switch. The only way to avoid that is "Use now" on vps, or changing "When to switch" permanently.
    *Suggestion:* add "Stay on vps for now" next to the countdown, and say what happens to calls ("calls may freeze for a few seconds").
24. **medium**, `failover-you-picked-vps-desktop.png`: "Using vps (you picked it)", yet "When to switch" still shows "Prefer homebox…" selected. I can't tell whether it will go back to homebox by itself, when that would happen, or how to undo my pick. "You" could also be someone else in the household.
    *Suggestion:* add "Stays on vps until you pick homebox" (or "until …"), plus a "Go back to automatic" button.
25. **medium**, `41-home-using-vps-homebox-unreachable.png`, `42-home-back-to-homebox-in-1-20.png`: the Machines list shows homebox with a green "online" dot while Servers says homebox is "Unreachable". My home has no power, so the green dot is wrong and makes me doubt the rest.
    *Suggestion:* match the Machines dot to the server state, or grey it out with "seen via Tailscale, not via tailbond".
26. **medium**, `49-home-error-no-server-reachable.png`, `43b-home-switching-to-vps.png`: when no server works, the phone shows "Not connected / Stopped". It doesn't say whether my traffic now goes out unprotected over hotel Wi-Fi, or whether it does during the few seconds of switching. The "Keep tailbond on" section even advises leaving "Block connections without VPN" off.
    *Suggestion:* say plainly "Your apps are using the hotel Wi-Fi directly right now" (or "are blocked"), and explain the trade-off of "Block connections without VPN" for people who must not leak work traffic.
27. **low**, `41-home-using-vps-homebox-unreachable.png`: the router says "Goes back to homebox by itself once it answers again", but the phone doesn't say this anywhere.
    *Suggestion:* add the same line on the phone.
28. **low**, `43b-home-switching-to-vps.png`: it says "Connecting… Reaching vps…" and also "Using vps (homebox is unreachable)", but it isn't using vps yet.
    *Suggestion:* "Switching to vps (homebox is unreachable)".
29. **low**, `failover-using-vps-home-unreachable-desktop.png`, `41-home-using-vps-homebox-unreachable.png`: the router's and phone's tunnel addresses change (10.77.0.2 → 10.83.142.2) with no explanation, and the page subtitle still says "one tunnel to your home server".
    *Suggestion:* say "address changes while on vps", and make the subtitle "to your tailbond server".
30. **low**, `49-home-error-no-server-reachable.png`: the header still says "vps · via backup server" while nothing is connected, and the server rows show grey "Main server" / "Backup" instead of each server's own status.
    *Suggestion:* show "Can't reach" on each server, and drop the server name from the header when not connected.

---

## Overall count

**high 2 / medium 16 / low 12**

- high: #1, #14
- medium: #2–#5, #11, #12, #15–#19, #22–#26
- low: #6–#10, #13, #20, #21, #27–#30
