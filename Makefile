
# Image URL to use all building/pushing image targets
IMG ?= controller:latest
TEST_POD_IMG = sajid2045/phpload:1.1.1
# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
CRD_OPTIONS ?= "crd:trivialVersions=true"

# Image URL to use all building/pushing image targets
timestamp := $(shell /bin/date "+%Y%m%d-%H%M%S")
# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
KIND_CLUSTER_NAME ?= "hpa-tuner-controller"
K8S_NODE_IMAGE ?= v1.15.3
PROMETHEUS_INSTANCE_NAME ?= prometheus-operator
CONFIG_MAP_NAME ?= hpa-tuner-controller-configmap
METRICS_SERVER_IMG = k8s.gcr.io/metrics-server-amd64:v0.3.6

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

all: manager

# Run tests
test: generate fmt vet manifests
	go test ./... -coverprofile cover.out

# Build manager binary
manager: generate fmt vet
	go build -o bin/manager main.go

# Run against the configured Kubernetes cluster in ~/.kube/config
run: generate fmt vet manifests
	go run ./main.go

# Install CRDs into a cluster
install: manifests
	kustomize build config/crd | kubectl apply -f -

# DEPLOYING:
# # - Kind
# deploy-kind: kind-start kind-load-img deploy-cluster

kind-delete:
	kind delete cluster --name ${KIND_CLUSTER_NAME}
	sleep 5

kind-start:
ifeq (1, $(shell kind get clusters | grep ${KIND_CLUSTER_NAME} | wc -l))
	@echo "Cluster already exists"
else
	@echo "Creating Cluster"
	kind create cluster --name ${KIND_CLUSTER_NAME} --config test-data/kind/kind-cluster-1.14.10.yaml
endif

kind-load-metrics-server:
	docker pull k8s.gcr.io/metrics-server-amd64:v0.3.6
	kind load docker-image ${METRICS_SERVER_IMG} --name ${KIND_CLUSTER_NAME} || echo loaded
	kubectl apply  -f test-data/kind/metrics-server.yaml


kind-test-setup: kind-delete kind-start kind-load-metrics-server docker-build-phpload
	kind load docker-image ${TEST_POD_IMG} --name ${KIND_CLUSTER_NAME}
	kubectl apply  -f test-data/phpload/php-apache-application.yaml
	sleep 10
	kubectl get pods -A
	kubectl get hpa -A

kind-load-img: docker-build
	@echo "Loading image into kind"
	kind load docker-image ${IMG} --name ${KIND_CLUSTER_NAME} -v 10

# Run integration tests in KIND
kind-tests: 
#	ginkgo -v --skip="LONG TEST:" --nodes 6 --race --randomizeAllSpecs --cover --trace --progress --coverprofile ../controllers.coverprofile ./controllers
	ginkgo -v --skip="LONG TEST:" --cover --trace --progress --coverprofile ../controllers.coverprofile ./controllers

#Start your test with It("WIP:... and only that will be executed
focus-test:
	ginkgo -v -focus="WIP:" --cover --trace --progress --coverprofile ../controllers.coverprofile ./controllers

# Uninstall CRDs from a cluster
uninstall: manifests
	kustomize build config/crd | kubectl delete -f -


# Deploy controller in the configured Kubernetes cluster in ~/.kube/config
deploy: manifests
	cd config/manager && kustomize edit set image controller=${IMG}
	kustomize build config/default | kubectl apply -f -

# Generate manifests e.g. CRD, RBAC etc.
manifests: controller-gen
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases

# Run go fmt against code
fmt:
	go fmt ./...

# Run go vet against code
vet:
	go vet ./...

# Generate code
generate: controller-gen
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

# Build the docker image
docker-build: test
	docker build . -t ${IMG}

docker-build-phpload:
	docker build -t ${TEST_POD_IMG}  ./test-data/phpload

# Push the docker image
docker-push:
	docker push ${IMG}

help:
	@grep '^[^#[:space:]].*:' Makefile

# find or download controller-gen
# download controller-gen if necessary
controller-gen:
ifeq (, $(shell which controller-gen))
	@{ \
	set -e ;\
	CONTROLLER_GEN_TMP_DIR=$$(mktemp -d) ;\
	cd $$CONTROLLER_GEN_TMP_DIR ;\
	go mod init tmp ;\
	go get sigs.k8s.io/controller-tools/cmd/controller-gen@v0.2.5 ;\
	rm -rf $$CONTROLLER_GEN_TMP_DIR ;\
	}
CONTROLLER_GEN=$(GOBIN)/controller-gen
else
CONTROLLER_GEN=$(shell which controller-gen)
endif

