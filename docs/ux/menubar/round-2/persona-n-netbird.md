# Round 2: persona N, NetBird at work

## Task 1: bonding, and which connections

Steps: menu-01: "Connected, to homebox · 2 of 3 links in use"; Wi-Fi (en0) and iPhone (USB) (en7) with delay and speed; Sam's phone on standby.

- **Medium:** "Sam's phone (sh5a1b2c)" is someone else's device sitting in my path on standby. Nothing says who paired it, what the code means, or how to remove it. With Bonded, my work traffic could leave through it. Suggest: "shared by Sam, paired <date>", plus a way to remove it.
- **Low:** the on/off switch has a grey track in both light and dark, so it looks off or disabled next to the green "On". Use the accent colour.
- **Low:** each link shows only ↓, but the header shows ↓/↑. The grey dot for standby has no legend.

Verdict: done in seconds.

## Task 2: work VPN goes direct

Steps: menu-02: "NetBird is on (100.92.1.2/16): its traffic goes Direct"; Work VPN traffic is set to **Direct**; "Now: Direct". Nothing to change.

- **Medium:** "the work VPN and its networks (100.92.0.0/16, 10.20.0.0/16) use this Mac's own internet" mixes up two things. One is the inner routes. The other is NetBird's own encrypted UDP to its peers, relays and management server. If tailbond is my default route, I can't tell whether NetBird's outer packets skip the tunnel. Say both, and name the interface: "NetBird's packets and 100.92/16, 10.20/16 leave via Wi-Fi (en0)".
- **Medium:** the MTU warning's numbers don't add up. NetBird's 1280 is already more than 1180, so adding "+80" either counts the overhead twice or means 1180 is a different figure. "This connection" doesn't say which link. Suggest: "NetBird needs 1280-byte packets; over Wi-Fi tailbond carries at most 1180."
- **Medium:** an orange ⚠ in the work-VPN section made me think my current setup was broken. It's only about Bonded, which I'm not using. Use an info icon while Direct is selected.
- **Medium:** **Bonded** still looks clickable although the warning says it isn't possible. Disable it and say "(not possible on this connection)".
- **Low:** no hint whether routes NetBird adds later are picked up automatically. No way to test that a name like corp.example resolves.

Verdict: I can confirm Direct, but I have to trust the text on the outer packets.

## Task 3: updates

Steps: menu-01 says "tailbond 0.2.0 is up to date". In menu-02 I'd change Install updates from **Tell me** to **Right away**.

- **Medium:** menu-03 shows "Update available: tailbond 0.2.1" but not the version I have now. Show "0.2.0 installed · 0.2.1 available".
- **Medium:** screens.md calls it a security update, but the UI doesn't mark it as one. I'd click **Install now** for a security fix, but I can't tell it is one.
- **Medium:** the screen doesn't say whether installing restarts the tunnel or drops Direct/NetBird traffic during a call. "No call or big download" partly reassures, but it should say whether work traffic is affected.
- **Low:** "Right away" can wait up to 30 minutes. Rename it "As soon as idle". Show a last-checked time.

Verdict: the setting is easy to change. Version and urgency are unclear while an update waits.

## Task 4: report the crash

Steps: menu-03 banner **Report a problem…**, then the dashboard (report-01), then the form (report-02). I'd fill in What happened, read Crash details and Logs, tick the box, and click **Open GitHub issue** (report-03).

- **High:** the notice says "Private addresses (10.x, 192.168.x, 100.x) stay". That publishes my NetBird IP (100.92.1.2) and the company's 10.20.0.0/16 in a public issue. Work VPN names get redacted but its addresses don't. tailbond already knows those ranges (menu-02), so redact them as [work VPN address]. As it stands I'd use **Save report file** and scrub it by hand.
- **Medium:** "your work VPN's internal names (where tailbond knows them)" doesn't say which names. List the redacted domains in the notice (corp.example, netbird.cloud), so I know anything else under another corp domain will leak.
- **Medium:** the log box is small and scrolls, and the issue gets only the newest lines that fit. I tick "nothing I don't want public" without seeing exactly which lines go into the link. Show the text that will be sent, or a count.
- **Low:** "homebox" and the /tmp paths stay in the text (device names, which the notice tells me to check).
- **Low:** the crash time is 6:51 AM in menu-03, 7:50:25 AM on the dashboard, and in UTC (Z) in the crash details. I can't tell it's the same crash. Use one time zone.
- **Low:** in menu-03, Report a problem appears only in the banner, so after Dismiss it's gone. menu-01 shows it with no crash, and the doc says it's only shown after a crash.

Verdict: I'd file only through a saved, hand-scrubbed file. The public path leaks the company network.
