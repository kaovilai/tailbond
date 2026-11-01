// SPDX-License-Identifier: Apache-2.0
// GL.iNet 4.x admin panel ("oui") view: Applications -> tailbond.
// GL's router fetches /views/gl-sdk4-ui-<view>.common.js and eval()s it into
// a Vue 2 component (menu entry: /usr/share/oui/menu.d/tailbond.json). This
// file is one expression evaluating to such a component; it only links to
// the dashboard tailbondd serves on :8077. Shipped gzipped, as GL's are.
(function () {
    return {
        name: "tailbond",
        render: function (h) {
            var url = window.location.protocol + "//" + window.location.hostname + ":8077/";
            return h("div", { style: { padding: "20px 0" } }, [
                h("p", { style: { marginBottom: "12px" } },
                    "tailbond bonds this router's WAN links into one tunnel to your home server."),
                h("a", {
                    attrs: { href: url, target: "_blank", rel: "noopener" },
                    style: {
                        display: "inline-block", padding: "8px 20px", borderRadius: "4px",
                        background: "#409eff", color: "#fff", textDecoration: "none"
                    }
                }, "Open the tailbond dashboard"),
                h("p", { style: { marginTop: "12px", opacity: 0.7 } }, url)
            ]);
        }
    };
})()
