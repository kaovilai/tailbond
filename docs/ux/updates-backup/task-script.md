# Updates + backup: persona task script

You are reviewing screenshots of **tailbond**, an add-on that bonds several
internet connections (hotel Wi-Fi, a phone, cellular) into one fast, stable
tunnel to the user's own home server, with their Tailscale network reachable
through it. It runs on:
- a GL.iNet travel router, in GL's admin panel under VPN → tailbond;
- a home server: a Fedora PC, a rented VPS, or a GL.iNet Flint 3 router, each with its own web dashboard;
- an Android app.

Two new features:
- **Updates**: each device checks for a new tailbond release about once a day
  and says "Update available".
  - The VPS and the travel router update with one click ("Update now"), or by
    themselves if you choose Automatic. A new version that doesn't start is
    undone automatically.
  - The Fedora home server gets updates from the system's package updates
    (dnf), like any other app installed there.
  - The phone app shows the update and downloads it; an app store
    (Obtainium) can install updates automatically.
  - You choose Tell me / Automatic / Off.
- **Backup and restore**: an encrypted backup file with all of tailbond's settings, made with a passphrase. It can be restored on the same device or moved to a new one.

You only see the PNGs listed in the manifests, not the product. Judge only
what a real user would see.

## Tasks

Do each task as your persona, using only what the screens show:

1. **Make sure everything stays up to date**: home server, travel router and
   phone. Can you tell what version each runs, how each one gets updated
   (and whether it happens by itself), and whether anything is out of date?
2. **Move the home server to a new machine using a backup**: make a backup of
   the old home server, restore it on the new one, and understand what happens
   to the travel router and the phone (do they need anything?).
3. **The router got a GL firmware update. Is tailbond still there?** What does
   the router panel tell you, and did you have to do anything?

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
