SHELL:=/bin/bash

PACKAGE=github.com/numaproj-contrib/aws-sqs-source-go
CURRENT_DIR=$(shell pwd)
DIST_DIR=${CURRENT_DIR}/dist
BINARY_NAME:=aws-sqs-source-go
IMAGE_NAMESPACE?=quay.io/numaio/numaflow-go
VERSION?=latest

DOCKER_PUSH?=true
BASE_VERSION:=latest
DOCKERIO_ORG=quay.io/numaio
PLATFORMS=linux/x86_64
TARGET=aws-sqs-source-go

IMAGE_TAG=$(TAG)
ifeq ($(IMAGE_TAG),)
    IMAGE_TAG=latest
endif

DOCKER:=$(shell command -v docker 2> /dev/null)
ifndef DOCKER
DOCKER:=$(shell command -v podman 2> /dev/null)
endif

CURRENT_CONTEXT:=$(shell [[ "`command -v kubectl`" != '' ]] && kubectl config current-context 2> /dev/null || echo "unset")
IMAGE_IMPORT_CMD:=$(shell [[ "`command -v k3d`" != '' ]] && [[ "$(CURRENT_CONTEXT)" =~ k3d-* ]] && echo "k3d image import -c `echo $(CURRENT_CONTEXT) | cut -c 5-`")
ifndef IMAGE_IMPORT_CMD
IMAGE_IMPORT_CMD:=$(shell [[ "`command -v minikube`" != '' ]] && [[ "$(CURRENT_CONTEXT)" =~ minikube* ]] && echo "minikube image load")
endif
ifndef IMAGE_IMPORT_CMD
IMAGE_IMPORT_CMD:=$(shell [[ "`command -v kind`" != '' ]] && [[ "$(CURRENT_CONTEXT)" =~ kind-* ]] && echo "kind load docker-image")
endif

.PHONY: build image lint clean test imagepush install-numaflow

build: clean
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -v -o ./dist/aws-sqs-source-go-amd64 main.go
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -v -o ./dist/aws-sqs-source-go-arm64 main.go

image: build
	docker buildx build --no-cache -t "$(DOCKERIO_ORG)/numaflow-go/aws-sqs-source-go:$(IMAGE_TAG)" --platform $(PLATFORMS) --target $(TARGET) . --load

lint:
	go mod tidy
	golangci-lint run --fix --verbose --concurrency 4 --timeout 5m

test:
	@echo "Running integration tests..."
	@go test -race ./pkg/sqs/*

imagepush: build
	docker buildx build --no-cache -t "$(DOCKERIO_ORG)/numaflow-go/aws-sqs-source-go:$(IMAGE_TAG)" --platform $(PLATFORMS) --target $(TARGET) . --push

.PHONY: dist/e2eapi
dist/e2eapi:
	ls -al ./pkg/e2e/  # Diagnostic command to list contents
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -v -o ${DIST_DIR}/e2eapi ./pkg/e2e/e2e-api

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
	cat pkg/e2e/manifests/e2e-api-pod.yaml |  sed 's@quay.io/numaio/numaflow-go/@$(IMAGE_NAMESPACE)/@' | sed 's/:latest/:$(VERSION)/' | kubectl -n numaflow-system apply -f -
	go generate $(shell find ./pkg/e2e/test$* -name '*.go')
	go test -v -timeout 15m -count 1 --tags test -p 1 ./pkg/e2e/test/sqs_e2e_test.go
	$(MAKE) cleanup-e2e

.PHONY: e2eapi-image
e2eapi-image: clean dist/e2eapi
				 DOCKER_BUILDKIT=1 $(DOCKER) build . --build-arg "ARCH=amd64" --target e2eapi --tag $(IMAGE_NAMESPACE)/e2eapi:$(VERSION)  --platform $(PLATFORMS) --build-arg VERSION="$(VERSION)"

clean:
	-rm -rf ${CURRENT_DIR}/dist

install-numaflow:
	kubectl create ns numaflow-system
	kubectl apply -n numaflow-system -f https://raw.githubusercontent.com/numaproj/numaflow/stable/config/install.yaml




