# Round 0 – Persona (c): less-technical family member

Who I am: I'm a retired teacher in my 60s. My son set up Tailscale on my phone. I use it for two things: flipping the big switch on, and finding "nas" in the list of machines so I can copy its address into my photo app. I don't know what IP, DNS, TLS or tunnels are. Red error text makes me nervous, and I won't touch a setting I can't explain.

I judged only from the screenshots in `out/ux/before/`. The light and dark versions show the same text.

---

## Task 1 – Install tailbond and pair it with the home server, from the phone

**Tap path**
1. `01-onboarding-pairing`: I read "Pair with your server" and tap **Scan QR code**. That part I understand ("scan the square code").
2. I have no square code to scan. The instruction says: *"On your tailbond server's dashboard, open Clients → Add client, then scan its QR code or paste the tailbond://pair link."* I don't know what a "server's dashboard" is or how to open one on this phone. None of the screenshots shows such a page.
3. Suppose my son texts me a link and I tap **Paste**. In `02-pairing-paste-error` the box turns red and says *"pairing data is not base64url"*. **Import** stays grey.
4. Suppose I open a link he sent from another app instead. `03-deeplink-confirm` asks *"Pair with this server?"* and lists *"Name: pixel"*, *"Server: home.example.net:443"*, *"TLS name: homebox.tail1234.ts.net"*, *"Mode: speed"*, *"DNS: 10.77.0.1"*, *"Tailnet routes: 100.64.0.0/10, fd7a:115c:a1e0::/48"*, *"Certificate: verified by the system trust store"*. Then **Cancel** / **Pair**.

**Where I get stuck**
- I can't finish this on my own from the phone. The only way in needs a "server's dashboard" that I can't find and that isn't shown anywhere. I would have to phone my son.
- *"pairing data is not base64url"* in red frightens me. I don't know if I broke something, and it doesn't say what to do next (for example, "ask whoever sent this for a new link").
- The confirmation box is a wall of numbers. I can't tell whether "Pair" is safe. "Name: pixel" means nothing to me; my phone isn't called that as far as I know. I'd probably tap Pair only because it's the bright word, without understanding what I agreed to.
- Nothing warns me that this will push my Tailscale app aside ("only one VPN at a time"). Nothing shows what happens after I tap Pair either: no Android permission pop-up, no "Done".

**What I expected (Tailscale-like)**
Tailscale's sign-in is one big friendly button, and the only screens are "Sign in" and then "Connected". I expected something like "Ask your family admin for a pairing code. Scan it here." Errors should be plain: "This code didn't work. Ask for a new one." The confirm box should say something like "Connect this phone to *Home*?" and hide the technical details behind a "Details" link.

**Severity:** high (I can't pair alone; dashboard needed). Medium for the error wording and the jargon in the confirm box.

---

## Task 2 – Connect

**Tap path**
`12-dashboard-disconnected`: I tap the big switch next to **Disconnected**. `11-dashboard-connecting` shows **Connecting…**, then `10-dashboard-connected` shows **Connected**.

**Where I get stuck**
- The switch itself is fine. It looks like Tailscale's switch.
- Under "Connected" it says *"pixel · homebox.tail1234.ts.net · 10.77.0.2"*, and lower down *"60 s↓ 1.84 GB ↑ 212.0 MB · RTT 27 ms · loss 0.3% · 12:35"*. I can't tell whether "loss 0.3%" is bad.
- When it fails I see `13-error-unreachable` **Error** *"Connection to the server failed"*, or `14-error-cert` **Error** *"TLS handshake failed"*. The switch flips back off and there is no "Try again" button and no explanation. I'd have no idea what "TLS handshake" means and would worry that something is broken or unsafe. I would just keep flipping the switch.
- Even when "Disconnected", the Wi-Fi and Cellular cards still show their switches **on** with *"idle"*. So is something still running? I can't tell.

**Expected**
Tailscale shows one clear line ("Connected" / "Not connected") and a plain sentence when something goes wrong. I'd want something like "Can't reach your home server. Check your internet, or ask the person who set it up," with a **Try again** button.

**Severity:** medium for the jargon on the connected screen; high for the "TLS handshake failed" error (I can't recover and wouldn't know who to tell what); medium for "Connection to the server failed".

---

## Task 3 – Find and copy the address of "nas"

**Tap path**
I look on **Dashboard** (`10-dashboard-connected-full`) and **Settings** (`30-settings`). No list of machines appears anywhere. There's nothing called "nas", no search and no copy button.

**Where I get stuck**
- I can't do this task at all. In Tailscale this is the main thing I do: open the app, find "nas", press on it, copy. Here no screen lists my machines.
- The only addresses on screen are mine and the server's (*"10.77.0.2"*, *"homebox.tail1234.ts.net"*). I might copy one of those by mistake and put the wrong address into my photo app.
- Settings shows *"Tailnet + home LAN only"* with *"100.64.0.0/10, fd7a:115c:a1e0::/48, 10.77.0.0/24"*, which looks like addresses but isn't anything I can use.

**Expected**
A "Machines" or "Devices" list like Tailscale's, with names. Pressing on "nas" should show its address with a **Copy** button.

**Severity:** high. I can't do it, and I might copy the wrong thing.

---

## Task 4 – Are Wi-Fi and 5G both being used right now?

**Tap path**
`10-dashboard-connected-full`: I scroll down. **Wi-Fi** shows *"active"* and **Cellular** shows *"active"*, each with a moving line. I'd answer "yes, both", once I work out that "Cellular" means my 5G.

**Where I get stuck**
- It never says "5G" or "mobile data", only "Cellular". I'd probably manage, but I'd hesitate.
- In `15-one-path-failed` the top still says **Connected**, but Cellular says *"failed"* with *"0 bps"* and *"RTT 0 ms"*. Is that bad? Should I do something? The word "failed" worries me, and nothing tells me "that's OK, Wi-Fi is carrying on".
- In `16-failover-standby` Cellular says *"standby"*, and in `17-failover-takeover` Wi-Fi says *"failed"* while Cellular says *"active"*. I can't tell whether "standby" means it's in use or not.
- Every card says *"loss: tunnel 0.3%"*, even the failed ones. I don't know what that means.
- Each card has a switch right beside the status word. I'd be scared that touching it turns off my phone's actual Wi-Fi.
- Ethernet says *"unavailable"* and I don't have one of those plugs on a phone. It also says *"Priority Primary"*, the same as Wi-Fi.

**Expected**
One plain sentence at the top, such as "Using Wi-Fi and mobile data together" or "Using Wi-Fi only (mobile data is a backup)". Hide the Ethernet card on a phone.

**Severity:** medium overall ("failed" wording and the scary switches); low for "Cellular" vs "5G" and for Ethernet.

---

## Task 5 – Make it save cellular data

**Tap path (my best guess)**
1. On the Dashboard I look at the row **Speed | Streamin | Redundan | Failover** (the last letters of two words are cut off). Their captions say: Speed *"Aggregate bandwidth across every link"*, Streaming (`20-mode-picker`) *"Calls and streams survive link trouble"*, Failover (`16`) *"One primary link, others only on failure"*. None of them says "save data". I might guess Failover from "others only on failure", but I'm not sure.
2. At the bottom of the Dashboard is **Cellular data this month – 1.23 GB (no cap)** with a **Reset** link. I wouldn't touch "Reset" because I'm afraid it deletes something.
3. In **Settings** (`30-settings`) under **Cellular data** there's *"Monthly cap (MB, empty = none)"* containing **5000**, plus *"Counts tunnel traffic carried over cellular. At the cap, cellular stops carrying paths until next month."*
4. The other option would be switching off the **Cellular** card's switch on the Dashboard. I wouldn't do that, because I don't know whether it turns off my phone's mobile data entirely.

**Where I get stuck**
- No button anywhere says "Save mobile data" or "Use Wi-Fi first". I'd have to guess that "Failover" means saving data.
- The Dashboard says *"(no cap)"* but Settings shows *"5000"*. Which is true? It makes me distrust both.
- "MB", "tunnel traffic" and "carrying paths" mean nothing to me. I don't know whether 5000 is a lot.
- Two half-words, "Streamin" and "Redundan", look broken.
- Settings → **Failover** has *"Experimental: backup_fec scheduler – Failover mode only. Uses mqvpn's FEC-protected backup scheduler."* That's exactly the kind of setting I'd be afraid to touch.

**Expected**
One clearly named choice, like "Save mobile data (use only when Wi-Fi drops)". Tailscale doesn't have this feature, so I'd need the words to do the explaining.

**Severity:** high (I can't confidently do it, and I might switch off Cellular or tap Reset by mistake); medium for the cap contradiction; low for the cut-off tab labels.

---

## Task 6 – Turn tailbond off and go back to the Tailscale app

**Tap path**
On `10-dashboard-connected` I tap the big switch so it shows **Disconnected** (`12`). Then I go to my home screen, open Tailscale and flip its switch on.

**Where I get stuck**
- Turning it off is easy, like Tailscale.
- Nothing tells me I now need to open Tailscale again. There's no "Switch back to Tailscale" link or hint.
- Settings → **VPN** says *"To keep tailbond on all the time, enable Always-on VPN (and optionally Block connections without VPN) in system settings."* If my son turned that on, I suspect Tailscale won't come back and I'd be stuck. The app gives no warning about that.
- Settings → **Pairing** shows **Pair again** and a red **Forget**. Red words make me think I might delete something by accident while looking for an "off" option.
- After switching off, the screen still shows Wi-Fi and Cellular switches *on* with *"idle"* (`12`), so I'm not sure it really stopped.

**Expected**
When switched off, a clear "tailbond is off". Ideally also a note like "To use Tailscale again, open the Tailscale app", or a button that opens it.

**Severity:** medium (I could finish, but I'm unsure and at risk of the Always-on trap); low for the switches still showing on.

---

## Summary table

| # | issue | severity | screen |
|---|-------|----------|--------|
| 1 | Pairing needs a "server's dashboard → Clients → Add client" I can't reach from the phone; I can't pair alone | high | 01 |
| 2 | Paste error *"pairing data is not base64url"* in red with no next step | medium | 02 |
| 3 | Confirm dialog full of jargon (TLS name, DNS, Tailnet routes, trust store); "Name: pixel" unexplained | medium | 03 |
| 4 | No warning that tailbond replaces Tailscale (one VPN at a time) | medium | 01, 03 |
| 5 | Status line and stats strip jargon (*"10.77.0.2"*, *"RTT 27 ms · loss 0.3%"*) | low | 10 |
| 6 | *"TLS handshake failed"*: no plain words, no "Try again", no advice | high | 14 |
| 7 | *"Connection to the server failed"*: no retry or next step | medium | 13 |
| 8 | No machines list; can't find or copy "nas" at all | high | 10, 30 (all) |
| 9 | Only my own and the server's addresses are visible, so I might copy the wrong address | medium | 10 |
| 10 | Cellular *"failed"* while the header says "Connected", with no reassurance; "standby" unclear | medium | 15, 16, 17 |
| 11 | "Cellular" not called mobile data or 5G; Ethernet card shown on a phone, also "Primary" | low | 10 |
| 12 | Per-link switches look like they turn off the phone's real Wi-Fi or mobile data | medium | 10 |
| 13 | No obvious "save mobile data" option; must guess that "Failover" means it | high | 10, 16, 20 |
| 14 | Dashboard *"1.23 GB (no cap)"* contradicts Settings cap *"5000"* | medium | 10, 30 |
| 15 | Cap field in MB with *"carrying paths"* wording | low | 30 |
| 16 | **Reset** link next to the data counter, which I'm afraid to tap | low | 10 |
| 17 | Mode tabs cut off: *"Streamin"*, *"Redundan"* | low | 10, 20 |
| 18 | Settings has scary options (*"Experimental: backup_fec scheduler… mqvpn's FEC"*, CIDR routes, red *"Forget"*) | medium | 30 |
| 19 | No hint for switching back to Tailscale; Always-on VPN advice could lock Tailscale out | medium | 12, 30 |
| 20 | When off, the link switches still show on with "idle", so it's unclear it really stopped | low | 12, 13, 14 |
