HOST_GOOS = $(shell go env GOHOSTOS)
HOST_GOARCH = $(shell go env GOHOSTARCH)

.PHONY: build build-amd64 build-arm64 build-native test check
build: build-amd64 build-arm64
build-amd64:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GOAMD64=v1 go build -trimpath -o bin/lazyora-linux-amd64 ./cmd/lazyora
build-arm64:
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -o bin/lazyora-linux-arm64 ./cmd/lazyora
build-native:
	CGO_ENABLED=0 GOOS=$(HOST_GOOS) GOARCH=$(HOST_GOARCH) GOAMD64=v1 go build -trimpath -o bin/lazyora-$(HOST_GOOS)-$(HOST_GOARCH) ./cmd/lazyora
test:
	go test -race ./...
check:
	go vet ./...
