# Bypass + failover: persona task script

You are reviewing screenshots of **tailbond**, an add-on that bonds several
internet connections (hotel Wi-Fi, a phone, cellular) into one fast, stable
tunnel to the user's own home server, with their Tailscale network reachable
through it. It runs on a GL.iNet travel router (GL's admin panel → VPN →
tailbond) and as an Android app.

Two new features:
- **Bypass tailbond** (router): some traffic skips the tunnel. You can pick
  a website, a device on the router's Wi-Fi/LAN, an address, or a port. It
  can use the internet directly, use only one of the router's connections,
  or always use tailbond. On Android: websites that skip the tunnel
  (Android 13+).
- **Several servers with failover** (router and app): a home server plus a
  backup, e.g. a rented VPS. If home stops answering, the router or phone
  switches to the backup after about 15 s. It switches back once home has
  been fine for 2 minutes.

You only see the PNGs listed in the manifests, not the product. Judge only
what a real user would see.

## Tasks

Do each task as your persona, using only what the screens show:

1. **Make your bank's website skip the tunnel**:
   - your bank blocks VPN addresses, so `mybank.com` must use the hotel's internet directly;
   - router panel first, then the phone app.
2. **Make your work laptop always use tailbond**: its traffic must never
   leave through the hotel's network, even for sites you bypass.
3. **Add a VPS as a backup server**: you rented a small VPS and installed
   tailbond on it. Add it to the router (and the phone) as a backup to
   `homebox`, and understand what happens if it clashes with homebox.
4. **Notice the switch, and that it came back**:
   - your home lost power while you travel;
   - can you tell from the router panel and the phone that you're now on the VPS, and why?
   - later, can you tell it's going back to homebox, and when it did?

## Report format

For each task, give:
- **Done?** yes / partly / no, and the screens you used.
- **Findings**, each with:
  - severity: **high** (can't complete, or would do something harmful or unsafe), **medium** (confusing, needs guessing, or risks a mistake), or **low** (polish);
  - the screen file name;
  - what's wrong, in one or two sentences;
  - a concrete suggestion.

End with an overall count: high / medium / low. Be concrete and skeptical.
Don't praise.
