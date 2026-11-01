# Round 1: Persona C, "family member"

Persona: a relative of the person who set tailbond up. Uses the phone app, and
the travel router's admin page when someone asks them to. Doesn't know what a
VPN server, subnet or port is. Worries about breaking things. Wants to know
two things: "is it working?" and "what should I do?"

Screens used: every PNG in `out/ux/bypass-failover/glinet/` and
`out/ux/bypass-failover/android/`, read alongside the two manifests.

---

## Task 1: Make the bank's website skip the tunnel

**Done?** Partly.
- Router: yes in the normal case. I used `bypass-empty`, then
  `bypass-add-website-presets`, then `bypass-rules`. I typed mybank.com, left
  "Use the internet directly" selected, and the rule showed up green.
- Router: no in the `bypass-website-rules-wait` case. The rule exists but it
  is "Waiting", and the fix is something I can't do.
- Phone: yes on Android 13 (`46c`, `47`, `47b`, `46`). On Android 12 (`46b`)
  it sends me to "Apps that skip tailbond", which I can follow.

**Findings**

- **high**: `bypass-website-rules-wait-desktop.png`. My bank rule says
  "Waiting", and the explanation is "dnsmasq can't fill nftables sets …
  Install dnsmasq-full". I have no idea what that means or where to install
  it, so my bank stays blocked. The status card at the top still says
  "Connected to homebox" as if everything is fine.
  *Suggestion:* lead with plain words, for example "Website rules can't work
  on this router yet. Your bank will still go through tailbond. Ask the person
  who set up tailbond to install the missing piece." Put the technical text
  behind a "Details" link, and add a one-click "Install" if the router can do
  it itself. Mention the problem in the top status card too.
- **medium**: `bypass-add-wifi-calling-desktop.png`. The manifest says this is
  the screen after clicking the "Wi-Fi calling" quick pick. On screen nothing
  changed: no dialog, no new rule, no message. I'd click it again or assume
  it's broken. (This could be a render bug, but it's what I'd see.)
  *Suggestion:* after a quick pick, always show either the filled-in dialog or
  the new rule with a "Wi-Fi calling added" confirmation.
- **medium**: `47b-add-website-preset-banking.png`. The hint says "Type your
  bank's website **below**", but the Website box is **above** the hint, so I
  scroll down looking for a second box. It also doesn't mention that I
  usually bank in the bank's **app**, which this page doesn't cover. The
  Android 12 screen (`46b`) does say so.
  *Suggestion:* change "below" to "above", or move the hint up. Add: "Use the
  bank's app? Add it under Apps that skip tailbond instead," with a link.
- **medium**: `46-websites-skip-tailbond.png` against
  `bypass-add-website-presets-desktop.png`. On the phone the rule says
  "mybank.com · also www.mybank.com". The router says it "includes all its
  pages and parts, like www.mybank.com and login.mybank.com". If my bank's
  login is on login.mybank.com, the phone might still send it through the
  tunnel, and nothing tells me.
  *Suggestion:* use the same behaviour and wording on both. If the phone
  really only covers www, say so in plain words next to the rule, for
  example "If the bank still blocks you, add its login address too."
- **low**: `bypass-add-website-presets-desktop.png`. The empty Website box
  shows "e.g. mybank.com", which looks like it's already filled in with my
  bank. The Add button is blue and clickable while the box is empty.
  *Suggestion:* use a clearly different example (`e.g. examplebank.com`), and
  grey out Add until something is typed, as the phone does.
- **low**: `bypass-add-website-presets-desktop.png`. "Uses this router's
  internet, whichever connection **GL** picks". I don't know who or what GL
  is. *Suggestion:* "whichever of the router's connections works best."
- **low**: `bypass-rules-desktop.png`. "Bank · 37 connections" means nothing
  to me. Is 37 good or bad? *Suggestion:* a plain status such as "Working:
  used in the last minute", with the count shown in a tooltip.
- **low**: `46-websites-skip-tailbond.png`. The phone list doesn't show
  whether a rule is working (the router shows a count, or "Waiting"). The
  paragraph at the bottom ("private DNS", "big hosting networks") is
  unreadable for me. *Suggestion:* add a small "working / not yet" status per
  rule, and put the long paragraph behind "Why might a site still not work?"
- **low**: `47d-add-website-preset-wifi-calling.png` against
  `bypass-add-wifi-calling-desktop.png`. The router adds a Wi-Fi calling rule,
  but the phone says "Nothing to add". The same button gives opposite answers,
  which makes me doubt both. *Suggestion:* add one line on the router
  explaining why it's needed there and not on the phone (the router handles
  traffic for other people's phones too).
- **low**: `47c-add-website-preset-streaming.png` against
  `bypass-add-streaming-preset-desktop.png`. The phone's Streaming pick adds 2
  Netflix sites. The router's adds 5 services. *Suggestion:* use the same list,
  or label the phone's pick "Netflix".

## Task 2: Make the work laptop always use tailbond

**Done?** Partly. The rule exists in `bypass-rules`, and I could create one
with `bypass-add-device-picker` (desktop and phone). I can't tell that it is
actually protecting the laptop.

**Findings**

- **medium**: `bypass-add-device-picker-desktop.png` and
  `bypass-rules-desktop.png`. The rule is tied to a string of letters and
  numbers (3c:22:fb:…). The dialog warns that "Phones and laptops may use a
  different (private) MAC address for each Wi-Fi network". So the rule could
  quietly stop matching, and then the laptop's traffic to bypassed sites would
  leave through the hotel, which is what the task says must never happen. The
  rule row gives no sign that it is matching the laptop right now.
  *Suggestion:* show a live status on the rule ("work-laptop is connected and
  covered" or "work-laptop not seen since …"). Warn on the rule if the device
  hasn't been seen with that address recently. Replace "MAC address" with
  "the device's network ID" and keep the technical term in a tooltip.
- **low**: `bypass-add-device-picker-desktop.png`. "If tailbond is down, it
  waits rather than going around it." I read "waits" as "maybe slow".
  *Suggestion:* say it directly: "If tailbond is down, this device has no
  internet until tailbond is back. That's on purpose, to keep work traffic
  private."
- **low**: `bypass-add-device-picker-phone.png`. Every device shows a MAC and
  an IP. "Unnamed device" gives me no clue which one it is.
  *Suggestion:* hide the MAC and IP behind a "details" tap. For unnamed
  devices, show the maker if known and "connected since …".
- **low**: `bypass-rules-desktop.png`. In the rules list and the device picker
  work-laptop is online, but in "Tailnet machines" it is grey, "Last seen 1
  day ago". It looks like two different laptops and makes me worry one is
  broken. *Suggestion:* explain the difference in one line, or mark the
  Tailnet entry "(its own Tailscale is off; it uses the router instead)".

## Task 3: Add the VPS as a backup server

**Done?** Partly.
- Router: `servers-single-home`, then `add-server-open` ("Found on your
  tailnet: vps", "Ask to add"), then `add-server-waiting-approval`, then
  `servers-home-active-vps-standby` worked.
- Phone: `44b`, then `45`, then `45b`, then "Add backup" worked.
- When there is a clash (`add-server-subnet-clash`, `add-server-paste-clash`,
  `45c`), I am stuck and not sure whether anything broke.

**Findings**

- **high**: `add-server-paste-clash-desktop.png`. A red banner says "That
  didn't work: subnet_in_use", and a green toast on the same screen says "The
  server added as a backup". I can't tell whether it worked. The Home server
  card shows only homebox, so I guess it didn't, but the green message says it
  did. "subnet_in_use" is a raw code.
  *Suggestion:* show only one outcome. On a clash, drop the success toast and
  show the same plain explanation as the name-based flow. Never show internal
  codes. Also fix the grammar ("vps was added as a backup").
- **high**: `add-server-subnet-clash-desktop.png`. The fix is "on vps, change
  the setting `server.subnet` to … `10.78.0.0/24`, restart tailbond there".
  I can't do any of that, and unlike the phone version, the router never says
  "ask whoever runs vps". It also doesn't say whether homebox is still fine.
  The big red box makes me think I broke something.
  *Suggestion:* lead with "vps wasn't added. Nothing changed: you're still
  using homebox." Then: "vps needs one setting changed before it can be a
  backup. Send this to the person who runs vps:" plus a "Copy instructions"
  button holding the technical text. Use amber instead of red, because
  nothing is broken.
- **medium**: `45c-add-another-server-same-addresses.png`. This is better than
  the router because it says "or ask whoever runs vps", but it is still full of
  "10.77.0.0/24", "/etc/tailbond/tailbond.json", and "Asking over Tailscale
  does this by itself while vps has no other devices". I don't know if I
  should try again some other way. *Suggestion:* the same structure as above:
  "Nothing changed, you're still on homebox", then a one-line next step, a
  "Share instructions" button, and the details folded away.
- **medium**: `45-add-another-server.png`. The screen is about adding a
  **backup** server, but two of the four boxes talk about the **home**
  server's page and dashboard ("Open home server page", "Scan code from your
  home server"). I can't tell whether I should be on homebox's page or vps's.
  With four different methods, I don't know which one is meant for me.
  *Suggestion:* say "the vps's page" in this flow. Show the one recommended
  way first ("Type vps and tap Ask to connect"), and fold the rest under
  "Other ways".
- **medium**: `servers-home-active-vps-standby-desktop.png`. A red "Forget
  all servers" button sits right next to "Add another server", and a red
  "Remove" sits on every row. For someone afraid of breaking things, these are
  scary to have one click away from what I came to do.
  *Suggestion:* move "Forget all servers" into a separate "Danger zone" or
  behind a ⋯ menu, and confirm it with the consequences spelled out ("Every
  device on this router loses tailbond until someone pairs it again").
- **medium**: `servers-home-active-vps-standby-desktop.png`, and also
  `failover-*` and `failback-countdown`. Once a second server is added, the
  "Bypass tailbond" card isn't on the page at all. It was there on
  `bypass-empty`, and my bank rule was there on `bypass-rules`. It looks as if
  adding vps deleted my rules. (This may be a fixture gap, but it's what the
  screens show.)
  *Suggestion:* keep the Bypass card on every page state, and say "These rules
  keep working whichever server is in use."
- **low**: `add-server-open-desktop.png`. It says "Found on your tailnet:
  vps", but vps isn't in the "Tailnet machines" list further down the same
  page. *Suggestion:* include vps in that list, or explain why it's missing.
- **low**: `add-server-waiting-approval-desktop.png`. I'm told to approve
  "slate7" on vps, but the top bar of the page says "GL-BE3600". I'd look for
  the wrong name. *Suggestion:* say "this router (shown as slate7)" and put
  the same name in the header badge, or explain both names.
- **low**: `servers-single-home-desktop.png` against `add-server-open-desktop.png`
  and `44-settings-servers.png`. The button says "Add a backup server", but
  the section that opens is "Add another server". The label is "Backup" in one
  place and "Backup 1" in another. *Suggestion:* use one name everywhere.
- **low**: `44b-settings-one-server-add-backup.png` and `48`. The red text
  "Remove from this phone" sits right next to "Pair again". With two servers
  it's hidden in ⋮, which is safer. *Suggestion:* put it in the ⋮ menu for
  one server too.

## Task 4: Notice the switch, and that it came back

**Done?** Partly.
- I can tell I'm on vps and that homebox "is unreachable" from
  `failover-using-vps-home-unreachable` (desktop and phone) and from Android
  `41`/`41b`.
- I can see it's going back from `failback-countdown` ("Back to homebox in
  1:20") and Android `42`.
- I **can't** tell **when** it switched over or when it came back: `43` just
  says "Back on homebox", and there's no router screen for after the switch
  back.
- "Why" is only "unreachable". That's acceptable, but nothing tells me whether
  I need to do anything.

**Findings**

- **medium**: `41-home-using-vps-homebox-unreachable.png` and `41b`. The
  router says "Goes back to homebox by itself once it answers again", but the
  phone doesn't. The only button under the warning is "Switch back to
  Tailscale". I'd press it thinking it fixes things, but Settings says it
  "Turns tailbond off".
  *Suggestion:* add the reassurance on the phone: "You're fine: still
  connected through vps. It goes back to homebox by itself. Nothing to do."
  Move or rename "Switch back to Tailscale" ("Turn off tailbond, use
  Tailscale") so it doesn't read as the fix for this message.
- **medium**: `41-home-using-vps-homebox-unreachable.png`. In "Servers"
  homebox has a red dot and says Unreachable. In "Machines" further down,
  homebox has a **green** dot. Is it on or off? *Suggestion:* make the
  Machines dot agree, or label it "Tailscale says online, tailbond can't
  reach it".
- **medium**: `43-home-back-on-homebox.png` and
  `failover-using-vps-home-unreachable-desktop.png`. Neither app shows when the
  switch happened or when it came back. "Up 1 min" and "last answered 2 min
  ago" make me do arithmetic. After it comes back the note just says "Back on
  homebox", and I don't know if that was 10 seconds or 3 hours ago.
  *Suggestion:* add times in plain words ("Switched to vps at 14:02 because
  homebox stopped answering", "Back on homebox since 16:40"), and a short
  "Recent switches" list on both the router and the phone.
- **medium**: `failover-using-vps-home-unreachable-desktop.png` and `-phone`,
  plus `44-settings-servers.png`. Next to the red "Can't reach" homebox there
  are a "Use now" / "Use this server now" button and a "Remove" link. If
  homebox looks broken I might press "Use now" to fix it, or "Remove" to get
  rid of the broken one. *Suggestion:* disable "Use now" while a server can't
  be reached, or say what will happen. Confirm "Remove" with "homebox will be
  removed from this router; you'll need to pair again to use it".
- **medium**: `failover-you-picked-vps-desktop.png`. "Using vps (you picked
  it)": I didn't, someone else did, and the page doesn't say whether it will
  ever go back to homebox or how I undo it. "When to switch" still shows
  "Prefer homebox", which seems to contradict this.
  *Suggestion:* "Using vps because someone chose it. It stays on vps until you
  pick homebox again," with a clear "Go back to homebox" button.
- **medium**: `49-home-error-no-server-reachable.png`. "(Connection timeout)"
  and "Check … that at least one of them is turned on" ask me to switch on a
  server I can't reach. It doesn't say whether my phone's internet works right
  now. *Suggestion:* "Your phone is using its normal internet now, without
  tailbond." Then "Try again", and "If this keeps happening, tell [the person
  who set it up]: neither homebox nor vps answers." Drop the technical words in
  brackets.
- **low**: `43b-home-switching-to-vps.png`. "Connecting… Reaching vps…" and
  "Using vps (homebox is unreachable)" show at the same time. Is it using vps
  or not yet? *Suggestion:* "Switching to vps… (homebox isn't answering)".
- **low**: `failover-using-vps-home-unreachable-desktop.png`. The page subtitle
  still says "Bond this router's internet connections into one tunnel to your
  **home** server" while on vps, and the top icon is the same green in every
  state. *Suggestion:* drop "home" from the subtitle, and change the top
  icon's colour or badge when on a backup.
- **low**: `44-settings-servers-dark.png`. In dark mode the "Unreachable" dot
  is pale pink and doesn't look like a problem. *Suggestion:* use a stronger
  red in dark mode.
- **low**: `40-home-on-homebox-vps-on-standby.png` and `41`. The "This device"
  paragraph ("Not a Tailscale address … tailbond route is approved in the
  Tailscale admin console") is the longest text on the main screen and means
  nothing to me. *Suggestion:* collapse it behind "What is this address?".
- **low**: `add-server-subnet-clash-desktop.png` against `45c`. The router and
  the phone explain the clash differently ("the Linux installer picks a free
  range by itself" against "asking over Tailscale does this by itself while
  vps has no other devices"). *Suggestion:* use one shared explanation.

---

## Overall count

| Severity | Count |
|---|---|
| High | 3 |
| Medium | 14 |
| Low | 19 |
