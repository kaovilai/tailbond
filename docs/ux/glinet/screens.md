# Screenshot manifest (GL.iNet panel)

Each screen exists as `<name>-desktop.png` (1280x800 window, full page) and
`<name>-phone.png` (390x844 phone, full page). Screens 02–13 are the GL
admin panel, APPLICATIONS → tailbond. The data is a test fixture.

| Name | Situation |
|---|---|
| 01-not-installed-luci-upload | Installing: GL admin → SYSTEM → Advanced Settings → Go To LuCI, then LuCI → System → Software → Upload Package… with the downloaded file picked |
| 02-just-installed | Slate 7 right after installing, opening APPLICATIONS → tailbond for the first time (Tailscale on) |
| 03-found-server | A few seconds later |
| 03b-tailscale-off | Same, but the Slate 7's Tailscale is off |
| 03c-no-server-found | Same, Tailscale on but no home server answered |
| 04-waiting-approval | After clicking "Ask to connect" |
| 05-connected-two-wans | At the hotel: repeater on hotel Wi-Fi + phone on USB tethering, no cable |
| 06-one-wan-failed | Same, the phone connection has trouble |
| 07-save-data-standby | Same, phone set to backup only |
| 07b-set-limit | Same, after clicking "Change limit" on Tethering |
| 08-server-unreachable | The home server can't be reached |
| 09-turned-off | After switching tailbond off |
| 09b-unpair-confirm | After clicking "Unpair" |
| 10-not-running | The tailbond service isn't running |
| 11-server-pending-request | Flint 3 (home) admin panel, APPLICATIONS → tailbond, while the Slate 7 is waiting |
| 12-server-approved | Flint 3, after approving |
| 13-server-route-pending | Flint 3 freshly set up: Tailscale route not yet approved, no internet address set |
| 14-fedora-dashboard-request | Fedora home server: its tailbond page (http://homebox:8077) while the Slate 7 is waiting |
