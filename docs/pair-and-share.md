# Pair & Share

Add your phones' mobile data to the travel router's bond, like Speedify's
Pair & Share. Your Android phone, your partner's Android phone and your iPhone
each lend their 5G/LTE over the hotel Wi-Fi to the **hub**, which bonds them
with its own Wi-Fi and Ethernet in one tunnel to your home server.

- **Hub** (bonds): the travel router (GL.iNet Slate 7 or another OpenWrt
  router running tailbond in the client role), or a Linux laptop running
  tailbondd in the client role.
- **Sharer** (lends): tailbond on Android (**Share my mobile data**), the
  **tailbond Share** app on iPhone ([ios.md](ios.md)), or a Linux machine with
  an LTE dongle (`client.share`, below).

Only devices paired with the **same home server** can share with each other.
Removing a device on the home server (Clients → remove) also ends its sharing,
both ways, within two days at most.

## At a hotel, step by step

1. Plug in the travel router and connect it to the hotel Wi-Fi as usual
   (GL admin → INTERNET → Repeater). tailbond bonds it.
2. Join your phones to the **travel router's own Wi-Fi** (the one it makes),
   not the hotel's.
3. Android: tailbond → Settings → **Share my mobile data** → switch on.
   iPhone: open **tailbond Share** → **Share mobile data**, and keep the app
   open on the screen (iOS pauses it otherwise; plug the phone in). Limits you
   change apply at once.
4. Within a few seconds each phone shows up in the GL panel (VPN → tailbond)
   under **Shared by nearby phones**, and as a connection labelled e.g.
   *Mobile data from sam-pixel* under **Internet connections**, with the
   usual **Use it / Backup only / Off** and a monthly data limit.

The phone shows *Sharing mobile data with slate7: 120 MB* in its
notification (Android) or on screen (iPhone).

**Low battery:** each phone stops sharing below a battery level you pick
(default 20 %) unless it's charging, and starts again by itself when you plug
it in. **Data limit:** each phone has its own monthly limit too, and the hub's
per-connection limit applies on top.

**If the hotel Wi-Fi keeps devices apart** (client isolation), phones on the
*hotel's* Wi-Fi can't reach a laptop hub there. tailbond says so: *This Wi-Fi
blocks devices from talking to each other; sharing isn't available here.*
With the travel router this doesn't happen: phones join the router's own
Wi-Fi. Otherwise use USB tethering (below).

## USB tethering is already bonded

A phone plugged into the travel router (or a laptop) with **USB tethering**
on, or a phone hotspot the router joins as its repeater uplink, is just another
WAN: tailbond bonds it with no app on the phone. An iPhone over USB shows as
**iPhone (USB)** (Apple's `ipheth` driver); Android shows as Tethering. USB
also charges the phone, and works where hotel Wi-Fi isolates devices.

## A Linux machine sharing its uplink

A laptop with an LTE dongle (or any Linux box) can be a sharer. In its
`tailbond.json` (client role, paired with the same server):

```json
"client": {
  "share": {"enabled": true, "iface": "wwan0", "label": "ThinkPad LTE", "cap_mb": 5000, "allowed_hubs": ["slate7"]}
}
```

or `POST /api/share {"share": {...}}`. It relays over `iface` only
(`SO_BINDTODEVICE`), advertises itself over mDNS and registers with the home
server while connected. `"use_shares": false` stops a client from using
other devices' shares.

## How it works

```
 iPhone / Android / Linux (sharer)                      hub (travel router, laptop)
 ┌──────────────────────────┐   LAN (Wi-Fi)   ┌──────────────────────────────────┐
 │ relay  UDP :47077  ◄─────┼─────────────────┼── share.Link  ◄── mqvpn relay path│
 │   │ HMAC session         │  DATA frames    │   127.0.0.1:port   sh3fa2c1@…     │
 │   ▼ cellular only        │                 │                  + eth0, wlan0 …  │
 └───┼──────────────────────┘                 └──────────────────────────────────┘
     ▼ UDP to the home server (mqvpn sees a new QUIC path from the phone's carrier IP)
```

1. **Discovery.** Sharers advertise `_tailbond-share._udp` over mDNS
   (Android NSD, iOS Bonjour, the agent's own responder) with TXT `v=1`,
   `id`, `n` (label) and `k` (a public tag of the sharer's share key, so a hub
   ignores other servers' sharers). Sharers that are connected also register
   their LAN address with the home server (`POST /api/shares`); a hub lists
   them (`GET /api/shares`) and uses those on one of its own subnets. That
   catches networks where multicast doesn't pass, and it's how the hub tells
   client isolation apart: the server lists a sharer on our subnet, mDNS never
   saw it, and it never answers.
2. **Session.** Each paired device has its **own share key**, which comes
   in its pairing (`share_key`; older pairings get it from `GET /api/shares`
   with the client token). The hub asks its home server for a **pass** for
   each nearby sharer: derived from that sharer's key, the hub's own name and
   an expiry (the end of tomorrow). The sharer derives the same pass from its
   key and what the hub says in its hello, so it checks the hub without
   reaching the server ([protocol](architecture.md#pair--share)). A hub
   can't pass itself off as another one (so **allowed hubs** really means
   those devices), and a removed device gets no new passes. The sharer
   forwards only to **its own home server**, so it isn't an open relay;
   frames from any address but the session's hub are dropped.
3. **Relay.** The hub's mqvpn runs the share as a **relay path**: it sends
   that path's QUIC datagrams to the hub agent on `127.0.0.1`, which frames
   them (9 bytes: type + session) to the sharer, which sends the datagram
   over cellular to the home server and frames the replies back. The home
   server's mqvpn sees one more path of the hub's connection, from the
   sharer's carrier address, like any NAT rebinding.
4. **Health.** The hub sends a keepalive a second; the sharer answers with
   its byte counters, battery, charging state and how much of its cap is
   left. A share that stops answering for 3.5 s is taken out of the path set
   at once (mqvpn can't tell a dead relay from a live one by itself), and comes
   back when the sharer does.

### Why a small mqvpn patch, not nftables

Two ways to make mqvpn send one path somewhere other than the server:

- **(a) Patch mqvpn** so a path can have its own peer address:
  `--path NAME@HOST:PORT` (`patches/mqvpn/0002-relay-paths.patch`). The
  POSIX bind sends that path's datagrams to `HOST:PORT` and hands what comes
  back to the core as if from the server; the slot isn't pinned to an
  interface. About 150 lines, no change to the core or xquic, unit tested
  over loopback, and general enough to upstream (any UDP relay or proxy).
- **(b) nftables DNAT** of a marked path to the relay through a dummy/veth
  interface. mqvpn pins every path socket to its device (`SO_BINDTODEVICE`),
  so after DNAT the kernel would still have to route through that device;
  it needs a veth pair or a namespace per share, policy routes for the server
  address that all paths share, conntrack state per share, and still no place
  to add the session framing and authentication the relay needs. It would
  differ between OpenWrt's firewall4 and a Linux client's nftables, and is
  hard to test.

(a) is smaller, the same on OpenWrt and Linux, and keeps the relay protocol
(authentication, framing, counters, caps) in Go, where it is tested next to
the sharer. The Android SDK fork doesn't need the patch: a phone only shares,
it never bonds a share.

## Limits and what isn't tested

- Hubs are Linux (OpenWrt or a Linux client). A phone running tailbond
  doesn't bond other phones' shares.
- iPhone sharing pauses whenever tailbond Share leaves the screen (iOS
  suspends it): free signing can't use background VPN/Network Extension
  modes.
- The emulator CI job checks that Android's share service comes up next to
  the VPN (mobile data held, UDP 47077 open, the tunnel stays up); it can't
  put a hub on the emulator's virtual Wi-Fi, so the Android relay's traffic is
  covered by unit tests over loopback (the same frames as Go, pinned by
  `agent/internal/share/testdata/vectors.json`). The full hub ↔ sharer ↔
  server path runs in `scripts/e2e-share.sh` with the Linux sharer.
- Not yet run on real phones and routers: Android's behaviour with
  `bindSocket`/`protect` while our own VPN runs, iOS Local Network
  permission prompts and Bonjour on real Wi-Fi, GL firmware's firewall with
  LAN-side UDP to the router.
