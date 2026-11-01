-- GL oui-httpd argument validation for the "tailbond" RPC object.
-- api{} carries JSON bodies (pairing links, names); the module allowlists
-- paths itself, so its arguments skip the default string pattern.
return {
	api = true,
	status = true,
	info = true,
	service = { action = '^[a-z]+$' },
	set_role = { role = '^[a-z]+$' },
}
