%define name tensor
%define tensor_version $VERSION

%if 0%{?rhel} == 5
%define __python /usr/bin/python26
%endif

Name:      %{name}
Version:   %{ansible_version}
Release:   1%{?dist}
Url:       http://www.ansible.com
Summary:   Comprehensive web-based automation framework and Centralized infrastructure management platform
License:   GPLv3
Group:     Tools/Tensor
Source:    https://pearson.com/tensor.git
BuildRoot: %{_tmppath}/%{name}-%{version}-%{release}-buildroot
%{!?python_sitelib: %global python_sitelib %(%{__python} -c "from distutils.sysconfig import get_python_lib; print(get_python_lib())")}

BuildArch: noarch

# RHEL <=5
%if 0%{?rhel} && 0%{?rhel} <= 5
BuildRequires: python26-devel
BuildRequires: python26-setuptools
Requires: python26-PyYAML
Requires: python26-paramiko
Requires: python26-jinja2
Requires: python26-keyczar
Requires: python26-httplib2
Requires: python26-setuptools
Requires: python26-six
%endif

# RHEL == 6
%if 0%{?rhel} == 6
Requires: python-crypto2.6
%endif

# RHEL > 5
%if 0%{?rhel} && 0%{?rhel} > 5
BuildRequires: python2-devel
BuildRequires: python-setuptools
Requires: PyYAML
Requires: python-paramiko
Requires: python-jinja2
Requires: python-keyczar
Requires: python-httplib2
Requires: python-setuptools
Requires: python-six
%endif

# FEDORA > 17
%if 0%{?fedora} >= 18
BuildRequires: python-devel
BuildRequires: python-setuptools
Requires: PyYAML
Requires: python-paramiko
Requires: python-jinja2
Requires: python-keyczar
Requires: python-httplib2
Requires: python-setuptools
Requires: python-six
%endif

# SuSE/openSuSE
%if 0%{?suse_version} 
BuildRequires: python-devel
BuildRequires: python-setuptools
Requires: python-paramiko
Requires: python-jinja2
Requires: python-keyczar
Requires: python-yaml
Requires: python-httplib2
Requires: python-setuptools
Requires: python-six
%endif

Requires: sshpass

%description

Comprehensive web-based automation framework and Centralized infrastructure management platform,
provides role-based access control, job scheduling, inventory management.

%prep
%setup -q

%build
%{__python} setup.py build

%install

mkdir -p %{buildroot}/etc/ansible/
cp examples/hosts %{buildroot}/etc/ansible/
cp examples/ansible.cfg %{buildroot}/etc/ansible/
mkdir -p %{buildroot}/%{_mandir}/man1/
cp -v docs/man/man1/*.1 %{buildroot}/%{_mandir}/man1/
mkdir -p %{buildroot}/%{_datadir}/ansible

%clean
rm -rf %{buildroot}

%files
%defattr(-,root,root)
%{_bindir}/tensor*
%dir %{_datadir}/tensor
%config(noreplace) %{_sysconfdir}/tensor
%doc README.md
%doc %{_mandir}/man1/tensor*

%post
if ! getent group tensor > /dev/null; then
    groupadd --system tensor
fi

%changelog

* Fri, 25 Nov 2016 Gamunu Balagalla <gamunu.balagalla@outlook.com> - 0.1.5
- Release 0.1.5

* Fri, 25 Nov 2016 Gamunu Balagalla <gamunu.balagalla@outlook.com> - 0.1.4
- Release 0.1.4

* Mon, 21 Nov 2016 Gamunu Balagalla <gamunu.balagalla@outlook.com> - 0.1.3
- Release 0.1.3

* Thu, 10 Nov 2016 Gamunu Balagalla <gamunu.balagalla@outlook.com> - 0.0.12
- Release 0.0.12

* Tue, 8 Nov 2016 Gamunu Balagalla <gamunu.balagalla@outlook.com> - 0.1.0
- Release 0.1.0

* Thu, 3 Nov 2016 Gamunu Balagalla <gamunu.balagalla@outlook.com> - 0.0.11
- Release 0.0.11

* Thu, 3 Nov 2016 Gamunu Balagalla <gamunu.balagalla@outlook.com> - 0.0.10
- Release 0.0.10

* Wed, 2 Nov 2016 Gamunu Balagalla <gamunu.balagalla@outlook.com> - 0.0.9
- Release 0.0.9

* Tue, 1 Nov 2016 Gamunu Balagalla <gamunu.balagalla@outlook.com> - 0.0.8
- Release 0.0.8

* Mon, 31 Oct 2016 Gamunu Balagalla <gamunu.balagalla@outlook.com> - 0.0.7
- Release 0.0.7