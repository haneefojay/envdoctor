.PHONY: build test vet fmt lint generate

build:
	go build ./...

test:
	go test ./...

vet:
	go vet ./...

fmt:
	gofmt -l -w .

generate:
	go generate ./...