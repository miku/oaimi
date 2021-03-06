Summary:    No frills OAI mirror.
Name:       oaimi
Version:    0.2.11
Release:    0
License:    GPL
BuildArch:  x86_64
BuildRoot:  %{_tmppath}/%{name}-build
Group:      System/Base
Vendor:     Leipzig University Library, https://www.ub.uni-leipzig.de
URL:        https://github.com/miku/oaimi

%description

No frills OAI mirror.

%prep

%build

%pre

%install
mkdir -p $RPM_BUILD_ROOT/usr/local/sbin
install -m 755 oaimi $RPM_BUILD_ROOT/usr/local/sbin
install -m 755 oaimi-id $RPM_BUILD_ROOT/usr/local/sbin
install -m 755 oaimi-sync $RPM_BUILD_ROOT/usr/local/sbin

%post

%clean
rm -rf $RPM_BUILD_ROOT
rm -rf %{_tmppath}/%{name}
rm -rf %{_topdir}/BUILD/%{name}

%files
%defattr(-,root,root)

/usr/local/sbin/oaimi
/usr/local/sbin/oaimi-id
/usr/local/sbin/oaimi-sync

%changelog
* Mon Sep 14 2015 Martin Czygan
- 0.1.0 initial release

