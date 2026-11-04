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
  var KIND_LABEL = { ethernet: "Ethernet", wifi: "Repeater", usb: "Tethering", cellular: "Cellular", "shared-cellular": "Shared by a phone", unknown: "Connection" };
  // Short nouns for the one-line status.
  var KIND_NOUN = { ethernet: "Ethernet", wifi: "Repeater (Wi-Fi)", usb: "USB tethering (phone)", cellular: "Cellular", "shared-cellular": "a phone's shared data", unknown: "a connection" };
  // a phone's share is named by its own label ("Pixel 8's mobile data")
  function nounOf(w) { return w.kind === "shared-cellular" ? String(w.label || "").replace(/ \(shared\)$/, "") : (KIND_NOUN[w.kind] || w.label); }
  var KIND_ORDER = { ethernet: 0, wifi: 1, usb: 2, cellular: 3, "shared-cellular": 4, unknown: 5 };
  var NOT_CONNECTED_FIX = {
    ethernet: "Plug a cable from the hotel or modem into the WAN port. It's used automatically once connected.",
    wifi: "Join a Wi-Fi network in INTERNET → Repeater. It's used automatically once connected.",
    usb: "Plug in your phone with a USB cable and turn on USB tethering on the phone. It's used automatically once connected.",
    cellular: "Insert a SIM card or plug in the modem, then check INTERNET → Cellular. It's used automatically once connected.",
    "shared-cellular": "The phone stopped sharing, left this Wi-Fi, or its battery or data limit stopped it. Turn sharing on again on the phone.",
    unknown: "Set it up on the INTERNET page. It's used automatically once connected."
  };

  var CSS = [
    /*brand-tokens:begin*/".tb{--tb-primary:#0f766e;--tb-on-primary:#ffffff;--tb-ink:#071a1d;--tb-teal:#0d9488;--tb-volt:#65a30d;--tb-beam:linear-gradient(90deg,#0d9488,#65a30d);--tb-ok:#15803d;--tb-warn:#b45309;--tb-error:#dc2626;--tb-wifi:#0284c7;--tb-cellular:#7c3aed;--tb-ethernet:#475569;--tb-usb:#ea580c;--tb-shared:#db2777;--tb-tailnet:#0d9488;}",/*brand-tokens:end*/
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
    ".tb-hero-icon{width:52px;height:52px;border-radius:12px;display:flex;align-items:center;justify-content:center;flex:none;background:color-mix(in srgb,var(--tb-primary,#0f766e) 10%,#fff);color:var(--tb-primary,#0f766e);}",
    ".tb-hero-icon.on{background:#e8f8ef;color:var(--tb-ok,#15803d);}",
    ".tb-hero-icon.warn{background:#fdf6ec;color:#e6a23c;}",
    ".tb-hero-icon.err{background:#fef0f0;color:#f56c6c;}",
    ".tb-hero-t{font-size:18px;font-weight:600;}",
    ".tb-hero-s{font-size:14px;color:#606266;margin-top:4px;line-height:1.45;}",
    ".tb-rates{display:flex;gap:24px;padding:0 20px 16px 88px;color:#606266;font-size:13px;}",
    ".tb-rates b{font-size:16px;color:#303133;font-weight:600;}",
    ".tb-switch{position:relative;width:44px;height:22px;border-radius:11px;background:#dcdfe6;cursor:pointer;flex:none;transition:background .2s;border:0;padding:0;}",
    ".tb-switch:after{content:'';position:absolute;left:2px;top:2px;width:18px;height:18px;border-radius:50%;background:#fff;transition:left .2s;}",
    ".tb-switch.on{background:var(--tb-primary,#0f766e);}",
    ".tb-switch.on:after{left:24px;}",
    ".tb-switch[disabled]{opacity:.5;cursor:not-allowed;}",
    ".tb-switch-big{width:56px;height:28px;border-radius:14px;}",
    ".tb-switch-big:after{width:24px;height:24px;}",
    ".tb-switch-big.on:after{left:30px;}",
    ".tb-btn{display:inline-flex;align-items:center;justify-content:center;gap:6px;height:34px;padding:0 16px;border-radius:4px;border:1px solid #dcdfe6;background:#fff;color:#606266;font-size:14px;cursor:pointer;white-space:nowrap;text-decoration:none;}",
    ".tb-btn:hover{color:var(--tb-primary,#0f766e);border-color:color-mix(in srgb,var(--tb-primary,#0f766e) 32%,#fff);background:color-mix(in srgb,var(--tb-primary,#0f766e) 10%,#fff);}",
    ".tb-btn.primary{background:var(--tb-primary,#0f766e);border-color:var(--tb-primary,#0f766e);color:#fff;}",
    ".tb-btn.primary:hover{background:var(--tb-teal,#0d9488);border-color:var(--tb-teal,#0d9488);}",
    ".tb-btn.danger{color:#f56c6c;border-color:#fbc4c4;background:#fef0f0;}",
    ".tb-btn.text{border:0;background:none;color:var(--tb-primary,#0f766e);padding:0 4px;height:auto;}",
    ".tb-btn.small{height:28px;padding:0 12px;font-size:13px;}",
    ".tb-btn[disabled]{opacity:.55;cursor:not-allowed;}",
    ".tb-input{height:34px;border:1px solid #dcdfe6;border-radius:4px;padding:0 12px;font-size:14px;color:#303133;width:100%;outline:none;}",
    ".tb-input:focus{border-color:var(--tb-primary,#0f766e);}",
    "textarea.tb-input{height:72px;padding:8px 12px;resize:vertical;font-family:inherit;}",
    ".tb-chip{display:inline-flex;align-items:center;gap:6px;height:22px;padding:0 8px;border-radius:11px;font-size:12px;white-space:nowrap;background:#f4f4f5;color:#909399;}",
    ".tb-chip:before{content:'';width:6px;height:6px;border-radius:50%;background:currentColor;}",
    ".tb-chip.ok{background:#e8f8ef;color:var(--tb-ok,#15803d);}",
    ".tb-chip.warn{background:#fdf6ec;color:#c88a1a;}",
    ".tb-chip.err{background:#fef0f0;color:#f56c6c;}",
    ".tb-chip.blue{background:color-mix(in srgb,var(--tb-primary,#0f766e) 10%,#fff);color:var(--tb-primary,#0f766e);}",
    ".tb-grid{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:16px;}",
    ".tb-wan{border:1px solid #ebeef5;border-radius:8px;padding:16px;display:flex;flex-direction:column;gap:12px;}",
    ".tb-wan.dim{background:#fafafa;}",
    ".tb-wan-icon{width:40px;height:40px;border-radius:8px;background:color-mix(in srgb,var(--tb-primary,#0f766e) 10%,#fff);color:var(--tb-primary,#0f766e);display:flex;align-items:center;justify-content:center;flex:none;}",
    ".tb-wan.dim .tb-wan-icon{background:#f4f4f5;color:#909399;}",
    ".tb-wan-t{font-weight:600;font-size:15px;}",
    ".tb-wan-ctl{display:flex;align-items:center;justify-content:space-between;border-top:1px dashed #ebeef5;padding-top:10px;gap:8px;}",
    ".tb-check{display:inline-flex;align-items:center;gap:8px;cursor:pointer;color:#606266;user-select:none;}",
    ".tb-check i{width:16px;height:16px;border:1px solid #dcdfe6;border-radius:3px;display:inline-flex;align-items:center;justify-content:center;background:#fff;}",
    ".tb-check.on i{background:var(--tb-primary,#0f766e);border-color:var(--tb-primary,#0f766e);}",
    ".tb-check.on i:after{content:'';width:4px;height:8px;border:solid #fff;border-width:0 2px 2px 0;transform:rotate(45deg) translate(-1px,-1px);}",
    ".tb-bar{height:6px;border-radius:3px;background:#ebeef5;overflow:hidden;margin-top:6px;}",
    ".tb-bar i{display:block;height:100%;background:var(--tb-beam,#0f766e);}",
    /* one colour per connection kind (docs/brand.md) */
    ".tb-wan-icon.k-wifi{--k:var(--tb-wifi)}.tb-wan-icon.k-cellular{--k:var(--tb-cellular)}.tb-wan-icon.k-ethernet{--k:var(--tb-ethernet)}",
    ".tb-wan-icon.k-usb{--k:var(--tb-usb)}.tb-wan-icon.k-shared-cellular{--k:var(--tb-shared)}.tb-wan-icon.k-tailnet{--k:var(--tb-tailnet)}",
    ".tb-wan-icon[class*=' k-']{color:var(--k);background:color-mix(in srgb,var(--k) 12%,#fff);}",
    ".tb-wan.dim .tb-wan-icon[class*=' k-']{color:#909399;background:#f4f4f5;}",
    ".tb-bar i.full{background:#f56c6c;}",
    ".tb-modes{display:grid;grid-template-columns:repeat(4,minmax(0,1fr));gap:12px;}",
    ".tb-modes.three{grid-template-columns:repeat(3,minmax(0,1fr));}",
    ".tb-mode{border:1px solid #dcdfe6;border-radius:8px;padding:12px 14px;cursor:pointer;background:#fff;text-align:left;font:inherit;color:inherit;}",
    ".tb-mode:hover{border-color:var(--tb-primary,#0f766e);}",
    ".tb-mode.on{border-color:var(--tb-primary,#0f766e);background:color-mix(in srgb,var(--tb-primary,#0f766e) 10%,#fff);box-shadow:inset 0 0 0 1px var(--tb-primary,#0f766e);}",
    ".tb-mode-t{font-weight:600;display:flex;align-items:center;gap:8px;}",
    ".tb-mode-t:before{content:'';width:14px;height:14px;border-radius:50%;border:1px solid #dcdfe6;background:#fff;flex:none;}",
    ".tb-mode.on .tb-mode-t:before{border:4px solid var(--tb-primary,#0f766e);}",
    ".tb-mode-b{font-size:12px;color:#909399;margin-top:6px;line-height:1.45;}",
    ".tb-list{border-top:1px solid #ebeef5;}",
    ".tb-li{display:flex;align-items:center;gap:12px;padding:12px 0;border-bottom:1px solid #ebeef5;}",
    ".tb-li:last-child{border-bottom:0;}",
    ".tb-dot{width:8px;height:8px;border-radius:50%;background:#c0c4cc;flex:none;}",
    ".tb-dot.ok{background:var(--tb-ok,#15803d);}",
    ".tb-mono{font-family:Menlo,Consolas,monospace;font-size:12px;color:#606266;}",
    ".tb-alert{display:flex;gap:12px;padding:14px 16px;border-radius:8px;background:#fdf6ec;color:#8a5d10;margin-bottom:16px;line-height:1.5;}",
    ".tb-alert.err{background:#fef0f0;color:#c45656;}",
    ".tb-alert.info{background:color-mix(in srgb,var(--tb-primary,#0f766e) 10%,#fff);color:#33689f;}",
    ".tb-alert.ok{background:#e8f8ef;color:#137a46;}",
    ".tb-alert b{color:inherit;}",
    ".tb-alert-t{font-weight:600;margin-bottom:2px;}",
    ".tb-actions{display:flex;gap:8px;flex-wrap:wrap;margin-top:10px;}",
    ".tb-req{border:1px solid #f5dab1;background:#fffaf2;border-radius:8px;padding:16px;display:flex;gap:16px;align-items:center;margin-bottom:12px;}",
    ".tb-found{border:1px solid #b3d8ff;background:#f5faff;border-radius:8px;padding:16px;display:flex;gap:16px;align-items:center;margin-bottom:12px;}",
    ".tb-spin{width:16px;height:16px;border:2px solid color-mix(in srgb,var(--tb-primary,#0f766e) 32%,#fff);border-top-color:var(--tb-primary,#0f766e);border-radius:50%;animation:tbspin 1s linear infinite;flex:none;}",
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
    ".tb-link{color:var(--tb-primary,#0f766e);cursor:pointer;text-decoration:none;}",
    ".tb-seg{display:inline-flex;border:1px solid #dcdfe6;border-radius:4px;overflow:hidden;}",
    ".tb-seg button{border:0;border-left:1px solid #dcdfe6;background:#fff;color:#606266;font:inherit;font-size:13px;height:30px;padding:0 12px;cursor:pointer;white-space:nowrap;}",
    ".tb-seg button:first-child{border-left:0;}",
    ".tb-seg button.on{background:var(--tb-primary,#0f766e);color:#fff;}",
    ".tb-seg button.on.off{background:#909399;}",
    ".tb-seg button[disabled]{cursor:not-allowed;opacity:.6;}",
    ".tb-nowrap{white-space:nowrap;}",
    ".tb-steps-n{margin:6px 0 0;padding-left:22px;line-height:1.7;color:#606266;}",
    ".tb-qr{width:150px;height:150px;flex:none;background:#fff;border:1px solid #ebeef5;border-radius:8px;padding:4px;box-sizing:border-box;}.tb-qr svg{width:100%;height:100%;display:block;}",
    ".tb-foot{text-align:center;color:#c0c4cc;font-size:12px;margin:8px 0 24px;}",
    // servers and failover
    ".tb-srv{display:flex;align-items:center;gap:12px;padding:12px 0;border-bottom:1px solid #ebeef5;flex-wrap:wrap;}",
    ".tb-srv:last-child{border-bottom:0;}",
    ".tb-srv-ctl{display:flex;align-items:center;gap:6px;flex-wrap:wrap;}",
    ".tb-btn.icon{width:28px;padding:0;}",
    ".tb-role{font-size:12px;color:#909399;font-weight:400;margin-left:8px;}",
    ".tb-role.first{color:var(--tb-primary,#0f766e);}",
    ".tb-sec{border-top:1px solid #ebeef5;margin-top:14px;padding-top:14px;}",
    ".tb-sec-t{font-weight:600;margin-bottom:8px;display:flex;align-items:center;justify-content:space-between;gap:8px;}",
    ".tb-label{font-size:13px;color:#606266;margin:14px 0 6px;font-weight:600;}",
    // bypass rules
    ".tb-rule{display:flex;align-items:center;gap:12px;padding:12px 0;border-bottom:1px solid #ebeef5;}",
    ".tb-rule:last-child{border-bottom:0;}",
    ".tb-rule.off .tb-wan-icon,.tb-rule.off .tb-grow{opacity:.5;}",
    ".tb-tgt{font-weight:600;}",
    ".tb-tgt.direct{color:var(--tb-ok,#15803d);}",
    ".tb-tgt.wan{color:#b07a12;}",
    ".tb-tgt.tunnel{color:var(--tb-primary,#0f766e);}",
    ".tb-presets{display:flex;gap:8px;flex-wrap:wrap;align-items:center;}",
    ".tb-preset{display:inline-flex;align-items:center;gap:6px;border:1px dashed #b3d8ff;background:#f5faff;color:var(--tb-primary,#0f766e);border-radius:14px;height:28px;padding:0 12px;font:inherit;font-size:13px;cursor:pointer;white-space:nowrap;}",
    ".tb-preset:hover,.tb-preset.on{border-style:solid;background:color-mix(in srgb,var(--tb-primary,#0f766e) 10%,#fff);}",
    ".tb-pick{max-height:232px;overflow:auto;border:1px solid #ebeef5;border-radius:6px;}",
    ".tb-pick-i{display:flex;align-items:center;gap:10px;padding:8px 12px;cursor:pointer;border-bottom:1px solid #f2f3f5;}",
    ".tb-pick-i:last-child{border-bottom:0;}",
    ".tb-pick-i:hover{background:#f5f7fa;}",
    ".tb-pick-i.on{background:color-mix(in srgb,var(--tb-primary,#0f766e) 10%,#fff);box-shadow:inset 3px 0 0 var(--tb-primary,#0f766e);}",
    ".tb-dialog.wide{max-width:560px;max-height:calc(100vh - 32px);overflow:auto;}",
    "select.tb-input{background:#fff;padding:0 8px;}",
    ".tb-help{font-size:12px;color:#909399;margin-top:6px;line-height:1.5;}",
    ".tb code{font-family:Menlo,Consolas,monospace;font-size:12px;background:rgba(0,0,0,.06);padding:1px 4px;border-radius:3px;}",
    ".tb-empty{display:flex;gap:16px;align-items:flex-start;}",
    // updates and backups
    ".tb-chg{grid-template-columns:minmax(120px,40%) 1fr;border-top:1px solid #ebeef5;padding-top:8px;}",
    ".tb-chg>div{word-break:break-word;}",
    "input.tb-file{height:auto;padding:6px 8px;}",
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
    " .tb-srv-ctl{width:100%;padding-left:52px;}",
    " .tb-empty .tb-wan-icon{display:none;}",
    " .tb-dialog{padding:16px;}",
    "}"
  ].join("\n");

  // Icons (24px, stroke), Element/GL-like.
  var ICONS = {
    // the Braid mark (assets/brand/tailbond-mark-mono.svg) on a 24 grid
    bond: "M4.5 6.75C9.75 6.75 10.5 12 14.25 12M4.5 17.25C9.75 17.25 10.5 12 14.25 12M4.5 12H18.75M16.9 8.6L20.25 12l-3.35 3.4M4.5 6.75h0M4.5 12h0M4.5 17.25h0",
    ethernet: "M4 6h16v9H4zM8 15v3h8v-3M8 9v2M11 9v2M14 9v2M17 9v2",
    wifi: "M2 9a15 15 0 0 1 20 0M5 12.5a10 10 0 0 1 14 0M8.5 16a5 5 0 0 1 7 0M12 19.5h.01",
    usb: "M8 3h8v6H8zM10 5v2M14 5v2M9 9v6a3 3 0 0 0 6 0V9M12 18v3",
    cellular: "M4 20v-3M9 20v-7M14 20v-11M19 20V4",
    "shared-cellular": "M8 3h8a1 1 0 0 1 1 1v16a1 1 0 0 1-1 1H8a1 1 0 0 1-1-1V4a1 1 0 0 1 1-1zM10 17v-2M12 17v-4M14 17v-6",
    unknown: "M12 3a9 9 0 1 0 0 18 9 9 0 0 0 0-18zM12 8v5M12 16h.01",
    server: "M4 4h16v6H4zM4 14h16v6H4zM8 7h.01M8 17h.01",
    phone: "M8 2h8a1 1 0 0 1 1 1v18a1 1 0 0 1-1 1H8a1 1 0 0 1-1-1V3a1 1 0 0 1 1-1zM11 18h2",
    router: "M3 13h18v6H3zM7 16h.01M11 16h.01M8 13l-2-6M16 13l2-6",
    check: "M5 12l5 5L20 7",
    warn: "M12 3l10 18H2zM12 10v5M12 18h.01",
    off: "M12 3v9M6.3 6.3a8 8 0 1 0 11.4 0",
    globe: "M12 3a9 9 0 1 0 0 18 9 9 0 0 0 0-18zM3 12h18M12 3c2.4 2.6 3.6 5.6 3.6 9s-1.2 6.4-3.6 9M12 3c-2.4 2.6-3.6 5.6-3.6 9s1.2 6.4 3.6 9",
    device: "M5 5h14v10H5zM2 19h20",
    pin: "M12 21s-7-6.2-7-11a7 7 0 0 1 14 0c0 4.8-7 11-7 11zM12 10h.01",
    port: "M9 3v5M15 3v5M6 8h12v3a6 6 0 0 1-12 0zM12 17v4",
    call: "M5 4h4l2 5-2.5 1.5a11 11 0 0 0 5 5L15 13l5 2v4a2 2 0 0 1-2 2A16 16 0 0 1 3 6a2 2 0 0 1 2-2z",
    up: "M12 19V5M6 11l6-6 6 6",
    down: "M12 5v14M6 13l6 6 6-6",
    trash: "M4 7h16M10 11v6M14 11v6M6 7l1 13h10l1-13M9 7V4h6v3",
    skip: "M3 17h5l8-10h5M17 3l4 4-4 4M3 7h5l2.5 3M14 14l2 3h5M17 13l4 4-4 4"
  };

  // "Bypass tailbond" presets (kept short; the dialog lets you edit them).
  var STREAMING = ["netflix.com", "nflxvideo.net", "disneyplus.com", "primevideo.com", "max.com"];
  var BP_KIND_ICON = { domain: "globe", device: "device", ip: "pin", port: "port" };

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
    var d = Math.round(s / 86400);
    return d + (d === 1 ? " day ago" : " days ago");
  }
  // "answered 5 s ago" (server checks run every 10 s, so seconds matter)
  function fmtAgoS(t) {
    var s = Math.max(0, Math.round((Date.now() - new Date(t).getTime()) / 1000));
    if (s < 60) return s + " s ago";
    return fmtAgo(t);
  }
  function fmtClock(sec) {
    sec = Math.max(0, Math.round(sec || 0));
    return Math.floor(sec / 60) + ":" + ("0" + (sec % 60)).slice(-2);
  }
  // "14:02" on this computer's clock
  function fmtHM(t) {
    var d = new Date(t);
    return isNaN(d) ? "" : ("0" + d.getHours()).slice(-2) + ":" + ("0" + d.getMinutes()).slice(-2);
  }
  function fmtDur(sec) {
    if (sec < 60) return sec + " s";
    var m = Math.round(sec / 60);
    return m + " min";
  }
  function words(s) { return (s || "").split(/[\s,;]+/).filter(Boolean); }
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

  // ---- updates and backups ----
  // JSON calls go through api() like everything else. Binary ones (the
  // backup file) and the restore calls (bodies up to
  // 2 MiB, more than /rpc should carry) use the same-origin proxy
  // /tailbond-api/ directly; it checks the same GL admin session.
  function proxy(method, path, body, type) {
    var opt = { method: method, credentials: "same-origin", headers: {} };
    if (body !== undefined) {
      opt.headers["Content-Type"] = type || "application/json";
      opt.body = type ? body : JSON.stringify(body);
    }
    return fetch("/tailbond-api/" + path.replace(/^\/api\//, ""), opt).then(function (r) {
      if (r.status === 502 || r.status === 504) throw { code: 0, error: "agent_down" };
      if (r.status === 401) throw { code: 401, error: "session" };
      if (!r.ok) {
        return r.text().then(function (t) {
          var d = {};
          try { d = JSON.parse(t); } catch (e) { /* not JSON */ }
          throw { code: r.status, error: d.error || (r.status === 413 ? "file too large" : r.statusText || "failed"), data: d };
        });
      }
      return r;
    });
  }
  function proxyJSON(method, path, body) {
    return proxy(method, path, body).then(function (r) { return r.json().catch(function () { return {}; }); });
  }
  function saveBlob(blob, name) {
    var a = document.createElement("a");
    a.href = URL.createObjectURL(blob);
    a.download = name;
    document.body.appendChild(a); a.click(); document.body.removeChild(a);
    setTimeout(function () { URL.revokeObjectURL(a.href); }, 10000);
  }
  function fileB64(file) {
    return new Promise(function (ok, no) {
      var r = new FileReader();
      r.onload = function () { ok(String(r.result).replace(/^data:[^,]*,/, "")); };
      r.onerror = function () { no({ error: "Couldn't read that file" }); };
      r.readAsDataURL(file);
    });
  }
  function fmtWhen(t) {
    var d = new Date(t);
    if (!t || isNaN(d)) return "";
    var mon = ["Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"][d.getMonth()];
    return mon + " " + d.getDate() + ", " + fmtHM(d);
  }
  // a rough strength hint; the rule is 10 characters
  function strength(p) {
    if (!p) return ["", ""];
    if (p.length < 10) return ["weak", "Too short: " + (10 - p.length) + " more character" + (p.length === 9 ? "" : "s")];
    var kinds = [/[a-z]/, /[A-Z]/, /\d/, /[^A-Za-z0-9]/].filter(function (r) { return r.test(p); }).length;
    var nw = p.trim().split(/[\s\-_.]+/).length;
    if (/^(.)\1+$/.test(p) || /^(?:0123456789|1234567890|password|qwerty)/i.test(p)) return ["weak", "Easy to guess"];
    if (p.length >= 20 || nw >= 4 || (p.length >= 14 && kinds >= 3)) return ["strong", "Strong"];
    return ["ok", "OK. Longer is better: four random words are strong and easy to type"];
  }
  // semver order of "0.2.0" / "v0.2.1-beta.1": <0, 0, >0 (a pre-release sorts before its release)
  function verCmp(a, b) {
    var p = function (v) { return String(v || "").replace(/^v/, "").split("-"); };
    var x = p(a), y = p(b), xs = x[0].split(".").map(Number), ys = y[0].split(".").map(Number);
    for (var i = 0; i < 3; i++) if ((xs[i] || 0) !== (ys[i] || 0)) return (xs[i] || 0) - (ys[i] || 0);
    return x[1] && !y[1] ? -1 : !x[1] && y[1] ? 1 : 0;
  }
  var KIND_NAME = { router: "travel router", android: "Android phone", phone: "Android phone" };
  var ROLE_WORDS = { server: "home server", client: "travel router" };
  // GET /api/backup/summary keys, in reading order
  var SUMMARY_LABEL = { role: "Role", mode: "Mode", clients: "Devices", servers: "Servers", public_addr: "Public address", port: "Tunnel port", subnet: "Tunnel addresses",
    approvers: "Allowed to approve", bypass_rules: "Rules that skip tailbond", paths: "Connection settings", password_set: "Page password (tailbond's own)", update_policy: "Updates",
    files: "Key and certificate files", client_backups: "Your devices' backups", uci: "Router settings (UCI)" };
  // <state_dir>/backups on GL routers (docs/backup-restore.md)
  var BACKUP_DIR = "/etc/tailbond/state/backups/";

  var UB = {
    methods: {
      // the live update object (status) over GET /api/update
      upd: function () { return Object.assign({}, this.updX || {}, (this.st && this.st.update) || {}); },
      loadUb: function () {
        var self = this, jobs = [api("GET", "/api/update").then(function (u) { self.updX = u; }).catch(function () {})];
        if (this.role === "server") jobs.push(api("GET", "/api/backups").then(function (b) { self.bks = Array.isArray(b) ? b : A(b && b.backups); }).catch(function () { self.bks = null; }));
        return Promise.all(jobs);
      },
      updPost: function (path, body, msg) {
        var self = this;
        return this.run("upd", api("POST", path, body || {}), msg).then(function () { return self.loadUb(); }).catch(function () {});
      },
      confirmInstall: function (u) {
        var self = this;
        this.confirm = {
          title: "Update to " + u.available + " now?",
          body: "tailbond restarts; devices drop for a few seconds. If the new version doesn't start, it goes back by itself.",
          note: "Downloaded from the GitHub release and checked against its SHA-256 list before anything is installed.",
          ok: "Update now",
          run: function () { self.restartUntil = Date.now() + 180000; return self.updPost("/api/update/install", {}, "Installing " + u.available + "…"); }
        };
      },
      // the one-line state of updates (chip, top card)
      updState: function (u) {
        var off = u.policy === "off" || u.state === "off";
        var busy = ["checking", "downloading", "installing"].indexOf(u.state) >= 0;
        return {
          off: off, busy: busy,
          retry: u.state === "rolled_back" && !!u.available && u.available === u.last_result_version,
          noCheck: !!u.error && !u.available && !busy && !off && u.state !== "failed" && u.state !== "rolled_back",
          reinstalled: u.detail === "reinstalled_after_firmware_upgrade"
        };
      },
      toUpdates: function () { var c = document.querySelector(".tb-upd"); if (c) c.scrollIntoView({ behavior: "smooth", block: "start" }); },
      // update news inside the top status card
      updTop: function (h) {
        var st = this.st || {};
        if (!st.update && !this.updX) return null;
        var u = this.upd(), s = this.updState(u), cur = u.current || st.version || "";
        var msg = null, ico = "up", color = "#303133";
        if (u.state === "installing") msg = "Installing tailbond " + u.available + "; this page comes back by itself in about a minute.";
        else if (u.state === "failed") { msg = "An update to " + (u.last_result_version || u.available || "a new version") + " didn't install. Still on " + cur + ", nothing changed."; color = "#c45656"; ico = "warn"; }
        else if (u.state === "rolled_back") { msg = "Everything works as before on " + cur + ": " + (u.last_result_version || "the new version") + " didn't start here, so tailbond went back by itself."; ico = "check"; }
        else if (s.reinstalled) { msg = "tailbond is back after the firmware update, with your settings."; ico = "check"; color = "#137a46"; }
        else if (u.available && !s.off) msg = "Update available: tailbond " + u.available + " (this router runs " + cur + ").";
        if (!msg) return null;
        return h("div", { class: "tb-row", style: { borderTop: "1px solid #ebeef5", padding: "12px 20px 14px 88px", gap: "10px", color: color, fontSize: "14px" } }, [
          icon(h, ico, 18), h("div", { class: "tb-grow" }, msg), h("a", { class: "tb-link", on: { click: this.toUpdates } }, "Show")
        ]);
      },
      bset: function () { return (this.settings && this.settings.backup) || (this.st && this.st.backup) || {}; },
      openUb: function (kind) {
        this.ubFiles = [];
        this.ub = { kind: kind, pass: "", pass2: "", err: "", busy: false, step: "pick", file: "", preview: null, summary: null, rollback: "" };
        if (kind === "summary") {
          var self = this;
          api("GET", "/api/backup/summary").then(function (s) { if (self.ub) self.ub.summary = s; }).catch(function (e) { if (self.ub) self.ub.err = self.humanError(e); });
        }
      },
      passCheck: function (u) {
        if (u.pass.length < 10) return "The passphrase needs at least 10 characters.";
        if (u.pass !== u.pass2) return "The two passphrases don't match.";
        return "";
      },
      doBackup: function () {
        var self = this, u = this.ub;
        u.err = this.passCheck(u);
        if (u.err) return;
        u.busy = true;
        var host = (this.settings && this.settings.paired_as) || (this.st.tailscale && this.st.tailscale.self) || "tailbond";
        return proxy("POST", "/api/backup", { passphrase: u.pass }).then(function (r) {
          var m = /filename="?([^";]+)"?/.exec(r.headers.get("Content-Disposition") || "");
          return r.blob().then(function (b) { saveBlob(b, m ? m[1] : host + "-" + new Date().toISOString().slice(0, 10) + ".tbbackup"); });
        }).then(function () { self.ub = null; self.say("Backup downloaded. Keep the passphrase somewhere safe."); },
          function (e) { u.busy = false; u.err = self.humanError(e); });
      },
      doPreview: function () {
        var self = this, u = this.ub, f = this.ubFiles[0];
        u.err = "";
        if (!f) { u.err = "Pick the backup file first (it ends in .tbbackup)."; return; }
        if (!u.pass) { u.err = "Type the passphrase of that backup."; return; }
        u.busy = true;
        return fileB64(f).then(function (b64) {
          self.ubBody = { file: b64, passphrase: u.pass };
          return proxyJSON("POST", "/api/restore/preview", self.ubBody);
        }).then(function (p) { u.busy = false; u.preview = p; u.step = "preview"; }, function (e) {
          u.busy = false;
          var msg = (e && e.error) || "";
          u.err = /passphrase|damaged|gcm|auth/i.test(msg) ? "Wrong passphrase, or the file is damaged. Check the passphrase (it's case-sensitive) and try again."
            : /role/i.test(msg) ? "That backup is from a " + (self.role === "server" ? "travel router" : "home server") + "; it can only be restored on one." : self.humanError(e);
        });
      },
      doRestore: function () {
        var self = this, u = this.ub;
        u.busy = true;
        this.restartUntil = Date.now() + 120000;
        return proxyJSON("POST", "/api/restore", Object.assign({}, this.ubBody, { confirm: true })).then(function (r) {
          u.busy = false; u.step = "done"; u.rollback = (r && r.rollback) || ""; self.ubBody = null;
        }, function (e) { u.busy = false; u.err = self.humanError(e); });
      },
      saveSched: function (body, msg) {
        var self = this;
        return this.runE("sched", api("POST", "/api/backup/settings", body), msg).then(function () { self.ubSched = null; }, function (e) { if (self.ubSched) self.ubSched.err = self.humanError(e); });
      },

      // ---- render ----
      rUpdates: function (h) {
        var self = this, st = this.st, u = this.upd(), srv = this.role === "server";
        if (!st.update && !this.updX) return null; // an agent without updates
        var cur = u.current || (this.info && this.info.version) || st.version || "";
        var policy = u.policy || "notify", S = this.updState(u), off = S.off;
        var selfUp = u.method === "opkg" || u.method === "static";
        var busyState = S.busy;
        var win = (u.auto_window || "03:00-05:00").split("-");
        var chip = u.state === "rolled_back" ? ["warn", "Went back to " + (cur || "the previous version")]
          : u.state === "failed" ? ["err", "Update failed"]
          : u.state === "installing" ? ["blue", "Installing"]
          : u.state === "checking" ? ["blue", "Checking"]
          : u.state === "downloading" ? ["blue", "Downloading"]
          : off ? ["", "Updates off"]
          : u.available ? ["warn", "Update available"]
          : S.noCheck ? ["warn", "Couldn't check"] : ["ok", "Up to date"];
        var notes = u.notes_url ? h("a", { class: "tb-link", attrs: { href: u.notes_url, target: "_blank", rel: "noopener" } }, "What's new") : null;
        var body = [h("div", { class: "tb-muted tb-small", style: { marginBottom: "10px" } }, "This router runs tailbond " + cur)];
        var tried = u.last_result_version || u.available || "The new version";
        if (S.reinstalled) {
          body.push(this.alert(h, "ok", "tailbond is back after the firmware update, with your settings",
            "The GL firmware update removed it; it reinstalled itself from the copy kept on this router (or, if that was gone, from the GitHub release)" + (u.last_result_at ? ", " + fmtWhen(u.last_result_at) : "") + ". Nothing to do. If you upgrade the firmware without Keep Settings: install tailbond again, then restore the router's backup (" + (this.serverShort || "your home server") + " keeps the last 3: Backup & restore there), or pair again."));
        }
        if (u.state === "rolled_back") {
          body.push(this.alert(h, "info", "Everything works as before on " + cur,
            tried + " didn't start here, so tailbond went back by itself" + (u.detail && !S.reinstalled ? " (" + u.detail + ")" : "") + ". Automatic won't install " + tried + " again; the next version installs as usual.", null, "check"));
        } else if (u.state === "failed") {
          body.push(this.alert(h, "err", "Couldn't update to " + tried, (u.error ? u.error + ". " : "") + "Still on " + cur + ", nothing changed."));
        }
        var big = function (kids) { return h("div", { class: "tb-row", style: { fontSize: "17px", fontWeight: 600, gap: "10px", flexWrap: "wrap" } }, kids); };
        var line = function (kids, extra) { return h("div", { class: "tb-small", style: Object.assign({ marginTop: "4px", lineHeight: 1.55 }, extra || {}) }, kids); };
        var cmdRow = function (cmd) { return h("div", { class: "tb-row", style: { marginTop: "6px" } }, [h("code", { style: { padding: "6px 8px" } }, cmd), self.btn(h, "Copy", function () { self.copy(cmd); }, "small")]); };
        if (u.state === "checking") body.push(big([h("div", { class: "tb-spin" }), "Checking for updates…"]));
        else if (u.state === "downloading") body.push(big([h("div", { class: "tb-spin" }), "Downloading " + u.available + "…"]));
        else if (u.state === "installing") body.push(big([h("div", { class: "tb-spin" }), "Installing " + u.available]), h("div", { class: "tb-muted", style: { marginTop: "4px" } }, "tailbond restarts. Leave the router on; it comes back by itself, and this page with it, in about a minute."));
        else if (off) body.push(big(["Updates are off"]), line("tailbond doesn't check for new versions. Choose Tell me below to hear about them.", { color: "#909399" }));
        else if (u.available && !S.retry) {
          body.push(big(["Update available: " + u.available, notes]));
          if (u.method === "dnf") body.push(line("Updates for this server come from dnf. In a terminal on this server, run:"), cmdRow("sudo dnf upgrade tailbond-server"));
          else if (selfUp) body.push(line("You can update now or later; everything keeps working meanwhile.", { color: "#909399" }));
          else {
            var repo = String(u.repo || "kaovilai/tailbond").replace(/^https?:\/\/github\.com\//, "").replace(/\/+$/, "");
            body.push(line(["This build can't update itself; install the new release from ", h("a", { class: "tb-link", attrs: { href: "https://github.com/" + repo + "/releases", target: "_blank", rel: "noopener" } }, "the releases page"), "."]));
          }
        } else if (S.noCheck) {
          body.push(big(["Couldn't check for updates"]), line("Couldn't reach GitHub (" + u.error + "). tailbond tries again tomorrow; Check now tries again now.", { color: "#b07a12" }));
        } else if (!S.retry) body.push(big(["Up to date"]));
        var meta = [], lv = u.last_result_version, la = u.last_result_at ? ", " + fmtWhen(u.last_result_at) : "";
        if (u.last_check && !off) meta.push("Checked " + fmtAgo(u.last_check));
        if (!S.reinstalled && lv) {
          if (u.last_result === "ok") meta.push("last update: " + lv + la + ", worked");
          else if (u.last_result === "rolled_back" && u.state !== "rolled_back") meta.push("last update: " + lv + la + " didn't start, went back");
          else if (u.last_result === "failed" && u.state !== "failed") meta.push("last update to " + lv + la + " failed, nothing changed");
        }
        var checkErr = u.error && !S.noCheck && u.state !== "failed" && !busyState && !off;
        if (meta.length || checkErr) body.push(h("div", { class: "tb-muted tb-small", style: { marginTop: "6px" } }, [meta.join(" · ").replace(/^./, function (c) { return c.toUpperCase(); }),
          checkErr ? h("span", { style: { color: "#b07a12" } }, (meta.length ? " · " : "") + "Couldn't reach GitHub (" + u.error + ")") : null]));
        var acts = [];
        if (u.available && selfUp && !busyState && !off) acts.push(S.retry ? this.btn(h, "Try " + u.available + " again", function () { self.confirmInstall(u); }, "", this.busy.upd)
          : this.btn(h, "Update now", function () { self.confirmInstall(u); }, "primary", this.busy.upd));
        if (!off) acts.push(this.btn(h, "Check now", function () { self.updPost("/api/update/check", {}, "Checking for updates…"); }, "", busyState || this.busy.upd));
        if (acts.length) body.push(h("div", { class: "tb-actions" }, acts));

        if (srv) {
          // out of date = behind the newest release known (what's available, else this router's version)
          var latest = u.available || cur, isPhone = function (c) { return c.kind === "android" || c.kind === "phone"; };
          var behind = function (c) { return !!(c.version && latest && verCmp(c.version, latest) < 0); };
          // "up to date" only after a check that worked
          var checked = !!u.last_check && !u.error && !off;
          var devs = A(st.clients).filter(function (c) { return c.registered !== false; });
          var outs = devs.filter(behind);
          var sec = [h("div", { class: "tb-sec-t" }, devs.length && outs.length ? "Your devices: " + outs.length + " of " + devs.length + " out of date" : "Your devices")];
          if (devs.length) {
            sec.push(h("div", { class: "tb-list" }, devs.map(function (c) {
              return h("div", { class: "tb-li", style: { padding: "8px 0" } }, [
                h("div", { class: "tb-wan-icon", style: { width: "32px", height: "32px" } }, [icon(h, isPhone(c) ? "phone" : "router", 18)]),
                h("div", { class: "tb-grow" }, [h("b", c.name), h("span", { class: "tb-muted" }, " · " + (KIND_NAME[c.kind] || "device") + " · " + (c.version ? c.version + (behind(c) ? " · behind the latest (" + latest + ")" : "") : "version not known yet"))]),
                behind(c) ? h("span", { class: "tb-chip warn" }, "Out of date") : c.version && latest && checked ? h("span", { class: "tb-small", style: { color: "var(--tb-ok,#15803d)" } }, "✓ up to date") : null
              ]);
            })));
            var how = [];
            if (outs.some(function (c) { return !isPhone(c); })) how.push("Routers: Update now in its tailbond panel, or Automatic.");
            if (outs.some(isPhone)) how.push("Phones: the app shows it; with Obtainium it installs by itself.");
            if (devs.some(function (c) { return !c.version; })) how.push("A version shows once the device has connected with this version of tailbond.");
            if (!checked && devs.some(function (c) { return c.version && !behind(c); })) how.push(off ? "Updates are off, so newer versions aren't checked." : "Couldn't check for newer versions.");
            if (how.length) sec.push(h("div", { class: "tb-muted tb-small", style: { marginTop: "6px" } }, how.join(" ")));
          } else sec.push(h("div", { class: "tb-muted tb-small" }, "No devices yet."));
          body.push(h("div", { class: "tb-sec" }, sec));
        }

        var segOf = function (opts, val, label) {
          return h("div", { class: "tb-seg", attrs: { "aria-label": label } }, opts.map(function (o) {
            return h("button", { class: o[0] === val ? "on" : "", attrs: { type: "button", disabled: !!self.busy.upd }, on: { click: function () { if (o[0] !== val) self.updPost("/api/update/policy", { policy: o[0] }, "Saved"); } } }, o[1]);
          }));
        };
        var set = [h("div", { class: "tb-sec-t" }, "When a new version is out")];
        if (u.method === "dnf") {
          // dnf installs; tailbond only checks, so the choice is just On / Off
          var da = u.dnf_automatic, auto = h("b", "Automatic updates: ");
          set.push(h("div", { class: "tb-row", style: { gap: "10px" } }, ["Check for new versions", segOf([["notify", "On"], ["off", "Off"]], policy === "off" ? "off" : "notify", "Check for new versions")]),
            h("div", { class: "tb-help" }, policy === "off" ? "Doesn't check; dnf still updates tailbond with the rest of the computer." : "Checks GitHub once a day and shows new versions here."));
          if (da === "install") set.push(line([auto, "dnf-automatic installs updates for every package on this computer, tailbond included."]));
          else if (da === "download") set.push(line([auto, "dnf-automatic only downloads updates (Fedora's default). To install them too: set ", h("code", "apply_updates = yes"), " in ", h("code", "/etc/dnf/automatic.conf"), "."]));
          else if (da === "off") set.push(line([auto, "off. To turn on dnf-automatic (it updates every package on this computer), run in a terminal on this server:"]), cmdRow("sudo systemctl enable --now dnf-automatic.timer"));
          else set.push(line([auto, "they come from dnf-automatic, which updates every package on this computer."]));
          set.push(h("div", { class: "tb-help" }, ["These are done on the computer itself, by whoever looks after it. A new version reaches dnf shortly after the release. If an update misbehaves: ", h("code", "sudo dnf downgrade tailbond-server")]));
        } else {
          set.push(segOf(selfUp ? [["notify", "Tell me"], ["auto", "Automatic"], ["off", "Off"]] : [["notify", "Tell me"], ["off", "Off"]], policy, "Update policy"),
            h("div", { class: "tb-help" }, policy === "off" ? "Doesn't check for new versions."
              : policy === "auto" ? "Installs new versions by itself between " + win[0] + " and " + (win[1] || "05:00") + " (this router's time). Devices drop for a few seconds; if a new version doesn't start, it goes back by itself."
              : "Checks GitHub once a day and shows new versions here" + (selfUp ? "; nothing installs until you click Update now." : ".")));
        }
        body.push(h("div", { class: "tb-sec" }, set));
        return this.card(h, "Updates", body, h("span", { class: "tb-chip " + chip[0] }, chip[1]), "tb-upd");
      },
      rBackup: function (h) {
        var self = this, srv = this.role === "server", b = this.bset();
        if (!this.st.update && !this.updX) return null; // an agent from before updates and backups
        var on = !!(b.passphrase_set && (b.every_hours || 24));
        var every = b.every_hours || 24, keep = b.keep || 7;
        var everyWords = every % 24 === 0 ? (every === 24 ? "every day" : "every " + (every / 24) + " days") : "every " + every + " hours";
        var body = [
          h("div", { class: "tb-muted", style: { lineHeight: 1.55 } }, srv
            ? "Settings, devices and their keys, locked with a passphrase. Move to a new router: the restore preview says what each device needs."
            : "Settings, servers and rules, locked with a passphrase. With it a new or reset router is back in a minute."),
          h("div", { class: "tb-actions", style: { marginTop: "12px" } }, [
            this.btn(h, "Download a backup", function () { self.openUb("backup"); }, "primary"),
            this.btn(h, "Restore from a backup", function () { self.openUb("restore"); }),
            this.btn(h, "See what's inside", function () { self.openUb("summary"); }, "text")
          ])
        ];
        var sched = [h("div", { class: "tb-sec-t" }, ["Automatic backups", h("span", { class: "tb-chip " + (on ? "ok" : "") }, on ? "On" : "Off")])];
        var s = this.ubSched;
        if (!s) {
          sched.push(h("div", { style: { lineHeight: 1.6 } }, on
            ? ["On: " + everyWords + ", keeps the last " + keep + (b.last ? " · last one " + fmtAgo(b.last) : "") + ".",
              !srv && b.push ? " A copy also goes to " + (this.serverShort || "your home server") + (b.last_push ? " (sent " + fmtAgo(b.last_push) + ")" : "") + ", so you can get it back if this router is lost or reset." : ""]
            : ["Off. Set a passphrase to have tailbond save a backup by itself" + (srv ? "." : ", and send a copy to " + (this.serverShort || "your home server") + ".")]));
          if (on) sched.push(h("div", { class: "tb-muted tb-small", style: { marginTop: "4px" } }, ["Saved in ", h("code", BACKUP_DIR), " on this router (kept through firmware upgrades with Keep Settings). Copy them somewhere else too."]));
          if (b.last_error) sched.push(h("div", { class: "tb-small", style: { color: "#f56c6c", marginTop: "4px" } }, "The last automatic backup didn't work: " + b.last_error));
          sched.push(h("div", { class: "tb-actions" }, [
            this.btn(h, on ? "Change" : "Turn on", function () { self.ubSched = { pass: "", pass2: "", every: String(every), keep: String(keep), push: b.push !== false, err: "" }; }, "small"),
            on ? this.btn(h, "Turn off", function () { self.saveSched({ passphrase: "" }, "Automatic backups off"); }, "small") : null
          ]));
        } else {
          sched.push(h("div", { class: "tb-muted tb-small" }, b.passphrase_set ? "A passphrase is set. Leave the fields empty to keep it, or type a new one." : "Choose a passphrase for the automatic backups (you need it to restore one)."),
            h("div", { class: "tb-muted tb-small", style: { marginTop: "4px" } }, ["The passphrase is kept on this router, in its settings, so it can back up by itself. The files go to ", h("code", BACKUP_DIR), "."]));
          sched = sched.concat(this.rPass(h, s));
          var num = function (k) { return h("input", { class: "tb-input", style: { width: "72px" }, attrs: { type: "number", min: 1 }, domProps: { value: s[k] }, on: { input: function (e) { s[k] = e.target.value; } } }); };
          sched.push(h("div", { class: "tb-row", style: { marginTop: "10px", flexWrap: "wrap", gap: "8px" } }, ["Every", num("every"), "hours, keep the last", num("keep")]));
          if (!srv) sched.push(h("div", { style: { marginTop: "10px" } }, [h("span", { class: "tb-check" + (s.push ? " on" : ""), on: { click: function () { s.push = !s.push; } } }, [h("i"), "Also send a copy to " + (this.serverShort || "your home server") + " (it keeps the last 3)"])]));
          if (s.err) sched.push(h("div", { class: "tb-small", style: { color: "#f56c6c", marginTop: "6px" } }, s.err));
          sched.push(h("div", { class: "tb-actions" }, [
            this.btn(h, this.busy.sched ? "Saving…" : "Save", function () {
              var body2 = { every_hours: Number(s.every) || 24, keep: Number(s.keep) || 7 };
              if (!srv) body2.push = s.push;
              if (s.pass || !b.passphrase_set) { s.err = self.passCheck(s); if (s.err) return; body2.passphrase = s.pass; }
              self.saveSched(body2, "Automatic backups saved");
            }, "small primary", this.busy.sched),
            this.btn(h, "Cancel", function () { self.ubSched = null; }, "small")
          ]));
        }
        body.push(h("div", { class: "tb-sec" }, sched));
        if (srv && this.bks) {
          var mine = this.bks.filter(function (x) { return !x.client; }), theirs = this.bks.filter(function (x) { return x.client; });
          var row = function (x) {
            return h("div", { class: "tb-li", style: { padding: "8px 0" } }, [
              h("div", { class: "tb-grow" }, [h("b", x.client || "this router"), h("span", { class: "tb-muted" }, " · " + fmtWhen(x.created) + " · " + fmtBytes(x.size))]),
              h("a", { class: "tb-btn small", attrs: { href: "/tailbond-api/backups/" + encodeURIComponent(x.name) + (x.client ? "?client=" + encodeURIComponent(x.client) : ""), download: x.name } }, "Download")
            ]);
          };
          body.push(h("div", { class: "tb-sec" }, [h("div", { class: "tb-sec-t" }, "Saved backups")].concat(this.bks.length
            ? [h("div", { class: "tb-list" }, mine.map(row).concat(theirs.map(row))),
              h("div", { class: "tb-muted tb-small", style: { marginTop: "6px" } }, "Each is locked with the passphrase it was made with" + (theirs.length ? "; the travel routers' with their own" : "") + ". Keep a copy somewhere other than this router too.")]
            : [h("div", { class: "tb-muted" }, "None yet. Automatic backups land here, and travel routers that send theirs.")])));
        }
        return this.card(h, "Backup & restore", body, null, "tb-bk");
      },

      ubWarns: function (pv) {
        return A(pv.warnings).map(function (w) { return typeof w === "string" ? w : (w && (w.text || w.message || w.code)) || ""; }).filter(Boolean);
      },
      // restore warnings: numbered steps ("1. …", in order) and plain notes; the
      // stop command inside a step gets its own code line with a Copy button
      restoreSteps: function (h, list, stepsTitle, notesTitle) {
        var self = this, cmd = "sudo systemctl disable --now tailbond";
        var steps = list.map(function (w) { return /^(\d+)\.\s+([\s\S]*)$/.exec(w); }).filter(Boolean);
        var notes = list.filter(function (w) { return !/^\d+\.\s/.test(w); });
        var body = function (t) {
          var i = t.indexOf(cmd);
          if (i < 0) return t;
          var rest = t.slice(i + cmd.length).replace(/^\s*there\)\s*/, "").replace(/^and\s+/, "Then ");
          return [t.slice(0, i).replace(/\s*\($/, "") + " (run this there):",
            h("div", { class: "tb-row", style: { margin: "6px 0", gap: "8px" } }, [h("code", { style: { padding: "4px 8px", whiteSpace: "nowrap", background: "#fff", borderRadius: "4px" } }, cmd), self.btn(h, "Copy", function () { self.copy(cmd); }, "small")]),
            rest === "." ? "" : rest];
        };
        var box = function (title, kids) {
          return h("div", { class: "tb-alert", style: { marginTop: "4px", marginBottom: "4px", fontSize: "14px", border: "2px solid #f3d19e" } }, [h("div", { class: "tb-grow" }, [h("div", { class: "tb-alert-t" }, title), kids])]);
        };
        var out = [];
        if (steps.length) out.push(box(stepsTitle, h("ol", { attrs: { start: steps[0][1] }, style: { margin: "6px 0 0", paddingLeft: "20px" } }, steps.map(function (m) { return h("li", { attrs: { value: m[1] }, style: { margin: "6px 0" } }, body(m[2])); }))));
        if (notes.length) out.push(box(notesTitle, h("ul", { style: { margin: "6px 0 0", paddingLeft: "18px" } }, notes.map(function (w) { return h("li", { style: { margin: "6px 0" } }, w); }))));
        return out.length ? out : null;
      },

      rPass: function (h, s) {
        var lv = strength(s.pass);
        var color = { weak: "#f56c6c", ok: "#e6a23c", strong: "var(--tb-ok,#15803d)" }[lv[0]] || "#ebeef5";
        var width = { weak: "25%", ok: "60%", strong: "100%" }[lv[0]] || "0";
        return [
          h("div", { class: "tb-label" }, "Passphrase"),
          h("input", { class: "tb-input", attrs: { type: "password", autocomplete: "new-password", placeholder: "At least 10 characters" }, domProps: { value: s.pass }, on: { input: function (e) { s.pass = e.target.value; } } }),
          h("div", { class: "tb-bar", style: { marginTop: "6px" } }, [h("i", { style: { width: width, background: color } })]),
          lv[1] ? h("div", { class: "tb-help", style: { marginTop: "4px" } }, lv[1]) : null,
          h("div", { class: "tb-label" }, "Type it again"),
          h("input", { class: "tb-input", attrs: { type: "password", autocomplete: "new-password", placeholder: "Same again" }, domProps: { value: s.pass2 }, on: { input: function (e) { s.pass2 = e.target.value; } } })
        ];
      },

      rUbDialog: function (h) {
        var self = this, u = this.ub, k = u.kind, kids = [], foot = [];
        var close = function () { self.ub = null; self.ubFiles = []; self.ubBody = null; };
        var err = u.err ? h("div", { class: "tb-small", style: { color: "#f56c6c", marginTop: "10px" } }, u.err) : null;
        var p = function (t, extra) { return h("div", { class: "tb-muted", style: Object.assign({ lineHeight: 1.55, marginBottom: "8px" }, extra || {}) }, t); };
        if (k === "backup") {
          kids.push(h("div", { class: "tb-dialog-t" }, "Download a backup"),
            p("Everything this router needs to start over: " + (this.role === "server" ? "settings, devices and their keys, the certificate devices check" : "settings, servers, connection settings and rules") + ". It's locked with a passphrase you choose now."));
          kids = kids.concat(this.rPass(h, u));
          kids.push(h("div", { class: "tb-alert", style: { marginTop: "14px", marginBottom: 0 } }, [h("div", [h("b", "Write it down. "), "Without the passphrase the backup can't be opened, by anyone, tailbond included. A password manager or a note in a drawer both work."])]), err);
          foot = [this.btn(h, "Cancel", close), this.btn(h, u.busy ? "Locking…" : "Download backup", function () { self.doBackup(); }, "primary", u.busy)];
        } else if (k === "restore" && u.step === "pick") {
          kids.push(h("div", { class: "tb-dialog-t" }, "Restore from a backup"),
            p("Pick a .tbbackup file and type its passphrase. Nothing changes yet: you see what it would replace first."),
            h("div", { class: "tb-label" }, "Backup file"),
            h("input", { class: "tb-input tb-file", attrs: { type: "file", accept: ".tbbackup" }, on: { change: function (e) { self.ubFiles = Array.prototype.slice.call(e.target.files || []); u.file = self.ubFiles[0] ? self.ubFiles[0].name : ""; } } }),
            h("div", { class: "tb-label" }, "Passphrase"),
            h("input", { class: "tb-input", attrs: { type: "password", autocomplete: "off", placeholder: "The passphrase of that backup" }, domProps: { value: u.pass }, on: { input: function (e) { u.pass = e.target.value; } } }),
            err);
          foot = [this.btn(h, "Cancel", close), this.btn(h, u.busy ? "Opening…" : "Check the backup", function () { self.doPreview(); }, "primary", u.busy)];
        } else if (k === "restore" && u.step === "preview") {
          var pv = u.preview || {}, ch = A(pv.changes), warns = this.ubWarns(pv);
          kids.push(h("div", { class: "tb-dialog-t" }, "Restore this backup?"),
            h("div", { class: "tb-found", style: { marginBottom: "12px", padding: "12px" } }, [
              h("div", { class: "tb-wan-icon" }, [icon(h, pv.role === "server" ? "server" : "router")]),
              h("div", { class: "tb-grow" }, [h("div", { style: { fontWeight: 600, wordBreak: "break-all" } }, u.file), h("div", { class: "tb-muted tb-small" }, (pv.host || "?") + " (" + (ROLE_WORDS[pv.role] || pv.role) + ") · made " + fmtWhen(pv.created) + " · tailbond " + (pv.agent_version || "?"))])
            ]),
            this.restoreSteps(h, warns, "Do these in order", "Before you restore"),
            h("div", { style: { fontWeight: 600, margin: "12px 0 6px" } }, ch.length ? "What it replaces" : "Nothing differs from the current settings"),
            ch.length ? h("div", { class: "tb-kv tb-small tb-chg" }, [].concat.apply([], ch.map(function (c) { return [h("div", c.what), h("div", [h("span", { class: "tb-muted" }, c.from || "—"), " → ", h("b", c.to || "—")])]; }))) : null,
            p("The current settings are saved first (same passphrase), so you can go back. tailbond restarts afterwards; devices here are offline for a few seconds.", { marginTop: "12px", fontSize: "12px" }),
            err);
          foot = [this.btn(h, "Back", function () { u.step = "pick"; u.err = ""; }), this.btn(h, u.busy ? "Restoring…" : "Restore", function () { self.doRestore(); }, "primary", u.busy)];
        } else if (k === "restore") {
          // after restoring, "Restore here while … still runs" is done
          var dw = this.ubWarns(u.preview || {}).filter(function (w) { return !/^1\.\s+Restore here while /.test(w); }), dhost = (u.preview && u.preview.host) || "the backup";
          kids.push(h("div", { class: "tb-dialog-t", style: { color: "var(--tb-ok,#15803d)", display: "flex", alignItems: "center", gap: "8px" } }, [icon(h, "check", 22), "Restored"]),
            p("The settings from " + dhost + " are in place. A copy of the previous settings was kept" + (u.rollback ? " (" + u.rollback + ")" : "") + ", locked with the same passphrase, in case you want them back.", { color: "#303133" }),
            dw.length ? this.restoreSteps(h, dw, "Next, in order:", "If you haven't already:")
              : p(this.role === "server" ? "Devices reconnect by themselves within a few minutes." : "tailbond connects to its home server again by itself within a minute.", { color: "#303133" }),
            p("Automatic backups now follow the backup: if they were on on " + dhost + ", they stay on here, with the same schedule and passphrase.", { fontSize: "12px" }),
            p("tailbond is restarting with them; this page comes back by itself in a few seconds.", { fontSize: "12px" }));
          foot = [this.btn(h, "Done", function () { close(); self.refreshAll(); }, "primary")];
        } else if (k === "summary") {
          var sm = u.summary ? Object.assign({}, u.summary) : null;
          if (sm && Array.isArray(sm.files)) {
            // a server's backup carries its devices' pushed backups: one row, per device
            var per = {};
            sm.files.forEach(function (f) { var m = /^client-backups\/([^/]+)\//.exec(f); if (m) per[m[1]] = (per[m[1]] || 0) + 1; });
            sm.files = sm.files.filter(function (f) { return f.indexOf("client-backups/") !== 0; });
            if (Object.keys(per).length) sm.client_backups = Object.keys(per).map(function (d) { return d + " (" + per[d] + ")"; });
          }
          var order = Object.keys(SUMMARY_LABEL);
          var rows = sm ? Object.keys(sm).filter(function (key) { return key !== "ok"; }).sort(function (a, b) { return (order.indexOf(a) < 0 ? 99 : order.indexOf(a)) - (order.indexOf(b) < 0 ? 99 : order.indexOf(b)); }).map(function (key) {
            var v = sm[key], val;
            if (v == null || v === "") val = "none";
            else if (Array.isArray(v)) val = v.length ? v.length + ": " + v.join(", ") : "none";
            else if (typeof v === "object") val = Object.keys(v).map(function (x) { return x + " " + v[x]; }).join(", ");
            else if (key === "password_set") val = v ? "set (in the backup, not shown)" : "not set";
            else if (typeof v === "boolean") val = v ? "included" : "no";
            else if (key === "role") val = ROLE_WORDS[v] || v;
            else val = String(v);
            return [h("div", SUMMARY_LABEL[key] || key.replace(/_/g, " ")), h("div", val)];
          }) : [];
          kids.push(h("div", { class: "tb-dialog-t" }, "What's in a backup"),
            p("What a backup made now would hold. Keys and passwords are in it too (locked), but never shown here.", { fontSize: "12px" }),
            sm ? h("div", { class: "tb-kv tb-small tb-chg" }, [].concat.apply([], rows)) : (u.err ? null : h("div", { class: "tb-row tb-muted" }, [h("div", { class: "tb-spin" }), "Loading…"])),
            this.role === "server" ? p("Your devices' own backups are included (still locked with their passphrases), so they move with this router.", { color: "#303133", marginTop: "12px" }) : null,
            p([h("b", "Not included: "), "the Tailscale login, the installed program, firewall settings. On a new router: install tailbond, sign in to Tailscale, then restore."], { color: "#303133", marginTop: this.role === "server" ? 0 : "12px" }),
            err);
          foot = [this.btn(h, "Close", close, "primary")];
        }
        return h("div", { class: "tb-modal", on: { click: function (e) { if (e.target === e.currentTarget && !u.busy) close(); } } }, [
          h("div", { class: "tb-dialog wide", attrs: { role: "dialog" } }, kids.concat([h("div", { class: "tb-actions", style: { justifyContent: "flex-end", marginTop: "18px" } }, foot)]))
        ]);
      }
    }
  };

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
        wantOn: null, actErr: "", showTyped: false, pwEdit: false, pwValue: "", health: null, showHealthOk: false,
        // client: more servers (failover) and bypass rules
        srvAdd: false, srvTyped: "", srvPaste: "", srvErr: null, srvShowPaste: false, backOnUntil: 0, srvMore: "", histAll: false,
        bp: null, lan: null, bpEdit: null,
        // updates and backups
        updX: null, bks: null, ub: null, ubSched: null, restartUntil: 0, restarting: false
      };
    },
    computed: {
      role: function () { return (this.st && this.st.role) || (this.info && this.info.role) || "client"; },
      paired: function () { return !!(this.st && this.st.paired); },
      join: function () { return (this.st && this.st.join) || null; },
      servers: function () { return A(this.st && this.st.servers); },
      serverShort: function () {
        if (this.st && this.st.active_server) return this.st.active_server;
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
          // "Back on homebox" for a minute after an automatic failback
          var was = self.st && self.st.failover && self.st.failover.reason;
          if (was && st.failover && !st.failover.reason && st.state === "connected") self.backOnUntil = Date.now() + 600000;
          self.st = st; self.down = false; self.loaded = true;
          if (self.wantOn !== null && self.isOn() === self.wantOn) self.wantOn = null;
          if (st.role === "client" && !st.paired && !self.disc && !self.discovering) self.discover(false);
          // asking a backup server: its full Tailscale name comes from discovery
          if (st.role === "client" && st.paired && st.join && st.join.add && !self.disc && !self.discovering) self.discover(false);
        }).catch(function (e) {
          self.loaded = true;
          if (e && e.error === "agent_down") self.down = true;
          self.restarting = self.down && Date.now() < self.restartUntil;
        });
      },
      refreshSlow: function () {
        var self = this;
        if (this.down || !this.st) return Promise.resolve();
        var jobs = [api("GET", "/api/settings").then(function (s) { self.settings = s; }).catch(function () {})];
        jobs.push(this.loadUb());
        if (this.st.role === "client") {
          jobs.push(api("GET", "/api/paths").then(function (p) { self.paths = A(p); }).catch(function () {}));
          if (this.st.paired && this.st.state === "connected") jobs.push(this.loadPeers());
          if (this.st.paired) {
            // older agents answer 404: the card stays hidden
            jobs.push(api("GET", "/api/bypass").then(function (b) { self.bp = b; }).catch(function () { self.bp = null; }));
            jobs.push(this.loadLan());
          }
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
      loadLan: function () {
        var self = this;
        return api("GET", "/api/lan-clients").then(function (l) { self.lan = A(l); }).catch(function () {});
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
      // Like run(), but rejects on error (the caller shows it in place)
      // and says okMsg only on success.
      runE: function (key, p, okMsg) {
        var self = this;
        this.$set(this.busy, key, true);
        return p.then(function (r) {
          self.$set(self.busy, key, false);
          if (okMsg) self.say(okMsg);
          return self.tick().then(function () { return self.refreshSlow(); }).then(function () { return r; });
        }, function (e) { self.$set(self.busy, key, false); throw e; });
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
        if (msg === "subnet_in_use") return "That server uses the same tunnel addresses as one you already have.";
        if (msg === "exists") return "That server is already in your list.";
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
          on ? "Turning on" : (self.role === "client" ? "Turned off. Devices on this router use GL's own internet connection directly." : "Turned off"));
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
        var off = !this.isOn();
        var mode = this.st && this.st.mode;
        var mobile = w.kind === "usb" || w.kind === "cellular" || w.kind === "shared-cellular";
        if (!w.enabled) return ["", "Not used", "tailbond leaves this connection alone."];
        if (!w.usable) return ["", "Not connected", NOT_CONNECTED_FIX[w.kind] || NOT_CONNECTED_FIX.unknown];
        if (w.capReached) return ["err", "Paused · monthly limit reached", "Used again next month, or raise the limit."];
        if (w.captive === "portal") {
          if (w.signinUntil) return ["blue", "Signing in…", "Devices here reach this network directly until you've signed in."];
          return ["warn", "Needs sign-in", "This network shows a sign-in page first. Until you sign in, tailbond keeps it out and your other connections carry traffic."];
        }
        var signing = this.wans.some(function (x) { return x.captive === "portal" && x.signinUntil; });
        if (signing && w.state === "active" && !(w.captive === "portal")) return ["blue", "Router's link only", "While devices sign in, only this router's own small link to your home server uses this (almost no data)."];
        if (!on && !off) return ["", "Ready", "Used as set below once tailbond reaches " + (this.serverShort || "your home server") + " again."];
        if (!on) return ["", "Ready", mobile ? "This choice applies when tailbond is on. While it's off, GL's own settings decide; to be sure your phone's data isn't used, unplug the phone or turn tethering off on it." : "This choice applies when tailbond is on. While it's off, GL's own settings (INTERNET page) decide which connection devices use."];
        if (w.state === "active") {
          if (w.priority === "backup" || (mode === "failover" && w.priority !== "primary")) return ["warn", "In use · taking over", "Your main connection stopped working, so this one stepped in."];
          return ["ok", "In use", ""];
        }
        if (w.state === "pending") return ["blue", "Starting…", ""];
        if (w.state === "failed") return ["err", "Not working right now", "It's connected, but can't reach " + (this.serverShort || "your home server") + ". tailbond keeps trying; the other connections carry your traffic."];
        if (w.priority === "backup") return ["", mobile ? "Backup · saving data" : "Backup · standing by", "Used only if your other connections stop working. Until then tailbond sends nothing over it (GL itself may check the connection now and then, a few KB)."];
        if (mode === "failover") return ["", mobile ? "Standing by · saving data" : "Standing by", "Save mobile data is on: used only if Ethernet and Wi-Fi stop working."];
        return ["", "Standing by", ""];
      },
      summary: function () {
        var st = this.st;
        if (!st) return "";
        var server = this.serverShort || "your home server";
        if (st.role === "server") return "";
        if (!this.paired) return "Not set up yet. Connect this router to your home server below.";
        if (!this.isOn()) return "Devices on this router use the internet directly, without your home server, over whichever connection GL's own settings pick. Turn on to bond your connections again.";
        if (st.state === "error" || (st.state === "connecting" && st.error)) return "Can't reach " + server + ". See below.";
        if (st.state === "connecting") return "Connecting to " + server + "…";
        var ws = this.wans, used = [], broken = [], saving = [];
        var self = this;
        ws.forEach(function (w) {
          var s = self.wanStatus(w)[1];
          if (s.indexOf("In use") === 0) used.push(nounOf(w));
          else if (s === "Not working right now") broken.push(nounOf(w));
          else if (/saving data|Backup/.test(s)) saving.push(nounOf(w));
        });
        var line = used.length ? "Using " + used.join(" + ") : "Connected, waiting for a connection to carry traffic";
        var portal = ws.filter(function (w) { return w.enabled && w.captive === "portal"; }).map(function (w) { return nounOf(w); });
        var signing = ws.filter(function (w) { return w.enabled && w.captive === "portal" && w.signinUntil; }).map(function (w) { return nounOf(w); });
        if (signing.length) line = "Signing in on " + signing.join(", ") + ": all traffic from devices here goes out " + signing.join(", ") + " directly, not through " + (this.serverShort || "your home server") + ", until you're signed in";
        else if (portal.length) line += " · " + portal.join(", ") + " needs sign-in";
        if (broken.length) line += " · " + broken.join(", ") + " not working right now";
        else if (saving.length) line += " · " + saving.join(", ") + " on standby";
        var fo = this.foInfo();
        return (fo && fo.line ? fo.line + " · " : "") + line;
      },
      onBackup: function () {
        var ss = this.servers;
        return ss.length > 1 && !!this.st.active_server && this.st.active_server !== ss[0].name;
      },
      // Failover in words for the top card, or null (one server, or on the first choice).
      // act: [link text, handler] shown after the line.
      foInfo: function () {
        var self = this, st = this.st || {}, ss = this.servers, fo = st.failover || {};
        if (ss.length < 2 || !this.isOn()) return null;
        var act = st.active_server || this.serverShort, first = ss[0].name;
        var back = fo.failback_in_sec !== undefined && fo.failback_in_sec !== null;
        var since = fo.since ? fmtHM(fo.since) : "";
        var stay = function () { self.useServer({ name: act }); };
        var goFirst = function () { self.useServer({ name: first }); };
        if (st.state === "connecting" && fo.reason === "primary_down") return { title: first + " is unreachable · switching to " + act + "…" };
        if (st.state !== "connected") return null;
        // what changes while a backup carries the tunnel
        var note = "On " + act + ": websites see " + act + "'s internet address, connections that were open restart once, and your Tailscale machines see devices here as coming from " + act + ".";
        if (fo.reason === "primary_down") {
          return back ? { title: "Using " + act + " (" + first + " is answering again)", line: "Back to " + first + " in " + fmtClock(fo.failback_in_sec) + ". Calls may pause for a few seconds then", act: ["Stay on " + act + " (until you pick " + first + " or " + act + " stops working)", stay], note: note }
            : { title: "Using " + act + (since ? " since " + since : "") + " (" + first + " is unreachable)", line: "You're still online; nothing to do. It goes back to " + first + " by itself once it answers again", note: note };
        }
        if (fo.reason === "manual") {
          var firstOk = ss[0].state === "standby";
          return { title: "Using " + act + (since ? " since " + since : "") + " (picked by hand)", line: "Stays on " + act + " until you pick " + first + " or " + act + " stops working", act: firstOk ? ["Go back to " + first, goFirst] : null, note: note };
        }
        if (fo.reason === "policy") return { title: "Always using " + act, note: act === first ? null : note };
        if (!fo.reason && this.backOnUntil > Date.now()) return { title: "Connected to " + act, line: "Back on " + act + (since ? " since " + since : "") };
        return null;
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
        var self = this, many = this.servers.length > 1;
        this.confirm = {
          title: many ? "Forget all servers?" : "Unpair from " + (this.serverShort || "the home server") + "?",
          body: many
            ? "This router stops using tailbond and forgets " + this.servers.map(function (s) { return s.name; }).join(" and ") + ". Devices here use the internet directly. To connect again, ask a server again (it will need to approve)."
            : "This router stops using tailbond and forgets the home server. Devices here use the internet directly. To connect again, ask the home server again (it will need to approve).",
          ok: many ? "Forget all" : "Unpair", danger: true,
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

      // ---- more servers (client) ----
      openAddServer: function () {
        this.srvAdd = true; this.srvErr = null; this.srvPaste = ""; this.srvShowPaste = false;
        if (!this.disc && !this.discovering) this.discover(false);
      },
      closeAddServer: function () {
        var j = this.join;
        this.srvAdd = false; this.srvErr = null;
        if (j && j.add && j.state !== "asking" && j.state !== "waiting") this.run("srv-ask", api("DELETE", "/api/join")).catch(function () {});
      },
      askAdd: function (server) {
        var self = this;
        this.srvErr = null;
        return this.runE("srv-ask", api("POST", "/api/join", { server: server, add: true })).then(function () { self.srvTyped = ""; })
          .catch(function (e) { self.srvErr = e; });
      },
      pasteAdd: function () {
        var self = this, uri = (this.srvPaste || "").trim();
        this.srvErr = null;
        if (uri.indexOf("tailbond://pair") !== 0) { this.srvErr = { error: "paste" }; return; }
        return this.runE("srv-paste", api("POST", "/api/servers", { uri: uri })).then(function (r) {
          self.say(((r && r.name) || "The server") + " added as a backup");
          self.srvPaste = ""; self.srvAdd = false;
        }, function (e) { self.srvErr = e; });
      },
      // [title, body] for a server that can't be added (409 or the join error).
      // [title, body] for a server that can't be added (409 or the join error).
      addSrvErr: function (h, code, conflict, newName, subnet) {
        var ss = this.servers, self0 = this;
        if (!conflict && ss.length === 1) conflict = ss[0].name;
        var nn = newName || "This server", other = newName || "the new server";
        if (code === "exists") return [nn + " is already in your list", (conflict ? "It's listed as " + conflict + ". " : "") + "Nothing to add; " + (conflict || "it") + " keeps working as before."];
        if (code === "subnet_in_use") {
          // suggest a range that isn't the clashing one
          var n = /^10\.(\d+)\./.exec(subnet || "");
          var free = "10." + (n && +n[1] >= 78 && +n[1] < 99 ? +n[1] + 1 : 78) + ".0.0/24";
          var asked = !!newName, me = (this.settings && this.settings.paired_as) || "my travel router";
          var msg = "Hi! I'd like to add " + (newName || "your tailbond server") + " as a backup server for " + me + ", but it uses the tunnel addresses " + (subnet || "") + ", like " + (conflict || "my other server") +
            ". Could you move it to another range, for example " + free + "? With tailbond's Linux installer: sudo ./install.sh --subnet " + free +
            ". On a GL router: uci set tailbond.server.subnet=" + free + "; uci commit tailbond; /etc/init.d/tailbond restart. Otherwise set \"server\": {\"subnet\": \"" + free + "\"} in /etc/tailbond/tailbond.json and restart tailbond. Devices already connected to it reconnect by themselves. Thanks!";
          return [nn + " can't be added yet: it uses the same tunnel addresses as " + (conflict || "one of your servers"), [
            h("div", (conflict || "Your server") + " keeps working; nothing was changed. Each server needs its own range of tunnel addresses" + (subnet ? " (" + subnet + " is taken)" : "") + "."),
            h("div", { style: { marginTop: "6px" } }, asked
              ? "A new server with no devices yet moves to a free range by itself when you ask it over Tailscale. " + other + " already has devices, so whoever runs it changes its range, then you ask again."
              : "A new server with no devices yet moves to a free range by itself when you ask it over Tailscale (a pasted link can't do that): try “Its name in Tailscale” above. If it already has devices, whoever runs it changes its range."),
            h("div", { style: { marginTop: "8px", fontWeight: 600 } }, "If you run " + other + " yourself:"),
            h("ul", { class: "tb-steps", style: { color: "inherit", marginTop: "2px" } }, [
              h("li", ["Set up with tailbond's Linux installer (most VPSs): on it, run the installer again with ", h("code", "sudo ./install.sh --subnet " + free), " (" + free + " is free here)"]),
              h("li", ["A GL router as server (over SSH): ", h("code", "uci set tailbond.server.subnet=" + free + "; uci commit tailbond; /etc/init.d/tailbond restart")]),
              h("li", ["Another Linux server: in ", h("code", "/etc/tailbond/tailbond.json"), " set ", h("code", "\"server\": {\"subnet\": \"" + free + "\"}"), " and restart tailbond"])
            ]),
            h("div", { class: "tb-small" }, "Devices already connected to " + other + " reconnect by themselves after the change. Then ask again here."),
            h("div", { style: { marginTop: "8px" } }, ["Someone else runs it? ", this.btn(h, "Copy a message to send them", function () { self0.copyMsg(msg); }, "small")])
          ]];
        }
        if (code === "paste") return [null, "Paste the whole link. It starts with tailbond://pair"];
        return [null, this.humanError({ error: code })];
      },
      // client.kill_switch in UCI; the hooks read it the next time the tunnel comes up
      setKillSwitch: function (on) {
        var self = this;
        return this.runE("ks", rpc("set_kill_switch", { on: on }).then(function (r) {
          if (r && r.ok === false) throw { error: r.error || "failed" };
          if (self.info) self.$set(self.info, "kill_switch", on); else self.info = { kill_switch: on };
        }), on ? "Kill switch on. Takes effect the next time the tunnel reconnects." : "Kill switch off. Takes effect the next time the tunnel reconnects.")
          .catch(function (e) { self.actErr = self.humanError(e); });
      },
      copyMsg: function (text) {
        var self = this;
        var done = function () { self.say("Message copied. Paste it in a chat or email to whoever runs that server."); };
        if (navigator.clipboard && window.isSecureContext) navigator.clipboard.writeText(text).then(done, done);
        else {
          var ta = document.createElement("textarea");
          ta.value = text; document.body.appendChild(ta); ta.select();
          try { document.execCommand("copy"); } catch (e) { /* ignore */ }
          document.body.removeChild(ta); done();
        }
      },
      moveServer: function (i, d) {
        var names = this.servers.map(function (s) { return s.name; }), j = i + d;
        if (j < 0 || j >= names.length) return;
        var t = names[i]; names[i] = names[j]; names[j] = t;
        return this.run("srv", api("POST", "/api/servers/order", { names: names }), names[0] + " is the first choice");
      },
      useServer: function (s) {
        return this.run("srv", api("POST", "/api/servers/active", { name: s.name }), "Switching to " + s.name + "…");
      },
      removeServer: function (s) {
        var self = this, rest = this.servers.filter(function (x) { return x.name !== s.name; });
        this.confirm = {
          title: "Remove " + s.name + "?",
          body: "This router forgets " + s.name + (s.active && rest.length ? " and switches to " + rest[0].name + " now" : "") + ". To add it back later, ask " + s.name + " again (someone approves it there).",
          ok: "Remove", danger: true,
          run: function () { return self.run("srv", api("DELETE", "/api/servers/" + encodeURIComponent(s.name)), s.name + " removed"); }
        };
      },
      setPolicy: function (policy, server) {
        var first = (this.servers[0] || {}).name;
        return this.run("srv-policy", api("POST", "/api/servers/policy", policy === "always" ? { policy: "always", server: server } : { policy: "prefer" }),
          policy === "always" ? "Always using " + server : "Prefers " + first + " again");
      },

      // ---- bypass rules (client) ----
      lanByMac: function (mac) {
        mac = (mac || "").toLowerCase();
        return A(this.lan).filter(function (c) { return (c.mac || "").toLowerCase() === mac; })[0] || null;
      },
      wanLabel: function (name) {
        var w = A(this.bp && this.bp.wans).filter(function (x) { return x.name === name; })[0];
        return (w && w.label) || name;
      },
      // What a rule does, in words: "Uses the internet directly" etc.
      targetWords: function (r) {
        if (r.target === "tunnel") return "Always uses tailbond";
        if (r.target === "wan") return "Uses only " + this.wanLabel(r.wan);
        var d = this.defaultWan();
        return "Uses the internet directly" + (d ? " (now: " + d + ")" : "");
      },
      // the WAN carrying the router's default route (GL's pick)
      defaultWan: function () {
        var w = A(this.bp && this.bp.wans).filter(function (x) { return x.default; })[0];
        return w ? w.label || w.name : "";
      },
      openBypass: function (preset) {
        var e = { kind: "domain", text: "", multi: false, mac: "", ip: "", name: "", macTyped: "", addr: "", target: "direct", touched: false, note: "", preset: preset || "", err: "", more: false };
        if (preset === "bank") e.note = "Bank";
        if (preset === "streaming") { e.multi = true; e.text = STREAMING.join("\n"); e.note = "Streaming location"; }
        if (preset === "wificall") { e.kind = "addr"; e.more = true; e.addr = "udp/4500, udp/500"; e.note = "Wi-Fi calling"; }
        if (preset === "laptop") { e.kind = "device"; e.target = "tunnel"; e.note = ""; }
        this.bpEdit = e;
        this.loadLan();
      },
      bpKind: function (k) {
        var e = this.bpEdit;
        e.kind = k; e.err = ""; e.preset = ""; e.note = ""; e.multi = false;
        if (k === "addr") e.more = true;
        if (!e.touched) e.target = k === "device" ? "tunnel" : "direct";
      },
      bpRules: function () {
        var e = this.bpEdit, out = [], bad = "";
        var t = e.target.indexOf("wan:") === 0 ? { target: "wan", wan: e.target.slice(4) } : { target: e.target };
        if (e.kind === "domain") {
          words(e.text).forEach(function (d) {
            d = d.toLowerCase().replace(/^[a-z]+:\/\//, "").replace(/[\/?#].*$/, "").replace(/^\*\./, "").replace(/^www\./, "").replace(/\.$/, "");
            if (!/^[a-z0-9-]+(\.[a-z0-9-]+)+$/.test(d)) bad = bad || d;
            else out.push({ kind: "domain", match: d });
          });
          if (bad) return "“" + bad + "” isn't a website name. Type it like mybank.com.";
          if (!out.length) return "Type a website, like mybank.com.";
        } else if (e.kind === "device") {
          var mac = e.mac || (e.macTyped || "").trim().toLowerCase().replace(/-/g, ":");
          if (!/^([0-9a-f]{2}:){5}[0-9a-f]{2}$/.test(mac)) return e.macTyped ? "That isn't a MAC address. It looks like aa:bb:cc:dd:ee:ff." : "Pick a device from the list.";
          // host: the device's DHCP name; the agent also follows new (private) Wi-Fi addresses under it
          out.push({ kind: "device", match: mac, ip: e.ip || undefined, host: e.mac && e.name ? e.name : undefined, note: e.name || undefined });
        } else {
          words(e.addr).forEach(function (a) {
            a = a.toLowerCase();
            if (/^(tcp|udp)\/\d{1,5}(-\d{1,5})?$/.test(a)) out.push({ kind: "port", match: a });
            else if (/^[0-9a-f:.]+(\/\d{1,3})?$/.test(a) && /[.:]/.test(a)) out.push({ kind: "ip", match: a });
            else bad = bad || a;
          });
          if (bad) return "“" + bad + "” isn't an address or port. Use 203.0.113.9, 203.0.113.0/24 or udp/4500.";
          if (!out.length) return "Type an address or a port.";
        }
        return out.map(function (r) {
          var x = Object.assign({ enabled: true }, r, t);
          if (e.note && !x.note) x.note = e.note.slice(0, 64);
          return x;
        });
      },
      saveBypass: function () {
        var self = this, e = this.bpEdit, rules = this.bpRules();
        if (typeof rules === "string") { e.err = rules; return; }
        e.err = "";
        var p = rules.reduce(function (acc, r) { return acc.then(function () { return api("POST", "/api/bypass", r); }); }, Promise.resolve());
        var what = rules.length > 1 ? (e.note ? e.note + ": " + this.targetWords(rules[0]).toLowerCase() : rules.length + " rules added") : (e.kind === "device" ? (e.name || rules[0].match) : rules[0].match) + ": " + this.targetWords(rules[0]).toLowerCase();
        return this.runE("bp-save", p, what).then(function () { self.bpEdit = null; },
          function (err) { if (self.bpEdit) self.bpEdit.err = self.humanError(err); });
      },
      toggleRules: function (rules, on) {
        var p = Promise.all(rules.map(function (r) { return api("POST", "/api/bypass", Object.assign({}, r, { enabled: on })); }));
        return this.run("bp-" + rules[0].id, p, on ? "Rule on" : "Rule off").catch(function () {});
      },
      deleteRules: function (rules, label) {
        var p = Promise.all(rules.map(function (r) { return api("DELETE", "/api/bypass/" + encodeURIComponent(r.id)); }));
        return this.run("bp-" + rules[0].id, p, label + " removed").catch(function () {});
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
      if (this.bpEdit) kids.push(this.rBypassDialog(h));
      if (this.ub) kids.push(this.rUbDialog(h));
      if (this.confirm) kids.push(this.rConfirm(h));
      if (this.toast) kids.push(h("div", { class: "tb-toast" }, this.toast));
      return h("div", { class: "tb" }, kids);
    },

    // Render helpers (a mixin keeps render() short; Vue binds `this`).
    mixins: [UB, {
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
        alert: function (h, cls, title, body, actions, ico) {
          return h("div", { class: "tb-alert " + (cls || "") }, [
            h("div", { style: { flex: "none", marginTop: "1px" } }, [icon(h, ico || (cls === "ok" ? "check" : "warn"), 20)]),
            h("div", { class: "tb-grow" }, [
              title ? h("div", { class: "tb-alert-t" }, title) : null,
              body ? h("div", body) : null,
              actions && actions.length ? h("div", { class: "tb-actions" }, actions) : null
            ])
          ]);
        },
        hero: function (h, iconCls, title, line, sw, rates, extra) {
          return h("div", { class: "tb-card" }, [
            h("div", { class: "tb-hero" }, [
              h("div", { class: "tb-hero-icon " + iconCls }, [icon(h, "bond", 28)]),
              h("div", { class: "tb-grow" }, [h("div", { class: "tb-hero-t" }, title), h("div", { class: "tb-hero-s" }, line)]),
              sw
            ]),
            rates || null,
            extra || null
          ]);
        },

        rDown: function (h) {
          var self = this;
          if (this.restarting) return this.card(h, null, [h("div", { class: "tb-row" }, [h("div", { class: "tb-spin" }), "tailbond is restarting (update or restore). This page comes back by itself in a minute."])]);
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
              c.note ? h("div", { class: "tb-muted tb-small", style: { marginTop: "8px", lineHeight: 1.5 } }, c.note) : null,
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
          var fo = this.paired && on && !err && !signingIn ? this.foInfo() : null;
          if (fo) title = fo.title;
          var rates = this.paired && on && st.state === "connected" ? h("div", { class: "tb-rates" }, [
            h("span", ["↓ ", h("b", fmtRate(st.totals.rx_bps))]), h("span", ["↑ ", h("b", fmtRate(st.totals.tx_bps))]),
            h("span", { class: "tb-hide-sm" }, ["Up ", h("b", fmtUptime(st.uptime_sec))])
          ]) : null;
          var line = this.summary();
          if (fo && fo.act) line = [line, " · ", h("a", { class: "tb-link", on: { click: fo.act[1] } }, fo.act[0])];
          if (fo && fo.note) line = [h("div", line), h("div", { class: "tb-small tb-muted", style: { marginTop: "4px" } }, fo.note)];
          out.push(this.hero(h, iconCls, title, line,
            this.paired ? this.sw(h, on, function () { self.toggle(); }, true, this.busy.power, "tailbond on/off") : null, rates, this.updTop(h)));
          if (this.actErr) out.push(this.alert(h, "err", null, this.actErr));

          if (!this.paired) return out.concat(this.rSignin(h), this.rPairing(h));
          // off: Backup only and limits don't apply, so GL may use a phone that's plugged in
          var phones = !on ? this.wans.filter(function (w) { return w.kind === "usb" || w.kind === "cellular"; }).map(function (w) { return w.label; }) : [];
          if (phones.length) out.push(this.alert(h, "warn", "While tailbond is off, GL may use " + phones.join(" and ") + " (your phone's data)",
            "“Backup only” and data limits only apply while tailbond is on. To be sure the phone's data isn't used, unplug it or turn tethering off on it, or turn tailbond back on.",
            [h("a", { class: "tb-btn small", attrs: { href: "#/internet" } }, "GL's INTERNET page")]));

          out = out.concat(this.rSignin(h));
          if (on && err) out.push(this.rConnError(h));
          out.push(this.rWans(h));
          out.push(this.rShares(h));
          out.push(this.rModes(h));
          out.push(this.rHome(h));
          if (this.bp) out.push(this.rBypass(h));
          out.push(this.rMachines(h));
          out.push(this.rUpdates(h), this.rBackup(h));
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
              ? "Meanwhile, devices on this router have no internet, so nothing goes around your home server. To use this internet directly for now, turn tailbond off (GL then picks the connection itself, and may use your phone's data: “Backup only” and limits don't apply while off)."
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

        // Pair & Share: phones on this router's Wi-Fi lending their mobile data
        rShares: function (h) {
          var self = this, st = this.st || {}, shares = A(st.shares), useOn = st.use_shares !== false;
          var STATE = { ready: ["ok", "Sharing"], connecting: ["", "Connecting…"], refused: ["warn", "Paused"], unreachable: ["err", "Can't reach it"], lost: ["err", "Not answering"] };
          var rows = shares.map(function (s) {
            var chip = STATE[s.state] || ["", s.state];
            var bits = [];
            if (s.battery >= 0) bits.push("Battery " + s.battery + "%" + (s.charging ? " · charging" : ""));
            if (s.state === "ready") bits.push("↓ " + fmtBytes(s.down_bytes) + " ↑ " + fmtBytes(s.up_bytes));
            if (s.reason && s.state !== "ready") bits.push(s.reason + (/battery/.test(s.reason) ? "; comes back once it's charging" : /limit/.test(s.reason) ? "; comes back next month or when the limit is raised" : ""));
            bits.push(s.via === "server" ? "listed by your home server" : "found on this Wi-Fi");
            return h("div", { class: "tb-row", style: { padding: "8px 0", borderTop: "1px solid #ebeef5" } }, [
              h("div", { class: "tb-wan-icon k-shared-cellular" }, [icon(h, "shared-cellular")]),
              h("div", { class: "tb-grow" }, [h("div", { class: "tb-wan-t" }, s.label), h("div", { class: "tb-muted tb-small" }, bits.join(" · "))]),
              h("span", { class: "tb-chip " + chip[0] }, chip[1])
            ]);
          });
          var body = [
            h("div", { class: "tb-muted", style: { marginBottom: "10px", lineHeight: 1.5 } },
              "A phone connected to this router's own Wi-Fi (not the hotel's) can lend its mobile data, and tailbond adds it to the bond like another SIM. On Android: tailbond → Settings → Share my mobile data. On iPhone: the tailbond Share app → Share mobile data, and keep the app open on screen. Each phone shows up above as a connection with the same Use it / Backup only / Off choice and data limit.")
          ];
          if (st.share_note) body.push(this.alert(h, "warn", st.share_note, "Devices here can't reach each other, so phones can't share. Use USB tethering instead: plug the phone into the router."));
          body = body.concat(rows.length ? rows : [h("div", { class: "tb-muted tb-small" }, useOn ? "No phone is sharing right now." : "Off: phones' shared data isn't used.")]);
          body.push(h("div", { class: "tb-row", style: { marginTop: "12px", borderTop: "1px solid #ebeef5", paddingTop: "12px" } }, [
            h("div", { class: "tb-grow" }, [h("div", "Use mobile data that nearby phones share"),
              h("div", { class: "tb-muted tb-small" }, "Only phones paired with your home server can share with this router.")]),
            this.sw(h, useOn, function () { self.run("use-shares", api("POST", "/api/share", { use_shares: !useOn }), useOn ? "Phones' shared data not used" : "Using phones' shared data"); }, false, this.busy["use-shares"], "Use nearby phones' shared data")
          ]));
          return this.card(h, "Shared by nearby phones", body);
        },

        rWan: function (h, w) {
          var self = this, s = this.wanStatus(w), busy = this.busy["path-" + w.key];
          var mobile = w.kind === "usb" || w.kind === "cellular" || w.kind === "shared-cellular";
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
              h("div", { class: "tb-wan-icon k-" + (w.kind || "unknown") }, [icon(h, w.kind === "unknown" ? "unknown" : w.kind)]),
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
                w.signinUntil ? null : h("div", "GL's Repeater page handles this sign-in. Open it and click “Login Mode for Public Hotspots” (or “Go to Login”). Then open any website on your laptop or phone: the hotel's sign-in page appears there. Accept the terms (or enter your room number)."),
                h("div", { style: { marginTop: "6px" } }, w.signinUntil
                  ? "Now open any website on your laptop or phone: the hotel's sign-in page appears there. Until you've signed in, everything devices here do goes out the hotel Wi-Fi directly (not only the sign-in page, and not through your home server), so your phone's data isn't used for them. Only this router's own link to your home server stays up over your other connections, and it carries almost nothing meanwhile. Login Mode ends by itself once you've signed in (or click “Quit Login Mode” on GL's Repeater page); tailbond then adds " + w.label + " back to bonding."
                  : "While you sign in (until it's done, or 10 minutes), everything devices here do goes out the hotel Wi-Fi directly, not through your home server, and they can't reach your Tailscale machines. This router keeps only its own small link home over your other connections. Afterwards tailbond adds " + w.label + " back to bonding by itself.")
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
              h("div", { class: "tb-small", style: { marginTop: "6px" } }, "While you sign in (up to 10 minutes), everything devices here do goes out " + w.label + " directly, not through your home server, and they can't reach your Tailscale machines; that's what lets the page load. This router keeps its own small link home over your other connections.")
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
          if (this.servers.length > 1) return this.rServers(h);
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
            this.rHomeKv(h, true),
            this.rKillSwitch(h),
            h("div", { class: "tb-actions", style: { marginTop: "14px" } }, [
              this.btn(h, "Ask to connect again", function () { self.repair(); }, "small"),
              this.btn(h, "Unpair", function () { self.unpair(); }, "small danger"),
              this.srvAdd || this.addInFlight() ? null : this.btn(h, "Add a backup server", function () { self.openAddServer(); }, "small")
            ]),
            this.rAddServer(h)
          ]);
        },
        rHomeKv: function (h, withAddr) {
          var st = this.st, s = this.settings || {};
          return h("div", { class: "tb-kv tb-small" }, [
            withAddr ? h("div", "Connects to") : null,
            withAddr ? h("div", [h("span", { class: "tb-mono" }, hostOf(s.paired_with || st.server || "—")), " over the internet (UDP port " + (portOf(s.paired_with) || "443") + ")"]) : null,
            h("div", "Tailnet names"), h("div", "Work on every device connected to this router while tailbond is on. Use the full name, like " + this.exampleName() + "."),
            h("div", "This router's tunnel address"), this.servers.length > 1
              ? h("div", [h("span", { class: "tb-mono" }, st.tunnel_ip || "—"), " on " + (st.active_server || "") + ". It changes with the server. Optional: to reach devices behind this router from your Tailscale machines, approve each server's route in Tailscale."])
              : h("div", { class: "tb-mono" }, st.tunnel_ip || "—")
          ]);
        },
        // http://vps.tail1234.ts.net:8077 for the server being asked
        joinLink: function (j) {
          var srv = j.server_name || shortName(j.server), host = hostOf(j.server);
          var hit = A(this.disc && this.disc.servers).filter(function (s) { return s.name === srv || s.ip === host || s.dns_name === host; })[0];
          var suffix = this.peers && this.peers.magic_dns_suffix;
          var name = (hit && hit.dns_name) || (/\./.test(host) && !/^[\d.]+$/.test(host) ? host : suffix ? srv + "." + suffix : host);
          return "http://" + name + ":8077/";
        },
        // The kill switch: everything "nothing goes out unprotected" and "always tailbond" rest on.
        rKillSwitch: function (h) {
          var self = this, ks = !(this.info && this.info.kill_switch === false);
          return h("div", { class: "tb-row", style: { marginTop: "14px", borderTop: "1px solid #ebeef5", paddingTop: "12px" } }, [
            h("div", { class: "tb-grow" }, [
              h("div", ["Kill switch: block devices here while tailbond can't connect ", h("span", { class: "tb-chip " + (ks ? "ok" : "warn"), style: { marginLeft: "4px" } }, ks ? "On (recommended)" : "Off")]),
              h("div", { class: "tb-muted tb-small", style: { lineHeight: 1.5 } }, (ks
                ? "If no server can be reached (or while it switches), devices here wait instead of using this internet unprotected. Rules that skip tailbond still work."
                : "If no server can be reached, devices here use this internet directly, without tailbond.") + " A change takes effect the next time the tunnel reconnects.")
            ]),
            this.sw(h, ks, function () { self.setKillSwitch(!ks); }, false, this.busy.ks, "Kill switch")
          ]);
        },
        addInFlight: function () {
          var j = this.join;
          return !!(j && j.add && (j.state === "asking" || j.state === "waiting" || j.state === "error"));
        },

        // Two or more servers: the ordered list, the switching choice, add/remove.
        rServers: function (h) {
          var self = this, st = this.st, ss = this.servers, fo = st.failover || {}, busy = this.busy.srv;
          var first = ss[0].name;
          var CHIP = { active: ["ok", "Connected"], connecting: ["blue", "Connecting…"], standby: ["", "Standby"], unreachable: ["err", "Can't reach"], unknown: ["", "Checking…"], off: ["", "Off"] };
          var rows = ss.map(function (sv, i) {
            var chip = !self.isOn() ? ["", "Off"] : CHIP[sv.state] || CHIP.unknown;
            var addr = hostOf(sv.server || "");
            var detail = sv.active && sv.state === "active" ? "Connected for " + fmtUptime(st.uptime_sec) + (addr ? " · " + addr : "")
              : (addr ? addr + " · " : "") + (sv.last_ok ? (sv.state === "unreachable" ? "last answered " : "answered ") + fmtAgoS(sv.last_ok) : sv.state === "unknown" ? "not checked yet" : "hasn't answered yet");
            return h("div", { class: "tb-srv" }, [
              h("div", { class: "tb-wan-icon" }, [icon(h, "server")]),
              h("div", { class: "tb-grow" }, [
                h("div", { class: "tb-wan-t" }, [sv.name, h("span", { class: "tb-role" + (i === 0 ? " first" : "") }, i === 0 ? "First choice" : "Backup")]),
                h("div", { class: "tb-muted tb-small" }, detail)
              ]),
              h("span", { class: "tb-chip " + chip[0] }, chip[1]),
              h("div", { class: "tb-srv-ctl" }, [
                h("button", { class: "tb-btn small icon", attrs: { type: "button", title: "Move up", "aria-label": "Move " + sv.name + " up", disabled: i === 0 || !!busy }, on: { click: function () { self.moveServer(i, -1); } } }, [icon(h, "up", 16)]),
                h("button", { class: "tb-btn small icon", attrs: { type: "button", title: "Move down", "aria-label": "Move " + sv.name + " down", disabled: i === ss.length - 1 || !!busy }, on: { click: function () { self.moveServer(i, 1); } } }, [icon(h, "down", 16)]),
                sv.active || sv.state === "unreachable" || sv.state === "off" ? h("span", { class: "tb-btn small", style: { visibility: "hidden" }, attrs: { "aria-hidden": "true" } }, "Use now")
                  : self.btn(h, "Use now", function () { self.useServer(sv); }, "small", busy || !self.isOn()),
                self.srvMore === sv.name
                  ? self.btn(h, "Remove " + sv.name, function () { self.srvMore = ""; self.removeServer(sv); }, "small danger", busy)
                  : self.btn(h, "More", function () { self.srvMore = sv.name; }, "small text", busy)
              ])
            ]);
          });
          // the switching choice, in the servers' own names
          var cur = fo.policy === "always" ? "always:" + fo.server : "prefer";
          var paused = cur === "prefer" && fo.reason === "manual";
          var preferLabel = ss.length === 2 ? "Prefer " + first + ", use " + ss[1].name + " if " + first + " is down" : "Prefer " + first + ", then the next one in the list that works";
          var opts = [{ key: "prefer", t: preferLabel, b: "Switches by itself when " + first + " is down, and back when it's up again." }]
            .concat(ss.map(function (sv) { return { key: "always:" + sv.name, t: "Always use " + sv.name, b: "Never switches by itself, even if " + sv.name + " is down." }; }));
          var me = this.settings && this.settings.paired_as;
          var ks = !(this.info && this.info.kill_switch === false);
          return this.card(h, "Servers", [
            h("div", { class: "tb-muted", style: { marginBottom: "4px", lineHeight: 1.5 } }, "tailbond uses one server at a time. If it stops answering, the next one in the list takes over. " +
              (ks ? "While it switches, devices here wait a few seconds; with the kill switch on (the default), nothing goes out unprotected." : "While it switches, devices here use the internet directly for a few seconds.")),
            h("div", rows),
            h("div", { class: "tb-sec" }, [
              h("div", { class: "tb-sec-t" }, "When to switch"),
              h("div", { class: "tb-modes" + (opts.length === 3 ? " three" : ""), attrs: { role: "radiogroup", "aria-label": "When to switch" } }, opts.map(function (o) {
                var on = cur === o.key;
                return h("button", { class: "tb-mode" + (on ? " on" : ""), attrs: { type: "button", role: "radio", "aria-checked": String(on), disabled: !!self.busy["srv-policy"] }, on: { click: function () {
                  if (on) return;
                  if (o.key === "prefer") self.setPolicy("prefer"); else self.setPolicy("always", o.key.slice(7));
                } } }, [h("div", { class: "tb-mode-t" }, o.t), h("div", { class: "tb-mode-b" }, o.b)]);
              })),
              paused ? h("div", { class: "tb-alert info", style: { marginTop: "10px", marginBottom: 0, padding: "10px 12px", fontSize: "13px" } }, [h("div", [
                h("b", "Paused: " + st.active_server + " was picked by hand" + (fo.since ? " at " + fmtHM(fo.since) : "") + ". "),
                ss[0].state === "standby" ? h("a", { class: "tb-link", on: { click: function () { self.useServer({ name: first }); } } }, "Go back to automatic (switches to " + first + " now)")
                  : "Automatic switching comes back if " + st.active_server + " stops working, or when you pick " + first + "."])]) : null,
              h("div", { class: "tb-help" }, "How it decides: tailbond checks the standby servers every 10 s with one small packet. It switches when " + first + " hasn't worked for " + fmtDur(fo.switch_after_sec || 15) +
                ", and comes back once " + first + " has answered for " + fmtDur(fo.failback_after_sec || 120) + ". It watches only whether " + first + " answers tailbond: a slow connection or a website that doesn't load doesn't make it switch.")
            ]),
            this.rHistory(h),
            h("div", { class: "tb-sec" }, [this.rHomeKv(h, false)]),
            this.rKillSwitch(h),
            h("div", { class: "tb-actions", style: { marginTop: "14px" } }, [
              this.srvAdd || this.addInFlight() ? null : this.btn(h, "Add another server", function () { self.openAddServer(); }, "small")
            ]),
            this.rAddServer(h),
            h("div", { class: "tb-sec tb-small tb-muted", style: { textAlign: "right" } }, ["Stop using tailbond on this router: ",
              h("a", { class: "tb-link", style: { color: "#f56c6c" }, on: { click: function () { self.unpair(); } } }, "Forget all servers")])
          ], me ? h("span", { class: "tb-muted tb-small", style: { fontWeight: 400 } }, "This router is " + me) : null);
        },

        // "Recent switches" from failover.history (newest first).
        rHistory: function (h) {
          var self = this, hist = A(this.st.failover && this.st.failover.history);
          if (!hist.length) return null;
          var today = new Date().toDateString();
          var when = function (t) {
            var d = new Date(t);
            return (d.toDateString() === today ? "" : d.toLocaleDateString("en-US", { month: "short", day: "numeric" }) + " ") + fmtHM(t);
          };
          var why = function (x) {
            if (x.reason === "failover") return x.detail || x.from + " stopped answering";
            if (x.reason === "failback") return x.from === "" ? "back to " + x.to : x.to + " answered again";
            if (x.reason === "manual") return x.to + " picked by hand";
            if (x.reason === "policy") return "“Always use " + x.to + "”";
            if (x.reason === "list") return "server list changed";
            return x.detail || "";
          };
          var shown = this.histAll ? hist : hist.slice(0, 3);
          return h("div", { class: "tb-sec" }, [
            h("div", { class: "tb-sec-t" }, "Recent switches"),
            h("div", { class: "tb-kv tb-small", style: { gridTemplateColumns: "110px 1fr" } }, [].concat.apply([], shown.map(function (x) {
              return [h("div", when(x.at)), h("div", [h("b", (x.from || "—") + " → " + x.to), h("span", { class: "tb-muted" }, " · " + why(x))])];
            }))),
            hist.length > 3 ? h("div", { class: "tb-small", style: { marginTop: "6px" } }, [h("a", { class: "tb-link", on: { click: function () { self.histAll = !self.histAll; } } }, this.histAll ? "Show fewer" : "Show all (" + hist.length + ")")]) : null
          ]);
        },

        // "Add another server": ask over Tailscale (join with add) or paste a link.
        rAddServer: function (h) {
          var self = this, j = this.join, d = this.disc, body = [];
          var jAdd = j && j.add ? j : null;
          if (!this.srvAdd && !this.addInFlight()) return null;
          var firstName = (this.servers[0] || {}).name || this.serverShort || "your home server";
          var have = {};
          this.servers.forEach(function (s) { have[s.name.toLowerCase()] = true; have[hostOf(s.server || "").toLowerCase()] = true; });
          if (this.serverShort) have[this.serverShort.toLowerCase()] = true;
          if (jAdd && (jAdd.state === "asking" || jAdd.state === "waiting")) {
            var srv = jAdd.server_name || shortName(jAdd.server);
            body.push(h("div", { class: "tb-found" }, [
              h("div", { class: "tb-spin", style: { width: "28px", height: "28px", borderWidth: "3px" } }),
              h("div", { class: "tb-grow" }, [
                h("div", { class: "tb-wan-t" }, jAdd.state === "asking" ? "Asking " + srv + "…" : "Waiting for approval on " + srv + "…"),
                h("div", { class: "tb-muted", style: { marginTop: "4px", lineHeight: 1.6 } }, [
                  "On " + srv + ", click ", h("b", "Approve"), " next to ", h("b", jAdd.name || "this router"), " (its page: ",
                  h("a", { class: "tb-link", attrs: { href: this.joinLink(jAdd), target: "_blank", rel: "noopener" } }, this.joinLink(jAdd).replace(/\/$/, "")),
                  " from any device on your Tailscale). It's added as a backup by itself; " + (this.serverShort || firstName) + " stays in use meanwhile."])
              ]),
              h("div", { class: "tb-actions" }, [this.btn(h, "Cancel", function () { self.run("srv-ask", api("DELETE", "/api/join")).catch(function () {}); })])
            ]));
          } else {
            if (jAdd && jAdd.state === "error") {
              // the join state carries only the code: name the clash from our list
              var em = this.addSrvErr(h, jAdd.error, jAdd.conflict, jAdd.server_name, jAdd.subnet);
              var calm = jAdd.error === "subnet_in_use" || jAdd.error === "exists";
              body.push(this.alert(h, calm ? "" : "err", em[0] || this.joinErrTitle(jAdd), em[0] ? em[1] : this.joinErrBody(jAdd), [this.btn(h, "OK", function () { self.closeAddServer(); }, "small")]));
            } else if (jAdd && jAdd.state === "approved") {
              body.push(this.alert(h, "ok", (jAdd.added || jAdd.server_name || "The server") + " added as a backup", "It takes over if " + firstName + " stops answering."));
            } else if (jAdd && jAdd.state === "declined") {
              body.push(this.alert(h, "err", (jAdd.server_name || "The server") + " declined", "Ask whoever runs it to approve this router, then ask again."));
            } else if (jAdd && jAdd.state === "expired") {
              body.push(this.alert(h, "", "Nobody approved it in time", "The request expired after 10 minutes. Ask again when someone can approve it."));
            }
            // a server that just failed to be added isn't offered again right below its error
            var failed = jAdd && jAdd.state === "error" ? (jAdd.server_name || "").toLowerCase() : "";
            var found = A(d && d.servers).filter(function (s) {
              return (s.name || "").toLowerCase() !== failed && !have[(s.name || "").toLowerCase()] && !have[shortName(s.dns_name).toLowerCase()] && !have[(s.ip || "").toLowerCase()];
            });
            if (this.discovering && !d) body.push(h("div", { class: "tb-row tb-muted", style: { padding: "4px 0 12px" } }, [h("div", { class: "tb-spin" }), "Looking for other tailbond servers on your tailnet…"]));
            found.forEach(function (s) {
              body.push(h("div", { class: "tb-found" }, [
                h("div", { class: "tb-wan-icon" }, [icon(h, "server")]),
                h("div", { class: "tb-grow" }, [h("div", { class: "tb-wan-t" }, "Found on your tailnet: " + s.name), h("div", { class: "tb-muted tb-small" }, (s.dns_name || "") + (s.ip ? " · " + s.ip : ""))]),
                h("div", { class: "tb-actions" }, [self.btn(h, self.busy["srv-ask"] ? "Asking…" : "Ask to add", function () { self.askAdd(s.dns_name || s.ip); }, "primary", self.busy["srv-ask"])])
              ]));
            });
            body.push(h("div", { class: "tb-label", style: { marginTop: found.length ? "4px" : "0" } }, found.length ? "Or type its name in Tailscale" : "Its name in Tailscale"));
            body.push(h("div", { class: "tb-row" }, [
              h("input", { class: "tb-input", attrs: { placeholder: "e.g. vps" }, domProps: { value: this.srvTyped }, on: { input: function (e) { self.srvTyped = e.target.value; }, keyup: function (e) { if (e.key === "Enter" && self.srvTyped.trim()) self.askAdd(self.srvTyped.trim()); } } }),
              this.btn(h, "Ask to add", function () { if (self.srvTyped.trim()) self.askAdd(self.srvTyped.trim()); }, "", !this.srvTyped.trim() || this.busy["srv-ask"])
            ]));
            body.push(h("div", { class: "tb-small", style: { marginTop: "10px" } }, [h("a", { class: "tb-link", on: { click: function () { self.srvShowPaste = !self.srvShowPaste; } } }, this.srvShowPaste ? "Hide the link box" : "No Tailscale there? Paste a pairing link from it")]));
            if (this.srvShowPaste) {
              body.push(h("div", { style: { marginTop: "8px" } }, [
                h("textarea", { class: "tb-input", attrs: { placeholder: "tailbond://pair?d=…" }, domProps: { value: this.srvPaste }, on: { input: function (e) { self.srvPaste = e.target.value; } } }),
                h("div", { class: "tb-actions" }, [this.btn(h, this.busy["srv-paste"] ? "Adding…" : "Add", function () { self.pasteAdd(); }, "primary", this.busy["srv-paste"])])
              ]));
            }
            if (this.srvErr) {
              var e = this.srvErr, ed = e.data || {};
              var pm = this.addSrvErr(h, e.error, ed.conflict, null, ed.subnet);
              body.push(h("div", { style: { marginTop: "12px" } }, [this.alert(h, e.error === "subnet_in_use" || e.error === "exists" ? "" : "err", pm[0], pm[1])]));
            }
          }
          return h("div", { class: "tb-sec" }, [
            h("div", { class: "tb-sec-t" }, ["Add another server", this.srvAdd ? h("a", { class: "tb-link tb-small", style: { fontWeight: 400 }, on: { click: function () { self.closeAddServer(); } } }, "Close") : null]),
            h("div", { class: "tb-muted", style: { marginBottom: "12px", lineHeight: 1.55 } }, "A second tailbond server, for example a VPS, takes over when " + firstName + " can't be reached. It needs tailbond running and the same Tailscale account."),
            h("div", body)
          ]);
        },

        // ---- Bypass tailbond (rules by website, device, address, port) ----
        rBypass: function (h) {
          var self = this, b = this.bp || {}, rules = A(b.rules), sup = b.support || {}, counters = b.counters || {};
          var ks = !(this.info && this.info.kill_switch === false);
          var waiting = sup.domains === "none";
          var hasDomain = rules.some(function (r) { return r.kind === "domain"; });
          var addBtn = rules.length ? this.btn(h, "Add a rule", function () { self.openBypass(); }, "small primary") : null;
          var presets = function (cls) {
            return h("div", { class: "tb-presets" + (cls || "") }, [
              h("button", { class: "tb-preset", attrs: { type: "button" }, on: { click: function () { self.openBypass("bank"); } } }, [icon(h, "globe", 14), "Banking"]),
              h("button", { class: "tb-preset", attrs: { type: "button" }, on: { click: function () { self.openBypass("streaming"); } } }, [icon(h, "globe", 14), "Streaming location"]),
              h("button", { class: "tb-preset", attrs: { type: "button" }, on: { click: function () { self.openBypass("wificall"); } } }, [icon(h, "call", 14), "Wi-Fi calling"]),
              h("button", { class: "tb-preset", attrs: { type: "button" }, on: { click: function () { self.openBypass("laptop"); } } }, [icon(h, "device", 14), "Work laptop"])
            ]);
          };
          if (!rules.length) {
            return this.card(h, "Bypass tailbond", [
              h("div", { class: "tb-empty" }, [
                h("div", { class: "tb-wan-icon" }, [icon(h, "skip")]),
                h("div", { class: "tb-grow" }, [
                  h("div", { style: { lineHeight: 1.6, color: "#606266" } }, "Some things work better without tailbond: bank websites that block VPNs, streaming that should see where you really are, Wi-Fi calling on your phone. Others must never skip it, like a work laptop. Add a rule and they skip tailbond (or always use it), while everything else keeps going through " + (this.serverShort || "your home server") + "."),
                  h("div", { class: "tb-actions", style: { marginTop: "12px", alignItems: "center" } }, [
                    this.btn(h, "Add a rule", function () { self.openBypass(); }, "primary small"),
                    h("span", { class: "tb-muted tb-small" }, "or start from:"),
                    presets()
                  ])
                ])
              ])
            ]);
          }
          // Wi-Fi calling is two port rules; show such a pair as one row
          var rows = [];
          rules.forEach(function (r) {
            var last = rows[rows.length - 1];
            if (last && r.kind === "port" && last.kind === "port" && r.note && r.note === last.note && r.target === last.target && (r.wan || "") === (last.wan || "") && (r.enabled !== false) === last.on) last.rules.push(r);
            else rows.push({ kind: r.kind, note: r.note, target: r.target, wan: r.wan, on: r.enabled !== false, rules: [r] });
          });
          var list = rows.map(function (row) {
            var r = row.rules[0], main, sub = [];
            if (r.kind === "domain") { main = r.match; if (r.note) sub.push(r.note); }
            else if (r.kind === "device") {
              var c = self.lanByMac(r.match) || (r.host ? A(self.lan).filter(function (x) { return x.name === r.host; })[0] : null);
              main = (c && c.name) || r.note || r.match || r.ip;
              sub.push(c && c.name ? (r.match || r.ip) : (r.match ? "device " + r.match : "device"));
              sub.push(!c ? "not seen on this router yet" : c.online ? "connected now" : "not connected now");
              sub.push(r.host ? "follows it by name, even with a new private Wi-Fi address" : "follows this address only");
            } else if (r.kind === "ip") { main = r.match; sub.push(r.note || "address"); }
            else {
              var ports = row.rules.map(function (x) { return x.match.split("/")[1]; });
              var proto = r.match.split("/")[0].toUpperCase();
              var pw = proto + " port" + (ports.length > 1 || /-/.test(ports[0]) ? "s " : " ") + ports.join(" and ");
              main = r.note || pw;
              if (r.note) sub.push(pw);
            }
            var n = row.rules.reduce(function (t, x) { return t + ((counters[x.id] && counters[x.id].connections) || 0); }, 0);
            if (n > 0) sub.push(n + (n === 1 ? " connection" : " connections"));
            var label = main;
            return h("div", { class: "tb-rule" + (row.on ? "" : " off") }, [
              h("div", { class: "tb-wan-icon" }, [icon(h, row.note === "Wi-Fi calling" ? "call" : BP_KIND_ICON[r.kind] || "unknown")]),
              h("div", { class: "tb-grow" }, [
                h("div", { class: "tb-wan-t", style: { overflowWrap: "anywhere" } }, main),
                h("div", { class: "tb-small", style: { marginTop: "2px" } }, [
                  r.kind === "domain" && waiting && row.on ? h("span", { class: "tb-tgt wan" }, "Waiting: goes through tailbond until website rules work")
                    : h("span", { class: "tb-tgt " + (r.target || "direct") }, row.on ? self.targetWords(r) : "Off"),
                  sub.length ? h("span", { class: "tb-muted" }, " · " + sub.join(" · ")) : null
                ])
              ]),
              self.sw(h, row.on, function () { self.toggleRules(row.rules, !row.on); }, false, self.busy["bp-" + r.id], (row.on ? "Turn off: " : "Turn on: ") + label),
              h("button", { class: "tb-btn small text icon", style: { color: "#909399" }, attrs: { type: "button", title: "Delete", "aria-label": "Delete " + label, disabled: !!self.busy["bp-" + r.id] }, on: { click: function () { self.deleteRules(row.rules, label); } } }, [icon(h, "trash", 18)])
            ]);
          });
          var body = [];
          if (waiting && sup.reason && (hasDomain || !sup.firewall)) {
            body.push(this.alert(h, "", hasDomain ? "Website rules are waiting" : "Rules can't be applied on this router", this.rDnsFix(h, sup)));
          }
          body.push(h("div", { class: "tb-muted", style: { marginBottom: "4px" } }, "Everything else goes through " + (this.serverShort || "your home server") + ". " + (ks ? "Rules that skip tailbond keep working while it's down." : "")));
          body.push(h("div", list));
          body.push(h("div", { class: "tb-sec tb-row", style: { flexWrap: "wrap" } }, [h("span", { class: "tb-muted tb-small" }, "Quick add:"), presets()]));
          if (hasDomain) body.push(h("div", { class: "tb-help" }, "Website rules match the addresses the router looks up for each name (sites sharing those addresses skip too). A device with its own secure DNS (Private DNS on Android, Secure DNS in Chrome) isn't matched: turn that off for this Wi-Fi on the device, or add the site's addresses as an address rule."));
          body.push(h("div", { class: "tb-help" }, "Skipped websites are looked up through the connection they use; everything else, IPv6 included, follows the same rules."));
          return this.card(h, "Bypass tailbond", body, addBtn);
        },

        // Website rules wait for dnsmasq-full: what it means and how to get it.
        rDnsFix: function (h, sup, short) {
          var self = this, r = sup.reason || "";
          if (!/dnsmasq-full/.test(r)) return [h("div", r)];
          var pk = /ipset/.test(r) ? " ipset" : "";
          // download first, then swap; put dnsmasq back if the install fails
          var cmd = "cd /tmp && opkg update && opkg download dnsmasq-full dnsmasq && opkg remove dnsmasq && { opkg install ./dnsmasq-full_*.ipk || opkg install ./dnsmasq_*.ipk; }";
          return [
            h("div", "This router's DNS part (dnsmasq) can't do website rules yet. Until it can, these websites keep going through tailbond as before, so a bank that blocks VPNs may stay blocked. Device, address and port rules work now."),
            short ? null : h("div", { style: { marginTop: "6px" } }, ["To turn website rules on, install ", h("b", "dnsmasq-full" + (pk ? " and ipset" : "")), ": in GL's ", h("b", "APPLICATIONS → Plug-ins"), ", search for it and click Install, then turn tailbond off and on here. Do it on a working connection, or leave it to whoever set the router up. If the Plug-ins page refuses, run this over SSH (it downloads both first, so the old one can go back if the install fails)" + (pk ? ", then install ipset the same way" : "") + ", then turn tailbond off and on:"]),
            short ? null : h("div", { class: "tb-row", style: { marginTop: "6px", alignItems: "flex-start" } }, [
              h("code", { class: "tb-grow", style: { display: "block", padding: "6px 8px", wordBreak: "break-all" } }, cmd),
              self.btn(h, "Copy", function () { self.copy(cmd); }, "small")
            ])
          ];
        },
        rBypassDialog: function (h) {
          var self = this, e = this.bpEdit, b = this.bp || {}, sup = b.support || {}, rules = A(b.rules);
          var ks = !(this.info && this.info.kill_switch === false), home = this.serverShort || "your home server";
          var close = function () { self.bpEdit = null; };
          var seg = [["domain", "A website"], ["device", "A device"]].concat(e.more ? [["addr", "An address or port"]] : []);
          var titles = { bank: "Your bank skips tailbond", streaming: "Streaming sees where you are", wificall: "Wi-Fi calling skips tailbond", laptop: "A work laptop always uses tailbond" };
          var body = [];
          body.push(h("div", { class: "tb-presets", style: { marginBottom: "4px" } }, [
            h("span", { class: "tb-muted tb-small" }, "Quick picks:"),
            [["bank", "Banking", "globe"], ["streaming", "Streaming location", "globe"], ["wificall", "Wi-Fi calling", "call"], ["laptop", "Work laptop", "device"]].map(function (p) {
              return h("button", { class: "tb-preset" + (e.preset === p[0] ? " on" : ""), attrs: { type: "button", "aria-pressed": String(e.preset === p[0]) }, on: { click: function () { self.openBypass(p[0]); } } }, [icon(h, p[2], 14), p[1]]);
            })
          ]));
          body.push(h("div", { class: "tb-label" }, "What"));
          body.push(h("div", { class: "tb-row", style: { flexWrap: "wrap" } }, [
            h("div", { class: "tb-seg", attrs: { role: "radiogroup", "aria-label": "What" } }, seg.map(function (o) {
              return h("button", { class: e.kind === o[0] ? "on" : "", attrs: { type: "button", role: "radio", "aria-checked": String(e.kind === o[0]) }, on: { click: function () { self.bpKind(o[0]); } } }, o[1]);
            })),
            e.more ? null : h("a", { class: "tb-link tb-small", on: { click: function () { self.bpKind("addr"); } } }, "More: an address or port")
          ]));
          if (e.kind === "domain") {
            body.push(h("div", { class: "tb-label" }, e.multi ? "Websites, one per line (remove any you don't use)" : e.preset === "bank" ? "Your bank's website" : "Website"));
            body.push(h(e.multi ? "textarea" : "input", { class: "tb-input", style: e.multi ? { height: "118px" } : {}, attrs: { placeholder: e.preset === "bank" ? "e.g. mybank.com" : "e.g. mybank.com", autofocus: true }, domProps: { value: e.text }, on: { input: function (ev) { e.text = ev.target.value; } } }));
            body.push(h("div", { class: "tb-help" }, e.multi
              ? "Each one includes all its pages and parts (www.netflix.com, api.netflix.com…)."
              : "Includes all its pages and parts, like www.mybank.com and login.mybank.com." + (e.preset === "bank" ? " The bank then sees this router's internet, not a VPN. It covers the bank's app too on phones and tablets connected to this router, when the app uses the same website." : "")));
            if (sup.domains === "none" && sup.reason) body.push(h("div", { class: "tb-alert", style: { marginTop: "10px", marginBottom: 0, padding: "10px 12px", fontSize: "13px" } }, [h("div", [h("b", "Saved, but waits for now. ")].concat(this.rDnsFix(h, sup, true)))]));
            body.push(h("div", { class: "tb-help" }, "It matches the addresses the router looks up for that name, so other sites on the same addresses (big hosting providers) may skip tailbond too. A device with its own secure DNS (Private DNS on Android, Secure DNS in Chrome) isn't matched: turn that off for this Wi-Fi on the device, or add the site's addresses as an address rule."));
            body.push(h("div", { class: "tb-help" }, "On this router a website rule covers every name under it; the tailbond phone app covers only the usual ones (www, login, …)."));
          } else if (e.kind === "device") {
            var taken = {};
            rules.forEach(function (r) { if (r.kind === "device" && r.match) taken[r.match.toLowerCase()] = true; });
            var lan = A(this.lan);
            body.push(h("div", { class: "tb-label" }, "Which device"));
            if (this.lan === null) body.push(h("div", { class: "tb-row tb-muted" }, [h("div", { class: "tb-spin" }), "Loading devices…"]));
            else if (!lan.length) body.push(h("div", { class: "tb-muted" }, "No devices found on this router yet. Type the device's MAC address below."));
            else body.push(h("div", { class: "tb-pick", attrs: { role: "radiogroup", "aria-label": "Devices" } }, lan.map(function (c) {
              var on = e.mac === (c.mac || "").toLowerCase();
              return h("div", { class: "tb-pick-i" + (on ? " on" : ""), attrs: { role: "radio", tabindex: 0, "aria-checked": String(on) }, on: { click: function () { e.mac = (c.mac || "").toLowerCase(); e.ip = c.ip || ""; e.name = c.name || ""; e.macTyped = ""; e.err = ""; } } }, [
                h("span", { class: "tb-dot" + (c.online ? " ok" : ""), attrs: { title: c.online ? "Connected" : "Not connected now" } }),
                h("div", { class: "tb-grow" }, [
                  h("div", { style: { fontWeight: 600 } }, c.name || "Unnamed device"),
                  h("div", { class: "tb-mono" }, (c.mac || "") + (c.ip ? " · " + c.ip : ""))
                ]),
                taken[(c.mac || "").toLowerCase()] ? h("span", { class: "tb-muted tb-small" }, "has a rule") : c.online ? null : h("span", { class: "tb-muted tb-small" }, "not connected")
              ]);
            })));
            body.push(h("div", { class: "tb-label" }, "Not listed? Its MAC address"));
            body.push(h("input", { class: "tb-input", attrs: { placeholder: "aa:bb:cc:dd:ee:ff" }, domProps: { value: e.macTyped }, on: { input: function (ev) { e.macTyped = ev.target.value; e.mac = ""; e.ip = ""; e.name = ""; } } }));
            body.push(h("div", { class: "tb-help" }, e.mac && e.name
              ? "tailbond follows " + e.name + " by its name too, so the rule keeps working if it gets a new private Wi-Fi address."
              : "A typed address is followed as it is: if the device switches to a new private Wi-Fi address, the rule stops matching it (and website rules then apply to it). Picking it from the list while it's connected also follows it by name."));
          } else {
            body.push(h("div", { class: "tb-label" }, "Addresses or ports"));
            body.push(h("input", { class: "tb-input", attrs: { placeholder: "e.g. 203.0.113.9, 203.0.113.0/24, udp/4500, tcp/5060-5061" }, domProps: { value: e.addr }, on: { input: function (ev) { e.addr = ev.target.value; } } }));
            body.push(h("div", { class: "tb-help" }, e.preset === "wificall"
              ? "Phones place Wi-Fi calls over UDP ports 500 and 4500. Many carriers refuse them through a VPN, so these skip tailbond. This also lets IPsec VPNs (some work VPNs) skip tailbond; for a work laptop, add a device rule set to “Always use tailbond”."
              : "Separate several with commas. A port looks like udp/4500 or tcp/5060-5061."));
          }
          // where it goes
          var wans = A(b.wans);
          var dw = this.defaultWan();
          var opts = [["direct", "Use the internet directly" + (dw ? " (now: " + dw + ")" : "")]].concat(wans.map(function (w) { return ["wan:" + w.name, "Use only " + (w.label || w.name) + (w.up ? "" : " (not connected now)")]; }), [["tunnel", "Always use tailbond"]]);
          var tgt = e.target.indexOf("wan:") === 0 ? "wan" : e.target;
          var wl = tgt === "wan" ? this.wanLabel(e.target.slice(4)) : "";
          var help = {
            direct: "Skips tailbond and uses the router's main internet connection" + (dw ? " (now " + dw + ")" : "") + ", as without tailbond." + (ks ? " Keeps working while tailbond can't reach your servers." : ""),
            wan: "Skips tailbond and goes out " + wl + " only.",
            tunnel: (e.kind === "device" ? "Everything from this device goes through the server in use (" + home + " now), even websites or ports set to skip tailbond." : "Goes through the server in use (" + home + " now) even if another rule says to skip tailbond.") + (ks ? " With the kill switch on (the default), " + (e.kind === "device" ? "it has" : "these have") + " no internet while tailbond can't connect." : "")
          }[tgt];
          body.push(h("div", { class: "tb-label" }, e.kind === "device" ? "This device should" : "It should"));
          body.push(h("select", { class: "tb-input", attrs: { "aria-label": "Where it goes" }, domProps: { value: e.target }, on: { change: function (ev) { e.target = ev.target.value; e.touched = true; } } },
            opts.map(function (o) { return h("option", { attrs: { value: o[0] }, domProps: { selected: e.target === o[0] } }, o[1]); })));
          body.push(h("div", { class: "tb-help" }, help));
          if (e.err) body.push(h("div", { style: { color: "#f56c6c", marginTop: "10px" } }, e.err));
          return h("div", { class: "tb-modal", on: { click: function (ev) { if (ev.target === ev.currentTarget) close(); } } }, [
            h("div", { class: "tb-dialog wide", attrs: { role: "dialog", "aria-label": "Add a rule" } }, [
              h("div", { class: "tb-dialog-t" }, titles[e.preset] || "Add a rule"),
              h("div", body),
              h("div", { class: "tb-actions", style: { justifyContent: "flex-end", marginTop: "18px" } }, [
                this.btn(h, "Cancel", close),
                this.btn(h, this.busy["bp-save"] ? "Adding…" : "Add", function () { self.saveBypass(); }, "primary", this.busy["bp-save"])
              ])
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
                : "Laptops and phones on this router's Wi-Fi or LAN reach these through the bond to " + (this.serverShort || "your home server") + " (not this router's own Tailscale), by their full name, e.g. " + this.exampleName() + ", or by their 100.x address. “Direct” / “via relay” is how " + (this.serverShort || "your home server") + " reaches each machine." +
                  (this.wans.some(function (w) { return w.captive === "portal" && w.signinUntil; }) ? " While devices are signing in to a hotel page they go out directly, so they don't reach these until that's done." : "") +
                  (this.onBackup() ? " You're on " + this.st.active_server + " now: if " + this.servers[0].name + "'s home is down, machines there may not answer even if this list shows them online." : "")),
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
                    " on any device signed in to your Tailscale account (a phone works too). At home with a Flint 3, its admin panel → VPN → tailbond works as well. Not you? Ask whoever runs your home server to approve it."])
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
            h("div", { class: "tb-muted", style: { marginBottom: "14px", lineHeight: 1.55 } }, "Your home server (a computer or a GL Flint 3 running tailbond) must be on the same Tailscale network as this router. You'll approve this router there with one tap."),
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
            on && conn.length ? h("div", { class: "tb-rates" }, [h("span", ["Sending to devices ", h("b", fmtRate(st.totals.tx_bps))]), h("span", ["Receiving ", h("b", fmtRate(st.totals.rx_bps))])]) : null, this.updTop(h)));
          if (this.actErr) out.push(this.alert(h, "err", null, this.actErr));

          if (this.pending.length) {
            out.push(this.card(h, "Waiting for your approval", this.pending.map(function (r) {
              var s0 = self.settings || {};
              var what = r.kind === "router" ? "Travel router" : r.kind === "phone" ? "Phone" : "Device";
              return h("div", { class: "tb-req" }, [
                h("div", { class: "tb-wan-icon" }, [icon(h, r.kind === "phone" ? "phone" : "router")]),
                h("div", { class: "tb-grow" }, [
                  h("div", { class: "tb-wan-t" }, (r.repair ? r.name + " is asking to connect again" : r.name + " wants to connect")),
                  h("div", { class: "tb-muted tb-small", style: { lineHeight: 1.6 } }, [
                    what + " " + (r.device && r.device !== r.name ? "“" + r.device + "” " : "") + "on your tailnet · signed in as " + (r.user || "unknown"),
                    h("br"), (r.hostname ? r.hostname + " · " : "") + (r.os ? r.os + " · " : "") + r.ip + " · asked " + fmtAgo(r.created)
                  ]),
                  r.repair ? h("div", { class: "tb-small", style: { marginTop: "4px", color: "#8a5d10" } }, "It was connected before (reset or reinstalled). Approving gives it its old access back.") : null,
                  r.user && s0.owner && r.user.toLowerCase() === s0.owner.toLowerCase() ? h("div", { class: "tb-small", style: { marginTop: "4px" } }, "Signed in to " + r.user + ", an account that may approve here. If this device stays with you, just approve. If you'll give it to someone: approve, then before handing it over sign its Tailscale in to their account (or tag it); tailbond keeps working, and until then whoever holds it could also approve devices here. Decline only refuses this request; it can ask again.") : null
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
          out.push(this.rPhoneGuide(h));
          out.push(this.rRouterGuide(h));
          out.push(this.rMachines(h, true));
          out.push(this.rUpdates(h), this.rBackup(h));
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
          var grants = " Approving lets a device use this router's internet connection (by default all its traffic goes out from here) and your home network, and reach your Tailscale machines as this router: they allow it whatever they allow this router, which may be more than its owner's own account gets.";
          return h("div", { class: "tb-muted tb-small", style: { lineHeight: 1.55 } }, (s.password_set
            ? "Who can approve: anyone logged in to this admin panel, or with the tailbond password at " + page + "."
            : s.owner
            ? "Who can approve: anyone logged in to this admin panel, and " + s.owner + " (the Tailscale account this router belongs to) at " + page + ". People you share this router with, and others on your Tailscale network, can ask to connect but can't approve."
            : "Who can approve: anyone logged in to this admin panel, and anyone on your Tailscale network who opens " + page + " (this router is a tagged machine, so it has no single owner). To require a password there, set one under Internet and Tailscale.") + grants);
        },

        // What's left before travel routers can connect from anywhere.
        setupSteps: function () {
          var ts = this.ts, s = this.settings, out = [];
          if (!ts || !s) return out;
          // (the tunnel route is optional: see rRoute)
          if (!ts.installed || !ts.running) out.push("tailscale");
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
          if (ts.running && !ts.routes_approved) {
            items.push(this.alert(h, "", "Optional: let your Tailscale machines reach tailbond devices",
              [h("div", ["Skip this unless you want your Tailscale machines (for example a laptop at home) to open something on a device behind a travel router. Devices behind travel routers reach your Tailscale machines either way. If you want it: in the Tailscale admin console, open ", h("b", ts.self || "this router"), ", choose ", h("b", "Edit route settings"), ", tick ", ts.route || "10.77.0.0/24", " and save (whoever manages your Tailscale account can do it)."])],
              [h("a", { class: "tb-btn small", attrs: { href: ts.admin_url || "https://login.tailscale.com/admin/machines", target: "_blank", rel: "noopener" } }, "Open admin console")]));
          }
          if (!steps.length) {
            items.push(this.alert(h, "ok", "Setup done", "Travel routers and phones can connect from anywhere" + (ts.routes_approved ? ", and your Tailscale machines reach them" : "") + ". The first connection from away from home confirms the port works."));
          }
          items.push(h("div", { class: "tb-kv", style: { marginTop: "4px" } }, [
            h("div", "Internet address"),
            steps.indexOf("address") >= 0 ? h("div", { class: "tb-muted" }, "not set yet (step above)") : (this.addrEdit ? this.addrEditor(h) : h("div", { class: "tb-row" }, [h("span", { class: "tb-mono" }, addr || "not set"),
              this.btn(h, "Change", function () { self.addrEdit = true; self.addrValue = s.public_addr || ""; }, "small text")])),
            h("div", "Port"), steps.indexOf("health:cgnat") >= 0 ? h("div", "UDP " + (s.port || 443) + " is opened on this router, but your provider's shared address (step above) keeps travel routers from reaching it.") : h("div", ["UDP " + (s.port || 443) + ", opened on this router automatically. ", h("span", { class: "tb-muted" }, "Is your internet cable plugged straight into this router? Then nothing else to do. If it's plugged into another box (your internet provider's modem-router), that box must forward UDP " + (s.port || 443) + " to this router; see the box's manual under \"port forwarding\". To test it now: turn tailbond on on a phone using mobile data (Wi-Fi off); Setup health then shows \"UDP " + (s.port || 443) + " reached from the internet\".")]),
            h("div", "Tailscale name"), h("div", { class: "tb-mono" }, ts.dns_name || "—"),
            h("div", "tailbond page password"), this.pwEdit ? h("div", { class: "tb-row" }, [
              h("input", { class: "tb-input", attrs: { type: "password", placeholder: "New password (empty: none)", autocomplete: "new-password" }, domProps: { value: this.pwValue }, on: { input: function (e) { self.pwValue = e.target.value; } } }),
              this.btn(h, "Save", function () { self.run("pw", api("POST", "/api/settings", { ui_password: self.pwValue || "" }), self.pwValue ? "Password set" : "Password removed").then(function () { self.pwEdit = false; self.pwValue = ""; }); }, "primary small", this.busy.pw),
              this.btn(h, "Cancel", function () { self.pwEdit = false; }, "small text")
            ]) : h("div", { class: "tb-row" }, [h("span", s.password_set ? "Set: needed to open http://" + ((ts.dns_name || "").split(".")[0] || "this router") + ":8077" : s.owner ? "None: only " + s.owner + "'s devices can use http://" + ((ts.dns_name || "").split(".")[0] || "this router") + ":8077; set one to let others approve" : "None: anyone on your Tailscale network can open http://" + ((ts.dns_name || "").split(".")[0] || "this router") + ":8077"),
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
          var glFix = "On this router: NETWORK → Firewall → Open Ports on Router, add UDP 41641. If another router sits in front of this one (for example your internet provider's box), also forward UDP port 41641 to this router there.";
          checks = checks.map(function (c) { return (c.id === "tailscale_direct" || c.id === "netcheck") && c.fix ? Object.assign({}, c, { fix: glFix }) : c; });
          var shown = {};
          checks = checks.map(function (c) { if (c.fix && shown[c.fix]) return Object.assign({}, c, { fix: "Same fix as above (one change fixes both).", fix_url: "" }); if (c.fix) shown[c.fix] = true; return c; });
          var bad = checks.filter(function (c) { return c.status === "warn" || c.status === "bad"; });
          var rest = checks.filter(function (c) { return bad.indexOf(c) < 0; });
          var row = function (c) {
            var color = c.status === "ok" ? "var(--tb-ok,#15803d)" : c.status === "bad" ? "#f56c6c" : c.status === "warn" ? "#c88a1a" : "#909399";
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
          var blocking = bad.filter(function (c) { return c.status === "bad"; }).length;
          var head = blocking
            ? h("div", { style: { marginBottom: "6px", color: "#c0392b", fontWeight: 600 } }, blocking + " thing" + (blocking === 1 ? "" : "s") + " to fix before travel routers and phones can connect from away (also the setup step above)" + (bad.length > blocking ? "; " + (bad.length - blocking) + " more for best speed" : "") + ":")
            : bad.length
            ? h("div", { style: { marginBottom: "6px", color: "#8a5d10" } }, "Optional speed-ups (tailbond works fine without them; they only speed up reaching the machines named):")
            : h("div", { style: { marginBottom: "6px", color: "#137a46", fontWeight: 600 } }, unknown
              ? "✓ Nothing to fix. " + unknown + " check" + (unknown === 1 ? " needs" : "s need") + " a device connecting from away (marked ?)."
              : "✓ Everything is set up");
          return this.card(h, "Setup health", [head,
            h("div", { class: "tb-list" }, bad.map(row).concat(bad.length && !this.showHealthOk ? [] : rest.map(row))),
            bad.length && rest.length ? h("a", { class: "tb-link tb-small", on: { click: function () { self.showHealthOk = !self.showHealthOk; } } }, this.showHealthOk ? "Hide the rest" : "Show " + rest.length + " more check" + (rest.length === 1 ? "" : "s")) : null,
            bad.length ? h("div", { class: "tb-small", style: { marginTop: "8px" } }, "Not sure how? Show this card to whoever set up your internet; the links explain each fix.") : null,
            h("div", { class: "tb-muted tb-small", style: { marginTop: "8px" } }, ["Direct connections are faster than relayed ones. ",
              h("a", { class: "tb-link", attrs: { href: "https://tailscale.com/docs/reference/connection-types", target: "_blank", rel: "noopener" } }, "About Tailscale connection types")])
          ], blocking ? h("span", { class: "tb-chip err" }, blocking + " to fix") : bad.length ? h("span", { class: "tb-chip warn" }, bad.length + " to improve") : h("span", { class: "tb-chip ok" }, "All good"));
        },

        // Phones get the app from this router's landing page on its
        // Tailscale name (http://<name>/), then ask to connect like routers.
        rPhoneGuide: function (h) {
          var s = this.settings || {};
          if (!s.landing_url) return null;
          var name = s.landing_typed || "this router's name";
          return this.card(h, "Add an Android phone", [
            h("div", { class: "tb-row", style: { alignItems: "flex-start", gap: "16px", flexWrap: "wrap" } }, [
              s.landing_qr_svg ? h("div", { class: "tb-qr", domProps: { innerHTML: s.landing_qr_svg } }) : null,
              h("div", { class: "tb-grow", style: { minWidth: "220px", lineHeight: 1.6 } }, [
                h("div", ["On the phone, with Tailscale on: point the camera at this code, or type ", h("b", name), " into its browser (with the slash at the end, so it opens the page instead of searching). That page has the tailbond app and three short steps. The phone then asks to connect and shows up at the top of this page: click ", h("b", "Approve"), "."]),
                h("div", { class: "tb-muted tb-small", style: { marginTop: "6px" } }, ["For family: send them ", h("span", { class: "tb-mono" }, s.landing_url), ", or tell them what to type. That page only offers the app; nothing on it can change this router."]),
                s.android_apk ? null : h("div", { class: "tb-small", style: { marginTop: "6px", color: "#8a5d10" } }, "The page can't offer the app yet: this router has no copy of it. Install the tailbond package that includes the Android app, or ask whoever set up tailbond.")
              ])
            ])
          ]);
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
