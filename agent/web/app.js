'use strict';
// tailbond dashboard: vanilla JS, no build step. Data comes from /api/events
// (SSE, one normalized status object per second) with /api/status as fallback.

const $ = (sel, el = document) => el.querySelector(sel);
const HISTORY = 60;
const KIND_LABEL = { wifi: 'Wi-Fi', cellular: 'Cellular', ethernet: 'Ethernet', usb: 'USB tether', unknown: 'Link' };

const state = {
  status: null,
  role: null,
  tab: 'connections',
  hist: new Map(), // series key -> {rx: [], tx: []}
  cards: new Map(), // card key -> element
  clientEls: new Map(),
  logSeq: 0,
  settings: null,
  modeBusy: false,
  powerBusyUntil: 0,
  lastTs: 0,
};

// ---------- helpers ----------
function h(tag, attrs = {}, ...kids) {
  const el = document.createElement(tag);
  for (const [k, v] of Object.entries(attrs)) {
    if (v == null || v === false) continue;
    if (k === 'class') el.className = v;
    else if (k.startsWith('on')) el.addEventListener(k.slice(2), v);
    else if (k === 'text') el.textContent = v;
    else el.setAttribute(k, v === true ? '' : v);
  }
  for (const kid of kids.flat()) if (kid != null) el.append(kid.nodeType ? kid : document.createTextNode(kid));
  return el;
}
function icon(id, cls = 'ico') {
  const ns = 'http://www.w3.org/2000/svg';
  const svg = document.createElementNS(ns, 'svg');
  svg.setAttribute('class', cls);
  const use = document.createElementNS(ns, 'use');
  use.setAttribute('href', '#i-' + id);
  svg.append(use);
  return svg;
}
function fmtRate(bps) {
  bps = bps || 0;
  if (bps >= 1e9) return [(bps / 1e9).toFixed(2), 'Gbps'];
  if (bps >= 1e6) return [(bps / 1e6).toFixed(bps >= 1e8 ? 0 : 1), 'Mbps'];
  if (bps >= 1e3) return [(bps / 1e3).toFixed(0), 'kbps'];
  return [String(Math.round(bps)), 'bps'];
}
const rateStr = (bps) => fmtRate(bps).join(' ');
function fmtBytes(b) {
  b = b || 0;
  const u = ['B', 'KB', 'MB', 'GB', 'TB'];
  let i = 0;
  while (b >= 1024 && i < u.length - 1) { b /= 1024; i++; }
  return (i === 0 ? b : b.toFixed(b >= 100 ? 0 : 1)) + ' ' + u[i];
}
function fmtDur(s) {
  if (!s) return '—';
  const d = Math.floor(s / 86400), hh = Math.floor(s / 3600) % 24, m = Math.floor(s / 60) % 60;
  if (d) return `${d}d ${hh}h`;
  if (hh) return `${hh}h ${m}m`;
  if (m) return `${m}m ${s % 60}s`;
  return `${s}s`;
}
function toast(msg, bad = false) {
  const t = $('#toast');
  t.textContent = msg;
  t.className = 'toast' + (bad ? ' bad' : '');
  clearTimeout(toast.t);
  toast.t = setTimeout(() => t.classList.add('hidden'), 3200);
}
async function api(method, path, body) {
  const opts = { method, headers: {} };
  // the API only accepts JSON-typed mutations (CSRF guard)
  if (body === undefined && method !== 'GET' && method !== 'DELETE') body = {};
  if (body !== undefined) {
    opts.headers['Content-Type'] = 'application/json';
    opts.body = JSON.stringify(body);
  }
  const r = await fetch(path, opts);
  if (r.status === 401) { showLogin(); throw new Error('login required'); }
  const data = await r.json().catch(() => ({}));
  if (r.status === 403 && data.owner) { showOwnerOnly(data.owner); throw new Error('owner only'); }
  if (!r.ok || data.ok === false) throw new Error(data.error || r.statusText);
  return data;
}
function push(key, rx, tx) {
  let s = state.hist.get(key);
  if (!s) { s = { rx: Array(HISTORY).fill(0), tx: Array(HISTORY).fill(0), seen: 0 }; state.hist.set(key, s); }
  s.rx.push(rx || 0); s.tx.push(tx || 0);
  if (s.rx.length > HISTORY) { s.rx.shift(); s.tx.shift(); }
  s.seen = Date.now();
  return s;
}

// ---------- sparklines ----------
function niceMax(v) {
  if (v <= 0) return 1e6;
  const p = Math.pow(10, Math.floor(Math.log10(v)));
  for (const m of [1, 2, 5, 10]) if (m * p >= v) return m * p;
  return 10 * p;
}
function linePath(vals, w, hgt, max, pad = 2) {
  const n = vals.length;
  const step = w / (n - 1);
  let d = '';
  vals.forEach((v, i) => {
    const x = i * step;
    const y = hgt - pad - (Math.min(v, max) / max) * (hgt - 2 * pad);
    d += (i ? 'L' : 'M') + x.toFixed(1) + ' ' + y.toFixed(1);
  });
  return d;
}
function drawSpark(svg, series, w, hgt, opts = {}) {
  const max = niceMax(Math.max(...series.rx, ...series.tx));
  const rx = linePath(series.rx, w, hgt, max);
  const tx = linePath(series.tx, w, hgt, max);
  const id = svg.dataset.gid || (svg.dataset.gid = 'g' + Math.random().toString(36).slice(2, 8));
  svg.innerHTML =
    `<defs><linearGradient id="${id}" x1="0" x2="0" y1="0" y2="1">` +
    `<stop offset="0" style="stop-color:var(--down);stop-opacity:.35"/><stop offset="1" style="stop-color:var(--down);stop-opacity:0"/></linearGradient></defs>` +
    (opts.grid ? `<line x1="0" x2="${w}" y1="${hgt / 2}" y2="${hgt / 2}" style="stroke:var(--line)" stroke-dasharray="4 6" vector-effect="non-scaling-stroke"/>` : '') +
    `<path d="${rx} L${w} ${hgt} L0 ${hgt}Z" fill="url(#${id})"/>` +
    `<path d="${rx}" fill="none" style="stroke:var(--down)" stroke-width="${opts.sw || 2}" stroke-linejoin="round" vector-effect="non-scaling-stroke"/>` +
    `<path d="${tx}" fill="none" style="stroke:var(--up)" stroke-width="${opts.sw || 2}" stroke-linejoin="round" vector-effect="non-scaling-stroke"/>` +
    (opts.label ? `<text x="${w - 4}" y="12" text-anchor="end" font-size="11" style="fill:var(--muted)">${rateStr(max)}</text>` : '');
}

// ---------- rendering ----------
function render(st) {
  state.status = st;
  if (state.role !== st.role) setRole(st.role);

  const [rxv, rxu] = fmtRate(st.totals.rx_bps);
  const [txv, txu] = fmtRate(st.totals.tx_bps);
  // the server's rx is what devices upload, its tx what they download
  const srv = st.role === 'server';
  $('#rx-val').textContent = rxv; $('#rx-unit').textContent = rxu + (srv ? ' from devices' : ' down');
  $('#tx-val').textContent = txv; $('#tx-unit').textContent = txu + (srv ? ' to devices' : ' up');
  $('#uptime').textContent = st.state === 'connected' ? fmtDur(st.uptime_sec) : '—';
  $('#tunnel-ip').textContent = st.tunnel_ip || '—';
  $('#total-bytes').textContent = st.role === 'server'
    ? `${fmtBytes(st.totals.tx_bytes)} to · ${fmtBytes(st.totals.rx_bytes)} from devices`
    : `${fmtBytes(st.totals.rx_bytes)} ↓ · ${fmtBytes(st.totals.tx_bytes)} ↑`;
  const chip = $('#state-chip');
  chip.textContent = st.state;
  chip.className = 'state-chip s-' + st.state;
  drawSpark($('#spark-total'), push('total', st.totals.rx_bps, st.totals.tx_bps), 600, 120, { grid: true, sw: 2.2 });

  const sub = st.role === 'server'
    ? `Home server · reachable at ${(state.settings && state.settings.effective_public_addr) || st.server || 'no public address'}`
    : (st.paired ? `Bonded to ${st.server}` : 'Not paired');
  $('#subtitle').textContent = sub;
  document.title = st.state === 'connected' ? `↓${rateStr(st.totals.rx_bps)} · tailbond` : 'tailbond';

  const banner = $('#error-banner');
  if (st.error) { banner.textContent = st.error; banner.classList.remove('hidden'); } else banner.classList.add('hidden');

  // power toggle
  if (Date.now() > state.powerBusyUntil) {
    const on = st.state !== 'disconnected';
    const p = $('#power');
    p.setAttribute('aria-pressed', String(on));
    p.classList.toggle('busy', st.state === 'connecting');
    $('#power-label').textContent = { connected: 'Connected', connecting: 'Connecting…', error: 'Retrying…', disconnected: st.role === 'server' ? 'Stopped' : 'Disconnected' }[st.state] || st.state;
  }

  if (!state.modeBusy) {
    for (const b of document.querySelectorAll('#modes button')) b.setAttribute('aria-checked', String(b.dataset.mode === st.mode));
  }

  renderPairRequests(st);
  if (st.role === 'client') renderClientPaths(st); else renderServerPaths(st);
  if (st.role === 'server' && state.tab === 'clients') renderClients(st);
  renderMaint(st);
  $('#foot').textContent = `tailbond ${st.version || ''} · ${st.control_api ? 'live path stats' : 'basic stats (mqvpn control API unavailable)'}`;
  // forget sparkline history for series that disappeared
  for (const [k, s] of state.hist) if (Date.now() - s.seen > 120000) state.hist.delete(k);
}

const IS_PHONE = /Android|iPhone|iPad/i.test(navigator.userAgent) || matchMedia('(pointer: coarse)').matches;

function setRole(role) {
  // the phone flow only makes sense on the server, viewed from the phone being added
  const phoneFlow = role === 'server' && (IS_PHONE || location.hash === '#add-phone');
  $('#add-phone').classList.toggle('hidden', !phoneFlow);
  // a phone here is being added: hide the server's own switch, modes and stats until asked
  if (!state.dashboardShown) document.body.classList.toggle('phone-add', phoneFlow);
  if (phoneFlow && !state.settings) loadSettings();
  state.role = role;
  for (const b of document.querySelectorAll('#tabs button[data-role]')) b.classList.toggle('hidden', b.dataset.role !== role);
  // each device picks its own mode; the server has none to choose
  $('#modes').classList.toggle('hidden', role === 'server');
  if (role === 'client') loadSettings();
  if (role === 'server') { loadSettings(); renderHealth(); }
}

function pathCard(key) {
  let el = state.cards.get(key);
  if (el) return el;
  el = h('article', { class: 'card path' });
  el.innerHTML = `
    <div class="path-head">
      <div class="kind-ico"></div>
      <div class="path-title"><b></b><small></small></div>
      <span class="chip"></span>
    </div>
    <svg class="path-spark" viewBox="0 0 300 46" preserveAspectRatio="none"></svg>
    <dl class="stats">
      <div class="stat"><dt>Down</dt><dd class="rx"></dd></div>
      <div class="stat"><dt>Up</dt><dd class="tx"></dd></div>
      <div class="stat"><dt>Latency</dt><dd class="lat"></dd></div>
      <div class="stat"><dt>Loss</dt><dd class="loss"></dd></div>
    </dl>
    <div class="path-foot hidden"></div>`;
  if (state.role === 'server') { // the server sees each device's upload as rx
    const dts = el.querySelectorAll('.stat dt');
    dts[0].textContent = 'From device'; dts[1].textContent = 'To device';
  }
  state.cards.set(key, el);
  return el;
}

function fillPathCard(el, p, key, opts) {
  const ki = $('.kind-ico', el);
  if (ki.dataset.kind !== p.kind) {
    ki.dataset.kind = p.kind;
    ki.className = 'kind-ico ' + p.kind;
    ki.replaceChildren(icon(p.kind in KIND_LABEL ? p.kind : 'unknown'));
  }
  $('.path-title b', el).textContent = opts.title || p.iface || `Path ${p.id}`;
  $('.path-title small', el).textContent = opts.subtitle || `${KIND_LABEL[p.kind] || 'Link'}${p.priority ? ' · ' + p.priority : ''}`;
  const chip = $('.chip', el);
  const st = p.enabled === false ? 'off' : p.state;
  chip.textContent = p.cap_reached ? 'cap reached' : (st === 'off' ? 'disabled' : st);
  chip.className = 'chip s-' + (p.cap_reached ? 'failed' : st);
  el.classList.toggle('off', p.enabled === false || p.state === 'failed');
  $('.rx', el).textContent = rateStr(p.rx_bps);
  $('.tx', el).textContent = rateStr(p.tx_bps);
  $('.lat', el).textContent = p.srtt_ms ? `${p.srtt_ms} ms` : '—';
  $('.loss', el).textContent = p.state === 'active' ? `${(p.loss_pct || 0).toFixed(1)}%` : '—';
  drawSpark($('.path-spark', el), push(key, p.rx_bps, p.tx_bps), 300, 46, { sw: 1.8 });
  if (opts.controls) fillControls(el, p);
}

function fillControls(el, p) {
  const foot = $('.path-foot', el);
  foot.classList.remove('hidden');
  if (!foot.dataset.ready) {
    foot.dataset.ready = '1';
    foot.innerHTML = `
      <div class="cap" style="flex:1">
        <div class="cap-row"><span class="used"></span><button type="button" class="cap-edit">Set cap</button></div>
        <div class="bar"><i></i></div>
      </div>
      <select class="prio" aria-label="Priority">
        <option value="primary">Primary</option><option value="secondary">Secondary</option><option value="backup">Backup</option>
      </select>
      <label class="toggle" title="Use this connection"><input type="checkbox" class="en" aria-label="Enabled"><span></span></label>`;
    const name = p.iface;
    $('.en', foot).addEventListener('change', (e) => updatePath({ name, enabled: e.target.checked }, e.target));
    $('.prio', foot).addEventListener('change', (e) => updatePath({ name, priority: e.target.value }, e.target));
    $('.cap-edit', foot).addEventListener('click', () => {
      const cur = state.status.paths.find((x) => x.iface === name);
      const v = prompt(`Monthly data cap for ${name} in MB (0 = unlimited).\nBackup paths drop out once they reach it.`, cur && cur.data_cap_mb ? cur.data_cap_mb : '0');
      if (v === null) return;
      const n = Number.parseInt(v, 10);
      if (!Number.isFinite(n) || n < 0) return toast('Enter a number of MB', true);
      updatePath({ name, data_cap_mb: n });
    });
  }
  const en = $('.en', foot), prio = $('.prio', foot);
  if (document.activeElement !== en && !en.dataset.busy) en.checked = p.enabled !== false;
  if (document.activeElement !== prio && !prio.dataset.busy) prio.value = p.priority || 'primary';
  const cap = p.data_cap_mb || 0;
  const used = p.used_bytes || 0;
  $('.used', foot).textContent = cap ? `${fmtBytes(used)} of ${fmtBytes(cap * 1048576)}` : `${fmtBytes(used)} this month`;
  const pct = cap ? Math.min(100, (used / (cap * 1048576)) * 100) : 0;
  const bar = $('.bar', foot);
  bar.classList.toggle('hidden', !cap);
  bar.classList.toggle('hot', pct >= 90);
  $('.bar > i', foot).style.width = pct + '%';
}

async function updatePath(body, ctl) {
  if (ctl) ctl.dataset.busy = '1';
  try {
    await api('POST', '/api/paths', body);
    toast('Saved');
  } catch (e) {
    toast(e.message, true);
  } finally {
    if (ctl) setTimeout(() => delete ctl.dataset.busy, 1500);
  }
}

function syncChildren(container, els) {
  // keep existing nodes in place to preserve focus; only move when order changes
  els.forEach((el, i) => { if (container.children[i] !== el) container.insertBefore(el, container.children[i] || null); });
  while (container.children.length > els.length) container.lastElementChild.remove();
}

function renderClientPaths(st) {
  const els = [];
  const paths = [...st.paths].sort((a, b) => (a.state === 'active' ? 0 : 1) - (b.state === 'active' ? 0 : 1) || a.iface.localeCompare(b.iface));
  for (const p of paths) {
    const key = 'p:' + p.iface;
    const el = pathCard(key);
    fillPathCard(el, p, key, { controls: true });
    els.push(el);
  }
  syncChildren($('#paths'), els);
  const empty = $('#paths-empty');
  if (!els.length) {
    empty.innerHTML = st.paired
      ? '<b>No connections yet</b>Waiting for mqvpn to bring up its paths.'
      : '<b>Not paired</b>Open the Setup tab and paste the pairing link from your home server.';
    empty.classList.remove('hidden');
  } else empty.classList.add('hidden');
}

function renderServerPaths(st) {
  const els = [];
  const live = st.clients.filter((c) => c.connected);
  for (const c of live) {
    const gkey = 'g:' + c.name + c.endpoint;
    let g = state.cards.get(gkey);
    if (!g) { g = h('div', { class: 'group-title' }); state.cards.set(gkey, g); }
    g.replaceChildren(icon('device', 'ico sm'), c.name, h('span', { class: 'muted', text: `${c.endpoint} · ${c.paths.filter((p) => p.state === 'active').length} active path(s) · ${fmtDur(c.connected_sec)}` }));
    els.push(g);
    for (const p of c.paths) {
      const key = `c:${c.name}:${c.endpoint}:${p.id}`;
      const el = pathCard(key);
      fillPathCard(el, p, key, { title: `Path ${p.id}`, subtitle: `${c.name} · ${p.min_rtt_ms ? 'min ' + p.min_rtt_ms + ' ms' : 'measuring'}` });
      els.push(el);
    }
  }
  syncChildren($('#paths'), els);
  const empty = $('#paths-empty');
  if (!live.length) {
    empty.innerHTML = st.state === 'connected'
      ? `<b>No devices connected</b>${st.clients.length ? 'Registered devices appear here with every link they bond.' : 'Add a device in the Clients tab to get a pairing QR code.'}`
      : `<b>Server ${st.state}</b>${st.error ? '' : 'mqvpn is starting…'}`;
    empty.classList.remove('hidden');
  } else empty.classList.add('hidden');
  for (const k of [...state.cards.keys()]) if (!els.includes(state.cards.get(k)) && !k.startsWith('p:')) state.cards.delete(k);
}

// the page's MagicDNS address works on any tailnet phone; fall back to how we were reached
// Setup health (GET /api/health): green when all good, else one line per
// problem with a "How to fix" link to Tailscale's docs.
async function renderHealth() {
  let v;
  try { v = await api('GET', '/api/health'); } catch { return; }
  const box = $('#health');
  const bad = v.checks.filter((c) => c.status === 'warn' || c.status === 'bad');
  const blocking = v.checks.filter((c) => c.status === 'bad').length;
  // the route is a choice (tailnet → bonded devices), not a speed problem
  const optional = bad.filter((c) => c.id === 'route').length;
  const security = bad.filter((c) => c.id === 'approvers').length;
  const improve = bad.length - blocking - optional - security;
  const plus = optional ? ` (and ${optional} optional)` : '';
  const headline = blocking
    ? `Setup health: ${blocking} thing${blocking === 1 ? '' : 's'} to fix before devices can connect from away${improve ? `, ${improve} more to improve` : ''}${plus}`
    : security ? `Setup health: 1 security setting to review${improve ? `, ${improve} thing${improve === 1 ? '' : 's'} to improve` : ''}${plus}`
    : improve ? `Setup health: ${improve} optional speed-up${improve === 1 ? '' : 's'} (tailbond works fine without ${improve === 1 ? 'it' : 'them'})${plus}`
    : optional ? 'Setup health: nothing to fix here (1 optional item below)' : 'Setup health: everything is set up';
  const addrTodo = state.settings && state.settings.public_addr_guessed;
  const seen = new Set();
  const rows = (bad.length ? bad : v.checks).map((c) => {
    let fix = c.fix && c.status !== 'ok' ? c.fix : '';
    const again = fix && seen.has(fix);
    if (fix) seen.add(fix);
    if (again) fix = 'Same fix as above (one forward fixes both).';
    const test = '';
    return h('li', { class: 'health-' + c.status },
      h('b', { text: c.title }),
      c.detail ? h('div', { class: 'muted small', text: c.detail }) : null,
      fix ? h('div', { class: 'small' }, fix + ' ', c.fix_url && !again ? h('a', { href: c.fix_url, target: '_blank', rel: 'noopener', text: 'How to fix' }) : null) : null,
      test ? h('div', { class: 'small', text: test }) : null);
  });
  box.replaceChildren(...[
    h('h3', { text: addrTodo && !blocking ? headline + ' · finish the setup step above first' : headline }),
    h('ul', { class: 'health-list' }, rows),
    bad.length ? h('p', { class: 'small', text: (security ? 'The security setting is fixed on this page (Who can approve and add devices). ' : '') + 'Router fixes are on your home router\'s admin page, route ones in the Tailscale admin console ("How to fix" explains each); this machine\'s own firewall is already set up by tailbond. Not sure how? Show this card to whoever set up your internet.' }) : null,
    h('p', { class: 'muted small' }, 'Direct connections are faster than relayed ones. ',
      h('a', { href: 'https://tailscale.com/docs/reference/connection-types', target: '_blank', rel: 'noopener', text: 'About Tailscale connection types' }))].filter(Boolean));
  box.classList.remove('hidden');
  const ban = $('#health-banner');
  ban.replaceChildren();
  const addr = state.settings && state.settings.public_addr_guessed;
  const shown = blocking + improve + security;
  if (shown || addr) {
    ban.append(addr ? 'Setup: tell devices where home is (public address). ' : '', shown ? headline + '. ' : '',
      h('a', { href: '#', text: 'Show', onclick: (e) => { e.preventDefault(); $('#tabs button[data-tab="clients"]').click(); } }));
  }
  ban.classList.toggle('blocking', !!(blocking || addr));
  ban.classList.toggle('hidden', !(shown || addr) || state.tab === 'clients');
}

function serverShort() { return (state.status && state.status.tailscale && state.status.tailscale.self) || 'this server'; }
// Who may approve and add devices, in one sentence.
function whoApproves() {
  const s = state.settings;
  if (!s) return '';
  if (s.password_set) return 'Who can approve: anyone with this page\'s password (you too), from any device on your Tailscale network.';
  const names = [s.owner ? `${s.owner} (the Tailscale account ${serverShort()} belongs to)` : '', ...(s.approvers || [])].filter(Boolean);
  if (names.length) return `Who can approve: ${names.join(', ')}, from any device signed in to that Tailscale account, and any user or program on ${serverShort()} itself (http://127.0.0.1:8077; set a password if ${serverShort()} runs things you don't trust). Others on your Tailscale network, and people you share ${serverShort()} with, can ask to connect but can't approve.`;
  return `Who can approve: anyone on your Tailscale network who can reach ${serverShort()} (it's a tagged machine, so it has no owner account). Name approvers below, or set a password.`;
}

// Devices on the tailnet that asked to connect (POST /api/pair-requests).
function renderPairRequests(st) {
  const box = $('#pair-requests');
  const reqs = st.role === 'server' && Array.isArray(st.pair_requests) ? st.pair_requests : [];
  const sig = reqs.map((r) => r.id).join(',');
  box.classList.toggle('hidden', !reqs.length);
  // on a phone with requests waiting, whoever opened this is here to approve,
  // not to add this phone: show the requests instead of "Add this phone"
  if (reqs.length && location.hash !== '#add-phone' && !$('#add-phone').classList.contains('hidden')) {
    $('#add-phone').classList.add('hidden');
    document.body.classList.remove('phone-add');
  }
  if (box.dataset.sig === sig) return;
  box.dataset.sig = sig;
  const who = h('p', { class: 'muted small', text: whoApproves() + ` Approving lets the device use ${serverShort()}'s internet connection (by default all its traffic goes out from here) and your home network, and reach your Tailscale machines as ${serverShort()}: they allow it whatever they allow ${serverShort()}, which may be more than its owner's own account gets.` });
  const approvers = new Set([state.settings && state.settings.owner, ...((state.settings && state.settings.approvers) || [])].filter(Boolean).map((x) => x.toLowerCase()));
  box.replaceChildren(h('h2', { text: 'Waiting for your approval' }), who, ...reqs.map((r) => {
    const what = r.kind === 'router' ? 'Travel router' : r.kind === 'phone' ? 'Phone' : 'Device';
    const decide = async (action, btn) => {
      btn.disabled = true;
      try {
        await api('POST', `/api/pair-requests/${r.id}/${action}`, {});
        toast(action === 'approve' ? `${r.name} can connect now. It connects by itself in a few seconds.` : 'Declined');
        box.dataset.sig = '';
        pollOnce();
      } catch (e) { btn.disabled = false; toast(e.message, true); }
    };
    return h('div', { class: 'pair-req' },
      h('div', { class: 'pair-req-text' },
        h('b', { text: r.repair ? `${r.name} is asking to connect again` : `${r.name} wants to connect` }),
        h('div', { class: 'muted small', text: `${what} on your tailnet · signed in as ${r.user || 'unknown'} · ${r.hostname ? r.hostname + ' · ' : ''}${r.ip}` }),
        r.repair ? h('div', { class: 'small', text: 'It was connected before (reset or reinstalled). Approving gives it its old access back.' }) : null,
        r.user && approvers.has(r.user.toLowerCase()) ? h('div', { class: 'small warn-text', text: `Signed in to ${r.user}, an account that may approve here. If this ${r.kind === 'router' ? 'router' : 'device'} stays with you, just approve. If you'll give it to someone: approve, then before handing it over sign its Tailscale in to their account (or tag it); tailbond keeps working, and until then whoever holds it could also approve devices here. Decline only refuses this request; it can ask again.` }) : null),
      h('div', { class: 'pair-req-actions' },
        h('button', { class: 'btn', type: 'button', text: 'Decline', onclick: (e) => decide('decline', e.currentTarget) }),
        h('button', { class: 'btn primary', type: 'button', text: 'Approve', onclick: (e) => decide('approve', e.currentTarget) })));
  }));
}

function renderClients(st) {
  const list = $('#clients');
  const els = [];
  for (const c of st.clients) {
    let el = state.clientEls.get(c.name);
    if (!el) {
      el = h('article', { class: 'card client' });
      el.innerHTML = `
        <div class="client-head">
          <div class="kind-ico"></div>
          <div class="client-name"><b></b><small></small></div>
          <div class="client-rates"><span class="rx"></span><span class="tx"></span></div>
          <span class="chip"></span>
          <button class="btn icon" type="button" title="Show pairing QR"></button>
          <button class="btn icon danger" type="button" title="Revoke"></button>
        </div>
        <div class="client-paths"></div>`;
      $('.kind-ico', el).append(icon('device'));
      const [qrBtn, delBtn] = el.querySelectorAll('button');
      qrBtn.append(icon('qr', 'ico sm'));
      delBtn.append(icon('trash', 'ico sm'));
      qrBtn.addEventListener('click', () => showPairing(c.name));
      delBtn.addEventListener('click', () => revoke(c.name));
      qrBtn.classList.toggle('hidden', !c.registered);
      delBtn.classList.toggle('hidden', !c.registered);
      state.clientEls.set(c.name, el);
    }
    $('.client-name b', el).textContent = c.name;
    $('.client-name small', el).textContent = c.connected ? `${c.endpoint} · ${c.scheduler || ''}` : 'not connected right now (approved devices connect by themselves when they\'re on and online)';
    $('.client-rates .rx', el).textContent = c.connected ? '↓ ' + rateStr(c.tx_bps) : '';
    $('.client-rates .tx', el).textContent = c.connected ? '↑ ' + rateStr(c.rx_bps) : '';
    const chip = $('.chip', el);
    chip.textContent = c.connected ? `${c.paths.filter((p) => p.state === 'active').length} paths` : 'not connected';
    chip.className = 'chip s-' + (c.connected ? 'active' : 'off');
    $('.client-paths', el).replaceChildren(...(c.connected ? c.paths : []).map((p) =>
      h('span', { class: 'mini-path' }, h('b', { text: `#${p.id} ` }), `${p.state} · ${p.srtt_ms} ms · ${(p.loss_pct || 0).toFixed(1)}%`)));
    els.push(el);
  }
  syncChildren(list, els);
  for (const k of [...state.clientEls.keys()]) if (!st.clients.some((c) => c.name === k)) state.clientEls.delete(k);
  if (!els.length) list.replaceChildren(h('div', { class: 'empty' }, h('b', { text: 'No devices yet' }), 'Add your phone or travel router above.'));
}

async function showPairing(name, res) {
  try {
    res = res || (await api('GET', '/api/clients/' + encodeURIComponent(name)));
  } catch (e) { return toast(e.message, true); }
  $('#qr-title').textContent = `Pair “${res.name}”`;
  // qr_svg is generated server-side from our own data; parse it as SVG, not HTML.
  const doc = new DOMParser().parseFromString(res.qr_svg, 'image/svg+xml');
  $('#qr-box').replaceChildren(document.importNode(doc.documentElement, true));
  $('#qr-uri').value = res.uri;
  $('#qr-dialog').showModal();
}

async function revoke(name) {
  if (!confirm(`Revoke “${name}”? The device disconnects and its key stops working.`)) return;
  try {
    await api('DELETE', '/api/clients/' + encodeURIComponent(name));
    state.clientEls.delete(name);
    toast(`${name} revoked`);
  } catch (e) { toast(e.message, true); }
}

async function loadSettings() {
  try {
    state.settings = await api('GET', '/api/settings');
  } catch { return; }
  // the approval card's "who can approve" depends on settings: redraw it
  $('#pair-requests').dataset.sig = '';
  if (state.status) renderPairRequests(state.status);
  if (state.status) $('#subtitle').textContent = state.status.role === 'server'
    ? `Home server · reachable at ${state.settings.effective_public_addr || state.status.server || 'no public address'}`
    : $('#subtitle').textContent;
  const s = state.settings;
  if (s.role === 'server') {
    $('#apk-yes').classList.toggle('hidden', !s.android_apk);
    $('#apk-no').classList.toggle('hidden', !!s.android_apk);
    $('#ipk-yes').classList.toggle('hidden', !s.glinet_ipk);
    $('#pw-state').textContent = s.password_set
      ? 'Only people with this page\'s password, from any device on your Tailscale network. Change it, or save an empty one to remove it. Family adding a phone don\'t need it: their phone asks, you approve.'
      : whoApproves() + ' A device you lend to someone while it\'s signed in to an approver\'s account can approve too. To let someone else approve, add their Tailscale login (shown in the Tailscale admin console under Users) below. A password instead replaces all this: then everyone who approves, you included, needs it; phones and routers can still ask to connect without it.';
    $('#approvers').value = (s.approvers || []).join(', ');
    $('#approvers-label').textContent = s.owner ? 'Also allowed to approve (Tailscale logins, comma separated)' : 'Allowed to approve (Tailscale logins, comma separated; on this tagged server, only they can)';
    $('#ipk-no').classList.toggle('hidden', !!s.glinet_ipk);
    // where phones get the app (the landing page on port 80, else /get)
    if (s.landing_url) {
      $('#invite-link').value = s.landing_url;
      $('#landing-typed').textContent = s.landing_typed;
      if (s.landing_qr_svg) {
        const doc = new DOMParser().parseFromString(s.landing_qr_svg, 'image/svg+xml');
        $('#landing-qr').replaceChildren(document.importNode(doc.documentElement, true));
      }
    }
    $('#landing-no-apk').classList.toggle('hidden', !!s.android_apk);
    const w = $('#addr-warning');
    if (s.public_addr_guessed) {
      w.innerHTML = '';
      const lan = (s.effective_public_addr || '').replace(/:\d+$/, '') || 'this machine';
      w.append(
        h('h3', { text: 'Setup step: tell devices where home is' }),
        h('p', { text: `Right now devices would connect to ${s.effective_public_addr || 'no address'}, an address inside your home, so they only work at home. Two things, on your home router's admin page:` }),
        h('ol', {},
          h('li', { text: `Forward UDP port ${s.port} to this machine (${lan}). The page is usually called Port forwarding, Virtual server or NAT. Give this machine a fixed address there too (DHCP reservation), so the forward keeps pointing at it.` }),
          h('li', { text: 'Find your home\'s public address: the router\'s status page shows it (WAN or Internet IP). Better, a dynamic DNS name (many routers have a DDNS page) keeps working when that address changes. Type it here:' })),
        h('div', { class: 'row' },
          h('input', { id: 'pub-addr', placeholder: 'e.g. home.example.net or 203.0.113.7' }),
          h('button', { class: 'btn primary', type: 'button', onclick: savePublicAddr, text: 'Save' })),
        h('p', { class: 'muted small', text: `The port (${s.port}) is added for you. If your provider shares one internet address between customers (carrier-grade NAT), a forward can't work; Setup health below checks for that and says what to do.` }));
      w.classList.remove('hidden');
    } else w.classList.add('hidden');
  } else {
    $('#pair-current').textContent = s.paired_with ? `Currently paired with ${s.paired_with} as “${s.paired_as}”.` : '';
  }
}

async function savePublicAddr() {
  const v = $('#pub-addr').value.trim();
  if (!v) return;
  try {
    await api('POST', '/api/settings', { public_addr: v });
    toast('Public address saved');
    loadSettings();
  } catch (e) { toast(e.message, true); }
}

async function renderTailscale() {
  const panel = $('#ts-panel');
  let ts;
  try { ts = await api('GET', '/api/tailscale'); } catch (e) { panel.textContent = e.message; return; }
  const s = state.settings || {};
  const head = h('div', { class: 'ts-head' },
    h('div', { class: 'kind-ico' }, icon('ts')),
    h('div', {}, h('h3', { text: 'Tailscale' }),
      h('span', { class: 'chip s-' + (ts.running ? 'active' : ts.installed ? 'pending' : 'off'), text: ts.running ? 'running' : ts.installed ? (ts.backend_state || 'stopped') : 'not installed' })));
  const items = [
    ['Machine', ts.dns_name || ts.self || '—'],
    ['Tailnet', ts.tailnet || '—'],
    ['Tailscale IPs', (ts.ips || []).join(', ') || '—'],
    ['Peers online', ts.running ? `${ts.peers_online} / ${ts.peers}` : '—'],
  ];
  if (state.role === 'server') {
    items.push(['Tunnel route', `${ts.route || s.subnet || ''} · ${ts.routes_approved ? 'approved' : ts.route_advertised ? 'awaiting approval' : 'not advertised'}`]);
    items.push(['TLS certificate', s.tls === 'tailscale' ? `Let's Encrypt via Tailscale (${s.tls_name})` : 'self-signed (pinned in pairing links)']);
  }
  const grid = h('dl', { class: 'ts-grid' }, items.map(([k, v]) => h('div', { class: 'ts-item' }, h('dt', { text: k }), h('dd', { text: v }))));
  const kids = [head, grid];
  if (state.role === 'server') {
    const steps = h('ol', { class: 'steps' });
    if (!ts.installed) steps.append(h('li', {}, 'Install Tailscale and run ', h('code', { text: 'sudo tailscale up' }), ' to reach your tailnet from bonded devices. Everything else works without it.'));
    else if (!ts.running) steps.append(h('li', {}, 'Run ', h('code', { text: 'sudo tailscale up' }), '.'));
    else {
      if (!ts.route_advertised) steps.append(h('li', {}, 'The server-up hook advertises ', h('code', { text: ts.route || '' }), '; start the server to apply it.'));
      if (!ts.routes_approved) steps.append(h('li', {}, 'Approve the subnet route ', h('code', { text: ts.route || '' }), ' for this machine in the ', h('a', { href: ts.admin_url || 'https://login.tailscale.com/admin/machines', target: '_blank', rel: 'noopener', text: 'admin console' }), ' (Edit route settings). Tailnet peers can then reach bonded devices by tunnel IP.'));
      if (s.tls !== 'tailscale') steps.append(h('li', {}, 'Optional: enable HTTPS certificates in the admin console (DNS → HTTPS Certificates) so devices verify the server by name instead of a pinned self-signed certificate.'));
      if (!steps.children.length) steps.append(h('li', { text: 'All set: bonded devices reach your tailnet through this server.' }));
    }
    kids.push(steps);
  }
  panel.replaceChildren(...kids);
}

async function pollLogs() {
  if (state.tab !== 'logs') return;
  try {
    const { entries } = await api('GET', `/api/logs?since=${state.logSeq}&limit=1000`);
    if (!entries.length) return;
    const pre = $('#logs');
    const frag = document.createDocumentFragment();
    for (const e of entries) {
      state.logSeq = e.seq;
      const t = new Date(e.t).toLocaleTimeString([], { hour12: false });
      frag.append(h('div', {}, h('span', { class: 't', text: t + ' ' }), h('span', { class: 'src-' + e.src, text: `[${e.src}] ` }), e.line));
    }
    pre.append(frag);
    while (pre.childElementCount > 2000) pre.firstElementChild.remove();
    if ($('#logs-follow').checked) pre.scrollTop = pre.scrollHeight;
  } catch { /* ignore */ }
}

// ---------- actions ----------
function setupModes() {
  for (const b of document.querySelectorAll('#modes button')) {
    b.addEventListener('click', async () => {
      if (!state.status || b.dataset.mode === state.status.mode) return;
      state.modeBusy = true;
      for (const x of document.querySelectorAll('#modes button')) { x.setAttribute('aria-checked', String(x === b)); x.disabled = true; }
      try {
        await api('POST', '/api/mode', { mode: b.dataset.mode });
        toast(`${b.querySelector('b').textContent} mode — reconnecting`);
      } catch (e) { toast(e.message, true); }
      setTimeout(() => {
        state.modeBusy = false;
        for (const x of document.querySelectorAll('#modes button')) x.disabled = false;
      }, 1200);
    });
  }
}

function setupPower() {
  $('#power').addEventListener('click', async () => {
    const st = state.status;
    if (!st) return;
    const on = st.state !== 'disconnected';
    if (on && st.role === 'server' && !confirm('Stop the bonding server? Connected devices lose their tunnel.')) return;
    state.powerBusyUntil = Date.now() + 2500;
    $('#power').setAttribute('aria-pressed', String(!on));
    $('#power-label').textContent = on ? 'Disconnecting…' : 'Connecting…';
    try { await api('POST', on ? '/api/disconnect' : '/api/connect'); } catch (e) { state.powerBusyUntil = 0; toast(e.message, true); }
  });
}

function setupTabs() {
  for (const b of document.querySelectorAll('#tabs button')) {
    b.addEventListener('click', () => {
      state.tab = b.dataset.tab;
      for (const x of document.querySelectorAll('#tabs button')) x.classList.toggle('active', x === b);
      for (const s of document.querySelectorAll('.tab')) s.classList.toggle('hidden', s.id !== 'tab-' + state.tab);
      if (state.tab === 'tailscale') renderTailscale();
      if (state.tab === 'logs') pollLogs();
      if (state.tab === 'clients' && state.status) { loadSettings(); renderClients(state.status); renderHealth(); }
      if (state.tab === 'setup') loadSettings();
      if (state.tab === 'maint') ubOpen();
    });
  }
}

function setupPassword() {
  $('#pw-form').addEventListener('submit', async (e) => {
    e.preventDefault();
    const v = $('#pw-new').value;
    try {
      await api('POST', '/api/settings', { ui_password: v });
      $('#pw-new').value = '';
      toast(v ? 'Password set: approving and adding devices now needs it' : 'Password removed');
      await loadSettings();
    } catch (err) { toast(err.message, true); }
  });
}

function setupClients() {
  $('#add-client').addEventListener('submit', async (e) => {
    e.preventDefault();
    const input = $('#client-name');
    const name = input.value.trim();
    if (!name) return;
    try {
      const res = await api('POST', '/api/clients', { name });
      input.value = '';
      showPairing(name, res);
    } catch (err) { toast(err.message, true); }
  });
  $('#qr-copy').addEventListener('click', async () => {
    const v = $('#qr-uri').value;
    try { await navigator.clipboard.writeText(v); toast('Link copied'); } catch { $('#qr-uri').select(); document.execCommand('copy'); toast('Link copied'); }
  });
}

// "pixel-8" from "Android 14; Pixel 8)"; reduced user agents ("Android 10; K") give "phone"
function guessPhoneName() {
  const m = /Android [\d.]+; ([^;)]+)\)/.exec(navigator.userAgent);
  const model = m && m[1].trim();
  if (!model || model.length < 2) return 'phone';
  return model.toLowerCase().replace(/[^a-z0-9._-]+/g, '-').replace(/^-+|-+$/g, '').slice(0, 40) || 'phone';
}

function setupAddPhone() {
  const input = $('#phone-name');
  input.value = guessPhoneName();
  if (location.hash === '#add-phone') $('#add-phone').scrollIntoView();
  let uri = '';
  $('#add-phone-btn').addEventListener('click', async () => {
    const name = input.value.trim();
    if (!input.checkValidity() || !name) { input.focus(); return toast('Use letters, digits, dots or dashes for the name', true); }
    const btn = $('#add-phone-btn');
    btn.disabled = true;
    let res;
    try {
      res = await api('POST', '/api/clients', { name });
    } catch (err) {
      if (!/already exists/.test(err.message)) { btn.disabled = false; return toast(err.message, true); }
      // tapped twice, or re-adding the same phone: reuse that device's key
      if (!confirm(`A device called “${name}” already exists. Use it on this phone?`)) { btn.disabled = false; return; }
      try { res = await api('GET', '/api/clients/' + encodeURIComponent(name)); } catch (e2) { btn.disabled = false; return toast(e2.message, true); }
    }
    btn.disabled = false;
    uri = res.uri;
    $('#phone-open').href = uri;
    for (const el of document.querySelectorAll('.server-name')) el.textContent = state.status?.tailscale?.self || 'this server';
    $('#add-phone-done').classList.remove('hidden');
    btn.classList.add('hidden');
    location.href = uri; // the tailbond app handles tailbond://pair
  });
  $('#approvers-save').addEventListener('click', async () => {
    const list = $('#approvers').value.split(',').map((x) => x.trim()).filter(Boolean);
    try { await api('POST', '/api/settings', { approvers: list }); toast(list.length ? 'Approvers saved' : 'Approvers removed'); loadSettings(); } catch (e) { toast(e.message, true); }
  });
  $('#invite-copy').addEventListener('click', async () => {
    try { await navigator.clipboard.writeText($('#invite-link').value); } catch { $('#invite-link').select(); document.execCommand('copy'); }
    toast('Link copied');
  });
  $('#ub-dialog').addEventListener('click', (e) => { if (e.target === e.currentTarget) closeDlg(); });
  $('#show-dashboard').addEventListener('click', () => {
    state.dashboardShown = true;
    document.body.classList.remove('phone-add');
    $('#show-dashboard').classList.add('hidden');
  });
  $('#phone-copy').addEventListener('click', async () => {
    try { await navigator.clipboard.writeText(uri); } catch { /* insecure context: fall through */ }
    toast('Link copied');
  });
}

function setupPairing() {
  $('#pair-form').addEventListener('submit', async (e) => {
    e.preventDefault();
    const uri = $('#pair-uri').value.trim();
    if (!uri) return;
    try {
      const r = await api('POST', '/api/pair', { uri });
      $('#pair-uri').value = '';
      toast(`Paired with ${r.server}`);
      loadSettings();
      $('#tabs button[data-tab="connections"]').click();
    } catch (err) { toast(err.message, true); }
  });
  const scanBtn = $('#scan-btn');
  if (!('BarcodeDetector' in window) || !navigator.mediaDevices) { scanBtn.classList.add('hidden'); return; }
  scanBtn.addEventListener('click', async () => {
    const video = $('#scan-video');
    let stream;
    try {
      stream = await navigator.mediaDevices.getUserMedia({ video: { facingMode: 'environment' } });
    } catch { return toast('Camera unavailable (needs HTTPS or localhost)', true); }
    video.srcObject = stream; video.classList.remove('hidden'); await video.play();
    const det = new BarcodeDetector({ formats: ['qr_code'] });
    const stop = () => { stream.getTracks().forEach((t) => t.stop()); video.classList.add('hidden'); };
    const started = Date.now();
    const tick = async () => {
      try {
        const codes = await det.detect(video);
        const hit = codes.find((c) => c.rawValue.startsWith('tailbond://'));
        if (hit) { stop(); $('#pair-uri').value = hit.rawValue; toast('QR code read — press Pair'); return; }
      } catch { /* keep trying */ }
      if (Date.now() - started > 60000) return stop();
      requestAnimationFrame(tick);
    };
    tick();
  });
}

// Without a password, only the server owner's devices may use this page
// from the tailnet; others asked to connect, or want the app.
function showOwnerOnly(owner) {
  const d = $('#owner-dialog');
  $('#owner-who').textContent = owner;
  $('#owner-get').href = '/get';
  if (!d.open) d.showModal();
}
function showLogin() {
  const d = $('#login-dialog');
  if (!d.open) d.showModal();
}
function setupLogin() {
  $('#login-form').addEventListener('submit', async (e) => {
    e.preventDefault();
    try {
      const r = await fetch('/api/login', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ password: $('#login-pw').value }) });
      if (!r.ok) throw new Error((await r.json()).error || 'login failed');
      $('#login-dialog').close();
      connect();
    } catch (err) { $('#login-err').textContent = err.message; }
  });
}

// ---------- Updates & backup tab ----------
// Updates (docs/architecture.md "Updates"): GET /api/update (+ the live
// `update` object in every status), POST /api/update/check,
// /api/update/install, /api/update/policy {"policy"}. Backups: POST
// /api/backup → file, GET /api/backup/summary, POST /api/restore/preview,
// POST /api/restore, GET /api/backups (+ /api/backups/{name}),
// POST /api/backup/settings. Scheduled-backup settings are read from
// GET /api/settings `backup` (never the passphrase, only `passphrase_set`).
const KIND_NAME = { router: 'travel router', android: 'Android phone', phone: 'Android phone' };
const ub = { upd: null, backups: null, sig: '', bsig: '', open: false, schedEdit: false };
const DNF_CMD = 'sudo dnf upgrade tailbond-server';

function fmtWhen(t) {
  if (!t) return '';
  return new Date(t).toLocaleString([], { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit', hourCycle: 'h23' });
}
function fmtAgo(t) {
  const s = (Date.now() - new Date(t).getTime()) / 1000;
  if (!t || !Number.isFinite(s)) return '';
  if (s < 90) return 'just now';
  if (s < 5400) return `${Math.round(s / 60)} min ago`;
  if (s < 36 * 3600) return `${Math.round(s / 3600)} h ago`;
  return `${Math.round(s / 86400)} days ago`;
}
function updOf(st) { return Object.assign({}, ub.upd || {}, (st && st.update) || {}); }
function homeName() {
  const st = state.status || {};
  return st.active_server || ((state.settings && state.settings.paired_with) || st.server || 'your home server').split(/[.:]/)[0];
}
// the GitHub releases page of update.repo ("owner/name" or a github.com URL)
function releasesUrl(u) {
  const r = String(u.repo || 'kaovilai/tailbond').replace(/^https?:\/\/github\.com\//, '').replace(/\/+$/, '');
  return `https://github.com/${r}/releases`;
}
// methods that install by themselves (one click, or Automatic)
const selfUpdates = (u) => u.method === 'static' || u.method === 'opkg';

function renderMaint(st) {
  const u = st.update || {};
  const outdated = st.role === 'server' && Array.isArray(u.outdated) ? u.outdated.length : 0;
  const badge = $('#maint-badge');
  const bad = u.state === 'rolled_back' || u.state === 'failed';
  const n = (u.available && u.policy !== 'off' ? 1 : 0) + outdated;
  badge.textContent = bad ? '!' : String(n);
  badge.className = 'tab-badge' + (bad ? ' bad' : '') + (n || bad ? '' : ' hidden');
  if (state.tab !== 'maint') return;
  renderUpdates(st);
}

async function ubOpen() {
  const jobs = [api('GET', '/api/update').then((r) => { ub.upd = r; }).catch(() => {}), loadSettings()];
  if (state.role === 'server') jobs.push(api('GET', '/api/backups').then((r) => { ub.backups = Array.isArray(r) ? r : (r.backups || []); }).catch(() => { ub.backups = null; }));
  await Promise.all(jobs);
  ub.sig = ''; ub.bsig = '';
  if (state.status) renderUpdates(state.status);
  renderBackups();
}

function seg(options, value, onPick, label) {
  return h('div', { class: 'seg', role: 'radiogroup', 'aria-label': label },
    options.map(([v, text]) => h('button', { type: 'button', role: 'radio', 'aria-checked': String(v === value), class: v === value ? 'on' : '', onclick: () => { if (v !== value) onPick(v); } }, text)));
}

async function updAction(path, body, okMsg) {
  try {
    const r = await api('POST', path, body);
    if (okMsg) toast(okMsg);
    await ubOpen();
    return r;
  } catch (e) { toast(e.message, true); return null; }
}

async function copyText(text, okMsg) {
  try { await navigator.clipboard.writeText(text); } catch {
    const t = h('textarea', { style: 'position:fixed;opacity:0' }); t.value = text; document.body.append(t); t.select();
    try { document.execCommand('copy'); } catch { /* nothing more to try */ }
    t.remove();
  }
  toast(okMsg || 'Copied');
}

function renderUpdates(st) {
  const box = $('#upd-card');
  const u = updOf(st);
  const srv = st.role === 'server';
  const sig = JSON.stringify([u, srv, (st.clients || []).length]);
  if (sig === ub.sig) return;
  ub.sig = sig;
  const cur = u.current || st.version || '';
  const policy = u.policy || 'notify';
  const off = policy === 'off' || u.state === 'off';
  const busy = ['checking', 'downloading', 'installing'].includes(u.state);
  const chip = u.state === 'rolled_back' ? ['s-failed', 'Went back to ' + (cur || 'the previous version')]
    : u.state === 'failed' ? ['s-failed', 'Update failed']
    : u.state === 'installing' ? ['s-pending', 'Installing']
    : u.state === 'checking' ? ['s-pending', 'Checking']
    : u.state === 'downloading' ? ['s-pending', 'Downloading']
    : off ? ['s-off', 'Updates off']
    : u.available ? ['s-pending', 'Update available'] : ['s-active', 'Up to date'];
  const kids = [h('div', { class: 'ub-head' },
    h('div', { class: 'kind-ico' }, icon('update')),
    h('div', { class: 'grow' }, h('h3', { text: 'Updates' }), h('div', { class: 'muted small', text: `This ${srv ? 'server' : 'machine'} runs tailbond ${cur}` })),
    h('span', { class: 'chip ' + chip[0], text: chip[1] }))];

  // what's going on now
  const now = [];
  const notes = u.notes_url ? h('a', { href: u.notes_url, target: '_blank', rel: 'noopener', text: "What's new" }) : null;
  const tried = u.last_result_version || u.available || 'The new version';
  if (u.state === 'rolled_back') {
    now.push(h('div', { class: 'ub-note bad' }, h('b', { text: `${tried} didn't start here, so tailbond went back to ${cur} by itself. ` }),
      u.detail ? `Why: ${u.detail}. ` : '', 'Everything works as before.'));
  } else if (u.state === 'failed') {
    now.push(h('div', { class: 'ub-note bad' }, h('b', { text: `Couldn't update to ${tried}. ` }),
      u.error ? `${u.error}. ` : '', `Still on ${cur}, nothing changed.`));
  }
  if (u.state === 'checking') now.push(h('p', { class: 'ub-big' }, h('span', { class: 'spin' }), 'Checking for updates…'));
  else if (u.state === 'downloading') now.push(h('p', { class: 'ub-big' }, h('span', { class: 'spin' }), `Downloading ${u.available}…`));
  else if (u.state === 'installing') now.push(h('p', { class: 'ub-big' }, h('span', { class: 'spin' }), `Installing ${u.available}`),
    h('p', { class: 'muted small', text: 'tailbond restarts; this page comes back by itself in about a minute.' }));
  else if (off) now.push(h('p', { class: 'ub-big', text: 'Updates are off' }),
    h('p', { class: 'muted small', text: "tailbond doesn't check for new versions. Choose Tell me below to hear about them." }));
  else if (u.available) {
    now.push(h('p', { class: 'ub-big' }, `Update available: ${u.available}`, notes ? ' · ' : '', notes));
    if (u.method === 'dnf') {
      now.push(h('p', { class: 'small' }, 'Updates for this server come from dnf: run ', h('code', { text: DNF_CMD }), ', or turn on dnf-automatic to get them by themselves.'),
        h('div', { class: 'ub-cmd' }, h('code', { text: DNF_CMD }),
          h('button', { class: 'btn', type: 'button', onclick: () => copyText(DNF_CMD, 'Command copied') }, icon('copy', 'ico sm'), 'Copy')));
    } else if (!selfUpdates(u)) {
      now.push(h('p', { class: 'small' }, "This build can't update itself; install the new release from ",
        h('a', { href: releasesUrl(u), target: '_blank', rel: 'noopener', text: 'the releases page' }), '.'));
    }
  } else now.push(h('p', { class: 'ub-big', text: 'Up to date' }));

  const meta = [];
  if (u.last_check && !off) meta.push(`Checked ${fmtAgo(u.last_check)}`);
  const lv = u.last_result_version, la = u.last_result_at ? ', ' + fmtWhen(u.last_result_at) : '';
  if (u.last_result === 'ok' && lv) meta.push(`last update: ${lv}${la}, worked`);
  else if (u.last_result === 'rolled_back' && lv && u.state !== 'rolled_back') meta.push(`last update: ${lv}${la} didn't start, went back`);
  else if (u.last_result === 'failed' && lv && u.state !== 'failed') meta.push(`last update to ${lv}${la} failed, nothing changed`);
  const checkErr = u.error && u.state !== 'failed' && !busy && !off;
  if (meta.length || checkErr) {
    now.push(h('p', { class: 'muted small' }, meta.join(' · ').replace(/^./, (c) => c.toUpperCase()),
      checkErr ? h('span', { class: 'warn-text', text: `${meta.length ? ' · ' : ''}Couldn't reach GitHub: ${u.error}` }) : null));
  }
  const acts = [];
  if (u.available && selfUpdates(u) && !busy && !off) acts.push(h('button', { class: 'btn primary', type: 'button', onclick: () => confirmInstall(u) }, 'Update now'));
  if (!off) acts.push(h('button', { class: 'btn', type: 'button', disabled: busy, onclick: () => updAction('/api/update/check', {}, 'Checking for updates…') }, 'Check now'));
  kids.push(h('div', { class: 'ub-now' }, now, acts.length ? h('div', { class: 'row' }, acts) : null));

  // server: devices still on an older version
  if (srv) {
    const out = Array.isArray(u.outdated) ? u.outdated : [];
    const total = (st.clients || []).length;
    const sec = [h('h4', { text: out.length ? `${out.length} device${out.length === 1 ? '' : 's'} out of date` : 'Your devices' })];
    if (out.length) {
      sec.push(h('ul', { class: 'ub-list' }, out.map((c) => h('li', {}, icon(c.kind === 'android' || c.kind === 'phone' ? 'device' : 'ethernet', 'ico sm'), h('b', { text: c.name }),
        h('span', { class: 'muted', text: ` ${KIND_NAME[c.kind] || c.kind || 'device'} · on ${c.version || 'an older version'}` })))));
      const how = [];
      if (out.some((c) => c.kind !== 'android' && c.kind !== 'phone')) how.push('Routers: Update now in their tailbond panel, or Automatic.');
      if (out.some((c) => c.kind === 'android' || c.kind === 'phone')) how.push('Phones: the app shows the update; Obtainium installs it automatically.');
      sec.push(h('p', { class: 'muted small', text: how.join(' ') }));
    } else sec.push(h('p', { class: 'muted small', text: total ? `All ${total} device${total === 1 ? ' is' : 's are'} on ${cur} or newer.` : 'No devices yet.' }));
    kids.push(h('div', { class: 'ub-sec' }, sec));
  }

  // the policy, in plain words
  const opts = selfUpdates(u) ? [['notify', 'Tell me (recommended)'], ['auto', 'Automatic'], ['off', 'Off']] : [['notify', 'Tell me (recommended)'], ['off', 'Off']];
  const help = policy === 'off' ? "Doesn't check for new versions."
    : policy === 'auto' ? 'Checks GitHub once a day and installs a new version by itself. tailbond restarts; devices drop for a few seconds; if the new version doesn\'t start, it goes back by itself.'
    : `Checks GitHub once a day and shows new versions here${selfUpdates(u) ? '; nothing installs until you click Update now' : ''}.`;
  kids.push(h('div', { class: 'ub-sec' }, h('h4', { text: 'When a new version is out' }),
    seg(opts, policy, (v) => updAction('/api/update/policy', { policy: v }, 'Saved'), 'Update policy'),
    h('p', { class: 'muted small', text: help }),
    u.method === 'dnf' ? h('p', { class: 'muted small', text: 'Automatic updates come from dnf-automatic.' }) : null));
  box.replaceChildren(...kids);
}

function confirmInstall(u) {
  dlg(h('h3', { text: `Update to ${u.available} now?` }),
    h('p', { text: "tailbond restarts; devices drop for a few seconds. If the new version doesn't start, it goes back by itself." }),
    h('div', { class: 'row end' },
      h('button', { class: 'btn', type: 'button', onclick: closeDlg }, 'Cancel'),
      h('button', { class: 'btn primary', type: 'button', onclick: async () => { closeDlg(); await updAction('/api/update/install', {}, `Installing ${u.available}…`); } }, 'Update now')));
}

// ---- generic dialog ----
function dlg(...kids) {
  $('#ub-dialog-body').replaceChildren(...kids);
  const d = $('#ub-dialog');
  if (!d.open) d.showModal();
}
function closeDlg() { $('#ub-dialog').close(); }

// ---- backup & restore ----
// a rough strength hint, not a gate (the gate is 10 characters)
function strength(p) {
  if (!p) return ['', ''];
  if (p.length < 10) return ['weak', `Too short: ${10 - p.length} more character${p.length === 9 ? '' : 's'}`];
  const kinds = [/[a-z]/, /[A-Z]/, /\d/, /[^A-Za-z0-9]/].filter((r) => r.test(p)).length;
  const words = p.trim().split(/[\s\-_.]+/).length;
  if (/^(.)\1+$/.test(p) || /^(?:0123456789|1234567890|password|qwerty)/i.test(p)) return ['weak', 'Easy to guess'];
  if (p.length >= 20 || words >= 4 || (p.length >= 14 && kinds >= 3)) return ['strong', 'Strong'];
  return ['ok', 'OK. Longer is better: four random words are strong and easy to type'];
}
function passFields() {
  const a = h('input', { type: 'password', id: 'bk-pass', autocomplete: 'new-password', placeholder: 'At least 10 characters' });
  const b = h('input', { type: 'password', id: 'bk-pass2', autocomplete: 'new-password', placeholder: 'Same again' });
  const meter = h('div', { class: 'meter' }, h('i'));
  const hint = h('span', { class: 'small muted' });
  const upd = () => { const [lvl, txt] = strength(a.value); meter.className = 'meter ' + lvl; hint.textContent = txt; };
  a.addEventListener('input', upd);
  const el = h('div', { class: 'ub-pass' },
    h('label', { class: 'field', for: 'bk-pass' }, h('span', { text: 'Passphrase' }), a), meter, hint,
    h('label', { class: 'field', for: 'bk-pass2' }, h('span', { text: 'Type it again' }), b));
  return { el, a, b, check() {
    if (a.value.length < 10) return 'The passphrase needs at least 10 characters.';
    if (a.value !== b.value) return "The two passphrases don't match.";
    return '';
  } };
}
function saveBlob(blob, name) {
  const a = h('a', { href: URL.createObjectURL(blob), download: name });
  document.body.append(a); a.click(); a.remove();
  setTimeout(() => URL.revokeObjectURL(a.href), 10000);
}
function openBackup() {
  const pf = passFields();
  const err = h('div', { class: 'error small' });
  const go = h('button', { class: 'btn primary', type: 'button' }, 'Download backup');
  go.addEventListener('click', async () => {
    err.textContent = pf.check();
    if (err.textContent) return;
    go.disabled = true; go.textContent = 'Locking…';
    try {
      const r = await fetch('/api/backup', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ passphrase: pf.a.value }) });
      if (r.status === 401) { showLogin(); throw new Error('sign in first'); }
      if (!r.ok) throw new Error((await r.json().catch(() => ({}))).error || r.statusText);
      const m = /filename="?([^";]+)"?/.exec(r.headers.get('Content-Disposition') || '');
      const host = (state.status && state.status.tailscale && state.status.tailscale.self) || 'tailbond';
      saveBlob(await r.blob(), m ? m[1] : `${host}-${new Date().toISOString().slice(0, 10)}.tbbackup`);
      closeDlg();
      toast('Backup downloaded. Keep the passphrase somewhere safe');
    } catch (e) { err.textContent = e.message; go.disabled = false; go.textContent = 'Download backup'; }
  });
  dlg(h('h3', { text: 'Download a backup' }),
    h('p', { class: 'small', text: `Everything this ${state.role === 'server' ? 'server' : 'machine'} needs to start over: settings, devices and their keys, ${state.role === 'server' ? 'the certificate devices check, ' : 'servers, '}rules. It's locked with a passphrase you choose now.` }),
    pf.el,
    h('div', { class: 'ub-note warn' }, h('b', { text: 'Write it down. ' }), "Without the passphrase the backup can't be opened, by anyone, tailbond included. A password manager or a note in a drawer both work."),
    err,
    h('div', { class: 'row end' }, h('button', { class: 'btn', type: 'button', onclick: closeDlg }, 'Cancel'), go));
}
function fileB64(file) {
  return new Promise((ok, no) => {
    const r = new FileReader();
    r.onload = () => ok(String(r.result).replace(/^data:[^,]*,/, ''));
    r.onerror = () => no(new Error("Couldn't read that file"));
    r.readAsDataURL(file);
  });
}
const ROLE_WORDS = { server: 'home server', client: 'travel router' };
function warnText(w) { return typeof w === 'string' ? w : (w && (w.text || w.message || w.code)) || ''; }
function openRestore() {
  const file = h('input', { type: 'file', id: 'rs-file', accept: '.tbbackup' });
  const pass = h('input', { type: 'password', id: 'rs-pass', autocomplete: 'off', placeholder: 'The passphrase of that backup' });
  const err = h('div', { class: 'error small' });
  const go = h('button', { class: 'btn primary', type: 'button' }, 'Check the backup');
  go.addEventListener('click', async () => {
    err.textContent = '';
    if (!file.files[0]) { err.textContent = 'Pick the backup file first (it ends in .tbbackup).'; return; }
    if (!pass.value) { err.textContent = 'Type the passphrase of that backup.'; return; }
    go.disabled = true; go.textContent = 'Opening…';
    try {
      const body = { file: await fileB64(file.files[0]), passphrase: pass.value };
      const p = await api('POST', '/api/restore/preview', body);
      showPreview(body, p, file.files[0].name);
    } catch (e) {
      err.textContent = /passphrase|damaged|gcm|auth/i.test(e.message) ? "Wrong passphrase, or the file is damaged. Check the passphrase (it's case-sensitive) and try again."
        : /role/i.test(e.message) ? `That backup is from a ${state.role === 'server' ? 'travel router' : 'home server'}; it can only be restored on one.` : e.message;
      go.disabled = false; go.textContent = 'Check the backup';
    }
  });
  dlg(h('h3', { text: 'Restore from a backup' }),
    h('p', { class: 'small', text: "Pick a .tbbackup file and type its passphrase. Nothing changes yet: you see what it would replace first." }),
    h('label', { class: 'field', for: 'rs-file' }, h('span', { text: 'Backup file' }), file),
    h('label', { class: 'field', for: 'rs-pass' }, h('span', { text: 'Passphrase' }), pass),
    err,
    h('div', { class: 'row end' }, h('button', { class: 'btn', type: 'button', onclick: closeDlg }, 'Cancel'), go));
}
function showPreview(body, p, fname) {
  const err = h('div', { class: 'error small' });
  const go = h('button', { class: 'btn primary danger-fill', type: 'button' }, 'Restore');
  go.addEventListener('click', async () => {
    go.disabled = true; go.textContent = 'Restoring…';
    try {
      const r = await api('POST', '/api/restore', { ...body, confirm: true });
      dlg(h('div', { class: 'ub-done' }, icon('backup'), h('h3', { text: 'Restored' })),
        h('p', { text: `The settings from ${p.host || 'the backup'} are in place. A copy of the previous settings was kept${r && r.rollback ? ` (${r.rollback})` : ''}, locked with the same passphrase, in case you want them back.` }),
        h('p', { class: 'muted small', text: 'tailbond is restarting with them; this page reconnects by itself in a few seconds. Devices reconnect by themselves too.' }),
        h('div', { class: 'row end' }, h('button', { class: 'btn primary', type: 'button', onclick: closeDlg }, 'Done')));
    } catch (e) { err.textContent = e.message; go.disabled = false; go.textContent = 'Restore'; }
  });
  const changes = Array.isArray(p.changes) ? p.changes : [];
  const warns = (Array.isArray(p.warnings) ? p.warnings : []).map(warnText).filter(Boolean);
  dlg(h('h3', { text: 'Restore this backup?' }),
    h('div', { class: 'ub-file' }, icon('backup', 'ico sm'), h('div', {},
      h('b', { text: fname }),
      h('div', { class: 'muted small', text: `${p.host || '?'} (${ROLE_WORDS[p.role] || p.role}) · made ${fmtWhen(p.created)} · tailbond ${p.agent_version || '?'}` }))),
    h('h4', { text: changes.length ? 'What it replaces' : 'Nothing differs from the current settings' }),
    changes.length ? h('table', { class: 'ub-changes' }, h('tbody', {}, changes.map((c) => h('tr', {},
      h('th', { text: c.what }), h('td', { class: 'muted', text: c.from || '—' }), h('td', { text: '→' }), h('td', { text: c.to || '—' }))))) : null,
    warns.length ? h('div', { class: 'ub-note warn' }, h('b', { text: 'Before you restore' }), h('ul', {}, warns.map((w) => h('li', { text: w })))) : null,
    h('p', { class: 'muted small', text: 'The current settings are saved first (same passphrase), so you can go back. tailbond restarts afterwards.' }),
    err,
    h('div', { class: 'row end' }, h('button', { class: 'btn', type: 'button', onclick: openRestore }, 'Back'), go));
}
// GET /api/backup/summary keys (backup.Summary), in reading order
const SUMMARY_LABEL = { role: 'Role', mode: 'Mode', clients: 'Devices', servers: 'Servers', public_addr: 'Public address', port: 'Tunnel port', subnet: 'Tunnel addresses',
  approvers: 'Allowed to approve', bypass_rules: 'Rules that skip tailbond', paths: 'Connection settings', password_set: 'Dashboard password', update_policy: 'Updates',
  files: 'Key and certificate files', uci: 'Router settings (UCI)', host: 'Machine', platform: 'System', agent_version: 'tailbond version' };
function summaryRows(s) {
  const order = Object.keys(SUMMARY_LABEL);
  const rank = (k) => (order.includes(k) ? order.indexOf(k) : 99);
  return Object.entries(s || {}).filter(([k]) => k !== 'ok').sort((a, b) => rank(a[0]) - rank(b[0])).map(([k, v]) => {
    let val;
    if (v == null || v === '') val = 'none';
    else if (Array.isArray(v)) val = v.length ? `${v.length}: ${v.map((x) => (typeof x === 'object' ? x.name || JSON.stringify(x) : x)).join(', ')}` : 'none';
    else if (v && typeof v === 'object') val = Object.entries(v).map(([a, b]) => `${a} ${b}`).join(', ');
    else if (k === 'password_set') val = v ? 'set (in the backup, not shown)' : 'not set';
    else if (typeof v === 'boolean') val = v ? 'included' : 'no';
    else if (k === 'role') val = ROLE_WORDS[v] || v;
    else val = String(v);
    return h('tr', {}, h('th', { text: SUMMARY_LABEL[k] || k.replace(/_/g, ' ') }), h('td', { text: val }));
  });
}
async function openSummary() {
  let s;
  try { s = await api('GET', '/api/backup/summary'); } catch (e) { return toast(e.message, true); }
  dlg(h('h3', { text: "What's in a backup" }),
    h('p', { class: 'small muted', text: 'What a backup made now would hold. Keys and passwords are in it too (locked), but never shown here.' }),
    h('table', { class: 'ub-changes ub-summary' }, h('tbody', {}, summaryRows(s))),
    h('div', { class: 'row end' }, h('button', { class: 'btn primary', type: 'button', onclick: closeDlg }, 'Close')));
}

function bset() { return (state.settings && state.settings.backup) || (state.status && state.status.backup) || {}; }
async function saveSched(body, msg) {
  try { await api('POST', '/api/backup/settings', body); toast(msg); ub.schedEdit = false; await loadSettings(); ub.bsig = ''; renderBackups(); } catch (e) { toast(e.message, true); }
}
function renderBackups() {
  const box = $('#bk-card');
  const srv = state.role === 'server';
  const b = bset();
  const sig = JSON.stringify([b, ub.backups, srv, ub.schedEdit]);
  if (sig === ub.bsig) return;
  ub.bsig = sig;
  const on = !!(b.passphrase_set && (b.every_hours || 24));
  const kids = [h('div', { class: 'ub-head' },
    h('div', { class: 'kind-ico' }, icon('backup')),
    h('div', { class: 'grow' }, h('h3', { text: 'Backup & restore' }),
      h('div', { class: 'muted small', text: srv ? 'Settings, devices and keys, locked with a passphrase. Move to a new machine and your devices keep working.' : 'Settings, servers and rules, locked with a passphrase.' })),
    h('span', { class: 'chip ' + (on ? 's-active' : 's-off'), text: on ? 'Automatic backups on' : 'Automatic backups off' }))];
  kids.push(h('div', { class: 'row' },
    h('button', { class: 'btn primary', type: 'button', onclick: openBackup }, 'Download a backup'),
    h('button', { class: 'btn', type: 'button', onclick: openRestore }, 'Restore from a backup'),
    h('button', { class: 'linkish small', type: 'button', onclick: openSummary }, "See what's inside")));

  // scheduled backups
  const sched = [h('h4', { text: 'Automatic backups' })];
  const every = b.every_hours || 24, keep = b.keep || 7;
  const everyWords = every % 24 === 0 ? (every === 24 ? 'every day' : `every ${every / 24} days`) : `every ${every} hours`;
  if (!ub.schedEdit) {
    sched.push(h('p', { class: 'small' }, on
      ? [`On: ${everyWords}, keeps the last ${keep}`, srv ? ' on this server' : '', b.last ? ` · last one ${fmtAgo(b.last)}` : '', '.',
        !srv && b.push ? ` A copy also goes to ${homeName()}${b.last_push ? ` (sent ${fmtAgo(b.last_push)})` : ''}, so you can get it back if this router is lost.` : '']
      : ['Off. Set a passphrase to have tailbond save a backup by itself.']));
    if (b.last_error) sched.push(h('p', { class: 'small error', text: `The last automatic backup didn't work: ${b.last_error}` }));
    sched.push(h('div', { class: 'row' }, h('button', { class: 'btn', type: 'button', onclick: () => { ub.schedEdit = true; ub.bsig = ''; renderBackups(); } }, on ? 'Change' : 'Turn on'),
      on ? h('button', { class: 'btn', type: 'button', onclick: () => saveSched({ passphrase: '' }, 'Automatic backups off') }, 'Turn off') : null));
  } else {
    const pf = passFields();
    const ev = h('input', { type: 'number', min: 1, max: 720, value: every, id: 'bk-every', class: 'num' });
    const kp = h('input', { type: 'number', min: 1, max: 100, value: keep, id: 'bk-keep', class: 'num' });
    const push = h('input', { type: 'checkbox', id: 'bk-push' });
    push.checked = b.push !== false;
    const err = h('div', { class: 'error small' });
    sched.push(
      b.passphrase_set ? h('p', { class: 'muted small', text: 'A passphrase is set. Leave the fields empty to keep it, or type a new one.' }) : h('p', { class: 'muted small', text: 'Choose a passphrase for the automatic backups (you need it to restore one).' }),
      pf.el,
      h('div', { class: 'row' }, 'Every ', ev, ' hours, keep the last ', kp),
      srv ? null : h('label', { class: 'check', for: 'bk-push' }, push, `Also send a copy to ${homeName()} (it keeps the last 3)`),
      err,
      h('div', { class: 'row' },
        h('button', { class: 'btn primary', type: 'button', onclick: () => {
          const body = { every_hours: Number(ev.value) || 24, keep: Number(kp.value) || 7 };
          if (!srv) body.push = push.checked;
          if (pf.a.value || !b.passphrase_set) { err.textContent = pf.check(); if (err.textContent) return; body.passphrase = pf.a.value; }
          saveSched(body, 'Automatic backups saved');
        } }, 'Save'),
        h('button', { class: 'btn', type: 'button', onclick: () => { ub.schedEdit = false; ub.bsig = ''; renderBackups(); } }, 'Cancel')));
  }
  kids.push(h('div', { class: 'ub-sec' }, sched));

  // server: every saved backup, this server's and the devices'
  if (srv && Array.isArray(ub.backups)) {
    const mine = ub.backups.filter((x) => !x.client);
    const theirs = ub.backups.filter((x) => x.client);
    const row = (x) => h('tr', {},
      h('td', {}, h('b', { text: x.client || 'this server' })),
      h('td', { text: fmtWhen(x.created) }), h('td', { class: 'muted', text: fmtBytes(x.size) }),
      h('td', {}, h('a', { href: '/api/backups/' + encodeURIComponent(x.name) + (x.client ? '?client=' + encodeURIComponent(x.client) : ''), download: x.name, text: 'Download' })));
    const sec = [h('h4', { text: 'Saved backups' })];
    if (!ub.backups.length) sec.push(h('p', { class: 'muted small', text: 'None yet. Automatic backups land here, and travel routers that send theirs.' }));
    else {
      sec.push(h('table', { class: 'ub-backups' },
        h('thead', {}, h('tr', {}, h('th', { text: 'From' }), h('th', { text: 'Made' }), h('th', { text: 'Size' }), h('th'))),
        h('tbody', {}, mine.map(row), theirs.map(row))));
      sec.push(h('p', { class: 'muted small', text: `Each is locked with the passphrase it was made with${theirs.length ? "; the devices' backups with their own" : ''}. Keep a copy somewhere other than this machine too.` }));
    }
    kids.push(h('div', { class: 'ub-sec' }, sec));
  }
  box.replaceChildren(...kids);
}

// ---------- data feed ----------
let es = null;
let pollTimer = null;
async function pollOnce() {
  try {
    const r = await fetch('/api/status');
    if (r.status === 401) { showLogin(); return; }
    render(await r.json());
  } catch { /* offline */ }
}
function connect() {
  if (es) es.close();
  clearInterval(pollTimer);
  es = new EventSource('/api/events');
  es.onmessage = (ev) => { state.lastTs = Date.now(); render(JSON.parse(ev.data)); };
  es.onerror = () => {
    // EventSource retries by itself; meanwhile poll so the UI stays live and
    // a 401 surfaces the login dialog.
    if (!pollTimer) pollTimer = setInterval(() => { if (Date.now() - state.lastTs > 3000) pollOnce(); }, 2000);
    pollOnce();
  };
}

setupModes();
setupPower();
setupTabs();
setupClients();
setupPassword();
setupAddPhone();
setupPairing();
setupLogin();
connect();
setInterval(pollLogs, 2000);
setInterval(() => { if (state.tab === 'tailscale') renderTailscale(); if (state.role === 'server') renderHealth(); }, 15000);
