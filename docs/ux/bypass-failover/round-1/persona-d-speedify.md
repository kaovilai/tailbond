# Round 1: Persona D, "Speedify user"

I've used Speedify for years: per-app and per-domain Bypass lists, and a server picker with an automatic "nearest server" option. At work I use Peplink/SpeedFusion. I expect two things. Setting up a bypass or a server should take a few clicks. And I should see which server I'm on, and since when, without scrolling.

Screens used: all 19 GL.iNet PNGs and all 23 Android PNGs from the manifests.

---

## Task 1: Make `mybank.com` skip the tunnel (router, then phone)

**Done?** Router: **partly**. With dnsmasq-full installed it takes about 4 clicks (Add a rule → A website → type → Add). Without it the rule just sits on "Waiting" and the panel gives me no way to fix that. Phone: **yes** on Android 13+, but the rule seems to cover less than the router's rule does.

Screens: `bypass-empty-desktop`, `bypass-add-website-presets-desktop`, `bypass-rules-desktop`, `bypass-rules-phone`, `bypass-website-rules-wait-desktop`, `bypass-add-streaming-preset-desktop`, `bypass-add-wifi-calling-desktop`; Android `44-settings-servers`, `46-websites-skip-tailbond` (+dark), `46c-websites-none-yet`, `46b-websites-needs-android-13`, `47-add-website-dialog`, `47b/c/d-add-website-preset-*`, `48-settings-skip-rows-android-12`.

### Findings

- **high**, `bypass-website-rules-wait-desktop.png`: The rule is saved and shown as "Waiting", but the notice says "dnsmasq can't fill nftables sets … Install dnsmasq-full". The panel has no button, no link, and no steps. My bank stays blocked, and I can't fix it from the panel. Speedify never asks me to install a DNS server.
  *Suggestion:* add an "Install now" button, or a link to GL's Applications → Plug-ins with the package pre-filled. Put the warning in plain words ("Website rules need one extra package. Install it, about 1 MB") and keep the technical line under "Details". Also say what happens meanwhile: "mybank.com still goes through tailbond until this is fixed."

- **medium**, `47-add-website-dialog.png`, `46-websites-skip-tailbond.png` vs `bypass-add-website-presets-desktop.png`: The router says a website rule "Includes all its pages and parts, like www.mybank.com and login.mybank.com". The phone says only "Its www. name is included", and adds "parts of a site loaded from other names still use tailbond". So on the phone, `login.mybank.com` or `secure.mybank.com` may still go through the tunnel and get blocked, with nothing to tell me why. Speedify's domain bypass covers subdomains.
  *Suggestion:* make the phone match subdomains like the router does. If it can't, say so clearly in the dialog ("login.mybank.com is NOT included: add it separately") and let me enter `*.mybank.com`.

- **medium**, `46-websites-skip-tailbond.png` / `46c-websites-none-yet.png`: "New addresses for a website reconnect tailbond for a moment", and addresses are looked up every 10 minutes. Big sites on CDNs change addresses all the time, so the whole tunnel (calls included) could drop every 10 minutes because I bypassed Netflix.
  *Suggestion:* say how often this happens in practice, or apply new addresses without reconnecting. At the very least, show a "last reconnect because of website list" time so I can link drops to this.

- **medium**, `bypass-add-wifi-calling-desktop.png`: The manifest says this is after clicking the "Wi-Fi calling" quick pick, but the screen is identical to the empty card. No dialog, no new rule, no toast. As a user I'd think the button did nothing, and click it again.
  *Suggestion:* open the pre-filled dialog (as Streaming does) or add the rule with a toast ("Wi-Fi calling added: UDP 4500 and 500 use the internet directly").

- **low**, `bypass-empty-desktop.png`, `bypass-rules-desktop.png`: "Bypass tailbond" is the 5th card, below Internet connections, Speed or reliability and Home server, about 1300 px down. On Android it's a row in the middle of Settings. In Speedify, Bypass is a top-level item.
  *Suggestion:* add an anchor or tab ("Rules") near the status card, or a line in the status card ("3 bypass rules active"), so it can be reached in one click.

- **low**, `bypass-website-rules-wait-desktop.png`: The Streaming preset added 5 domains on the router, but the rule row shows only "netflix.com · Streaming location". I can't see that disneyplus.com and max.com are also covered without opening the rule.
  *Suggestion:* show "netflix.com + 4 more" and expand on click.

- **low**, `47c-add-website-preset-streaming.png` vs `bypass-add-streaming-preset-desktop.png`: The phone's "Streaming" preset adds only netflix.com and nflxvideo.net. The router's adds five services. The same word does different things in the two apps.
  *Suggestion:* use the same list on both, or rename the phone's to "Netflix".

- **low**, `47b-add-website-preset-banking.png`: Tapping "Banking" adds nothing. It only shows "Type your bank's website below", but the field is *above* the chips.
  *Suggestion:* change it to "above", or move focus to the field and put the cursor there.

---

## Task 2: Make the work laptop always use tailbond

**Done?** **Yes** on the router: one quick pick or "A device" → work-laptop → "Always use tailbond". The help text says plainly that it overrides website and port rules and waits rather than leaking when tailbond is down. That's good. I do have doubts about whether the rule keeps matching the laptop.

Screens: `bypass-add-device-picker-desktop`, `bypass-add-device-picker-phone`, `bypass-rules-desktop`, `bypass-rules-phone`, `bypass-website-rules-wait-desktop`.

### Findings

- **medium**, `bypass-rules-desktop.png`, `bypass-add-device-picker-desktop.png`: The rule is tied to a MAC address (`3c:22:fb:10:aa:01`). The picker itself warns that laptops use private, per-network MACs. If the laptop's MAC changes (it forgets and rejoins the network, or an OS update), the rule silently stops matching. The laptop then follows the website rules and `mybank.com` leaves through the hotel, which breaks the "never" requirement. The row shows no sign of whether the rule is matching anything right now: Fire-TV-Stick shows "212 connections", work-laptop shows nothing.
  *Suggestion:* show "Online now · N connections" or "Not seen since …" on device rules. Warn when a device with the same hostname shows up under a different MAC ("work-laptop has a new address, update the rule?").

- **low**, `bypass-add-device-picker-desktop.png`: The help text says "Everything from this device goes through homebox". With a backup server it goes through vps during a failover. A work laptop going through a rented VPS may matter to my employer, and the text hides that.
  *Suggestion:* say "goes through tailbond (homebox, or your backup server when homebox is down)".

---

## Task 3: Add a VPS as a backup server, and understand a clash

**Done?** Router: **partly**. Discovery ("Found on your tailnet: vps" → Ask to add) is the fastest path and is close to Speedify-easy. But the pasted-link path gives contradictory results, and the clash fix assumes shell access. Phone: **partly**. The add page is a wall of four methods with copy reused from home-server setup. The clash dialog is clearer than the router's.

Screens: `servers-single-home-desktop`, `add-server-open-desktop`, `add-server-waiting-approval-desktop`, `add-server-subnet-clash-desktop`, `add-server-paste-clash-desktop`, `servers-home-active-vps-standby-desktop`; Android `44b-settings-one-server-add-backup`, `45-add-another-server`, `45b-add-another-server-confirm`, `45c-add-another-server-same-addresses`, `44-settings-servers` (+dark).

### Findings

- **high**, `add-server-paste-clash-desktop.png`: The screen shows a red banner "That didn't work: subnet_in_use" (a raw error code) and, at the same time, a green toast "The server added as a backup". The Home server card still shows only homebox. I can't tell whether vps was added. I might leave thinking I have a backup when I don't.
  *Suggestion:* never show the success toast when the add fails. Replace `subnet_in_use` with the same plain explanation and fix steps as `add-server-subnet-clash`. Fix the grammar of the toast too ("vps added as a backup").

- **medium**, `add-server-subnet-clash-desktop.png`: "On vps, change the setting `server.subnet`" doesn't say where that setting lives. The Android version (`45c`) at least names `/etc/tailbond/tailbond.json`. A Speedify user won't know there's a config file at all.
  *Suggestion:* give the file path and the exact commands, with a copy button. Better: since the router can already ask vps to approve, offer "Ask vps to pick a free range" or a one-click fix on vps's own page (http://vps:8077).

- **medium**, `45-add-another-server.png`: The backup-server page offers 4 methods. Two of them talk about the *home* server ("Open home server page", "Scan code from your home server", "http://homebox:8077"), which I'm not adding. It also lacks the router's "Found on your tailnet: vps" shortcut. I had to read the whole screen to work out which box applies.
  *Suggestion:* lead with discovered tailnet machines that run tailbond (a tap-to-add list, like the router). Rename the other options to "the backup server's page" / "code from the backup server", and fold them under "Other ways".

- **medium**, `45b-add-another-server-confirm.png`, `44-settings-servers.png`: The confirm dialog says "your internet traffic goes out from home", and Settings says "Through your home server: all internet traffic leaves from home". When I'm on vps, my traffic leaves from a datacenter IP, not home. That's exactly the kind of address banks and streaming services block, so more sites may break during a failover. Nothing here or on the router tells me that.
  *Suggestion:* reword to "goes out from the server you're using (homebox, or vps when homebox is down)". On the failover status, add "Websites now see vps's address (203.0.113.9)".

- **medium**, `servers-single-home-desktop.png`, `add-server-*.png`, `servers-home-active-vps-standby-desktop.png`, `failover-*.png`, `failback-countdown-desktop.png`: The "Bypass tailbond" card is gone on every server or failover screen, even though rules exist in other states. I can't confirm that my bank rule and work-laptop rule still apply after adding vps or during a failover.
  *Suggestion:* keep the Bypass card on the page in every state. During a failover, show a line like "Rules still apply; 'Always use tailbond' devices go through vps".

- **low**, `servers-home-active-vps-standby-desktop.png`, `44-settings-servers.png`: There are two controls for one decision. The up/down order arrows ("First choice" / "Backup") and the "When to switch" radio ("Prefer homebox, use vps…") both set which server is preferred. It's unclear what happens to the radio label if I move vps to the top.
  *Suggestion:* make the radio wording follow the list order ("Prefer the first server…"), or drop the arrows when there are only two servers.

- **low**, router vs Android, several screens: The same things have different names. "First choice" vs "Main", "Standby" vs "Ready as backup", "Backup" vs "Backup 1", the "Home server" card vs the "Servers" card, "Add a backup server" vs "Add another server", and "UDP port 443" vs "203.0.113.9:8443" (port missing on the router).
  *Suggestion:* use one set of words on both apps.

- **low**, `servers-home-active-vps-standby-desktop.png`, `44-settings-servers.png`: The server list shows no latency or speed per server. Speedify shows ping per server, which is how I decide whether the backup is worth it.
  *Suggestion:* show the last round-trip time next to "answered 11 s ago".

---

## Task 4: Notice the switch to vps, and that it came back

**Done?** "Am I on vps, and why?": **yes** on both. The router headline "Using vps (homebox is unreachable)" and the Android title "vps" plus the status line are clear at a glance, better than Speedify's small server label. "When did it switch, and when did it come back?": **partly**. I can see "Back to homebox in 1:20", but no time is shown for either switch, and there's no history.

Screens: `servers-home-active-vps-standby-desktop`, `failover-using-vps-home-unreachable-desktop`, `failover-using-vps-home-unreachable-phone`, `failback-countdown-desktop`, `failover-you-picked-vps-desktop`; Android `40` (+dark), `41`, `41b`, `42`, `43`, `43b`, `49`, `44-settings-servers` (+dark).

### Findings

- **medium**, `failover-using-vps-home-unreachable-desktop.png`, `43-home-back-on-homebox.png`, `failback-countdown-desktop.png`: Nothing records when it switched. The router's "Up 1 min" resets with the server, so the uptime is lost. Android's "Back on homebox" has no time, and the router has no "back on homebox" state at all. I can't answer "when did it come back?" an hour later. I also can't see from the screens whether the phone posts a notification when it switches. Peplink has an event log, and Speedify shows a notification.
  *Suggestion:* add "Switched to vps at 14:02 (homebox stopped answering)" and "Back on homebox since 14:31". Keep a short "Recent switches" list under Servers, and post an Android notification on each switch.

- **medium**, `failover-you-picked-vps-desktop.png`, `44-settings-servers.png`: "Using vps (you picked it)" doesn't say whether it will go back to homebox. "When to switch" still shows "Prefer homebox…", which suggests it will, but the status line has no countdown. There's no "Go back to automatic" button either. Both apps also offer "Use now" / "Use this server now" on a homebox marked "Can't reach" / "Unreachable", without warning that this would drop me.
  *Suggestion:* show "Stays on vps until you choose otherwise · [Back to automatic]". Disable "Use now" on unreachable servers, or confirm first ("homebox isn't answering; switch anyway?").

- **medium**, `41-home-using-vps-homebox-unreachable.png`, `42-home-back-to-homebox-in-1-20.png`: The Machines list shows homebox with a green "online" dot while the Servers section above says "Unreachable". Two contradictory signals on one screen make me doubt both.
  *Suggestion:* when tailbond can't reach a server, mark that machine to match (or explain: "online in Tailscale, but tailbond can't reach it").

- **low**, `43b-home-switching-to-vps.png`: The status says "Connecting… Reaching vps…" and "Using vps (homebox is unreachable)" at the same time, and the title already says "vps". It isn't using vps yet.
  *Suggestion:* show "Switching to vps (homebox stopped answering)" until it's connected.

- **low**, `49-home-error-no-server-reachable.png`: The title still reads "vps via backup server" while nothing is connected. The server rows lose their status and show just "Main server" and "Backup" with grey dots.
  *Suggestion:* use the title "Not connected". Keep "Unreachable · last answered 3 min ago" on each row.

- **low**, `44-settings-servers.png`: "Reached at home.example.net:443" next to "Unreachable" sounds like it was reached.
  *Suggestion:* change it to "Address: home.example.net:443".

- **low**, `failover-using-vps-home-unreachable-desktop.png`, `41-home-using-vps-homebox-unreachable.png`: The tunnel address changes silently (10.77.0.2 → 10.83.142.2). Anything I'd set up with the old address (e.g. a port opened to my phone) breaks with no notice.
  *Suggestion:* add a hint: "This address changes with the server in use."

---

## Overall count

| Severity | Count |
|---|---|
| high | 2 |
| medium | 11 |
| low | 12 |
