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
