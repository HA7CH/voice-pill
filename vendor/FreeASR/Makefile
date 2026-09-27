VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE    ?= $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
LDFLAGS := -s -w \
	-X github.com/WEIFENG2333/FreeASR/cmd.Version=$(VERSION) \
	-X github.com/WEIFENG2333/FreeASR/cmd.GitCommit=$(COMMIT) \
	-X github.com/WEIFENG2333/FreeASR/cmd.BuildDate=$(DATE)

.PHONY: build test lint clean proto

build:
	go build -ldflags "$(LDFLAGS)" -o freeasr .

test:
	go test ./... -race -count=1 -timeout=60s

lint:
	golangci-lint run ./...

clean:
	rm -f freeasr

proto:
	protoc -I proto --go_out=internal/pb --go_opt=paths=source_relative proto/asr.proto

# Cross-compilation targets
.PHONY: build-linux-amd64 build-linux-arm64 build-darwin-amd64 build-darwin-arm64 build-windows-amd64

build-linux-amd64:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=1 go build -ldflags "$(LDFLAGS)" -o freeasr-linux-amd64 .

build-linux-arm64:
	GOOS=linux GOARCH=arm64 CGO_ENABLED=1 CC=aarch64-linux-gnu-gcc go build -ldflags "$(LDFLAGS)" -o freeasr-linux-arm64 .

build-darwin-amd64:
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=1 go build -ldflags "$(LDFLAGS)" -o freeasr-darwin-amd64 .

build-darwin-arm64:
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=1 go build -ldflags "$(LDFLAGS)" -o freeasr-darwin-arm64 .

build-windows-amd64:
	GOOS=windows GOARCH=amd64 CGO_ENABLED=1 go build -ldflags "$(LDFLAGS)" -o freeasr-windows-amd64.exe .
