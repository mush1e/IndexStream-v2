BINARY   := index-stream-v2
CMD_PATH := ./cmd/index-stream

.PHONY: all
all: build

.PHONY: build
build:
	@echo "👉 Building $(BINARY)..."
	@mkdir -p bin
	@go build -o bin/$(BINARY) $(CMD_PATH)

.PHONY: run
run:
	@echo "🚀 Running server..."
	@mkdir -p data/webpages cache/disk
	@go run $(CMD_PATH)

.PHONY: clean
clean:
	@echo "🧹 Cleaning up..."
	@rm -rf bin

.PHONY: test
test:
	@echo "🧪 Running tests..."
	@go test ./...

.PHONY: clear
clear:
	@echo "🗑️  Clearing data dumps..."
	@rm -rf ./data/webpages/*
	@rm -rf ./cache/disk/*

.PHONY: cache-clear
cache-clear:
	@echo "💾 Clearing cache..."
	@rm -rf ./cache/disk/*

.PHONY: setup
setup:
	@echo "🛠️  Setting up directories..."
	@mkdir -p data/webpages cache/disk

.PHONY: install
install: build
	@echo "📦 Installing $(BINARY)..."
	@cp bin/$(BINARY) /usr/local/bin/

.PHONY: dev
dev: setup
	@echo "🔧 Starting development server with auto-reload..."
	@go run $(CMD_PATH)