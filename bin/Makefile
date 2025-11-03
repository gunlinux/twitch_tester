# Makefile for RTMP Tester project

# Variables
BINARY_NAME := rtmp_tester
GO_FILES := main.go
GO_MOD := go.mod

# Default target
.PHONY: all
all: build

# Build the project
.PHONY: build
build:
	go build -o $(BINARY_NAME) $(GO_FILES)

# Run the project
.PHONY: run
run:
	go run $(GO_FILES)

# Clean build artifacts
.PHONY: clean
clean:
	rm -f $(BINARY_NAME)

# Default target
.DEFAULT_GOAL := build
