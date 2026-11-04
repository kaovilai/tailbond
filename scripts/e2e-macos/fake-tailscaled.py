#!/usr/bin/env python3
# SPDX-License-Identifier: Apache-2.0
"""A stand-in for Tailscale's interface on a Mac, for scripts/e2e-macos-server.sh.

Opens a utun (like tailscaled does), gives it the "Tailscale address"
100.101.0.1 (peer 100.101.0.2) and routes 100.64.0.0/10 to it, as
Tailscale does. Behind it lives a pretend tailnet:

- 100.101.0.2 (a tailnet machine) answers ICMP echo and logs the source
  address it saw: the Mac's server must NAT bonded clients' traffic to
  100.101.0.1 on its way into the tailnet.
- 192.0.2.0/24 is routed here too: a source that is neither the tailnet
  nor the tunnel, to show the dashboard and the phone page refuse it.

A control socket (UDP 127.0.0.1:PORT, one command per datagram, the reply
is one datagram) injects packets into the Mac from behind this interface:

  syn SRC DST DPORT    a TCP SYN; reply "synack", "rst" or "none" (3 s)
  ping SRC DST         an ICMP echo; reply "reply FROM" or "none" (5 s)
  addr                 the utun's name

Every packet that arrives is logged to the file named on the command line.
Usage: fake-tailscaled.py LOG CONTROL_PORT IFNAME_FILE   (root; macOS)
"""
import os
import random
import select
import socket
import struct
import subprocess
import sys
import time

LOG, CTL_PORT, IFNAME_FILE = sys.argv[1], int(sys.argv[2]), sys.argv[3]
SELF, PEER = "100.101.0.1", "100.101.0.2"
UTUN_OPT_IFNAME = 2


def log(msg):
    with open(LOG, "a") as f:
        f.write("%.3f %s\n" % (time.time(), msg))


def csum(b):
    if len(b) % 2:
        b += b"\0"
    s = sum(struct.unpack("!%dH" % (len(b) // 2), b))
    while s >> 16:
        s = (s & 0xFFFF) + (s >> 16)
    return (~s) & 0xFFFF


def ip4(src, dst, proto, payload, ident=None):
    hdr = struct.pack("!BBHHHBBH4s4s", 0x45, 0, 20 + len(payload),
                      ident if ident is not None else random.randint(0, 0xFFFF), 0, 64, proto, 0,
                      socket.inet_aton(src), socket.inet_aton(dst))
    hdr = hdr[:10] + struct.pack("!H", csum(hdr)) + hdr[12:]
    return hdr + payload


def tcp_syn(src, dst, sport, dport, flags=0x02, seq=None, ack=0):
    seq = random.randint(0, 0xFFFFFFFF) if seq is None else seq
    t = struct.pack("!HHIIBBHHH", sport, dport, seq, ack, 5 << 4, flags, 65535, 0, 0)
    pseudo = socket.inet_aton(src) + socket.inet_aton(dst) + struct.pack("!BBH", 0, 6, len(t))
    t = t[:16] + struct.pack("!H", csum(pseudo + t)) + t[18:]
    return ip4(src, dst, 6, t)


def icmp_echo(src, dst, typ, ident, seq, data):
    m = struct.pack("!BBHHH", typ, 0, 0, ident, seq) + data
    m = m[:2] + struct.pack("!H", csum(m)) + m[4:]
    return ip4(src, dst, 1, m)


tun = socket.socket(socket.PF_SYSTEM, socket.SOCK_DGRAM, socket.SYSPROTO_CONTROL)
tun.connect("com.apple.net.utun_control")
ifname = tun.getsockopt(socket.SYSPROTO_CONTROL, UTUN_OPT_IFNAME, 32).rstrip(b"\0").decode()
subprocess.run(["ifconfig", ifname, "inet", SELF, PEER, "mtu", "1280", "up"], check=True)
for net in ("100.64.0.0/10", "192.0.2.0/24"):
    subprocess.run(["route", "-q", "-n", "delete", "-net", net], stderr=subprocess.DEVNULL)
    subprocess.run(["route", "-q", "-n", "add", "-net", net, "-interface", ifname], check=True)
with open(IFNAME_FILE, "w") as f:
    f.write(ifname + "\n")
log("up on %s: %s, peer %s" % (ifname, SELF, PEER))

ctl = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
ctl.bind(("127.0.0.1", CTL_PORT))
AF_HDR = struct.pack("!I", socket.AF_INET)
pending = []  # [kind, key, requester, deadline]


def send(pkt):
    tun.send(AF_HDR + pkt)


def answer(i, text):
    _, _, who, _ = pending.pop(i)
    ctl.sendto(text.encode(), who)


while True:
    r, _, _ = select.select([tun, ctl], [], [], 0.2)
    now = time.time()
    for i in range(len(pending) - 1, -1, -1):
        if pending[i][3] < now:
            answer(i, "none")
    if ctl in r:
        data, who = ctl.recvfrom(512)
        f = data.decode().split()
        if f and f[0] == "syn" and len(f) == 4:
            sport = random.randint(20000, 60000)
            pending.append(["syn", (f[2], int(f[3]), f[1], sport), who, now + 3])
            send(tcp_syn(f[1], f[2], sport, int(f[3])))
            log("inject SYN %s:%d -> %s:%s" % (f[1], sport, f[2], f[3]))
        elif f and f[0] == "ping" and len(f) == 3:
            ident = random.randint(1, 0xFFFF)
            pending.append(["ping", ident, who, now + 5])
            send(icmp_echo(f[1], f[2], 8, ident, 1, b"tailbond-e2e"))
            log("inject ping %s -> %s id %d" % (f[1], f[2], ident))
        elif f and f[0] == "addr":
            ctl.sendto(ifname.encode(), who)
        else:
            ctl.sendto(b"bad command", who)
    if tun in r:
        pkt = tun.recv(65535)[4:]
        if len(pkt) < 20 or pkt[0] >> 4 != 4:
            continue
        ihl = (pkt[0] & 15) * 4
        proto = pkt[9]
        src, dst = socket.inet_ntoa(pkt[12:16]), socket.inet_ntoa(pkt[16:20])
        body = pkt[ihl:]
        if proto == 1 and len(body) >= 8:
            typ, _, _, ident, seq = struct.unpack("!BBHHH", body[:8])
            if typ == 8 and dst == PEER:
                log("icmp echo from %s to %s" % (src, dst))
                send(icmp_echo(dst, src, 0, ident, seq, body[8:]))
            elif typ == 0:
                log("icmp reply from %s to %s id %d" % (src, dst, ident))
                for i, p in enumerate(pending):
                    if p[0] == "ping" and p[1] == ident:
                        answer(i, "reply " + src)
                        break
            else:
                log("icmp type %d from %s to %s" % (typ, src, dst))
        elif proto == 6 and len(body) >= 20:
            sport, dport = struct.unpack("!HH", body[:4])
            flags = body[13]
            kind = "synack" if flags & 0x12 == 0x12 else "rst" if flags & 0x04 else "tcp"
            log("%s %s:%d -> %s:%d" % (kind, src, sport, dst, dport))
            for i, p in enumerate(pending):
                if p[0] == "syn" and p[1] == (src, sport, dst, dport):
                    if kind == "synack":  # drop the half-open connection again
                        send(tcp_syn(dst, src, dport, sport, flags=0x04, seq=struct.unpack("!I", body[8:12])[0]))
                    answer(i, kind)
                    break
        else:
            log("proto %d from %s to %s" % (proto, src, dst))
