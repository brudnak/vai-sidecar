# ------------ configurable -------------
REGISTRY     ?= docker.io/brudnak
IMAGE_NAME   ?= vai-sidecar
VERSION      ?= latest
PLATFORM     := linux/amd64

# ------------ build / push -------------
.PHONY: build
build:
	DOCKER_DEFAULT_PLATFORM=$(PLATFORM) \
	docker build -t $(REGISTRY)/$(IMAGE_NAME):$(VERSION) .

.PHONY: push
push: build
	docker push $(REGISTRY)/$(IMAGE_NAME):$(VERSION)

# ------------ local run ---------------
.PHONY: run
run:
	go run main.go

.PHONY: test-local
test-local: run &
	@sleep 1
	@curl -s http://localhost:8080/health && echo " -> HEALTHY"

# ------------ clean -------------------
.PHONY: clean
clean:
	-docker rmi $(REGISTRY)/$(IMAGE_NAME):$(VERSION) 2>/dev/null || true
