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
  $('#rx-val').textContent = rxv; $('#rx-unit').textContent = rxu + ' down';
  $('#tx-val').textContent = txv; $('#tx-unit').textContent = txu + ' up';
  $('#uptime').textContent = st.state === 'connected' ? fmtDur(st.uptime_sec) : '—';
  $('#tunnel-ip').textContent = st.tunnel_ip || '—';
  $('#total-bytes').textContent = `${fmtBytes(st.totals.rx_bytes)} ↓ · ${fmtBytes(st.totals.tx_bytes)} ↑`;
  const chip = $('#state-chip');
  chip.textContent = st.state;
  chip.className = 'state-chip s-' + st.state;
  drawSpark($('#spark-total'), push('total', st.totals.rx_bps, st.totals.tx_bps), 600, 120, { grid: true, sw: 2.2 });

  const sub = st.role === 'server'
    ? `Home server · ${st.server || 'no public address'}`
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
  $('#foot').textContent = `tailbondd ${st.version || ''} · ${st.control_api ? 'live path stats' : 'basic stats (mqvpn control API unavailable)'}`;
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
  if (role === 'client') loadSettings();
  if (role === 'server') loadSettings();
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
// Devices on the tailnet that asked to connect (POST /api/pair-requests).
function renderPairRequests(st) {
  const box = $('#pair-requests');
  const reqs = st.role === 'server' && Array.isArray(st.pair_requests) ? st.pair_requests : [];
  const sig = reqs.map((r) => r.id).join(',');
  box.classList.toggle('hidden', !reqs.length);
  if (box.dataset.sig === sig) return;
  box.dataset.sig = sig;
  box.replaceChildren(h('h2', { text: 'Waiting for your approval' }), ...reqs.map((r) => {
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
        r.repair ? h('div', { class: 'small', text: 'It was connected before (reset or reinstalled). Approving gives it its old access back.' }) : null),
      h('div', { class: 'pair-req-actions' },
        h('button', { class: 'btn', type: 'button', text: 'Decline', onclick: (e) => decide('decline', e.currentTarget) }),
        h('button', { class: 'btn primary', type: 'button', text: 'Approve', onclick: (e) => decide('approve', e.currentTarget) })));
  }));
}

function inviteLink(st) {
  const dns = st.tailscale && st.tailscale.dns_name;
  const host = dns ? `${dns}:${location.port || 80}` : location.host;
  return `http://${host}/#add-phone`;
}

function renderClients(st) {
  $('#invite-link').value = inviteLink(st);
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
    $('.client-name small', el).textContent = c.connected ? `${c.endpoint} · ${c.scheduler || ''}` : 'offline';
    $('.client-rates .rx', el).textContent = c.connected ? '↓ ' + rateStr(c.tx_bps) : '';
    $('.client-rates .tx', el).textContent = c.connected ? '↑ ' + rateStr(c.rx_bps) : '';
    const chip = $('.chip', el);
    chip.textContent = c.connected ? `${c.paths.filter((p) => p.state === 'active').length} paths` : 'offline';
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
  const s = state.settings;
  if (s.role === 'server') {
    $('#apk-yes').classList.toggle('hidden', !s.android_apk);
    $('#apk-no').classList.toggle('hidden', !!s.android_apk);
    const w = $('#addr-warning');
    if (s.public_addr_guessed) {
      w.innerHTML = '';
      w.append(
        h('b', { text: 'Public address not set. ' }),
        `Pairing links will use ${s.effective_public_addr || 'no address'}, which is probably your LAN address. Enter your public IP or DDNS name (and forward UDP ${s.port} on your router).`,
        h('div', { class: 'row' },
          h('input', { id: 'pub-addr', placeholder: `home.example.net:${s.port}` }),
          h('button', { class: 'btn', type: 'button', onclick: savePublicAddr, text: 'Save' })));
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
      if (state.tab === 'clients' && state.status) { loadSettings(); renderClients(state.status); }
      if (state.tab === 'setup') loadSettings();
    });
  }
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
  $('#invite-copy').addEventListener('click', async () => {
    try { await navigator.clipboard.writeText($('#invite-link').value); } catch { $('#invite-link').select(); document.execCommand('copy'); }
    toast('Link copied');
  });
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
setupAddPhone();
setupPairing();
setupLogin();
connect();
setInterval(pollLogs, 2000);
setInterval(() => { if (state.tab === 'tailscale') renderTailscale(); }, 15000);
