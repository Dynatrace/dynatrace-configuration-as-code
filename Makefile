BINARY=monaco

.PHONY: lint format mocks build install clean test integration-test test-package default add-license-headers

default: test

setup:
	@echo "Installing build tools..."
	@go get github.com/google/addlicense@latest
	@go get gotest.tools/gotestsum@latest
	@go get github.com/golang/mock/mockgen

lint: setup
ifeq ($(OS),Windows_NT)
	@.\tools\check-format.cmd
else
	@sh ./tools/check-format.sh
	@sh ./tools/check-license-headers.sh
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
	@go generate ./...

build: clean lint
	@echo "Building ${BINARY}..."
	@go build ./...
	@go build -o ./bin/${BINARY} ./cmd/monaco

install: clean lint
	@echo "Installing ${BINARY}..."
	@go install ./...

clean:
	@echo "Removing ${BINARY} and bin/..."
ifeq ($(OS),Windows_NT)
	@if exist ${BINARY} del /Q ${BINARY}
	@if exist bin rd /S /Q bin
else
	@rm -f ${BINARY}
	@rm -rf bin/
endif

test: setup mocks build
	@echo "Testing ${BINARY}..."
	@gotestsum ${testopts} -- -tags=unit -v ./...

integration-test: setup build
	@echo "Integration testing ${BINARY}..."
	@gotestsum ${testopts} -- -tags=cleanup -v ./...
	@gotestsum ${testopts} -- -tags=integration -v ./...

# Build and Test a single package supplied via pgk variable, without using test cache
# Run as e.g. make test-package pkg=project
pkg=...
test-package: setup mocks build
	@echo "Testing ${pkg}..."
	@gotestsum -- -tags=unit -count=1 -v ./pkg/${pkg}
