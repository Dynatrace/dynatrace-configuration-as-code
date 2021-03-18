BINARY=monaco

.PHONY: lint format mocks build install clean test integration-test test-package default add-license-headers

default: build

lint:
ifeq ($(OS),Windows_NT)
	@.\tools\check-format.cmd
else
	@go get github.com/google/addlicense
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

mocks:
	@go get github.com/golang/mock/mockgen
	@go generate ./...

build: clean lint
	@echo Build ${BINARY}
	@go build ./...
	@go build -o ./bin/${BINARY} ./cmd/monaco

install: clean lint
	@echo Install ${BINARY}
	@go install ./...

clean:
	@echo Remove ${BINARY} and bin/
ifeq ($(OS),Windows_NT)
	@if exist ${BINARY} del /Q ${BINARY}
	@if exist bin rd /S /Q bin
else
	@rm -f ${BINARY}
	@rm -rf bin/
endif

test: mocks build
	@go test -tags=unit -v ./...

integration-test: build
	@go test -tags=cleanup -v ./...
	@go test -tags=integration -v ./...

# Build and Test a single package supplied via pgk variable, without using test cache
# Run as e.g. make test-package pkg=project
pkg=...
test-package: mocks build
	@go test -tags=unit -count=1 -v ./pkg/${pkg}
