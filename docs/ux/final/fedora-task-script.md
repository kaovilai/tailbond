# Simulated user test: the Fedora home server (task script)

You run the home end of **tailbond** on a Fedora 44 machine called
`homebox`, which is already on your Tailscale network (Tailscale installed
and logged in). What you were told: "tailbond bonds a phone's or travel
router's Wi-Fi, 5G and cable into one connection to your own home server,
like Speedify but to your house. Your Tailscale machines stay reachable
through it."

You have the install guide (`fedora-server.md`) and the file
`tailbond-server-0.1.0-1.fc44.x86_64.rpm`.

Tasks:
1. Install tailbond on homebox following the guide, and open its dashboard.
2. Right after installing (screen 15c): what's left to set up, and how?
3. A family member's GL.iNet travel router is asking to connect. Approve it.
4. Get tailbond onto a family member's Android phone (they have Tailscale on
   it), and get that phone connected.
5. Read the setup health and fix what it reports, including the machine that
   connects "through a relay".
6. Decide and set who may approve new devices.

For each task report:
- where you would click (refer to the screenshot file, the guide's section,
  and the on-screen text);
- where you would get stuck or misunderstand something (quote the exact
  text);
- what you expected instead, based on your background;
- severity: **high** (can't finish the task / would do something wrong),
  **medium** (finishes with real confusion or extra effort), **low** (polish).

You only have the guide, the screenshots and the manifest. A screen that
doesn't exist means the task can't be done that way; say so. Don't invent
features that aren't shown. End with a table of all findings:
# | task | finding | severity, and a line `Totals: H high / M medium / L low`.
