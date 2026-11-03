// Updates and backup/restore screens (updates-backup-screens.mjs): the GL
// panel (Slate 7 travel router, Flint 3 home server) and the agent's own web
// dashboard (Fedora home server "homebox", a VPS "vps", a new VPS "vps2").
// Shapes follow docs/architecture.md "Updates" and "Backup and restore": the
// status object `update` {current, available, notes_url, policy, repo,
// method, state, last_check, last_result, last_result_version,
// last_result_at, detail, error, auto_window, dnf_automatic, outdated} and
// `clients[].version/kind`. Scheduled-backup settings are read from GET
// /api/settings `backup` (passphrase_set, every_hours, keep, push, last,
// last_push, last_error). Restore warnings are the texts of
// agent/internal/agent/backupapi.go restoreWarnings.
//
// One clock for every screen: today 22:00 (UTC). On Tell me, slate7 and
// homebox were updated to 0.2.0 by hand at 14:20; on Automatic it installs
// at 03:06. Scheduled backups run at 03:00, slate7 pushes its backup at
// 17:00; pixel-8's are sent by hand from the app (Nov 3 08:12, Oct 31
// 11:36, Oct 26 20:54).
import { connected, serverBase, serverStatus, withApi, twoWans, landing } from './fixtures.mjs';

const T = Math.floor(Date.now() / 864e5) * 864e5 + 22 * 3600e3;
export const clock = new Date(T).toISOString();
const hAgo = (hrs) => new Date(T - hrs * 3600e3).toISOString();
const notes = (v) => `https://github.com/kaovilai/tailbond/releases/tag/v${v}`;
// file names as tailbondd writes them (UTC): scheduled <host>-YYYYMMDD-HHMMSS,
// pushed by a device YYYYMMDD-HHMMSS, downloaded tailbond-<host>-YYYYMMDD-HHMM
const stamp = (iso, secs = true) => iso.replace(/[-:]/g, '').replace('T', '-').slice(0, secs ? 15 : 13);
const own = (host, hrs, size) => ({ name: `${host}-${stamp(hAgo(hrs))}.tbbackup`, client: '', size, created: hAgo(hrs) });
const pushed = (client, hrs, size) => ({ name: `${stamp(hAgo(hrs))}.tbbackup`, client, size, created: hAgo(hrs) });
const download = (host, hrs) => `tailbond-${host}-${stamp(hAgo(hrs), false)}.tbbackup`;
const rollback = `before-restore-${stamp(new Date(T + 60e3).toISOString())}.tbbackup`;

// method: opkg (GL routers), static (the Linux bundle on the VPS), dnf
// (the Fedora RPM on homebox)
const upd = (over = {}) => ({ current: '0.1.0', available: '', notes_url: '', policy: 'notify', repo: 'kaovilai/tailbond', method: 'opkg', state: 'idle', last_check: hAgo(1.5),
  last_result: '', last_result_version: '', last_result_at: '', detail: '', error: '', auto_window: '03:00-05:00', ...over });
const avail = (over = {}) => upd({ available: '0.2.0', notes_url: notes((over.available) || '0.2.0'), ...over });
// updated to 0.2.0 by hand at 14:20 (Tell me), or at 03:06 (Automatic)
const went = { last_result: 'ok', last_result_version: '0.2.0', last_result_at: hAgo(7 + 40 / 60) };
const wentNight = { ...went, last_result_at: hAgo(18.9) };

// ---- GL travel router (Slate 7) ----
const sched = { passphrase_set: true, every_hours: 24, keep: 7, push: true, last: hAgo(5), last_push: hAgo(5) };
const router = (u, bk = sched) => {
  const fx = connected(twoWans, { update: u, settings: { backup: bk } });
  return { ...fx, clock, info: { installed: true, version: u.current === '0.2.0' ? '0.2.0-r1' : '0.1.0-r42', role: 'client', enabled: true, running: true, kill_switch: true } };
};
// slate7's own backup back on a factory-reset Slate 7: same machine, so no
// warnings (restoreWarnings warns about the old machine only for a server)
const slateFile = download('slate7', 29);
const routerPreview = {
  role: 'client', host: 'slate7', created: hAgo(29), agent_version: '0.2.0',
  changes: [
    { what: 'Mode', from: '(default)', to: 'Speed' },
    { what: 'Servers', from: '(none)', to: 'homebox, vps' },
    { what: 'Bypass rules', from: '0', to: '4' },
    { what: 'Connection settings', from: '0', to: '3' },
  ],
  warnings: [],
};
const routerApi = (u, extra = {}) => ({
  'GET /api/update': u,
  'GET /api/backup/summary': { role: 'client', mode: 'speed', servers: ['homebox (home.example.net:443)', 'vps (203.0.113.9:8443)'], bypass_rules: 4, paths: 3, password_set: false, update_policy: 'Tell me', files: ['global.key'], uci: true },
  'POST /api/restore/preview': routerPreview,
  'POST /api/restore': { ok: true, rollback },
  ...extra,
});
const r = (u, bk) => withApi(router(u, bk), routerApi(u));

export const glStates = {
  'r01-router-up-to-date': r(upd({ current: '0.2.0', ...went })),
  'r02-router-update-available': r(avail()),
  'r03-router-installing': r(avail({ state: 'installing' })),
  'r04-router-rolled-back': r(avail({ state: 'rolled_back', last_result: 'rolled_back', last_result_version: '0.2.0', last_result_at: hAgo(0.2), detail: "the tunnel didn't come back within 60 seconds" })),
  'r05-router-automatic': r(upd({ current: '0.2.0', policy: 'auto', ...wentNight })),
  'r06-router-reinstalled-after-firmware-upgrade': r(upd({ current: '0.2.0', last_check: hAgo(0.3), last_result: 'ok', last_result_version: '0.2.0', last_result_at: hAgo(0.4), detail: 'reinstalled_after_firmware_upgrade' })),
  'r08-router-backups-off': r(upd({ current: '0.2.0', ...went }), { passphrase_set: false, every_hours: 24, keep: 7, push: true }),
};

// ---- GL Flint 3 as the home server ----
// beryl is off and hasn't connected since the servers learned versions
const flintClients = [
  { name: 'slate7', registered: true, connected: true, connected_sec: 5400, tx_bps: 18e6, rx_bps: 2.1e6, paths: [{ state: 'active' }, { state: 'active' }], version: '0.1.0', kind: 'router' },
  { name: 'pixel-8', registered: true, connected: true, connected_sec: 3 * 3600, tx_bps: 12e6, rx_bps: 1.1e6, paths: [{ state: 'active' }, { state: 'active' }], version: '0.1.0', kind: 'android' },
  { name: 'beryl', registered: true, connected: false, connected_sec: 0, tx_bps: 0, rx_bps: 0, paths: [], version: '', kind: 'router' },
];
const outdated = [{ name: 'slate7', kind: 'router', version: '0.1.0' }, { name: 'pixel-8', kind: 'android', version: '0.1.0' }];
// homebox's saved backups: its own (03:00), and what slate7, pixel-8 and
// beryl pushed (the last 3 each)
const devBackups = [
  pushed('slate7', 5, 6100), pushed('slate7', 29, 6100), pushed('slate7', 53, 6050),
  // the phone's are sent by hand from the app, whenever
  pushed('pixel-8', 13.8, 3900), pushed('pixel-8', 82.4, 3880), pushed('pixel-8', 193.1, 3790),
  pushed('beryl', 77.2, 5900),
];
const flintBackups = [own('homebox', 19, 11200), own('homebox', 43, 11150), ...devBackups];
const serverFiles = ['global.key', 'server.crt', 'server.key', ...devBackups.map((b) => `client-backups/${b.client}/${b.name}`)];
const flint = (u) => {
  const fx = serverBase({ clients: flintClients, update: u }, {}, { backup: { passphrase_set: true, every_hours: 24, keep: 7, last: hAgo(19) } });
  return { ...withApi(fx, {
    'GET /api/update': u,
    'GET /api/backups': flintBackups,
    'GET /api/backup/summary': { role: 'server', mode: 'speed', clients: ['slate7', 'pixel-8', 'beryl'], public_addr: 'home.example.net', port: 443, subnet: '10.77.0.0/24', approvers: null, password_set: false, update_policy: 'Tell me', files: serverFiles, uci: true },
  }), clock, model: 'GL-BE9300', info: { ...fx.info, version: u.current === '0.2.0' ? '0.2.0-r1' : '0.1.0-r42' } };
};
glStates['s01-flint-devices-out-of-date'] = flint(upd({ current: '0.2.0', ...went, outdated }));

// steps: {text} clicks the first element with that text, {click} a selector,
// {button} a button by exact name, {fill: [selector, value]},
// {files: [selector, [[name, size], ...]]} picks files.
const pass = 'maple tunnel violet seven';
const restoreSteps = (sel, file, size) => [{ button: 'Restore from a backup' }, { files: [sel.file, [[file, size]]] }, { fill: [sel.pass, pass] }, { button: 'Check the backup' }, { wait: 300 }];
const glSel = { file: '.tb-dialog input.tb-file', pass: '.tb-dialog input[type=password]' };
export const glActions = {
  'r07-router-update-confirm': { base: 'r02-router-update-available', steps: [{ button: 'Update now' }] },
  'r09-router-backup-download-dialog': { base: 'r01-router-up-to-date', steps: [{ button: 'Download a backup' }, { fill: ['.tb-dialog input[type=password] >> nth=0', pass] }, { fill: ['.tb-dialog input[type=password] >> nth=1', pass] }] },
  'r10-router-restore-preview': { base: 'r08-router-backups-off', steps: restoreSteps(glSel, slateFile, 6100) },
  'r11-router-restore-done': { base: 'r08-router-backups-off', steps: [...restoreSteps(glSel, slateFile, 6100), { button: 'Restore' }, { wait: 300 }] },
  'r13-router-whats-inside': { base: 'r01-router-up-to-date', steps: [{ button: "See what's inside" }, { wait: 200 }] },
  's02-flint-whats-inside': { base: 's01-flint-devices-out-of-date', steps: [{ button: "See what's inside" }, { wait: 200 }] },
};
// also rendered at phone width (390x844)
export const glPhone = ['r02-router-update-available', 'r09-router-backup-download-dialog', 'r10-router-restore-preview', 's01-flint-devices-out-of-date'];

// ---- the agent's own dashboard (agent/web): Fedora "homebox", a VPS ----
const srvPath = (id, rx, tx, rtt) => ({ id, iface: '', kind: 'unknown', state: 'active', srtt_ms: rtt, min_rtt_ms: rtt, loss_pct: 0, rx_bps: rx, tx_bps: tx, rx_bytes: 0, tx_bytes: 0 });
const webClients = [
  { name: 'slate7', registered: true, connected: true, endpoint: '198.51.100.23:40112', scheduler: 'wlb', connected_sec: 5400, tx_bps: 18e6, rx_bps: 2.1e6, tx_bytes: 0, rx_bytes: 0, paths: [srvPath(0, 1.2e6, 11e6, 31), srvPath(1, 0.9e6, 7e6, 48)], version: '0.1.0', kind: 'router' },
  { name: 'pixel-8', registered: true, connected: true, endpoint: '203.0.113.77:51820', scheduler: 'wlb', connected_sec: 3 * 3600, tx_bps: 6e6, rx_bps: 0.4e6, tx_bytes: 0, rx_bytes: 0, paths: [srvPath(0, 0.4e6, 6e6, 40)], version: '0.1.0', kind: 'android' },
  { name: 'beryl', registered: true, connected: false, connected_sec: 0, tx_bps: 0, rx_bps: 0, tx_bytes: 0, rx_bytes: 0, paths: [], version: '', kind: 'router' },
];
const webStatus = (u, over = {}) => ({
  ...serverStatus({ clients: webClients }), totals: { tx_bps: 24e6, rx_bps: 2.5e6, tx_bytes: 8.1e9, rx_bytes: 0.9e9 }, pair_requests: [],
  version: u.current, update: u, ...over,
});
const webSettings = (over = {}) => ({ role: 'server', mode: 'speed', public_addr: 'home.example.net', effective_public_addr: 'home.example.net:443', port: 443, subnet: '10.77.0.0/24', tls: 'tailscale', tls_name: 'homebox.tail1234.ts.net', version: '0.2.0', glinet_ipk: true, password_set: false, ...landing,
  backup: { passphrase_set: true, every_hours: 24, keep: 7, last: hAgo(19) }, ...over });
const webBackups = [own('homebox', 19, 11200), own('homebox', 43, 11150), own('homebox', 67, 11150), ...devBackups];
const webSummary = { role: 'server', mode: 'speed', clients: ['slate7', 'pixel-8', 'beryl'], public_addr: 'home.example.net', port: 443, subnet: '10.77.0.0/24', approvers: ['sam@example.com'], password_set: false, update_policy: 'Tell me', files: serverFiles, uci: false };
// a new VPS "vps2" (198.51.100.20, port 443) getting the old one's backup
// (vps, 203.0.113.9, port 8443, tailscale cert): restoreWarnings' four
// numbered steps; the restore keeps this machine's public address and port
// (backup.KeepOnRestore), so the change list doesn't show them
const vpsFile = `vps-${stamp(hAgo(43))}.tbbackup`;
const vpsPreview = {
  role: 'server', host: 'vps', created: hAgo(43), agent_version: '0.2.0',
  changes: [
    { what: 'Mode', from: '(default)', to: 'Speed' },
    { what: 'Devices (clients)', from: '(none)', to: 'slate7, pixel-8, beryl' },
    { what: 'Tunnel addresses', from: '10.77.0.0/24', to: '10.83.142.0/24' },
    { what: 'Approvers', from: '(none)', to: 'sam@example.com' },
    { what: 'Update policy', from: 'Tell me', to: 'Automatic' },
  ],
  warnings: [
    '1. Restore here while vps still runs: slate7, pixel-8, beryl keep using it meanwhile.',
    "2. Devices reached the old machine at 203.0.113.9:8443; this one is reached at 198.51.100.20:443 (kept). Devices that have only a link (no Tailscale on them): give each its new link from this machine's Devices tab. A GL router: paste it in its tailbond panel (Add another server, Paste a link; it replaces the old one). A phone: scan the new QR code in the tailbond app. A travel router with the kill switch on needs this before step 3.",
    '3. Then turn tailbond off on vps (sudo systemctl disable --now tailbond there) and give this machine its Tailscale name vps: in the Tailscale admin console remove (or rename) the old machine, then rename this one. Devices with Tailscale on follow by themselves within a few minutes.',
    "4. If you approved this machine's route in the Tailscale admin console, approve 10.83.142.0/24 instead (the backup's tunnel addresses).",
  ],
};
const webApi = (u, extra = {}) => ({
  'GET /api/update': u,
  'GET /api/backups': webBackups,
  'GET /api/backup/summary': webSummary,
  'POST /api/restore/preview': vpsPreview,
  'POST /api/restore': { ok: true, rollback },
  ...extra,
});
const vpsTs = (self) => ({ tailscale: { running: true, self, routes_approved: true, installed: true, dns_name: `${self}.tail1234.ts.net` }, server: `${self}.tail1234.ts.net` });
const webVps = (u) => ({ status: webStatus(u, vpsTs('vps')),
  settings: webSettings({ public_addr: '203.0.113.9', effective_public_addr: '203.0.113.9:8443', port: 8443, subnet: '10.83.142.0/24', tls_name: 'vps.tail1234.ts.net' }), api: webApi(u) });
const webHome = (u, setOver) => ({ status: webStatus(u), settings: webSettings(setOver), api: webApi(u) });
// vps2 just installed: no devices yet, up 12 minutes, nothing saved
const webFresh = (u) => ({
  status: webStatus(u, { ...vpsTs('vps2'), clients: [], uptime_sec: 12 * 60, totals: { tx_bps: 0, rx_bps: 0, tx_bytes: 0, rx_bytes: 0 }, subnet: '10.77.0.0/24' }),
  settings: webSettings({ public_addr: '198.51.100.20', effective_public_addr: '198.51.100.20:443', tls_name: 'vps2.tail1234.ts.net', backup: { passphrase_set: false, every_hours: 24, keep: 7 } }),
  api: webApi(u, { 'GET /api/backups': [], 'GET /api/backup/summary': { role: 'server', mode: 'speed', clients: [], public_addr: '198.51.100.20', port: 443, subnet: '10.77.0.0/24', approvers: null, password_set: false, update_policy: 'Tell me', files: ['global.key', 'server.crt', 'server.key'], uci: false } }),
});

// homebox: Fedora RPM (dnf), on 0.2.0; 0.2.1 is out
const homeU = (over = {}) => upd({ method: 'dnf', current: '0.2.0', ...went, outdated, dnf_automatic: 'install', ...over });
// vps: the Linux bundle (static), on 0.1.0
const vpsU = (over = {}) => upd({ method: 'static', outdated: [], ...over });
export const webStates = {
  'w01-homebox-dnf-update-available': webHome(homeU({ available: '0.2.1', notes_url: notes('0.2.1'), dnf_automatic: 'download' })),
  'w02-updated-devices-out-of-date-backups': webHome(homeU()),
  'w03-vps-update-available': webVps(vpsU({ available: '0.2.0', notes_url: notes('0.2.0') })),
  'w08-vps-rolled-back': webVps(vpsU({ available: '0.2.0', notes_url: notes('0.2.0'), policy: 'auto', state: 'rolled_back', last_result: 'rolled_back', last_result_version: '0.2.0', last_result_at: hAgo(18.9), detail: "the service wasn't up on 0.2.0 within 60 seconds" })),
  'w09-vps-check-failed': webVps(vpsU({ last_check: hAgo(0.1), error: 'api.github.com: i/o timeout' })),
  'w10-vps-new-machine': webFresh(vpsU({ current: '0.2.0', last_check: hAgo(0.15) })),
  'w13-vps-updates-off': webVps(vpsU({ policy: 'off', state: 'off', last_check: '' })),
};
const webSel = { file: '#rs-file', pass: '#rs-pass' };
export const webActions = {
  'w04-vps-update-confirm': { base: 'w03-vps-update-available', steps: [{ button: 'Update now' }] },
  'w05-backup-download-dialog': { base: 'w02-updated-devices-out-of-date-backups', steps: [{ button: 'Download a backup' }, { fill: ['#bk-pass', pass] }, { fill: ['#bk-pass2', pass] }] },
  'w06-restore-preview-new-vps': { base: 'w10-vps-new-machine', steps: restoreSteps(webSel, vpsFile, 11200) },
  'w07-restore-done': { base: 'w10-vps-new-machine', steps: [...restoreSteps(webSel, vpsFile, 11200), { button: 'Restore' }, { wait: 300 }] },
  'w11-scheduled-backups-change': { base: 'w02-updated-devices-out-of-date-backups', full: true, steps: [{ button: 'Change' }] },
  'w12-whats-inside': { base: 'w02-updated-devices-out-of-date-backups', steps: [{ button: "See what's inside" }, { wait: 300 }] },
};
export const webPhone = ['w01-homebox-dnf-update-available', 'w02-updated-devices-out-of-date-backups', 'w05-backup-download-dialog', 'w06-restore-preview-new-vps'];
