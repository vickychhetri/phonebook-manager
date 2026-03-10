# Phone Book Application Makefile
.PHONY: all clean install windows linux-amd64 linux-386

APP_NAME = phonebook
VERSION = 1.0.0
BUILD_DIR = build

# Default target - only build what you need
all: clean linux-amd64 linux-386 windows

# Clean build directory
clean:
	rm -rf $(BUILD_DIR)
	mkdir -p $(BUILD_DIR)

# Linux 64-bit build
linux-amd64:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=1 go build -o $(BUILD_DIR)/$(APP_NAME)-linux-amd64-$(VERSION) -ldflags="-s -w" main.go
	@echo "✅ Built linux-amd64"

# Linux 32-bit build
linux-386:
	GOOS=linux GOARCH=386 CGO_ENABLED=1 go build -o $(BUILD_DIR)/$(APP_NAME)-linux-386-$(VERSION) -ldflags="-s -w" main.go
	@echo "✅ Built linux-386"

# Windows 64-bit build (requires mingw64)
windows:
	GOOS=windows GOARCH=amd64 CGO_ENABLED=1 CC=x86_64-w64-mingw32-gcc \
		go build -o $(BUILD_DIR)/$(APP_NAME)-windows-amd64-$(VERSION).exe -ldflags="-s -w" main.go
	@echo "✅ Built windows-amd64"

# Install locally
install:
	go build -o $(BUILD_DIR)/$(APP_NAME) -ldflags="-s -w" main.go
	sudo cp $(BUILD_DIR)/$(APP_NAME) /usr/local/bin/

# Run the app
run:
	go run main.go

# Show help
help:
	@echo "Available targets:"
	@echo "  all         : Build for Linux (64/32-bit) and Windows"
	@echo "  linux-amd64 : Build 64-bit Linux"
	@echo "  linux-386   : Build 32-bit Linux"
	@echo "  windows     : Build 64-bit Windows"
	@echo "  install     : Install locally"
	@echo "  clean       : Clean build directory"
	@echo "  run         : Run the application"