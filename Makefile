# ==================================================================================
# Copyright © 2026 Sean Beard - All Rights Reserved
# Unauthorized copying of this file, via any medium is strictly prohibited
# Proprietary and Confidential
# ==================================================================================
SHELL := /bin/bash
LOCALDIR := $(`pwd')
MAKEFILEPATH := $(abspath $(lastword $(MAKEFILE_LIST)))
LOCALDIR := $(dir $(MAKEFILEPATH))
BINARY := dirsynctos3

# ==============================================================================
# Building binary
all:

build-linux: clean
	docker build \
	-f resources/docker/Dockerfile.amd64 \
	-t dirsynctos3-interim:build \
	--build-arg VCS_REF=`git rev-parse HEAD` \
	--build-arg BUILD_DATE=`date -u +"%Y-%m-%dT%H:%M:%SZ"` \
	--build-arg VERSION=0.0.1 \
	--build-arg GO_OS=linux \
	--build-arg GO_ARCH=amd64 \
	--build-arg BINARY=$(BINARY) \
	.
	docker run --rm -it -v $(LOCALDIR):/build dirsynctos3-interim:build
	sudo chown krone:krone $(LOCALDIR)$(BINARY)
	docker rmi dirsynctos3-interim:build

build-win: clean
	export GOWORK=off && \
	go mod tidy && \
	go mod vendor && \
	docker build \
	-f resources/docker/Dockerfile.amd64 \
	-t dirsynctos3-interim:build \
	--build-arg VCS_REF=`git rev-parse HEAD` \
	--build-arg BUILD_DATE=`date -u +"%Y-%m-%dT%H:%M:%SZ"` \
	--build-arg VERSION=0.0.1 \
	--build-arg GO_OS=windows \
	--build-arg GO_ARCH=amd64 \
	--build-arg BINARY=$(BINARY).exe \
	.
	docker run --rm -it -v $(LOCALDIR):/build dirsynctos3-interim:build
	sudo chown krone:krone $(LOCALDIR)$(BINARY).exe
	docker rmi dirsynctos3-interim:build

clean:
	$(info Cleaning...)
ifneq ("$(wildcard $(LOCALDIR)$(BINARY)*)","")
	rm $(LOCALDIR)$(BINARY)*
endif

# ==============================================================================
# Build
#all:

#build-amd64:
#	docker build \
	-f resources/docker/Dockerfile.unio.amd64 \
	-t unio-amd64:1.0 \
	--build-arg VCS_REF=`git rev-parse HEAD` \
	--build-arg BUILD_DATE=`date -u +"%Y-%m-%dT%H:%M:%SZ"` \
	--build-arg VERSION=0.0.1 \
	.
#	docker image prune -f

# ==================================================================================
# Modules support

deps-reset: ## Reset the project Go dependencies based on what is in the respository
	git checkout -- go.mod
	go mod tidy
	@if [ -f "${ROOTPATH}go.work" ]; then \
		go work vendor; \
	else \
		go mod vendor; \
	fi

tidy: ## Update the project Go dependencies
	go mod tidy
	@echo "Adding dependencies..."
	@if [ -f "${ROOTPATH}go.work" ]; then \
		go work vendor; \
	else \
		go mod vendor; \
	fi

deps-upgrade: ## Upgrade the project Go dependencies to the latest versions
	go get -u -t -d -v ./...
	go mod tidy
	@if [ -f "${ROOTPATH}go.work" ]; then \
		go work vendor; \
	else \
		go mod vendor; \
	fi

deps-cleancache: ## Clean the Go dependency cache
	go clean -modcache

# ==================================================================================

about: ## Display info related to the build
	@echo "OS: $(OS)"
	@echo "Shell: $(SHELL) $(SHELL_VERSION)"
	@echo "Protoc version: $(shell protoc --version)"
	@echo "Go version: $(shell go version)"
	@echo "Go package: $(PACKAGE)"
	@echo "Openssl version: $(shell openssl version)"

help: ## Show this help
	@${HELP_CMD}

# ==================================================================================
