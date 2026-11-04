"""Can an App Store Python app use mobile data while the iPhone is on Wi-Fi?

Run on the iPhone in a-Shell (free) or Pythonista 3, with Wi-Fi ON and
mobile data ON:

    curl -LO https://raw.githubusercontent.com/kaovilai/tailbond/feat/deploy-templates/scripts/ios/probe_paths.py
    python3 probe_paths.py

It pins sockets to each interface (IP_BOUND_IF, as tailbond's share relay
would) and asks which public IP each path uses. Two different public IPs
(Wi-Fi's and the carrier's) mean an iPhone script can be a Pair & Share
sharer with no app install or sideloading. Nothing is sent except a DNS
query to 1.1.1.1 and an HTTP request to api.ipify.org on each path.

Options (for testing; the defaults are what the iPhone needs):
    --wifi IF --cell IF        interfaces to compare (default en0, first pdp_ip*)
    --dns HOST:PORT            UDP DNS server (default 1.1.1.1:53)
    --ip-url HOST:PORT         plain-HTTP "what's my IP" service (default api.ipify.org:80)
"""
import argparse
import socket
import struct
import sys
import time

IP_BOUND_IF = 25  # <netinet/in.h> on Darwin (macOS/iOS)
QUERY_ID = 0x7462


def interfaces():
    try:
        return [name for _, name in socket.if_nameindex()]
    except OSError:
        return []


def pin(sock, ifname):
    """Pin sock to ifname: IP_BOUND_IF on Darwin, SO_BINDTODEVICE on Linux."""
    if sys.platform == "darwin" or sys.platform == "ios":
        sock.setsockopt(socket.IPPROTO_IP, IP_BOUND_IF, socket.if_nametoindex(ifname))
    elif hasattr(socket, "SO_BINDTODEVICE"):
        sock.setsockopt(socket.SOL_SOCKET, socket.SO_BINDTODEVICE, ifname.encode())
    else:
        raise OSError("can't pin a socket to an interface on " + sys.platform)


def bound_socket(kind, ifname, timeout=6):
    s = socket.socket(socket.AF_INET, kind)
    s.settimeout(timeout)
    pin(s, ifname)
    return s


def build_query(name="example.com", qid=QUERY_ID):
    """A DNS A-record query for name."""
    q = struct.pack(">HHHHHH", qid, 0x0100, 1, 0, 0, 0)
    q += b"".join(bytes([len(p)]) + p.encode() for p in name.split(".")) + b"\x00"
    return q + struct.pack(">HH", 1, 1)


def is_answer(data, qid=QUERY_ID):
    """True if data is a DNS response to our query id."""
    if len(data) < 12:
        return False
    rid, flags = struct.unpack(">HH", data[:4])
    return rid == qid and bool(flags & 0x8000)


def split_hostport(s, default_port):
    host, _, port = s.rpartition(":")
    return (host, int(port)) if host else (s, default_port)


def dns_query(ifname, server):
    s = bound_socket(socket.SOCK_DGRAM, ifname)
    try:
        t = time.time()
        s.sendto(build_query(), server)
        data, _ = s.recvfrom(512)
        if not is_answer(data):
            raise OSError("not a DNS answer")
        return "UDP ok (%d bytes, %.0f ms)" % (len(data), (time.time() - t) * 1000)
    finally:
        s.close()


def public_ip(ifname, ip_service):
    s = bound_socket(socket.SOCK_STREAM, ifname)
    try:
        s.connect(ip_service)
        s.sendall(b"GET / HTTP/1.0\r\nHost: " + ip_service[0].encode() + b"\r\n\r\n")
        resp = b""
        while True:
            chunk = s.recv(1024)
            if not chunk:
                break
            resp += chunk
        return resp.split(b"\r\n\r\n", 1)[-1].decode().strip()
    finally:
        s.close()


def classify(results):
    """results: {"Wi-Fi": ip, "Mobile data": ip} for the paths that answered."""
    if "Mobile data" not in results:
        return "NO"
    if "Wi-Fi" in results and results["Wi-Fi"] != results["Mobile data"]:
        return "YES"
    return "UNCLEAR"


VERDICT_TEXT = {
    "YES": ["RESULT: YES - both paths work at once (two public IPs).",
            "An iPhone script can share mobile data while on Wi-Fi."],
    "NO": ["RESULT: NO - mobile data wasn't reachable while on Wi-Fi.",
           "Check mobile data is on (and allowed for this app in Settings > Mobile Data)."],
    "UNCLEAR": ["RESULT: UNCLEAR - both paths answered with the same public IP, or Wi-Fi failed;",
                "iOS may have routed both over one connection. Send me the output."],
}


def main(argv=None):
    ap = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    ap.add_argument("--wifi", default="en0")
    ap.add_argument("--cell", default=None)
    ap.add_argument("--dns", default="1.1.1.1:53")
    ap.add_argument("--ip-url", default="api.ipify.org:80")
    args = ap.parse_args(argv)

    names = interfaces()
    print("interfaces:", ", ".join(names) or "(couldn't list)")
    cell = args.cell or next((n for n in ("pdp_ip0", "pdp_ip1", "pdp_ip2") if n in names), "pdp_ip0")
    dns = split_hostport(args.dns, 53)
    ip_service = split_hostport(args.ip_url, 80)

    results = {}
    for label, ifname in (("Wi-Fi", args.wifi), ("Mobile data", cell)):
        print("\n%s (%s):" % (label, ifname))
        try:
            print("  " + dns_query(ifname, dns))
        except Exception as e:  # noqa: BLE001 - report every failure plainly
            print("  UDP failed:", e)
        try:
            ip = public_ip(ifname, ip_service)
            results[label] = ip
            print("  public IP:", ip)
        except Exception as e:  # noqa: BLE001
            print("  TCP failed:", e)

    verdict = classify(results)
    print()
    print("\n".join(VERDICT_TEXT[verdict]))
    return verdict


if __name__ == "__main__":
    main()
