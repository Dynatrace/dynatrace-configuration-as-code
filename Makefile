BINARY=monaco

.PHONY: lint format mocks build install clean test integration-test test-package default add-license-headers

default: build

lint:
ifeq ($(OS),Windows_NT)
	@.\tools\check-format.cmd
else
	@go install github.com/google/addlicense@latest
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

build-release: clean lint
	@echo Release build ${BINARY}
	@ GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -tags netgo -ldflags '-w -extldflags "-static"' -o ./build/${BINARY}-windows-amd64.exe ./cmd/monaco
	@ GOOS=windows GOARCH=386   CGO_ENABLED=0 go build -tags netgo -ldflags '-w -extldflags "-static"' -o ./build/${BINARY}-windows-386.exe   ./cmd/monaco
	@ GOOS=linux   GOARCH=amd64 CGO_ENABLED=0 go build -tags netgo -ldflags '-w -extldflags "-static"' -o ./build/${BINARY}-linux-amd64       ./cmd/monaco
	@ GOOS=linux   GOARCH=386   CGO_ENABLED=0 go build -tags netgo -ldflags '-w -extldflags "-static"' -o ./build/${BINARY}-linux-386         ./cmd/monaco
	@ GOOS=darwin  GOARCH=amd64 CGO_ENABLED=0 go build -tags netgo -ldflags '-w -extldflags "-static"' -o ./build/${BINARY}-darwin-amd64      ./cmd/monaco
	@ GOOS=darwin  GOARCH=arm64 CGO_ENABLED=0 go build -tags netgo -ldflags '-w -extldflags "-static"' -o ./build/${BINARY}-darwin-arm64      ./cmd/monaco

install: clean lint
	@echo Install ${BINARY}
	@go install ./...

clean:
	@echo Remove bin/ and build/
ifeq ($(OS),Windows_NT)
	@if exist bin rd /S /Q bin
	@if exist bin rd /S /Q build
else
	@rm -rf bin/
	@rm -rf build/
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
