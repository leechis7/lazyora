.PHONY: build build-native test check
build:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GOAMD64=v1 go build -trimpath -o bin/lazyora ./cmd/lazyora
build-native:
	CGO_ENABLED=0 go build -trimpath -o bin/lazyora-native ./cmd/lazyora
test:
	go test -race ./...
check:
	go vet ./...
