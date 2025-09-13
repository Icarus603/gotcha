APP_NAME := gotcha
BIN_DIR := bin

.PHONY: build run fmt lint test tidy clean

build:
	@echo "Building $(APP_NAME)"
	@mkdir -p $(BIN_DIR)
	@go build -o $(BIN_DIR)/$(APP_NAME) ./cmd/gotcha

run: build
	@./$(BIN_DIR)/$(APP_NAME)

fmt:
	@go fmt ./...

tidy:
	@go mod tidy

test:
	@go test ./...

clean:
	rm -rf $(BIN_DIR)

