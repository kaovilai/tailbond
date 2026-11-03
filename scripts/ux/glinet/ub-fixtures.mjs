// Updates and backup/restore screens (updates-backup-screens.mjs): the GL
// panel (Slate 7 travel router, Flint 3 home server) and the agent's own web
// dashboard (Fedora home server "homebox", a VPS "vps"). Shapes follow
// docs/architecture.md "Updates" and "Backup and restore", and the status
// fields tailbondd fills (`update.last_result` + `detail`, `update.mirror`
// {version, files, bytes}, `clients[].version/kind`). Scheduled-backup
// settings are read from GET /api/settings `backup` (passphrase_set,
// every_hours, keep, push, last, last_push, last_error).
//
// One clock for every screen: today 22:00 (UTC). The servers install
// between 03:00 and 05:00; homebox updated to 0.2.0 at 03:04 this morning.
import { connected, serverBase, serverStatus, withApi, twoWans, landing } from './fixtures.mjs';

const T = Math.floor(Date.now() / 864e5) * 864e5 + 22 * 3600e3;
export const clock = new Date(T).toISOString();
const hAgo = (hrs) => new Date(T - hrs * 3600e3).toISOString();
const NOTES = 'https://github.com/tailbond/tailbond/releases/tag/v0.2.0';

// verify/verified/repo: how an update is checked (GitHub attestation of the
// release workflow; a router trusts its home server's copy; a GL router as
// the home server has no attestation check: checksums only)
const upd = (over = {}) => ({ current: '0.1.0', policy: 'auto', channel: 'stable', state: 'idle', last_check: hAgo(1.5), window: '03:00-05:00', verify: 'attestation', repo: 'tailbond/tailbond', ...over });
const avail = (over = {}) => upd({ available: '0.2.0', notes_url: NOTES, verified: 'attestation', ...over });
const mirror = { version: '0.2.0', files: 5, bytes: 58.4e6 };

// ---- GL travel router (Slate 7) ----
const sched = { passphrase_set: true, every_hours: 24, keep: 7, push: true, last: hAgo(5), last_push: hAgo(5) };
const router = (u0, bk = sched) => {
  const u = { ...u0, ...(u0.available && u0.verified === 'attestation' ? { verified: 'mirror', source: u0.source || 'mirror' } : {}) };
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
  'GET /api/update': { ...u, manifest: u.available ? { version: u.available, channel: 'stable', released: hAgo(30), security: false } : null },
  'GET /api/backup/summary': { role: 'client', mode: 'speed', servers: ['homebox (home.example.net:443)', 'vps (203.0.113.9:8443)'], bypass_rules: 4, paths: 3, password_set: false, update_policy: 'auto', files: ['global.key'], uci: true },
  'POST /api/restore/preview': routerPreview,
  'POST /api/restore': { ok: true, rollback: 'before-restore-2026-11-02T2201.tbbackup' },
  ...extra,
});
const r = (u, bk) => withApi(router(u, bk), routerApi(u));

export const glStates = {
  'r01-router-up-to-date': r(upd({ current: '0.2.0', source: 'mirror', last_result: 'ok' })),
  'r02-router-update-available-automatic': r(avail({ state: 'waiting', deferred: 'traffic', source: 'mirror' })),
  'r03-router-installing': r(avail({ state: 'installing', source: 'mirror' })),
  'r04-router-rolled-back': r(avail({ state: 'rolled_back', source: 'mirror', last_result: 'rolled_back', detail: "the tunnel didn't come back within 90 seconds" })),
  'r05-router-updates-from-homebox': r(avail({ policy: 'notify', source: 'mirror' }), { ...sched, push: true }),
  'r06-router-reinstalled-after-firmware-upgrade': r(upd({ current: '0.1.0', source: 'mirror', last_result: 'ok', detail: 'reinstalled_after_firmware_upgrade' })),
  'r07-router-pinned-by-setup': r(avail({ pin: '0.1.0', source: 'mirror' })),
  'r08-router-backups-off': r(upd({ current: '0.2.0', source: 'mirror', last_result: 'ok' }), { passphrase_set: false, every_hours: 24, keep: 7, push: true }),
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
const flint = (u0) => {
  const u = { ...u0, verify: 'sha256-only', repo: undefined, ...(u0.verified ? { verified: 'sha256' } : {}) };
  const fx = serverBase({ clients: flintClients, update: u }, {}, { backup: { passphrase_set: true, every_hours: 24, keep: 7, last: hAgo(19) } });
  return { ...withApi(fx, {
    'GET /api/update': u,
    'GET /api/backups': flintBackups,
    'GET /api/backup/summary': { role: 'server', mode: 'speed', clients: ['slate7', 'pixel-8', 'beryl'], public_addr: 'home.example.net', port: 443, subnet: '10.77.0.0/24', approvers: null, password_set: false, update_policy: 'auto', files: ['global.key', 'server.crt', 'server.key'], uci: true },
  }), clock, model: 'GL-BE9300', info: { ...fx.info, version: u.current === '0.2.0' ? '0.2.0-r1' : '0.1.0-r42' } };
};
const flintUpdated = upd({ current: '0.2.0', last_result: 'ok', outdated, mirror });
glStates['s01-flint-devices-out-of-date-mirror'] = flint(flintUpdated);
glStates['s03-flint-update-waiting-for-window'] = flint(avail({ state: 'waiting', deferred: 'window', mirror, outdated: [] }));

const upFiles = [
  ['tailbond_0.2.0-r1_aarch64_cortex-a53_neon-vfpv4.ipk', 8.9e6],
  ['stable.json', 2140],
  ['tailbond-0.2.0.apk', 27.8e6],
  ['stable.json.sigstore.json', 13987],
  ['tailbond_0.2.0-r1_aarch64_cortex-a53.ipk', 8.8e6],
];
// steps: {text} clicks the first element with that text, {click} a selector,
// {button} a button by exact name, {fill: [selector, value]},
// {files: [selector, [[name, size], ...]]} picks files.
const pass = 'maple tunnel violet seven';
export const glActions = {
  'r09-router-backup-download-dialog': { base: 'r01-router-up-to-date', steps: [{ button: 'Download a backup' }, { fill: ['.tb-dialog input[type=password] >> nth=0', pass] }, { fill: ['.tb-dialog input[type=password] >> nth=1', pass] }] },
  'r10-router-restore-preview': { base: 'r08-router-backups-off', steps: [{ button: 'Restore from a backup' }, { files: ['.tb-dialog input.tb-file', [['slate7-2026-11-01.tbbackup', 6100]]] }, { fill: ['.tb-dialog input[type=password]', pass] }, { button: 'Check the backup' }, { wait: 300 }] },
  'r11-router-restore-done': { base: 'r08-router-backups-off', steps: [{ button: 'Restore from a backup' }, { files: ['.tb-dialog input.tb-file', [['slate7-2026-11-01.tbbackup', 6100]]] }, { fill: ['.tb-dialog input[type=password]', pass] }, { button: 'Check the backup' }, { wait: 300 }, { button: 'Restore' }, { wait: 300 }] },
  'r12-router-install-confirm': { base: 'r02-router-update-available-automatic', steps: [{ button: 'Install 0.2.0 now' }] },
  'r13-router-whats-inside': { base: 'r01-router-up-to-date', steps: [{ button: "See what's inside" }, { wait: 200 }] },
  's02-flint-upload-an-update': { base: 's01-flint-devices-out-of-date-mirror', steps: [{ button: 'Upload an update' }, { files: ['.tb-dialog input.tb-up', upFiles] }, { button: 'Upload' }, { wait: 600 }] },
};
// also rendered at phone width (390x844)
export const glPhone = ['r02-router-update-available-automatic', 'r09-router-backup-download-dialog', 'r10-router-restore-preview', 's01-flint-devices-out-of-date-mirror'];

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
const webSummary = { role: 'server', mode: 'speed', clients: ['slate7', 'pixel-8', 'beryl'], public_addr: 'home.example.net', port: 443, subnet: '10.77.0.0/24', approvers: ['sam@example.com'], password_set: false, update_policy: 'auto', files: ['global.key', 'server.crt', 'server.key'], uci: false };
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

const waitingWindow = avail({ state: 'waiting', deferred: 'window', mirror, outdated: [] });
export const webStates = {
  'w01-update-available-waiting-for-window': webHome(waitingWindow),
  'w02-updated-devices-out-of-date-backups': webHome(flintUpdated),
  'w08-vps-rolled-back': webVps(avail({ state: 'rolled_back', last_result: 'rolled_back', detail: "the API didn't answer within 90 seconds after the restart", mirror: { version: '0.1.0', files: 4, bytes: 49e6 } })),
  'w09-vps-pinned-by-setup': webVps(avail({ pin: '0.1.0', mirror: { version: '0.1.0', files: 4, bytes: 49e6 } })),
  'w10-vps-new-machine': { ...webVps(upd({ current: '0.2.0', last_result: 'ok', mirror })), settings: webSettings({ public_addr: '198.51.100.20', effective_public_addr: '198.51.100.20:443', backup: { passphrase_set: false, every_hours: 24, keep: 7 } }) },
};
const vpsFiles = [
  ['tailbond-server-0.2.0-1.fc44.x86_64.rpm', 9.8e6],
  ['stable.json.sigstore.json', 13987],
  ['tailbond_0.2.0-r1_aarch64_cortex-a53_neon-vfpv4.ipk', 8.9e6],
  ['stable.json', 2140],
  ['tailbond-0.2.0.apk', 27.8e6],
];
export const webActions = {
  'w03-install-confirm': { base: 'w01-update-available-waiting-for-window', steps: [{ button: 'Install 0.2.0 now' }] },
  'w04-upload-an-update': { base: 'w08-vps-rolled-back', steps: [{ button: 'Upload an update' }, { files: ['#up-files', vpsFiles] }, { button: 'Upload' }, { wait: 800 }] },
  'w05-backup-download-dialog': { base: 'w02-updated-devices-out-of-date-backups', steps: [{ button: 'Download a backup' }, { fill: ['#bk-pass', pass] }, { fill: ['#bk-pass2', pass] }] },
  'w06-restore-preview-new-vps': { base: 'w10-vps-new-machine', steps: [{ button: 'Restore from a backup' }, { files: ['#rs-file', [['vps-2026-11-01T0300.tbbackup', 11200]]] }, { fill: ['#rs-pass', pass] }, { button: 'Check the backup' }, { wait: 300 }] },
  'w07-restore-done': { base: 'w10-vps-new-machine', steps: [{ button: 'Restore from a backup' }, { files: ['#rs-file', [['vps-2026-11-01T0300.tbbackup', 11200]]] }, { fill: ['#rs-pass', pass] }, { button: 'Check the backup' }, { wait: 300 }, { button: 'Restore' }, { wait: 300 }] },
  'w11-scheduled-backups-change': { base: 'w02-updated-devices-out-of-date-backups', full: true, steps: [{ button: 'Change' }] },
  'w12-whats-inside': { base: 'w02-updated-devices-out-of-date-backups', steps: [{ button: "See what's inside" }, { wait: 300 }] },
};
export const webPhone = ['w02-updated-devices-out-of-date-backups', 'w05-backup-download-dialog', 'w06-restore-preview-new-vps'];
