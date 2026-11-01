// Panel states for the GL admin screenshots (glinet-screens.mjs). Each state
// is what tailbondd's API returns through the "tailbond" RPC object on a
// Slate 7 (client) or Flint 3 (server). Shapes follow docs/architecture.md.

const now = Date.now();
const ago = (s) => new Date(now - s * 1000).toISOString();

const peers = {
  server: 'homebox', tailnet: 'alex@example.com', magic_dns_suffix: 'tail1234.ts.net', tailscale_running: true, client: 'slate7',
  self: { hostname: 'homebox', dns_name: 'homebox.tail1234.ts.net', tailscale_ips: ['100.101.1.1'], os: 'linux', online: true },
  peers: [
    { hostname: 'nas', dns_name: 'nas.tail1234.ts.net', tailscale_ips: ['100.101.2.2', 'fd7a:115c:a1e0::2'], os: 'linux', online: true },
    { hostname: 'apple-tv', dns_name: 'apple-tv.tail1234.ts.net', tailscale_ips: ['100.101.4.4'], os: 'tvOS', online: true },
    { hostname: 'pixel-8', dns_name: 'pixel-8.tail1234.ts.net', tailscale_ips: ['100.101.8.8'], os: 'android', online: true },
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
const serverBase = (statusOver = {}, tsOver = {}, setOver = {}) => ({
  model: 'GL-BE9300', role: 'server',
  info: { installed: true, version: '0.1.0-r42', role: 'server', enabled: true, running: true, model: 'GL.iNet GL-BE9300' },
  api: {
    'GET /api/status': serverStatus(statusOver),
    'GET /api/settings': { role: 'server', mode: 'speed', public_addr: 'home.example.net', effective_public_addr: 'home.example.net:443', port: 443, subnet: '10.77.0.0/24', tls: 'tailscale', tls_name: 'homebox.tail1234.ts.net', version: '0.1.0-r42', ...setOver },
    'GET /api/tailscale': { installed: true, running: true, self: 'homebox', dns_name: 'homebox.tail1234.ts.net', tailnet: 'alex@example.com', route: '10.77.0.0/24', route_advertised: true, routes_approved: true, admin_url: 'https://login.tailscale.com/admin/machines', ...tsOver },
    'GET /api/peers': { ...peers, client: '' },
  },
});

const pendingReq = { id: '4b79980f2bc88e95', name: 'slate7', kind: 'router', device: 'slate7', dns_name: 'slate7.tail1234.ts.net', hostname: 'GL-BE3600', os: 'linux', user: 'alex@example.com', ip: '100.101.9.9', status: 'pending', created: ago(40), expires: new Date(now + 560e3).toISOString() };

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
  ], { state: 'disconnected', uptime_sec: 0, tunnel_ip: '', totals: { tx_bps: 0, rx_bps: 0 }, settings: { disconnected: true } }),
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
  '13-server-route-pending': serverBase({ clients: [] }, { routes_approved: false }, { public_addr: '', effective_public_addr: '192.168.8.1:443', public_addr_guessed: true, tls: 'self-signed' }),
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
