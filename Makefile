IMG ?= bob:latest

.PHONY: run
run:
	go run ./main.go

.PHONY: test
test:
	go test ./...

.PHONY: build
build:
	go build -o bin/bob-operator ./main.go

.PHONY: build-cli
build-cli:
	go build -o bin/bob ./cmd/bob

.PHONY: manifests
manifests:
	@echo "CRD manifests are maintained under config/crd/bases"

.PHONY: docker-build
docker-build:
	docker build -t ${IMG} .

.PHONY: docker-push
docker-push:
	docker push ${IMG}

.PHONY: install
install:
	kubectl apply -f config/crd/bases/

.PHONY: uninstall
uninstall:
	kubectl delete -f config/crd/bases/ --ignore-not-found

.PHONY: deploy
deploy: install
	kubectl apply -k config/default
