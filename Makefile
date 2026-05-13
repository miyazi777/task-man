BINARY := task-man
PKG    := ./cmd/task-man
GOFLAGS ?=
LDFLAGS ?= -s -w

.PHONY: all build run test fmt vet lint tidy clean install help

all: build

build:
	go build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BINARY) $(PKG)

run: build
	./$(BINARY)

test:
	go test ./...

fmt:
	go fmt ./...

vet:
	go vet ./...

lint:
	golangci-lint run ./...

tidy:
	go mod tidy

install:
	go install $(GOFLAGS) -ldflags "$(LDFLAGS)" $(PKG)

clean:
	rm -f $(BINARY)
	go clean ./...

help:
	@echo "Targets:"
	@echo "  build    バイナリを ./$(BINARY) にビルド"
	@echo "  run      ビルドして実行"
	@echo "  test     全パッケージのテスト"
	@echo "  fmt      go fmt"
	@echo "  vet      go vet"
	@echo "  lint     golangci-lint run"
	@echo "  tidy     go mod tidy"
	@echo "  install  \$$GOPATH/bin にインストール"
	@echo "  clean    バイナリ削除"
