#!/usr/bin/make

NAME = tensor
OS ?= $(shell uname -s)
-DEB_OS ?= $(shell lsb_release -si)

# VERSION file provides one place to update the software version
VERSION := $(shell cat VERSION | cut -f1 -d' ')
RELEASE := $(shell cat VERSION | cut -f2 -d' ')

ifeq ($(DEB_OS), Ubuntu)
DEB_MIRROR = "http://archive.ubuntu.com/ubuntu"
DEB_REPO = "universe"
else
DEB_MIRROR = "http://httpredir.debian.org/debian"
DEB_REPO = "non-free"
endif

# Get the branch information from git
ifneq ($(shell which git),)
GIT_DATE := $(shell git log -n 1 --format="%ai")
GIT_HASH := $(shell git log -n 1 --format="%h")
GIT_BRANCH := $(shell git rev-parse --abbrev-ref HEAD | sed 's/[-_.\/]//g')
GITINFO = .$(GIT_HASH).$(GIT_BRANCH)
else
GITINFO = ""
endif

ifeq ($(shell echo $(OS) | egrep -c 'Darwin|FreeBSD|OpenBSD'),1)
DATE := $(shell date -j -r $(shell git log -n 1 --format="%at") +%Y%m%d%H%M)
else
DATE := $(shell date --utc --date="$(GIT_DATE)" +%Y%m%d%H%M)
endif

# DEB build parameters
DEBUILD_BIN ?= debuild
DEBUILD_OPTS = --source-option="-I"
DEB_DATE := $(shell LC_TIME=C date +"%a, %d %b %Y %T %z")

ifeq ($(OFFICIAL),yes)
DEB_RELEASE = $(RELEASE)ppa
else
DEB_RELEASE = 0.git$(DATE)$(GITINFO)
# Do not sign unofficial builds
DEBUILD_OPTS += -uc -us
endif
# Sign OFFICIAL builds using 'DEBSIGN_KEYID'
# DEBSIGN_KEYID is required when signing
ifdef ($(DEBSIGN_KEYID))
DEBUILD_OPTS += -k$(DEBSIGN_KEYID)
endif
DEBUILD = $(DEBUILD_BIN) $(DEBUILD_OPTS)
# Choose the desired Ubuntu release: trusty xenial yakkety
DEB_DIST ?= unstable

# pbuilder parameters
PBUILDER_ARCH ?= amd64
PBUILDER_CACHE_DIR = /var/cache/pbuilder
PBUILDER_BIN ?= pbuilder
PBUILDER_OPTS ?= --debootstrapopts --variant=build --architecture $(PBUILDER_ARCH) --debbuildopts -b

# RPM build parameters
RPMSPECDIR= packaging/rpm
RPMSPEC = $(RPMSPECDIR)/tensor.spec
RPMRELEASE = $(RELEASE)
ifneq ($(OFFICIAL),yes)
RPMRELEASE = 100.git$(DATE)$(GITINFO)
endif
RPMNVR = "$(NAME)-$(VERSION)-$(RPMRELEASE)"

sdist: vet
	mkdir -p build/$(NAME)-$(VERSION)/bin/
	mkdir -p build/$(NAME)-$(VERSION)/systemd/
	mkdir -p build/$(NAME)-$(VERSION)/etc/
	mkdir -p build/$(NAME)-$(VERSION)/lib/plugins/inventory
	mkdir -p build/$(NAME)-$(VERSION)/lib/playbooks
	cp packaging/config/tensor.conf build/$(NAME)-$(VERSION)/etc/
	cp packaging/systemd/tensord.service build/$(NAME)-$(VERSION)/systemd/
	cp -a docs build/$(NAME)-$(VERSION)/
	go build -v -o build/$(NAME)-$(VERSION)/bin/tensord ./tensord/...
	go build -v -o build/$(NAME)-$(VERSION)/bin/tensor ./cmd/...
	cp -a packaging/ansible/playbooks/* build/$(NAME)-$(VERSION)/lib/playbooks/
	cp -a packaging/ansible/playbooks/* build/$(NAME)-$(VERSION)/lib/playbooks/
	cp -a packaging/ansible/plugins/inventory/* build/$(NAME)-$(VERSION)/lib/plugins/inventory/
	chmod 774 build/$(NAME)-$(VERSION)/lib/plugins/inventory/*
	cd build/ && env GZIP=-9 tar -cJf $(NAME)-$(VERSION).tar.xz $(NAME)-$(VERSION)
	cd build/ && env GZIP=-9 tar -cvf $(NAME)-$(VERSION).tar.gz $(NAME)-$(VERSION)
	rm -rf build/$(NAME)-$(VERSION)/

clean:
	@echo "Cleaning up distutils stuff"
	rm -rf build

debian:	sdist
	@echo "Creating distribution specific build directories"
	@for DIST in $(DEB_DIST) ; do \
		mkdir -p build/deb-build/$${DIST} ; \
		tar -C build/deb-build/$${DIST} -xvf build/$(NAME)-$(VERSION).tar.xz ; \
		mkdir -p build/deb-build/$${DIST}/$(NAME)-$(VERSION)/debian/ ; \
		cp -a packaging/debian/* build/deb-build/$${DIST}/$(NAME)-$(VERSION)/debian/ ; \
		sed -ie "s|%VERSION%|$(VERSION)|g;s|%RELEASE%|$(DEB_RELEASE)|;s|%DIST%|$${DIST}|g;s|%DATE%|$(DEB_DATE)|g" build/deb-build/$${DIST}/$(NAME)-$(VERSION)/debian/changelog ; \
	done

# Create debian package
deb: deb-src
	@for DIST in $(DEB_DIST) ; do \
		PBUILDER_OPTS="$(PBUILDER_OPTS) --distribution $${DIST} --basetgz $(PBUILDER_CACHE_DIR)/$${DIST}-$(PBUILDER_ARCH)-base.tgz --buildresult $(CURDIR)/build/deb-build/$${DIST}" ; \
		$(PBUILDER_BIN) create $${PBUILDER_OPTS} --othermirror "deb ${DEB_MIRROR} $${DIST} ${DEB_REPO}" ; \
		$(PBUILDER_BIN) update $${PBUILDER_OPTS} ; \
		$(PBUILDER_BIN) build $${PBUILDER_OPTS} build/deb-build/$${DIST}/$(NAME)_$(VERSION)-$(DEB_RELEASE)~$${DIST}.dsc ; \
	done
	@echo "#############################################"
	@echo "Tensor DEB artifacts:"
	@for DIST in $(DEB_DIST) ; do \
		echo build/deb-build/$${DIST}/$(NAME)_$(VERSION)-$(DEB_RELEASE)~$${DIST}_amd64.changes ; \
	done

# Create debian sources package
deb-src: debian
	@for DIST in $(DEB_DIST) ; do \
		(cd build/deb-build/$${DIST}/$(NAME)-$(VERSION)/ && $(DEBUILD) -S) ; \
		rm -rf build/deb-build/$${DIST}/$(NAME)-$(VERSION)/ ; \
	done
	@echo "#############################################"
	@echo "Tensor DEB artifacts:"
	@for DIST in $(DEB_DIST) ; do \
		echo buid/deb-build/$${DIST}/$(NAME)_$(VERSION)-$(DEB_RELEASE)~$${DIST}_source.changes ; \
	done
	@echo "#############################################"

rpmcommon: sdist
	@mkdir -p build/rpm-build
	@cp build/*.tar.gz build/rpm-build/
	@sed -e 's#^Version:.*#Version: $(VERSION)#' -e 's#^Release:.*#Release: $(RPMRELEASE)%{?dist}#' $(RPMSPEC) > build/rpm-build/$(NAME).spec

srpm: rpmcommon
	@rpmbuild --define "_topdir %(pwd)/build/rpm-build" \
	--define "_builddir %{_topdir}" \
	--define "_rpmdir %{_topdir}" \
	--define "_srcrpmdir %{_topdir}" \
	--define "_specdir $(RPMSPECDIR)" \
	--define "_sourcedir %{_topdir}" \
	-bs build/rpm-build/$(NAME).spec
	@rm -f build/rpm-build/$(NAME).spec
	@echo "#############################################"
	@echo "Tensor SRPM is built:"
	@echo "    build/rpm-build/$(RPMNVR).src.rpm"
	@echo "#############################################"

rpm: rpmcommon
	@rpmbuild --define "_topdir %(pwd)/build/rpm-build" \
	--define "_builddir %{_topdir}" \
	--define "_rpmdir %{_topdir}" \
	--define "_srcrpmdir %{_topdir}" \
	--define "_specdir $(RPMSPECDIR)" \
	--define "_sourcedir %{_topdir}" \
	--define "_rpmfilename %%{NAME}-%%{VERSION}-%%{RELEASE}.%%{ARCH}.rpm" \
	-ba build/rpm-build/$(NAME).spec
	@rm -f build/rpm-build/$(NAME).spec
	@echo "#############################################"
	@echo "Tensor RPM is built:"
	@echo "    build/rpm-build/$(RPMNVR).noarch.rpm"
	@echo "#############################################"

mongo:
	mongo tensordb --eval 'db.createUser({user:"tensor",pwd:"tensor",roles:["readWrite","dbAdmin" ] });'
	mongo tensordb --eval 'db.users.insert({"_id":new ObjectId(),"username":"admin","first_name":"Gamunu",'\
	'"last_name":"Balagalla","email":"gamunu.balagalla@outlook.com","is_superuser" : true,"is_system_auditor":false});'

travis:
	$(MAKE) test
	@echo "" > coverage.txt
	@for d in $$(go list ./...) ; do \
		go test -race -coverprofile=profile.out -covermode=atomic $$d; \
		if [ -f profile.out ]; then \
			cat profile.out >> coverage.txt; \
			rm profile.out; \
		fi; \
	done
	@if [ "$$OFFICIAL" = 'yes' ]; then \
		openssl aes-256-cbc -K $encrypted_6b996d8977fa_key -iv $encrypted_6b996d8977fa_iv -in codesigning.asc.enc -out codesigning.asc -d; \
		gpg --fast-import codesigning.asc; \
	fi;
	$(MAKE) DEB_DIST='xenial trusty precise' DEB_OS='Ubuntu' deb-src
	$(MAKE) DEB_OS='Debian' DEB_DIST='jessie' deb-src
	$(MAKE) srpm
	@if [ "$$OFFICIAL" = 'yes' ]; then \
		gpg --sign --armor build/$(NAME)-$(VERSION).tar.xz; \
		gpg --sign --armor build/$(NAME)-$(VERSION).tar.gz; \
		openssl dgst -sha512 build/$(NAME)-$(VERSION).tar.xz > build/$(NAME)-$(VERSION).tar.xz.sha512; \
		openssl dgst -sha512 build/$(NAME)-$(VERSION).tar.gz > build/$(NAME)-$(VERSION).tar.gz.sha512; \
	fi;
	@rm -f codesigning.asc

# Build tensor docker image and tag with current version
docker:
	cd packaging/docker/tensor/ && docker build -t gamunu/tensor:$(VERSION) -t gamunu/tensor:latest .

# Spin-up, Remove, Stop containers using docker compose file located in packaging/docker/docker-compose.yml
docker-stop:
	docker-compose -f packaging/docker/docker-compose.yml stop

docker-down:
	rm -f packaging/docker/tensor/tensor.deb
	docker-compose -f packaging/docker/docker-compose.yml down

docker-up:
	docker-compose -f packaging/docker/docker-compose.yml up

# Serve godoc
# requres: sudo apt install golang-golang-x-tools (ubuntu)
doc:
	godoc -http=:6060 -index

# Format go source code
# http://golang.org/cmd/go/#hdr-Run_gofmt_on_package_sources
fmt:
	go fmt ./...

# Vet examines Go source code and reports suspicious constructs
# http://godoc.org/code.google.com/p/go.tools/cmd/vet
# go get code.google.com/p/go.tools/cmd/vet
vet:
	go vet ./...

# https://github.com/golang/lint
# go get github.com/golang/lint/golint
lint:
	@golint ./... || true
	@eslint client || true

# run tensor locally using reflex 
run:
	reflex -r '\.go$$' -s -- sh -c 'go run tensord/main.go'

runsetup:
	go run cmd/main.go -setup

# run test suite
test:
	go test -v ./...
