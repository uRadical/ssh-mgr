BINARY := sshmgr
PKG     := uradical.io/go/sshmgr

.PHONY: build run install lint clean

build:
	CGO_ENABLED=0 go build -o bin/$(BINARY) .

run:
	go run .

install:
	CGO_ENABLED=0 go install $(PKG)

lint:
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not found; falling back to go vet"; \
		go vet ./...; \
	fi

clean:
	rm -rf bin
	go clean
