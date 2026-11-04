# Mac menu bar app and Report a problem: persona task script

tailbond bonds several internet connections into one tunnel to your home
server ("homebox"). On a Mac it runs in the background; **tailbond.app**
puts an icon in the menu bar that opens a small window. You see the
product only as the screenshots in [screens.md](screens.md) and the text
excerpts listed there; you can't click, but say where you'd click and what
you'd expect.

Setting: your MacBook is paired with homebox. At work you also use
**NetBird**, your company's VPN, on the same Mac.

Tasks:

1. **Check that tailbond is bonding, and which connections it uses.**
   You plugged your iPhone in over USB at a café.
2. **Make sure your work VPN keeps working.** NetBird is on. Find out
   whether your work traffic goes through tailbond or not, and make it go
   directly (not through tailbond).
3. **Updates.** Find out which version you have, whether an update is
   waiting, and set tailbond to install updates by itself right away.
4. **Something went wrong: report it.** tailbond crashed this morning.
   Report it to the developers without publishing anything private (the
   report screens are the dashboard's; the menu sends you there).

For each task: what you'd do step by step, where you'd hesitate or get it
wrong, and anything missing, confusing or wrong. Rate each point **high**
(you'd fail or give up), **medium** (you'd get stuck or need help), or
**low** (polish). End with a one-line verdict per task.

Personas:

- **A, traveller:** uses Tailscale daily, travels a lot, fine with apps,
  doesn't know networking words (MTU, split DNS, SOCKS).
- **N, NetBird at work:** a developer whose company requires NetBird; cares
  that work traffic and internal names keep working, and that nothing
  private ends up in a public bug report.
