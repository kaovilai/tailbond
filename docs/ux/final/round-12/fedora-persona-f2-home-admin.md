# Round 12 — Fedora persona F2 (home admin)

**Who I am:** I look after the family's home server, a spare PC running Fedora 44 called `homebox`. I can install packages with `dnf` and copy commands from a guide. I set up Tailscale by following its website. I don't really understand firewalls, NAT or port forwarding. I set things up so the rest of the family can use them.

**What I had:** `task-script.md`, `screens.md`, `fedora-server.md` and the screenshots in `glinet/` and `web/`. Nothing else.

---

## Task 1: Install tailbond and open the dashboard

**What I'd do:** I'd follow the guide's section 1, *Install*, and copy the two commands:

```sh
sudo dnf install ./tailbond-server-*.rpm
sudo systemctl enable --now tailbond
```

The guide says it supports "Fedora 43 or newer", and I have 44, so that's fine. I'd check it with `systemctl status tailbond` as the guide shows. Then I'd open `http://127.0.0.1:8077/` on homebox, or `http://homebox:8077` from my laptop. The guide covers both.

**What went well:** The long firewalld list in section 1 starts with "There's nothing to do here yourself; this list is for checking". That told me I could skip a part I wouldn't understand. The SELinux line ("SELinux needs no changes") answers a question I'd otherwise have searched for.

**Where I hesitated:**

- The same section continues: "To keep them off your home LAN, add a rule of your own" and gives a command with `destination address=192.168.1.0/24`. I'd be tempted to copy it as written. The guide doesn't tell me how to check that `192.168.1.0/24` matches my home network, so if mine is `192.168.0.x`, the rule silently does nothing. (low)
- In 15c, right after install, the header says "Home server · reachable at 192.168.1.2:443". The orange card just below says devices "only work at home". "Reachable" made me think it was already working from outside. (low)
- The top right shows a "Connected" switch and a green "Connected" badge, but there are "No devices yet". I didn't know what was connected. I also worried that clicking the switch would turn something off. (low)

## Task 2: What's left after installing (15c), and how

**What I'd do:** The orange card at the top of the Devices tab is clear about it: "Setup step: tell devices where home is". It gives two steps on my router's admin page. Step 1: "Forward UDP port 443 to this machine (192.168.1.2)… usually called Port forwarding, Virtual server or NAT. Give this machine a fixed address there too (DHCP reservation)". Step 2: find the public address "(WAN or Internet IP)", or better a DDNS name, then type it in and click **Save**. The guide's section 2 says the same thing, in the same order. Without knowing networking, I could follow the router-menu names it gives. I liked "The port (443) is added for you" and the reassurance that Setup health checks for carrier-grade NAT.

**Where I got stuck:**

- **The optional route item.** Setup health says "nothing to fix here (1 optional item below)". The item is still shown with an orange "!", though: "Route 10.77.0.0/24 isn't approved yet (optional)… If you want that: Tailscale admin console → Machines → this machine → Edit route settings… If you do not want it: leave it". I can't tell whether *I* want my Tailscale machines to "reach those devices too". I don't know what that means for my family's phones, and nothing recommends a choice for a home setup. The orange "!" makes it look like a problem to clear. Then 15d, the "everything is set up" screen, lists "✓ Optional route 10.77.0.0/24 approved" as one of the green ticks, so the finished state *looks like* approving it. I'd probably approve it to make the "!" go away. That exposes the bonded devices to my tailnet, which the guide warns is a deliberate choice ("approve it only if your Tailscale machines should reach the bonded devices"). (medium)
- The card says "Router fixes are on your home router's admin page, route ones in the Tailscale admin console". On this screen the only item is about a route, and no router fix is listed, so the sentence reads like boilerplate. (low)
- This card only asks for the UDP 443 forward. A second router forward, UDP 41641, comes up later in task 5. Someone like me, who finds the router admin page hard, would rather do both forwards in one visit. (low)

## Task 3: Approve a family member's GL.iNet travel router

**What I'd do:** 14 shows a "Waiting for your approval" card at the top, then "slate7 wants to connect · Travel router on your tailnet · signed in as alex@example.com · GL-BE3600". I'd click **Approve**. 14b then shows a toast: "slate7 can connect now. It connects by itself in a few seconds." The phone layout (14 -phone) works too: Approve is a big full-width button.

**Where I got stuck:**

- **The router is signed in to *my* Tailscale account.** That's normal for me, since I set up the family's devices myself. The orange warning says: "If you'll give it to someone: approve, then before handing it over sign its Tailscale in to their account (or tag it); tailbond keeps working, and until then whoever holds it could also approve devices here." This router belongs to a family member, so it applies to me. But nothing in the packet tells me *how* to sign a GL.iNet router's Tailscale in to a different account, and "tag it" is not something I know how to do. The guide's "Who can approve" section repeats the advice ("sign those in with the borrower's account, or tag them") but has no steps either. I'd approve the router, then either ignore the warning or spend an evening on Tailscale's website. (medium)
- The grey paragraph above the request is long. It includes "they allow it whatever they allow homebox, which may be more than its owner's own account gets", which I had to read three times. On the phone layout, that paragraph plus the warning push Approve down by about a screen. The same paragraph appears again in the "Who can approve and add devices" card on the Devices tab (15). (low)
- After approving, slate7 never shows up on Connections (14b, 14d). In 15d it's listed as "Not Connected" with a **phone** icon, even though it's a router. From the screenshots I can't confirm that "It connects by itself in a few seconds" actually happened. If the router were simply off, I couldn't tell. (low)
- Nothing tells me a request is waiting unless I have the dashboard open. The family member would have to tell me. (low)

## Task 4: Get tailbond onto a family member's Android phone and connected

**What I'd do:**

1. They need to be on my Tailscale network. The "Add an Android phone" card (15c/15) says "Simplest: invite them (Tailscale admin console → Users → Invite)". I can do that from Tailscale's website.
2. I'd copy `http://homebox.tail1234.ts.net/` with the copy button and send it, or tell them to type `homebox/` ("with the slash at the end, so it opens the page instead of searching", which is a good tip).
3. On their phone (web-00), the page "Get tailbond on this phone" shows drawings of the five Android prompts and a short line: "Download anyway → Open → Settings → Allow from this source → back → Install (→ Scan app, if asked)". Then they tap "Download tailbond (27 MB)", then **Open tailbond and ask homebox** (step 2), then turn tailbond on (step 3). I could talk a relative through this over the phone. "Is this safe? This page starts with http://, but your phone reached it through Tailscale…" answers the question they would ask me.
4. On homebox (14c): "pixel-7a wants to connect · Phone on your tailnet · signed in as sam@example.com · Pixel 7a". I recognise Sam, so I click **Approve**. In 14d, pixel-7a is on Connections with 2 active paths. Done.

web-00c (the same page opened on a computer) shows a QR code and the same steps, so if a relative opens the link on a laptop, they still get to the phone.

**Where I got stuck:**

- **What I'm actually giving Sam's phone.** The card says "Once you approve their phone here, it reaches what this server reaches, whichever way they joined". The phone page says "They see this phone as homebox… so they allow it whatever they allow homebox". So Sam's phone gets the same access homebox has to all my Tailscale machines (my NAS, my laptop…). Inviting Sam as a Tailscale user would normally give them less than that. The only fix the guide offers is to "tag the server and give the tag only what bonded devices should reach", using a `tagOwners`/`grants` JSON policy. I can't write that, and it comes with a warning that tagging removes the server's owner. I'd approve anyway, but without really knowing what I agreed to. (medium)
- In 14d the totals at the top still say "1.1 MBPS FROM DEVICES / 12.0 MBPS TO DEVICES", exactly as before, even though pixel-7a now adds 200 kbps + 100 kbps up and 2.8 + 1.4 Mbps down. It looks like the totals don't include the new phone. (low)
- Labels like "Min 31 Ms", "MBPS" in capitals and "198.51.100.20:51234 · wlb" in the device row (15, 15d) mean nothing to me. I can't tell whether "Ms" is milliseconds and what "Min" is the minimum of, and "wlb" is unexplained. (low)

## Task 5: Read setup health and fix it, including the relayed machine

**What I'd do:** 15, Devices tab: "Setup health: 2 optional speed-ups (tailbond works fine without them)".

- "Tailscale: 2 machines direct, 1 through a relay (slower). Through a relay: apple-tv." The fix is plain: "On your home router's admin page (Port forwarding, also called Virtual server or NAT), forward UDP port 41641 to this machine (192.168.1.2)."
- "This network makes direct Tailscale connections hard… Same fix as above (one forward fixes both)."

So it's one more router forward, on the same page I used in task 2. The guide's table matches ("on your router, forward UDP 41641 to this machine (this machine's firewall already allows it…)"), and it gives `tailscale ping <machine>` to check. 15d then shows all green: "✓ Tailscale: 3 machines direct", "✓ Network is friendly to direct connections", "✓ UDP 443 reached from the internet · pixel-8 connected from 198.51.100.20, 2 h ago." Clear. The "(tailbond works fine without them)" line told me this wasn't urgent, which I appreciated.

**Where I got stuck:**

- "The cause can also be on that machine's own network." The apple-tv is in my living room. I don't know whether that sentence means the forward won't help in my case. (low; 15d shows it did help)
- The guide says "see [setup tips](setup-tips.md)", but I don't have that file. The "How to fix" link on the relay item doesn't say where it goes. (low)
- The "Router fixes are on your home router's admin page, route ones in the Tailscale admin console" line shows up here too, with no route item listed. (low, same as task 2)
- The arrows contradict each other. The top card shows "↓ 1.1 MBPS FROM DEVICES / ↑ 12.0 MBPS TO DEVICES", but the pixel-8 row on the same page shows "↓ 12.0 Mbps ↑ 1.1 Mbps". The down arrow means "from devices" in one place and "to the device" in the other. (low)

## Task 6: Decide and set who may approve new devices

**What I'd do:** Devices → "Who can approve and add devices" (15/15d). It tells me who can approve right now: "alex@example.com (the Tailscale account homebox belongs to), from any device signed in to that Tailscale account, and any user or program on homebox itself". I'd add my partner's Tailscale login, which the card says I can find "in the Tailscale admin console under Users", into "Also allowed to approve (Tailscale logins, comma separated)" and click **Save**.

**Where I got stuck:**

- **Approvers or password: I have to choose one.** The card warns "any user or program on homebox itself… set a password if homebox runs things you don't trust". homebox is the family server and runs other software (media server, containers), and I can't judge whether I "trust" those. But a password "replaces the above": then the approver list stops applying and "everyone who approves, you included, needs it". So I can either have per-person approvers, or protect against local programs, not both. Nothing recommends which one a family server should use. I'd probably leave it as approvers only and feel uneasy about it. (medium)
- The slate7 issue from task 3 matters here too: the card says "A device you lend to someone while it's signed in to an approver's account can approve too". So the family member who has the router is effectively an approver until I sort out its sign-in, which I don't know how to do (see task 3).
- No screenshot shows the card after **Save**, so I can't confirm the login was accepted or see a current list of approvers. (low)

---

## Findings

| # | task | finding | severity |
|---|---|---|---|
| 1 | 2 (also 5) | The optional route item has an orange "!" and the "everything is set up" screen (15d) shows it approved as a green tick. Nothing recommends a choice for a home setup, so a non-expert approves it to clear the "!" and exposes bonded devices to the tailnet, which the guide calls a deliberate choice | medium |
| 2 | 3 | The family's router is "signed in to alex@example.com". The warning says to "sign its Tailscale in to their account (or tag it)", but neither the dashboard nor the guide says how to do that on a GL.iNet router, or what tagging is | medium |
| 3 | 4 | An approved phone "reaches what this server reaches… they allow it whatever they allow homebox, which may be more than its owner's own account gets". The only way to limit this is a tag plus a `grants` ACL in JSON, which is out of reach for a home admin | medium |
| 4 | 6 | Approvers and password are either/or ("replaces the above"). The card says to set a password "if homebox runs things you don't trust", but a family server runs other software and nothing says which option to pick | medium |
| 5 | 1 | The guide's LAN-block rule hard-codes `192.168.1.0/24` and doesn't say how to check that it matches my network | low |
| 6 | 1/2 | Header says "reachable at 192.168.1.2:443" while the setup card says devices "only work at home" | low |
| 7 | 1 | "Connected" switch and badge shown with "No devices yet". It's unclear what is connected and what the switch would turn off | low |
| 8 | 2/5 | "Router fixes are on your home router's admin page, route ones in the Tailscale admin console" is shown even when no router item (15c) or no route item (15) is listed | low |
| 9 | 2 | The setup step only asks for the UDP 443 forward. UDP 41641 comes up later, so it takes two visits to the router admin page | low |
| 10 | 3/6 | The long "Who can approve…" paragraph appears in the approval card and again in the Who-can-approve card. On the phone layout it pushes Approve about a screen down | low |
| 11 | 3 | After "slate7 can connect now. It connects by itself in a few seconds.", slate7 never appears on Connections. 15d shows it "Not Connected" with a phone icon although it's a router | low |
| 12 | 3/4 | Nothing tells me a request is waiting unless the dashboard is open; family has to tell me | low |
| 13 | 4 | In 14d the top totals stay at 1.1 / 12.0 Mbps after pixel-7a connects with its own traffic | low |
| 14 | 4/5 | Unexplained labels: "Min 31 Ms", "MBPS", "· wlb" | low |
| 15 | 5 | Down/up arrows mean opposite things in the top card ("↓ 1.1 from devices") and the device row ("↓ 12.0 Mbps") | low |
| 16 | 5 | The guide links "setup tips (setup-tips.md)", which I don't have. "The cause can also be on that machine's own network" leaves me unsure whether the forward helps for an Apple TV in my own house | low |
| 17 | 6 | No screen after saving approvers, so I can't confirm the login was accepted or see the current list | low |

Totals: 0 high / 4 medium / 13 low
