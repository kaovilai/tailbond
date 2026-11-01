'use strict';
'require view';

// LuCI: Services -> tailbond. Links to the dashboard tailbondd serves on :8077.
return view.extend({
	render: function () {
		var url = window.location.protocol + '//' + window.location.hostname + ':8077/';
		return E('div', { 'class': 'cbi-map' }, [
			E('h2', {}, _('tailbond')),
			E('div', { 'class': 'cbi-map-descr' },
				_('Bonds the WAN links into one tunnel to your home server.')),
			E('p', {}, E('a', {
				'class': 'btn cbi-button cbi-button-action',
				'href': url, 'target': '_blank', 'rel': 'noopener'
			}, _('Open the tailbond dashboard'))),
			E('p', {}, E('code', {}, url))
		]);
	},
	handleSave: null,
	handleSaveApply: null,
	handleReset: null
});
