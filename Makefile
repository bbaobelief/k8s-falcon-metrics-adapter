REGISTRY?=192.168.117.6
IMAGE?=test/k8s-falcon-metrics-adapter
VERSION?=latest
DOCKER_USER?=admin

ifneq ("$(REGISTRY)", "")
	FULL_IMAGE=$(REGISTRY)/$(IMAGE)
else
	FULL_IMAGE=$(IMAGE)
endif

OUT_DIR?=./_output
SEMVER=""
PUSH_LATEST=true

BRANCH=$(shell git rev-parse --abbrev-ref HEAD)

.PHONY: all build-local build vendor test version push verify-deploy gen-deploy dev save tag-ci

all: build push

build-local: test
	CGO_ENABLED=0 go build -a -tags netgo -o $(OUT_DIR)/adapter github.com/bbaobelief/k8s-falcon-metrics-adapter

build: vendor verify-apis
	docker build -t $(FULL_IMAGE):$(VERSION) .

vendor:
	go mod vendor

test: vendor
	# hack/run-tests.sh

version: build
ifeq ("$(SEMVER)", "")
	@echo "Please set sem version bump: can be 'major', 'minor', or 'patch'"
	exit
endif
ifeq ("$(BRANCH)", "master")
	@echo "versioning on master"
	gitsem $(SEMVER)
else
	@echo "must be on clean master branch"
endif

push:
ifdef DOCKER_PASS
	# non interactive login
	@echo $(DOCKER_PASS) | docker login -u $(DOCKER_USER) --password-stdin $(REGISTRY)
else
	# interactive login (needed for WSL)
	docker login -u $(DOCKER_USER) $(REGISTRY)
endif
	docker push $(FULL_IMAGE):$(VERSION)
ifeq ("$(PUSH_LATEST)", "true")
	@echo "pushing to latest"
	docker tag $(FULL_IMAGE):$(VERSION) $(FULL_IMAGE):latest
	docker push $(FULL_IMAGE):latest
endif

# dev setup
dev:
	skaffold dev

# CI specific commands used during CI build
save:
	docker save -o app.tar $(FULL_IMAGE):$(VERSION)

tag-ci:
	docker tag $(FULL_IMAGE):$(CIRCLE_WORKFLOW_ID) $(FULL_IMAGE):$(VERSION)

# Code gen helpers
gen-apis: vendor
	hack/update-codegen.sh

verify-apis: vendor
	hack/verify-codegen.sh
