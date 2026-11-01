# Persona A: daily Tailscale Android user, not a networking person

Inputs: task-script.md, screens.md, screenshots in out/ux/after-glinet/ (light theme; dark variants skipped).

My starting point: the Tailscale app is on and Connected. In its machine list I can see `homebox`. Nobody gave me a link or a web address, only the sentence about what tailbond does.

## Task 1: Install tailbond and pair it with the home server, from the phone

**Installing.** The only download I can find is on the home server's web page (web-01-add-this-phone): "1 Get the tailbond app (skip if it's installed)" with the button "Download tailbond for Android". The text below it explains the Android warning about installing apps from outside the Play Store well ("this app comes from your own home server").

Where I get stuck: none of the screens tell me how to reach that page before I have the app. screens.md gives its address as `http://<server>.<tailnet>.ts.net:8077`. I could probably guess the `homebox` part from the Tailscale app, but I would never guess `:8077`. The first time I see the address is in the app itself, in 02b ("http://homebox:8077"), and that is after installing. With the Tailscale app I would open the Play Store and search. Nothing shows tailbond is in the Play Store, so I'd have to text whoever set it up and ask where to get the app. **From the phone alone, with only what I was told, I can't install it.**

**Pairing (assuming someone sent me the page link, or the APK).**

Route A, from the app (01-onboarding, then 02-pairing):
- 01: the title "Use tailbond instead of Tailscale on this phone" and the three points are clear. The footer, "Next you'll type your home server's name (the one in the Tailscale app) and ask it to connect", tells me exactly what comes next. I tap "Get started".
- 02: under "Ask your home server" I type `homebox`, the name I can see in my Tailscale machine list, and tap "Ask to connect". This feels natural to me. The hint "Not sure of the name? It's the machine that runs tailbond, in the Tailscale app's list of machines" helps.
- 02b: "Waiting for approval on homebox… Ask whoever runs homebox to open its tailbond page (http://homebox:8077 on a computer, or a Flint 3's admin panel → VPN → tailbond) and tap Approve next to "pixel-8". Keep this screen open; tailbond continues by itself once approved."
  - If someone else runs homebox, this works: I message them and wait. Then 02e shows "Add this phone to homebox?" and I tap "Add phone". Done.
  - If **I** run homebox, I'm stuck on this screen. It says "on a computer" and "Keep this screen open", so I don't think I can approve from this phone. The only way I see is to leave the app, type http://homebox:8077 into Chrome (Tailscale is still on, so that should load) and look for "Approve". But the page I get on the phone (web-01) shows "Add this phone", not an approve list with "pixel-8" on it. I can't tell whether tapping "+ Add this phone" there approves the waiting request or starts a second, separate pairing. In the end I would probably succeed through web-02 ("Open in tailbond", then 03 "Add phone"), but with real doubt about whether I had set things up twice.
  - "a Flint 3's admin panel": I don't know what a Flint 3 is. I'd ignore it, but it makes me feel this message is meant for someone else.

Route B, from the web page (web-01, then web-02, then 03): "Two steps, then tailbond takes over from Tailscale on this phone. No code to scan." Step 2 already fills in the name "pixel-8". After I tap "+ Add this phone", web-02 says "Opening tailbond… In the app, tap Add phone, then turn tailbond on." The app shows 03, "Add this phone to homebox?", and I tap "Add phone". This route is clear and needs no approval, which makes the app's "someone approves this phone" message even more confusing.

Smaller things on 02:
- In the "Or use the home server's page" card, the "Open home server page" button is greyed out, and nothing says why. The card's text, "Came here from your home server's page? Go back to it in your browser", doesn't apply to me, so I skip the card.
- In 02c ("Can't find homebox") and 02d ("homebox said no"), the name box is empty again and "Ask again" is greyed out. I'd have to type the name again before "Ask again" works, which is odd because the error message itself names homebox.
- 02e/03 confirm: "your internet traffic goes out from home (you can change that in Settings)". In Tailscale that is like turning on an exit node, and it is off unless I choose it. Here it is on by default. The dialog does say so, so I would only notice it as a surprise.
- 02-pairing-paste-error: "That isn't a complete tailbond link. Copy the whole link starting with tailbond://pair" is clear.

Expected (from Tailscale): get the app from the Play Store, sign in, and the device shows up. Here, installing and approving both depend on someone who knows the web page.

## Task 2: Connect

The screen after pairing is not shown; I assume it looks like 12-home-disconnected ("Disconnected · Stopped. Turn on to reach homebox and your Tailscale machines, or switch back to Tailscale below."). I tap the switch, just like Tailscale's big toggle.
- With Tailscale on, 04-handoff-other-vpn appears: "Switch from Tailscale to tailbond? … Turning on tailbond turns off Tailscale … You still reach your Tailscale machines, like your NAS, through homebox. Next, Android asks you to allow tailbond's connection request. Tap OK." I tap "Turn on tailbond". This is clear and reassuring.
- 05-handoff-first-permission is the same idea. 11 shows "Connecting… Reaching homebox…", then 10 shows "Connected · Through homebox · Wi-Fi + mobile data". No problems.
- When things fail, 13 ("Can't reach your home server … forward UDP port 443 … show them this message") and 14 ("Can't confirm this is your home server … Don't pair this phone again unless they ask you to") both tell me what to do. "UDP port 443" means nothing to me, but "show them this message" covers it.

## Task 3: Find and copy the NAS's MagicDNS name or Tailscale IP

In 10, under Machines, I can see "nas · nas.tail1234.ts.net · 100.101.2.2 · linux" right away. I type `nas` into "Search machines" (18) and tap the copy icon. The message says "Copied 100.101.2.2", so the copy icon copied the IP, not the name. If I wanted the name, I tap the row (19): "Tap an address to copy it", with MagicDNS name, Short name, Tailscale IPv4 and IPv6, each with its own copy button. This matches Tailscale's machine details. Done easily.
- Small things: the copy icon on the list row doesn't say which address it will copy (I only found out from the message). And "This device · pixel-8 (this phone) · 10.77.0.2" is not a 100.x address like my phone has in Tailscale. The line below explains it ("Your Tailscale machines can reach this phone at this address, through homebox"), but it looks odd at first.

## Task 4: Are Wi-Fi and 5G both being used right now?

In 10, the status card says "Connected · Through homebox · Wi-Fi + mobile data". Below it, "Connections · Using both Wi-Fi and mobile data", and the tiles "Wi-Fi ● In use ↓38.4 ↑4.1 Mbps" and "Mobile data ● In use ↓22.7 ↑2.9 Mbps". The answer is yes, and obvious. 15 shows the opposite case just as clearly ("Using Wi-Fi only", "Mobile data · Not working right now", red dot). 20-connections adds graphs, not needed. No problems.

## Task 5: Save cellular data

On the home screen, the "Save mobile data" switch says "Use mobile data only when Wi-Fi stops working". I tap it. 16 then shows "Using Wi-Fi · saving mobile data (backup only)", "Mobile data · Not in use · saving data", and "Mode: Save mobile data". 17 shows it taking over when Wi-Fi drops ("Wi-Fi isn't working · using mobile data", "In use · taking over"). Very clear.
- In Connections (20/21) there is also a "Mobile data limit" ("1.23 GB of 5.00 GB used this month"), which is a nice extra. One risk: the "Mobile data · Use for tailbond" switch looks like another way to save data. If I turned it off, I would lose the backup when Wi-Fi drops. Its subtitle, "Turning this off doesn't turn off the phone's mobile data", doesn't warn about that. I'd still use the home-screen switch first.

## Task 6: Turn tailbond off and go back to the Tailscale app

"Switch back to Tailscale" is a full-width button right under the status card on every main screen (10, 11, 13, 15, 16, 17). After tapping it, 12b says: "Disconnected · Stopped. Tailscale is in charge on this phone." A card adds: "Tailscale is in charge on this phone · tailbond stays off until you turn it on again. In the Tailscale app, turn on its switch if it says Disconnected." I tap "Open Tailscale", then turn on Tailscale's switch. Clear. Settings (30) repeats this and warns about Always-on and "Block connections without VPN", which is useful if I ever turn that on.
- Turning Tailscale back on is a manual step, but the app says so in advance. Fine.

## Other observations

- 16c (hotel Wi-Fi): the instructions "Tap the Wi-Fi network (it says "Sign in to network")" and the "Sign in to Wi-Fi" button appear together, and I'm not sure which one to use. I'd tap the button. Low.
- 30 Settings: "Through your home server … like using an exit node" vs "Only Tailscale machines" is explained in Tailscale terms I understand. Good.
- 31 "Apps that skip tailbond" works like Tailscale's app split tunneling, and the text warns that "They can't reach your Tailscale machines". Good.

## Findings

| # | task | finding | severity |
|---|---|---|---|
| 1 | 1 | Nothing tells me where to get the app. The only download is "Download tailbond for Android" on the home server page (web-01), and I can't find that page's address (":8077") before installing. The first time I see "http://homebox:8077" is inside the app (02b). From the phone alone, with what I was told, I can't install it. | high |
| 2 | 1 | If I run homebox myself, 02b ("Ask whoever runs homebox to open its tailbond page (http://homebox:8077 on a computer …) and tap Approve … Keep this screen open") doesn't tell me I can approve from this phone. When I open the page on the phone, I get "Add this phone" (web-01), not an Approve list, so I can't tell if I'm approving the request or pairing a second time. | medium |
| 3 | 1 | "a Flint 3's admin panel → VPN → tailbond" in the waiting message is unfamiliar jargon and makes the message seem meant for someone else | low |
| 4 | 1 | The "Open home server page" button on 02 is greyed out with no reason given | low |
| 5 | 1 | After "Can't find homebox" / "homebox said no" (02c/02d), the name box is empty and "Ask again" is disabled until I retype the name | low |
| 6 | 1 | Internet traffic going through home is on by default ("your internet traffic goes out from home"). In Tailscale, an exit node is opt-in. The dialog does say so. | low |
| 7 | 3 | The copy icon on the machine row doesn't say what it copies. It gave "Copied 100.101.2.2" when I might have wanted the MagicDNS name. | low |
| 8 | 3 | This phone shows "10.77.0.2" instead of a 100.x Tailscale address. It's explained, but surprising. | low |
| 9 | 5 | The "Mobile data · Use for tailbond" switch looks like a way to save data, but turning it off would remove the Wi-Fi backup, and nothing warns about that | low |
| 10 | — | 16c gives two sign-in instructions at once ("Tap the Wi-Fi network (it says "Sign in to network")" and the "Sign in to Wi-Fi" button) | low |
| 11 | 2 | 13: "forward UDP port 443" is jargon, though "show them this message" makes up for it | low |

Totals: high 1, medium 1, low 9.
