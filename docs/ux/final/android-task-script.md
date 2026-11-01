# Simulated user test: Android, whole journey (task script)

You are trying out **tailbond**, an Android app. What the person who set it
up told you: "tailbond bonds your phone's Wi-Fi and 5G together to our home
server, homebox. homebox is on our Tailscale network, so with tailbond on you
still reach all the Tailscale machines (like the NAS). Android only allows
one VPN app at a time, so you use tailbond instead of the Tailscale app. To
get it: with Tailscale on, type **homebox/** into your phone's browser (with the slash at the end)."

Today the phone only has the official Tailscale app installed and connected.
tailbond isn't in the Play Store.

Tasks:
1. Get the tailbond app onto the phone.
2. Add the phone to homebox (pair), from the phone.
3. Connect.
4. Find and copy the NAS's MagicDNS name or Tailscale IP (the NAS is called `nas`).
5. Tell whether Wi-Fi and 5G are both being used right now.
6. Make it save mobile data.
7. At a hotel, the phone joins a Wi-Fi that shows a sign-in page. Get that
   Wi-Fi working and back into the bond.
8. Turn tailbond off and go back to the Tailscale app.

For each task report:
- where you would tap (refer to the screenshot file and the on-screen text);
- where you would get stuck or misunderstand something (quote the exact
  on-screen text);
- what you expected instead, based on how the official Tailscale Android app
  (and Android itself) behaves;
- severity: **high** (can't finish the task / would do something wrong),
  **medium** (finishes with real confusion or extra effort), **low** (polish).

You only have the screenshots and the manifest. A screen that doesn't exist
means the task can't be done that way; say so. Don't invent features that
aren't shown. End with a table of all findings: # | task | finding | severity,
and a line `Totals: H high / M medium / L low`.
