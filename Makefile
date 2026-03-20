BINARY  := texutil
GO      ?= go
DIST    := dist

VERSION ?= dev
LDFLAGS := -s -w -X texUtil/cmd.version=$(VERSION)

PLATFORMS := \
	linux/amd64 \
	linux/arm64 \
	linux/386 \
	linux/arm \
	windows/amd64 \
	windows/386 \
	darwin/amd64 \
	darwin/arm64

.PHONY: build build-all clean $(PLATFORMS)

## build: build for the current platform
build:
	$(GO) build -ldflags "$(LDFLAGS)" -o $(BINARY) .

## build-all: build for all supported platforms
build-all: $(PLATFORMS)

$(PLATFORMS):
	$(eval OS   := $(word 1, $(subst /, ,$@)))
	$(eval ARCH := $(word 2, $(subst /, ,$@)))
	$(eval EXT  := $(if $(filter windows,$(OS)),.exe,))
	$(eval OUT  := $(DIST)/$(BINARY)-$(OS)-$(ARCH)$(EXT))
	GOOS=$(OS) GOARCH=$(ARCH) $(GO) build -ldflags "$(LDFLAGS)" -o $(OUT) .
	@echo "built $(OUT)"

## clean: remove build artifacts
clean:
	rm -f $(BINARY)
	rm -rf $(DIST)
