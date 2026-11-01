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
  hc('netcheck', 'ok', 'Network is friendly to direct connections', { detail: 'no port mapping on the router' }),
  hc('route', 'ok', 'Route 10.77.0.0/24 approved'),
  hc('cgnat', 'ok', 'Public internet address 203.0.113.7'),
  hc('port', 'ok', 'UDP 443 reached from the internet', { detail: 'pixel-8 connected from 198.51.100.20, 2 h ago.' }),
] };
// texts as the agent's health.go writes them
const directFix = "On the router in front of this machine, forward UDP port 41641 to it. (Turning on UPnP on that router also works, but lets any device on your network open ports.)";
const directDetail = (m) => `Through a relay: ${m}. Only traffic between this machine and that machine is slower (including bonded devices reaching it); bonding itself doesn't go through Tailscale. To check: tailscale ping ${m} ("via DERP" means relayed).`;
const healthRelayed = { all_good: false, direct: 2, relayed: 1, relayed_peers: ['apple-tv'], checks: [
  hc('tailscale_direct', 'warn', 'Tailscale: 2 machines direct, 1 through a relay (slower)', { detail: directDetail('apple-tv'),
    fix: directFix, fix_url: 'https://tailscale.com/docs/reference/connection-types' }),
  hc('netcheck', 'warn', 'This network makes direct Tailscale connections hard', { detail: 'The router in front of this machine picks a new outside port for every destination ("hard NAT"), so other machines can\'t connect straight back; no port mapping on the router.',
    fix: directFix, fix_url: 'https://tailscale.com/docs/integrations/firewalls' }),
  hc('route', 'ok', 'Route 10.77.0.0/24 approved'),
  hc('cgnat', 'ok', 'Behind your home router (public address 203.0.113.7)', { detail: 'Forward UDP 443 on that router to 192.168.1.2.' }),
  hc('port', 'ok', 'UDP 443 reached from the internet', { detail: 'pixel-8 connected from 198.51.100.20, 3 h ago.' }),
] };
const healthCgnat = { all_good: false, direct: 3, relayed: 0, checks: [
  hc('tailscale_direct', 'ok', 'Tailscale: 3 machines direct'),
  hc('netcheck', 'ok', 'Network is friendly to direct connections', { detail: 'no port mapping on the router' }),
  hc('route', 'ok', 'Route 10.77.0.0/24 approved'),
  hc('cgnat', 'bad', 'Your internet line uses carrier-grade NAT', { detail: "This machine's internet address 100.72.3.4 is shared by your provider, so travel routers and phones away from home can't reach UDP 443 here. Bonding needs that direct connection; it can't run through Tailscale's relays (they'd make it slower than a single link).",
    fix: "If your line has IPv6, enter this machine's public IPv6 address as the internet address (the router's status page shows it). Otherwise ask your provider for a public IPv4 address (often free or a small monthly fee).", fix_url: 'https://tailscale.com/docs/reference/device-connectivity' }),
  hc('port', 'unknown', 'UDP 443: not confirmed from outside yet', { detail: "It can't be checked from inside your network; the first device that connects from away confirms it." }),
] };
const healthRoute = { all_good: false, direct: 3, relayed: 0, checks: [
  hc('tailscale_direct', 'ok', 'Tailscale: 3 machines direct'),
  hc('route', 'warn', "Route 10.77.0.0/24 isn't approved yet", { fix: 'Approve it once in the Tailscale admin console (Machines → this machine → Edit route settings).', fix_url: 'https://tailscale.com/docs/features/subnet-routers' }),
  hc('cgnat', 'unknown', 'Behind your home router', { detail: 'Forward UDP 443 on that router to 192.168.8.1.' }),
  hc('port', 'unknown', 'UDP 443: not confirmed from outside yet', { detail: "It can't be checked from inside your network; the first device that connects from away confirms it." }),
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
  '17b-captive-own-signing-in': connected([
    livePath('phy0-sta0', 'wwan', 'Repeater', 'wifi', 'active', 36e6, 3.9e6, 25),
    livePath('eth0', 'wan', 'Ethernet', 'ethernet', 'standby', 0, 0, 0, { captive: 'portal', portal_url: 'http://portal.hotel.example/login', captive_source: 'tailbond', signin_until: new Date(now + 560e3).toISOString() }),
    livePath('usb0', 'tethering', 'Tethering', 'usb', 'standby', 0, 0, 0, { priority: 'backup', data_cap_mb: 5000, used_bytes: 1.2e9 }),
  ], { totals: { tx_bps: 3.9e6, rx_bps: 36e6 } }, { wan: { carrier: true, addrs: ['10.8.1.23/24'] }, tethering: { priority: 'backup', in_use: false } }),
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
