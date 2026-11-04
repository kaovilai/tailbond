#!/usr/bin/env python3
"""Is a Dependabot PR eligible for auto-merge? Exit 0 if yes.

Reads the PR's commits (JSON from the GitHub API, `gh api
repos/OWNER/REPO/pulls/N/commits`). Every commit must be Dependabot's, and
every dependency each one lists must be a semver patch or minor update.
A major, an unclassified entry (git submodule SHA bumps carry no
update-type) or a commit by anyone else keeps the PR for a human.

Used by .github/workflows/dependabot-automerge.yml both to label the PR and,
again, right before merging (the label alone isn't trusted).
"""
import json
import re
import sys

ALLOWED = ("version-update:semver-patch", "version-update:semver-minor")


def check(commits):
    ok, out = bool(commits), []
    if not commits:
        out.append("no commits")
    for c in commits:
        sha = c["sha"][:7]
        author = (c.get("author") or {}).get("login", "")
        if author != "dependabot[bot]":
            ok = False
            out.append("commit %s by %s" % (sha, author or "unknown"))
            continue
        block = c["commit"]["message"].split("updated-dependencies:", 1)
        if len(block) < 2:
            ok = False
            out.append("commit %s has no dependency metadata" % sha)
            continue
        entries = re.split(r"^- dependency-name:", block[1].split("\n...", 1)[0], flags=re.M)[1:]
        if not entries:
            ok = False
            out.append("commit %s lists no dependencies" % sha)
        for e in entries:
            name = e.splitlines()[0].strip()
            m = re.search(r"update-type:\s*(\S+)", e)
            kind = m.group(1) if m else "unknown"
            out.append("%s: %s" % (name, kind))
            if kind not in ALLOWED:
                ok = False
    return ok, out


if __name__ == "__main__":
    ok, lines = check(json.load(open(sys.argv[1])))
    print("\n".join(lines))
    print("eligible" if ok else "not eligible")
    sys.exit(0 if ok else 1)
