# Round 2: persona A, traveller (MacBook, iPhone)

## Task 1: is it bonding, which connections

Steps: click the menu bar icon (menu-01-connected). "Connected, to homebox, 2 of 3 links in use". Under Connections I see Wi-Fi, iPhone (USB) and Sam's phone on standby.

- **Medium**: the on/off switch looks grey, like a disabled control, in both light and dark mode. Only the small green "On" tells me it's on. Use the normal macOS accent colour for the "on" state.
- **Low**: "(en0)", "(en7)", "(sh5a1b2c)" mean nothing to me. Hide them or move them to a tooltip.
- **Low**: "Sam's phone: standby". I don't know why it's sitting idle or when it would be used. Add a short reason ("used if the others drop").
- **Low**: the header says "links" and the section says "Connections". Pick one word.

Verdict: done in seconds. The grey switch made me doubt it for a moment.

## Task 2: work VPN goes direct

Steps: menu-02-netbird, "Next to other VPNs". The first line says NetBird's traffic goes Direct (not through tailbond). Below it, Work VPN traffic is already set to Direct.

- **Medium**: an orange warning full of numbers ("1280 bytes… wrapping (80)… 1180"). I can't tell whether something is broken or whether I need to act. Say it plainly: "Bonded won't work on this connection, so work traffic goes Direct. Nothing to do."
- **Medium**: "Bonded" looks clickable even though the warning says it isn't possible. Grey it out and show the reason next to it.
- **Low**: "Now: Direct" repeats what the selected button already shows, and it adds one more line to a long block of text.
- **Low**: "100.92.0.0/16, 10.20.0.0/16" and "DNS" are words I don't know. Write "work sites and names" and put the details in a tooltip.

Verdict: I got the answer from the first line. The warning made me worry for no reason.

## Task 3: updates

Steps: menu-01 says "tailbond 0.2.0 is up to date". In menu-02 the policy is "Tell me", so I'd open that menu and pick "Right away".

- **Medium**: menu-03 shows that "Right away" actually means "when no call or download is running, within 30 minutes". The doc also lists "Automatic (night)". I can't tell the two automatic options apart from their names. Rename to "Soon (when idle)" / "Overnight", or describe each option inside the menu.
- **Medium**: menu-03 says "Update available: tailbond 0.2.1" but no longer shows which version I have now. The task says it's a security update, and the screen doesn't say so. Show both versions and mark security updates.
- **Low**: "Check now" never says when it last checked.

Verdict: findable. The policy names don't tell me when an update will actually install.

## Task 4: report the crash

Steps: menu-03 has the orange banner, so I click "Report a problem…". The dashboard opens (report-01) and I click "Report a problem" in its red banner. Then the form (report-02, report-03): type what happened, tick the box, Open GitHub issue.

- **Medium**: menu-03's button doesn't take me straight to the form. I land on a dashboard with a second, scarier red "Setup health: 1 thing to fix" banner and have to find the report link again. Open the form directly.
- **Medium**: the notice is one dense paragraph ("telemetry", "tailnet", "100.x", "[brackets]"). The key fact, that the issue will be public, is in the middle. Put "This will be public on GitHub" in bold first and turn the rest into bullets.
- **Medium**: report-02 shows "Open GitHub issue" greyed out with no reason next to it. The checkbox is below the fold. Put the "what's missing" line beside the button.
- **Medium**: the checkbox asks me to confirm the report has nothing private in it, but the logs ("homebox.[tailnet].ts.net", "/tmp/tmp.KL4o…", "token=[redacted] user [email]") are in a small scrolling box I can't judge. Highlight anything I should check, for example Wi-Fi and device names like "Sam's phone".
- **Low**: the times are written two ways ("Nov 4, 2026 at 6:51AM" vs "11/4/2026, 7:50:25 AM"), and "6:51AM" has no space.
- **Low**: the form doesn't mention that 0.2.1 is waiting. Suggest "an update is ready, it may fix this".
- **Low**: on the phone the four buttons wrap unevenly (report-02/03 phone).

Verdict: I'd finish it, but I'd tick the box without really being able to check the report.
