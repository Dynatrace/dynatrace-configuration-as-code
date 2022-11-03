BINARY=monaco
VERSION=2.x

.PHONY: lint format mocks build install clean test integration-test integration-test-v1 test-package default add-license-headers

default: build

lint:
ifeq ($(OS),Windows_NT)
	@.\tools\check-format.cmd
else
	@go install github.com/google/addlicense@latest
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

mocks:
	@echo "Generating mocks"
	@go install github.com/golang/mock/mockgen@latest
	@go generate ./...

vet:
	@echo "Vetting files"
	@go vet ./...

build: clean
	@echo Build ${BINARY}
	@CGO_ENABLED=0 go build -a -tags netgo -ldflags '-w -extldflags "-static"' -o ./bin/${BINARY} ./cmd/monaco

build-release: clean
	@echo Release build ${BINARY} ${VERSION}
	@ GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -a -tags netgo -ldflags '-X github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/version.MonitoringAsCode=${VERSION} -w -extldflags "-static"' -o ./build/${BINARY}-windows-amd64.exe ./cmd/monaco
	@ GOOS=windows GOARCH=386   CGO_ENABLED=0 go build -a -tags netgo -ldflags '-X github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/version.MonitoringAsCode=${VERSION} -w -extldflags "-static"' -o ./build/${BINARY}-windows-386.exe   ./cmd/monaco
	@ GOOS=linux   GOARCH=amd64 CGO_ENABLED=0 go build -a -tags netgo -ldflags '-X github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/version.MonitoringAsCode=${VERSION} -w -extldflags "-static"' -o ./build/${BINARY}-linux-amd64       ./cmd/monaco
	@ GOOS=linux   GOARCH=386   CGO_ENABLED=0 go build -a -tags netgo -ldflags '-X github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/version.MonitoringAsCode=${VERSION} -w -extldflags "-static"' -o ./build/${BINARY}-linux-386         ./cmd/monaco
	@ GOOS=darwin  GOARCH=amd64 CGO_ENABLED=0 go build -a -tags netgo -ldflags '-X github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/version.MonitoringAsCode=${VERSION} -w -extldflags "-static"' -o ./build/${BINARY}-darwin-amd64      ./cmd/monaco
	@ GOOS=darwin  GOARCH=arm64 CGO_ENABLED=0 go build -a -tags netgo -ldflags '-X github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/version.MonitoringAsCode=${VERSION} -w -extldflags "-static"' -o ./build/${BINARY}-darwin-arm64      ./cmd/monaco

install:
	@echo Install ${BINARY}
	@CGO_ENABLED=0 go install -a -tags netgo -ldflags '-w -extldflags "-static"' ./cmd/monaco

run:
	@CGO_ENABLED=0 go run -a -tags netgo -ldflags '-w -extldflags "-static"' ./cmd/monaco

clean:
	@echo Remove bin/ and build/
ifeq ($(OS),Windows_NT)
	@if exist bin rd /S /Q bin
	@if exist bin rd /S /Q build
else
	@rm -rf bin/
	@rm -rf build/
endif

test: mocks lint
	@go test -tags=unit -v ./...

integration-test:
	@go test -tags=integration -timeout=30m -v ./...

integration-test-v1:
	@go test -tags=integration_v1 -timeout=30m -v ./...

download-restore-test:
	@go test -tags=download_restore -timeout=30m -v ./...

clean-environments:
	@go test -tags=cleanup -v ./...


# Build and Test a single package supplied via pgk variable, without using test cache
# Run as e.g. make test-package pkg=project
pkg=...
test-package: mocks lint
	@go test -tags=unit -count=1 -v ./pkg/${pkg}

update-dependencies:
	@echo Update go dependencies
	@go get -u ./...
	@go mod tidy
