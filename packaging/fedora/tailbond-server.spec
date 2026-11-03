%global debug_package %{nil}
%global mqvpn_version 0.19.0

Name:           tailbond-server
Version:        %{?tb_version}%{!?tb_version:0.1.0}
Release:        %{?tb_release}%{!?tb_release:1}%{?dist}
Summary:        Bond Wi-Fi, 5G and Ethernet to your own home server (mqvpn + tailbondd)
License:        Apache-2.0
URL:            https://github.com/mp0rta/mqvpn
# Built by packaging/fedora/build-rpm.sh: prebuilt mqvpn (from the
# third_party/mqvpn submodule, v%{mqvpn_version}) and tailbondd, plus the
# packaging files. See build-rpm.sh for how each binary is produced.
Source0:        tailbond-server-%{version}.tar.gz
ExclusiveArch:  x86_64 aarch64

BuildRequires:  systemd-rpm-macros
Requires:       iproute
Requires:       iptables-nft
Recommends:     firewalld
Suggests:       tailscale
Provides:       mqvpn = %{mqvpn_version}
Conflicts:      mqvpn

%description
tailbond bonds several WAN links (Wi-Fi, 5G, Ethernet) of a phone or travel
router to a server you run at home, using mqvpn (Multipath QUIC / MASQUE
CONNECT-IP). This package is the home-server side: the mqvpn binary, the
tailbondd management agent with its web dashboard, a systemd unit,
firewalld service/zone/policies and the Fedora platform hooks. When
Tailscale runs on the server, bonded devices reach the tailnet through it.

%prep
%setup -q

%build
# Binaries are prebuilt by build-rpm.sh.

%install
install -Dpm 0755 bin/mqvpn %{buildroot}%{_bindir}/mqvpn
install -Dpm 0755 bin/tailbondd %{buildroot}%{_bindir}/tailbondd
install -Dpm 0755 lib/mqvpn-server-nat.sh %{buildroot}%{_prefix}/lib/tailbond/mqvpn/mqvpn-server-nat.sh
install -Dpm 0755 lib/mqvpn-nat-config %{buildroot}%{_prefix}/lib/tailbond/mqvpn/mqvpn-nat-config
install -Dpm 0755 hooks/server-up %{buildroot}%{_prefix}/lib/tailbond/hooks/server-up
install -Dpm 0755 hooks/server-down %{buildroot}%{_prefix}/lib/tailbond/hooks/server-down
install -Dpm 0644 tailbond.service %{buildroot}%{_unitdir}/tailbond.service
install -Dpm 0644 firewalld/service-tailbond.xml %{buildroot}%{_prefix}/lib/firewalld/services/tailbond.xml
install -Dpm 0644 firewalld/zone-tailbond.xml %{buildroot}%{_prefix}/lib/firewalld/zones/tailbond.xml
install -Dpm 0644 firewalld/policy-tailbond-out.xml %{buildroot}%{_prefix}/lib/firewalld/policies/tailbond-out.xml
install -Dpm 0644 firewalld/policy-tailbond-in.xml %{buildroot}%{_prefix}/lib/firewalld/policies/tailbond-in.xml
install -Dpm 0600 tailbond.json %{buildroot}%{_sysconfdir}/tailbond/tailbond.json
install -dm 0700 %{buildroot}%{_sharedstatedir}/tailbond
# The Android app and the GL router package, offered for download (phone
# page, Devices tab) when the build had them.
install -dm 0755 %{buildroot}%{_datadir}/tailbond
for f in share/*; do if [ -f "$f" ]; then install -pm 0644 "$f" %{buildroot}%{_datadir}/tailbond/; fi; done
install -Dpm 0644 doc/fedora-server.md %{buildroot}%{_docdir}/%{name}/fedora-server.md
install -Dpm 0644 doc/architecture.md %{buildroot}%{_docdir}/%{name}/architecture.md
install -Dpm 0644 doc/LICENSE.mqvpn %{buildroot}%{_licensedir}/%{name}/LICENSE.mqvpn
install -Dpm 0644 doc/NOTICE.mqvpn %{buildroot}%{_licensedir}/%{name}/NOTICE.mqvpn

%post
%systemd_post tailbond.service
if command -v firewall-cmd >/dev/null 2>&1 && firewall-cmd --state >/dev/null 2>&1; then
    firewall-cmd -q --reload || :
fi

%preun
%systemd_preun tailbond.service

%postun
%systemd_postun_with_restart tailbond.service
if [ "$1" -eq 0 ] && command -v firewall-cmd >/dev/null 2>&1 && firewall-cmd --state >/dev/null 2>&1; then
    z="$(firewall-cmd --get-default-zone 2>/dev/null)"
    for zz in $(firewall-cmd --permanent --get-zones 2>/dev/null); do
        firewall-cmd -q --permanent --zone="$zz" --remove-service=tailbond 2>/dev/null || :
        firewall-cmd -q --permanent --zone="$zz" --remove-port=41641/udp 2>/dev/null || :
    done
    tz="$(firewall-cmd --permanent --get-zone-of-interface=tailscale0 2>/dev/null)" || tz="$z"
    firewall-cmd --permanent --zone="$tz" --list-rich-rules 2>/dev/null | grep -E 'source address="(100\.64\.0\.0/10|fd7a:115c:a1e0::/48)" port port="(80|8077)" protocol="tcp" accept' |
        while read -r r; do firewall-cmd -q --permanent --zone="$tz" --remove-rich-rule="$r" || :; done
    firewall-cmd -q --reload || :
fi

%files
%license %{_licensedir}/%{name}/LICENSE.mqvpn
%license %{_licensedir}/%{name}/NOTICE.mqvpn
%doc %{_docdir}/%{name}/fedora-server.md
%doc %{_docdir}/%{name}/architecture.md
%{_bindir}/mqvpn
%{_bindir}/tailbondd
%dir %{_prefix}/lib/tailbond
%{_prefix}/lib/tailbond/mqvpn/
%{_prefix}/lib/tailbond/hooks/
%{_unitdir}/tailbond.service
%{_prefix}/lib/firewalld/services/tailbond.xml
%{_prefix}/lib/firewalld/zones/tailbond.xml
%{_prefix}/lib/firewalld/policies/tailbond-out.xml
%{_prefix}/lib/firewalld/policies/tailbond-in.xml
%{_datadir}/tailbond/
%dir %attr(0755,root,root) %{_sysconfdir}/tailbond
%config(noreplace) %attr(0600,root,root) %{_sysconfdir}/tailbond/tailbond.json
%dir %attr(0700,root,root) %{_sharedstatedir}/tailbond

%changelog
* Sat Oct 31 2026 tailbond <noreply@tailbond.invalid> - 0.1.0-1
- Initial package: mqvpn %{mqvpn_version}, tailbondd, systemd unit, firewalld integration
