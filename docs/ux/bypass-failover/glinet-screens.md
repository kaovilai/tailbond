# Screenshot manifest: bypass rules and more servers (GL.iNet panel)

Slate 7 admin panel, VPN → tailbond, client role, at a hotel (Repeater +
Tethering). Test fixture data (`scripts/ux/glinet/fixtures.mjs`, `bfStates` /
`bfActions`). Render with `scripts/ux/glinet/glinet-screens.sh bypass-failover`
→ `out/ux/bypass-failover/glinet/<name>-desktop.png` (1280x800 window; pages
full length, dialogs at window size) and `<name>-phone.png` (390x844) for the
three marked "phone too". The failover screens follow one story on one
clock: homebox stops answering at minute 2 (→ vps), comes back at minute 23,
someone picks vps at 38 and homebox again at 40.5. Every client screen has the bypass card (the four
rules below unless the row says otherwise).

| Name | Situation |
|---|---|
| bypass-empty | No bypass rules yet: what the card is for, "Add a rule" and quick picks |
| bypass-rules | Four rules: bank website (internet directly, 37 connections), work laptop (always tailbond), Fire TV stick (internet directly), Wi-Fi calling (two ports shown as one row). Phone too |
| bypass-website-rules-wait | Website rules saved, but this router's DNS server can't do website rules yet (no dnsmasq-full): notice (what to install and where) + "Waiting" on each website rule |
| bypass-add-website-presets | After clicking "Add a rule": a website, quick picks at the top |
| bypass-add-streaming-preset | Same, after picking "Streaming location": an editable list of streaming websites |
| bypass-add-device-picker | Same, "A device" chosen: devices on this router (online first), alex-iphone picked; defaults to "Always use tailbond". Phone too |
| bypass-add-wifi-calling | From the empty card, after clicking the "Wi-Fi calling" quick pick (UDP 4500 and 500, internet directly) |
| bypass-wifi-calling-added | Same, after clicking Add: the card now has the Wi-Fi calling row |
| servers-single-home | One server (homebox): the familiar "Home server" card, with "Add a backup server" |
| add-server-open | After clicking "Add a backup server": vps found on the tailnet, or type its name, or paste a link |
| add-server-waiting-approval | After "Ask to add": waiting for someone to approve on vps |
| add-server-subnet-clash | vps approved, but it uses the same tunnel addresses as homebox: what to change on vps |
| add-server-paste-clash | Same clash, after pasting a pairing link instead |
| servers-home-active-vps-standby | Two servers: homebox in use, vps on standby, "When to switch", the last 3 of 4 recent switches, the kill switch (on) |
| servers-kill-switch-off | Same, with the kill switch turned off: the switch and the wording that depends on it |
| servers-recent-switches-all | Same, after clicking "Show all (4)" under Recent switches |
| failover-using-vps-home-unreachable | homebox stopped answering; tailbond switched to vps. Phone too |
| failback-countdown | homebox answers again: back to homebox in 1:20 |
| failover-you-picked-vps | Someone clicked "Use now" on vps (picked by hand; automatic switching paused) |
| back-on-homebox-since | The page was open while homebox came back: back on homebox, with the time |
