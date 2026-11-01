# Round 8 — Fedora persona F2 (home admin)

**Persona:** I run the family's home server, a spare PC with Fedora on it. I can
install things with `dnf` and paste commands from a guide. I set up Tailscale by
following its website. I know roughly what a router admin page is, but firewalls,
NAT and port forwarding aren't my thing. Whatever I set up has to work for my
family without me standing next to them.

**Materials used:** `task-script.md`, `screens.md`, `fedora-server.md`, and the
screenshots in `glinet/` and `web/`. Nothing else.

---

## Task 1 — Install tailbond on homebox and open the dashboard

**What I'd do:** guide section 1, "Install". I'd paste
`sudo dnf install ./tailbond-server-*.rpm` and `sudo systemctl enable --now tailbond`,
then `systemctl status tailbond` to check it. To open the dashboard, the guide
says "browse to <http://127.0.0.1:8077/>" on the server, or `http://<server>:8077`
from another device on Tailscale. That gets me to screen `15c-fedora-fresh-install-desktop`.
This went smoothly. The commands are copy-paste and the dashboard opens.

**Where I hesitated:**

- **Header text (15c):** "Home server · reachable at 192.168.1.2:443". The word
  "reachable" reads like good news, as if everything is fine. Only the orange card
  below explains that this is "an address inside your home, so they only work at
  home". I expected the header to say something like "not reachable from outside
  yet". **Low.**
- **"Connected" toggle (15c):** the top right of a server with "No devices yet"
  shows "Connected" with an on/off switch. Connected to what? I can't tell what
  turning it off would do (stop the server? cut off the family?), so I'd be afraid
  to touch it. **Low.**
- **Guide section 1 is heavy on firewall detail.** It says "UDP 443 (the `tailbond`
  service) in the zone of the interface that has the default route, the one your
  router's port forward arrives on". I haven't made a port forward yet. That's
  section 2, so this reads like I already missed a step. The package table also
  says the Android app is included "(when the package was built with it)", and
  nothing tells me whether my RPM has it. (The phone page later shows "Download
  tailbond for Android (27 MB)", so mine does.) I skimmed past both. **Low.**

## Task 2 — Right after installing (15c): what's left, and how?

**What I see on 15c:** two things are left.

1. The orange card **"Setup step: tell devices where home is"**:
   - "Forward UDP port 443 to this machine (192.168.1.2). The page is usually
     called Port forwarding, Virtual server or NAT. Give this machine a fixed
     address there too (DHCP reservation)".
   - "Find your home's public address … Type it here:" with a box and **Save**.

   Guide section 2 has the same steps. Even though I'm not a networking person, I
   can follow this. It tells me the menu names to look for on my router, and it
   already filled in my server's LAN IP. Nice.
2. The **Setup health** card: "Route 10.77.0.0/24 isn't approved yet. Approve it
   once in the Tailscale admin console (Machines → this machine → Edit route
   settings)." I know the Tailscale admin console, so I can do that.

**Where I'd get stuck or be unsure:**

- **I can't tell if the port forward worked.** After I click Save, nothing on the
  page lets me test the forward. The guide's health table says "UDP 443 not
  confirmed from outside yet (?) … nothing has connected from away yet … none: the
  first device connecting from away confirms it". So I only find out whether I
  set up my router right when a family member is away from home and tries to
  connect. That's the worst moment to find a mistake. The Troubleshooting section
  suggests checking "from the phone on 5G", but the dashboard doesn't say that.
  I expected a "Test from outside" button, or at least "turn off Wi-Fi on your
  phone and try" right in the card. **Medium.**
- **The carrier-grade NAT hint clashes with Tailscale addresses.** The small print
  says "If the router's own WAN address starts with 100.64–100.127 … your provider
  shares one address between customers". All my Tailscale addresses also start
  with 100. (the Tailscale tab shows "100.101.1.1"). I'd wonder whether that
  counts and whether I'm in trouble. I'd only work it out by rereading "router's
  own WAN address". **Low.**
- **The route item is labelled as a speed problem.** The card heading is "Setup
  health: 1 thing to improve for best speed". But per the guide, the route is
  about "Tailscale machines can't reach bonded devices". It has nothing to do with
  speed. I'd approve it anyway, but I couldn't have told you why I needed it.
  **Low.**
- **Nothing says this PC's own firewall is already handled for port 443.** The
  card asks me to forward UDP 443 on the router and says nothing about Fedora's
  firewall. Later the relay fix explicitly says "allow 41641/udp in this machine's
  firewall", so I'd wonder whether I also needed to open 443 here. (The guide says
  the package does it. The card doesn't.) **Low.**

## Task 3 — Approve the family's GL.iNet travel router (slate7)

**What I'd do:** screen `14-fedora-dashboard-request-desktop` (and `-phone`). At
the top there's a box: "Waiting for your approval … slate7 wants to connect ·
Travel router on your tailnet · signed in as alex@example.com · GL-BE3600 ·
100.101.9.9", with **Decline** and **Approve**. I click **Approve**. Screen
`14b` shows a toast: "slate7 can connect now. It connects by itself in a few
seconds." Done. That's easy, and it works the same on my phone (14-phone).

**Where I'd get stuck or be unsure:**

- **The approval box scares me and points to a field that isn't there.** It says:
  "Who can approve: anyone on your Tailscale network who can reach homebox (it's a
  tagged machine, so it has no single owner). Set a password below to limit
  that."
  - I never tagged homebox. The Tailscale tab (15b) shows "TAILNET
    alex@example.com", meaning my account.
  - Reading "anyone on your Tailscale network" can approve, I'd panic a bit about
    the kids or guests approving things.
  - There is no password box "below" on the Connections tab I'm on (14, 14b). It's
    on the Devices tab.

  I'd still click Approve, but I'd be worried and would go hunting for the
  password field. **Medium.** (See Task 6 for the bigger problem.)
- **After approving, I don't see slate7 anywhere.** 14b's Connections list still
  shows only pixel-8. The toast says it connects "in a few seconds", but there's
  no "slate7 — approved, waiting for it to connect" entry. On the later Devices
  list (15d), slate7 shows a phone icon and "offline", which doesn't look like a
  router. I'd be unsure whether it worked. **Low.**

## Task 4 — Get tailbond onto a family member's Android phone and connect it

**What I'd do:**

1. Devices tab, **"Add an Android phone"** card (15c / 15): "On the phone, with
   Tailscale on: point the camera at this code, or type homebox/ into its browser
   (with the slash at the end …)". "For family: send them this link" with
   `http://homebox.tail1234.ts.net/` and a copy button. I'd text them that link.
2. On their phone, the page `web-00-phone-page` titled "Get tailbond on this phone":
   - **Step 1:** "Download tailbond for Android (27 MB)", with pictures of each
     Android warning ("File might be harmful", "Install unknown apps", "Install").
     These pictures are great for my family. Otherwise they'd call me the moment
     they saw "File might be harmful".
   - **Step 2:** a button, "Open tailbond and ask homebox".
   - **Step 3:** "tap Add phone. Then tap the big switch".
3. Back on my dashboard, `14c-fedora-phone-request-desktop` shows "pixel-7a wants
   to connect · Phone on your tailnet · signed in as sam@example.com · Pixel 7a".
   I click **Approve**.

**Where I'd get stuck or be unsure:**

- **Does the family member need to be on *my* Tailscale?** My family member "has
  Tailscale on it", but nothing tells me whether their Tailscale has to be logged
  into my network, or whether I have to share homebox with them first. The request
  in 14c shows "signed in as sam@example.com", a different account, so some kind
  of sharing must be involved. The texts don't agree on who can open the page:
  - The phone page on a computer (web-00c) says "Anyone on your Tailscale account
    can open it".
  - The dashboard says "People you share homebox with, and others on your Tailscale
    network, can ask to connect".

  If sam's phone is on sam's own tailnet, typing `homebox/` or opening my
  `homebox.tail1234.ts.net` link just fails, and no screen explains why or what to
  do. I expected a line like "Their phone must be on your tailnet, or share homebox
  with them in the Tailscale admin console". **Medium.**
- **I can't confirm the phone connected.** No screen shows what happens after I
  approve pixel-7a: no toast, no device entry, no screen of the app connected. The
  later Devices list (15d) shows pixel-8, ipad and slate7, but **no pixel-7a**.
  From what I have, I can't tell whether the phone actually got connected, so I
  can't say the task is done. **Medium.**
- **I don't get told when someone is waiting.** The phone page tells my family
  "Whoever runs homebox taps Approve on homebox's page; you'll see it in the app
  right away." Nothing tells *me* that a request came in. They'll be sitting
  there until they phone me and I open the dashboard. I expected some kind of
  notification. **Low.**
- **Their whole internet goes through my house.** This is only in small grey text
  at the bottom of step 3: "by default this phone's internet traffic goes out from
  homebox, like using it as an exit node". For a family member, that means all
  their browsing uses my home connection. I'd want to know that before I approve,
  and it's easy to miss. **Low.**

## Task 5 — Read setup health and fix it (including the relay)

**What I see (15):** "Setup health: 2 things to improve for best speed (tailbond
works meanwhile)":

- "Tailscale: 2 machines direct, 1 through a relay (slower). Through a relay:
  apple-tv … To check: tailscale ping apple-tv ("via DERP" means relayed)."
- "This network makes direct Tailscale connections hard … ("hard NAT")."

Both items say: "On the router in front of this machine, forward UDP port 41641
to it, and allow 41641/udp in this machine's firewall." The guide's health table
(section 3) gives the command:
`sudo firewall-cmd --permanent --zone=<wan zone> --add-port=41641/udp && sudo firewall-cmd --reload`.

**What I'd do:** add a second port forward on my router (UDP 41641 to
192.168.1.2), like I did for 443, then run the firewall command, then run
`tailscale ping apple-tv`. Screen 15d shows the result I'm hoping for:
"everything is set up for best speed".

**Where I'd get stuck or be unsure:**

- **What is `<wan zone>`?** This is exactly where I'd get stuck. I paste commands
  from guides, and I don't know what a firewalld "zone" is, let alone which one is
  "wan". Section 1 mentions `sudo firewall-cmd --get-active-zones`, but nothing
  connects that command to this placeholder. I'd probably paste `<wan zone>`
  literally and get an error, or guess. It also confuses me that the package
  opened UDP 443 for me automatically ("in the zone of the interface that has the
  default route"), but for 41641 I have to do it myself. I expected either a
  ready-to-paste command with the real zone name filled in, or tailbond doing it
  for me. **Medium.**
- **The same fix is shown twice and never says one fix covers both.** Both items
  give identical instructions, word for word. The guide says "same fix as the
  relay item (one forward fixes both)", but the dashboard doesn't. I'd wonder
  whether I need two forwards. **Low.**
- **The card contradicts itself about where the fixes are.** The footer says "Each
  fix is on your home router or in the Tailscale admin console". Right above
  that, the fix includes "allow 41641/udp in this machine's firewall", which is
  neither the router nor the admin console. **Low.**
- **The relay may not be fixable from my side.** Only the guide says "A relay can
  also come from the other machine's network, which you may not control". The
  dashboard just says forward 41641. If apple-tv still shows "via DERP" after I've
  done everything, I'd think I did it wrong. **Low.**
- **The "fixed" screen shows the method I was warned about.** 15d shows "Network
  is friendly to direct connections · port mapping: UPnP". The fix text on 15
  warned that UPnP "lets any device on your network open ports". So the
  "everything is set up" screen celebrates the option I was told was risky. Was
  that OK or not? **Low.**

## Task 6 — Decide and set who may approve new devices

**What I see (15, Devices tab):** two different answers on the same page.

- Top approval box: "Who can approve: **anyone on your Tailscale network who can
  reach homebox (it's a tagged machine, so it has no single owner)**. Set a
  password below to limit that."
- The "Who can approve and add devices" card further down: "Who can approve:
  **alex@example.com (the Tailscale account homebox belongs to)**, from any of
  their devices. People you share homebox with, and others on your Tailscale
  network, can ask to connect but can't approve."

**Where I'd get stuck:**

- **I can't tell which one is true, so I can't decide.** If the first is right,
  anyone in my house who's on Tailscale can approve devices, and I need a password
  now. If the second is right, only I can approve, and I should leave it alone.
  The Tailscale tab ("TAILNET alex@example.com") suggests homebox is mine and not
  tagged, but the scarier message is the one at the top of the page. As a careful
  home admin I'd probably set a password "just in case". That changes how the
  dashboard is protected without me needing to (see the next point). **High.**
- **What a password actually does isn't explained on the card.** The card says "To
  let someone else approve too, set a password and give it to them." I read that
  as "add a second approver". The guide explains it's really a shared password for
  "the whole dashboard API": "anyone with the password, from any tailnet device".
  Then I'd have to log in too, my curl commands would need a login first, and
  whoever I give it to can also remove devices, not just approve them. I expected
  either a way to name a person (like sam@example.com) or a clear warning that
  this replaces the account check. **Medium.**
- **Jargon:** "People you share homebox with" assumes I know Tailscale's machine
  sharing feature. I've heard of it but never used it. **Low.**

**My decision, if the page agreed with itself:** leave the password empty, so only
my account (alex@example.com) approves. Family members ask to connect from their
devices, and I approve from my phone or laptop.

---

## Findings

| # | Task | Finding | Severity |
|---|---|---|---|
| 1 | 1 | Header "reachable at 192.168.1.2:443" reads as success, but it's a home-only address | low |
| 2 | 1 | "Connected" toggle on a server with "No devices yet"; unclear what turning it off does | low |
| 3 | 1 | Guide §1 talks about "the one your router's port forward arrives on" before §2 explains forwarding; "(when the package was built with it)" leaves me unsure the app is included | low |
| 4 | 2 | No way to test the UDP 443 forward from home: "the first device connecting from away confirms it" | medium |
| 5 | 2 | CGNAT hint "starts with 100.64–100.127" clashes with my Tailscale 100.x addresses | low |
| 6 | 2 | Route 10.77.0.0/24 listed under "improve for best speed", but it isn't about speed | low |
| 7 | 2 | Setup step doesn't say this PC's firewall for 443 is already handled (the relay fix asks me to open 41641 here) | low |
| 8 | 3 | Approval box: "it's a tagged machine, so it has no single owner … Set a password below", but homebox isn't tagged and there's no password field below on the Connections tab | medium |
| 9 | 3 | After Approve, slate7 doesn't appear in Connections; later shown with a phone icon and "offline" | low |
| 10 | 4 | Nothing says the family member's phone must be on my tailnet or have homebox shared; "Anyone on your Tailscale account" vs "People you share homebox with" | medium |
| 11 | 4 | No screen confirms pixel-7a connected after approval; 15d's device list has no pixel-7a | medium |
| 12 | 4 | No notification to me when a request arrives ("Whoever runs homebox taps Approve") | low |
| 13 | 4 | "this phone's internet traffic goes out from homebox, like using it as an exit node" is only in small grey text | low |
| 14 | 5 | Firewall fix needs `--zone=<wan zone>`; I don't know which zone that is, and the package opened 443 itself but not 41641 | medium |
| 15 | 5 | The two health items repeat an identical fix; the dashboard doesn't say "one forward fixes both" | low |
| 16 | 5 | "Each fix is on your home router or in the Tailscale admin console" contradicts "allow 41641/udp in this machine's firewall" | low |
| 17 | 5 | Dashboard doesn't say the relay may come from apple-tv's own network (only the guide does) | low |
| 18 | 5 | "All good" screen shows "port mapping: UPnP", the option the fix text warned about | low |
| 19 | 6 | Same page says both "anyone on your Tailscale network … tagged machine" and "alex@example.com (the Tailscale account homebox belongs to)"; I can't decide who approves | high |
| 20 | 6 | "set a password and give it to them" reads as adding an approver, but it's a shared password for the whole dashboard, and I'd need it too | medium |
| 21 | 6 | "People you share homebox with" assumes I know Tailscale machine sharing | low |

Totals: 1 high / 6 medium / 14 low
