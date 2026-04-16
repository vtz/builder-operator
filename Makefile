IMG ?= bob:latest
GOLANGCI_LINT ?= golangci-lint

.PHONY: all
all: fmt vet lint test build

## Development

.PHONY: run
run:
	go run ./main.go

.PHONY: fmt
fmt:
	go fmt ./...

.PHONY: vet
vet:
	go vet ./...

.PHONY: lint
lint:
	$(GOLANGCI_LINT) run

.PHONY: lint-fix
lint-fix:
	$(GOLANGCI_LINT) run --fix

.PHONY: test
test: fmt vet
	go test ./... -count=1 -race -coverprofile=coverage.out
	@echo "Coverage report: coverage.out"

.PHONY: test-unit
test-unit:
	go test ./... -count=1 -race -short

## Build

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X main.version=$(VERSION)

.PHONY: build
build:
	go build -o bin/bob-operator ./main.go

.PHONY: build-cli
build-cli:
	go build -ldflags "$(LDFLAGS)" -o bin/bob ./cmd/bob

.PHONY: dist
dist:
	@mkdir -p dist
	GOOS=darwin  GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o dist/bob-darwin-amd64  ./cmd/bob
	GOOS=darwin  GOARCH=arm64 CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o dist/bob-darwin-arm64  ./cmd/bob
	GOOS=linux   GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o dist/bob-linux-amd64   ./cmd/bob
	GOOS=linux   GOARCH=arm64 CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o dist/bob-linux-arm64   ./cmd/bob
	@echo ""
	@echo "Binaries:"
	@ls -lh dist/bob-*
	@echo ""
	@echo "Checksums:"
	@cd dist && shasum -a 256 bob-* | tee SHA256SUMS

## Manifests

.PHONY: manifests
manifests:
	@echo "CRD manifests are maintained under config/crd/bases"

.PHONY: generate
generate:
	@echo "DeepCopy methods are maintained under api/v1alpha1/zz_generated.deepcopy.go"

## Container

.PHONY: docker-build
docker-build:
	docker build -t ${IMG} .

.PHONY: docker-push
docker-push:
	docker push ${IMG}

## Deploy

.PHONY: install
install:
	kubectl apply -f config/crd/bases/

.PHONY: uninstall
uninstall:
	kubectl delete -f config/crd/bases/ --ignore-not-found

.PHONY: deploy
deploy: install
	kubectl apply -k config/default

.PHONY: undeploy
undeploy:
	kubectl delete -k config/default --ignore-not-found
	$(MAKE) uninstall
