# Screenshot manifest (GL.iNet panel)

Each screen exists as `<name>-desktop.png` (1280x800 window, full page) and
`<name>-phone.png` (390x844 phone, full page). Screens 02–13 and 16–18 are the GL
admin panel, VPN → tailbond. The data is a test fixture.

| Name | Situation |
|---|---|
| 01a-install-gl-advanced-settings | Installing, step 1: the Slate 7's GL admin → SYSTEM → Advanced Settings (the "Go To LuCI" button) |
| 01b-install-luci-upload | Installing, step 2: in LuCI → System → Software → Upload Package…, the downloaded file picked, LuCI's confirmation dialog |
| 02-just-installed | Slate 7 right after installing, opening VPN → tailbond for the first time (Tailscale on) |
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
| 11-server-pending-request | Flint 3 (home) admin panel, VPN → tailbond, while the Slate 7 is waiting |
| 12-server-approved | Flint 3, after approving |
| 13-server-route-pending | Flint 3 freshly set up: Tailscale route not yet approved, no internet address set |
| 14-fedora-dashboard-request | Fedora home server: its tailbond page (http://homebox:8077) while the Slate 7 is waiting |
| 15-fedora-devices-tab | Fedora home server's page, Devices tab, incl. setup health (desktop only) |
| 16-captive-gl-repeater | At a hotel: the repeater's Wi-Fi has a sign-in page (GL's repeater detected it); phone tethering in use |
| 16b-captive-gl-login-mode | Same, after starting GL's "Login Mode for Public Hotspots" on GL's Repeater page |
| 17-captive-own | Another hotel: the Ethernet cable has a sign-in page; hotel Wi-Fi in use, phone on backup |
| 17b-captive-own-signing-in | Same, after clicking "Open sign-in page" (the hotel's page opened in a new tab) |
| 18-server-health-relayed | Flint 3: some Tailscale machines connect through a relay |
| 18b-server-health-cgnat | Flint 3: the home internet line uses carrier-grade NAT |
