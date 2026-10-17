HOST_GOOS = $(shell go env GOHOSTOS)
HOST_GOARCH = $(shell go env GOHOSTARCH)

HOST_EXE = $(if $(filter windows,$(HOST_GOOS)),.exe,)

.PHONY: build build-all build-linux build-amd64 build-arm64 build-windows build-windows-amd64 build-windows-arm64 build-native test check
build: build-arm64
build-all: build-linux build-windows
build-linux: build-amd64 build-arm64
build-windows: build-windows-amd64 build-windows-arm64
build-windows-amd64:
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 GOAMD64=v1 go build -trimpath -o bin/lazyora-windows-amd64.exe ./cmd/lazyora
build-windows-arm64:
	CGO_ENABLED=0 GOOS=windows GOARCH=arm64 go build -trimpath -o bin/lazyora-windows-arm64.exe ./cmd/lazyora
build-amd64:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GOAMD64=v1 go build -trimpath -o bin/lazyora-linux-amd64 ./cmd/lazyora
build-arm64:
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -o bin/lazyora-linux-arm64 ./cmd/lazyora
build-native:
	CGO_ENABLED=0 GOOS=$(HOST_GOOS) GOARCH=$(HOST_GOARCH) GOAMD64=v1 go build -trimpath -o bin/lazyora-$(HOST_GOOS)-$(HOST_GOARCH)$(HOST_EXE) ./cmd/lazyora
test:
	go test -race ./...
check:
	go vet ./...
