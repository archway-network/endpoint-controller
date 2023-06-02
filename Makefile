# build parameters
BUILD_FOLDER = dist
APP_VERSION = $(git describe --tags --always)
PACKAGES = $(go list ./...)
GOLANG_VERSION = 1.20.4
PACKAGE_NAME = github.com/archway-network/endpoint-controller
DOCKER := $(shell which docker)

all: install

install: go.sum
	go install cmd/endpoint-controller.go

install-debug: go.sum
	go build -gcflags="all=-N -l" -o $(BUILD_FOLDER)/endpoint-controller cmd/endpoint-controller.go

build: clean
	@echo build binary to $(BUILD_FOLDER)
	goreleaser build --single-target --config .goreleaser.yaml --snapshot --clean
	@echo create deployment manifest
	kustomize build k8s > dist/bundle.yaml
	@echo done

release:
	$(DOCKER) run \
		--rm \
		-v $(HOME)/.docker/config.json:/root/.docker/config.json \
		-v /var/run/docker.sock:/var/run/docker.sock \
		-v `pwd`:/go/src/$(PACKAGE_NAME) \
		-e GITHUB_TOKEN="$(GITHUB_TOKEN)" \
		-w /go/src/$(PACKAGE_NAME) \
		ghcr.io/goreleaser/goreleaser-cross:v$(GOLANG_VERSION) \
		--clean \

clean:
	@echo clean build folder $(BUILD_FOLDER)
	rm -rf $(BUILD_FOLDER)
	@echo done

go.sum: go.mod
	@echo "--> Ensure dependencies have not been modified"
	GO111MODULE=on go mod verify

tests-ci:
	@go get ./...
	@go test ./...

test:
	@go test ./...

test_coverage:
	@go test ./... -coverprofile=coverage.out

vet:
	@go vet

lint:
	@echo "--> Running linter"
	@golangci-lint run 
	@go mod verify

