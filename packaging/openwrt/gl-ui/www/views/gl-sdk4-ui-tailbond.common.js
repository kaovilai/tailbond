// SPDX-License-Identifier: Apache-2.0
// GL.iNet 4.x admin panel ("oui") view: VPN -> tailbond.
//
// GL's admin app fetches /views/gl-sdk4-ui-<view>.common.js and eval()s it
// (menu entry: /usr/share/oui/menu.d/tailbond.json); the value of this file
// is the Vue 2 component (runtime-only Vue, so render functions, no
// templates). It talks to tailbondd through GL's own session:
//   window.$rpcRequest("call", ["sid", "tailbond", <method>, {...}])
// -> /usr/lib/oui-httpd/rpc/tailbond -> http://127.0.0.1:8077/api/...
// If that object is unavailable it falls back to the same-origin proxy
// /tailbond-api/ (etc/nginx/gl-conf.d/tailbond.conf), which checks the
// same GL session. No second password, no :8077 page needed.
// Self-contained styles (Element-UI look) so it doesn't depend on which GL
// components a firmware registers. Shipped gzipped, as GL's views are.
(function () {
  "use strict";

  var MODES = [
    { wire: "speed", label: "Fastest", blurb: "Adds all your connections together for the most speed." },
    { wire: "streaming", label: "Smooth calls and video", blurb: "Calls and video keep going when one connection has trouble." },
    { wire: "redundant", label: "Most reliable", blurb: "Sends everything over every connection. Uses the most data." },
    { wire: "failover", label: "Save mobile data", blurb: "Uses Ethernet and Wi-Fi; your phone and cellular only when those stop working." }
  ];

  // GL's own names for its WAN types (Internet page).
  var KIND_LABEL = { ethernet: "Ethernet", wifi: "Repeater", usb: "Tethering", cellular: "Cellular", unknown: "Connection" };
  // Short nouns for the one-line status.
  var KIND_NOUN = { ethernet: "Ethernet", wifi: "Repeater (Wi-Fi)", usb: "Tethering (phone)", cellular: "Cellular", unknown: "a connection" };
  var KIND_ORDER = { ethernet: 0, wifi: 1, usb: 2, cellular: 3, unknown: 4 };
  var NOT_CONNECTED_FIX = {
    ethernet: "Plug a cable from the hotel or modem into the WAN port. It's used automatically once connected.",
    wifi: "Join a Wi-Fi network in INTERNET → Repeater. It's used automatically once connected.",
    usb: "Plug in your phone with a USB cable and turn on USB tethering on the phone. It's used automatically once connected.",
    cellular: "Insert a SIM card or plug in the modem, then check INTERNET → Cellular. It's used automatically once connected.",
    unknown: "Set it up on the INTERNET page. It's used automatically once connected."
  };

  var CSS = [
    ".tb{font-size:14px;color:#303133;max-width:1080px;margin:0 auto;}",
    ".tb *{box-sizing:border-box;}",
    ".tb-card{background:#fff;border-radius:8px;box-shadow:0 1px 4px rgba(0,21,41,.08);margin-bottom:16px;}",
    ".tb-card-h{display:flex;align-items:center;justify-content:space-between;padding:14px 20px;border-bottom:1px solid #ebeef5;font-size:15px;font-weight:600;gap:12px;}",
    ".tb-card-b{padding:16px 20px;}",
    ".tb-muted{color:#909399;font-size:13px;}",
    ".tb-small{font-size:12px;}",
    ".tb-row{display:flex;align-items:center;gap:12px;}",
    ".tb-grow{flex:1;min-width:0;}",
    ".tb-hero{display:flex;align-items:center;gap:16px;padding:20px;}",
    ".tb-hero-icon{width:52px;height:52px;border-radius:12px;display:flex;align-items:center;justify-content:center;flex:none;background:#ecf5ff;color:#409eff;}",
    ".tb-hero-icon.on{background:#e8f8ef;color:#13a35a;}",
    ".tb-hero-icon.warn{background:#fdf6ec;color:#e6a23c;}",
    ".tb-hero-icon.err{background:#fef0f0;color:#f56c6c;}",
    ".tb-hero-t{font-size:18px;font-weight:600;}",
    ".tb-hero-s{font-size:14px;color:#606266;margin-top:4px;line-height:1.45;}",
    ".tb-rates{display:flex;gap:24px;padding:0 20px 16px 88px;color:#606266;font-size:13px;}",
    ".tb-rates b{font-size:16px;color:#303133;font-weight:600;}",
    ".tb-switch{position:relative;width:44px;height:22px;border-radius:11px;background:#dcdfe6;cursor:pointer;flex:none;transition:background .2s;border:0;padding:0;}",
    ".tb-switch:after{content:'';position:absolute;left:2px;top:2px;width:18px;height:18px;border-radius:50%;background:#fff;transition:left .2s;}",
    ".tb-switch.on{background:#409eff;}",
    ".tb-switch.on:after{left:24px;}",
    ".tb-switch[disabled]{opacity:.5;cursor:not-allowed;}",
    ".tb-switch-big{width:56px;height:28px;border-radius:14px;}",
    ".tb-switch-big:after{width:24px;height:24px;}",
    ".tb-switch-big.on:after{left:30px;}",
    ".tb-btn{display:inline-flex;align-items:center;justify-content:center;gap:6px;height:34px;padding:0 16px;border-radius:4px;border:1px solid #dcdfe6;background:#fff;color:#606266;font-size:14px;cursor:pointer;white-space:nowrap;text-decoration:none;}",
    ".tb-btn:hover{color:#409eff;border-color:#c6e2ff;background:#ecf5ff;}",
    ".tb-btn.primary{background:#409eff;border-color:#409eff;color:#fff;}",
    ".tb-btn.primary:hover{background:#66b1ff;border-color:#66b1ff;}",
    ".tb-btn.danger{color:#f56c6c;border-color:#fbc4c4;background:#fef0f0;}",
    ".tb-btn.text{border:0;background:none;color:#409eff;padding:0 4px;height:auto;}",
    ".tb-btn.small{height:28px;padding:0 12px;font-size:13px;}",
    ".tb-btn[disabled]{opacity:.55;cursor:not-allowed;}",
    ".tb-input{height:34px;border:1px solid #dcdfe6;border-radius:4px;padding:0 12px;font-size:14px;color:#303133;width:100%;outline:none;}",
    ".tb-input:focus{border-color:#409eff;}",
    "textarea.tb-input{height:72px;padding:8px 12px;resize:vertical;font-family:inherit;}",
    ".tb-chip{display:inline-flex;align-items:center;gap:6px;height:22px;padding:0 8px;border-radius:11px;font-size:12px;white-space:nowrap;background:#f4f4f5;color:#909399;}",
    ".tb-chip:before{content:'';width:6px;height:6px;border-radius:50%;background:currentColor;}",
    ".tb-chip.ok{background:#e8f8ef;color:#13a35a;}",
    ".tb-chip.warn{background:#fdf6ec;color:#c88a1a;}",
    ".tb-chip.err{background:#fef0f0;color:#f56c6c;}",
    ".tb-chip.blue{background:#ecf5ff;color:#409eff;}",
    ".tb-grid{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:16px;}",
    ".tb-wan{border:1px solid #ebeef5;border-radius:8px;padding:16px;display:flex;flex-direction:column;gap:12px;}",
    ".tb-wan.dim{background:#fafafa;}",
    ".tb-wan-icon{width:40px;height:40px;border-radius:8px;background:#ecf5ff;color:#409eff;display:flex;align-items:center;justify-content:center;flex:none;}",
    ".tb-wan.dim .tb-wan-icon{background:#f4f4f5;color:#909399;}",
    ".tb-wan-t{font-weight:600;font-size:15px;}",
    ".tb-wan-ctl{display:flex;align-items:center;justify-content:space-between;border-top:1px dashed #ebeef5;padding-top:10px;gap:8px;}",
    ".tb-check{display:inline-flex;align-items:center;gap:8px;cursor:pointer;color:#606266;user-select:none;}",
    ".tb-check i{width:16px;height:16px;border:1px solid #dcdfe6;border-radius:3px;display:inline-flex;align-items:center;justify-content:center;background:#fff;}",
    ".tb-check.on i{background:#409eff;border-color:#409eff;}",
    ".tb-check.on i:after{content:'';width:4px;height:8px;border:solid #fff;border-width:0 2px 2px 0;transform:rotate(45deg) translate(-1px,-1px);}",
    ".tb-bar{height:6px;border-radius:3px;background:#ebeef5;overflow:hidden;margin-top:6px;}",
    ".tb-bar i{display:block;height:100%;background:#409eff;}",
    ".tb-bar i.full{background:#f56c6c;}",
    ".tb-modes{display:grid;grid-template-columns:repeat(4,minmax(0,1fr));gap:12px;}",
    ".tb-modes.three{grid-template-columns:repeat(3,minmax(0,1fr));}",
    ".tb-mode{border:1px solid #dcdfe6;border-radius:8px;padding:12px 14px;cursor:pointer;background:#fff;text-align:left;font:inherit;color:inherit;}",
    ".tb-mode:hover{border-color:#409eff;}",
    ".tb-mode.on{border-color:#409eff;background:#ecf5ff;box-shadow:inset 0 0 0 1px #409eff;}",
    ".tb-mode-t{font-weight:600;display:flex;align-items:center;gap:8px;}",
    ".tb-mode-t:before{content:'';width:14px;height:14px;border-radius:50%;border:1px solid #dcdfe6;background:#fff;flex:none;}",
    ".tb-mode.on .tb-mode-t:before{border:4px solid #409eff;}",
    ".tb-mode-b{font-size:12px;color:#909399;margin-top:6px;line-height:1.45;}",
    ".tb-list{border-top:1px solid #ebeef5;}",
    ".tb-li{display:flex;align-items:center;gap:12px;padding:12px 0;border-bottom:1px solid #ebeef5;}",
    ".tb-li:last-child{border-bottom:0;}",
    ".tb-dot{width:8px;height:8px;border-radius:50%;background:#c0c4cc;flex:none;}",
    ".tb-dot.ok{background:#13a35a;}",
    ".tb-mono{font-family:Menlo,Consolas,monospace;font-size:12px;color:#606266;}",
    ".tb-alert{display:flex;gap:12px;padding:14px 16px;border-radius:8px;background:#fdf6ec;color:#8a5d10;margin-bottom:16px;line-height:1.5;}",
    ".tb-alert.err{background:#fef0f0;color:#c45656;}",
    ".tb-alert.info{background:#ecf5ff;color:#33689f;}",
    ".tb-alert.ok{background:#e8f8ef;color:#137a46;}",
    ".tb-alert b{color:inherit;}",
    ".tb-alert-t{font-weight:600;margin-bottom:2px;}",
    ".tb-actions{display:flex;gap:8px;flex-wrap:wrap;margin-top:10px;}",
    ".tb-req{border:1px solid #f5dab1;background:#fffaf2;border-radius:8px;padding:16px;display:flex;gap:16px;align-items:center;margin-bottom:12px;}",
    ".tb-found{border:1px solid #b3d8ff;background:#f5faff;border-radius:8px;padding:16px;display:flex;gap:16px;align-items:center;margin-bottom:12px;}",
    ".tb-spin{width:16px;height:16px;border:2px solid #c6e2ff;border-top-color:#409eff;border-radius:50%;animation:tbspin 1s linear infinite;flex:none;}",
    "@keyframes tbspin{to{transform:rotate(360deg)}}",
    ".tb-steps{margin:8px 0 0 0;padding-left:20px;color:#606266;line-height:1.7;}",
    ".tb-kv{display:grid;grid-template-columns:160px 1fr;gap:8px 16px;align-items:center;}",
    ".tb-kv>div:nth-child(odd){color:#909399;}",
    ".tb-qr{width:200px;height:200px;}",
    ".tb-qr svg{width:100%;height:100%;}",
    ".tb-modal{position:fixed;inset:0;background:rgba(0,0,0,.45);display:flex;align-items:center;justify-content:center;z-index:3000;padding:16px;}",
    ".tb-dialog{background:#fff;border-radius:8px;max-width:440px;width:100%;padding:20px;box-shadow:0 8px 30px rgba(0,0,0,.2);}",
    ".tb-dialog-t{font-size:16px;font-weight:600;margin-bottom:10px;}",
    ".tb-toast{position:fixed;left:50%;top:72px;transform:translateX(-50%);background:#f0f9eb;color:#529b2e;border:1px solid #e1f3d8;padding:10px 18px;border-radius:4px;z-index:3001;box-shadow:0 2px 12px rgba(0,0,0,.1);}",
    ".tb-link{color:#409eff;cursor:pointer;text-decoration:none;}",
    ".tb-seg{display:inline-flex;border:1px solid #dcdfe6;border-radius:4px;overflow:hidden;}",
    ".tb-seg button{border:0;border-left:1px solid #dcdfe6;background:#fff;color:#606266;font:inherit;font-size:13px;height:30px;padding:0 12px;cursor:pointer;white-space:nowrap;}",
    ".tb-seg button:first-child{border-left:0;}",
    ".tb-seg button.on{background:#409eff;color:#fff;}",
    ".tb-seg button.on.off{background:#909399;}",
    ".tb-seg button[disabled]{cursor:not-allowed;opacity:.6;}",
    ".tb-nowrap{white-space:nowrap;}",
    ".tb-steps-n{margin:6px 0 0;padding-left:22px;line-height:1.7;color:#606266;}",
    ".tb-foot{text-align:center;color:#c0c4cc;font-size:12px;margin:8px 0 24px;}",
    "@media (max-width:760px){",
    " .tb-grid{grid-template-columns:1fr;}",
    " .tb-modes,.tb-modes.three{grid-template-columns:1fr;}",
    " .tb-card-b{padding:14px;}",
    " .tb-card-h{padding:12px 14px;}",
    " .tb-hero{padding:16px 14px;gap:12px;}",
    " .tb-hero-icon{width:44px;height:44px;}",
    " .tb-hero-t{font-size:16px;}",
    " .tb-rates{padding:0 14px 14px 70px;gap:16px;}",
    " .tb-req,.tb-found{flex-direction:column;align-items:stretch;}",
    " .tb-req .tb-actions,.tb-found .tb-actions{margin-top:0;}",
    " .tb-req .tb-btn,.tb-found .tb-btn{flex:1;}",
    " .tb-kv{grid-template-columns:1fr;gap:2px;}",
    " .tb-kv>div:nth-child(even){margin-bottom:8px;}",
    " .tb-hide-sm{display:none;}",
    "}"
  ].join("\n");

  // Icons (24px, stroke), Element/GL-like.
  var ICONS = {
    bond: "M4 7h6M4 12h16M4 17h6M14 7h6M14 17h6M10 7l4 5-4 5",
    ethernet: "M4 6h16v9H4zM8 15v3h8v-3M8 9v2M11 9v2M14 9v2M17 9v2",
    wifi: "M2 9a15 15 0 0 1 20 0M5 12.5a10 10 0 0 1 14 0M8.5 16a5 5 0 0 1 7 0M12 19.5h.01",
    usb: "M8 3h8v6H8zM10 5v2M14 5v2M9 9v6a3 3 0 0 0 6 0V9M12 18v3",
    cellular: "M4 20v-3M9 20v-7M14 20v-11M19 20V4",
    unknown: "M12 3a9 9 0 1 0 0 18 9 9 0 0 0 0-18zM12 8v5M12 16h.01",
    server: "M4 4h16v6H4zM4 14h16v6H4zM8 7h.01M8 17h.01",
    phone: "M8 2h8a1 1 0 0 1 1 1v18a1 1 0 0 1-1 1H8a1 1 0 0 1-1-1V3a1 1 0 0 1 1-1zM11 18h2",
    router: "M3 13h18v6H3zM7 16h.01M11 16h.01M8 13l-2-6M16 13l2-6",
    check: "M5 12l5 5L20 7",
    warn: "M12 3l10 18H2zM12 10v5M12 18h.01",
    off: "M12 3v9M6.3 6.3a8 8 0 1 0 11.4 0"
  };

  function icon(h, name, size) {
    return h("svg", { attrs: { width: size || 22, height: size || 22, viewBox: "0 0 24 24", fill: "none", stroke: "currentColor", "stroke-width": 1.8, "stroke-linecap": "round", "stroke-linejoin": "round" } },
      [h("path", { attrs: { d: ICONS[name] || ICONS.unknown } })]);
  }

  function fmtBytes(b) {
    b = b || 0;
    if (b < 1e6) return Math.round(b / 1e3) + " KB";
    if (b < 1e9) return (b / 1e6).toFixed(b < 1e7 ? 1 : 0) + " MB";
    return (b / 1e9).toFixed(1) + " GB";
  }
  function fmtRate(bps) {
    bps = bps || 0;
    if (bps < 1e6) return Math.round(bps / 1e3) + " Kbps";
    return (bps / 1e6).toFixed(bps < 1e7 ? 1 : 0) + " Mbps";
  }
  function fmtCap(mb) { return mb >= 1000 ? (mb / 1000).toFixed(mb % 1000 ? 1 : 0) + " GB" : mb + " MB"; }
  function fmtAgo(t) {
    var s = Math.max(0, Math.round((Date.now() - new Date(t).getTime()) / 1000));
    if (s < 60) return "just now";
    if (s < 3600) return Math.round(s / 60) + " min ago";
    if (s < 86400) return Math.round(s / 3600) + " h ago";
    return Math.round(s / 86400) + " days ago";
  }
  function fmtUptime(s) {
    if (!s) return "";
    if (s < 60) return "just now";
    if (s < 3600) return Math.floor(s / 60) + " min";
    var hrs = Math.floor(s / 3600);
    if (hrs < 48) return hrs + " h " + Math.floor((s % 3600) / 60) + " min";
    return Math.floor(hrs / 24) + " days";
  }
  // Lists may arrive as {} when empty (Lua cjson on the router re-encodes).
  function A(x) { return Array.isArray(x) ? x : []; }
  function shortName(n) { return (n || "").split(".")[0].split(":")[0]; }
  function hostOf(a) {
    a = a || "";
    var m = a.match(/^\[([^\]]+)\](?::\d+)?$/);
    if (m) return m[1];
    return a.split(":").length === 2 ? a.split(":")[0] : a;
  }
  function portOf(a) {
    var m = (a || "").match(/:(\d+)$/);
    return m ? m[1] : "";
  }
  // "nas.tail1234.ts.net" may wrap after a dot, never inside a word
  function dotBreaks(h, name) {
    var out = [];
    (name || "").split(".").forEach(function (part, i, all) {
      out.push(part + (i < all.length - 1 ? "." : ""));
      if (i < all.length - 1) out.push(h("wbr"));
    });
    return out;
  }
  function lastSeen(p) {
    if (p.online) return "Online";
    return p.last_seen ? "Last seen " + fmtAgo(p.last_seen) : "Offline";
  }

  // ---- backend ----
  var useFetch = false;

  function viaFetch(method, path, body) {
    var opt = { method: method, credentials: "same-origin", headers: {} };
    if (method !== "GET" && method !== "DELETE") {
      opt.headers["Content-Type"] = "application/json";
      opt.body = JSON.stringify(body || {});
    }
    return fetch("/tailbond-api/" + path.replace(/^\/api\//, ""), opt).then(function (r) {
      return r.text().then(function (t) {
        var d = null;
        try { d = t ? JSON.parse(t) : null; } catch (e) { d = { raw: t }; }
        if (r.status === 502 || r.status === 504) return { ok: false, code: 0, error: "agent_down" };
        if (r.status === 401 && !d) return { ok: false, code: 401, error: "session" };
        return { ok: r.ok, code: r.status, data: d };
      });
    });
  }

  function rpc(method, params) {
    var call = window.$rpcRequest || (window.Vue && window.Vue.prototype && window.Vue.prototype.$rpcRequest);
    if (!call) return Promise.reject({ type: "noRpc" });
    return call("call", ["sid", "tailbond", method, params || {}], { timeout: 20000 });
  }

  // api() resolves to the agent's JSON; rejects with {code, error, data}.
  function api(method, path, body) {
    var p = useFetch ? viaFetch(method, path, body) : rpc("api", { method: method, path: path, body: body }).catch(function (e) {
      // the RPC object isn't there (firmware restricts it): same-origin proxy
      if (e && (e.type === "noRpc" || e.type === "accessDenied" || e.type === "invalidParams" || e.code === -32601 || e.code === -32000)) {
        useFetch = true;
        return viaFetch(method, path, body);
      }
      throw e;
    });
    return p.then(function (res) {
      if (!res || !res.ok) {
        throw { code: res ? res.code : 0, error: (res && (res.error || (res.data && res.data.error))) || "failed", data: res && res.data };
      }
      return res.data;
    });
  }

  // ---- component ----
  return {
    name: "tailbond",
    data: function () {
      return {
        loaded: false, down: false, info: null,
        st: null, paths: [], settings: null, ts: null,
        peers: null, peersErr: null, peerFilter: "",
        disc: null, discovering: false, typed: "", showOther: false, pasteUri: "", pasteErr: "",
        busy: {}, toast: "", confirm: null, capEdit: null, capValue: "",
        addName: "", added: null, addErr: "", addrEdit: false, addrValue: "",
        wantOn: null, actErr: "", showTyped: false, pwEdit: false, pwValue: "", health: null, showHealthOk: false
      };
    },
    computed: {
      role: function () { return (this.st && this.st.role) || (this.info && this.info.role) || "client"; },
      paired: function () { return !!(this.st && this.st.paired); },
      join: function () { return (this.st && this.st.join) || null; },
      serverShort: function () {
        var s = this.settings && this.settings.paired_with;
        var n = (this.st && this.st.server) || s || "";
        return shortName(n);
      },
      wans: function () { return this.buildWans(); },
      pending: function () { return A(this.st && this.st.pair_requests); }
    },
    created: function () {
      var self = this;
      if (!document.getElementById("tb-style")) {
        var s = document.createElement("style");
        s.id = "tb-style";
        s.textContent = CSS;
        document.head.appendChild(s);
      }
      this.refreshAll();
      this.timer = setInterval(function () { self.tick(); }, 2000);
      this.slow = setInterval(function () { self.refreshSlow(); }, 15000);
    },
    beforeDestroy: function () { clearInterval(this.timer); clearInterval(this.slow); },
    methods: {
      // ---- data ----
      refreshAll: function () {
        var self = this;
        rpc("info").then(function (i) { self.info = i; }).catch(function () {});
        return this.tick().then(function () { return self.refreshSlow(); });
      },
      tick: function () {
        var self = this;
        return api("GET", "/api/status").then(function (st) {
          self.st = st; self.down = false; self.loaded = true;
          if (self.wantOn !== null && self.isOn() === self.wantOn) self.wantOn = null;
          if (st.role === "client" && !st.paired && !self.disc && !self.discovering) self.discover(false);
        }).catch(function (e) {
          self.loaded = true;
          if (e && e.error === "agent_down") self.down = true;
        });
      },
      refreshSlow: function () {
        var self = this;
        if (this.down || !this.st) return Promise.resolve();
        var jobs = [api("GET", "/api/settings").then(function (s) { self.settings = s; }).catch(function () {})];
        if (this.st.role === "client") {
          jobs.push(api("GET", "/api/paths").then(function (p) { self.paths = A(p); }).catch(function () {}));
          if (this.st.paired && this.st.state === "connected") jobs.push(this.loadPeers());
        } else {
          jobs.push(api("GET", "/api/tailscale").then(function (t) { self.ts = t; }).catch(function () {}));
          jobs.push(api("GET", "/api/health").then(function (hh) { self.health = hh; }).catch(function () {}));
          jobs.push(this.loadPeers());
        }
        return Promise.all(jobs);
      },
      loadPeers: function () {
        var self = this;
        return api("GET", "/api/peers").then(function (p) { self.peers = p; self.peersErr = null; })
          .catch(function (e) { self.peersErr = e; });
      },
      discover: function (refresh) {
        var self = this;
        this.discovering = true;
        return api("GET", "/api/discover" + (refresh ? "?refresh=1" : "")).then(function (d) { self.disc = d; })
          .catch(function () { self.disc = { servers: [], tailscale_running: false, failed: true }; })
          .then(function () { self.discovering = false; });
      },
      run: function (key, p, okMsg) {
        var self = this;
        this.$set(this.busy, key, true);
        this.actErr = "";
        return p.then(function (r) {
          if (okMsg) self.say(okMsg);
          return self.tick().then(function () { return self.refreshSlow(); }).then(function () { return r; });
        }).catch(function (e) {
          self.actErr = self.humanError(e);
          throw e;
        }).then(function (r) { self.$set(self.busy, key, false); return r; }, function () { self.$set(self.busy, key, false); });
      },
      exampleName: function () {
        var p = this.peers || {}, first = A(p.peers)[0];
        var suffix = p.magic_dns_suffix || ((this.st && this.st.server) || "").split(".").slice(1).join(".");
        return (first && first.dns_name) || "nas." + (suffix || "your-tailnet.ts.net");
      },
      say: function (msg) {
        var self = this;
        this.toast = msg;
        clearTimeout(this.toastT);
        this.toastT = setTimeout(function () { self.toast = ""; }, 2500);
      },
      humanError: function (e) {
        var msg = (e && (e.error || e.message)) || "";
        if (msg === "agent_down") return "tailbond isn't running on this router.";
        if (msg === "session") return "Your admin session ended. Reload the page and log in again.";
        if (/already exists/.test(msg)) return "That name is taken. Pick another one.";
        if (/name must be/.test(msg)) return "Use letters, numbers and - only, up to 63 characters.";
        if (/pairing|base64|tailbond:\/\/pair|JSON/i.test(msg)) return "That doesn't look like a tailbond pairing link. Copy the whole link from your home server and try again.";
        if (/public_addr/.test(msg)) return "Enter a host name or IP address, like home.example.com.";
        return msg ? "That didn't work: " + msg : "That didn't work. Try again.";
      },
      isOn: function () {
        if (!this.st) return false;
        if (this.st.role === "client") return this.paired && !(this.settings && this.settings.disconnected) && this.st.state !== "disconnected";
        return this.st.state !== "disconnected";
      },
      toggle: function () {
        var on = !this.isOn();
        this.wantOn = on;
        var self = this;
        return this.run("power", api("POST", on ? "/api/connect" : "/api/disconnect", {}),
          on ? "Turning on" : (self.role === "client" ? "Turned off. Devices on this router use the internet directly." : "Turned off"));
      },

      // ---- WANs (client) ----
      buildWans: function () {
        var st = this.st || {}, live = {}, out = [], seen = {};
        A(st.paths).forEach(function (p) { live[p.iface] = p; });
        A(this.paths).forEach(function (v) {
          if (!v.wan && !v.configured) return;
          var key = v.network || v.name;
          if (seen[key]) return;
          seen[key] = true;
          var p = live[v.name] || {};
          out.push({
            key: key, dev: v.name, kind: v.kind || p.kind || "unknown", label: v.label || p.label || KIND_LABEL[v.kind] || v.name,
            present: v.present, usable: v.present && v.up && v.carrier && A(v.addrs).length > 0,
            enabled: v.enabled, priority: v.priority, cap: v.data_cap_mb || 0, used: v.used_bytes || p.used_bytes || 0,
            state: p.state || (v.in_use ? "active" : "standby"), capReached: !!p.cap_reached,
            captive: p.captive || "", portalUrl: p.portal_url || "", signinUntil: p.signin_until || "", captiveSource: p.captive_source || "tailbond",
            rx: p.rx_bps || 0, tx: p.tx_bps || 0, rtt: p.srtt_ms || 0, addr: A(v.addrs)[0] || ""
          });
        });
        // paths the agent reports that GET /api/paths hasn't returned yet
        A(st.paths).forEach(function (p) {
          var key = p.network || p.iface;
          if (seen[key]) return;
          seen[key] = true;
          out.push({ key: key, dev: p.iface, kind: p.kind, label: p.label || KIND_LABEL[p.kind] || p.iface, present: p.state !== "failed",
            usable: p.state !== "failed", enabled: p.enabled, priority: p.priority, cap: p.data_cap_mb || 0, used: p.used_bytes || 0,
            state: p.state, capReached: !!p.cap_reached, captive: p.captive || "", portalUrl: p.portal_url || "", signinUntil: p.signin_until || "", captiveSource: p.captive_source || "tailbond", rx: p.rx_bps || 0, tx: p.tx_bps || 0, rtt: p.srtt_ms || 0, addr: "" });
        });
        out.sort(function (a, b) { return (KIND_ORDER[a.kind] - KIND_ORDER[b.kind]) || a.label.localeCompare(b.label); });
        return out;
      },
      // What a WAN card says: [chip class, chip text, explanation]
      wanStatus: function (w) {
        var on = this.st && this.st.state === "connected";
        var mode = this.st && this.st.mode;
        var mobile = w.kind === "usb" || w.kind === "cellular";
        if (!w.enabled) return ["", "Not used", "tailbond leaves this connection alone."];
        if (!w.usable) return ["", "Not connected", NOT_CONNECTED_FIX[w.kind] || NOT_CONNECTED_FIX.unknown];
        if (w.capReached) return ["err", "Paused · monthly limit reached", "Used again next month, or raise the limit."];
        if (w.captive === "portal") {
          if (w.signinUntil) return ["blue", "Signing in…", "Devices here reach this network directly until you've signed in."];
          return ["warn", "Needs sign-in", "This network shows a sign-in page first. Until you sign in, tailbond keeps it out and your other connections carry traffic."];
        }
        if (!on) return ["", "Ready", "Used when tailbond is on."];
        if (w.state === "active") {
          if (w.priority === "backup" || (mode === "failover" && w.priority !== "primary")) return ["warn", "In use · taking over", "Your main connection stopped working, so this one stepped in."];
          return ["ok", "In use", ""];
        }
        if (w.state === "pending") return ["blue", "Starting…", ""];
        if (w.state === "failed") return ["err", "Not working right now", "It's connected, but can't reach " + (this.serverShort || "your home server") + ". tailbond keeps trying; the other connections carry your traffic."];
        if (w.priority === "backup") return ["", mobile ? "Backup · saving data" : "Backup · standing by", "Used only if your other connections stop working. Until then it isn't used at all, so it uses no data."];
        if (mode === "failover") return ["", mobile ? "Standing by · saving data" : "Standing by", "Save mobile data is on: used only if Ethernet and Wi-Fi stop working."];
        return ["", "Standing by", ""];
      },
      summary: function () {
        var st = this.st;
        if (!st) return "";
        var server = this.serverShort || "your home server";
        if (st.role === "server") return "";
        if (!this.paired) return "Not set up yet. Connect this router to your home server below.";
        if (!this.isOn()) return "Devices on this router use the internet directly, without your home server. Turn on to bond your connections again.";
        if (st.state === "error" || (st.state === "connecting" && st.error)) return "Can't reach " + server + ". See below.";
        if (st.state === "connecting") return "Connecting to " + server + "…";
        var ws = this.wans, used = [], broken = [], saving = [];
        var self = this;
        ws.forEach(function (w) {
          var s = self.wanStatus(w)[1];
          if (s.indexOf("In use") === 0) used.push(KIND_NOUN[w.kind] || w.label);
          else if (s === "Not working right now") broken.push(KIND_NOUN[w.kind] || w.label);
          else if (/saving data|Backup/.test(s)) saving.push(KIND_NOUN[w.kind] || w.label);
        });
        var line = used.length ? "Using " + used.join(" + ") : "Connected, waiting for a connection to carry traffic";
        var portal = ws.filter(function (w) { return w.enabled && w.captive === "portal"; }).map(function (w) { return KIND_NOUN[w.kind] || w.label; });
        var signing = ws.filter(function (w) { return w.enabled && w.captive === "portal" && w.signinUntil; }).map(function (w) { return KIND_NOUN[w.kind] || w.label; });
        if (signing.length) line = "Signing in on " + signing.join(", ") + ": all traffic from devices here goes out " + signing.join(", ") + " directly, not through " + (this.serverShort || "your home server") + ", until you're signed in";
        else if (portal.length) line += " · " + portal.join(", ") + " needs sign-in";
        if (broken.length) line += " · " + broken.join(", ") + " not working right now";
        else if (saving.length) line += " · " + saving.join(", ") + " on standby";
        return line;
      },
      setPath: function (w, patch, msg) {
        var body = Object.assign({ name: w.key }, patch);
        return this.run("path-" + w.key, api("POST", "/api/paths", body), msg);
      },
      setMode: function (m) {
        if (this.st && this.st.mode === m) return;
        var label = MODES.filter(function (x) { return x.wire === m; })[0].label;
        return this.run("mode", api("POST", "/api/mode", { mode: m }), label + " is on. Reconnecting…");
      },

      // ---- pairing (client) ----
      ask: function (server) {
        var self = this;
        return this.run("ask", api("POST", "/api/join", { server: server })).then(function () { self.typed = ""; });
      },
      cancelAsk: function () { return this.run("ask", api("DELETE", "/api/join")); },
      pastePair: function () {
        var self = this;
        this.pasteErr = "";
        var uri = (this.pasteUri || "").trim();
        if (uri.indexOf("tailbond://pair") !== 0) { this.pasteErr = "Paste the whole link. It starts with tailbond://pair"; return; }
        return this.run("paste", api("POST", "/api/pair", { uri: uri }), "Paired. Connecting…").then(function () { self.pasteUri = ""; self.showOther = false; })
          .catch(function () { self.pasteErr = self.actErr; self.actErr = ""; });
      },
      unpair: function () {
        var self = this;
        this.confirm = {
          title: "Unpair from " + (this.serverShort || "the home server") + "?",
          body: "This router stops using tailbond and forgets the home server. Devices here use the internet directly. To connect again, ask the home server again (it will need to approve).",
          ok: "Unpair", danger: true,
          run: function () { return self.run("unpair", api("DELETE", "/api/pair"), "Unpaired").then(function () { self.disc = null; }); }
        };
      },
      repair: function () {
        var self = this;
        this.confirm = {
          title: "Ask " + (this.serverShort || "the home server") + " again?",
          body: "Only needed if the home server removed this router or was reset. tailbond stops, then asks " + (this.serverShort || "the home server") + " again; someone there approves it.",
          ok: "Ask again",
          run: function () {
            var server = self.serverShort;
            return self.run("unpair", api("DELETE", "/api/pair")).then(function () { self.disc = null; if (server) return self.ask(server); });
          }
        };
      },

      // ---- server ----
      decide: function (r, approve) {
        return this.run("req-" + r.id, api("POST", "/api/pair-requests/" + r.id + "/" + (approve ? "approve" : "decline"), {}),
          approve ? r.name + " can connect now. It connects by itself in a few seconds." : "Declined");
      },
      removeClient: function (c) {
        var self = this;
        this.confirm = {
          title: "Remove " + c.name + "?",
          body: c.name + " can't connect any more. To add it back, it asks again and you approve.",
          ok: "Remove", danger: true,
          run: function () { return self.run("rm-" + c.name, api("DELETE", "/api/clients/" + encodeURIComponent(c.name)), c.name + " removed"); }
        };
      },
      addClient: function () {
        var self = this;
        this.addErr = "";
        var n = (this.addName || "").trim();
        if (!n) { this.addErr = "Give it a name, like pixel-8."; return; }
        return this.run("add", api("POST", "/api/clients", { name: n })).then(function (r) { self.added = r; self.addName = ""; })
          .catch(function () { self.addErr = self.actErr; self.actErr = ""; });
      },
      saveAddr: function () {
        var self = this;
        return this.run("addr", api("POST", "/api/settings", { public_addr: (this.addrValue || "").trim() }), "Saved").then(function () { self.addrEdit = false; });
      },
      setRole: function (role) {
        var self = this;
        this.confirm = {
          title: role === "server" ? "Use this router as your home server?" : "Use this router as a travel router?",
          body: role === "server"
            ? "Travel routers and phones will connect to this router to bond their connections. Choose this on the router at home (for example a Flint 3), not on the one you travel with."
            : "This router will bond its own internet connections to your home server.",
          ok: "Switch",
          run: function () {
            return rpc("set_role", { role: role }).then(function () {
              self.say("Switching…");
              self.st = null; self.disc = null; self.peers = null;
              setTimeout(function () { self.refreshAll(); }, 4000);
            }).catch(function (e) { self.actErr = self.humanError(e); });
          }
        };
      },
      startService: function () {
        var self = this;
        return this.run("svc", rpc("service", { action: "start" }), "Starting tailbond…").then(function () { setTimeout(function () { self.refreshAll(); }, 2000); });
      },
      copy: function (text) {
        var self = this;
        var done = function () { self.say("Copied " + text); };
        if (navigator.clipboard && window.isSecureContext) navigator.clipboard.writeText(text).then(done, done);
        else {
          var ta = document.createElement("textarea");
          ta.value = text; document.body.appendChild(ta); ta.select();
          try { document.execCommand("copy"); } catch (e) { /* ignore */ }
          document.body.removeChild(ta); done();
        }
      }
    },

    render: function (h) {
      var self = this;
      var kids = [];
      if (!this.loaded) {
        kids.push(this.card(h, null, [h("div", { class: "tb-row" }, [h("div", { class: "tb-spin" }), "Loading tailbond…"])]));
      } else if (this.down) {
        kids.push(this.rDown(h));
      } else if (this.role === "server") {
        kids = kids.concat(this.rServer(h));
      } else {
        kids = kids.concat(this.rClient(h));
      }
      if (this.loaded && !this.down && this.st) {
        var ver = (this.info && this.info.version) || this.st.version || "";
        kids.push(h("div", { class: "tb-foot" }, [
          "tailbond " + ver,
          this.info && this.info.lan_dashboard ? h("span", [" · ", h("a", { class: "tb-link", attrs: { href: "http://" + location.hostname + ":8077/", target: "_blank", rel: "noopener" } }, "Advanced page (:8077)")]) : null
        ]));
      }
      if (this.confirm) kids.push(this.rConfirm(h));
      if (this.toast) kids.push(h("div", { class: "tb-toast" }, this.toast));
      return h("div", { class: "tb" }, kids);
    },

    // Render helpers (a mixin keeps render() short; Vue binds `this`).
    mixins: [{
      methods: {
        card: function (h, title, body, extra, cls) {
          return h("div", { class: "tb-card" + (cls ? " " + cls : "") }, [
            title ? h("div", { class: "tb-card-h" }, [h("span", title), extra || null]) : null,
            body ? h("div", { class: "tb-card-b" }, body) : null
          ]);
        },
        btn: function (h, text, onClick, cls, disabled) {
          return h("button", { class: "tb-btn " + (cls || ""), attrs: { type: "button", disabled: !!disabled }, on: { click: onClick } }, text);
        },
        sw: function (h, on, onClick, big, disabled, label) {
          return h("button", { class: "tb-switch" + (on ? " on" : "") + (big ? " tb-switch-big" : ""), attrs: { type: "button", role: "switch", "aria-checked": String(!!on), "aria-label": label || "", disabled: !!disabled }, on: { click: onClick } });
        },
        alert: function (h, cls, title, body, actions) {
          return h("div", { class: "tb-alert " + (cls || "") }, [
            h("div", { style: { flex: "none", marginTop: "1px" } }, [icon(h, cls === "ok" ? "check" : "warn", 20)]),
            h("div", { class: "tb-grow" }, [
              title ? h("div", { class: "tb-alert-t" }, title) : null,
              body ? h("div", body) : null,
              actions && actions.length ? h("div", { class: "tb-actions" }, actions) : null
            ])
          ]);
        },
        hero: function (h, iconCls, title, line, sw, rates) {
          return h("div", { class: "tb-card" }, [
            h("div", { class: "tb-hero" }, [
              h("div", { class: "tb-hero-icon " + iconCls }, [icon(h, "bond", 28)]),
              h("div", { class: "tb-grow" }, [h("div", { class: "tb-hero-t" }, title), h("div", { class: "tb-hero-s" }, line)]),
              sw
            ]),
            rates || null
          ]);
        },

        rDown: function (h) {
          var self = this;
          return this.card(h, null, [this.alert(h, "err", "tailbond isn't running on this router",
            "It may have been turned off or stopped after an error. Starting it is safe: devices here keep their internet while you set it up.",
            [this.btn(h, this.busy.svc ? "Starting…" : "Start tailbond", function () { self.startService(); }, "primary", this.busy.svc)])]);
        },

        rConfirm: function (h) {
          var self = this, c = this.confirm;
          return h("div", { class: "tb-modal", on: { click: function (e) { if (e.target === e.currentTarget) self.confirm = null; } } }, [
            h("div", { class: "tb-dialog", attrs: { role: "dialog" } }, [
              h("div", { class: "tb-dialog-t" }, c.title),
              h("div", { style: { color: "#606266", lineHeight: 1.55 } }, c.body),
              h("div", { class: "tb-actions", style: { justifyContent: "flex-end", marginTop: "18px" } }, [
                this.btn(h, "Cancel", function () { self.confirm = null; }),
                this.btn(h, c.ok, function () { var r = c.run; self.confirm = null; r(); }, c.danger ? "danger" : "primary")
              ])
            ])
          ]);
        },

        // ================= client role =================
        rClient: function (h) {
          var self = this, st = this.st, out = [];
          var on = this.wantOn !== null ? this.wantOn : this.isOn();
          var err = st.state === "error" || (st.state === "connecting" && !!st.error);
          var iconCls = !this.paired ? "" : !on ? "" : err ? "err" : st.state === "connected" ? "on" : "warn";
          var signingIn = this.wans.some(function (w) { return w.captive === "portal" && w.signinUntil; });
          var title = !this.paired ? "Not connected to a home server" : !on ? "tailbond is off" :
            signingIn && st.state === "connected" ? "Router connected to " + this.serverShort + " · devices signing in" : err ? "Can't reach " + (this.serverShort || "home server") :
            st.state === "connected" ? "Connected to " + this.serverShort : "Connecting…";
          var rates = this.paired && on && st.state === "connected" ? h("div", { class: "tb-rates" }, [
            h("span", ["↓ ", h("b", fmtRate(st.totals.rx_bps))]), h("span", ["↑ ", h("b", fmtRate(st.totals.tx_bps))]),
            h("span", { class: "tb-hide-sm" }, ["Up ", h("b", fmtUptime(st.uptime_sec))])
          ]) : null;
          out.push(this.hero(h, iconCls, title, this.summary(),
            this.paired ? this.sw(h, on, function () { self.toggle(); }, true, this.busy.power, "tailbond on/off") : null, rates));
          if (this.actErr) out.push(this.alert(h, "err", null, this.actErr));

          if (!this.paired) return out.concat(this.rSignin(h), this.rPairing(h));

          out = out.concat(this.rSignin(h));
          if (on && err) out.push(this.rConnError(h));
          out.push(this.rWans(h));
          out.push(this.rModes(h));
          out.push(this.rHome(h));
          out.push(this.rMachines(h));
          return out;
        },

        rConnError: function (h) {
          var self = this, st = this.st, server = this.serverShort || "your home server";
          var e = st.error || "", title, body;
          var addr = (this.settings && this.settings.paired_with) || st.server || "";
          var anyUp = this.wans.some(function (w) { return w.usable && w.enabled; });
          if (!anyUp) {
            title = "No internet connection to use";
            body = "None of this router's connections is online. Join hotel Wi-Fi in INTERNET → Repeater, or plug in your phone and turn on USB tethering.";
          } else if (/resolve/i.test(e)) {
            title = "Can't find " + shortName(addr);
            body = "The router is online, but the name " + shortName(addr) + " doesn't resolve. If the home server's internet address changed, whoever set it up needs to update it.";
          } else if (/pin|certificate|x509|tls/i.test(e)) {
            title = server + "'s security certificate changed";
            body = "tailbond stopped to be safe. If " + server + " was reinstalled, pair again; otherwise tell whoever set it up.";
          } else {
            title = "Can't reach " + server;
            body = "This router is online, but " + server + " isn't answering at " + addr + ". It may be off, or its internet connection is down. tailbond keeps trying.";
          }
          var ks = !(this.info && this.info.kill_switch === false);
          return this.alert(h, "err", title, [h("div", body),
            h("div", { style: { marginTop: "8px", fontWeight: 600 } }, ks
              ? "Meanwhile, devices on this router have no internet, so nothing goes around your home server. To use this internet directly for now, turn tailbond off."
              : "Meanwhile, devices on this router use this internet directly."),
            h("div", { class: "tb-small", style: { marginTop: "6px", opacity: 0.85 } }, "Still not working? Whoever set up " + server + " should check that it's on and that UDP port 443 reaches " + addr + " from the internet.")],
            [this.btn(h, this.busy.svc ? "Trying…" : "Try again", function () { self.run("svc", rpc("service", { action: "restart" }), "Trying again…"); }, "primary", this.busy.svc),
              ks ? this.btn(h, "Turn tailbond off", function () { self.toggle(); }, "", this.busy.power) : null]);
        },

        rWans: function (h) {
          var self = this, ws = this.wans;
          var cards = ws.map(function (w) { return self.rWan(h, w); });
          if (!cards.length) cards = [h("div", { class: "tb-muted" }, "No internet connections found yet. Set one up on the INTERNET page: cable, Repeater, Tethering or Cellular.")];
          var s = this.settings || {}, capOn = s.captive_portal_detect !== false, self2 = this;
          return this.card(h, "Internet connections", [
            h("div", { class: "tb-muted", style: { marginBottom: "12px" } }, "tailbond uses every connection the router is online with. Turn one off, or keep it as a backup."),
            h("div", { class: "tb-grid" }, cards),
            this.info && this.info.acceleration === "on" ? h("div", { class: "tb-muted tb-small", style: { marginTop: "12px", lineHeight: 1.5 } }, [
              "GL's Network Acceleration is on. tailbond works with it; connections that were open before tailbond connected may keep going around it until they end. ",
              h("a", { class: "tb-link", attrs: { href: "#/acceleration" } }, "NETWORK → Network Acceleration")]) : null,
            h("div", { class: "tb-row", style: { marginTop: "14px", borderTop: "1px solid #ebeef5", paddingTop: "12px" } }, [
              h("div", { class: "tb-grow" }, [h("div", "Check for sign-in pages (hotel, airport Wi-Fi)"),
                h("div", { class: "tb-muted tb-small" }, "A connection that needs a sign-in is kept out until you've signed in, so it can't stall the others.")]),
              this.sw(h, capOn, function () { self2.run("cap", api("POST", "/api/settings", { captive_portal_detect: !capOn }), capOn ? "Sign-in page check off" : "Sign-in page check on"); }, false, this.busy.cap, "Check for sign-in pages")
            ])
          ]);
        },

        rWan: function (h, w) {
          var self = this, s = this.wanStatus(w), busy = this.busy["path-" + w.key];
          var mobile = w.kind === "usb" || w.kind === "cellular";
          var live = s[1].indexOf("In use") === 0;
          var usage = [];
          var capTxt = w.cap ? fmtBytes(w.used) + " of " + fmtCap(w.cap) + " this month" : fmtBytes(w.used) + " this month";
          usage.push(h("div", { class: "tb-row tb-small" }, [
            h("span", { class: "tb-grow tb-muted" }, "Data used: " + capTxt),
            this.capEdit === w.key ? null : h("a", { class: "tb-link", on: { click: function () { self.capEdit = w.key; self.capValue = w.cap ? String(w.cap / 1000) : ""; } } }, w.cap ? "Change limit" : "Set limit")
          ]));
          if (w.cap) usage.push(h("div", { class: "tb-bar" }, [h("i", { class: w.used >= w.cap * 1e6 ? "full" : "", style: { width: Math.min(100, (w.used / (w.cap * 1e6)) * 100) + "%" } })]));
          if (this.capEdit === w.key) {
            usage.push(h("div", { class: "tb-row", style: { marginTop: "8px" } }, [
              h("input", { class: "tb-input", style: { width: "110px" }, attrs: { type: "number", min: 0, step: 0.5, placeholder: "GB" }, domProps: { value: this.capValue }, on: { input: function (e) { self.capValue = e.target.value; } } }),
              h("span", { class: "tb-muted" }, "GB a month"),
              this.btn(h, "Save", function () {
                var gb = parseFloat(self.capValue || "0") || 0;
                self.setPath(w, { data_cap_mb: Math.round(gb * 1000) }, gb ? "Limit set: " + gb + " GB a month" : "No limit").then(function () { self.capEdit = null; });
              }, "primary small"),
              this.btn(h, "Cancel", function () { self.capEdit = null; }, "small text")
            ]));
            usage.push(h("div", { class: "tb-muted tb-small", style: { marginTop: "4px" } }, "Counts everything tailbond sends and receives over it. At the limit, tailbond stops using this connection until the count restarts on the 1st of each month. 0 = no limit."));
          }
          return h("div", { class: "tb-wan" + (live ? "" : " dim") }, [
            h("div", { class: "tb-row" }, [
              h("div", { class: "tb-wan-icon" }, [icon(h, w.kind === "unknown" ? "unknown" : w.kind)]),
              h("div", { class: "tb-grow" }, [
                h("div", { class: "tb-wan-t" }, w.label),
                h("div", { class: "tb-muted tb-small" }, live ? "↓ " + fmtRate(w.rx) + "  ↑ " + fmtRate(w.tx) + (w.rtt ? "  ·  " + w.rtt + " ms" : "") : (w.addr ? w.addr.split("/")[0] : w.dev))
              ]),
              h("span", { class: "tb-chip " + s[0] }, s[1])
            ]),
            s[2] ? h("div", { class: "tb-muted tb-small", style: { lineHeight: 1.5 } }, s[2]) : null,
            h("div", usage),
            h("div", { class: "tb-wan-ctl" }, [
              h("span", { class: "tb-muted" }, "Use it"),
              this.rUseChoice(h, w, busy)
            ])
          ]);
        },

        // A WAN behind a captive portal: one card per WAN, either our own
        // sign-in mode or, on the repeater when GL handles it, GL's page.
        rSignin: function (h) {
          var self = this;
          return this.wans.filter(function (w) { return w.enabled && w.captive === "portal"; }).map(function (w) {
            var noun = w.kind === "wifi" ? "This hotel or airport Wi-Fi" : w.label;
            var until = w.signinUntil ? new Date(w.signinUntil) : null;
            var hhmm = until ? ("0" + until.getHours()).slice(-2) + ":" + ("0" + until.getMinutes()).slice(-2) : "";
            if (w.captiveSource === "glinet") {
              return self.alert(h, "info", w.signinUntil ? "Signing in to " + w.label + " through GL's Login Mode" : w.label + " needs you to sign in", [
                w.signinUntil ? null : h("div", "GL's Repeater page handles this sign-in. Open it, click " + "“Login Mode for Public Hotspots” (or “Go to Login”) and accept the hotel's terms."),
                h("div", { style: { marginTop: "6px" } }, w.signinUntil
                  ? "GL's Login Mode is on: all traffic from devices here goes out the hotel Wi-Fi directly (not just the sign-in page), not through your home server, so the page loads. The router itself stays connected to your home server over your other connections (that's what the top card shows), so they keep being used. It ends by itself once you've signed in (or click “Quit Login Mode” on GL's Repeater page); tailbond then adds " + w.label + " back to bonding."
                  : "While you sign in, devices here use the hotel Wi-Fi directly, not through your home server. Afterwards tailbond adds " + w.label + " back to bonding by itself. Your other connections keep working meanwhile.")
              ], [h("a", { class: "tb-btn small" + (w.signinUntil ? "" : " primary"), attrs: { href: "#/internet" } }, w.signinUntil ? "Back to GL's Repeater page" : "Open GL's Repeater page")]);
            }
            if (w.signinUntil) {
              var mins = Math.max(1, Math.round((until - Date.now()) / 60000));
              return self.alert(h, "info", "Signing in to " + w.label + "…", [
                h("div", ["Sign in on the page that opened (accept the terms, enter the room number…). For up to " + mins + " more minute" + (mins === 1 ? "" : "s") + " (until " + hhmm + " on this router's clock), all traffic from devices here goes out " + w.label + " directly, not just the sign-in page, and not through your home server. The router's own tunnel stays up on your other connections. tailbond adds " + w.label + " back by itself once you're signed in."])
              ], [self.btn(h, "Open the sign-in page again", function () { window.open(w.portalUrl || "http://neverssl.com/", "_blank", "noopener"); }, "small primary"),
                self.btn(h, "I've signed in, check now", function () { self.run("chk-" + w.key, api("POST", "/api/captive/check", { name: w.key }), "Checking…"); }, "small"),
                self.btn(h, "Stop sign-in mode", function () { self.run("si-" + w.key, api("DELETE", "/api/captive/signin"), "Sign-in mode stopped"); }, "small text")]);
            }
            return self.alert(h, "", noun + " needs you to sign in", [
              w.kind === "wifi" ? null : h("div", { class: "tb-small", style: { marginBottom: "4px" } }, "GL's Login Mode covers the repeater Wi-Fi only; tailbond handles sign-in pages on other connections like this one."),
              h("div", "It shows a sign-in page before it lets anyone online. Until then tailbond keeps " + w.label + " out of bonding" + (self.wans.some(function (x) { return x !== w && x.enabled && x.usable && x.captive !== "portal"; }) ? " and your other connections carry your traffic." : ".")),
              h("div", { class: "tb-small", style: { marginTop: "6px" } }, "Opening the sign-in page lets devices here reach " + w.label + " directly (without your home server) for up to 10 minutes, so the page can load.")
            ], [self.btn(h, self.busy["si-" + w.key] ? "Opening…" : "Open sign-in page", function () {
              var win = window.open("about:blank", "_blank");
              self.run("si-" + w.key, api("POST", "/api/captive/signin", { name: w.key })).then(function (r) {
                var u = (r && r.portal_url) || w.portalUrl || "http://neverssl.com/";
                if (win) win.location = u; else window.open(u, "_blank", "noopener");
              }).catch(function () { if (win) win.close(); });
            }, "primary", self.busy["si-" + w.key])]);
          });
        },

        // Speedify-style priority in GL words: Use / Backup only / Off.
        rUseChoice: function (h, w, busy) {
          var self = this, mobile = w.kind === "usb" || w.kind === "cellular";
          var cur = !w.enabled ? "off" : w.priority === "backup" ? "backup" : "use";
          var opts = [
            ["use", "Always", { enabled: true, priority: mobile ? "secondary" : "primary" }, w.label + " is used together with the others"],
            ["backup", "Backup only", { enabled: true, priority: "backup" }, w.label + " is used only if the others stop working"],
            ["off", "Off", { enabled: false }, "tailbond won't use " + w.label]
          ];
          return h("div", { class: "tb-seg", attrs: { role: "radiogroup", "aria-label": "Use " + w.label } }, opts.map(function (o) {
            return h("button", { class: (cur === o[0] ? "on" : "") + (o[0] === "off" ? " off" : ""), attrs: { type: "button", role: "radio", "aria-checked": String(cur === o[0]), disabled: !!busy },
              on: { click: function () { if (cur !== o[0]) self.setPath(w, o[2], o[3]); } } }, o[1]);
          }));
        },

        rModes: function (h) {
          var self = this, cur = this.st.mode;
          // Saving phone data is per connection ("Backup only" above); the
          // failover mode is shown only if something else picked it.
          var shown = MODES.filter(function (m) { return m.wire !== "failover" || cur === "failover"; });
          return this.card(h, "Speed or reliability", [
            h("div", { class: "tb-modes" + (shown.length === 3 ? " three" : "") }, shown.map(function (m) {
              return h("button", { class: "tb-mode" + (cur === m.wire ? " on" : ""), attrs: { type: "button", title: m.wire }, on: { click: function () { self.setMode(m.wire); } } }, [
                h("div", { class: "tb-mode-t" }, m.label), h("div", { class: "tb-mode-b" }, m.blurb)
              ]);
            })),
            h("div", { class: "tb-muted tb-small", style: { marginTop: "10px" } }, "Saving your phone's data? Set Tethering or Cellular to “Backup only” under Internet connections.")
          ]);
        },

        rHome: function (h) {
          var self = this, st = this.st, s = this.settings || {};
          var stateChip = st.state === "connected" ? ["ok", "Connected"] : !this.isOn() ? ["", "Off"] : st.state === "connecting" && !st.error ? ["blue", "Connecting…"] : ["err", "Can't reach"];
          return this.card(h, "Home server", [
            h("div", { class: "tb-row", style: { marginBottom: "14px" } }, [
              h("div", { class: "tb-wan-icon" }, [icon(h, "server")]),
              h("div", { class: "tb-grow" }, [
                h("div", { class: "tb-wan-t" }, this.serverShort || "Home server"),
                h("div", { class: "tb-muted tb-small" }, st.state === "connected" ? "Connected for " + fmtUptime(st.uptime_sec) + " · this router is " + (s.paired_as || "") : (s.paired_with || st.server || ""))
              ]),
              h("span", { class: "tb-chip " + stateChip[0] }, stateChip[1])
            ]),
            h("div", { class: "tb-kv tb-small" }, [
              h("div", "Connects to"), h("div", [h("span", { class: "tb-mono" }, hostOf(s.paired_with || st.server || "—")), " over the internet (UDP port " + (portOf(s.paired_with) || "443") + ")"]),
              h("div", "Tailnet names"), h("div", "Work on every device connected to this router while tailbond is on. Use the full name, like " + this.exampleName() + "."),
              h("div", "This router's tunnel address"), h("div", { class: "tb-mono" }, st.tunnel_ip || "—")
            ]),
            h("div", { class: "tb-actions", style: { marginTop: "14px" } }, [
              this.btn(h, "Ask to connect again", function () { self.repair(); }, "small"),
              this.btn(h, "Unpair", function () { self.unpair(); }, "small danger")
            ])
          ]);
        },

        rMachines: function (h, serverSide) {
          var self = this, p = this.peers;
          var body;
          if (!serverSide && (!this.st || this.st.state !== "connected") && !p) {
            body = [h("div", { class: "tb-muted" }, "The list appears once tailbond is connected.")];
          } else if (!p && this.peersErr) {
            body = [h("div", { class: "tb-muted" }, "Couldn't load the list from " + (this.serverShort || "the home server") + " right now.")];
          } else if (!p) {
            body = [h("div", { class: "tb-row tb-muted" }, [h("div", { class: "tb-spin" }), "Loading…"])];
          } else if (!p.tailscale_running) {
            body = [h("div", { class: "tb-muted" }, serverSide ? "Tailscale isn't on on this router. Turn it on in APPLICATIONS → Tailscale." : "Tailscale isn't on at " + (p.server || "the home server") + ", so your tailnet machines can't be reached right now.")];
          } else {
            var q = (this.peerFilter || "").toLowerCase();
            var list = A(p.peers).filter(function (x) { return !q || (x.hostname + " " + x.dns_name + " " + A(x.tailscale_ips).join(" ")).toLowerCase().indexOf(q) >= 0; });
            body = [
              h("div", { class: "tb-muted", style: { marginBottom: "10px" } }, serverSide
                ? "Devices connected through tailbond reach these machines by name or 100.x address. “Direct” / “via relay” is how this router reaches each one."
                : "Laptops and phones on this router's Wi-Fi or LAN reach these through the bond to " + (this.serverShort || "your home server") + " (not this router's own Tailscale), by their full name, e.g. " + this.exampleName() + ", or by their 100.x address. “Direct” / “via relay” is how " + (this.serverShort || "your home server") + " reaches each machine."),
              A(p.peers).length > 6 ? h("input", { class: "tb-input", style: { marginBottom: "8px" }, attrs: { placeholder: "Search machines" }, domProps: { value: this.peerFilter }, on: { input: function (e) { self.peerFilter = e.target.value; } } }) : null,
              h("div", { class: "tb-list" }, list.map(function (m) {
                var ip = A(m.tailscale_ips)[0] || "";
                return h("div", { class: "tb-li" }, [
                  h("span", { class: "tb-dot" + (m.online ? " ok" : "") }),
                  h("div", { class: "tb-grow" }, [
                    h("div", { style: { fontWeight: 600 } }, m.hostname),
                    h("div", { class: "tb-mono" }, dotBreaks(h, m.dns_name)),
                    ip ? h("div", { class: "tb-mono tb-nowrap" }, ip) : null,
                    h("div", { class: "tb-muted tb-small" }, [(m.os ? m.os + " · " : "") + lastSeen(m),
                      m.conn === "relay" ? h("span", { style: { color: "#c88a1a" } }, " · via relay (slower)") : m.conn === "direct" ? " · direct" : null])
                  ]),
                  self.btn(h, "Copy name", function () { self.copy(m.dns_name); }, "small")
                ]);
              }))
            ];
          }
          var extra = p && p.tailnet ? h("span", { class: "tb-muted tb-small", style: { fontWeight: 400 } }, p.tailnet) : null;
          return this.card(h, "Tailnet machines", body, extra);
        },

        // --- not paired ---
        rPairing: function (h) {
          var self = this, out = [], j = this.join, d = this.disc;
          var body = [];
          if (j && (j.state === "asking" || j.state === "waiting")) {
            var srv = j.server_name || shortName(j.server);
            body.push(h("div", { class: "tb-found" }, [
              h("div", { class: "tb-spin", style: { width: "28px", height: "28px", borderWidth: "3px" } }),
              h("div", { class: "tb-grow" }, [
                h("div", { class: "tb-wan-t" }, j.state === "asking" ? "Asking " + srv + "…" : "Waiting for approval on " + srv + "…"),
                h("div", { class: "tb-muted", style: { marginTop: "4px", lineHeight: 1.6 } }, [
                  "On " + srv + ", tap ", h("b", "Approve"), " next to ", h("b", j.name || "this router"), ". This router connects by itself once approved; you can close this page.",
                  h("div", { style: { marginTop: "6px" } }, ["From anywhere: open ", h("a", { class: "tb-link", attrs: { href: "http://" + hostOf(j.server) + ":8077/", target: "_blank", rel: "noopener" } }, "http://" + srv + ":8077"),
                    " on any device signed in to your Tailscale account (a phone works too). At home with a Flint 3, its admin panel → VPN → tailbond works as well."])
                ])
              ]),
              h("div", { class: "tb-actions" }, [this.btn(h, "Cancel", function () { self.cancelAsk(); })])
            ]));
          } else {
            if (j && j.state === "declined") body.push(this.alert(h, "err", (j.server_name || "The home server") + " declined", "Ask whoever runs it to approve this router, then ask again."));
            if (j && j.state === "expired") body.push(this.alert(h, "", "Nobody approved it in time", "The request expired after 10 minutes. Ask again when someone can approve it on " + (j.server_name || "the home server") + "."));
            if (j && j.state === "error") body.push(this.alert(h, "err", this.joinErrTitle(j), this.joinErrBody(j)));

            if (this.discovering && !d) {
              body.push(h("div", { class: "tb-row", style: { padding: "8px 0 16px" } }, [h("div", { class: "tb-spin" }), "Looking for your home server on your tailnet…"]));
            } else if (d && !d.tailscale_running) {
              body.push(this.alert(h, "info", "Turn on Tailscale first", [
                h("div", "tailbond finds your home server through Tailscale. Turn on Tailscale on this router and log in with the same account as your home server."),
              ], [h("a", { class: "tb-btn primary small", attrs: { href: "#/tailscaleview" } }, "Open Tailscale settings"),
                this.btn(h, "Search again", function () { self.discover(true); }, "small")]));
            } else if (d && A(d.servers).length) {
              A(d.servers).forEach(function (s) {
                body.push(h("div", { class: "tb-found" }, [
                  h("div", { class: "tb-wan-icon" }, [icon(h, "server")]),
                  h("div", { class: "tb-grow" }, [
                    h("div", { class: "tb-wan-t" }, "Found your home server: " + s.name),
                    h("div", { class: "tb-muted tb-small" }, s.dns_name + " · " + s.ip)
                  ]),
                  h("div", { class: "tb-actions" }, [self.btn(h, self.busy.ask ? "Asking…" : "Ask to connect", function () { self.ask(s.dns_name || s.ip); }, "primary", self.busy.ask)])
                ]));
              });
            } else if (d) {
              body.push(this.alert(h, "", "No home server found on your tailnet", [
                h("div", "Checked " + (d.checked || 0) + " online machine" + (d.checked === 1 ? "" : "s") + ". On the home server, tailbond must be installed and running, and Tailscale on. Or type its Tailscale name below.")
              ], [this.btn(h, this.discovering ? "Searching…" : "Search again", function () { self.discover(true); }, "small", this.discovering)]));
            }
            var found = d && A(d.servers).length;
            if (found && !this.showTyped) {
              body.push(h("div", { class: "tb-small" }, [h("a", { class: "tb-link", on: { click: function () { self.showTyped = true; } } }, "Not your home server? Type its name instead")]));
            } else {
              body.push(h("div", { class: "tb-muted", style: { margin: "6px 0 6px" } }, "Home server's name in Tailscale"));
              body.push(h("div", { class: "tb-row" }, [
                h("input", { class: "tb-input", attrs: { placeholder: "e.g. homebox" }, domProps: { value: this.typed }, on: { input: function (e) { self.typed = e.target.value; }, keyup: function (e) { if (e.key === "Enter" && self.typed.trim()) self.ask(self.typed.trim()); } } }),
                this.btn(h, "Ask to connect", function () { if (self.typed.trim()) self.ask(self.typed.trim()); }, "", !this.typed.trim() || this.busy.ask)
              ]));
            }
          }
          out.push(this.card(h, "Connect to your home server", [
            h("div", { class: "tb-muted", style: { marginBottom: "14px", lineHeight: 1.55 } }, "Your home server (a computer or a GL Flint 3 running tailbond) must be signed in to the same Tailscale account as this router. You'll approve this router there with one tap."),
            h("div", body)
          ]));

          // fallback: pairing link / code
          out.push(this.card(h, null, [
            h("div", { class: "tb-row" }, [
              h("div", { class: "tb-grow" }, [h("div", { style: { fontWeight: 600 } }, "Other ways to connect"),
                h("div", { class: "tb-muted tb-small" }, "No Tailscale on this router? Paste a pairing link made on the home server's Devices page.")]),
              this.btn(h, this.showOther ? "Hide" : "Paste a link", function () { self.showOther = !self.showOther; }, "small")
            ]),
            this.showOther ? h("div", { style: { marginTop: "12px" } }, [
              h("textarea", { class: "tb-input", attrs: { placeholder: "tailbond://pair?d=…" }, domProps: { value: this.pasteUri }, on: { input: function (e) { self.pasteUri = e.target.value; } } }),
              this.pasteErr ? h("div", { style: { color: "#f56c6c", marginTop: "6px" } }, this.pasteErr) : null,
              h("div", { class: "tb-actions" }, [this.btn(h, this.busy.paste ? "Pairing…" : "Pair", function () { self.pastePair(); }, "primary", this.busy.paste)])
            ]) : null
          ]));
          out.push(h("div", { class: "tb-muted tb-small", style: { textAlign: "center", marginBottom: "16px" } }, [
            "Setting up the router that stays at home? ", h("a", { class: "tb-link", on: { click: function () { self.setRole("server"); } } }, "Make this the home server"), " (you can switch back here)"
          ]));
          return out;
        },
        joinErrTitle: function (j) {
          var s = j.server_name || "the home server";
          return { unreachable: "Couldn't reach " + s, not_tailnet: s + " doesn't accept requests from here", too_many: "Too many requests", not_server: s + " isn't a tailbond home server", bad_pairing: "Approved, but the pairing didn't work", server_error: s + " had a problem" }[j.error] || "Couldn't ask " + s;
        },
        joinErrBody: function (j) {
          var s = j.server_name || "the home server";
          return {
            unreachable: "Check that " + s + " is on, tailbond is running there, and the name is right. Both must be on the same tailnet.",
            not_tailnet: "It only accepts requests over Tailscale. Make sure Tailscale is on here and you typed its Tailscale name.",
            too_many: "Wait 10 minutes, then ask again.",
            not_server: "tailbond on " + s + " is set up as a travel router. Pick your home server instead.",
            bad_pairing: "Ask again. If it keeps happening, update tailbond on both routers.",
            server_error: "Ask again in a minute."
          }[j.error] || "Ask again in a minute.";
        },

        // ================= server role =================
        rServer: function (h) {
          var self = this, st = this.st, out = [];
          var on = this.wantOn !== null ? this.wantOn : this.isOn();
          var conn = A(st.clients).filter(function (c) { return c.connected; });
          var err = st.state === "error";
          var todo = this.setupSteps().length;
          var line = !on ? "Off. Travel routers and phones can't connect." : err ? "Couldn't start: " + (st.error || "unknown error") :
            todo ? "Finish setup below so travel routers can connect from anywhere." :
            "Ready for travel routers and phones · " + (conn.length ? conn.length + " connected now" : "none connected right now");
          var title = !on ? "Home server is off" : err ? "Home server has a problem" : todo ? "Home server is on · " + todo + " setup step" + (todo === 1 ? "" : "s") + " left" : "Home server is on";
          out.push(this.hero(h, !on ? "" : err ? "err" : todo ? "warn" : "on", title, line,
            this.sw(h, on, function () { self.toggle(); }, true, this.busy.power, "Home server on/off"),
            on && conn.length ? h("div", { class: "tb-rates" }, [h("span", ["Sending to devices ", h("b", fmtRate(st.totals.tx_bps))]), h("span", ["Receiving ", h("b", fmtRate(st.totals.rx_bps))])]) : null));
          if (this.actErr) out.push(this.alert(h, "err", null, this.actErr));

          if (this.pending.length) {
            out.push(this.card(h, "Waiting for your approval", this.pending.map(function (r) {
              var what = r.kind === "router" ? "Travel router" : r.kind === "phone" ? "Phone" : "Device";
              return h("div", { class: "tb-req" }, [
                h("div", { class: "tb-wan-icon" }, [icon(h, r.kind === "phone" ? "phone" : "router")]),
                h("div", { class: "tb-grow" }, [
                  h("div", { class: "tb-wan-t" }, (r.repair ? r.name + " is asking to connect again" : r.name + " wants to connect")),
                  h("div", { class: "tb-muted tb-small", style: { lineHeight: 1.6 } }, [
                    what + " " + (r.device && r.device !== r.name ? "“" + r.device + "” " : "") + "on your tailnet · signed in as " + (r.user || "unknown"),
                    h("br"), (r.hostname ? r.hostname + " · " : "") + (r.os ? r.os + " · " : "") + r.ip + " · asked " + fmtAgo(r.created)
                  ]),
                  r.repair ? h("div", { class: "tb-small", style: { marginTop: "4px", color: "#8a5d10" } }, "It was connected before (reset or reinstalled). Approving gives it its old access back.") : null
                ]),
                h("div", { class: "tb-actions" }, [
                  self.btn(h, "Decline", function () { self.decide(r, false); }, "", self.busy["req-" + r.id]),
                  self.btn(h, "Approve", function () { self.decide(r, true); }, "primary", self.busy["req-" + r.id])
                ])
              ]);
            }).concat([this.whoApproves(h)]), h("span", { class: "tb-chip warn" }, this.pending.length + " new"), "tb-pending"));
          }

          // unfinished setup first, else devices first
          if (todo) out.push(this.rRoute(h), this.rClients(h));
          else out.push(this.rClients(h), this.rRoute(h));
          out.push(this.rHealth(h));
          out.push(this.rRouterGuide(h));
          out.push(this.rMachines(h, true));
          out.push(h("div", { class: "tb-muted tb-small", style: { textAlign: "center", marginBottom: "16px" } }, [
            "Is this a travel router? ", h("a", { class: "tb-link", on: { click: function () { self.setRole("client"); } } }, "Use it as a travel router instead"), " (you can switch back here)"
          ]));
          return out;
        },

        rClients: function (h) {
          var self = this, st = this.st;
          var cs = A(st.clients).slice().sort(function (a, b) { return (b.connected - a.connected) || a.name.localeCompare(b.name); });
          var rows = cs.map(function (c) {
            var n = A(c.paths).filter(function (p) { return p.state === "active"; }).length;
            return h("div", { class: "tb-li" }, [
              h("span", { class: "tb-dot" + (c.connected ? " ok" : "") }),
              h("div", { class: "tb-grow" }, [
                h("div", { style: { fontWeight: 600 } }, c.name),
                h("div", { class: "tb-muted tb-small" }, c.connected
                  ? "Connected " + fmtUptime(c.connected_sec) + " · " + n + " connection" + (n === 1 ? "" : "s") + " bonded · downloading " + fmtRate(c.tx_bps) + " · uploading " + fmtRate(c.rx_bps)
                  : "Not connected right now")
              ]),
              self.btn(h, "Remove", function () { self.removeClient(c); }, "small text", self.busy["rm-" + c.name])
            ]);
          });
          if (!rows.length) rows = [h("div", { class: "tb-muted", style: { padding: "8px 0" } }, "No devices yet. On a travel router, open VPN → tailbond and tap Ask to connect; it shows up above for approval.")];
          var add = [];
          if (this.added) {
            add.push(h("div", { class: "tb-row", style: { alignItems: "flex-start", gap: "16px", marginTop: "12px", flexWrap: "wrap" } }, [
              h("div", { class: "tb-qr", domProps: { innerHTML: this.added.qr_svg } }),
              h("div", { class: "tb-grow", style: { minWidth: "220px" } }, [
                h("div", { style: { fontWeight: 600 } }, "Scan with the tailbond app on " + this.added.name),
                h("div", { class: "tb-muted tb-small", style: { margin: "6px 0" } }, "Or copy the link to a travel router without Tailscale. It's just for this device; keep it to yourself."),
                h("div", { class: "tb-actions" }, [this.btn(h, "Copy link", function () { self.copy(self.added.uri); }, "small"), this.btn(h, "Done", function () { self.added = null; }, "small")])
              ])
            ]));
          } else {
            add.push(h("div", { class: "tb-row", style: { marginTop: "12px" } }, [
              h("input", { class: "tb-input", attrs: { placeholder: "Name, e.g. pixel-8" }, domProps: { value: this.addName }, on: { input: function (e) { self.addName = e.target.value; } } }),
              this.btn(h, "Add with a code", function () { self.addClient(); }, "", this.busy.add)
            ]));
            if (this.addErr) add.push(h("div", { style: { color: "#f56c6c", marginTop: "6px" } }, this.addErr));
          }
          return this.card(h, "Devices", [
            h("div", { class: "tb-list" }, rows),
            h("div", { style: { borderTop: "1px solid #ebeef5", marginTop: "4px", paddingTop: "12px" } }, [
              h("div", { style: { fontWeight: 600 } }, "Add a phone or a router without Tailscale"),
              h("div", { class: "tb-muted tb-small" }, "Routers and phones on your tailnet just ask to connect. For others, make a code here.")
            ].concat(add))
          ]);
        },

        whoApproves: function (h) {
          var s = this.settings || {}, ts = this.ts || {};
          var page = "http://" + ((ts.dns_name || "").split(".")[0] || "this router") + ":8077";
          return h("div", { class: "tb-muted tb-small", style: { lineHeight: 1.55 } }, s.password_set
            ? "Who can approve: anyone logged in to this admin panel, or with the tailbond password at " + page + "."
            : "Who can approve: anyone logged in to this admin panel, and anyone on your Tailscale account who opens " + page + ". To require a password there, set one under Internet and Tailscale.");
        },

        // What's left before travel routers can connect from anywhere.
        setupSteps: function () {
          var ts = this.ts, s = this.settings, out = [];
          if (!ts || !s) return out;
          if (!ts.installed || !ts.running) out.push("tailscale");
          else if (!ts.routes_approved) out.push("route");
          if (s.public_addr_guessed) out.push("address");
          // a problem that stops travel routers connecting at all (e.g.
          // carrier-grade NAT) is a setup step too
          A(this.health && this.health.checks).forEach(function (c) { if (c.status === "bad" && c.id !== "tailscale") out.push("health:" + c.id); });
          return out;
        },

        rRoute: function (h) {
          var self = this, ts = this.ts || {}, s = this.settings || {}, steps = this.setupSteps(), items = [], n = 0;
          var addr = s.effective_public_addr || "";
          if (steps.indexOf("tailscale") >= 0) {
            items.push(this.alert(h, "", "Step " + (++n) + ": turn on Tailscale", "Turn on Tailscale on this router and sign in. Travel routers find this server through it, and connected devices reach your Tailscale machines through it.",
              [h("a", { class: "tb-btn small primary", attrs: { href: "#/tailscaleview" } }, "Open Tailscale settings")]));
          }
          if (steps.indexOf("route") >= 0) {
            items.push(this.alert(h, "", "Step " + (++n) + ": allow tailbond devices on your Tailscale network",
              [h("div", ["In the Tailscale admin console, open ", h("b", ts.self || "this router"), ", choose ", h("b", "Edit route settings"), ", tick the one route listed (", ts.route || "10.77.0.0/24", ") and save. That lets your Tailscale machines reach devices that connect through tailbond. Devices can already reach your machines without it."])],
              [h("a", { class: "tb-btn small primary", attrs: { href: ts.admin_url || "https://login.tailscale.com/admin/machines", target: "_blank", rel: "noopener" } }, "Open admin console")]));
          }
          if (steps.indexOf("address") >= 0) {
            items.push(this.alert(h, "", "Step " + (++n) + ": tell travel routers where home is",
              [h("div", ["Travel routers reach this router over the internet, so they need a name for your home. Easiest: turn on ", h("a", { class: "tb-link", attrs: { href: "#/ddns" } }, "APPLICATIONS → Dynamic DNS"), " and paste the name it shows here (it looks like ", h("span", { class: "tb-mono" }, "abc1234.glddns.com"), "). Right now they'd use " + (addr || "a home-network address") + ", which only works at home."]),
                this.addrEditor(h)]));
          }
          steps.filter(function (x) { return x.indexOf("health:") === 0; }).forEach(function (x) {
            var c = A(self.health.checks).filter(function (k) { return "health:" + k.id === x; })[0];
            if (c) items.push(self.alert(h, "err", "Step " + (++n) + ": " + c.title, [h("div", c.detail || ""), c.fix ? h("div", { style: { marginTop: "4px" } }, [c.fix + " ", c.fix_url ? h("a", { class: "tb-link", attrs: { href: c.fix_url, target: "_blank", rel: "noopener" } }, "How to fix") : null]) : null,
              c.id === "cgnat" ? h("div", { class: "tb-small", style: { marginTop: "4px" } }, "Devices on your home Wi-Fi can still connect; from away they can't until this is fixed.") : null,
              h("div", { class: "tb-small", style: { marginTop: "4px" } }, "Not sure? Show this to whoever set up your internet.")]));
          });
          if (!steps.length) {
            items.push(this.alert(h, "ok", "Setup done", "Travel routers and phones can connect from anywhere, and your Tailscale machines reach them. The first connection from away from home confirms the port works."));
          }
          items.push(h("div", { class: "tb-kv", style: { marginTop: "4px" } }, [
            h("div", "Internet address"),
            steps.indexOf("address") >= 0 ? h("div", { class: "tb-muted" }, "not set yet (step above)") : (this.addrEdit ? this.addrEditor(h) : h("div", { class: "tb-row" }, [h("span", { class: "tb-mono" }, addr || "not set"),
              this.btn(h, "Change", function () { self.addrEdit = true; self.addrValue = s.public_addr || ""; }, "small text")])),
            h("div", "Port"), steps.indexOf("health:cgnat") >= 0 ? h("div", "UDP " + (s.port || 443) + " is opened on this router, but your provider's shared address (step above) keeps travel routers from reaching it.") : h("div", ["UDP " + (s.port || 443) + ", opened on this router automatically. ", h("span", { class: "tb-muted" }, "Is your internet cable plugged straight into this router? Then nothing else to do. If it's plugged into another box (your internet provider's modem-router), that box must forward UDP " + (s.port || 443) + " to this router; see the box's manual under \"port forwarding\".")]),
            h("div", "Tailscale name"), h("div", { class: "tb-mono" }, ts.dns_name || "—"),
            h("div", "tailbond page password"), this.pwEdit ? h("div", { class: "tb-row" }, [
              h("input", { class: "tb-input", attrs: { type: "password", placeholder: "New password (empty: none)", autocomplete: "new-password" }, domProps: { value: this.pwValue }, on: { input: function (e) { self.pwValue = e.target.value; } } }),
              this.btn(h, "Save", function () { self.run("pw", api("POST", "/api/settings", { ui_password: self.pwValue || "" }), self.pwValue ? "Password set" : "Password removed").then(function () { self.pwEdit = false; self.pwValue = ""; }); }, "primary small", this.busy.pw),
              this.btn(h, "Cancel", function () { self.pwEdit = false; }, "small text")
            ]) : h("div", { class: "tb-row" }, [h("span", s.password_set ? "Set: needed to open http://" + ((ts.dns_name || "").split(".")[0] || "this router") + ":8077" : "None: anyone on your Tailscale account can open http://" + ((ts.dns_name || "").split(".")[0] || "this router") + ":8077"),
              this.btn(h, s.password_set ? "Change" : "Set", function () { self.pwEdit = true; }, "small text")]),
            h("div", "Secure connection"), h("div", s.tls === "tailscale" ? "Certificate from Tailscale for " + (s.tls_name || "") : "This router's own certificate, checked by each device")
          ]));
          return this.card(h, steps.length ? "Finish setting up" : "Internet and Tailscale", items);
        },

        addrEditor: function (h) {
          var self = this;
          return h("div", { class: "tb-row", style: { marginTop: "8px" } }, [
            h("input", { class: "tb-input", attrs: { placeholder: "e.g. home.example.com" }, domProps: { value: this.addrValue }, on: { input: function (e) { self.addrValue = e.target.value; } } }),
            this.btn(h, "Save", function () { self.saveAddr(); }, "primary small", this.busy.addr || !(this.addrValue || "").trim()),
            this.addrEdit ? this.btn(h, "Cancel", function () { self.addrEdit = false; }, "small text") : null
          ]);
        },

        // Setup health: green when all good, else one line per problem with
        // a "How to fix" link to Tailscale's docs. Items the setup steps
        // card already shows (Tailscale off, route) are left out here.
        rHealth: function (h) {
          var self = this, v = this.health;
          if (!v) return null;
          var steps = this.setupSteps();
          // finish the setup steps first (health problems shown as steps excepted)
          if (steps.some(function (x) { return x.indexOf("health:") !== 0; })) return null;
          var checks = A(v.checks);
          // on the Flint, "this machine" is the router itself: say where in GL
          var glFix = "On this router: NETWORK → UPnP, turn it on. If another router sits in front of this one (for example your internet provider's box), forward UDP port 41641 to this router there.";
          checks = checks.map(function (c) { return (c.id === "tailscale_direct" || c.id === "netcheck") && c.fix ? Object.assign({}, c, { fix: glFix }) : c; });
          var shown = {};
          checks = checks.map(function (c) { if (c.fix && shown[c.fix]) return Object.assign({}, c, { fix: "Same fix as above.", fix_url: "" }); if (c.fix) shown[c.fix] = true; return c; });
          var bad = checks.filter(function (c) { return c.status === "warn" || c.status === "bad"; });
          var rest = checks.filter(function (c) { return bad.indexOf(c) < 0; });
          var row = function (c) {
            var color = c.status === "ok" ? "#13a35a" : c.status === "bad" ? "#f56c6c" : c.status === "warn" ? "#c88a1a" : "#909399";
            var mark = c.status === "ok" ? "✓" : c.status === "unknown" ? "?" : "!";
            return h("div", { class: "tb-li", style: { alignItems: "flex-start" } }, [
              h("span", { style: { color: color, fontWeight: 700, width: "16px", flex: "none", textAlign: "center" } }, mark),
              h("div", { class: "tb-grow" }, [
                h("div", { style: { fontWeight: c.status === "ok" ? 400 : 600 } }, c.title),
                c.detail ? h("div", { class: "tb-muted tb-small" }, c.detail) : null,
                c.fix && c.status !== "ok" ? h("div", { class: "tb-small", style: { marginTop: "2px" } }, [c.fix + " ",
                  c.fix_url ? h("a", { class: "tb-link", attrs: { href: c.fix_url, target: "_blank", rel: "noopener" } }, "How to fix") : null]) : null
              ])
            ]);
          };
          var unknown = checks.filter(function (c) { return c.status === "unknown"; }).length;
          var head = bad.length
            ? h("div", { style: { marginBottom: "6px", color: "#8a5d10" } }, bad.length + " thing" + (bad.length === 1 ? "" : "s") + " to improve for best speed (tailbond works meanwhile, just slower):")
            : h("div", { style: { marginBottom: "6px", color: "#137a46", fontWeight: 600 } }, unknown
              ? "✓ Nothing to fix. " + unknown + " check" + (unknown === 1 ? " needs" : "s need") + " a device connecting from away (marked ?)."
              : "✓ Everything is set up for best speed");
          return this.card(h, "Setup health", [head,
            h("div", { class: "tb-list" }, bad.map(row).concat(bad.length && !this.showHealthOk ? [] : rest.map(row))),
            bad.length && rest.length ? h("a", { class: "tb-link tb-small", on: { click: function () { self.showHealthOk = !self.showHealthOk; } } }, this.showHealthOk ? "Hide the rest" : "Show " + rest.length + " more check" + (rest.length === 1 ? "" : "s")) : null,
            bad.length ? h("div", { class: "tb-small", style: { marginTop: "8px" } }, "Not sure how? Show this card to whoever set up your internet; the links explain each fix.") : null,
            h("div", { class: "tb-muted tb-small", style: { marginTop: "8px" } }, ["Direct connections are faster than relayed ones. ",
              h("a", { class: "tb-link", attrs: { href: "https://tailscale.com/docs/reference/connection-types", target: "_blank", rel: "noopener" } }, "About Tailscale connection types")])
          ], bad.length ? h("span", { class: "tb-chip warn" }, bad.length + " to improve") : h("span", { class: "tb-chip ok" }, "All good"));
        },

        // How to put tailbond on a GL travel router (the install itself
        // happens on that router, which doesn't have tailbond yet).
        rRouterGuide: function (h) {
          var s = this.settings || {};
          return this.card(h, "Add a GL.iNet travel router", [
            h("ol", { class: "tb-steps-n" }, [
              h("li", s.glinet_ipk
                ? [h("a", { class: "tb-btn small primary", attrs: { href: "/tailbond-glinet.ipk", download: "tailbond_aarch64_cortex-a53_neon-vfpv4.ipk" } }, "Download the router package"), " to this computer. It's the same tailbond this router runs."]
                : ["Get the router package on your computer: the tailbond file ending in ", h("b", "_neon-vfpv4.ipk"), "."]),
              h("li", ["On the travel router's admin page: ", h("b", "SYSTEM → Advanced Settings → Go To LuCI"), " (log in with the same password), then ", h("b", "System → Software → Upload Package…"), ", pick the file and click ", h("b", "Install"), " (leave the other options as they are). The router's internet keeps working."]),
              h("li", ["Back on the travel router's admin page, open ", h("b", "VPN → tailbond"), " and click ", h("b", "Ask to connect"), ". It shows up at the top of this page; click ", h("b", "Approve"), "."])
            ])
          ]);
        }
      }
    }]
  };
})()
