# Round 3: Persona A, a GL Tailscale traveller

**Who I am:** I travel a lot and I'm not a network engineer. I've used GL.iNet travel routers for years. GL's own Tailscale page is the model in my head: APPLICATIONS → Tailscale, one Enable toggle, Apply, and a bind-account link. New GL features should feel like that: one switch and very little to set up.

**Screens I looked at:** all phone shots 01a–14, plus desktop 05, 11, 13 and 15.

**Overall:** This round felt close to "GL-like" on the Slate 7. After pairing, the page is a big toggle at the top, a status line that says what is in use, and per-connection "Always / Backup only / Off" buttons. I finished every task. The one real snag was on the home side: if my home server is a Flint 3, the Flint's own page never gives me the router package.

---

## Task 1: Install tailbond on the Slate 7 without SSH

**Where I'd click**
- I already have the file from the Fedora page (15-desktop). The card "Add a GL.iNet travel router" has a "Download the router package" button and three numbered steps. That's easy to follow.
- 01a-phone: SYSTEM / Advanced Settings → "Go To LuCI". The hint "Use the same admin password to log in" removes my usual worry about which password LuCI wants.
- 01b-phone: LuCI's "Manually install package" dialog. It shows `tailbond_0.1.0-r42_aarch64_cortex-a53_neon-vfpv4.ipk (6.1 MB)` and I click **Install**.

**Where I hesitated**
- If my home server were the **Flint 3** (11, 12, 13), the "Add a GL.iNet travel router" card has no download button. Step 1 says: "Get the router package on your computer: the tailbond file ending in **_neon-vfpv4.ipk** (from the tailbond releases page, or from a Fedora home server's tailbond page)." I have no Fedora box in that setup, and "the tailbond releases page" isn't a link. I'd have to search the web for it and then pick one file out of a list by matching a cryptic ending. The Fedora page gives me a button for this; the Flint page should too.
- 01b: the LuCI dialog has a checkbox, "Allow overwriting conflicting package files". Nothing tells me whether to tick it. I'd leave it unticked and hope.
- From habit I'd first look under APPLICATIONS → **Plug-ins**, which is where GL installs add-ons. Going through LuCI feels like a back door, but the steps are spelled out exactly, so I'd follow them.

**What I expected:** a download button wherever the instructions are shown, ideally a GL Plug-ins install.

## Task 2: Pair the Slate 7 with the home server

**Where I'd click**
- 02: "Looking for your home server on your tailnet…" with a spinner. Good, I don't have to type anything.
- 03: "Found your home server: homebox", then the big **Ask to connect** button. This is the one-tap experience I want.
- 04: "Waiting for approval on homebox… On homebox, tap **Approve** next to **slate7**. This router connects by itself once approved; you can close this page." This is clear, and it tells me where to approve (the http://homebox:8077 link, or the Flint's APPLICATIONS → tailbond).
- 03b: if GL Tailscale were off, "Turn on Tailscale first" with **Open Tailscale settings** sends me to the GL page I already know. 03c, "No home server found… Checked 4 online machines", also tells me what to check.

**Where I hesitated:** nowhere. "Other ways to connect / Paste a link" is clearly marked for "No Tailscale on this router?", so I'd skip it.

## Task 3: Hotel Wi-Fi plus phone tethering, both in use

**Where I'd click:** 05: the top card reads "Connected to homebox · Using Repeater (Wi-Fi) + Tethering (phone)". Under Internet connections, Repeater and Tethering are both green "In use" with their own speeds. Confirmed in one glance, on phone and on desktop.

**Where I hesitated**
- Tethering defaults to **Always**, so the phone's data gets used straight away, with no question when I plug it in. There is a hint under Speed or reliability: "Saving your phone's data? Set Tethering or Cellular to 'Backup only'". It isn't next to the Tethering card, though, so I might not see it until I've used a GB.
- 06: when the phone link breaks, the card says "Not working right now" with "It's connected, but can't reach homebox. tailbond keeps trying; the other connections carry your traffic." That's reassuring and plain.

## Task 4: Phone as backup only

**Where I'd click:** 05/07, Tethering card: "Use it" → **Backup only**. 07 then shows "Backup · saving data" and "Used only if your other connections stop working. Until then it isn't used at all, so it uses no data." The top card changes to "Tethering (phone) on standby". Very clear.

In 07b, "Change limit" opens a "5 GB a month" field with Save. The explanation ("At the limit, tailbond stops using this connection until the count restarts on the 1st of each month. 0 = no limit.") is fine.

**Where I hesitated:** "Most reliable: Sends everything over every connection. Uses the most data." I can't tell whether choosing that would also wake up my Backup-only phone. I'd leave it on Fastest, so it doesn't block me.

## Task 5: Reach the NAS by its MagicDNS name

**Where I'd click:** 05, the "Tailnet machines" card: **nas**, `nas.tail1234.ts.net`, 100.101.2.2, with a **Copy name** button. The Home server card also says "Tailnet names: Work on every device connected to this router while tailbond is on. Use the full name, like nas.tail1234.ts.net."

**Where I hesitated:** with Tailscale on my laptop I'm used to typing just `nas`. The page hints that the short name won't work from behind the Slate ("Use the full name"). It's a small surprise, but the copy button makes it painless.

## Task 6: Turn tailbond off

**Where I'd click:** the toggle in the top card (05). 09 confirms it: "tailbond is off. Devices on this router use the internet directly, without your home server. Turn on to bond your connections again." It works exactly like GL's Tailscale Enable switch, without the Apply step.

**Where I hesitated**
- "Unpair" sits right next to "Ask to connect again". It's red, and 09b asks for confirmation ("Unpair from homebox? … (it will need to approve)"), so I wouldn't mix it up with off.
- On 08/09, the Tailnet names line says "like nas.your-tailnet.ts.net", but elsewhere it shows my real "tail1234" name. It's a little inconsistent but harmless.
- 08 has a good safety net: "Meanwhile, devices on this router have no internet… To use this internet directly for now, turn tailbond off", with a **Turn tailbond off** button right there.

## Task 7: Approve the new travel router at home

**Where I'd click**
- Flint 3, 11-desktop/phone: "Waiting for your approval · 1 new · slate7 wants to connect · GL-BE3600 · … asked just now", then **Approve**. 12 shows slate7 under Devices, "Connected just now · 2 connections bonded". That was easy.
- Fedora, 14-phone/15-desktop: the same "Waiting for your approval" card at the top, then **Approve**.

**Where I hesitated**
- 14-phone: when I open the home server's page on my phone just to approve, a big "Add this phone" block with an Android download sits right under the approval card. I didn't come here to add my phone, but the Approve card is first, so I'd ignore it.
- 13 (Flint freshly set up): "Step 1: allow tailbond devices on your Tailscale network… open homebox, choose Edit route settings, tick the one route listed (10.77.0.0/24)". I've never needed the Tailscale admin console on GL; the bind link did everything. The step is explicit, and it says "Devices can already reach your machines without it", so it doesn't stop me. Step 2 (Dynamic DNS) uses a GL feature I know.

---

## Findings

| # | task | finding | severity |
|---|---|---|---|
| 1 | 1 | Flint 3 home-server page (11–13) has no download button for the router package. It says "from the tailbond releases page" with no link, and I must match a file "ending in _neon-vfpv4.ipk". Fedora's page (15) has a button. | medium |
| 2 | 1 | LuCI dialog (01b) "Allow overwriting conflicting package files" checkbox: nothing says whether to tick it. | low |
| 3 | 1 | A GL user's first instinct is APPLICATIONS → Plug-ins. Install via LuCI works but feels like a back door. The instructions are exact, so it isn't blocking. | low |
| 4 | 3 | Tethering defaults to "Always", so phone data is used at once. The "Saving your phone's data?" hint sits under Speed or reliability, not on the Tethering card. | low |
| 5 | 4 | Unclear whether "Most reliable" ("Sends everything over every connection") also uses a Backup-only phone. | low |
| 6 | 5 | The short MagicDNS name (`nas`) apparently doesn't work behind the Slate ("Use the full name"), unlike on a normal Tailscale device. "Copy name" makes up for it. | low |
| 7 | 6 | Tailnet names example switches to "nas.your-tailnet.ts.net" on 08/09 vs the real "nas.tail1234.ts.net" on 05. | low |
| 8 | 7 | Fedora phone page (14): the big "Add this phone" block under the approval card distracts when I only came to approve. | low |
| 9 | 7 | Flint setup step 1 (13) sends a GL user into the Tailscale admin console ("Edit route settings… 10.77.0.0/24"). It's unfamiliar, but the page says it's optional for reaching machines. | low |
| 10 | 3 | Phone status card on 05 shows no uptime ("Up 2 h 13 min" appears only on desktop). | low |

**Totals: high 0, medium 1, low 9.**
