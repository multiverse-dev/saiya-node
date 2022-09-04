BRANCH = "master"
BINARY=./bin/saiya$(shell go env GOEXE)
REPO ?= "$(shell go list -m)"
VERSION ?= "$(shell git describe --tags 2>/dev/null | sed 's/^v//')"
MODVERSION ?= "$(shell cat go.mod | cat go.mod | sed -r -n -e 's|.*pkg/interop (.*)|\1|p')"
BUILD_FLAGS = "-X '$(REPO)/pkg/config.Version=$(VERSION)'"

# All of the targets are phony here because we don't really use make dependency
# tracking for files
.PHONY: build deps push-tag test vet lint fmt cover
	
build: deps
	@echo "=> Building binary"
	@set -x \
		&& export GOGC=off \
		&& export CGO_ENABLED=0 \
		&& go build -trimpath -v -ldflags $(BUILD_FLAGS) -o ${BINARY} ./cli/main.go

check-version:
	git fetch && (! git rev-list ${VERSION})

deps:
	cd zk && \
	cargo build --release
	cp -f zk/target/release/libzk.dylib pkg/crypto/zk/lib
	@CGO_ENABLED=0 \
	go mod download
	@CGO_ENABLED=0 \
	go mod tidy -v

push-tag:
	git checkout ${BRANCH}
	git pull origin ${BRANCH}
	git tag ${VERSION}
	git push origin ${VERSION}

test:
	@go test ./... -cover

vet:
	@go vet ./...

lint:
	@golangci-lint run

fmt:
	@gofmt -l -w -s $$(find . -type f -name '*.go'| grep -v "/vendor/")

cover:
	@go test -v -race ./... -coverprofile=coverage.txt -covermode=atomic -coverpkg=./pkg/...,./cli/...
	@go tool cover -html=coverage.txt -o coverage.html
