# Round 11 — Fedora persona F2 (home admin)

**Persona:** I run the family's home server on a spare Fedora PC (`homebox`). I can `dnf install`, paste commands from a guide, and I set up Tailscale from its website. I don't really know firewalls, NAT or port forwarding. I set things up for the family.

**Materials used:** `task-script.md`, `screens.md`, `fedora-server.md`, and the screenshots in `glinet/` and `web/`. Nothing else.

---

## Task 1: Install tailbond on homebox and open its dashboard

**What I'd do**
- Guide §1 "Install": `sudo dnf install ./tailbond-server-*.rpm`, then `sudo systemctl enable --now tailbond`. I'm on Fedora 44 and the guide says "Fedora 43 or newer", so that's fine. I'd check it with `systemctl status tailbond`.
- To open the dashboard on homebox itself I'd go to `http://127.0.0.1:8077/`. From my laptop (on Tailscale, same account) I'd use `http://homebox:8077`. The guide says this clearly (§1, "To open the dashboard…").
- The guide has a long firewalld list, but it says "There's nothing to do here yourself; this list is for checking". That helps a lot, so I'd skip the list. I'd also skip "Optional HTTPS" because it says "It isn't needed".

**Where I hesitate**
- The header in 15c reads "Home server · reachable at 192.168.1.2:443". Right below it, the orange card says devices "only work at home". "Reachable" made me think it already worked from anywhere for a moment. (low)
- There's a green **Connected** toggle in the top-right of a *server* dashboard. I don't know what turning it off does. Does it kick everybody off? Stop the service? I'd be afraid to touch it. (low)

Otherwise this was smooth. I'd finish with no real trouble.

## Task 2: Right after installing (15c): what's left to set up, and how?

**What I'd do**
- The orange card at the top of **Devices** is titled "Setup step: tell devices where home is". It gives two numbered things to do on my router: "Forward UDP port 443 to this machine (192.168.1.2)… Give this machine a fixed address there too (DHCP reservation)" and "Find your home's public address… Type it here", then **Save**. It matches guide §2 step for step.
- I like "The page is usually called Port forwarding, Virtual server or NAT". I like "Not sure how? Show this card to whoever set up your internet" even more. That's me calling my brother-in-law.
- Setup health says "nothing to fix here (1 optional item below) · finish the setup step above first", so I know the orange card comes first.

**Where I hesitate**
- **Two trips to the router.** This card only asks for UDP 443. Later (screen 15) the health card asks me to "forward UDP port 41641 to this machine". The guide's §1 already says 41641 is opened "so a forward on your router is all direct Tailscale connections need". While I'm logged into the router I'd rather add both forwards at once, but the card never mentions 41641. (low)
- **How do I know the forward worked?** The card only says "Setup health below checks for that" about carrier-grade NAT. Nothing on screen tells me how to check the forward from home. Only guide §2 step 4 does: "on that phone, turn Wi-Fi off; once tailbond connects over mobile data, the dashboard's *UDP 443* health item turns green". I can live with that, but after clicking Save I'd have no idea whether my router setup was right. (low)
- **The optional route.** "Route 10.77.0.0/24 isn't approved yet (optional)" gets the same orange "!" as real problems. The text is clear ("Devices connected through tailbond reach your Tailscale machines without it… If you want that…"), so I'd leave it alone. But on the all-good screen (15d) the heading "Setup health: everything is set up for best speed" lists "✓ Route 10.77.0.0/24 approved" among the checks. The guide's table says it's "a choice, not a speed problem". After seeing 15d, I'd feel I *should* approve it just to get all green ticks. (low)

## Task 3: A family member's GL.iNet travel router is asking to connect. Approve it.

**What I'd do**
- Screen 14 (desktop, and the phone layout too): a "Waiting for your approval" card sits at the top. It says "slate7 wants to connect · Travel router on your tailnet · signed in as alex@example.com · GL-BE3600 · 100.101.9.9", with **Decline** and **Approve** buttons. I'd click **Approve**.
- Screen 14b: the toast "slate7 can connect now. It connects by itself in a few seconds." is reassuring.

**Where I get stuck**
- **The orange warning stops me.** "It's signed in to alex@example.com, an account that may approve here: whoever holds this router can also open this dashboard and approve devices. If it's for someone else, sign it in to their own Tailscale account first, then let it ask again." This router *is* for someone else; I set it up with my own account because that was easiest. So now I'm unsure:
  - Do I press **Decline** first, or just leave the request sitting there? "Then let it ask again" doesn't say.
  - How do I sign a GL.iNet router into a different Tailscale account? Nothing in the packet shows that screen. The guide's only other option is "or tag them", and I don't know how to tag a device.
  - Do my family members even have their own Tailscale accounts? Inviting them comes up later in the phone card, not here.

  In practice I'd probably click Approve anyway because it's my kid, which means I'd be ignoring the warning. That works, but I'm uneasy and I don't know the "right" path. (**medium**)
- The explanation above the request ends with "reach your Tailscale machines as homebox: they allow it whatever they allow homebox, which may be more than its owner's own account gets." I had to read "they allow it whatever they allow homebox" three times. Who is "they"? My machines? My access rules? (low)
- After approving I expected slate7 to show up somewhere right away. On 14b the Connections tab still shows only pixel-8. On 14d, later, slate7 still isn't under Connections. I finally find it in the Devices list (15d) as "not connected right now (approved devices connect by themselves when they're on and online)", **with a phone icon**, even though it's a router. I'd wonder whether the approval actually worked. (low)

## Task 4: Get tailbond onto a family member's Android phone and get it connected

**What I'd do**
- On the Devices tab, the "Add an Android phone" card (15c / web-04) gives me a QR code, "type homebox/ into its browser (with the slash at the end…)", and a copy button for `http://homebox.tail1234.ts.net/`. The family member isn't next to me, so I'd copy that link and text it to them.
- On their phone (web-00), the page "Get tailbond on this phone" walks through: (1) five pictures of the Android install prompts, then **Download tailbond (27 MB)**; (2) **Open tailbond and ask homebox**; (3) "tap Add phone. Then tap the big switch". This is really good. I could talk my mother through it on the phone with these pictures.
- Back on my dashboard (14c): "pixel-7a wants to connect · Phone on your tailnet · signed in as sam@example.com · Pixel 7a". There's no orange warning this time, so I'd click **Approve**.
- 14d: pixel-7a appears under Connections with "2 active path(s)". Done.

**Where I get stuck**
- **Invite or share? I can't tell which to pick.** The card says: "their phone needs Tailscale signed in to your Tailscale network: invite them as users (Tailscale admin console → Users → Invite; your access rules decide what they reach), or share just this machine with their own account (Machines → this machine → Share; they then see only this machine; they type its full name, shown in the link below)." The task says the phone "has Tailscale on it", but I don't know whose account it's on. I'm offered two methods with no recommendation for "my family". I don't know what "your access rules decide what they reach" means for my NAS. I'd guess, or ask someone. (**medium**)
- **"They then see only this machine" isn't the whole story.** That line made me lean towards *Share*, because it sounds limited and safe. But the approval card (14c) says an approved device can "reach your Tailscale machines as homebox… which may be more than its owner's own account gets". The phone page (step 3) says "Your Tailscale machines stay reachable through homebox. They see this phone as homebox… so they allow it whatever they allow homebox." Guide §4 says outright: "your Tailscale access rules for this server apply to every bonded device". So once I approve a shared person, they reach my whole tailnet and home LAN, not "only this machine". It's family so I'd accept it, but the Add-phone card points me the wrong way on privacy. (**medium**)
- The phone page tells the family member "Whoever runs homebox taps Approve on homebox's page; you'll see it in the app right away." Nothing tells *me* a request is waiting unless I happen to open the dashboard. The page doesn't tell them to message me either, only, under step 1, "Stuck… Ask whoever runs homebox". I'd add "let them know you're waiting" to the text I send them. (low)

## Task 5: Read the setup health and fix what it reports, including the relay machine

**What I'd do**
- Screen 15 (Devices tab): "Setup health: 2 things to improve (tailbond works meanwhile)". On the other tabs it's the beige "Show" bar. I'm glad it says it works meanwhile.
  1. "Tailscale: 2 machines direct, 1 through a relay (slower)". "Through a relay: apple-tv." The fix: "On your home router's admin page (Port forwarding, also called Virtual server or NAT), forward UDP port 41641 to this machine (192.168.1.2)." I already did the 443 forward on the same page, so I can do this. I'd skip UPnP because it "lets any device on your network open ports".
  2. "This network makes direct Tailscale connections hard". "Same fix as above (one forward fixes both)." Good, one change for both.
- After that, 15d shows "Tailscale: 3 machines direct" and "Network is friendly to direct connections". Guide §3 also gives me `tailscale ping <machine>` to check: "via DERP(…)" means relayed.

**Where I hesitate**
- "The cause can also be on that machine's own network." The screen doesn't say *where* apple-tv is (here at home, or at my parents'?), and it doesn't say what to do if the cause is on the other side. The guide only says "which you may not control". If my forward doesn't fix it, I'm stuck. (low)
- "…so other machines can't connect straight back; no port mapping on the router." I just learned "port forwarding", and now there's "port mapping". Is that something else I need to set up? (low)
- Guide §3 says "For best speed… see [setup tips](setup-tips.md)". I don't have that file. The dashboard's "How to fix" links probably go somewhere, but I can't see where. The health table in the guide was enough for me anyway. (low)

## Task 6: Decide and set who may approve new devices

**What I'd do**
- Devices tab → "Who can approve and add devices" (15c, 15, 15d). The current rule is spelled out: "alex@example.com (the Tailscale account homebox belongs to), from any device signed in to that Tailscale account, and anyone logged in on homebox itself (http://127.0.0.1:8077). Others on your Tailscale network… can ask to connect but can't approve."
- My decision: keep the default and add my partner's login in "Also allowed to approve (Tailscale logins, comma separated)", e.g. `sam@example.com`, then **Save**. I wouldn't set a password, because then "everyone who approves, you included, needs it" and I'd forget it.
- web-04 (another server, tagged) shows what I want to avoid: "Anyone on your Tailscale network can approve devices". The card and guide explain that only tagged servers have this problem. homebox isn't tagged, so I'm fine.

**Where I hesitate**
- There are two separate **Save** buttons. One is for approvers; the other is "Or a password for this page (replaces the above; empty: none)". If I save a password later, does it delete the approvers list, or just ignore it while a password is set? I'd be afraid of losing what I typed. (low)
- Guide §4, "What an approved device reaches", suggests "To limit it, tag the server and give the tag only what bonded devices should reach" and shows a block of `tagOwners` / `grants` JSON. That's far beyond me, and nothing in the dashboard does it. I'd skip it and accept that family devices reach what homebox reaches. (low)
- It ties back to Task 3: slate7 is signed in as me, so by this card's own rule ("A device you lend to someone while it's signed in to an approver's account can approve too") my kid's router can approve devices. Setting approvers doesn't fix that, and the packet doesn't show how to re-sign the router (already counted in Task 3).

---

## Findings

| # | task | finding | severity |
|---|---|---|---|
| 1 | 3 | Warning "It's signed in to alex@example.com… If it's for someone else, sign it in to their own Tailscale account first, then let it ask again": no guidance on whether to Decline first, or how to re-sign a GL.iNet router; I'd likely just Approve and ignore it | medium |
| 2 | 4 | "invite them as users… or share just this machine with their own account": two methods, no recommendation for family; "your access rules decide what they reach" means nothing to me | medium |
| 3 | 4 | "share just this machine… they then see only this machine" sounds limited, but once approved the phone reaches the tailnet and home LAN "as homebox" (14c, phone page step 3, guide §4); this pushes me towards a false sense of privacy | medium |
| 4 | 1 | Header "reachable at 192.168.1.2:443" right after install, while the card says devices "only work at home" | low |
| 5 | 1 | "Connected" toggle on a server dashboard; I don't know what switching it off does | low |
| 6 | 2 | Setup step asks only for UDP 443; the UDP 41641 forward comes later as a health item, so a second trip to the router | low |
| 7 | 2 | No on-screen way to confirm the 443 forward from home; only guide §2 step 4 (phone on mobile data) explains it | low |
| 8 | 2 | Optional route shown with the same orange "!" as problems, and listed under "everything is set up for best speed" on 15d although the guide calls it "a choice, not a speed problem" | low |
| 9 | 3 | "they allow it whatever they allow homebox": unclear who "they" is | low |
| 10 | 3 | After Approve, slate7 isn't under Connections (14b/14d); in Devices (15d) it shows with a phone icon and "Not Connected", so I can't tell whether it worked | low |
| 11 | 4 | Phone page says "Whoever runs homebox taps Approve" but doesn't tell the family member to let me know; I only see the request when I open the dashboard | low |
| 12 | 5 | "The cause can also be on that machine's own network": no location shown for apple-tv, no next step if my forward doesn't fix it | low |
| 13 | 5 | "no port mapping on the router": a new term next to "port forwarding" | low |
| 14 | 5 | Guide links "setup tips (setup-tips.md)", which isn't in what I was given | low |
| 15 | 6 | Approvers and password have separate Save buttons; "replaces the above" doesn't say whether saving a password erases the approvers list | low |
| 16 | 6 | Guide's "To limit it, tag the server…" `tagOwners`/`grants` JSON is beyond a home admin; no dashboard equivalent | low |

Totals: 0 high / 3 medium / 13 low
