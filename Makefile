BINARY_NAME=torsen
BUILD_DIR=bin

.PHONY: build clean install uninstall

build:
	@echo "Building $(BINARY_NAME)..."
	@go build -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/torsen/
	@echo "Done: $(BUILD_DIR)/$(BINARY_NAME)"

clean:
	@rm -rf $(BUILD_DIR)
	@echo "Cleaned."

install: build
	@echo "Installing to /usr/local/bin/$(BINARY_NAME)..."
	@sudo cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/$(BINARY_NAME)
	@sudo chmod +x /usr/local/bin/$(BINARY_NAME)
	@sudo mkdir -p /var/lib/torsen
	@sudo mkdir -p /var/log/torsen
	@echo "Installed."

uninstall:
	@sudo rm -f /usr/local/bin/$(BINARY_NAME)
	@echo "Uninstalled."
