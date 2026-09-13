IMAGE        := lynxzp/printloop
PLATFORM     := linux/amd64
LINTER_IMAGE := golangci/golangci-lint:v2.13.2-alpine

.DEFAULT_GOAL := all

test:
	go test -race -count=1 ./...

lint:
	docker run --rm \
		-v $(CURDIR):/app \
		-v printloop-lint-cache:/root/.cache \
		-w /app \
		$(LINTER_IMAGE) golangci-lint run

build: test lint
	docker build --platform $(PLATFORM) -t $(IMAGE) .

push: build
	docker push $(IMAGE)

all: push

.PHONY: test lint build push all
