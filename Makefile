REGISTRY ?= docker.io/brudnak
IMAGE_NAME ?= vai-sidecar
VERSION ?= latest

# Always build for amd64 since that's what Rancher is running
DOCKER_PLATFORM := linux/amd64

.PHONY: build
build:
	DOCKER_DEFAULT_PLATFORM=$(DOCKER_PLATFORM) docker build -t $(REGISTRY)/$(IMAGE_NAME):$(VERSION) .

.PHONY: push
push: build
	docker push $(REGISTRY)/$(IMAGE_NAME):$(VERSION)

.PHONY: run
run:
	go run main.go

.PHONY: test-local
test-local:
	curl -s http://localhost:8080/health

# Convenience targets
.PHONY: all
all: build push

.PHONY: clean
clean:
	docker rmi $(REGISTRY)/$(IMAGE_NAME):$(VERSION) || true