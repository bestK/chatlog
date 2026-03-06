BINARY_NAME := chatlog
GO := go
ifeq ($(VERSION),)
	VERSION := $(shell git describe --tags --always --dirty="-dev")
endif
LDFLAGS := -ldflags '-X "github.com/sjzar/chatlog/pkg/version.Version=$(VERSION)" -w -s'
WINDOWS_LDFLAGS := -ldflags '-H windowsgui -X "github.com/sjzar/chatlog/pkg/version.Version=$(VERSION)" -w -s'

PLATFORMS := \
	darwin/amd64 \
	darwin/arm64 \
	linux/amd64 \
	linux/arm64 \
	windows/amd64 \
	windows/arm64

UPX_PLATFORMS := \
	darwin/amd64 \
	linux/amd64 \
	linux/arm64 \
	windows/amd64

.PHONY: all clean lint tidy test build crossbuild upx

all: clean lint tidy test build

clean:
	@echo "🧹 Cleaning..."
	@rm -rf bin/

lint:
	@echo "🕵️‍♂️ Running linters..."
	golangci-lint run ./...

tidy:
	@echo "🧼 Tidying up dependencies..."
	$(GO) mod tidy

test:
	@echo "🧪 Running tests..."
	$(GO) test ./... -cover

build:
	@echo "🔨 Building for current platform..."
	@if [ "$(OS)" = "Windows_NT" ]; then \
		CGO_ENABLED=1 $(GO) build -trimpath $(WINDOWS_LDFLAGS) -o bin/$(BINARY_NAME).exe main.go; \
	else \
		CGO_ENABLED=1 $(GO) build -trimpath $(LDFLAGS) -o bin/$(BINARY_NAME) main.go; \
	fi

crossbuild: clean
	@echo "🌍 Building for multiple platforms..."
	for platform in $(PLATFORMS); do \
		os=$$(echo $$platform | cut -d/ -f1); \
		arch=$$(echo $$platform | cut -d/ -f2); \
		float=$$(echo $$platform | cut -d/ -f3); \
		output_name=bin/chatlog_$${os}_$${arch}; \
		[ "$$float" != "" ] && output_name=$$output_name_$$float; \
		build_ldflags='$(LDFLAGS)'; \
		if [ "$$os" = "windows" ]; then build_ldflags='$(WINDOWS_LDFLAGS)'; fi; \
		if [ "$$os" = "windows" ]; then output_name=$$output_name.exe; fi; \
		echo "🔨 Building for $$os/$$arch..."; \
		echo "🔨 Building for $$output_name..."; \
		GOOS=$$os GOARCH=$$arch CGO_ENABLED=1 GOARM=$$float $(GO) build -trimpath $$build_ldflags -o $$output_name main.go ; \
		if [ "$(ENABLE_UPX)" = "1" ] && echo "$(UPX_PLATFORMS)" | grep -q "$$os/$$arch"; then \
			echo "⚙️ Compressing binary $$output_name..." && upx --best $$output_name; \
		fi; \
	done
