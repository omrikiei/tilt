.PHONY: all proto install lint test test-go check-js test-js integration wire-check wire ensure check-go goimports

check-go: lint errcheck verify_goimports wire-check test-go
all: check-go check-js test-js

# There are 2 Go bugs that cause problems on CI:
# 1) Linker memory usage blew up in Go 1.11
# 2) Go incorrectly detects the number of CPUs when running in containers,
#    and sets the number of parallel jobs to the number of CPUs.
# This makes CI blow up frequently without out-of-memory errors.
# Manually setting the number of parallel jobs helps fix this.
# https://github.com/golang/go/issues/26186#issuecomment-435544512
GO_PARALLEL_JOBS := 4

SYNCLET_IMAGE := gcr.io/windmill-public-containers/tilt-synclet
SYNCLET_DEV_IMAGE_TAG_FILE := .synclet-dev-image-tag

CIRCLECI := $(if $(CIRCLECI),$(CIRCLECI),false)

GOIMPORTS_LOCAL_ARG := -local github.com/windmilleng/tilt

scripts/protocc/protocc.py: scripts/protocc
	git submodule init
	git submodule update

proto: scripts/protocc/protocc.py
	python3 scripts/protocc/protocc.py --out go

# Build a binary that uses synclet:latest
# TODO(nick): We should have a release build that bakes in a particular
# SYNCLET_IMAGE tag.
install:
	go install ./cmd/tilt/...

install-dev:
	@if ! [[ -e "$(SYNCLET_DEV_IMAGE_TAG_FILE)" ]]; then echo "No dev synclet found. Run make synclet-dev."; exit 1; fi
	go install -ldflags "-X 'github.com/windmilleng/tilt/internal/synclet/sidecar.SyncletTag=$$(<$(SYNCLET_DEV_IMAGE_TAG_FILE))'" ./...

# disable optimizations and inlining, to allow more complete information when attaching a debugger or capturing a profile
install-debug:
	go install -gcflags "all=-N -l" ./...

install-sail:
	go install ./cmd/sail/...

define synclet-build-dev
	echo $1 > $(SYNCLET_DEV_IMAGE_TAG_FILE)
	docker tag $(SYNCLET_IMAGE):dirty $(SYNCLET_IMAGE):$1
	docker push $(SYNCLET_IMAGE):$1
endef

synclet-dev: synclet-cache
	docker build --build-arg baseImage=synclet-cache -t $(SYNCLET_IMAGE):dirty -f synclet/Dockerfile .
	$(call synclet-build-dev,$(shell docker inspect $(SYNCLET_IMAGE):dirty -f '{{.Id}}' | sed -E 's/sha256:(.{20}).*/dirty-\1/'))

build-synclet-and-install: synclet-dev install-dev

lint:
	go vet -all -printfuncs=Verbosef,Infof,Debugf,PrintColorf ./...

build:
	go test -p $(GO_PARALLEL_JOBS) -timeout 60s ./... -run nonsenseregex

test-go:
ifneq ($(CIRCLECI),true)
		go test -p $(GO_PARALLEL_JOBS) -timeout 80s ./...
else
		mkdir -p test-results
		gotestsum --format standard-quiet --junitfile test-results/unit-tests.xml -- ./... -p $(GO_PARALLEL_JOBS) -timeout 80s
endif

test: test-go test-js

# skip some tests that are slow and not always relevant
shorttest:
	go test -p $(GO_PARALLEL_JOBS) -tags 'skipcontainertests' -timeout 60s ./...

integration:
ifneq ($(CIRCLECI),true)
		go test -p $(GO_PARALLEL_JOBS) -tags 'integration' -timeout 500s ./integration
else
		mkdir -p test-results
		gotestsum --format standard-quiet --junitfile test-results/unit-tests.xml -- ./integration -p $(GO_PARALLEL_JOBS) -tags 'integration' -timeout 600s
endif

dev-js:
	cd web && yarn install && yarn run start

check-js:
	cd web && yarn install
	cd web && yarn run check

build-js:
	cd web && yarn install
	cd web && yarn build

test-js:
	cd web && yarn install
	cd web && CI=true yarn test

ensure:
	dep ensure
	# Patch reflector until we can upgrade to k8s 1.16
	cp scripts/patch/reflector.go.txt vendor/k8s.io/client-go/tools/cache/reflector.go

goimports:
	goimports -w -l $(GOIMPORTS_LOCAL_ARG) $$(go list -f {{.Dir}} ./...)

verify_goimports:
	# any files printed here need to be formatted by `goimports $(GOIMPORTS_LOCAL_ARG)`
	bash -c 'diff <(goimports -l $(GOIMPORTS_LOCAL_ARG) $$(go list -f {{.Dir}} ./...)) <(echo -n)'

benchmark:
	go test -run=XXX -bench=. ./...

errcheck:
	errcheck -ignoretests -ignoregenerated ./...

timing: install
	./scripts/timing.py

WIRE_PATHS = engine cli synclet sail/client
wire:
	$(foreach path,$(WIRE_PATHS),wire ./internal/$(path) && goimports -w $(GOIMPORTS_LOCAL_ARG) internal/$(path) &&) true

wire-check:
	wire check ./internal/engine
	wire check ./internal/cli
	wire check ./internal/synclet
	wire check ./internal/sail/client

ci-container:
	docker build -t gcr.io/windmill-public-containers/tilt-ci -f .circleci/Dockerfile .circleci
	docker push gcr.io/windmill-public-containers/tilt-ci

ci-integration-container:
	docker build -t gcr.io/windmill-public-containers/tilt-integration-ci -f .circleci/Dockerfile.integration .circleci
	docker push gcr.io/windmill-public-containers/tilt-integration-ci

clean:
	go clean -cache -testcache -r -i ./...
	docker rmi synclet-cache

synclet-cache:
	if [ "$(shell docker images synclet-cache -q)" = "" ]; then \
		docker build -t synclet-cache -f synclet/Dockerfile --target=go-cache .; \
	fi;

synclet-release:
	$(eval TAG := $(shell date +v%Y%m%d))
	docker build -t $(SYNCLET_IMAGE):$(TAG) -f synclet/Dockerfile .
	docker push $(SYNCLET_IMAGE):$(TAG)
	sed -i 's/var SyncletTag = ".*"/var SyncletTag = "$(TAG)"/' internal/synclet/sidecar/sidecar.go

release:
	goreleaser --rm-dist

deploy-sail:
	$(eval TAG := $(shell date +built-%s))
	docker build -t gcr.io/windmill-public-containers/sail:$(TAG) -f deployments/sail.dockerfile .
	docker push gcr.io/windmill-public-containers/sail:$(TAG)
	cat deployments/sail.yaml | sed 's!gcr.io/windmill-public-containers/sail!gcr.io/windmill-public-containers/sail:$(TAG)!g' | kubectl apply -f -
	kubectl apply -f deployments/sail-networking.yaml

deploy-sail-staging:
	$(eval TAG := $(shell date +built-%s))
	docker build -t gcr.io/windmill-public-containers/sail-staging:$(TAG) -f deployments/sail-staging.dockerfile .
	docker push gcr.io/windmill-public-containers/sail-staging:$(TAG)
	cat deployments/sail-staging.yaml | sed 's!gcr.io/windmill-public-containers/sail-staging!gcr.io/windmill-public-containers/sail-staging:$(TAG)!g' | kubectl apply -f -
	kubectl apply -f deployments/sail-staging-networking.yaml

prettier:
	cd web && yarn install
	cd web && yarn run prettier --write "src/**/*.ts*"

