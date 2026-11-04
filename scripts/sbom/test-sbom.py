#!/usr/bin/env python3
# SPDX-License-Identifier: Apache-2.0
"""Self-test for scripts/sbom/sbom.py (CI job sbom; docs/sbom.md).

Builds tiny release files (an APK, an ipk that ships it, a renamed copy of
the ipk, SHA256SUMS) and a Gradle graph, runs sbom.py build with a stub
syft, then checks:
  - check passes on the good SBOM; snapshot gives the ipk the APK's Maven
    libraries (indirect) and its own Go modules (direct), none to the copy
  - mqvpn STATIC_LINK <library> (SPDX: A statically links B), and check
    rejects the reverse
  - check rejects a SHA256SUMS that disagrees with the files or the SBOM
  - check rejects an ipk that no longer CONTAINS the APK
Needs the mqvpn submodules (sbom.py components), not syft.
"""
import atexit
import hashlib
import io
import json
import os
import shutil
import subprocess
import sys
import tarfile
import tempfile
import zipfile

HERE = os.path.dirname(os.path.abspath(__file__))
SBOM = os.path.join(HERE, "sbom.py")
MAVEN = [f"pkg:maven/androidx.test/lib{i}@1.{i}.0" for i in range(12)]
GO = "pkg:golang/golang.org/x/net@v0.1.0"
ELF_STATIC = b"\x7fELF" + b"\0" * 60 + b"\x002.1.13-stable\x00" + b"\0" * 16
ELF = b"\x7fELF" + b"\0" * 80

STUB_SYFT = r'''#!/usr/bin/env python3
import json, os, sys
if sys.argv[1] == "version":
    print(json.dumps({"version": "0.0.0-stub"})); sys.exit(0)
src = next(a[4:] for a in sys.argv if a.startswith("dir:"))
out = next(a.split("=", 1)[1] for a in sys.argv if a.startswith("spdx-json="))
pk = [{"SPDXID": "SPDXRef-DocumentRoot-Directory-x", "name": src}]
if any("tailbondd" in fs for _, _, fs in os.walk(src)):
    pk.append({"SPDXID": "SPDXRef-Package-go-net", "name": "golang.org/x/net", "versionInfo": "v0.1.0",
               "downloadLocation": "NOASSERTION", "filesAnalyzed": False,
               "externalRefs": [{"referenceCategory": "PACKAGE-MANAGER", "referenceType": "purl",
                                 "referenceLocator": "''' + GO + r'''"}]})
json.dump({"packages": pk, "relationships": []}, open(out, "w"))
'''

fails = 0


def tgz(members):
    buf = io.BytesIO()
    with tarfile.open(fileobj=buf, mode="w:gz") as t:
        for name, data in members.items():
            ti = tarfile.TarInfo(name)
            ti.size = len(data)
            ti.mode = 0o755
            t.addfile(ti, io.BytesIO(data))
    return buf.getvalue()


def write_sums(d, override=None, drop=None, extra=None):
    lines = []
    for n in sorted(os.listdir(d)):
        if n == "SHA256SUMS" or n == drop:
            continue
        h = hashlib.sha256(open(os.path.join(d, n), "rb").read()).hexdigest()
        lines.append(f"{(override or {}).get(n, h)}  {n}\n")
    if extra:
        lines.append(extra)
    open(os.path.join(d, "SHA256SUMS"), "w").write("".join(lines))


def run(*args, env=None):
    return subprocess.run([sys.executable, SBOM, *args], capture_output=True, text=True, env=env)


def expect(name, r, ok, needle=""):
    global fails
    good = (r.returncode == 0) == ok and needle in (r.stdout + r.stderr)
    print(("ok   " if good else "FAIL ") + name)
    if not good:
        fails += 1
        print(r.stdout[-1500:], r.stderr[-1500:])


def main():
    w = tempfile.mkdtemp(prefix="tb-sbom-test-")
    atexit.register(shutil.rmtree, w, True)
    assets = os.path.join(w, "assets")
    os.makedirs(assets)
    apk = io.BytesIO()
    with zipfile.ZipFile(apk, "w") as z:
        z.writestr("lib/arm64-v8a/libmqvpn_jni.so", ELF)
        z.writestr("classes.dex", b"dex\n")
    apk = apk.getvalue()
    open(os.path.join(assets, "tailbond-android.apk"), "wb").write(apk)
    data = tgz({"./usr/bin/mqvpn": ELF_STATIC, "./usr/bin/tailbondd": ELF,
                "./usr/share/tailbond/tailbond.apk": apk})
    ipk = tgz({"./debian-binary": b"2.0\n", "./control.tar.gz": tgz({"./control": b"Package: tailbond\n"}),
               "./data.tar.gz": data})
    open(os.path.join(assets, "tailbond-openwrt-aarch64_cortex-a53.ipk"), "wb").write(ipk)
    open(os.path.join(assets, "tailbond_1.0.0-r1_aarch64_cortex-a53.ipk"), "wb").write(ipk)
    write_sums(assets)
    graph = os.path.join(w, "graph.json")
    json.dump({"manifests": {"project :app": {"resolved": {
        **{p: {"package_url": p} for p in MAVEN},
        "own": {"package_url": "pkg:maven/tailbond-android/sdk-core@unspecified"}}}}}, open(graph, "w"))
    syft = os.path.join(w, "syft")
    open(syft, "w").write(STUB_SYFT)
    os.chmod(syft, 0o755)
    env = dict(os.environ, SYFT=syft, TMPDIR=w)
    doc = os.path.join(w, "sbom.spdx.json")

    r = run("build", "--assets", assets, "--version", "1.0.0", "--android-graph", graph, "--out", doc, env=env)
    expect("build", r, True)
    if r.returncode:
        sys.exit(1)
    chk = ("check", doc, "--assets", assets, "--require-android-deps", "--require-sums")
    expect("check: good SBOM", run(*chk), True, "OK")

    snap = json.loads(run("snapshot", doc, "--sha", "0" * 40, "--ref", "refs/heads/x").stdout)
    m = snap["manifests"]
    ipk_m = m.get("tailbond-openwrt-aarch64_cortex-a53.ipk", {}).get("resolved", {})
    expect("snapshot: ipk has the APK's Maven libraries (indirect)",
           subprocess.CompletedProcess([], 0 if all(ipk_m.get(p, {}).get("relationship") == "indirect" for p in MAVEN) else 1, "", ""), True)
    expect("snapshot: ipk's own Go modules are direct",
           subprocess.CompletedProcess([], 0 if ipk_m.get(GO, {}).get("relationship") == "direct" else 1, "", ""), True)
    expect("snapshot: no manifest for the copy, none of tailbond's own",
           subprocess.CompletedProcess([], 0 if "tailbond_1.0.0-r1_aarch64_cortex-a53.ipk" not in m
                                       and not any("unspecified" in p for v in m.values() for p in v["resolved"]) else 1, "", ""), True)

    d = json.load(open(doc))
    pk = {p["SPDXID"]: p for p in d["packages"]}
    sl = [(pk[r["spdxElementId"]]["name"], pk[r["relatedSpdxElement"]]["name"])
          for r in d["relationships"] if r["relationshipType"] == "STATIC_LINK"]
    want = {("mqvpn", lib) for lib in ("xquic", "boringssl", "lwip", "yyjson", "libevent")}
    expect("build: mqvpn STATIC_LINK each library",
           subprocess.CompletedProcess([], 0 if want <= set(sl) and all(a == "mqvpn" for a, _ in sl) else 1, "", str(sl)), True)

    bad = os.path.join(w, "bad.spdx.json")

    def mutate(f):
        x = json.load(open(doc))
        f(x)
        json.dump(x, open(bad, "w"))

    def reverse(x):
        for r in x["relationships"]:
            if r["relationshipType"] == "STATIC_LINK":
                r["spdxElementId"], r["relatedSpdxElement"] = r["relatedSpdxElement"], r["spdxElementId"]
    mutate(reverse)
    expect("check: rejects <library> STATIC_LINK mqvpn", run("check", bad, *chk[2:]), False, "the source must be mqvpn")

    ipk_id = next(i for i, p in pk.items() if p["name"] == "tailbond-openwrt-aarch64_cortex-a53.ipk")
    apk_id = next(i for i, p in pk.items() if p["name"] == "tailbond-android.apk")
    mutate(lambda x: x.update(relationships=[r for r in x["relationships"] if not (
        r["spdxElementId"] == ipk_id and r["relatedSpdxElement"] == apk_id)]))
    expect("check: rejects an ipk that doesn't CONTAIN the APK", run("check", bad, *chk[2:]), False, "CONTAIN the release APK")

    write_sums(assets, override={"tailbond-android.apk": "0" * 64})
    expect("check: rejects a wrong hash in SHA256SUMS", run(*chk), False, "SHA256SUMS says")
    write_sums(assets, drop="tailbond-android.apk")
    expect("check: rejects a file missing from SHA256SUMS", run(*chk), False, "not in SHA256SUMS")
    write_sums(assets, extra="0" * 64 + "  gone.tar.gz\n")
    expect("check: rejects SHA256SUMS listing a file that isn't there", run(*chk), False, "isn't a release file")
    open(os.path.join(assets, "tailbond-android.apk"), "ab").write(b"x")
    write_sums(assets)
    expect("check: rejects an SBOM older than the files and SHA256SUMS", run(*chk), False, "SHA256SUMS")
    os.remove(os.path.join(assets, "SHA256SUMS"))
    expect("check --require-sums: rejects no SHA256SUMS", run(*chk), False, "no SHA256SUMS")

    if fails:
        print(f"{fails} failed")
        sys.exit(1)
    print("all passed")


if __name__ == "__main__":
    main()
