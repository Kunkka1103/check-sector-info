# The name of the executable (default is current directory name)
TARGET := check-sector-info

# Go related variables.
GOBASE := $(shell pwd)

# Get Git commit hash
COMMIT := $(shell git rev-parse HEAD)

# Build the project
all: clean build

build:
	@echo "  >  Building binary..."
	go build -mod=mod -ldflags "-X main.commit=$(COMMIT)" -o $(GOBASE)/$(TARGET) $(GOBASE)/$(TARGET).go

clean:
	@echo "  >  Cleaning build cache"
	go clean -mod=mod

.PHONY: all build clean