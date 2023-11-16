# Ensure Make is run with bash shell as some syntax below is bash-specific
SHELL:=/usr/bin/env bash

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif
GO_INSTALL := ./scripts/go_install.sh

REGISTRY ?= projectsveltos
IMAGE_NAME ?= k8s-collector
export COLLECTOR_IMG ?= $(REGISTRY)/$(IMAGE_NAME)
TAG ?= main
ARCH ?= amd64
OS ?= $(shell uname -s | tr A-Z a-z)
K8S_LATEST_VER ?= $(shell curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt)

# Directories.
ROOT_DIR:=$(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))
TOOLS_DIR := hack/tools
BIN_DIR := bin
TOOLS_BIN_DIR := $(abspath $(TOOLS_DIR)/$(BIN_DIR))

GOBUILD=go build

GENERATED_FILES:=./manifest/manifest.yaml

## Tool Binaries
GOLANGCI_LINT := $(TOOLS_BIN_DIR)/golangci-lint
GOIMPORTS := $(TOOLS_BIN_DIR)/goimports
GINKGO := $(TOOLS_BIN_DIR)/ginkgo
KUBECTL := $(TOOLS_BIN_DIR)/kubectl
SETUP_ENVTEST := $(TOOLS_BIN_DIR)/setup_envs
CONTROLLER_GEN := $(TOOLS_BIN_DIR)/controller-gen
KIND := $(TOOLS_BIN_DIR)/kind

GOLANGCI_LINT_VERSION := "v1.52.2"


$(CONTROLLER_GEN): $(TOOLS_DIR)/go.mod # Build controller-gen from tools folder.
	cd $(TOOLS_DIR); $(GOBUILD) -tags=tools -o $(subst $(TOOLS_DIR)/hack/tools/,,$@) sigs.k8s.io/controller-tools/cmd/controller-gen

$(GOLANGCI_LINT): # Build golangci-lint from tools folder.
	cd $(TOOLS_DIR); ./get-golangci-lint.sh $(GOLANGCI_LINT_VERSION)

$(SETUP_ENVTEST): $(TOOLS_DIR)/go.mod # Build setup-envtest from tools folder.
	cd $(TOOLS_DIR); $(GOBUILD) -tags=tools -o $(subst $(TOOLS_DIR)/hack/tools/,,$@) sigs.k8s.io/controller-runtime/tools/setup-envtest

$(GOIMPORTS):
	cd $(TOOLS_DIR); $(GOBUILD) -tags=tools -o $(subst $(TOOLS_DIR)/hack/tools/,,$@) golang.org/x/tools/cmd/goimports

$(GINKGO): $(TOOLS_DIR)/go.mod
	cd $(TOOLS_DIR) && $(GOBUILD) -tags tools -o $(subst $(TOOLS_DIR)/hack/tools/,,$@) github.com/onsi/ginkgo/v2/ginkgo

$(KIND): $(TOOLS_DIR)/go.mod
	cd $(TOOLS_DIR) && $(GOBUILD) -tags tools -o $(subst $(TOOLS_DIR)/hack/tools/,,$@) sigs.k8s.io/kind

$(KUBECTL):
	curl -L https://storage.googleapis.com/kubernetes-release/release/$(K8S_LATEST_VER)/bin/$(OS)/$(ARCH)/kubectl -o $@
	chmod +x $@


.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Tools

.PHONY: tools
tools: $(GOLANGCI_LINT) $(GOIMPORTS) $(GINKGO) $(KUBECTL)  $(SETUP_ENVTEST) $(CONTROLLER_GEN) ## build all tools

.PHONY: clean
clean: ## Remove all built tools
	rm -rf $(TOOLS_BIN_DIR)/*

##@ generate

.PHONY: generate-modules
generate-modules: ## Run go mod tidy to ensure modules are up to date
	go mod tidy
	cd $(TOOLS_DIR); go mod tidy

set-manifest-image:
	sed -i'' -e 's@image: .*@image: '"${MANIFEST_IMG}:$(MANIFEST_TAG)"'@' ./manifest/manifest.yaml

##@ docker
PKEY ?= id_rsa

.PHONY: docker-build
docker-build: ## Build the docker image for k8s-collector
	docker build -t $(REGISTRY)/$(IMAGE_NAME)-$(ARCH):$(TAG) -f Dockerfile . 

##@ Build

.PHONY: vet
vet: ## Run go vet against code
	go vet ./...

.PHONY: fmt
fmt goimports: $(GOIMPORTS) ## Format and adjust import modules.
	$(GOIMPORTS) -local github.com/projectsveltos -w .

.PHONY: lint
lint: $(GOLANGCI_LINT) ## Lint codebase
	$(GOLANGCI_LINT) run -v --fast=false --max-issues-per-linter 0 --max-same-issues 0 --timeout 5m	

.PHONY: build
build: fmt vet ## Build manager binary.
	 go build -o $(BIN_DIR)/k8s-collector cmd/main.go

##@ Testing

# KUBEBUILDER_ENVTEST_KUBERNETES_VERSION refers to the version of kubebuilder assets to be downloaded by envtest binary.
KUBEBUILDER_ENVTEST_KUBERNETES_VERSION = 1.28.0

ifeq ($(shell go env GOOS),darwin) # Use the darwin/amd64 binary until an arm64 version is available
KUBEBUILDER_ASSETS ?= $(shell $(SETUP_ENVTEST) use --use-env -p path --arch amd64 $(KUBEBUILDER_ENVTEST_KUBERNETES_VERSION))
else
KUBEBUILDER_ASSETS ?= $(shell $(SETUP_ENVTEST) use --use-env -p path $(KUBEBUILDER_ENVTEST_KUBERNETES_VERSION))
endif

.PHONY: test
test: | fmt vet $(SETUP_ENVTEST) ## Run uts.
	KUBEBUILDER_ASSETS="$(KUBEBUILDER_ASSETS)" go test $(shell go list ./... |grep -v test/fv |grep -v pkg/deployer/fake |grep -v test/helpers) $(TEST_ARGS) -coverprofile cover.out 

KIND_CONFIG ?= kind-cluster.yaml
CONTROL_CLUSTER_NAME ?= sveltos-management

# K8S_VERSION for the Kind cluster can be set as environment variable. If not defined,
# this default value is used
ifndef K8S_VERSION
K8S_VERSION := v1.28.0
endif


.PHONY: load-image
load-image: docker-build $(KIND)
	$(KIND) load docker-image $(COLLECTOR_IMG)-$(ARCH):$(TAG) --name $(CONTROL_CLUSTER_NAME)

.PHONY: create-cluster
create-cluster: $(KIND) $(KUBECTL) ## Create a new kind cluster designed for development
	sed -e "s/K8S_VERSION/$(K8S_VERSION)/g"  test/$(KIND_CONFIG) > test/$(KIND_CONFIG).tmp
	$(KIND) create cluster --name=$(CONTROL_CLUSTER_NAME) --config test/$(KIND_CONFIG).tmp

	@echo "apply PersistentVolumeClaim"
	$(KUBECTL) apply -f test/pvc.yaml

	$(KUBECTL) apply -f test/configmap.yaml

	@echo 'Load k8s-collector image into cluster'
	$(MAKE) load-image
	$(KUBECTL) apply -f test/collector.yaml

.PHONY: delete-cluster
delete-cluster: $(KIND) ## Deletes the kind cluster $(CONTROL_CLUSTER_NAME)
	$(KIND) delete cluster --name $(CONTROL_CLUSTER_NAME)

.PHONY: kind-test
kind-test: test create-cluster fv ## Build docker image; start kind cluster; load docker image; install all cluster api components and run fv

.PHONY: fv
fv: $(KUBECTL) $(GINKGO) ## Run Sveltos Controller tests using existing cluster
	cd test/fv; $(GINKGO) -nodes 1 --label-filter='FV' --v --trace --randomize-all