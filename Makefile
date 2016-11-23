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
DEB_PPA ?= ppa#
DEB_DIST = unstable

DEB_BUILD_DIR = build/deb-build/$(NAME)-$(VERSION)

default: build

build: vet
	export GOARCH="amd64"
	go build -v -o ./build/$(NAME)-$(VERSION)/tensord ./tensord/...
	go build -v -o ./build/$(NAME)-$(VERSION)/tensor ./cmd/...

clean:
	@echo "Cleaning up distutils stuff"
	rm -rf build
	rm -f packaging/docker/tensor/tensor.deb

debian:	build
	# create directories
	mkdir -p $(DEB_BUILD_DIR)/bin/
	mkdir -p $(DEB_BUILD_DIR)/systemd/
	mkdir -p $(DEB_BUILD_DIR)/etc/
	mkdir -p $(DEB_BUILD_DIR)/debian/
	mkdir -p $(DEB_BUILD_DIR)/lib/plugins/inventory
	mkdir -p $(DEB_BUILD_DIR)/lib/playbooks
	#Copy configuration files and manpages
	cp packaging/config/tensor.conf $(DEB_BUILD_DIR)/etc/
	cp packaging/systemd/tensord.service $(DEB_BUILD_DIR)/systemd/
	cp -a docs $(DEB_BUILD_DIR)/ 
	# Copy generated binaries
	cp build/$(NAME)-$(VERSION)/tensord $(DEB_BUILD_DIR)/bin
	cp build/$(NAME)-$(VERSION)/tensor $(DEB_BUILD_DIR)/bin
	# Generate changelog copy postinstall 
	cp -a packaging/debian/* $(DEB_BUILD_DIR)/debian/
	packaging/genchangelog.rb > $(DEB_BUILD_DIR)/debian/changelog
	# Copy inventory plugins and playbooks
	cp -a packaging/ansible/playbooks/* $(DEB_BUILD_DIR)/lib/playbooks/
	cp -a packaging/ansible/plugins/inventory/* $(DEB_BUILD_DIR)/lib/plugins/inventory/
	chmod 774 $(DEB_BUILD_DIR)/lib/plugins/inventory/*
	#sed -ri "s|%VERSION%|$(VERSION)|g;s|%RELEASE%|$(DEB_RELEASE)|;s|%DIST%|$(DEB_DIST)|g;" $(DEB_BUILD_DIR)/debian/control

# Create debian package
deb: debian
	cd $(DEB_BUILD_DIR) && $(DEBUILD) -b --lintian-opts --profile debian
	@echo "#############################################" 
	@echo "Tensor DEB artifacts:" 
	@echo build/deb-build/$(NAME)_$(VERSION)-$(DEB_RELEASE)~amd64.changes 
	@echo "#############################################" 

# Create debian sources package
deb-src: debian
	cd $(DEB_BUILD_DIR) && env GZIP=-9 tar --exclude='../../../build' -cvzf $(NAME)_$(VERSION)-2.orig.tar.gz ../../../
	cd $(DEB_BUILD_DIR) && $(DEBUILD) -S --lintian-opts --profile debian
	@echo "#############################################"
	@echo "Tensor DEB artifacts:"
	@echo build/deb-build/$(NAME)_$(VERSION)-$(DEB_RELEASE)~source.changes
	@echo "#############################################"

# Build tensor docker image and tag with current version
docker-build-image: deb
	cp build/deb-build/*.deb packaging/docker/tensor/tensor.deb
	cd packaging/docker/tensor/ && docker build -t gamunu/tensor:$(VERSION) -t gamunu/tensor:latest .

# Build using docker compose file located in packaging/docker/docker-compose.yml
docker-build: docker-build-image
	cp build/deb-build/*.deb packaging/docker/tensor/tensor.deb
	docker-compose -f packaging/docker/docker-compose.yml build

# Build and Spin-up containers using docker compose file located in packaging/docker/docker-compose.yml
docker-build-up: docker-build
	cp build/deb-build/*.deb packaging/docker/tensor/tensor.deb
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
	export PROJECTS_HOME=/data
	export TENSOR_PORT=8010
	export TENSOR_DB_USER=tensor
	export TENSOR_DB_PASSWORD=tensor
	export TENSOR_DB_NAME=tensordb
	export ENSOR_DB_REPLICA=""
	export TENSOR_DB_HOSTS="mongo:27017"
	export KRB5_CONFIG="/data/krb5.conf"
	reflex -r '\.go$$' -s -d none -- sh -c 'go run tensord/main.go'

# run test suite
test:
	go test ./...

# Vet examines Go source code and reports suspicious constructs
# http://godoc.org/code.google.com/p/go.tools/cmd/vet
# go get code.google.com/p/go.tools/cmd/vet
vet:
	go vet ./...
