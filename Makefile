# crunchwl Makefile
# The default targets build natively for the host OS (Linux/macOS).
# A Windows cross-compile target is provided for convenience.

BINARY_NAME := crunchwl
BINARY_DIR  := bin
PKG         := ./...

# Shrink binary footprint (strip symbol table + DWARF info)
LDFLAGS         := -ldflags="-s -w"
LDFLAGS_WINDOWS := -ldflags="-s -w -H=windowsgui"

GO ?= go

.PHONY: all build build-windows install clean test fmt vet help

all: clean build ## Clean workspace and compile the binary

build: ## Compile the binary into $(BINARY_DIR)
	@mkdir -p $(BINARY_DIR)
	$(GO) build $(LDFLAGS) -o $(BINARY_DIR)/$(BINARY_NAME) .

build-windows: ## Cross-compile a Windows binary into $(BINARY_DIR)
	@mkdir -p $(BINARY_DIR)
	GOOS=windows $(GO) build $(LDFLAGS_WINDOWS) -o $(BINARY_DIR)/$(BINARY_NAME).exe .

install: build ## Build and install the binary into GOBIN
	$(GO) install $(LDFLAGS) .

clean: ## Remove build artifacts
	@rm -rf $(BINARY_DIR)

test: ## Run the test suite
	$(GO) test $(PKG)

fmt: ## Format the code
	$(GO) fmt $(PKG)

vet: ## Run go vet
	$(GO) vet $(PKG)

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-14s\033[0m %s\n", $$1, $$2}'
