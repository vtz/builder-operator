IMG ?= controller:latest

.PHONY: run
run:
	go run ./main.go

.PHONY: test
test:
	go test ./...

.PHONY: manifests
manifests:
	@echo "CRD manifests are maintained under config/crd/bases"

.PHONY: docker-build
docker-build:
	docker build -t ${IMG} .

.PHONY: install
install:
	kubectl apply -f config/crd/bases/build.mycompany.io_softwarebuilds.yaml

.PHONY: uninstall
uninstall:
	kubectl delete -f config/crd/bases/build.mycompany.io_softwarebuilds.yaml --ignore-not-found
