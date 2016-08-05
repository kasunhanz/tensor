#!/usr/bin/make
# WARN: gmake syntax
########################################################
# Makefile for Ansible
#
# useful targets:
#   make sdist ---------------- produce a tarball
#   make srpm ----------------- produce a SRPM
#   make rpm  ----------------- produce RPMs
#   make deb-src -------------- produce a DEB source
#   make deb ------------------ produce a DEB
#   make docs ----------------- rebuild the manpages (results are checked in)
#   make tests ---------------- run the tests (see test/README.md for requirements)
#   make pyflakes, make pep8 -- source code checks

########################################################
# variable section

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
DEB_PPA ?= ppa#
Choose the desired Ubuntu release: lucid precise saucy trusty xenial
DEB_DIST = unstable

default: build

build: vet
	go build -v -o ./build/$(NAME)-$(VERSION)/tensord ./service
	go build -v -o ./build/$(NAME)-$(VERSION)/tensor ./cmd

clean:
	@echo "Cleaning up distutils stuff"
	rm -rf build
	rm -rf dist
	@echo "Cleaning up Debian building stuff"
	rm -rf deb-build

debian:	build
    # create directories
	mkdir -p deb-build/$(NAME)-$(VERSION)/etc/
	mkdir -p deb-build/$(NAME)-$(VERSION)/bin
	mkdir -p deb-build/$(NAME)-$(VERSION)/systemd/

	cp packaging/config/tensor.conf deb-build/$(NAME)-$(VERSION)/etc/
	cp -a packaging/systemd/tensord.service deb-build/$(NAME)-$(VERSION)/systemd/
	cp build/$(NAME)-$(VERSION)/tensord deb-build/$(NAME)-$(VERSION)/bin
	cp build/$(NAME)-$(VERSION)/tensor deb-build/$(NAME)-$(VERSION)/bin
	cp -a docs deb-build/$(NAME)-$(VERSION)/
	cp -a packaging/debian deb-build/$(NAME)-$(VERSION)/
	sed -ie "s|%VERSION%|$(VERSION)|g;s|%RELEASE%|$(DEB_RELEASE)|;s|%DIST%|$(DEB_DIST)|g;s|%DATE%|$(DEB_DATE)|g;" deb-build/$(NAME)-$(VERSION)/debian/changelog
	sed -ie "s|%VERSION%|$(VERSION)|g;s|%RELEASE%|$(DEB_RELEASE)|;s|%DIST%|$(DEB_DIST)|g;" deb-build/$(NAME)-$(VERSION)/debian/control

	#fix permission issues
	chmod +x deb-build/$(NAME)-$(VERSION)/systemd/tensord.service
	chmod +x deb-build/$(NAME)-$(VERSION)/bin/tensord
	chmod +x deb-build/$(NAME)-$(VERSION)/bin/tensor

deb: debian
	cd deb-build/$(NAME)-$(VERSION)/ && $(DEBUILD) -b
	@echo "#############################################"
	@echo "Tensor DEB artifacts:"
	@echo deb-build/$(NAME)_$(VERSION)-$(DEB_RELEASE)~amd64.changes
	@echo "#############################################"

deb-src: debian
	cd deb-build/$(NAME)-$(VERSION)/ && $(DEBUILD) -S
	@echo "#############################################"
	@echo "Tensor DEB artifacts:"
	@echo deb-build/$(NAME)_$(VERSION)-$(DEB_RELEASE)~source.changes
	@echo "#############################################"

#sudo apt install golang-golang-x-tools (ubuntu)
doc:
	godoc -http=:6060 -index

# http://golang.org/cmd/go/#hdr-Run_gofmt_on_package_sources
fmt:
	go fmt ./...

# https://github.com/golang/lint
# go get github.com/golang/lint/golint
lint:
	golint ./...

run:
	reflex -r '\.go$$' -s -d none -- sh -c 'go run cli/tensord.go'

test:
	go test ./...

# http://godoc.org/code.google.com/p/go.tools/cmd/vet
# go get code.google.com/p/go.tools/cmd/vet
vet:
	go vet ./...
