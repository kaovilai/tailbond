# Screens: Mac menu bar app and Report a problem

Menu bar window (CI job macos, artifact `menubar-screens`, from
`macos/TailbondMenu` SnapshotTests; light, `-dark` for dark):

| File | State |
|---|---|
| menu-01-connected.png | connected to homebox: Wi-Fi, iPhone over USB, a nearby phone's share on standby; updates Right away, up to date |
| menu-02-netbird.png | connected; NetBird on, its traffic Direct; work names on NetBird's DNS; a warning; Ethernet not working; updates Tell me |
| menu-03-crash-update.png | after a crash; a security update waiting for a quiet moment |
| menu-04-not-paired.png | a fresh install, not paired |
| menu-05-not-running.png | the agent doesn't answer |

Report a problem (CI job report-ux, artifact `report-screens-web`;
`scripts/ux/report-screens.sh`): report-01-crash-banner, report-02-preview,
report-03-ready (desktop and phone), and the GL.iNet panel's
report-11/12/13.

Text: [docs/macos.md, "Menu bar"](../../macos.md#menu-bar) and
[docs/reporting.md](../../reporting.md) "What you see".
