.PHONY: all build deps clean test 

# Binary name
BINARY_NAME=launcher

# Go commands
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test

# Build flags
LDFLAGS=-ldflags "-s -w"

all: deps build ## Download dependencies and build the project

deps: ## Download and verify dependencies
	@echo "Downloading dependencies..."
	$(GOMOD) download
	@echo "Verifying dependencies..."
	$(GOMOD) verify
	@echo "Tidying dependencies..."
	$(GOMOD) tidy
	@echo "✓ Dependencies ready"

build: ## Build the launcher binary
	@echo "Building $(BINARY_NAME)..."
	mkdir -p dist
	$(GOBUILD) $(LDFLAGS) -o dist/$(BINARY_NAME) .
	@echo "✓ Build complete: ./$(BINARY_NAME)"

test: ## Run unit tests
	@echo "Running unit tests..."
	$(GOTEST) -v -race -coverprofile=coverage.out ./...
	@echo "✓ Tests complete"

clean: ## Remove build artifacts and test files
	@echo "Cleaning..."
	$(GOCLEAN)
	@rm -f dist/$(BINARY_NAME)
	@rm -f test-rotel.sh test-rotel-verify.sh
	@echo "✓ Clean complete"
