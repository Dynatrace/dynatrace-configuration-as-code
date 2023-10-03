BINARY_NAME ?= monaco
VERSION ?= 2.x
RELEASES = $(BINARY_NAME)-windows-amd64.exe $(BINARY_NAME)-windows-386.exe $(BINARY_NAME)-linux-arm64 $(BINARY_NAME)-linux-amd64 $(BINARY_NAME)-linux-386 $(BINARY_NAME)-darwin-amd64 $(BINARY_NAME)-darwin-arm64

.PHONY: lint format mocks build install clean test integration-test integration-test-v1 test-package default add-license-headers compile build-release $(RELEASES) docker-container sign-image

default: build

setup:
	@echo "Installing build tools..."
	@go install github.com/google/addlicense@v1.1.1
	@go install gotest.tools/gotestsum@v1.11.0
	@go install go.uber.org/mock/mockgen@v0.2.0
	@go install github.com/sigstore/cosign/v2/cmd/cosign@v2.1.1

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
	@go vet -tags '!unit' ./...

check:
	@echo "Static code analysis"
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1
	@golangci-lint run ./...

compile: mocks
	@echo "Compiling sources..."
	@go build -tags "unit integration nightly cleanup integration_v1 download_restore" ./...
	@echo "Compiling tests..."
	@go test -tags "unit integration nightly cleanup integration_v1 download_restore" -run "NON_EXISTENT_TEST_TO_ENSURE_NOTHING_RUNS_BUT_ALL_COMPILE" ./...

build: mocks
	@echo "Building $(BINARY_NAME)..."
	@CGO_ENABLED=0 go build -a -tags netgo -ldflags '-w -extldflags "-static"' -o ./bin/${BINARY_NAME} ./cmd/monaco

build-release: clean $(RELEASES)
	@echo Release build $(BINARY_NAME) $(VERSION)

#argument - splits the name of command to array and return required element
argument = $(word $1, $(subst -,$(empty) $(empty), $(subst .exe,$(empty) $(empty) , $2)))
# OUTPUT - name (and path) of output binaries
$(RELEASES):
	@# Do not build Windows binaries with Go native DNS resolver
	$(eval GO_TAGS := $(shell if [ "$(call argument, 2, $@)" != "windows" ]; then echo "-tags netgo"; fi))
	$(eval OUTPUT ?= ./build/$@)
	@echo Building binaries for $@...
	@GOOS=$(call argument, 2, $@) GOARCH=$(call argument, 3, $@) CGO_ENABLED=0 go build -a $(GO_TAGS) -ldflags '-X github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/version.MonitoringAsCode=$(VERSION) -w -extldflags "-static"' -o $(OUTPUT) ./cmd/monaco

install:
	@echo "Installing $(BINARY_NAME)..."
	@CGO_ENABLED=0 go install -a -tags netgo -ldflags '-w -extldflags "-static"' ./cmd/monaco

clean:
	@echo "Removing $(BINARY_NAME), bin/ and /build ..."
ifeq ($(OS),Windows_NT)
	@echo "Windows"
	@if exist bin rd /S /Q bin
	@echo "Windows 2"
	@if exist bin rd /S /Q build
else
	@echo "Linux"
	@rm -rf bin/
	@rm -rf build/
endif

test: mocks
	@echo "Testing $(BINARY_NAME)..."
	@gotestsum ${testopts} --format testdox -- -tags=unit -v -race ./...

integration-test: mocks
	@gotestsum ${testopts} --format testdox -- -tags=integration -timeout=30m -v -race ./...

integration-test-v1: mocks
	@gotestsum ${testopts} --format testdox -- -tags=integration_v1 -timeout=30m -v -race ./...

download-restore-test: mocks
	@gotestsum ${testopts} --format testdox -- -tags=download_restore -timeout=30m -v -race ./...

clean-environments:
    @MONACO_ENABLE_DANGEROUS_COMMANDS=1 go run ./cmd/monaco purge cmd/monaco/integrationtest/v2/test-resources/test_environments_manifest.yaml

nightly-test:mocks
	@gotestsum ${testopts} --format testdox -- -tags=nightly -timeout=240m -v -race ./...

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



#TAG - specify tag value. The main purpose is to define public tag during a release build.
CONTAINER_NAME ?= $(BINARY_NAME)
.PHONY: docker-container
docker-container: $(BINARY_NAME)-linux-amd64
	@echo Building docker container...
	DOCKER_BUILDKIT=1 docker build --build-arg NAME=$(BINARY_NAME) --build-arg SOURCE=$(OUTPUT) --tag $(CONTAINER_NAME):$(VERSION) .

sign-verify-image: setup
	COSIGN_PASSWORD=$(COSIGN_PASSWORD) cosign sign --key env://cosign_key $(FULL_IMAGE_NAME) -y
	cosign verify --key env://cosign_pub $(FULL_IMAGE_NAME)
