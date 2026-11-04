"""Tests for probe_paths.py (run in CI on Linux and macOS; no cellular needed).

    python3 -m unittest discover -s scripts/ios -p 'test_*.py' -v

Pure logic runs everywhere. The socket tests use loopback servers and pin to
the loopback interface: IP_BOUND_IF on macOS; on Linux SO_BINDTODEVICE needs
CAP_NET_RAW, so those tests run there only as root.
"""
import contextlib
import io
import os
import socket
import struct
import sys
import threading
import unittest

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import probe_paths as pp  # noqa: E402

LOOPBACK = "lo0" if sys.platform == "darwin" else "lo"
CAN_PIN = sys.platform == "darwin" or (hasattr(os, "geteuid") and os.geteuid() == 0)


class Logic(unittest.TestCase):
    def test_query_shape(self):
        q = pp.build_query("example.com")
        qid, flags, qd, an, ns, ar = struct.unpack(">HHHHHH", q[:12])
        self.assertEqual((qid, flags, qd, an, ns, ar), (pp.QUERY_ID, 0x0100, 1, 0, 0, 0))
        self.assertEqual(q[12:25], b"\x07example\x03com\x00")
        self.assertEqual(q[-4:], struct.pack(">HH", 1, 1))

    def test_is_answer(self):
        resp = struct.pack(">HH", pp.QUERY_ID, 0x8180) + b"\x00" * 8
        self.assertTrue(pp.is_answer(resp))
        self.assertFalse(pp.is_answer(struct.pack(">HH", pp.QUERY_ID, 0x0100) + b"\x00" * 8))
        self.assertFalse(pp.is_answer(struct.pack(">HH", 1, 0x8180) + b"\x00" * 8))
        self.assertFalse(pp.is_answer(b"short"))

    def test_classify(self):
        self.assertEqual(pp.classify({"Wi-Fi": "1.1.1.1", "Mobile data": "2.2.2.2"}), "YES")
        self.assertEqual(pp.classify({"Wi-Fi": "1.1.1.1", "Mobile data": "1.1.1.1"}), "UNCLEAR")
        self.assertEqual(pp.classify({"Mobile data": "2.2.2.2"}), "UNCLEAR")
        self.assertEqual(pp.classify({"Wi-Fi": "1.1.1.1"}), "NO")
        self.assertEqual(pp.classify({}), "NO")

    def test_split_hostport(self):
        self.assertEqual(pp.split_hostport("1.1.1.1:53", 53), ("1.1.1.1", 53))
        self.assertEqual(pp.split_hostport("api.ipify.org", 80), ("api.ipify.org", 80))
        self.assertEqual(pp.split_hostport("127.0.0.1:8053", 53), ("127.0.0.1", 8053))

    def test_parse_ip_response(self):
        self.assertEqual(pp.parse_ip_response(b"HTTP/1.1 200 OK\r\nX: y\r\n\r\n203.0.113.9\n"), "203.0.113.9")
        self.assertEqual(pp.parse_ip_response(b"HTTP/1.0 200 OK\r\n\r\n2001:db8::1"), "2001:db8::1")
        # A captive portal's redirect or login page must not count as an IP.
        with self.assertRaises(ValueError):
            pp.parse_ip_response(b"HTTP/1.1 302 Found\r\nLocation: http://portal/\r\n\r\n")
        with self.assertRaises(ValueError):
            pp.parse_ip_response(b"HTTP/1.1 200 OK\r\n\r\n<html>Log in to Hotel Wi-Fi</html>")
        with self.assertRaises(ValueError):
            pp.parse_ip_response(b"")


@contextlib.contextmanager
def loopback_servers(public_ip="203.0.113.9"):
    """A UDP DNS echo-answerer and an HTTP 'what's my IP' server on 127.0.0.1."""
    udp = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
    udp.bind(("127.0.0.1", 0))
    tcp = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    tcp.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
    tcp.bind(("127.0.0.1", 0))
    tcp.listen(4)
    stop = threading.Event()

    def dns():
        udp.settimeout(0.2)
        while not stop.is_set():
            try:
                data, addr = udp.recvfrom(512)
            except socket.timeout:
                continue
            udp.sendto(data[:2] + struct.pack(">H", 0x8180) + data[4:], addr)

    def http():
        tcp.settimeout(0.2)
        while not stop.is_set():
            try:
                c, _ = tcp.accept()
            except socket.timeout:
                continue
            with c:
                c.recv(1024)
                c.sendall(b"HTTP/1.0 200 OK\r\n\r\n" + public_ip.encode())

    threads = [threading.Thread(target=f, daemon=True) for f in (dns, http)]
    for t in threads:
        t.start()
    try:
        yield ("127.0.0.1", udp.getsockname()[1]), ("127.0.0.1", tcp.getsockname()[1])
    finally:
        stop.set()
        for t in threads:
            t.join(1)
        udp.close()
        tcp.close()


@unittest.skipUnless(CAN_PIN, "pinning to an interface needs macOS, or root on Linux")
class Sockets(unittest.TestCase):
    def test_pinned_paths_against_loopback(self):
        with loopback_servers() as (dns, ipsvc):
            self.assertTrue(pp.dns_query(LOOPBACK, dns).startswith("UDP ok"))
            self.assertEqual(pp.public_ip(LOOPBACK, ipsvc), "203.0.113.9")

    def test_main_reports_unclear_when_both_paths_share_an_ip(self):
        with loopback_servers() as (dns, ipsvc):
            out = io.StringIO()
            with contextlib.redirect_stdout(out):
                verdict = pp.main(["--wifi", LOOPBACK, "--cell", LOOPBACK,
                                   "--dns", "%s:%d" % dns, "--ip-url", "%s:%d" % ipsvc])
        self.assertEqual(verdict, "UNCLEAR", out.getvalue())

    def test_main_reports_no_when_cellular_missing(self):
        with loopback_servers() as (dns, ipsvc):
            out = io.StringIO()
            with contextlib.redirect_stdout(out):
                verdict = pp.main(["--wifi", LOOPBACK, "--cell", "no-such-if0",
                                   "--dns", "%s:%d" % dns, "--ip-url", "%s:%d" % ipsvc])
        self.assertEqual(verdict, "NO", out.getvalue())
        self.assertIn("TCP failed", out.getvalue())


if __name__ == "__main__":
    unittest.main()
