# Makefile for RTMP Tester project

# Variables
# Default target
.PHONY: all
all: build

# Build the project
.PHONY: build
build:
	make -C bin
