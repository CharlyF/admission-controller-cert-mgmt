# Setting SHELL to bash allows bash commands to be executed by recipes.
# This is a requirement for 'setup-envtest.sh' in the test target.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec
PLATFORM=$(shell uname -s)-$(shell uname -m)
GIT_TAG?=$(shell git tag -l --contains HEAD | tail -1)

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

##@ Development

.PHONY: all
all: test ## Build test

.PHONY: fmt
fmt: ## Run go fmt against code
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code
	go vet ./...

.PHONY: install-tools
install-tools: bin/$(PLATFORM)/golangci-lint

##@ Test

.PHONY: test
test: fmt vet verify-license gotest

.PHONY: gotest
gotest:
	go test ./... -coverprofile cover.out

.PHONY: tidy
tidy: ## Run go tidy
	go mod tidy -v

.PHONY: vendor
vendor: ## Run go vendor
	go mod vendor

.PHONY: lint
lint: vendor bin/$(PLATFORM)/golangci-lint fmt vet ## Lint
	bin/$(PLATFORM)/golangci-lint run ./...

.PHONY: verify-license
verify-license: bin/$(PLATFORM)/wwhrd vendor ## Verify licenses
	hack/verify-license.sh

.PHONY: license
license: bin/$(PLATFORM)/wwhrd vendor
	hack/license.sh

bin/$(PLATFORM)/wwhrd: Makefile
	hack/install-wwhrd.sh 0.2.4

bin/$(PLATFORM)/golangci-lint: Makefile
	hack/golangci-lint.sh -b "bin/$(PLATFORM)" v1.45.2