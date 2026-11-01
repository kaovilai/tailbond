# Simulated user test: GL.iNet routers (task script)

You are trying out **tailbond** on GL.iNet routers. What the person who
recommended it told you: "tailbond bonds all of a travel router's internet
connections (hotel Wi-Fi, your phone, a cable) into one fast, reliable
connection to your own home server, like Speedify but to your house. The
home server is on your Tailscale network, so everything behind the travel
router also reaches your Tailscale machines (like the NAS)."

Your setup:
- a **GL.iNet Slate 7 (GL-BE3600)** travel router, stock GL firmware 4.x,
  with GL's built-in Tailscale already turned on and logged in;
- at home, a home server on the same tailnet: either a **GL.iNet Flint 3
  (GL-BE9300)** router or a **Fedora** box, already running tailbond;
- a laptop that will connect to the Slate 7's Wi-Fi.

Tasks:
1. Install tailbond on the Slate 7 **without SSH**. You downloaded the
   package file to your laptop from the home server's tailbond page
   (screen 15, or the Flint 3's panel on 11–13).
2. Pair the Slate 7 with the home server.
3. At a hotel: use the hotel Wi-Fi (Slate 7 in Repeater mode) **and** your
   phone over USB tethering, and confirm both are being used.
4. Limit the phone to backup only, so it doesn't use your phone's data
   unless the hotel Wi-Fi fails.
5. From the laptop on the Slate 7's Wi-Fi, reach the NAS by its MagicDNS
   name. Find that name.
6. Turn tailbond off.
7. On the home side (Flint 3 admin panel, or the Fedora server's tailbond
   page), approve the new travel router.

For each task report:
- where you would click (refer to the screenshot file and the on-screen
  text);
- where you would get stuck or misunderstand something (quote the exact
  on-screen text);
- what you expected instead, based on your background;
- severity: **high** (can't finish the task / would do something wrong),
  **medium** (finishes with real confusion or extra effort), **low** (polish).

You only have the screenshots and the manifest. A screen that doesn't exist
means the task can't be done that way; say so. Don't invent features that
aren't shown. End with a table of all findings: # | task | finding | severity.
