# Accessibility

Every web page tailbond ships is checked in CI with
[axe-core](https://github.com/dequelabs/axe-core) (the engine behind most
browser accessibility checkers) in headless Chromium, in each state that
matters: dialogs open, tabs switched, errors shown, light and dark, phone and
desktop widths. A **serious** or **critical** finding fails the build;
**moderate** and **minor** ones are only reported.

## What runs

[`scripts/ux/a11y.sh`](../scripts/ux/a11y.sh) (CI job *Report a problem
(dashboard + GL panel, Chromium)*, a required check) starts a throwaway
server-role and client-role `tailbondd` like the screenshot scripts do,
then [`scripts/ux/a11y.mjs`](../scripts/ux/a11y.mjs) opens each page and
runs axe-core (pinned in `a11y.sh`) plus four checks axe doesn't make:

| Check | Fails when |
|---|---|
| `tb-pointer-only` | something looks clickable (`cursor: pointer`) but a keyboard can't reach it: not a link, button, field, label or `tabindex="0"`, nor inside one |
| `tb-no-key-handler` | a control that isn't a native one (`role="checkbox"`, `radio`, `switch`, `button`… on a `div` or `span`) gets keyboard focus but has no key handler, so Space and Enter do nothing |
| `tb-motion` | an endless animation keeps running with **reduce motion** on (every page is opened that way) |
| `tb-modal-focus` | with a dialog open (`.tb-dialog`, or any `aria-modal` dialog), Tab and Shift+Tab stay inside it: an escaped focus strands a keyboard user behind a dialog they can see |

A self-test page with one of each known problem runs first; if the run stops
seeing them, it fails instead of passing everything.

| Surface | Pages and states |
|---|---|
| Site (`site/index.html`) | light and dark, desktop and phone |
| Phone page ("Get tailbond", `landing.html`) | Android, Mac, iPhone and Linux tabs at phone and desktop widths, light and dark, "no app yet" |
| Dashboard (`agent/web`) | Connections, Devices, Setup, Tailscale, Updates & backup, Logs; Report a problem (empty and ready to send); Add this phone (before and after the tap); the pairing-code dialog; sign-in and "home server's own page" dialogs; a waiting pair request; setup health; the error banner; phone hub (found, in use, lost); sharing (setup, on, unplugged, a hub listing a computer's share); update, backup, restore and "what's inside" dialogs; phone width and dark mode for the main ones |
| GL.iNet panel (`packaging/openwrt/gl-ui`) | every state and dialog in `scripts/ux/glinet/fixtures.mjs` and `ub-fixtures.mjs` (setup, connected, failures, captive portals, server side, Pair & Share, bypass rules, failover, updates and backups) plus Report a problem against a real agent; phone width for a subset |
| LuCI view (`packaging/openwrt/luci`) | desktop and phone |

The GL panel and the LuCI view run inside someone else's page (GL's admin
app, LuCI's theme), so only tailbond's part of the page is checked, and the
`region` rule (content outside landmarks) is off there: the host page owns
the landmarks. Each rule turned off for a page says why in `a11y.mjs`.
Findings accepted on purpose go in
[`scripts/ux/a11y-allow.json`](../scripts/ux/a11y-allow.json), each with a
reason and a link; it's empty.

The report lands in `out/ux/a11y/` (`report.html` to read, `report.json` for
tools) and is uploaded as the `a11y-report` artifact; the log has one line
per surface.

## Run it locally

Needs Go, Node (for `npm`) and Podman; Chromium runs in the
`mcr.microsoft.com/playwright` image, and Playwright, Vue 2 and axe-core go
into `~/.cache/tailbond-ux` on first use.

```sh
scripts/ux/a11y.sh                          # everything → out/ux/a11y/report.html
TB_A11Y_ONLY='^glinet/' scripts/ux/a11y.sh  # only pages matching "surface/name"
TB_A11Y_SHOTS=1 scripts/ux/a11y.sh          # plus a screenshot of each page → out/ux/a11y/shots/
```

`TB_A11Y_JOBS` sets how many pages run at a time (default 4); `TB_UX_DEBUG=1`
prints every page and its JavaScript errors.

## What it can't see

Automated checks catch only part of what makes a page usable. Not covered:

- **Screen readers:** whether VoiceOver, TalkBack or NVDA read the pages in a
  sensible order and announce changes (the live connection numbers, toasts).
  axe checks names, roles and labels, not how they sound.
- **Real devices and zoom:** only Chromium, at two widths. Not Safari or
  Firefox, 200% text zoom, Windows high contrast or a phone's large-text
  setting.
- **Keyboard flow:** focus order and traps beyond what the checks above
  catch, and whether every task can be finished without a mouse.
- **Plain language and visuals:** whether instructions are clear, icons
  make sense, or a chart's meaning comes across without color.
- **The apps:** the Android, Android TV, iOS and macOS apps are native, not
  web; axe doesn't apply. Their screens are rendered in CI for review but not
  checked for accessibility yet ([#35](https://github.com/kaovilai/tailbond/issues/35)).
