IMAGE    := lynxzp/printloop
PLATFORM := linux/amd64

.DEFAULT_GOAL := all

test:
	go test -race ./...

lint:
	golangci-lint run

build: test lint
	docker build --platform $(PLATFORM) -t $(IMAGE) .

push: build
	docker push $(IMAGE)

all: push

.PHONY: test lint build push all
