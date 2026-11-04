'use strict';
'require view';

// LuCI: Services -> tailbond. Links to the dashboard tailbondd serves on :8077.
// Brand colours (docs/brand.md) come from assets/brand/tokens.json through
// scripts/brand/gen-tokens.py (the block between the markers).
var BRAND = /*brand-tokens:begin*/'--tb-primary:#0f766e;--tb-on-primary:#ffffff;--tb-ink:#071a1d;--tb-teal:#0d9488;--tb-volt:#65a30d;--tb-beam:linear-gradient(90deg,#0d9488,#65a30d);--tb-ok:#15803d;--tb-warn:#b45309;--tb-error:#dc2626;--tb-wifi:#0284c7;--tb-cellular:#7c3aed;--tb-ethernet:#475569;--tb-usb:#ea580c;--tb-shared:#db2777;--tb-tailnet:#0d9488'/*brand-tokens:end*/;

return view.extend({
	render: function () {
		var url = window.location.protocol + '//' + window.location.hostname + ':8077/';
		return E('div', { 'class': 'cbi-map', 'style': BRAND }, [
			E('h2', { 'style': 'display:flex;align-items:center;gap:10px' }, [
				E('img', { 'src': url + 'icon.svg', 'alt': '', 'width': 32, 'height': 32, 'style': 'border-radius:7px' }),
				_('tailbond')
			]),
			E('div', { 'class': 'cbi-map-descr' },
				_('Bonds the WAN links into one tunnel to your home server.')),
			E('p', {}, E('a', {
				'class': 'btn cbi-button cbi-button-action',
				'style': 'background:var(--tb-primary);border-color:var(--tb-primary);color:var(--tb-on-primary)',
				'href': url, 'target': '_blank', 'rel': 'noopener'
			}, _('Open the tailbond dashboard'))),
			E('p', {}, E('code', {}, url))
		]);
	},
	handleSave: null,
	handleSaveApply: null,
	handleReset: null
});
