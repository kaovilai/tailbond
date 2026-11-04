import { readFileSync } from 'node:fs';
// Panel states for the GL admin screenshots (glinet-screens.mjs). Each state
// is what tailbondd's API returns through the "tailbond" RPC object on a
// Slate 7 (client) or Flint 3 (server). Shapes follow docs/architecture.md.

const now = Date.now();
const ago = (s) => new Date(now - s * 1000).toISOString();

const peers = {
  server: 'homebox', tailnet: 'alex@example.com', magic_dns_suffix: 'tail1234.ts.net', tailscale_running: true, client: 'slate7',
  self: { hostname: 'homebox', dns_name: 'homebox.tail1234.ts.net', tailscale_ips: ['100.101.1.1'], os: 'linux', online: true },
  peers: [
    { hostname: 'nas', dns_name: 'nas.tail1234.ts.net', tailscale_ips: ['100.101.2.2', 'fd7a:115c:a1e0::2'], os: 'linux', online: true, conn: 'direct' },
    { hostname: 'apple-tv', dns_name: 'apple-tv.tail1234.ts.net', tailscale_ips: ['100.101.4.4'], os: 'tvOS', online: true, conn: 'direct' },
    // bonded through tailbond, so its own Tailscale is off
    { hostname: 'pixel-8', dns_name: 'pixel-8.tail1234.ts.net', tailscale_ips: ['100.101.8.8'], os: 'android', online: false, last_seen: ago(3 * 3600) },
    { hostname: 'work-laptop', dns_name: 'work-laptop.tail1234.ts.net', tailscale_ips: ['100.101.3.3'], os: 'macOS', online: false, last_seen: ago(26 * 3600) },
  ],
};

// GET /api/paths on the Slate 7 at the hotel: Ethernet unplugged, repeater
// on hotel Wi-Fi, phone on USB tethering.
const slatePaths = (over = {}) => [
  { name: 'eth0', network: 'wan', label: 'Ethernet', kind: 'ethernet', wan: true, present: true, up: true, carrier: false, addrs: [], enabled: true, priority: 'primary', data_cap_mb: 0, used_bytes: 0, in_use: false, configured: false, ...(over.wan || {}) },
  { name: 'phy0-sta0', network: 'wwan', label: 'Repeater', kind: 'wifi', wan: true, present: true, up: true, carrier: true, addrs: ['192.168.20.57/24'], enabled: true, priority: 'primary', data_cap_mb: 0, used_bytes: 3.4e9, in_use: true, configured: false, ...(over.wwan || {}) },
  { name: 'usb0', network: 'tethering', label: 'Tethering', kind: 'usb', wan: true, present: true, up: true, carrier: true, addrs: ['192.168.42.129/24'], enabled: true, priority: 'secondary', data_cap_mb: 5000, used_bytes: 1.2e9, in_use: true, configured: true, ...(over.tethering || {}) },
];

const livePath = (iface, network, label, kind, state, rx, tx, rtt, extra = {}) =>
  ({ id: 0, iface, network, label, kind, state, srtt_ms: rtt, min_rtt_ms: rtt, loss_pct: 0, rx_bps: rx, tx_bps: tx, rx_bytes: 0, tx_bytes: 0, priority: 'primary', enabled: true, ...extra });

const clientStatus = (over = {}) => ({
  role: 'client', mode: 'speed', state: 'connected', server: 'homebox.tail1234.ts.net', tunnel_ip: '10.77.0.2', uptime_sec: 2 * 3600 + 13 * 60,
  totals: { tx_bps: 7.1e6, rx_bps: 59e6, tx_bytes: 0, rx_bytes: 0 }, paths: [], clients: [], paired: true, control_api: true, version: '0.1.0-r42',
  tailscale: { running: true, self: 'slate7', routes_approved: false, installed: true },
  ...over,
});

const settings = (over = {}) => ({ role: 'client', mode: 'speed', password_set: false, paired_with: 'home.example.net:443', paired_as: 'slate7', version: '0.1.0-r42', android_apk: false, ...over });

const twoWans = [
  livePath('phy0-sta0', 'wwan', 'Repeater', 'wifi', 'active', 38e6, 4.2e6, 24),
  livePath('usb0', 'tethering', 'Tethering', 'usb', 'active', 21e6, 2.9e6, 48, { priority: 'secondary', data_cap_mb: 5000, used_bytes: 1.2e9 }),
  livePath('eth0', 'wan', 'Ethernet', 'ethernet', 'failed', 0, 0, 0),
];

const notPaired = (extra = {}) => ({
  model: 'GL-BE3600',
  api: {
    // paths/clients as {}: how Lua cjson re-encodes an empty list
    'GET /api/status': clientStatus({ state: 'disconnected', paired: false, paths: {}, clients: {}, server: '', tunnel_ip: '', uptime_sec: 0, totals: { tx_bps: 0, rx_bps: 0 }, ...(extra.status || {}) }),
    'GET /api/settings': settings({ paired_with: undefined, paired_as: undefined }),
    'GET /api/paths': slatePaths(),
    ...(extra.api || {}),
  },
  hang: extra.hang || [],
});

const connected = (paths, over = {}, pathsOver = {}) => ({
  model: 'GL-BE3600',
  api: {
    'GET /api/status': clientStatus({ paths, ...over }),
    'GET /api/settings': settings(over.settings || {}),
    'GET /api/paths': slatePaths(pathsOver),
    'GET /api/peers': peers,
  },
});

// ---- server (Flint 3) ----
const serverStatus = (over = {}) => ({
  role: 'server', mode: 'speed', state: 'connected', server: 'homebox.tail1234.ts.net', tunnel_ip: '10.77.0.1', uptime_sec: 5 * 86400,
  totals: { tx_bps: 61e6, rx_bps: 7.4e6 }, paths: [], control_api: true, version: '0.1.0-r42',
  tailscale: { running: true, self: 'homebox', routes_approved: true, installed: true, dns_name: 'homebox.tail1234.ts.net' },
  clients: [
    { name: 'pixel-8', registered: true, connected: true, connected_sec: 3 * 3600, tx_bps: 12e6, rx_bps: 1.1e6, paths: [{ state: 'active' }, { state: 'active' }] },
    { name: 'ipad', registered: true, connected: false, connected_sec: 0, tx_bps: 0, rx_bps: 0, paths: [] },
  ],
  ...over,
});
// the phone page ("Get tailbond") on the server's Tailscale name
export const landing = { owner: 'alex@example.com', android_apk: true, landing_url: 'http://homebox.tail1234.ts.net/', landing_typed: 'homebox/',
  landing_qr_svg: readFileSync(new URL('./landing-qr.svg', import.meta.url), 'utf8') };
const serverBase = (statusOver = {}, tsOver = {}, setOver = {}) => ({
  model: 'GL-BE9300', role: 'server',
  info: { installed: true, version: '0.1.0-r42', role: 'server', enabled: true, running: true, model: 'GL.iNet GL-BE9300' },
  api: {
    'GET /api/status': serverStatus(statusOver),
    'GET /api/settings': { role: 'server', mode: 'speed', public_addr: 'home.example.net', effective_public_addr: 'home.example.net:443', port: 443, subnet: '10.77.0.0/24', tls: 'tailscale', tls_name: 'homebox.tail1234.ts.net', version: '0.1.0-r42', glinet_ipk: true, ...landing, ...setOver },
    'GET /api/tailscale': { installed: true, running: true, self: 'homebox', dns_name: 'homebox.tail1234.ts.net', tailnet: 'alex@example.com', route: '10.77.0.0/24', route_advertised: true, routes_approved: true, admin_url: 'https://login.tailscale.com/admin/machines', ...tsOver },
    'GET /api/peers': { ...peers, client: '' },
    'GET /api/health': healthGood,
  },
});

const withApi = (fx, api) => ({ ...fx, api: { ...fx.api, ...api } });

const pendingReq = { id: '4b79980f2bc88e95', name: 'slate7', kind: 'router', device: 'slate7', dns_name: 'slate7.tail1234.ts.net', hostname: 'GL-BE3600', os: 'linux', user: 'alex@example.com', ip: '100.101.9.9', status: 'pending', created: ago(40), expires: new Date(now + 560e3).toISOString() };

// GET /api/health on the Flint 3
const hc = (id, status, title, extra = {}) => ({ id, status, title, ...extra });
const healthGood = { all_good: true, direct: 3, relayed: 0, checks: [
  hc('tailscale_direct', 'ok', 'Tailscale: 3 machines direct'),
  hc('netcheck', 'ok', 'Network is friendly to direct connections'),
  hc('route', 'ok', 'Optional route 10.77.0.0/24 approved: your Tailscale machines can reach tailbond devices'),
  hc('cgnat', 'ok', 'Public internet address 203.0.113.7'),
  hc('port', 'ok', 'UDP 443 reached from the internet', { detail: 'pixel-8 connected from 198.51.100.20, 2 h ago.' }),
] };
// texts as the agent's health.go writes them
const directFix = "On your home router's admin page (Port forwarding, also called Virtual server or NAT), forward UDP port 41641 to this machine (192.168.1.2). (Turning on UPnP on that router also works, but lets any device on your network open ports.)";
const directDetail = (m) => `Through a relay: ${m}. Only traffic between this machine and that machine is slower (including bonded devices reaching it); bonding itself doesn't go through Tailscale. The cause can also be on that machine's own network.`;
const healthRelayed = { all_good: false, direct: 2, relayed: 1, relayed_peers: ['apple-tv'], checks: [
  hc('tailscale_direct', 'warn', 'Tailscale: 2 machines direct, 1 through a relay (slower)', { detail: directDetail('apple-tv'),
    fix: directFix, fix_url: 'https://tailscale.com/docs/reference/connection-types' }),
  hc('netcheck', 'warn', 'This network makes direct Tailscale connections hard', { detail: 'The router in front of this machine picks a new outside port for every destination ("hard NAT"), so other machines can\'t connect straight back; no port mapping on the router.',
    fix: directFix, fix_url: 'https://tailscale.com/docs/integrations/firewalls' }),
  hc('route', 'ok', 'Optional route 10.77.0.0/24 approved: your Tailscale machines can reach tailbond devices'),
  hc('cgnat', 'ok', 'Behind your home router (public address 203.0.113.7)', { detail: 'Forward UDP 443 on that router to 192.168.1.2.' }),
  hc('port', 'ok', 'UDP 443 reached from the internet', { detail: 'pixel-8 connected from 198.51.100.20, 3 h ago.' }),
] };
const healthCgnat = { all_good: false, direct: 3, relayed: 0, checks: [
  hc('tailscale_direct', 'ok', 'Tailscale: 3 machines direct'),
  hc('netcheck', 'ok', 'Network is friendly to direct connections'),
  hc('route', 'ok', 'Optional route 10.77.0.0/24 approved: your Tailscale machines can reach tailbond devices'),
  hc('cgnat', 'bad', 'Your internet line uses carrier-grade NAT', { detail: "This machine's internet address 100.72.3.4 is shared by your provider, so travel routers and phones away from home can't reach UDP 443 here. Bonding needs that direct connection; it can't run through Tailscale's relays (they'd make it slower than a single link).",
    fix: "Ask your provider for a public IPv4 address (often free or a small monthly fee; say it's for reaching your home from outside). Advanced: a public IPv6 address also works, but only from networks that have IPv6 too (most mobile networks, few hotels).", fix_url: 'https://tailscale.com/docs/reference/device-connectivity' }),
  hc('port', 'unknown', 'UDP 443: not confirmed from outside yet', { detail: "It can't be checked from inside your network; the first device that connects from away confirms it. Test it now: a phone with tailbond, Wi-Fi off (on mobile data), confirms it within a minute." }),
] };
const healthRoute = { all_good: false, direct: 3, relayed: 0, checks: [
  hc('tailscale_direct', 'ok', 'Tailscale: 3 machines direct'),
  hc('route', 'warn', "Route 10.77.0.0/24 isn't approved yet (optional)", { detail: 'Devices connected through tailbond reach your Tailscale machines without it. Approving it lets your Tailscale machines reach those devices too, under your access rules.', fix: 'If you want that: Tailscale admin console → Machines → this machine → Edit route settings (an autoApprovers rule in your policy approves it automatically). If you do not want it: leave it, or set "no_advertise_route": true in tailbond.json to stop advertising it.', fix_url: 'https://tailscale.com/docs/features/subnet-routers' }),
  hc('cgnat', 'unknown', 'Behind your home router', { detail: 'Forward UDP 443 on that router to 192.168.8.1.' }),
  hc('port', 'unknown', 'UDP 443: not confirmed from outside yet', { detail: "It can't be checked from inside your network; the first device that connects from away confirms it. Test it now: a phone with tailbond, Wi-Fi off (on mobile data), confirms it within a minute." }),
] };

export const states = {
  // 02: right after install from LuCI; Tailscale on; the panel is looking.
  '02-just-installed': notPaired({ hang: ['GET /api/discover'] }),
  '03-found-server': notPaired({ api: { 'GET /api/discover': { tailscale_installed: true, tailscale_running: true, self: 'slate7', servers: [{ name: 'homebox', dns_name: 'homebox.tail1234.ts.net', ip: '100.101.1.1', os: 'linux', version: '0.1.0-r42' }], checked: 4 } } }),
  '03b-tailscale-off': notPaired({ api: { 'GET /api/discover': { tailscale_installed: true, tailscale_running: false, servers: [], checked: 0 } } }),
  '03c-no-server-found': notPaired({ api: { 'GET /api/discover': { tailscale_installed: true, tailscale_running: true, servers: [], checked: 4 } } }),
  '04-waiting-approval': notPaired({
    status: { join: { state: 'waiting', server: '100.101.1.1:8077', server_name: 'homebox', id: '4b79980f2bc88e95', name: 'slate7', since: ago(20) } },
    api: { 'GET /api/discover': { tailscale_running: true, servers: [{ name: 'homebox', dns_name: 'homebox.tail1234.ts.net', ip: '100.101.1.1' }], checked: 4 } },
  }),
  '05-connected-two-wans': connected(twoWans),
  '06-one-wan-failed': connected([
    livePath('phy0-sta0', 'wwan', 'Repeater', 'wifi', 'active', 41e6, 4.6e6, 26),
    livePath('usb0', 'tethering', 'Tethering', 'usb', 'failed', 0, 0, 0, { priority: 'secondary', data_cap_mb: 5000, used_bytes: 1.2e9 }),
    livePath('eth0', 'wan', 'Ethernet', 'ethernet', 'failed', 0, 0, 0),
  ], { totals: { tx_bps: 4.6e6, rx_bps: 41e6 } }),
  '07-save-data-standby': connected([
    livePath('phy0-sta0', 'wwan', 'Repeater', 'wifi', 'active', 36e6, 3.9e6, 25),
    livePath('usb0', 'tethering', 'Tethering', 'usb', 'standby', 0, 0, 0, { priority: 'backup', data_cap_mb: 5000, used_bytes: 1.2e9 }),
    livePath('eth0', 'wan', 'Ethernet', 'ethernet', 'failed', 0, 0, 0),
  ], { totals: { tx_bps: 3.9e6, rx_bps: 36e6 } }, { tethering: { priority: 'backup', in_use: false } }),
  '08-server-unreachable': connected([
    livePath('phy0-sta0', 'wwan', 'Repeater', 'wifi', 'standby', 0, 0, 0),
    livePath('usb0', 'tethering', 'Tethering', 'usb', 'standby', 0, 0, 0, { priority: 'secondary', data_cap_mb: 5000, used_bytes: 1.2e9 }),
    livePath('eth0', 'wan', 'Ethernet', 'ethernet', 'failed', 0, 0, 0),
  ], { state: 'error', error: 'mqvpn exited: connection to home.example.net:443 timed out', uptime_sec: 0, tunnel_ip: '', totals: { tx_bps: 0, rx_bps: 0 } }),
  '09-turned-off': connected([
    livePath('phy0-sta0', 'wwan', 'Repeater', 'wifi', 'standby', 0, 0, 0),
    livePath('usb0', 'tethering', 'Tethering', 'usb', 'standby', 0, 0, 0, { priority: 'secondary', data_cap_mb: 5000, used_bytes: 1.2e9 }),
    livePath('eth0', 'wan', 'Ethernet', 'ethernet', 'failed', 0, 0, 0),
  ], { state: 'disconnected', uptime_sec: 0, tunnel_ip: '', totals: { tx_bps: 0, rx_bps: 0 }, settings: { disconnected: true } }, { tethering: { priority: 'backup', in_use: false } }),
  '10-not-running': { model: 'GL-BE3600', down: true },
  // Flint 3 at home
  '11-server-pending-request': serverBase({ pair_requests: [pendingReq] }),
  '12-server-approved': serverBase({
    clients: [
      { name: 'slate7', registered: true, connected: true, connected_sec: 25, tx_bps: 48e6, rx_bps: 6.1e6, paths: [{ state: 'active' }, { state: 'active' }] },
      { name: 'pixel-8', registered: true, connected: true, connected_sec: 3 * 3600, tx_bps: 12e6, rx_bps: 1.1e6, paths: [{ state: 'active' }, { state: 'active' }] },
      { name: 'ipad', registered: true, connected: false, connected_sec: 0, tx_bps: 0, rx_bps: 0, paths: [] },
    ],
  }),
  '18-server-health-relayed': withApi(serverBase(), { 'GET /api/health': healthRelayed, 'GET /api/peers': { ...peers, client: '', peers: peers.peers.map((m) => (m.hostname === 'apple-tv' ? { ...m, conn: 'relay', relay: 'fra' } : m)) } }),
  '18c-server-health-good': serverBase(),
  '18b-server-health-cgnat': withApi(serverBase({ clients: [
    { name: 'pixel-8', registered: true, connected: true, endpoint: '192.168.8.120:51234', connected_sec: 3 * 3600, tx_bps: 12e6, rx_bps: 1.1e6, paths: [{ state: 'active' }] },
    { name: 'slate7', registered: true, connected: false, connected_sec: 0, tx_bps: 0, rx_bps: 0, paths: [] },
  ] }), { 'GET /api/health': healthCgnat }),
  // hotel: the repeater is behind a portal, GL's repeater handles it
  '16-captive-gl-repeater': connected([
    livePath('usb0', 'tethering', 'Tethering', 'usb', 'active', 21e6, 2.9e6, 48, { priority: 'secondary', data_cap_mb: 5000, used_bytes: 1.2e9 }),
    livePath('phy0-sta0', 'wwan', 'Repeater', 'wifi', 'standby', 0, 0, 0, { captive: 'portal', portal_url: 'http://10.20.0.1/login', captive_source: 'glinet' }),
    livePath('eth0', 'wan', 'Ethernet', 'ethernet', 'failed', 0, 0, 0),
  ], { totals: { tx_bps: 2.9e6, rx_bps: 21e6 } }),
  '16b-captive-gl-login-mode': connected([
    // devices go out the hotel Wi-Fi directly meanwhile: the tunnel only carries the router's own trickle
    livePath('usb0', 'tethering', 'Tethering', 'usb', 'active', 0.012e6, 0.008e6, 48, { priority: 'secondary', data_cap_mb: 5000, used_bytes: 1.2e9 }),
    livePath('phy0-sta0', 'wwan', 'Repeater', 'wifi', 'standby', 0, 0, 0, { captive: 'portal', portal_url: 'http://10.20.0.1/login', captive_source: 'glinet', signin_until: new Date(now + 540e3).toISOString() }),
    livePath('eth0', 'wan', 'Ethernet', 'ethernet', 'failed', 0, 0, 0),
  ], { totals: { tx_bps: 0.008e6, rx_bps: 0.012e6 } }),
  // a hotel cable (Ethernet) with a sign-in page: tailbond's own sign-in
  '17-captive-own': connected([
    livePath('phy0-sta0', 'wwan', 'Repeater', 'wifi', 'active', 36e6, 3.9e6, 25),
    livePath('eth0', 'wan', 'Ethernet', 'ethernet', 'standby', 0, 0, 0, { captive: 'portal', portal_url: 'http://portal.hotel.example/login', captive_source: 'tailbond' }),
    livePath('usb0', 'tethering', 'Tethering', 'usb', 'standby', 0, 0, 0, { priority: 'backup', data_cap_mb: 5000, used_bytes: 1.2e9 }),
  ], { totals: { tx_bps: 3.9e6, rx_bps: 36e6 } }, { wan: { carrier: true, addrs: ['10.8.1.23/24'] }, tethering: { priority: 'backup', in_use: false } }),
  // devices go out the hotel cable directly meanwhile: the tunnel carries only the router's trickle
  '17b-captive-own-signing-in': connected([
    livePath('phy0-sta0', 'wwan', 'Repeater', 'wifi', 'active', 0.011e6, 0.007e6, 25),
    livePath('eth0', 'wan', 'Ethernet', 'ethernet', 'standby', 0, 0, 0, { captive: 'portal', portal_url: 'http://portal.hotel.example/login', captive_source: 'tailbond', signin_until: new Date(now + 560e3).toISOString() }),
    livePath('usb0', 'tethering', 'Tethering', 'usb', 'standby', 0, 0, 0, { priority: 'backup', data_cap_mb: 5000, used_bytes: 1.2e9 }),
  ], { totals: { tx_bps: 0.007e6, rx_bps: 0.011e6 } }, { wan: { carrier: true, addrs: ['10.8.1.23/24'] }, tethering: { priority: 'backup', in_use: false } }),
  '13-server-route-pending': withApi(serverBase({ clients: [] }, { routes_approved: false }, { public_addr: '', effective_public_addr: '192.168.8.1:443', public_addr_guessed: true, tls: 'self-signed' }), { 'GET /api/health': healthRoute }),
};

// Interactions after mount, for states that show a dialog or an open panel.
export const actions = {
  '05-connected-two-wans': null,
  '09b-unpair-confirm': { base: '05-connected-two-wans', click: 'button.tb-btn.danger' },
  '07b-set-limit': { base: '07-save-data-standby', clickText: 'Change limit' },
};

// The Fedora home server's own dashboard (agent web UI) with a request.
const srvPath = (id, rx, tx, rtt) => ({ id, iface: '', kind: 'unknown', state: 'active', srtt_ms: rtt, min_rtt_ms: rtt, loss_pct: 0, rx_bps: rx, tx_bps: tx, rx_bytes: 0, tx_bytes: 0, enabled: true });
export const fedoraStatus = {
  ...serverStatus({ pair_requests: [pendingReq] }), server: 'homebox.tail1234.ts.net:443',
  totals: { tx_bps: 12e6, rx_bps: 1.1e6, tx_bytes: 8.1e9, rx_bytes: 0.9e9 },
  clients: [
    { name: 'pixel-8', registered: true, connected: true, endpoint: '198.51.100.20:51234', scheduler: 'wlb', connected_sec: 3 * 3600, tx_bps: 12e6, rx_bps: 1.1e6, tx_bytes: 8.1e9, rx_bytes: 0.9e9, paths: [srvPath(0, 0.7e6, 8e6, 31), srvPath(1, 0.4e6, 4e6, 58)] },
    { name: 'ipad', registered: true, connected: false, connected_sec: 0, tx_bps: 0, rx_bps: 0, tx_bytes: 0, rx_bytes: 0, paths: [] },
  ],
};
export const fedoraHealth = healthRelayed;
export const fedoraHealthGood = healthGood;
export const fedoraHealthRoute = healthRoute;
// after tapping Approve: the request is gone, slate7 is a registered device
export const fedoraStatusApproved = {
  ...fedoraStatus, pair_requests: [],
  clients: [...fedoraStatus.clients, { name: 'slate7', registered: true, connected: false, connected_sec: 0, tx_bps: 0, rx_bps: 0, tx_bytes: 0, rx_bytes: 0, paths: [] }],
};
// a family member's phone asking (from the phone page)
export const fedoraStatusPhone = {
  ...fedoraStatusApproved,
  pair_requests: [{ ...pendingReq, id: '9c1d22e07a4f61b3', name: 'pixel-7a', kind: 'phone', device: 'pixel-7a', dns_name: 'pixel-7a.tail1234.ts.net', hostname: 'Pixel 7a', os: 'android', user: 'sam@example.com', ip: '100.101.7.7' }],
};
// after approving pixel-7a: it's on the device list, connected
export const fedoraStatusPhoneDone = {
  ...fedoraStatusApproved,
  clients: [...fedoraStatusApproved.clients, { name: 'pixel-7a', registered: true, connected: true, endpoint: '198.51.100.44:40112', scheduler: 'wlb', connected_sec: 40, tx_bps: 4.2e6, rx_bps: 0.3e6, tx_bytes: 2.1e7, rx_bytes: 1.5e6, paths: [srvPath(0, 0.2e6, 2.8e6, 28), srvPath(1, 0.1e6, 1.4e6, 61)] }],
};
// right after dnf install + systemctl enable --now: nothing paired yet
export const fedoraStatusFresh = { ...serverStatus({ clients: [] }), uptime_sec: 40, server: '192.168.1.2:443', clients: [], pair_requests: [], totals: { tx_bps: 0, rx_bps: 0, tx_bytes: 0, rx_bytes: 0 } };

// ---- bypass rules and more servers (failover) on the Slate 7 ----
// out/ux/bypass-failover/glinet/ (glinet-screens.sh bypass-failover)
const srv = (name, role, state, active, lastOkSec, subnet, server) =>
  ({ name, role, state, active, ...(lastOkSec == null ? {} : { last_ok: ago(lastOkSec) }), subnet, server });
const failover = (over = {}) => ({ policy: 'prefer', server: '', switch_after_sec: 15, failback_after_sec: 120, reason: '', ...over });
const oneServer = { servers: [srv('homebox', 'primary', 'active', true, 1, '10.77.0.0/24', 'home.example.net:443')], active_server: 'homebox', failover: failover() };
// One story on one clock (the panel's Date is fixed per screen, see `clock`):
// minute 2 homebox stops answering → vps; 23 back on homebox; 38 someone
// picks vps; 40.5 picks homebox again. Times shown (since, history,
// "answered … ago") all come from this timeline.
const T0 = Math.floor(now / 60000) * 60000 - 30 * 60000;
const at = (min) => new Date(T0 + min * 60000).toISOString();
const srvAt = (name, role, state, active, lastOkMin, subnet, server) => ({ name, role, state, active, ...(lastOkMin == null ? {} : { last_ok: at(lastOkMin) }), subnet, server });
const home = (state, active, lastOkMin) => srvAt('homebox', 'primary', state, active, lastOkMin, '10.77.0.0/24', 'home.example.net:443');
const vps = (state, active, lastOkMin) => srvAt('vps', 'backup', state, active, lastOkMin, '10.83.142.0/24', '203.0.113.9:8443');
const sw = (min, from, to, reason, detail) => ({ at: at(min), from, to, reason, ...(detail ? { detail } : {}) });
const h2 = sw(2, 'homebox', 'vps', 'failover', 'homebox stopped answering for 15 s');
const h23 = sw(23, 'vps', 'homebox', 'failback');
const h38 = sw(38, 'homebox', 'vps', 'manual');
const h40 = sw(40.5, 'vps', 'homebox', 'manual');
const onVpsAt = (st) => ({ server: '203.0.113.9:8443', tunnel_ip: '10.83.142.3', active_server: 'vps', ...st });
// minute 60: homebox in use, vps on standby; four switches so far
const homeAndVps = { uptime_sec: 19.5 * 60, servers: [home('active', true, 59.98), vps('standby', false, 59.92)], active_server: 'homebox', failover: failover({ since: at(40.5), history: [h40, h38, h23, h2] }) };
// minute 4: homebox stopped answering at 1:45; vps since 2
const onVps = onVpsAt({ uptime_sec: 120, servers: [home('unreachable', false, 1.75), vps('active', true, 3.98)], failover: failover({ reason: 'primary_down', since: at(2), history: [h2] }) });
// minute 21:40: homebox answers again, back in 1:20 (at 23)
const failingBack = onVpsAt({ uptime_sec: 19 * 60 + 40, servers: [home('standby', false, 21.6), vps('active', true, 21.65)], failover: failover({ reason: 'primary_down', since: at(2), failback_in_sec: 80, history: [h2] }) });
// minute 40: someone picked vps at 38
const pickedVps = onVpsAt({ uptime_sec: 120, servers: [home('standby', false, 39.9), vps('active', true, 39.98)], failover: failover({ reason: 'manual', since: at(38), history: [h38, h23, h2] }) });

const bpWans = [{ name: 'wan', label: 'Ethernet', up: false }, { name: 'wwan', label: 'Repeater', up: true, default: true }, { name: 'tethering', label: 'Tethering', up: true }];
const lanClients = [
  { mac: '3c:22:fb:10:aa:01', ip: '192.168.8.23', name: 'work-laptop', online: true },
  { mac: 'a4:83:e7:5c:21:9e', ip: '192.168.8.31', name: 'alex-iphone', online: true },
  { mac: 'f0:27:2d:8b:44:10', ip: '192.168.8.40', name: 'Fire-TV-Stick', online: true },
  { mac: '5e:91:0c:aa:73:2b', ip: '192.168.8.52', name: '', online: true },
  { mac: 'b8:27:eb:12:34:56', ip: '192.168.8.60', name: 'kindle', online: false },
];
const bpOk = { domains: 'nftset', firewall: 'nft' };
const bpRules = [
  { id: 'b3f9', kind: 'domain', match: 'mybank.com', target: 'direct', enabled: true, note: 'Bank' },
  { id: 'c1d2', kind: 'device', match: '3c:22:fb:10:aa:01', ip: '192.168.8.23', host: 'work-laptop', target: 'tunnel', enabled: true, note: 'work-laptop' },
  { id: 'd4e5', kind: 'device', match: 'f0:27:2d:8b:44:10', ip: '192.168.8.40', host: 'Fire-TV-Stick', target: 'direct', enabled: true, note: 'Fire-TV-Stick' },
  { id: 'a7b8', kind: 'port', match: 'udp/4500', target: 'direct', enabled: true, note: 'Wi-Fi calling' },
  { id: 'a7b9', kind: 'port', match: 'udp/500', target: 'direct', enabled: true, note: 'Wi-Fi calling' },
];
const bpCounters = { b3f9: { connections: 37 }, c1d2: { connections: 0 }, d4e5: { connections: 212 }, a7b8: { connections: 4 }, a7b9: { connections: 2 } };
const bypassApi = (rules, support = bpOk, counters = {}) => ({ 'GET /api/bypass': { rules, support, wans: bpWans, counters } });
// every client screen has the bypass card (the four rules unless a state says otherwise)
// the router is 10.77.0.3 (the phone screens use .2); work-laptop is on the LAN, so not in the tailnet list
const bfPeers = { ...peers, peers: peers.peers.filter((m) => m.hostname !== 'work-laptop') };
const bf = (statusOver = {}, api = {}) => withApi(connected(twoWans, { tunnel_ip: '10.77.0.3', ...oneServer, ...statusOver }),
  { 'GET /api/lan-clients': lanClients, 'GET /api/peers': bfPeers, ...bypassApi(bpRules, bpOk, bpCounters), ...api });
const discoverVps = { 'GET /api/discover': { tailscale_installed: true, tailscale_running: true, self: 'slate7', checked: 5, servers: [
  { name: 'homebox', dns_name: 'homebox.tail1234.ts.net', ip: '100.101.1.1', os: 'linux', version: '0.1.0-r42' },
  { name: 'vps', dns_name: 'vps.tail1234.ts.net', ip: '100.101.5.5', os: 'linux', version: '0.1.0-r42' },
] } };
const joinVps = (over) => ({ join: { server: '100.101.5.5:8077', server_name: 'vps', id: '7d0c3e55a1b2c3d4', name: 'slate7', since: ago(15), add: true, ...over } });

export const bfStates = {
  'bypass-empty': bf({}, bypassApi([])),
  'bypass-rules': bf({}, bypassApi(bpRules, bpOk, bpCounters)),
  'bypass-website-rules-wait': bf({}, bypassApi([
    { id: 'b3f9', kind: 'domain', match: 'mybank.com', target: 'direct', enabled: true, note: 'Bank' },
    { id: 'e1f2', kind: 'domain', match: 'netflix.com', target: 'direct', enabled: true, note: 'Streaming location' },
    bpRules[1],
  ], { domains: 'none', firewall: 'nft', reason: "This router's DNS server (dnsmasq) can't fill nftables sets, so website rules wait. Install dnsmasq-full; device, address and port rules work now." })),
  'servers-single-home': bf({}, discoverVps),
  'servers-home-active-vps-standby': { ...bf(homeAndVps), clock: at(60) },
  'failover-using-vps-home-unreachable': { ...bf(onVps), clock: at(4) },
  'failback-countdown': { ...bf(failingBack), clock: at(21 + 40 / 60) },
  'failover-you-picked-vps': { ...bf(pickedVps), clock: at(40) },
  // someone turned the kill switch off (UCI client.kill_switch=0)
  'servers-kill-switch-off': { ...bf(homeAndVps), clock: at(60), info: { installed: true, version: '0.1.0-r42', role: 'client', enabled: true, running: true, kill_switch: false } },
  'add-server-waiting-approval': bf(joinVps({ state: 'waiting' }), discoverVps),
  'add-server-subnet-clash': bf(joinVps({ state: 'error', error: 'subnet_in_use', conflict: 'homebox', subnet: '10.77.0.0/24' }), discoverVps),
  // the panel was open while homebox came back: one status on vps, then on homebox
  // the panel was open while homebox came back at minute 23 (clock 26)
  'back-on-homebox-since': { ...bf(homeAndVps), clock: at(26), settle: 2600, seq: { 'GET /api/status': [
    bf({ ...failingBack, servers: [home('standby', false, 25.9), vps('active', true, 25.95)], failover: failover({ reason: 'primary_down', since: at(2), failback_in_sec: 1, history: [h2] }) }).api['GET /api/status'],
    bf({ uptime_sec: 180, servers: [home('active', true, 25.98), vps('standby', false, 25.9)], active_server: 'homebox', failover: failover({ since: at(23), history: [h23, h2] }) }).api['GET /api/status'],
  ] } },
};
// steps: {text} clicks the first element with that text, {click} a selector,
// {button} a button by exact name, {fill: [selector, value]}.
export const bfActions = {
  'bypass-add-website-presets': { base: 'bypass-rules', steps: [{ text: 'Add a rule' }] },
  'bypass-add-streaming-preset': { base: 'bypass-rules', steps: [{ text: 'Add a rule' }, { click: '.tb-dialog >> text=Streaming location' }] },
  'bypass-add-device-picker': { base: 'bypass-rules', steps: [{ text: 'Add a rule' }, { click: '.tb-dialog >> text=A device' }, { click: '.tb-dialog >> text=alex-iphone' }] },
  'bypass-add-wifi-calling': { base: 'bypass-empty', steps: [{ click: '.tb-card button.tb-preset >> text=Wi-Fi calling' }] },
  // after clicking Add in that dialog: the card with the new row (and the toast)
  'bypass-wifi-calling-added': { base: 'bypass-empty', full: true, after: { 'POST /api/bypass': bypassApi(bpRules.slice(3), bpOk, { a7b8: { connections: 0 }, a7b9: { connections: 0 } }) },
    steps: [{ click: '.tb-card button.tb-preset >> text=Wi-Fi calling' }, { click: '.tb-dialog .tb-btn.primary' }, { wait: 500 }] },
  'servers-recent-switches-all': { base: 'servers-home-active-vps-standby', full: true, steps: [{ text: 'Show all (4)' }] },
  'add-server-open': { base: 'servers-single-home', full: true, steps: [{ text: 'Add a backup server' }] },
  'add-server-paste-clash': { base: 'servers-single-home', full: true, api: { 'POST /api/servers': { __err: 409, data: { ok: false, error: 'subnet_in_use', conflict: 'homebox', subnet: '10.77.0.0/24' } } },
    steps: [{ text: 'Add a backup server' }, { text: 'No Tailscale there? Paste a pairing link from it' }, { fill: ['.tb-sec textarea', 'tailbond://pair?d=eyJ2IjoxLCJzIjoiMjAzLjAuMTEzLjk6ODQ0MyJ9'] }, { button: 'Add' }] },
};
// also rendered at phone width
export const bfPhone = ['bypass-rules', 'bypass-add-device-picker', 'failover-using-vps-home-unreachable'];

// building blocks for other screen sets (ub-fixtures.mjs: updates and backups)
// ---- Pair & Share (Slate 7 at the hotel, phones on its Wi-Fi) ----
const sharePath = (name, label, extra = {}) => ({ name, network: '', label, kind: 'shared-cellular', wan: true, present: true, up: true, carrier: true, addrs: ['127.0.0.1/8'], enabled: true, priority: 'secondary', data_cap_mb: 0, used_bytes: 0, in_use: true, configured: false, ...extra });
const share = (id, label, state, extra = {}) => ({ id, label, path: 'sh' + id, addr: '192.168.8.' + (20 + id.length), via: 'mdns', state, up_bytes: 0, down_bytes: 0, battery: -1, charging: false, ...extra });
const sharesState = (paths, shares, over = {}, extraPaths = []) => {
  const fx = connected(paths, { shares, use_shares: true, ...over });
  fx.api['GET /api/paths'] = [...slatePaths(), ...extraPaths];
  return fx;
};
export const shareStates = {
  'share-01-two-phones': sharesState([
    livePath('phy0-sta0', 'wwan', 'Repeater', 'wifi', 'active', 22e6, 3.1e6, 24),
    livePath('sh3fa2c1', '', "Mobile data from sam-pixel", 'shared-cellular', 'active', 31e6, 4.4e6, 41, { priority: 'secondary' }),
    livePath('sh91be07', '', "Mobile data from dana-iphone", 'shared-cellular', 'active', 18e6, 2.2e6, 52, { priority: 'secondary' }),
    livePath('eth0', 'wan', 'Ethernet', 'ethernet', 'failed', 0, 0, 0),
  ], [
    share('3fa2c1', "Mobile data from sam-pixel", 'ready', { battery: 64, up_bytes: 0.3e9, down_bytes: 1.9e9 }),
    share('91be07', "Mobile data from dana-iphone", 'ready', { battery: 88, charging: true, up_bytes: 0.1e9, down_bytes: 0.6e9 }),
  ], { totals: { tx_bps: 9.7e6, rx_bps: 71e6 } }, [
    sharePath('sh3fa2c1', "Mobile data from sam-pixel", { used_bytes: 2.2e9 }),
    sharePath('sh91be07', "Mobile data from dana-iphone", { used_bytes: 0.7e9 }),
  ]),
  'share-02-low-battery': sharesState([
    livePath('phy0-sta0', 'wwan', 'Repeater', 'wifi', 'active', 22e6, 3.1e6, 24),
    livePath('sh91be07', '', "Mobile data from dana-iphone", 'shared-cellular', 'active', 18e6, 2.2e6, 52, { priority: 'secondary' }),
  ], [
    share('3fa2c1', "Mobile data from sam-pixel", 'refused', { battery: 18, reason: 'its battery is low' }),
    share('91be07', "Mobile data from dana-iphone", 'ready', { battery: 88, charging: true, up_bytes: 0.1e9, down_bytes: 0.6e9 }),
  ], {}, [sharePath('sh91be07', "Mobile data from dana-iphone", { used_bytes: 0.7e9 })]),
  'share-03-isolated': sharesState([
    livePath('phy0-sta0', 'wwan', 'Repeater', 'wifi', 'active', 22e6, 3.1e6, 24),
  ], [share('3fa2c1', "Mobile data from sam-pixel", 'unreachable', { via: 'server' })],
  { share_note: "This Wi-Fi blocks devices from talking to each other; sharing isn't available here." }),
  'share-04-none': sharesState([livePath('phy0-sta0', 'wwan', 'Repeater', 'wifi', 'active', 22e6, 3.1e6, 24)], []),
};

export { connected, serverBase, serverStatus, withApi, twoWans, peers as fxPeers };
