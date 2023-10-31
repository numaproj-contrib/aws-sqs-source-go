SHELL:=/bin/bash

PACKAGE=github.com/numaproj-contrib/aws-sqs-source-go
CURRENT_DIR=$(shell pwd)
DIST_DIR=${CURRENT_DIR}/dist
BINARY_NAME:=aws-sqs-source-go
IMAGE_NAMESPACE?=quay.io/numaproj
VERSION?=latest


DOCKERIO_ORG=quay.io/numaio
PLATFORMS=linux/amd64,linux/arm64
TARGET=aws-sqs-source-go

IMAGE_TAG=$(TAG)
ifeq ($(IMAGE_TAG),)
    IMAGE_TAG=latest
endif

.PHONY: build image lint clean test integ-test imagepush

build: clean
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -v -o ./dist/aws-sqs-source-go-amd64 main.go
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -v -o ./dist/aws-sqs-source-go-arm64 main.go

image: build
	docker buildx build --no-cache -t "$(DOCKERIO_ORG)/numaflow-go/aws-sqs-source-go:$(IMAGE_TAG)" --platform $(PLATFORMS) --target $(TARGET) . --load

lint:
	go mod tidy
	golangci-lint run --fix --verbose --concurrency 4 --timeout 5m

clean:
	-rm -rf ./dist

test:
	@echo "Running all tests..."
	@go test ./...

integ-test:
	@echo "Running integration tests..."
	@go test ./pkg/sqs -run Integ

imagepush: build
	docker buildx build --no-cache -t "$(DOCKERIO_ORG)/numaflow-go/aws-sqs-source-go:$(IMAGE_TAG)" --platform $(PLATFORMS) --target $(TARGET) . --push



.PHONY: dist/e2eapi
dist/e2eapi:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -v -o ${DIST_DIR}/e2eapi ./pkg/e2e/e2e-api

.PHONY: cleanup-e2e
cleanup-e2e:
	kubectl -n numaflow-system delete svc -lnumaflow-e2e=true --ignore-not-found=true
	kubectl -n numaflow-system delete sts -lnumaflow-e2e=true --ignore-not-found=true
	kubectl -n numaflow-system delete deploy -lnumaflow-e2e=true --ignore-not-found=true
	kubectl -n numaflow-system delete cm -lnumaflow-e2e=true --ignore-not-found=true
	kubectl -n numaflow-system delete secret -lnumaflow-e2e=true --ignore-not-found=true
	kubectl -n numaflow-system delete po -lnumaflow-e2e=true --ignore-not-found=true

.PHONY: test-e2e
test-e2e:
	$(MAKE) cleanup-e2e
	$(MAKE) e2eapi-image
	kubectl -n numaflow-system delete po -lapp.kubernetes.io/component=controller-manager,app.kubernetes.io/part-of=numaflow
	kubectl -n numaflow-system delete po e2e-api-pod  --ignore-not-found=true
	cat pkg/e2e/manifests/e2e-api-pod.yaml |  sed 's@quay.io/numaproj/@$(IMAGE_NAMESPACE)/@' | sed 's/:latest/:$(VERSION)/' | kubectl -n numaflow-system apply -f -
	go test -v -timeout 15m -count 1 --tags test -p 1 ./test/e2e
	$(MAKE) cleanup-e2e

.PHONY: e2eapi-image
e2eapi-image: clean dist/e2eapi
	DOCKER_BUILDKIT=1 $(DOCKER) build . --build-arg "ARCH=amd64" --target e2eapi --tag $(IMAGE_NAMESPACE)/e2eapi:$(VERSION) --build-arg VERSION="$(VERSION)"

clean:
	-rm -rf ${CURRENT_DIR}/dist

