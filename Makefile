# build paramters
BUILD_FOLDER = dist
APP_VERSION = $(git describe --tags --always)

###############################################################################
###                           Basic Golang Commands                         ###
###############################################################################

all: install

install: go.sum
	go install cmd/endpoint-controller.go

install-debug: go.sum
	go build -gcflags="all=-N -l" -o $(BUILD_FOLDER)/endpoint-controller cmd/endpoint-controller.go

build: clean
	@echo build binary to $(BUILD_FOLDER)
	goreleaser build --single-target --config .goreleaser.yaml --snapshot --rm-dist
	@echo create deployment manifest
	kustomize build k8s > dist/bundle.yaml
	@echo done

clean:
	@echo clean build folder $(BUILD_FOLDER)
	rm -rf $(BUILD_FOLDER)
	@echo done

go.sum: go.mod
	@echo "--> Ensure dependencies have not been modified"
	GO111MODULE=on go mod verify

test:
	@go test -mod=readonly $(PACKAGES) -cover -race

lint:
	@echo "--> Running linter"
	@golangci-lint run 
	@go mod verify

