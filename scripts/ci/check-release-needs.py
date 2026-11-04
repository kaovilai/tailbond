#!/usr/bin/env python3
"""Fail if a CI job isn't a prerequisite of the release job.

Releases publish automatically (auto-release.yml), so "the release only
publishes if every job passes" must stay true as jobs are added.
Jobs that run after the release (needs: release) are exempt.
"""
import sys

import yaml

wf = yaml.safe_load(open(sys.argv[1] if len(sys.argv) > 1 else ".github/workflows/ci.yml"))
jobs = wf["jobs"]
needs = set(jobs["release"].get("needs", []))


def after_release(job):
    n = jobs[job].get("needs", [])
    return "release" in ([n] if isinstance(n, str) else n)


missing = sorted(j for j in jobs if j != "release" and j not in needs and not after_release(j))
if missing:
    print("release.needs is missing: " + ", ".join(missing))
    print("Add them to the release job's needs in .github/workflows/ci.yml.")
    sys.exit(1)
print("release.needs covers all %d jobs" % len(needs))
