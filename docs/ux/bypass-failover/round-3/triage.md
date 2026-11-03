# Round 3 triage (last review round)

Fresh reviewers, same inputs. Totals, high / medium / low:

| Persona | H | M | L |
|---|---|---|---|
| A traveller | 0 | 7 | 16 |
| B homelab | 1 | 15 | 11 |
| C family | 1 | 10 | 14 |
| D Speedify | 0 | 9 | 15 |

Round 3 was the last review round (cap: 3). The fixes below were made
after it and **have not been re-reviewed**.

## High

| # | Finding | Who | Decision |
|---|---|---|---|
| H1 | The kill switch that "always tailbond" and "nothing goes out unprotected" depend on isn't shown anywhere, as a control or a state | B (A medium) | **fix**: GL servers card shows the kill switch state with a switch (`client.kill_switch` in UCI through the panel's `/rpc`), and the wording names it |
| H2 | "Always use tailbond" follows a MAC; a new private Wi-Fi address silently puts the laptop under the website rules | C (A, B, D medium) | **fix**: device rules also follow the device's DHCP name (`host`): the agent adds every MAC leased under that name, re-checked every 30 s. The panel saves the name and says so on the row |

## Medium (merged)

| # | Finding | Decision |
|---|---|---|
| M1 | dnsmasq-full SSH fallback still risky (the reinstall needs a download after removal) | **fix**: download both packages first, install from the files; the notice puts GL's Plug-ins page first and says to do it on a working connection or leave it to whoever set the router up |
| M2 | Clash fix written for "whoever runs vps"; steps collapsed (router) or missing (phone), though the persona runs it | **fix**: "If you run vps yourself" steps shown expanded first (install.sh `--subnet` with a free range), the copy-a-message button second, on both |
| M3 | Phone suggests any tailnet machine as a server | **fix**: suggestions are probed through the tunnel (`GET /api/hello`) and only tailbond servers are offered |
| M4 | Phone "Banking" quick pick adds nothing | **fix**: it focuses the field with a hint; nothing to pre-fill without the bank's name |
| M5 | Phone falls back to its own internet during a switch / when no server answers unless "Block connections without VPN" is on | **partial**: Android's always-on lockdown is a system setting the app can't turn on; recommended with a button. With it on, hotel sign-in pages need it off for a moment, which is said |
| M6 | "use a device rule" as the fix for devices with their own secure DNS is the wrong advice | **fix**: turn off the device's own secure DNS for this Wi-Fi, or add the site's addresses as an address rule |
| M7 | "you picked it" when someone else did | **fix**: "picked by hand" |
| M8 | Notification during failover has only "Disconnect" | **fix**: "Still protected, nothing to do" in the text |
| M9 | "Stay on vps" undefined | **fix**: "until you pick homebox or vps stops working" |
| M10 | Fixture inconsistencies (work-laptop "connected now" vs "last seen 1 day ago"; router and phone with the same tunnel address) | **fix** (fixtures) |
| M11 | Which DNS resolves website rules; IPv6/DNS for always-tailbond devices; rule order on the list | **partial**: one line in the card (bypassed websites are looked up through the connection they use; everything else, including IPv6, follows the same rules); rule order is documented in glinet.md, not on screen |
| M12 | UDP 500/4500 (Wi-Fi calling preset) also catches IPsec work VPNs | **fix** wording on the preset: it also lets IPsec VPNs (some work VPNs) skip tailbond; use a device rule for the work laptop |
| M13 | Phone website status: no counters / matched names | **reject**: Android exposes no per-route counters to a VPN app; the covered names are listed |
| M14 | Router has no alert channel for switches | **reject**: GL's panel has no notification mechanism for add-ons; the history and the top card cover it |
| M15 | Both servers down: phone stops and doesn't retry by itself | **partial**: documented; automatic retry with backoff is future work on Android |
| M16 | Subnet route approval: which prefix, which server is primary in Tailscale | **partial**: the prefix is shown; Tailscale's own primary-router choice is documented in vps.md |
| M17 | `:8077` approval page on a public VPS | **reject**: the agent binds 127.0.0.1 and the Tailscale address only (vps.md); already stated there |
