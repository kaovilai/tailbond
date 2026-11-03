# Round 2: persona C, "family member"

I'm a relative of the person who set this up. I use the phone app, and I open
the travel router's page when someone asks me to. I don't know what a VPN
server, a subnet or a port is. I worry about breaking things. On every screen
I want two answers: "is it working?" and "what should I do?".

Sources: the task script, both screenshot manifests, and every PNG in
`out/ux/bypass-failover/glinet/` and `out/ux/bypass-failover/android/`.

---

## Task 1: make my bank's website skip the tunnel

**Done?** Router: **partly**. With the rule saved I couldn't tell whether it
worked, and in the "waiting" state it doesn't work at all. Phone: **yes** on
Android 13+, **partly** on Android 12.

Screens: bypass-empty-desktop, bypass-add-website-presets-desktop,
bypass-rules-desktop, bypass-rules-phone, bypass-website-rules-wait-desktop,
bypass-add-streaming-preset-desktop, bypass-add-wifi-calling-desktop,
bypass-wifi-calling-added-desktop, 46-websites-skip-tailbond (+dark),
46c-websites-none-yet, 47-add-website-dialog, 47b/47c/47d-add-website-preset-*,
46b-websites-needs-android-13, 48-settings-skip-rows-android-12.

The router's "Banking" quick pick and the phone's "Add a website" were easy to
find. The opening text on both says plainly what the feature is for. The
problems are below.

### Findings

1. **high**, `bypass-website-rules-wait-desktop.png`: My bank rule shows
   "Waiting", so the bank is still blocked. The fix is to install
   "dnsmasq-full" from Plug-ins, or to run an SSH command that includes
   `opkg remove dnsmasq`. I won't remove part of the router on my own. I'd stop
   here, and the bank still wouldn't work.
   *Suggestion:* add an "Install it for me" button that confirms first and
   shows progress, and say plainly that the internet keeps working while it
   installs. Move the SSH command behind an "Advanced" link. Also add a line
   such as "Not sure? Send this to whoever set up your router", with a Copy
   button.

2. **medium**, `bypass-add-website-presets-desktop.png` vs
   `47-add-website-dialog.png` / `47b-add-website-preset-banking.png`: The
   router says `mybank.com` "includes all its pages and parts, like
   www.mybank.com and login.mybank.com". The phone says it covers "the name and
   its www. name only; add login.mybank.com separately". So the same
   `mybank.com` works on the router but may still be blocked on the phone, and
   I can't tell which explanation to trust.
   *Suggestion:* make both cover the same names, preferably every
   `*.mybank.com` name on the phone too. If they can't match, show the same
   explanation on both and add `login.<site>` automatically for the Banking
   pick.

3. **medium**, `bypass-add-website-presets-desktop.png` (also the footer on
   `bypass-rules-desktop.png`): The rule "may not be matched" on devices with
   "Private DNS on Android, Secure DNS in Chrome". I don't know whether my
   phone or laptop has those, so I can't tell whether my bank rule works for
   me. "37 connections" on the rule row doesn't tell me either.
   *Suggestion:* show whether the rule is working for each device, e.g. "Used by
   alex-iphone 2 min ago", or "No device has used this yet: open your bank's
   site to check". Warn on the device itself when its own DNS setting will get
   around the rule.

4. **medium**, `46b-websites-needs-android-13.png`: On Android 12, "for a
   browser use a separate one just for banking" assumes I know how to install
   and choose a second browser. I'd give up on the website and might not have
   the bank's app.
   *Suggestion:* add a button that opens the Play Store with a suggested
   browser, and two lines on what to do next ("open your bank's website in
   that browser only"). Or say plainly "Not possible on this phone; ask whoever
   set it up".

5. **low**, `47b-add-website-preset-banking.png`: Tapping "Banking" adds
   nothing and "Add" stays grey. At first it looks broken, until I read the
   paragraph.
   *Suggestion:* put the cursor in the Website box and change its placeholder
   to "Your bank's website, e.g. mybank.com".

6. **low**, `bypass-add-website-presets-desktop.png`: "Use the internet
   directly" on hotel Wi-Fi sounds like I'm turning protection off for my bank,
   of all sites. Nothing reassures me that this is safe.
   *Suggestion:* add one line: "Your bank's site is still encrypted (padlock
   in the browser); it just doesn't go through your home server."

7. **low**, `bypass-add-wifi-calling-desktop.png` vs
   `47d-add-website-preset-wifi-calling.png`: The router says Wi-Fi calling
   must skip tailbond because carriers refuse it through a VPN. The phone says
   "Nothing to add" because it already skips. Two different answers to the
   same pick leave me worried I've set one of them up wrong.
   *Suggestion:* on the phone, say "Already handled on this phone: nothing to
   do", and mention that the router needs the rule for other phones on its
   Wi-Fi.

8. **low**, `bypass-add-streaming-preset-desktop.png` vs
   `47c-add-website-preset-streaming.png`: The same "Streaming" pick adds five
   sites on the router (Netflix, Disney+, Prime, Max) and only two Netflix
   names on the phone.
   *Suggestion:* use the same list on both, or name the pick "Netflix" on the
   phone.

9. **low**, `46-websites-skip-tailbond.png` / `46c-websites-none-yet.png`: The
   two footer paragraphs ("looked up through your server…", "big hosting
   networks", "private DNS", "the connection blip") are dense technical text
   in the main area of the page.
   *Suggestion:* keep the first sentence ("Changes apply within a few
   seconds") and move the rest behind "How this works".

10. **low**, `bypass-wifi-calling-added-desktop.png`: The success toast floats
    over the Home server card, far from the Bypass card I was using, so I
    looked in the wrong place.
    *Suggestion:* show the confirmation inside the Bypass card, or highlight
    the new row.

11. **low**, `bypass-rules-desktop.png` / `46-websites-skip-tailbond.png`:
    Each row has a trash can right next to an unlabelled switch. I'm afraid of
    tapping the wrong one, and the screens don't show whether delete asks
    first.
    *Suggestion:* ask before deleting, or offer Undo, and give the switch an
    "On/Off" label or tooltip.

---

## Task 2: make my work laptop always use tailbond

**Done?** Router: **partly**. I could add the rule. I couldn't be sure it
keeps matching, or that "waits" means no internet. The phone doesn't apply.

Screens: bypass-add-device-picker-desktop, bypass-add-device-picker-phone,
bypass-rules-desktop, bypass-rules-phone.

### Findings

12. **medium**, `bypass-add-device-picker-desktop.png`: The rule only keeps
    working if I turn off "Private Wi-Fi address" (or "Random hardware
    addresses") on the laptop for this Wi-Fi. If I don't, or a work IT policy
    turns it back on, the rule silently stops matching. The laptop would then
    follow the bypass rules and could leave through the hotel's network, which
    is exactly what I was asked to prevent. Nothing on the rule row would tell
    me.
    *Suggestion:* when the selected device uses a random address, warn me as I
    add the rule. On the row, show "work-laptop hasn't been seen since …; its
    address may have changed" whenever the rule stops matching.

13. **medium**, `bypass-add-device-picker-desktop.png`: "If tailbond is down,
    it waits rather than going around it" doesn't tell me the laptop will have
    **no internet at all** while tailbond is down. When that happens I'll think
    the router broke and may delete the rule.
    *Suggestion:* say "If tailbond is down, this laptop has no internet until
    it's back. This is on purpose, so work traffic never uses the hotel Wi-Fi."
    Show the same text on the rule row while it's happening.

14. **low**, `bypass-add-device-picker-desktop.png`: Devices are listed by
    MAC address and IP address, and one is "Unnamed device". I can't match
    these to "my work laptop" except by name.
    *Suggestion:* hide the MAC and IP behind a details toggle, and show the
    device type or maker ("Apple laptop") for unnamed devices.

15. **low**, `bypass-rules-desktop.png`: In the Bypass card, `work-laptop`
    says "connected now". Lower on the same page, under Tailnet machines,
    `work-laptop` says "Last seen 1 day ago". It's the same name with two
    answers to "is it on?".
    *Suggestion:* label the second list "seen on Tailscale", or don't repeat
    the name without context.

16. **low**, `bypass-add-device-picker-phone.png`: On a phone-sized screen,
    the Cancel and Add buttons are cut off at the bottom of the dialog.
    *Suggestion:* keep the dialog buttons fixed at the bottom where they're
    always visible.

---

## Task 3: add the VPS as a backup server

**Done?** Router: **partly**. vps was found and "Ask to add" was clear. When
it clashed, I couldn't fix it myself. Phone: **partly**, for the same reason,
and I also had to know the server's name.

Screens: servers-single-home-desktop, add-server-open-desktop,
add-server-waiting-approval-desktop, add-server-subnet-clash-desktop,
add-server-paste-clash-desktop, servers-home-active-vps-standby-desktop,
44b-settings-one-server-add-backup, 45-add-another-server,
45b-add-another-server-confirm, 45c-add-another-server-same-addresses.

### Findings

17. **medium**, `add-server-subnet-clash-desktop.png` /
    `add-server-paste-clash-desktop.png` / `45c-add-another-server-same-addresses.png`:
    The clash message gives three sets of terminal commands and file paths,
    with subnets like `10.78.0.0/24`. I don't know which applies to "my" vps,
    and I can't run any of them. The router says "homebox keeps working;
    nothing was changed", which helped. The phone doesn't say that, and "Asking
    over Tailscale does this by itself while vps has no other devices" leaves
    me unsure whether to just try again. The router still shows an active "Ask
    to add" button under the error, which invites a retry that will fail.
    *Suggestion:* lead with "Nothing changed; homebox still works. vps needs a
    small change by whoever set it up," plus a "Copy message to send them"
    button. Put the commands behind "Show how". Add the "nothing changed" line
    on the phone, and disable "Ask to add" until vps changes.

18. **medium**, `45-add-another-server.png`: On the phone I have to type the
    server's exact Tailscale name and pick from four methods. The router simply
    showed "Found on your tailnet: vps · Ask to add".
    *Suggestion:* list the tailbond servers the phone can see, with one tap to
    ask. Put the code, link and page methods under "Other ways".

19. **medium**, `45-add-another-server.png`: "Someone approves this phone on
    that server" doesn't say who, where or how. The router at least gave the
    page link and said "click Approve next to slate7".
    *Suggestion:* after "Ask to connect", show the same waiting card as the
    router, with the vps page link and "Approve 'pixel-8' there".

20. **medium**, `servers-home-active-vps-standby-desktop.png` (and "This
    device" on `40-home-on-homebox-vps-on-standby.png`): "for your Tailscale
    machines to reach devices here through each server, approve each server's
    route in Tailscale" reads like an unfinished job. I don't know whether I
    must do it, or whether something is broken until I do.
    *Suggestion:* only show this when the route really isn't approved, as a
    clear "Action needed (optional): …" line. Otherwise hide it behind
    "Details".

21. **low**, `add-server-waiting-approval-desktop.png`: "click Approve next to
    slate7". I didn't know the router's name was slate7. It only appears in
    small grey text elsewhere on the page.
    *Suggestion:* write "next to slate7 (this router)".

22. **low**, `servers-single-home-desktop.png` / `44b-settings-one-server-add-backup.png`:
    "Add a backup server" sits right next to a red "Unpair" or "Remove from
    this phone". An anxious user could hit the wrong one.
    *Suggestion:* move Unpair/Remove into a menu, or put space between it and
    the add button. Always ask before removing.

23. **low**, `45b-add-another-server-confirm.png`: "It connects over the
    internet to 203.0.113.9:8443" is noise to me. The rest of the confirmation
    is clear.
    *Suggestion:* move it under "Technical details".

---

## Task 4: notice the switch, and that it came back

**Done?** Router: **partly**. While the page is open, the header says "Using
vps since 14:12 (homebox is unreachable)". It never says I'm fine or that I
don't need to do anything, and once it's back there's no record of the outage.
Phone: **yes** for "now on vps, and why" ("You're still connected…"), and
**partly** for "when it came back".

Screens: servers-home-active-vps-standby-desktop,
failover-using-vps-home-unreachable-desktop/-phone, failback-countdown-desktop,
failover-you-picked-vps-desktop, back-on-homebox-since-desktop,
40-home-on-homebox-vps-on-standby (+dark), 41/41b/41c, 42, 43, 43b, 44 (+dark),
44c, 49.

### Findings

24. **medium**, `failover-using-vps-home-unreachable-desktop.png`: The router
    says "homebox is unreachable" and shows a red "Can't reach". It never says
    "you're still online, nothing to do", and it doesn't suggest why (power
    cut, or the internet at home is down). The phone's "You're still
    connected. It goes back to homebox by itself…" is exactly what I needed,
    and the router doesn't have it.
    *Suggestion:* add the phone's sentence to the router's header, plus "Common
    cause: homebox lost power or its internet at home."

25. **medium**, `failover-using-vps-home-unreachable-desktop.png`: The blue
    "Remove" link sits on the same row as the red "Can't reach" badge. An
    anxious person "fixing the red thing" might remove homebox, the main
    server.
    *Suggestion:* put Remove behind a ⋮ menu, as the phone does, and ask
    "Remove homebox? You'll need to pair it again."

26. **medium**, `41-home-using-vps-homebox-unreachable.png`,
    `42-home-back-to-homebox-in-1-20.png`, `43b-home-switching-to-vps.png`,
    `49-home-error-no-server-reachable.png`: A large "Switch back to Tailscale"
    button sits right under every warning. When something looks wrong, "switch
    back" sounds like "go back to normal / go home", and I'd tap it. That turns
    tailbond off.
    *Suggestion:* move it into Settings or a menu on the main screen, or rename
    it "Turn off tailbond (use the Tailscale app)" and confirm before acting.

27. **medium**, `42-home-back-to-homebox-in-1-20.png`: "Stay on vps" is the
    only filled button in the status card. It looks like the "OK" to tap.
    Tapping it keeps the phone on vps until someone picks homebox again
    (41c), which is the opposite of what I want.
    *Suggestion:* make it a text link like on the router, and add "Nothing to
    do: it goes back by itself."

28. **medium**, `back-on-homebox-since-desktop.png` / `43-home-back-on-homebox.png`:
    Once it's back, all I get is "Back on homebox since 14:11 / 14:20". I can't
    see that there was an outage, when it started, or how long vps carried
    things. According to the manifest, the router line exists only because the
    page was open at the time.
    *Suggestion:* add a short "Recent" line that stays visible for a day, e.g.
    "homebox was unreachable 14:02 to 14:20; vps took over." Show it on both
    the router and the phone.

29. **medium**, (no screen): None of the screenshots shows a phone
    notification or a router alert for the switch. Unless I happen to open the
    app or the router page, I won't notice anything happened.
    *Suggestion:* show a quiet Android notification ("Using backup server vps:
    homebox isn't answering") and another when it switches back. If these
    already exist, add them to the screenshot set.

30. **low**, `41-home-using-vps-homebox-unreachable.png` / `43b-home-switching-to-vps.png`:
    "(Tailscale may still show it online below)" gives me two opposite answers
    on the same screen.
    *Suggestion:* drop it from the main screen, or say "Ignore the green dot in
    Tailscale; tailbond checks more closely."

31. **low**, `43b-home-switching-to-vps.png`: "apps use the phone's own
    connection, unless Android's 'Block connections without VPN' is on". I
    don't know whether that's bad, or whether I should change it.
    *Suggestion:* say "For a few seconds your phone uses its normal internet.
    Nothing to do."

32. **low**, `failover-using-vps-home-unreachable-desktop.png` /
    `failback-countdown-desktop.png`: "Up 1 min" resets when it switches, which
    looks as if the internet was down.
    *Suggestion:* keep the total time online, or label it "On vps for 1 min".

33. **low**, `failover-using-vps-home-unreachable-desktop.png` vs
    `back-on-homebox-since-desktop.png` (and the phone version vs the desktop
    version): The times don't line up. It says "Using vps since 14:12" but also
    "Back on homebox since 14:11", and homebox "last answered" 2 min ago on
    desktop vs 3 min ago on the phone render. A careful reader trying to piece
    together the story gets confused.
    *Suggestion:* make the fixture times follow one consistent story.

34. **low**, `41c-home-using-vps-you-picked-it.png` / `failover-you-picked-vps-desktop.png`:
    "(you picked it)". I didn't. Someone else did, on the router or in
    Settings, and now I think I broke something.
    *Suggestion:* say "picked by hand".

35. **low**, `44c-settings-servers-homebox-answering-again.png`: Settings
    offers "Use this server now" for homebox but doesn't show the 1:20
    countdown from the main screen. I'd tap it and wonder whether I needed to.
    *Suggestion:* show "Goes back by itself in 1:20" on the homebox card.

36. **low**, `44-settings-servers-dark.png`: In dark theme, the
    "Unreachable" dot is pale pink and barely reads as an error.
    *Suggestion:* use a stronger red, or an icon, in dark theme.

37. **low**, `44-settings-servers.png`: Settings has "exit node", "MagicDNS",
    "100.x addresses" and "LANs behind other subnet routers", with radio
    buttons I could tap by accident ("Only Tailscale machines") and the
    unlabelled ↑/↓ arrows that change which server is "Main".
    *Suggestion:* put "Internet traffic" and the "How it decides" text under
    "Advanced", and label the arrows "Move up / Move down".

---

## Overall count

| Severity | Count |
|---|---|
| high | 1 |
| medium | 15 |
| low | 21 |

High: #1. Medium: #2, #3, #4, #12, #13, #17, #18, #19, #20, #24, #25, #26,
#27, #28, #29.
