default: fmt lint install

build:
	go build -v ./...

install: build
	go install -v ./...

lint:
	golangci-lint run

fmt:
	gofmt -s -w .

generate:
	cd tools; go generate ./...

test:
	go test -v -cover -timeout=120s -parallel=10 ./...

testacc:
	TF_ACC=1 go test -v -cover -timeout 120m ./internal/provider/...

.PHONY: build install lint fmt generate test testacc
