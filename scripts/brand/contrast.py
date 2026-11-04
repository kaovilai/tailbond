#!/usr/bin/env python3
# SPDX-License-Identifier: Apache-2.0
"""WCAG 2.x contrast of the brand tokens (assets/brand/tokens.json) the way
the UIs use them: body text needs 4.5:1 (AA), icons, chart lines and dots
3:1 (non-text contrast). Connection and status colours are for icons, lines
and dots only: in light mode Wi-Fi, USB and tailnet are under 4.5:1 on
white, so text stays in the text colour; light UIs use status-light (the
brand status set is under 3:1 on white) (docs/brand.md).
Exit 1 when a use the UIs rely on fails.

  scripts/brand/contrast.py          the table, failures marked
"""
import json
import os
import sys

ROOT = os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
T = json.load(open(os.path.join(ROOT, "assets/brand/tokens.json")))


def lum(c):
    c = c.lstrip("#")
    ch = [int(c[i:i + 2], 16) / 255 for i in (0, 2, 4)]
    ch = [x / 12.92 if x <= 0.03928 else ((x + 0.055) / 1.055) ** 2.4 for x in ch]
    return 0.2126 * ch[0] + 0.7152 * ch[1] + 0.0722 * ch[2]


def ratio(a, b):
    la, lb = sorted((lum(a), lum(b)), reverse=True)
    return (la + 0.05) / (lb + 0.05)


def checks():
    out = []  # (what, fg, bg, need)
    for mode in ("dark", "light"):
        s, k = T["surface"][mode], T["connection"][mode]
        for surf in ("bg", "card"):
            out.append((f"{mode}: text on {surf}", s["text"], s[surf], 4.5))
            out.append((f"{mode}: muted text on {surf}", s["muted"], s[surf], 4.5))
            out.append((f"{mode}: primary (links, active tab text) on {surf}", s["primary"], s[surf], 4.5))
            for n in k:
                # connection colours: icons and chart lines (3:1), and the kind
                # chip's label text (4.5:1) on cards
                out.append((f"{mode}: {n} icon/line on {surf}", k[n], s[surf], 3.0))
        for n, c in (T["status"] if mode == "dark" else T["status-light"]).items():
            # status colours: dots and icons (3:1); status words are body text
            out.append((f"{mode}: status {n} dot/icon on card", c, s["card"], 3.0))
        out.append((f"{mode}: on-primary on primary (buttons)", s["on-primary"], s["primary"], 4.5))
    return out


def main():
    bad = 0
    for what, fg, bg, need in checks():
        r = ratio(fg, bg)
        ok = r >= need
        bad += not ok
        print(f"{'ok  ' if ok else 'FAIL'} {r:5.2f}:1 (need {need}) {what}  {fg} on {bg}")
    print(f"\n{bad} failing")
    sys.exit(1 if bad else 0)


if __name__ == "__main__":
    main()
