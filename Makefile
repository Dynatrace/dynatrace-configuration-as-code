BINARY=monaco

.PHONY: build clean test integration-test

lint:
	@sh ./tools/check-format.sh

format:
	@gofmt -w .

build: clean
	@echo Build ${BINARY}
	@go build ./...
	@go build -o ./bin/${BINARY} ./cmd/monaco

clean:
	@echo Remove ${BINARY} and bin/
	@rm -f ${BINARY}
	@rm -rf bin/

test: build
	@go test -tags=unit -v ./...

integration-test: build
	@go test -tags=cleanup -v ./...
	@go test -tags=integration -v ./...

# Build and Test a single package supplied via pgk variable, without using test cache
# Run as e.g. make test-package pkg=project
pkg=...
test-package: build
	@go test -tags=unit -count=1 -v ./${pgk}
