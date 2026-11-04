# Pair & Share: simulated user test

Two rounds with one persona ("Dana", 41, travels with a partner; knows GL's
admin page, not networking). Renders: `android/app/src/test/.../ux/ShareScreens.kt`
(`-Ptailbond.uxOut`), `scripts/ux/glinet/glinet-screens.sh share`; the iPhone
app was judged from its SwiftUI text (no simulator renders on Linux).

Tasks: (1) at a hotel, add your iPhone's and your partner's Android mobile
data to the travel router's bond; (2) make sure sharing stops when the
battery is low.

| Round | Task 1 | Task 2 |
|---|---|---|
| 1 | with hesitation | with hesitation |
| 2 | yes | yes |

Round 1 findings and what changed:

1. "Same Wi-Fi as your travel router" read as "the hotel's Wi-Fi" (major) →
   every screen now says the travel router's **own** Wi-Fi, not the hotel's.
2. iPhone limits applied only on the next start (major) → they apply at once.
3. The iPhone's keep-on-screen requirement came after the button (major) →
   at the top, and an alert on the first share.
4. Phones named inconsistently (model vs owner) → "Mobile data from
   <its name on the home server>" on Android, iPhone and the router.
5. Router said "Stopped", phone said "Paused" → "Paused", with when it comes
   back (charging, next month).
6. "Tunnel" jargon → "You don't need to turn tailbond on for this, and this
   phone's own apps aren't affected."
7. "Pair & Share" heading sounded like pairing again → "Lend your mobile data".
8. iPhone first run didn't say where the code comes from → "Ask whoever set
   up your home server to send you the link."
9. Battery icon on the selected chip looked like a warning → a check mark.

Left as is: no rename of a phone's share on the router (its name comes from
the home server; rename the device there); no shortcut on the Android home
screen; the phone itself can't tell that hotel Wi-Fi isolates it (the router
says so).
