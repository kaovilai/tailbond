// Updates and backup/restore screens (updates-backup-screens.mjs): the GL
// panel (Slate 7 travel router, Flint 3 home server) and the agent's own web
// dashboard (Fedora home server "homebox", a VPS "vps"). Shapes follow
// docs/architecture.md "Updates" and "Backup and restore": the status
// object `update` {current, available, notes_url, policy, repo, method,
// state, last_check, last_result, last_result_version, last_result_at,
// detail, error, outdated} and `clients[].version/kind`. Scheduled-backup
// settings are read from GET /api/settings `backup` (passphrase_set,
// every_hours, keep, push, last, last_push, last_error).
//
// One clock for every screen: today 22:00 (UTC). homebox updated to 0.2.0
// at 03:06 this morning.
import { connected, serverBase, serverStatus, withApi, twoWans, landing } from './fixtures.mjs';

const T = Math.floor(Date.now() / 864e5) * 864e5 + 22 * 3600e3;
export const clock = new Date(T).toISOString();
const hAgo = (hrs) => new Date(T - hrs * 3600e3).toISOString();
const notes = (v) => `https://github.com/kaovilai/tailbond/releases/tag/v${v}`;

// method: opkg (GL routers), static (the Linux bundle on the VPS), dnf
// (the Fedora RPM on homebox)
const upd = (over = {}) => ({ current: '0.1.0', available: '', notes_url: '', policy: 'notify', repo: 'kaovilai/tailbond', method: 'opkg', state: 'idle', last_check: hAgo(1.5),
  last_result: '', last_result_version: '', last_result_at: '', detail: '', error: '', ...over });
const avail = (over = {}) => upd({ available: '0.2.0', notes_url: notes((over.available) || '0.2.0'), ...over });
// updated to 0.2.0 at 03:06 this morning
const went = { last_result: 'ok', last_result_version: '0.2.0', last_result_at: hAgo(18.9) };

// ---- GL travel router (Slate 7) ----
const sched = { passphrase_set: true, every_hours: 24, keep: 7, push: true, last: hAgo(5), last_push: hAgo(5) };
const router = (u, bk = sched) => {
  const fx = connected(twoWans, { update: u, settings: { backup: bk } });
  return { ...fx, clock, info: { installed: true, version: u.current === '0.2.0' ? '0.2.0-r1' : '0.1.0-r42', role: 'client', enabled: true, running: true, kill_switch: true } };
};
const routerPreview = {
  role: 'client', host: 'slate7', created: hAgo(29), agent_version: '0.2.0',
  changes: [
    { what: 'Servers', from: '(none)', to: 'homebox, vps' },
    { what: 'Bypass rules', from: '0', to: '4' },
    { what: 'Connection settings', from: '0', to: '3' },
    { what: 'Update policy', from: 'auto', to: 'notify' },
  ],
  warnings: [
    "This backup is from another router (slate7). Both can't use it at once: if the old one still runs, turn tailbond off there first, or it and this router keep pushing each other off homebox.",
    "Tailscale isn't part of the backup. To find servers by name again, sign this router in to Tailscale (APPLICATIONS → Tailscale); bonding works without it.",
  ],
};
const routerApi = (u, extra = {}) => ({
  'GET /api/update': u,
  'GET /api/backup/summary': { role: 'client', mode: 'speed', servers: ['homebox (home.example.net:443)', 'vps (203.0.113.9:8443)'], bypass_rules: 4, paths: 3, password_set: false, update_policy: 'notify', files: ['global.key'], uci: true },
  'POST /api/restore/preview': routerPreview,
  'POST /api/restore': { ok: true, rollback: 'before-restore-2026-11-02T2201.tbbackup' },
  ...extra,
});
const r = (u, bk) => withApi(router(u, bk), routerApi(u));

export const glStates = {
  'r01-router-up-to-date': r(upd({ current: '0.2.0', ...went })),
  'r02-router-update-available': r(avail()),
  'r03-router-installing': r(avail({ state: 'installing' })),
  'r04-router-rolled-back': r(avail({ state: 'rolled_back', last_result: 'rolled_back', last_result_version: '0.2.0', last_result_at: hAgo(0.2), detail: "the tunnel didn't come back within 60 seconds" })),
  'r05-router-automatic': r(upd({ current: '0.2.0', policy: 'auto', ...went })),
  'r06-router-reinstalled-after-firmware-upgrade': r(upd({ current: '0.2.0', last_result: 'ok', last_result_version: '0.2.0', last_result_at: hAgo(0.4), detail: 'reinstalled_after_firmware_upgrade' })),
  'r08-router-backups-off': r(upd({ current: '0.2.0', ...went }), { passphrase_set: false, every_hours: 24, keep: 7, push: true }),
};

// ---- GL Flint 3 as the home server ----
const flintClients = [
  { name: 'slate7', registered: true, connected: true, connected_sec: 5400, tx_bps: 18e6, rx_bps: 2.1e6, paths: [{ state: 'active' }, { state: 'active' }], version: '0.1.0', kind: 'router' },
  { name: 'pixel-8', registered: true, connected: true, connected_sec: 3 * 3600, tx_bps: 12e6, rx_bps: 1.1e6, paths: [{ state: 'active' }, { state: 'active' }], version: '0.1.0', kind: 'android' },
  { name: 'beryl', registered: true, connected: false, connected_sec: 0, tx_bps: 0, rx_bps: 0, paths: [], version: '0.2.0', kind: 'router' },
];
const outdated = [{ name: 'slate7', kind: 'router', version: '0.1.0' }, { name: 'pixel-8', kind: 'android', version: '0.1.0' }];
const flintBackups = [
  { name: 'homebox-2026-11-02T0300.tbbackup', client: '', size: 11200, created: hAgo(19) },
  { name: 'homebox-2026-11-01T0300.tbbackup', client: '', size: 11150, created: hAgo(43) },
  { name: 'slate7-2026-11-02T1700.tbbackup', client: 'slate7', size: 6100, created: hAgo(5) },
  { name: 'beryl-2026-10-30T2210.tbbackup', client: 'beryl', size: 5900, created: hAgo(72) },
];
const flint = (u) => {
  const fx = serverBase({ clients: flintClients, update: u }, {}, { backup: { passphrase_set: true, every_hours: 24, keep: 7, last: hAgo(19) } });
  return { ...withApi(fx, {
    'GET /api/update': u,
    'GET /api/backups': flintBackups,
    'GET /api/backup/summary': { role: 'server', mode: 'speed', clients: ['slate7', 'pixel-8', 'beryl'], public_addr: 'home.example.net', port: 443, subnet: '10.77.0.0/24', approvers: null, password_set: false, update_policy: 'notify', files: ['global.key', 'server.crt', 'server.key'], uci: true },
  }), clock, model: 'GL-BE9300', info: { ...fx.info, version: u.current === '0.2.0' ? '0.2.0-r1' : '0.1.0-r42' } };
};
glStates['s01-flint-devices-out-of-date'] = flint(upd({ current: '0.2.0', ...went, outdated }));

// steps: {text} clicks the first element with that text, {click} a selector,
// {button} a button by exact name, {fill: [selector, value]},
// {files: [selector, [[name, size], ...]]} picks files.
const pass = 'maple tunnel violet seven';
export const glActions = {
  'r09-router-backup-download-dialog': { base: 'r01-router-up-to-date', steps: [{ button: 'Download a backup' }, { fill: ['.tb-dialog input[type=password] >> nth=0', pass] }, { fill: ['.tb-dialog input[type=password] >> nth=1', pass] }] },
  'r10-router-restore-preview': { base: 'r08-router-backups-off', steps: [{ button: 'Restore from a backup' }, { files: ['.tb-dialog input.tb-file', [['slate7-2026-11-01.tbbackup', 6100]]] }, { fill: ['.tb-dialog input[type=password]', pass] }, { button: 'Check the backup' }, { wait: 300 }] },
  'r11-router-restore-done': { base: 'r08-router-backups-off', steps: [{ button: 'Restore from a backup' }, { files: ['.tb-dialog input.tb-file', [['slate7-2026-11-01.tbbackup', 6100]]] }, { fill: ['.tb-dialog input[type=password]', pass] }, { button: 'Check the backup' }, { wait: 300 }, { button: 'Restore' }, { wait: 300 }] },
  'r07-router-update-confirm': { base: 'r02-router-update-available', steps: [{ button: 'Update now' }] },
  'r13-router-whats-inside': { base: 'r01-router-up-to-date', steps: [{ button: "See what's inside" }, { wait: 200 }] },
};
// also rendered at phone width (390x844)
export const glPhone = ['r02-router-update-available', 'r09-router-backup-download-dialog', 'r10-router-restore-preview', 's01-flint-devices-out-of-date'];

// ---- the agent's own dashboard (agent/web): Fedora "homebox", a VPS ----
const srvPath = (id, rx, tx, rtt) => ({ id, iface: '', kind: 'unknown', state: 'active', srtt_ms: rtt, min_rtt_ms: rtt, loss_pct: 0, rx_bps: rx, tx_bps: tx, rx_bytes: 0, tx_bytes: 0 });
const webClients = [
  { name: 'slate7', registered: true, connected: true, endpoint: '198.51.100.23:40112', scheduler: 'wlb', connected_sec: 5400, tx_bps: 18e6, rx_bps: 2.1e6, tx_bytes: 0, rx_bytes: 0, paths: [srvPath(0, 1.2e6, 11e6, 31), srvPath(1, 0.9e6, 7e6, 48)], version: '0.1.0', kind: 'router' },
  { name: 'pixel-8', registered: true, connected: true, endpoint: '203.0.113.77:51820', scheduler: 'wlb', connected_sec: 3 * 3600, tx_bps: 6e6, rx_bps: 0.4e6, tx_bytes: 0, rx_bytes: 0, paths: [srvPath(0, 0.4e6, 6e6, 40)], version: '0.1.0', kind: 'android' },
  { name: 'beryl', registered: true, connected: false, connected_sec: 0, tx_bps: 0, rx_bps: 0, tx_bytes: 0, rx_bytes: 0, paths: [], version: '0.2.0', kind: 'router' },
];
const webStatus = (u, over = {}) => ({
  ...serverStatus({ clients: webClients }), totals: { tx_bps: 24e6, rx_bps: 2.5e6, tx_bytes: 8.1e9, rx_bytes: 0.9e9 }, pair_requests: [],
  version: u.current, update: u, ...over,
});
const webSettings = (over = {}) => ({ role: 'server', mode: 'speed', public_addr: 'home.example.net', effective_public_addr: 'home.example.net:443', port: 443, subnet: '10.77.0.0/24', tls: 'tailscale', tls_name: 'homebox.tail1234.ts.net', version: '0.2.0', glinet_ipk: true, password_set: false, ...landing,
  backup: { passphrase_set: true, every_hours: 24, keep: 7, last: hAgo(19) }, ...over });
const webBackups = [
  { name: 'homebox-2026-11-02T0300.tbbackup', client: '', size: 11200, created: hAgo(19) },
  { name: 'homebox-2026-11-01T0300.tbbackup', client: '', size: 11150, created: hAgo(43) },
  { name: 'homebox-2026-10-31T0300.tbbackup', client: '', size: 11150, created: hAgo(67) },
  { name: 'slate7-2026-11-02T1700.tbbackup', client: 'slate7', size: 6100, created: hAgo(5) },
  { name: 'slate7-2026-11-01T1700.tbbackup', client: 'slate7', size: 6100, created: hAgo(29) },
  { name: 'beryl-2026-10-30T2210.tbbackup', client: 'beryl', size: 5900, created: hAgo(72) },
];
const webSummary = { role: 'server', mode: 'speed', clients: ['slate7', 'pixel-8', 'beryl'], public_addr: 'home.example.net', port: 443, subnet: '10.77.0.0/24', approvers: ['sam@example.com'], password_set: false, update_policy: 'notify', files: ['global.key', 'server.crt', 'server.key'], uci: false };
// a new VPS (198.51.100.20) getting the old one's backup (vps, 203.0.113.9)
const vpsPreview = {
  role: 'server', host: 'vps', created: hAgo(30), agent_version: '0.2.0',
  changes: [
    { what: 'Devices (clients)', from: '(none)', to: 'slate7, pixel-8, beryl' },
    { what: 'Public address', from: '(none)', to: '203.0.113.9' },
    { what: 'Tunnel port', from: '443', to: '8443' },
    { what: 'Tunnel addresses', from: '10.77.0.0/24', to: '10.83.142.0/24' },
    { what: 'Approvers', from: '(none)', to: 'sam@example.com' },
  ],
  warnings: [
    'This backup was made on another machine with the public address 203.0.113.9; this one is reached at 198.51.100.20. After restoring, set the new public address (Devices tab). Devices then pick it up by themselves within a few hours, because vps has a Tailscale name (vps.tail1234.ts.net).',
    'Devices that were paired with a link and have no Tailscale (a router without Tailscale, a phone added by QR code): send them a new link once the address is set (Devices tab, the QR button next to each).',
  ],
};
const webApi = (u, extra = {}) => ({
  'GET /api/update': u,
  'GET /api/backups': webBackups,
  'GET /api/backup/summary': webSummary,
  'POST /api/restore/preview': vpsPreview,
  'POST /api/restore': { ok: true, rollback: 'before-restore-2026-11-02T2201.tbbackup' },
  ...extra,
});
const webVps = (u) => ({ status: webStatus(u, { tailscale: { running: true, self: 'vps', routes_approved: true, installed: true, dns_name: 'vps.tail1234.ts.net' }, server: 'vps.tail1234.ts.net' }),
  settings: webSettings({ public_addr: '198.51.100.20', effective_public_addr: '198.51.100.20:8443', port: 8443, tls_name: 'vps.tail1234.ts.net' }), api: webApi(u) });
const webHome = (u, setOver) => ({ status: webStatus(u), settings: webSettings(setOver), api: webApi(u) });

// homebox: Fedora RPM (dnf), on 0.2.0; 0.2.1 is out
const homeU = (over = {}) => upd({ method: 'dnf', current: '0.2.0', ...went, outdated, ...over });
// vps: the Linux bundle (static), on 0.1.0
const vpsU = (over = {}) => upd({ method: 'static', outdated: [], ...over });
export const webStates = {
  'w01-homebox-dnf-update-available': webHome(homeU({ available: '0.2.1', notes_url: notes('0.2.1') })),
  'w02-updated-devices-out-of-date-backups': webHome(homeU()),
  'w03-vps-update-available': webVps(vpsU({ available: '0.2.0', notes_url: notes('0.2.0') })),
  'w08-vps-rolled-back': webVps(vpsU({ available: '0.2.0', notes_url: notes('0.2.0'), state: 'rolled_back', last_result: 'rolled_back', last_result_version: '0.2.0', last_result_at: hAgo(0.3), detail: "the service wasn't up on 0.2.0 within 60 seconds" })),
  'w09-vps-check-failed': webVps(vpsU({ last_check: hAgo(0.1), error: 'api.github.com: i/o timeout' })),
  'w10-vps-new-machine': { ...webVps(vpsU({ current: '0.2.0', ...went })), settings: webSettings({ public_addr: '198.51.100.20', effective_public_addr: '198.51.100.20:443', backup: { passphrase_set: false, every_hours: 24, keep: 7 } }) },
  'w13-vps-updates-off': webVps(vpsU({ policy: 'off', state: 'off', last_check: '' })),
};
export const webActions = {
  'w04-vps-update-confirm': { base: 'w03-vps-update-available', steps: [{ button: 'Update now' }] },
  'w05-backup-download-dialog': { base: 'w02-updated-devices-out-of-date-backups', steps: [{ button: 'Download a backup' }, { fill: ['#bk-pass', pass] }, { fill: ['#bk-pass2', pass] }] },
  'w06-restore-preview-new-vps': { base: 'w10-vps-new-machine', steps: [{ button: 'Restore from a backup' }, { files: ['#rs-file', [['vps-2026-11-01T0300.tbbackup', 11200]]] }, { fill: ['#rs-pass', pass] }, { button: 'Check the backup' }, { wait: 300 }] },
  'w07-restore-done': { base: 'w10-vps-new-machine', steps: [{ button: 'Restore from a backup' }, { files: ['#rs-file', [['vps-2026-11-01T0300.tbbackup', 11200]]] }, { fill: ['#rs-pass', pass] }, { button: 'Check the backup' }, { wait: 300 }, { button: 'Restore' }, { wait: 300 }] },
  'w11-scheduled-backups-change': { base: 'w02-updated-devices-out-of-date-backups', full: true, steps: [{ button: 'Change' }] },
  'w12-whats-inside': { base: 'w02-updated-devices-out-of-date-backups', steps: [{ button: "See what's inside" }, { wait: 300 }] },
};
export const webPhone = ['w01-homebox-dnf-update-available', 'w02-updated-devices-out-of-date-backups', 'w05-backup-download-dialog', 'w06-restore-preview-new-vps'];
