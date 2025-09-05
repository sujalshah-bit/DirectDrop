# Makefile

BINS = bin/app bin/server

.PHONY: all build clean run-app run-server dev-app dev-server test lint fmt release

all: build

build: $(BINS)

bin/app: cmd/app/main.go
	@mkdir -p bin
	go build -o bin/app ./cmd/app

bin/server: cmd/server/server.go
	@mkdir -p bin
	go build -o bin/server ./cmd/server

run-app: bin/app
	./bin/app $(ARGS)

run-server: bin/server
	./bin/server $(ARGS)

dev-app:
	go run ./cmd/app $(ARGS)

dev-server:
	go run ./cmd/server $(ARGS)

test:
	go test ./...

lint:
	go vet ./...

fmt:
	go fmt ./...

clean:
	rm -rf bin dist

goreleaser_build:
	goreleaser build --clean

release:
	goreleaser release --rm-dist
