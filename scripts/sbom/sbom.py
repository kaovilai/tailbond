#!/usr/bin/env python3
# SPDX-License-Identifier: Apache-2.0
"""tailbond's release SBOM (docs/sbom.md): one SPDX 2.3 document for every
file of a release.

  sbom.py components [--root DIR]
      mqvpn's bundled C libraries (scripts/sbom/mqvpn-components.json) with
      versions from the pinned submodule commits and sources; fails when one
      can't be derived or a third_party directory isn't listed.
  sbom.py build --assets DIR --version V --out DOC [--android-graph JSON]...
      One package per release file (name, SHA256), each CONTAINS what syft
      finds in it (Go modules in tailbondd, ...), mqvpn and its C libraries
      where an mqvpn binary is inside (libevent's version read from that
      binary), the Android app's Maven runtime dependencies (the Gradle
      dependency graph), and any other release file shipped inside it (the
      APK in the ipk and rpm).
  sbom.py check DOC --assets DIR [--schema JSON] [--require-android-deps]
      Every release file described with its SHA256; mqvpn, xquic, boringssl,
      lwip, yyjson (and libevent where linked statically) present with
      versions in each file that ships mqvpn.
  sbom.py snapshot DOC --sha SHA --ref REF [--run-id N]
      A GitHub dependency snapshot (the dependency submission API): one
      manifest per release file, every package with a purl except tailbond's
      own.

Stdlib only (jsonschema for check --schema). Needs syft on PATH (SYFT), and
for an rpm bsdtar or rpm2cpio + cpio.
"""
import argparse
import collections
import datetime
import hashlib
import json
import os
import re
import shutil
import subprocess
import sys
import tarfile
import tempfile
import uuid
import zipfile

ROOT = os.path.abspath(os.path.join(os.path.dirname(__file__), "..", ".."))
COMPONENTS = os.path.join(os.path.dirname(os.path.abspath(__file__)), "mqvpn-components.json")
# mqvpn's static libraries: every file that ships mqvpn has all of them
MQVPN_LIBS = ("mqvpn", "xquic", "boringssl", "lwip", "yyjson")
OWN_PURLS = ("pkg:golang/github.com/kaovilai/tailbond", "pkg:github/kaovilai/tailbond")


def die(msg):
    print("::error::sbom: " + msg, file=sys.stderr)
    sys.exit(1)


def spdx_id(*parts):
    return "SPDXRef-" + "-".join(re.sub(r"[^A-Za-z0-9.]+", "-", p).strip("-") for p in parts)


def sha256(path):
    h = hashlib.sha256()
    with open(path, "rb") as f:
        for chunk in iter(lambda: f.read(1 << 20), b""):
            h.update(chunk)
    return h.hexdigest()


# ---------------------------------------------------------------- components

def git(*args, cwd):
    r = subprocess.run(["git", *args], cwd=cwd, capture_output=True, text=True)
    return r.stdout.strip() if r.returncode == 0 else ""


def gitlink(root, path):
    """The commit PATH is pinned at: the gitlink in its closest enclosing
    repository (the superproject or a submodule). '' when PATH isn't one."""
    parts = path.split("/")
    for i in range(len(parts) - 1, -1, -1):
        repo = os.path.join(root, *parts[:i])
        if not os.path.exists(os.path.join(repo, ".git")):
            continue
        rel = "/".join(parts[i:])
        out = git("ls-tree", "HEAD", "--", rel, cwd=repo)
        m = re.match(r"160000 commit ([0-9a-f]{40})\t", out)
        return m.group(1) if m else ""
    return ""


def components(root=ROOT):
    spec = json.load(open(COMPONENTS))
    listed = {c["path"] for c in spec["components"]} | set(spec.get("notShipped", {}))
    tp = os.path.join(root, "third_party", "mqvpn", "third_party")
    if not os.path.isdir(tp):
        die(f"{tp} missing: git submodule update --init --recursive")
    for d in sorted(os.listdir(tp)):
        p = "third_party/mqvpn/third_party/" + d
        if os.path.isdir(os.path.join(root, p)) and p not in listed:
            die(f"{p} is in mqvpn but not in {os.path.relpath(COMPONENTS, root)}: add it (components) or say why it isn't shipped (notShipped)")
    out = []
    for c in spec["components"]:
        path = c["path"]
        if c.get("vendored"):
            commit = gitlink(root, path.rsplit("/third_party/", 1)[0])
        else:
            commit = gitlink(root, path)
        if not commit:
            die(f"{c['name']}: no pinned commit for {path} (git submodule update --init --recursive)")
        version = ""
        if "version" in c:
            f = os.path.join(root, path, c["version"]["file"])
            try:
                text = open(f, encoding="utf-8", errors="replace").read()
            except OSError:
                die(f"{c['name']}: {f} missing: can't derive its version (git submodule update --init --recursive)")
            m = re.search(c["version"]["regex"], text)
            if not m:
                die(f"{c['name']}: no version in {f} (regex {c['version']['regex']!r}): update {os.path.relpath(COMPONENTS, root)}")
            version = ".".join(m.groups())
        purl_ver = version if c.get("vendored") else commit
        item = {
            "name": c["name"],
            "versionInfo": version or commit,
            "commit": commit,
            "purl": f"pkg:github/{c['repo']}@{purl_ver}",
            "downloadLocation": f"git+https://github.com/{c['repo']}@{commit}" if not c.get("vendored")
            else f"https://github.com/{c['repo']}",
            "license": c["license"],
            "supplier": c["supplier"],
            "comment": c.get("comment", ""),
        }
        if c.get("cpe"):
            item["cpe"] = c["cpe"].format(version=version, commit=commit)
        out.append(item)
    return out


# ---------------------------------------------------------------- extraction

def extract(path, dest):
    """Unpack a release file into DEST, and the archives inside it (an
    ipk's data.tar.gz), but not a nested APK/ipk/rpm (matched by hash)."""
    with open(path, "rb") as f:
        head = f.read(8)
    os.makedirs(dest, exist_ok=True)
    if head.startswith(b"PK\x03\x04"):
        with zipfile.ZipFile(path) as z:
            z.extractall(dest)
    elif head.startswith(b"\x1f\x8b") or tarfile.is_tarfile(path):
        try:
            t = tarfile.open(path)
        except tarfile.ReadError:  # gzip but not a tar: the opkg feed's Packages.gz
            return False
        with t:
            t.extractall(dest, filter="data")
    elif head.startswith(b"\xed\xab\xee\xdb"):  # rpm
        if shutil.which("bsdtar"):
            subprocess.run(["bsdtar", "-xf", path, "-C", dest], check=True)
        elif shutil.which("rpm2cpio") and shutil.which("cpio"):
            subprocess.run(f"rpm2cpio '{path}' | cpio -idm --quiet", shell=True, cwd=dest, check=True)
        else:
            die("an rpm needs bsdtar (libarchive-tools) or rpm2cpio + cpio")
    else:
        return False
    # an ipk: control.tar.gz + data.tar.gz inside a tar.gz
    for dirpath, _, files in os.walk(dest):
        for n in files:
            if n in ("data.tar.gz", "control.tar.gz", "data.tar.zst", "data.tar.xz"):
                p = os.path.join(dirpath, n)
                with tarfile.open(p) as t:
                    t.extractall(os.path.join(dirpath, n.split(".")[0]), filter="data")
                os.remove(p)
    return True


def is_binary(path):
    with open(path, "rb") as f:
        m = f.read(4)
    return m == b"\x7fELF" or m in (b"\xcf\xfa\xed\xfe", b"\xca\xfe\xba\xbe")


LIBEVENT_STATIC = re.compile(rb"\x00(2\.\d+\.\d+-(?:stable|alpha|beta|rc\d*))\x00")
LIBEVENT_DYNAMIC = re.compile(rb"libevent[-_.0-9a-z]*\.(?:so|dylib)")


def mqvpn_binaries(dest):
    out = []
    for dirpath, _, files in os.walk(dest):
        for n in files:
            p = os.path.join(dirpath, n)
            if (n == "mqvpn" or re.fullmatch(r"lib\w*mqvpn\w*\.so", n)) and not os.path.islink(p) and is_binary(p):
                out.append(p)
    return sorted(out)


def libevent_of(binary):
    """('static', version) | ('dynamic', None) | ('none', None): libevent
    in a shipped mqvpn, from the binary itself. Its LIBEVENT_VERSION string
    is compiled in only when it's linked statically."""
    data = open(binary, "rb").read()
    found = sorted({m.decode() for m in LIBEVENT_STATIC.findall(data)})
    if len(found) > 1:
        die(f"{binary}: several libevent-like versions {found}: can't tell which is libevent")
    if found:
        return "static", found[0]
    if LIBEVENT_DYNAMIC.search(data):
        return "dynamic", None
    return "none", None


# ---------------------------------------------------------------- build

def is_openwrt_apk(name):
    # OpenWrt 25.12+ packages are .apk too (tailbond_<version>_<arch>.apk,
    # tailbond-openwrt-<arch>.apk): not the Android app
    return name.endswith(".apk") and (name.startswith("tailbond_") or name.startswith("tailbond-openwrt-"))


def purpose(name):
    if is_openwrt_apk(name):
        return "INSTALL"
    if name.endswith(".apk"):
        return "APPLICATION"
    if name.endswith((".ipk", ".rpm")):
        return "INSTALL"
    if name.endswith((".tar.gz", ".tgz", ".zip")):
        return "ARCHIVE"
    return "FILE"


def syft_packages(dest, prefix):
    syft = os.environ.get("SYFT", "syft")
    with tempfile.NamedTemporaryFile(suffix=".json") as tmp:
        env = dict(os.environ, SYFT_CHECK_FOR_APP_UPDATE="false")
        subprocess.run([syft, "scan", "dir:" + dest, "-q", "-o", "spdx-json=" + tmp.name], check=True, env=env)
        sub = json.load(open(tmp.name))
    keep = {}
    for p in sub.get("packages", []):
        if p["SPDXID"].startswith("SPDXRef-DocumentRoot"):
            continue
        q = dict(p)
        q["SPDXID"] = spdx_id(prefix, p["SPDXID"][len("SPDXRef-"):])
        q.pop("hasFiles", None)
        q.pop("packageVerificationCode", None)
        q["filesAnalyzed"] = False
        keep[p["SPDXID"]] = q
    rels = []
    for r in sub.get("relationships", []):
        a, b = r["spdxElementId"], r["relatedSpdxElement"]
        if a in keep and b in keep:
            rels.append(dict(r, spdxElementId=keep[a]["SPDXID"], relatedSpdxElement=keep[b]["SPDXID"]))
    return list(keep.values()), rels, sub.get("hasExtractedLicensingInfos", [])


def maven_packages(graph_files, prefix):
    """The Android app's runtime dependencies from Gradle dependency graph
    files (gradle/actions dependency-submission, the dependency snapshot
    format), as SPDX packages. The graph holds only the configurations the
    workflow asks for (releaseRuntimeClasspath: what the APK ships)."""
    seen = {}
    for g in graph_files:
        snap = json.load(open(g))
        for man in snap.get("manifests", {}).values():
            for dep in man.get("resolved", {}).values():
                purl = dep.get("package_url", "")
                m = re.match(r"pkg:maven/([^/]+)/([^@]+)@([^?#]+)", purl)
                # a project of this build (":sdk-core") isn't a dependency
                if not m or m.group(3) == "unspecified":
                    continue
                seen[purl] = m.groups()
    out = []
    for purl, (group, name, ver) in sorted(seen.items()):
        out.append({
            "SPDXID": spdx_id(prefix, "maven", group, name, ver),
            "name": f"{group}:{name}",
            "versionInfo": ver,
            "supplier": "NOASSERTION",
            "downloadLocation": "NOASSERTION",
            "filesAnalyzed": False,
            "licenseConcluded": "NOASSERTION",
            "licenseDeclared": "NOASSERTION",
            "copyrightText": "NOASSERTION",
            "externalRefs": [{"referenceCategory": "PACKAGE-MANAGER", "referenceType": "purl", "referenceLocator": purl}],
        })
    return out


def component_package(c, prefix, version=None):
    p = {
        "SPDXID": spdx_id(prefix, c["name"]),
        "name": c["name"],
        "versionInfo": version or c["versionInfo"],
        "supplier": c["supplier"],
        "downloadLocation": c["downloadLocation"],
        "filesAnalyzed": False,
        "licenseConcluded": "NOASSERTION",
        "licenseDeclared": c["license"],
        "copyrightText": "NOASSERTION",
        "primaryPackagePurpose": "LIBRARY" if c["name"] != "mqvpn" else "APPLICATION",
        "externalRefs": [{"referenceCategory": "PACKAGE-MANAGER", "referenceType": "purl", "referenceLocator": c["purl"]}],
    }
    if c.get("commit"):
        p["sourceInfo"] = f"git commit {c['commit']}"
    if c.get("cpe"):
        p["externalRefs"].append({"referenceCategory": "SECURITY", "referenceType": "cpe23Type", "referenceLocator": c["cpe"]})
    if c.get("comment"):
        p["comment"] = c["comment"]
    return p


def libevent_component(version):
    v = version.split("-")[0]
    return {
        "name": "libevent",
        "versionInfo": version,
        "purl": f"pkg:github/libevent/libevent@release-{version}",
        "downloadLocation": f"https://github.com/libevent/libevent/releases/tag/release-{version}",
        "license": "BSD-3-Clause",
        "supplier": "Organization: libevent contributors (https://libevent.org)",
        "cpe": f"cpe:2.3:a:libevent_project:libevent:{v}:*:*:*:*:*:*:*",
        "comment": "Linked statically into mqvpn from the build host's libevent (Alpine's libevent-static on Linux, Homebrew's on macOS); version read from the shipped binary.",
    }


def build(a):
    version = a.version[1:] if a.version.startswith("v") else a.version
    comps = components()
    names = sorted(n for n in os.listdir(a.assets)
                   if os.path.isfile(os.path.join(a.assets, n)) and n != "SHA256SUMS"
                   and not n.endswith((".spdx.json", ".cdx.json")))
    if not names:
        die(f"no release files in {a.assets}")
    created = datetime.datetime.fromtimestamp(
        int(os.environ.get("SOURCE_DATE_EPOCH", "0")) or datetime.datetime.now(datetime.timezone.utc).timestamp(),
        datetime.timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")
    syft_ver = subprocess.run([os.environ.get("SYFT", "syft"), "version", "-o", "json"],
                              capture_output=True, text=True, check=True)
    syft_ver = json.loads(syft_ver.stdout)["version"]
    doc = {
        "spdxVersion": "SPDX-2.3",
        "dataLicense": "CC0-1.0",
        "SPDXID": "SPDXRef-DOCUMENT",
        "name": f"tailbond-{version}",
        "documentNamespace": a.namespace or f"https://github.com/{a.repo}/sbom/tailbond-{version}-{uuid.uuid4()}",
        "creationInfo": {
            "created": created,
            "creators": ["Organization: tailbond", "Tool: tailbond-sbom.py", f"Tool: syft-{syft_ver}"],
            "comment": "Release files of tailbond " + version + " (docs/sbom.md). "
                       "C libraries linked into mqvpn come from scripts/sbom/mqvpn-components.json.",
        },
        "packages": [],
        "relationships": [],
    }
    licenses = {}
    rel = doc["relationships"]
    sums, by_sum, ids = {}, {}, {}
    for n in names:
        sums[n] = sha256(os.path.join(a.assets, n))
        ids[n] = spdx_id("File", n)
        by_sum.setdefault(sums[n], n)
    if len(set(ids.values())) != len(ids):
        die("two release file names map to the same SPDX id")
    sizes = {os.path.getsize(os.path.join(a.assets, n)): n for n in names}
    work = tempfile.mkdtemp(prefix="tb-sbom-")
    try:
        for n in names:
            pkg = {
                "SPDXID": ids[n],
                "name": n,
                "versionInfo": version,
                "supplier": "Organization: tailbond",
                "downloadLocation": f"https://github.com/{a.repo}/releases/download/{a.tag}/{n}" if a.tag else "NOASSERTION",
                "filesAnalyzed": False,
                "checksums": [{"algorithm": "SHA256", "checksumValue": sums[n]}],
                "licenseConcluded": "NOASSERTION",
                "licenseDeclared": "Apache-2.0",
                "copyrightText": "NOASSERTION",
                "primaryPackagePurpose": purpose(n),
            }
            if a.tag:
                pkg["externalRefs"] = [{"referenceCategory": "PACKAGE-MANAGER", "referenceType": "purl",
                                        "referenceLocator": f"pkg:github/{a.repo}@{a.tag}"}]
            doc["packages"].append(pkg)
            rel.append({"spdxElementId": "SPDXRef-DOCUMENT", "relationshipType": "DESCRIBES", "relatedSpdxElement": ids[n]})
            canon = by_sum[sums[n]]
            if canon != n:  # same bytes under another name (stable ipk/rpm names)
                rel.append({"spdxElementId": ids[n], "relationshipType": "COPY_OF", "relatedSpdxElement": ids[canon]})
                continue
            dest = os.path.join(work, ids[n])
            if not extract(os.path.join(a.assets, n), dest):
                # an OpenWrt .apk (apkv3) is a binary container no extractor
                # here reads: copied as is, its contents covered via the .ipk
                if is_binary(os.path.join(a.assets, n)) or is_openwrt_apk(n):
                    os.makedirs(dest, exist_ok=True)  # extract() may have made it before failing
                    shutil.copy(os.path.join(a.assets, n), dest)
                else:
                    continue  # a text file (the Homebrew formula)
            pkgs, prels, lic = syft_packages(dest, ids[n][len("SPDXRef-"):])
            for l in lic:
                licenses.setdefault(l["licenseId"], l)
            if n.endswith(".apk") and not is_openwrt_apk(n):
                if a.android_graph:
                    pkgs += maven_packages(a.android_graph, ids[n][len("SPDXRef-"):])
                else:
                    print(f"::warning::{n}: no Gradle dependency graph given; its Maven dependencies are missing")
                    pkg["comment"] = "Maven dependencies not listed (no Gradle dependency graph)."
            elif is_openwrt_apk(n):
                # apkv3 isn't extracted: the .ipk in the same release carries
                # the same files, and check() skips them for this one
                pkg["comment"] = "OpenWrt 25.12+ package (apkv3; contents not extracted -- the .ipk carries the same files)."
            notes = []
            bins = mqvpn_binaries(dest)
            if bins:
                prefix = ids[n][len("SPDXRef-"):]
                cp = {c["name"]: component_package(c, prefix) for c in comps}
                if n.endswith(".rpm"):  # packaging/fedora/build-rpm.sh builds the submodule as is
                    cp["mqvpn"]["comment"] = "Built from the pinned submodule without tailbond's patches."
                pkgs += cp.values()
                for c in comps:
                    if c["name"] != "mqvpn":
                        # SPDX: A STATIC_LINK B = A statically links B
                        prels.append({"spdxElementId": cp["mqvpn"]["SPDXID"], "relationshipType": "STATIC_LINK",
                                      "relatedSpdxElement": cp[c["name"]]["SPDXID"]})
                levels = {}
                for b in bins:
                    kind, ver = libevent_of(b)
                    rp = os.path.relpath(b, dest)
                    if kind == "static":
                        levels[ver] = rp
                    elif kind == "dynamic":
                        notes.append(f"{rp} links the system's libevent (a package dependency, not shipped).")
                    elif not b.endswith(".so"):
                        die(f"{n}: {rp} has neither a static libevent version string nor a libevent library dependency: can't derive libevent's version")
                for ver, rp in sorted(levels.items()):
                    lp = component_package(libevent_component(ver), prefix + ("" if len(levels) == 1 else "-" + ver))
                    pkgs.append(lp)
                    prels.append({"spdxElementId": cp["mqvpn"]["SPDXID"], "relationshipType": "STATIC_LINK",
                                  "relatedSpdxElement": lp["SPDXID"]})
            # release files shipped inside this one (the APK in the ipk/rpm)
            for dirpath, _, files in os.walk(dest):
                for f in files:
                    p = os.path.join(dirpath, f)
                    other = sizes.get(os.path.getsize(p)) if not os.path.islink(p) else None
                    if other and other != n and sha256(p) == sums[other]:
                        rel.append({"spdxElementId": ids[n], "relationshipType": "CONTAINS", "relatedSpdxElement": ids[by_sum[sums[other]]]})
                        notes.append(f"{os.path.relpath(p, dest)} is the release file {by_sum[sums[other]]}.")
            if notes:
                pkg["comment"] = " ".join(([pkg["comment"]] if "comment" in pkg else []) + notes)
            doc["packages"] += pkgs
            rel += [{"spdxElementId": ids[n], "relationshipType": "CONTAINS", "relatedSpdxElement": p["SPDXID"]} for p in pkgs]
            rel += prels
            print(f"{n}: {len(pkgs)} packages" + (f", mqvpn in {len(bins)} binaries" if bins else ""), file=sys.stderr)
    finally:
        shutil.rmtree(work, ignore_errors=True)
    if licenses:
        doc["hasExtractedLicensingInfos"] = [licenses[k] for k in sorted(licenses)]
    seen = set()
    for p in doc["packages"]:
        if p["SPDXID"] in seen:
            die("duplicate SPDX id " + p["SPDXID"])
        seen.add(p["SPDXID"])
    with open(a.out, "w") as f:
        json.dump(doc, f, indent=1, sort_keys=False)
        f.write("\n")
    print(f"wrote {a.out}: {len(names)} release files, {len(doc['packages'])} packages", file=sys.stderr)


# ---------------------------------------------------------------- check

def purl_of(p):
    for r in p.get("externalRefs", []):
        if r.get("referenceType") == "purl":
            return r["referenceLocator"]
    return ""


def graph(doc):
    """(packages by id, described ids, CONTAINS children by id, COPY_OF)."""
    pk = {p["SPDXID"]: p for p in doc["packages"]}
    described = [r["relatedSpdxElement"] for r in doc["relationships"] if r["relationshipType"] == "DESCRIBES"]
    contains, copy_of = {}, {}
    for r in doc["relationships"]:
        if r["relationshipType"] == "CONTAINS":
            contains.setdefault(r["spdxElementId"], []).append(r["relatedSpdxElement"])
        elif r["relationshipType"] == "COPY_OF":
            copy_of[r["spdxElementId"]] = r["relatedSpdxElement"]
    return pk, described, contains, copy_of


def reach(sid, contains, copy_of):
    """Everything SID ships, through nested release files (the APK in the
    ipk/rpm) and COPY_OF aliases: {id: depth}, 1 = directly inside."""
    root = copy_of.get(sid, sid)
    out, seen, todo = {}, {root}, collections.deque([(root, 0)])
    while todo:
        s, d = todo.popleft()
        for c in contains.get(s, []):
            c = copy_of.get(c, c)
            if c in seen:
                continue
            seen.add(c)
            out[c] = d + 1
            todo.append((c, d + 1))
    return out


def manifests(doc):
    """Dependency snapshot manifests: one per release file (copies skipped),
    every package it ships with a purl, tailbond's own excepted."""
    pk, described, contains, copy_of = graph(doc)
    out = {}
    for sid in described:
        if sid in copy_of:
            continue
        name = pk[sid]["name"]
        resolved = {}
        for c, depth in sorted(reach(sid, contains, copy_of).items(), key=lambda x: x[1]):
            purl = purl_of(pk[c])
            if purl and not purl.startswith(OWN_PURLS) and purl not in resolved:
                resolved[purl] = {"package_url": purl, "relationship": "direct" if depth == 1 else "indirect",
                                  "scope": "runtime"}
        if resolved:
            out[name] = {"name": name, "file": {"source_location": "release/" + name}, "resolved": resolved}
    return out


def read_sums(path):
    """SHA256SUMS (sha256sum's output) as {file name: hash}."""
    sums = {}
    for line in open(path):
        m = re.fullmatch(r"([0-9a-f]{64}) [ *](.+)", line.rstrip("\n"))
        if not m:
            die(f"{path}: not a sha256sum line: {line.rstrip()!r}")
        sums[m.group(2)] = m.group(1)
    return sums


def check(a):
    doc = json.load(open(a.doc))
    errs = []
    if a.schema:
        import jsonschema
        v = jsonschema.Draft7Validator(json.load(open(a.schema)))
        for e in sorted(v.iter_errors(doc), key=lambda e: list(e.path)):
            errs.append("schema: " + "/".join(map(str, e.path)) + ": " + e.message[:200])
    pk = {p["SPDXID"]: p for p in doc["packages"]}
    ids = set(pk) | {"SPDXRef-DOCUMENT"}
    for r in doc["relationships"]:
        for k in ("spdxElementId", "relatedSpdxElement"):
            if r[k] not in ids:
                errs.append(f"relationship to unknown {r[k]}")
    _, described, kids, copy_of = graph(doc)
    contains = {k: [pk.get(c, {}) for c in v] for k, v in kids.items()}
    byname = {pk[i]["name"]: pk[i] for i in described if i in pk}
    # SPDX: A STATIC_LINK B = A statically links B; only mqvpn links anything
    links = set()
    for r in doc["relationships"]:
        if r["relationshipType"] == "STATIC_LINK":
            links.add((r["spdxElementId"], r["relatedSpdxElement"]))
            if pk.get(r["spdxElementId"], {}).get("name") != "mqvpn":
                errs.append(f"{r['spdxElementId']} STATIC_LINK {r['relatedSpdxElement']}: the source must be mqvpn (mqvpn STATIC_LINK <library>)")
    files = sorted(n for n in os.listdir(a.assets)
                   if n != "SHA256SUMS" and not n.endswith((".spdx.json", ".cdx.json")))
    # SHA256SUMS is what the attestation's subjects come from: it must list
    # exactly these files with their hashes, as the SBOM does.
    sumsf = os.path.join(a.assets, "SHA256SUMS")
    if a.require_sums and not os.path.isfile(sumsf):
        errs.append("no SHA256SUMS")
    elif os.path.isfile(sumsf):
        sums = read_sums(sumsf)
        for n in sorted(set(sums) - set(files)):
            errs.append(f"SHA256SUMS lists {n}, which isn't a release file here")
        for n in files:
            if n not in sums:
                errs.append(f"{n}: not in SHA256SUMS")
            elif sums[n] != sha256(os.path.join(a.assets, n)):
                errs.append(f"{n}: SHA256SUMS says {sums[n]}, the file is {sha256(os.path.join(a.assets, n))}")
        for n, p in sorted(byname.items()):
            got = [c["checksumValue"] for c in p.get("checksums", []) if c["algorithm"] == "SHA256"]
            if n not in sums:
                errs.append(f"SBOM package {n} is not in SHA256SUMS")
            elif got != [sums[n]]:
                errs.append(f"SBOM package {n}: SHA256 {got}, SHA256SUMS {sums[n]}")
    apks = [byname[n]["SPDXID"] for n in files if n.endswith(".apk") and n in byname]
    snap = manifests(doc)
    for n in files:
        p = byname.get(n)
        if not p:
            errs.append(f"{n}: not described")
            continue
        want = sha256(os.path.join(a.assets, n))
        got = [c["checksumValue"] for c in p.get("checksums", []) if c["algorithm"] == "SHA256"]
        if got != [want]:
            errs.append(f"{n}: SHA256 {got} is not {want}")
        sid = copy_of.get(p["SPDXID"], p["SPDXID"])
        inside = {c.get("name"): c for c in contains.get(sid, [])}
        # an OpenWrt .apk (apkv3) isn't extracted: its mqvpn, Go modules and
        # the Android APK inside are the .ipk's, which the checks below cover
        if re.search(r"(\.ipk|\.rpm|\.apk|^tailbond-(linux|macos)-.*\.tar\.gz)$", n) and not is_openwrt_apk(n):
            for lib in MQVPN_LIBS:
                c = inside.get(lib)
                if not c or c.get("versionInfo", "NOASSERTION") in ("", "NOASSERTION") or not purl_of(c):
                    errs.append(f"{n}: no {lib} with a version and purl")
        if re.search(r"(\.ipk|^tailbond-(linux|macos)-.*\.tar\.gz)$", n):
            c = inside.get("libevent")
            if not c or not re.fullmatch(r"\d+\.\d+\.\d+-\w+", c.get("versionInfo", "")):
                errs.append(f"{n}: no statically linked libevent with a version")
        if re.search(r"^tailbondd-.*\.tar\.gz$|^tailbond-(linux|macos)-.*\.tar\.gz$|\.ipk$|\.rpm$", n):
            if not any(purl_of(c).startswith("pkg:golang/") for c in inside.values()):
                errs.append(f"{n}: no Go modules (tailbondd)")
        if n.endswith(".apk") and not is_openwrt_apk(n) and a.require_android_deps:
            if sum(purl_of(c).startswith("pkg:maven/") for c in inside.values()) < 10:
                errs.append(f"{n}: fewer than 10 Maven dependencies (the Gradle dependency graph)")
        mq = inside.get("mqvpn")
        if mq:
            for lib in MQVPN_LIBS[1:] + ("libevent",):
                c = inside.get(lib)
                if c and (mq["SPDXID"], c["SPDXID"]) not in links:
                    errs.append(f"{n}: no mqvpn STATIC_LINK {lib}")
        # the ipk and rpm ship the APK: it, and its Maven libraries, are in them
        if re.search(r"\.(ipk|rpm)$", n) and apks:
            if not set(apks) & {copy_of.get(c, c) for c in kids.get(sid, [])}:
                errs.append(f"{n}: doesn't CONTAIN the release APK")
            m = snap.get(pk[sid]["name"], {})  # a copy: its original's manifest
            maven = sum(k.startswith("pkg:maven/") for k in m.get("resolved", {}))
            if a.require_android_deps and maven < 10:
                errs.append(f"{n}: its dependency snapshot manifest has {maven} Maven libraries (the APK's)")
    if errs:
        for e in errs[:60]:
            print("::error::sbom check: " + e, file=sys.stderr)
        sys.exit(1)
    print(f"{a.doc}: OK ({len(byname)} release files, {len(pk)} packages)")
    if os.environ.get("GITHUB_STEP_SUMMARY"):
        libs = MQVPN_LIBS + ("libevent",)
        rows = ["| release file | packages | Go | Maven | " + " | ".join(libs) + " |",
                "|---|---|---|---|" + "---|" * len(libs)]
        for n, p in sorted(byname.items()):
            sid = copy_of.get(p["SPDXID"], p["SPDXID"])
            inside = contains.get(sid, [])
            vers = {c.get("name"): c.get("versionInfo", "") for c in inside}
            cell = lambda v: (v[:12] if re.fullmatch(r"[0-9a-f]{40}", v) else v) if v else ""
            rows.append(f"| {n}{' (copy)' if sid != p['SPDXID'] else ''} | {len(inside)} | "
                        f"{sum(purl_of(c).startswith('pkg:golang/') for c in inside)} | "
                        f"{sum(purl_of(c).startswith('pkg:maven/') for c in inside)} | "
                        + " | ".join(cell(vers.get(l, "")) for l in libs) + " |")
        with open(os.environ["GITHUB_STEP_SUMMARY"], "a") as f:
            f.write(f"### SBOM {doc['name']}\n\n" + "\n".join(rows) + "\n\n")


# ---------------------------------------------------------------- snapshot

def snapshot(a):
    doc = json.load(open(a.doc))
    manifests_ = manifests(doc)
    snap = {
        "version": 0,
        "sha": a.sha,
        "ref": a.ref,
        "job": {"correlator": a.correlator, "id": str(a.run_id)},
        "detector": {"name": "tailbond-release-sbom", "version": "1", "url": f"https://github.com/{a.repo}/blob/main/docs/sbom.md"},
        "scanned": datetime.datetime.now(datetime.timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ"),
        "manifests": manifests_,
    }
    json.dump(snap, sys.stdout, indent=1)
    sys.stdout.write("\n")
    print(f"{len(manifests_)} manifests, {sum(len(m['resolved']) for m in manifests_.values())} dependencies", file=sys.stderr)


def main():
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    sub = ap.add_subparsers(dest="cmd", required=True)
    s = sub.add_parser("components")
    s.add_argument("--root", default=ROOT)
    s = sub.add_parser("build")
    s.add_argument("--assets", required=True)
    s.add_argument("--version", required=True)
    s.add_argument("--out", required=True)
    s.add_argument("--repo", default=os.environ.get("GITHUB_REPOSITORY", "kaovilai/tailbond"))
    s.add_argument("--tag", default="", help="the release tag: download URLs and purls")
    s.add_argument("--namespace", default="")
    s.add_argument("--android-graph", action="extend", nargs="+", default=[], help="Gradle dependency graph JSON files")
    s = sub.add_parser("check")
    s.add_argument("doc")
    s.add_argument("--assets", required=True)
    s.add_argument("--schema")
    s.add_argument("--require-android-deps", action="store_true")
    s.add_argument("--require-sums", action="store_true", help="SHA256SUMS must exist (it always is checked when present)")
    s = sub.add_parser("snapshot")
    s.add_argument("doc")
    s.add_argument("--sha", required=True)
    s.add_argument("--ref", required=True)
    s.add_argument("--repo", default=os.environ.get("GITHUB_REPOSITORY", "kaovilai/tailbond"))
    s.add_argument("--run-id", default=os.environ.get("GITHUB_RUN_ID", "0"))
    s.add_argument("--correlator", default="tailbond-release-sbom")
    a = ap.parse_args()
    if a.cmd == "components":
        json.dump(components(a.root), sys.stdout, indent=1)
        sys.stdout.write("\n")
    elif a.cmd == "build":
        build(a)
    elif a.cmd == "check":
        check(a)
    else:
        snapshot(a)


if __name__ == "__main__":
    main()
