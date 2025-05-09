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
	@echo "cleaning data dumps..."
	@rm ./data/webpages/*
