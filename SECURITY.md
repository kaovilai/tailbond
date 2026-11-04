# Security policy

## Reporting a vulnerability

Please **don't** open a public issue for a security problem. Report it
privately through GitHub's
[private vulnerability reporting](https://github.com/kaovilai/tailbond/security/advisories/new)
(the repository's **Security** tab → **Report a vulnerability**). Only the
maintainers see it.

Include what you found, how to reproduce it, the tailbond version and the
platform (GL.iNet router, Fedora, VPS, Mac, Android, iPhone). Take secrets
out first: pairing links, keys, passwords. tailbond's **Report a problem**
(every app, [docs/reporting.md](docs/reporting.md)) builds a redacted report
you can paste in: use its **Copy full report** or **Save report file** rather
than **Open GitHub issue**, which makes a public issue.

What to expect: an acknowledgement within a week, a fix in a release as soon
as it's ready (a security release reaches "Right away" installs at once,
[docs/updates.md](docs/updates.md)), and credit in the advisory if you want
it.

## Supported versions

The latest release. tailbond updates itself; fixes aren't backported.

## Scope

tailbondd, mqvpn as patched here, the apps, the router and server
packages, the installers, and the release process (signing keys, the
update check). Bugs in Tailscale, GL.iNet firmware or upstream mqvpn belong
to those projects; tell us too if tailbond makes them worse.
