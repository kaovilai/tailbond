# Round 1: persona A, traveller (MacBook, iPhone)

## Task 1: is it bonding, and which connections?

Steps: I click the menu bar icon (menu-01-connected.png). It says "Connected, to homebox · 2 links", and under CONNECTIONS there are Wi-Fi and iPhone (USB) with green dots and speeds. Sam's phone says "standby".

- **Medium**: The switch at top right is pale grey in both light and dark. I can't tell if it's on. Toggles that are on usually have a colour. *Give it the accent colour when on.*
- **Low**: The header says "2 links" but the list has three rows, and nowhere does it say "bonding". *Write "Bonding 2 connections" in the header, and use "connections" everywhere.*
- **Low**: "(en0)", "(en7)" and "(sh5a1b2c)" mean nothing to me. *Hide them or move them to a tooltip.*
- **Low**: "standby" doesn't say whether it will use Sam's data on its own. *Add "used if the others fail".*
- **Low**: ↓23.4 / ↑1.2 Mbps has no label. Is that my speed right now, or all the connections added up?

Verdict: I'd pass in seconds. The unclear switch is the only thing that slows me down.

## Task 2: work VPN goes direct

Steps: I see NEXT TO OTHER VPNS (menu-02-netbird.png). "Its traffic goes Direct (not through tailbond)" answers the question. "Work VPN traffic: Direct | Bonded" is the control.

- **Medium**: The Direct/Bonded control barely shows which side is picked: light grey on light grey, and the same weak contrast in dark. *Fill the picked side with a strong colour.*
- **Medium**: The orange warning, "NetBird's network 10.20.0.0/16 is also your home LAN's: NetBird keeps it", is all jargon to me. I can't tell if something is broken or if I have to do something. *Say "Some devices at home can't be reached while NetBird is on. Nothing to do unless you need them."*
- **Medium**: Ethernet shows red "not working" at a café where nothing is plugged in. That makes me think something is broken. *Show "not plugged in" in grey, or hide it.*
- **Low**: "(100.92.1.2/16)" and "DNS" in the info lines are noise for me. *Remove them.*
- **Low**: Nothing tells me what happens if I pick Bonded. *Add a one-line hint.*

Verdict: I'd pass because it's already Direct, but the warning would worry me.

## Task 3: updates

Steps: UPDATES shows "tailbond 0.2.0 · up to date · Right away" (menu-01). The "Install updates" menu is already on Right away. In menu-02 it's "Tell me", so I'd open that menu and pick Right away.

- **Medium**: In menu-03-crash-update.png, the line says "Update available: 0.2.1 (security) (waiting for a quiet moment) · Right away". That contradicts "Right away", doesn't say when it will happen, and there's no button to install it now. Only "Check now" is there. *Add "Install now", and say what a quiet moment is ("when no call is running").*
- **Low**: The setting is shown twice, once in the status line and once in the menu next to it. In menu-02, "up to date · Tell me" reads oddly. *Drop it from the status line.*
- **Low**: I don't know if "Check now" installs anything or just checks.

Verdict: I'd pass the setting part. With a security update waiting, I'd be stuck not knowing when it lands.

## Task 4: report the crash

Steps: I click "Report a problem…" in the crash banner (menu-03). The browser opens the dashboard (report-01-crash-banner-desktop.png), where I click "Report a problem" again. That opens the form (report-02-preview-desktop.png). I type what happened, scroll down, tick the box and click "Open GitHub issue" (report-03-ready-desktop.png).

- **High**: The crash details and log lines run off the right edge of the box, so I can't read them all. The checkbox makes me promise "I've read this report", and I can't check it for private info. Even if I could read it, it's code to me. I'd either tick it blindly or give up. *Wrap the lines, and add a plain summary: "We hid 1 key, 2 IP addresses, 1 email. Check names of your Wi-Fi or devices."*
- **Medium**: The fields are filled in wrong for my Mac: "Where: VPS / Linux install.sh" and "Fedora Linux 44 (WSL)". I don't know what "VPS" is or what to choose instead. *Detect the Mac, and offer plain choices like "Mac app" or "home server".*
- **Medium**: The dashboard I land on says "Home server", "No devices connected" and a red "1 thing to fix before devices can connect from away". Is this my Mac or homebox? Did I break something? *Open the report form directly from the menu, and say whose dashboard this is.*
- **Medium**: "Open GitHub issue" is greyed out with no reason. The checkbox that unlocks it is below the fold. *Add "Tick the box below to continue".*
- **Medium**: Nothing says I need a GitHub account. I don't have one. Copy and Save don't say where to send the file. *Say so, and give an email or another way to send it.*
- **Low**: The crash time in the menu is "2026-11-04T06:51:24Z" (UTC, raw), but the dashboard says 7:14:10 AM. I'd wonder whether there were two crashes. *Show local time, in the same format everywhere.*
- **Low**: menu-03 has two identical "Report a problem…" buttons, one in the banner and one in the footer.
- **Low**: The grey example text in "What happened" looks like an answer that's already filled in.

Dark mode: the menus read fine. The only problems are the weak on/off switch and the Direct/Bonded control, as in light.

Verdict: I'd get a report sent, but I couldn't honestly tick the privacy box. I might quit at the GitHub sign-in.
