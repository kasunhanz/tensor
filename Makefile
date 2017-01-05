#!/usr/bin/make

NAME = tensor
OS = $(shell uname -s)

# VERSION file provides one place to update the software version
VERSION := $(shell cat VERSION | cut -f1 -d' ')
RELEASE := $(shell cat VERSION | cut -f2 -d' ')

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
DPUT_BIN ?= dput
DPUT_OPTS ?=
DEB_DATE := $(shell LC_TIME=C date +"%a, %d %b %Y %T %z")
ifeq ($(OFFICIAL),yes)
	DEB_RELEASE = $(RELEASE)ppa
	# Sign OFFICIAL builds using 'DEBSIGN_KEYID'
	# DEBSIGN_KEYID is required when signing
	ifneq ($(DEBSIGN_KEYID),)
		DEBUILD_OPTS += -k$(DEBSIGN_KEYID)
	endif
else
	DEB_RELEASE = 0.git$(DATE)$(GITINFO)
	# Do not sign unofficial builds
	DEBUILD_OPTS += -uc -us
	DPUT_OPTS += -u
endif
DEBUILD = $(DEBUILD_BIN) $(DEBUILD_OPTS)
DEB_PPA ?= ppa
# Choose the desired Ubuntu release: trusty xenial yakkety
DEB_DIST ?= unstable

# pbuilder parameters
PBUILDER_ARCH ?= amd64
PBUILDER_CACHE_DIR = /var/cache/pbuilder
PBUILDER_BIN ?= pbuilder
PBUILDER_OPTS ?= --debootstrapopts --variant=buildd --architecture $(PBUILDER_ARCH) --debbuildopts -b

sdist: vet
	mkdir -p build/deb-build/$(NAME)-$(VERSION)/bin/
	mkdir -p build/deb-build/$(NAME)-$(VERSION)/systemd/
	mkdir -p build/deb-build/$(NAME)-$(VERSION)/etc/
	mkdir -p build/deb-build/$(NAME)-$(VERSION)/debian/
	mkdir -p build/deb-build/$(NAME)-$(VERSION)/lib/plugins/inventory
	mkdir -p build/deb-build/$(NAME)-$(VERSION)/lib/playbooks
	cp packaging/config/tensor.conf build/deb-build/$(NAME)-$(VERSION)/etc/
	cp packaging/systemd/tensord.service build/deb-build/$(NAME)-$(VERSION)/systemd/
	cp -a docs build/deb-build/$(NAME)-$(VERSION)/
	go build -v -o build/deb-build/$(NAME)-$(VERSION)/bin/tensord ./tensord/...
	go build -v -o build/deb-build/$(NAME)-$(VERSION)/bin/tensor ./cmd/...
	cp -a packaging/ansible/playbooks/* build/deb-build/$(NAME)-$(VERSION)/lib/playbooks/
	cp -a packaging/ansible/playbooks/* build/deb-build/$(NAME)-$(VERSION)/lib/playbooks/
	cp -a packaging/ansible/plugins/inventory/* build/deb-build/$(NAME)-$(VERSION)/lib/plugins/inventory/
	chmod 774 build/deb-build/$(NAME)-$(VERSION)/lib/plugins/inventory/*
	cd build/deb-build/ && env GZIP=-9 tar -cvzf $(NAME)-$(VERSION).tar.gz $(NAME)-$(VERSION)

clean:
	@echo "Cleaning up distutils stuff"
	rm -rf build
	rm -f packaging/docker/tensor/tensor.deb

debian:	sdist
	@echo "Creating destribution specific build directories"
	@for DIST in $(DEB_DIST) ; do \
		mkdir -p build/deb-build/$${DIST} ; \
		tar -C build/deb-build/$${DIST} -xvf build/deb-build/$(NAME)-$(VERSION).tar.gz ; \
		cp -a packaging/debian/* build/deb-build/$${DIST}/$(NAME)-$(VERSION)/debian/ ; \
		sed -ie "s|%VERSION%|$(VERSION)|g;s|%RELEASE%|$(DEB_RELEASE)|;s|%DIST%|$${DIST}|g;s|%DATE%|$(DEB_DATE)|g" build/deb-build/$${DIST}/$(NAME)-$(VERSION)/debian/changelog ; \
	done

# Create debian package
deb: deb-src
	@for DIST in $(DEB_DIST) ; do \
		PBUILDER_OPTS="$(PBUILDER_OPTS) --distribution $${DIST} --basetgz $(PBUILDER_CACHE_DIR)/$${DIST}-$(PBUILDER_ARCH)-base.tgz --buildresult $(CURDIR)/build/deb-build/$${DIST}" ; \
		$(PBUILDER_BIN) create $${PBUILDER_OPTS} --othermirror "deb http://archive.ubuntu.com/ubuntu $${DIST} universe" ; \
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
	done
	@echo "#############################################"
	@echo "Tensor DEB artifacts:"
	@for DIST in $(DEB_DIST) ; do \
		echo buid/deb-build/$${DIST}/$(NAME)_$(VERSION)-$(DEB_RELEASE)~$${DIST}_source.changes ; \
	done
	@echo "#############################################"

# Build package outside of pbuilder, with locally installed dependencies.
# Install BuildRequires as noted in packaging/debian/control.
local_deb: debian
	@for DIST in $(DEB_DIST) ; do \
	    (cd build/deb-build/$${DIST}/$(NAME)-$(VERSION)/ && $(DEBUILD) -b) ; \
	done
	@echo "#############################################"
	@echo "Ansible DEB artifacts:"
	@for DIST in $(DEB_DIST) ; do \
	    echo build/deb-build/$${DIST}/$(NAME)_$(VERSION)-$(DEB_RELEASE)~$${DIST}_amd64.changes ; \
	done

# Build tensor docker image and tag with current version
docker-build-image: deb
	cp build/deb-build/xenial/*.deb packaging/docker/tensor/tensor.deb
	cd packaging/docker/tensor/ && docker build -t gamunu/tensor:$(VERSION) -t gamunu/tensor:latest .

# Build using docker compose file located in packaging/docker/docker-compose.yml
docker-build: docker-build-image
	cp build/deb-build/xenial/*.deb packaging/docker/tensor/tensor.deb
	docker-compose -f packaging/docker/docker-compose.yml build

# Build and Spin-up containers using docker compose file located in packaging/docker/docker-compose.yml
docker-build-up: docker-build
	cp build/deb-build/xenial/*.deb packaging/docker/tensor/tensor.deb
	docker-compose -f packaging/docker/docker-compose.yml up


# Spin-up, Remove, Stop containers using docker compose file located in packaging/docker/docker-compose.yml
docker-stop:
	docker-compose -f packaging/docker/docker-compose.yml stop

docker-down:
	rm -f packaging/docker/tensor/tensor.deb
	docker-compose -f packaging/docker/docker-compose.yml down

docker-up:
	docker-compose -f packaging/docker/docker-compose.yml up

# up, stop, down tensor container
docker-up-tensor: docker-build-image
	docker-compose -f packaging/docker/docker-compose.yml up tensor

docker-stop-tensor:
	docker-compose -f packaging/docker/docker-compose.yml stop tensor

docker-rm-tensor:
	docker-compose -f packaging/docker/docker-compose.yml rm tensor


# Serve godoc
# requres: sudo apt install golang-golang-x-tools (ubuntu)
doc:
	godoc -http=:6060 -index

# Format go source code
# http://golang.org/cmd/go/#hdr-Run_gofmt_on_package_sources
fmt:
	go fmt ./...

# https://github.com/golang/lint
# go get github.com/golang/lint/golint
lint:
	@golint ./... || true
	@eslint client || true

# run tensor locally using reflex 
run:
	reflex -r '\.go$$' -s -d none -- sh -c 'go run tensord/main.go'

# run test suite
test:
	go test ./...

# Vet examines Go source code and reports suspicious constructs
# http://godoc.org/code.google.com/p/go.tools/cmd/vet
# go get code.google.com/p/go.tools/cmd/vet
vet:
	go vet ./...
