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
"""
import socket
import struct
import time

IP_BOUND_IF = 25  # <netinet/in.h> on Darwin (macOS/iOS)


def interfaces():
    try:
        return [name for _, name in socket.if_nameindex()]
    except OSError:
        return []


def bound_socket(kind, ifname):
    s = socket.socket(socket.AF_INET, kind)
    s.settimeout(6)
    s.setsockopt(socket.IPPROTO_IP, IP_BOUND_IF, socket.if_nametoindex(ifname))
    return s


def dns_query(ifname):
    """A-record query for example.com over UDP on this interface."""
    q = struct.pack(">HHHHHH", 0x7462, 0x0100, 1, 0, 0, 0)
    q += b"".join(bytes([len(p)]) + p.encode() for p in "example.com".split(".")) + b"\x00"
    q += struct.pack(">HH", 1, 1)
    s = bound_socket(socket.SOCK_DGRAM, ifname)
    try:
        t = time.time()
        s.sendto(q, ("1.1.1.1", 53))
        data, _ = s.recvfrom(512)
        return "UDP ok (%d bytes, %.0f ms)" % (len(data), (time.time() - t) * 1000)
    finally:
        s.close()


def public_ip(ifname):
    s = bound_socket(socket.SOCK_STREAM, ifname)
    try:
        s.connect(("api.ipify.org", 80))
        s.sendall(b"GET / HTTP/1.0\r\nHost: api.ipify.org\r\n\r\n")
        resp = b""
        while True:
            chunk = s.recv(1024)
            if not chunk:
                break
            resp += chunk
        return resp.split(b"\r\n\r\n", 1)[-1].decode().strip()
    finally:
        s.close()


def main():
    names = interfaces()
    print("interfaces:", ", ".join(names) or "(couldn't list)")
    wifi = "en0"
    cell = next((n for n in ("pdp_ip0", "pdp_ip1", "pdp_ip2") if n in names), "pdp_ip0")
    results = {}
    for label, ifname in (("Wi-Fi", wifi), ("Mobile data", cell)):
        print("\n%s (%s):" % (label, ifname))
        try:
            print("  " + dns_query(ifname))
        except Exception as e:  # noqa: BLE001 - report every failure plainly
            print("  UDP failed:", e)
        try:
            ip = public_ip(ifname)
            results[label] = ip
            print("  public IP:", ip)
        except Exception as e:  # noqa: BLE001
            print("  TCP failed:", e)

    print()
    if len(results) == 2 and results["Wi-Fi"] != results["Mobile data"]:
        print("RESULT: YES - both paths work at once (two public IPs).")
        print("An iPhone script can share mobile data while on Wi-Fi.")
    elif "Mobile data" not in results:
        print("RESULT: NO - mobile data wasn't reachable while on Wi-Fi.")
        print("Check mobile data is on (and allowed for this app in Settings > Mobile Data).")
    else:
        print("RESULT: UNCLEAR - both paths answered with the same public IP;")
        print("iOS may have routed both over one connection. Send me the output.")


if __name__ == "__main__":
    main()
