BINARY=monaco
VERSION=2.x
RELEASES = $(BINARY)-windows-amd64.exe $(BINARY)-windows-386.exe $(BINARY)-linux-amd64 $(BINARY)-linux-386 $(BINARY)-darwin-amd64 $(BINARY)-darwin-arm64

.PHONY: lint format mocks build install clean test integration-test integration-test-v1 test-package default add-license-headers build-release RELEASES

default: build

setup:
	@echo "Installing build tools..."
	@go install github.com/google/addlicense@latest
	@go install gotest.tools/gotestsum@latest
	@go install github.com/golang/mock/mockgen@latest

lint: setup
ifeq ($(OS),Windows_NT)
	@.\tools\check-format.cmd
else
	@go install github.com/google/addlicense@v1
	@sh ./tools/check-format.sh
	@sh ./tools/check-license-headers.sh
	@go mod tidy
endif

format:
	@gofmt -w .

add-license-headers:
ifeq ($(OS),Windows_NT)
	@echo "This is currently not supported on windows"
	@exit 1
else
	@sh ./tools/add-missing-license-headers.sh
endif

mocks: setup
	@echo "Generating mocks"
	@go generate ./...

vet: mocks
	@echo "Vetting files"
	@go vet ./...

check:
	@echo "Static code analysis"
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1
	@golangci-lint run ./...

build: clean mocks
	@echo "Building ${BINARY}..."
	@CGO_ENABLED=0 go build -a -tags netgo -ldflags '-w -extldflags "-static"' -o ./bin/${BINARY} ./cmd/monaco

build-release: clean $(RELEASES)
	@echo Release build ${BINARY} ${VERSION}

argument = $(word $1, $(subst -,$(empty) $(empty), $(subst .exe,$(empty) $(empty) , $2)))
OUTDIR ?= ./build
$(RELEASES):
	GOOS=$(call argument, 2, $@) GOARCH=$(call argument, 3, $@) CGO_ENABLED=0 go build -a -tags netgo -ldflags '-X github.com/dynatrace/dynatrace-configuration-as-code/pkg/version.MonitoringAsCode=${VERSION} -w -extldflags "-static"' -o ${OUTDIR}/$@ ./cmd/monaco


install:
	@echo "Installing ${BINARY}..."
	@CGO_ENABLED=0 go install -a -tags netgo -ldflags '-w -extldflags "-static"' ./cmd/monaco

run:
	@CGO_ENABLED=0 go run -a -tags netgo -ldflags '-w -extldflags "-static"' ./cmd/monaco

clean:
	@echo "Removing ${BINARY}, bin/ and /build ..."
ifeq ($(OS),Windows_NT)
	@if exist bin rd /S /Q bin
	@if exist bin rd /S /Q build
else
	@rm -rf bin/
	@rm -rf build/
endif

test: setup mocks lint
	@echo "Testing ${BINARY}..."
	@gotestsum ${testopts} -- -tags=unit -v -race ./...

integration-test: mocks
	@gotestsum ${testopts} --format standard-verbose -- -tags=integration -timeout=30m -v -race ./...

integration-test-v1: mocks
	@gotestsum ${testopts} --format standard-verbose -- -tags=integration_v1 -timeout=30m -v -race ./...

download-restore-test: mocks
	@gotestsum ${testopts} --format standard-verbose -- -tags=download_restore -timeout=30m -v -race ./...

clean-environments:
	@gotestsum ${testopts} --format standard-verbose -- -tags=cleanup -v -race ./...

nightly-test:mocks
	@gotestsum ${testopts} --format standard-verbose -- -tags=nightly -timeout=60m -v -race ./...

# Build and Test a single package supplied via pgk variable, without using test cache
# Run as e.g. make test-package pkg=project
pkg=...
test-package: setup mocks lint
	@echo "Testing ${pkg}..."
	@gotestsum -- -tags=unit -count=1 -v -race ./pkg/${pkg}

update-dependencies:
	@echo Update go dependencies
	@go get -u ./...
	@go mod tidy
