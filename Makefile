# DOS Capsule - unified build/pack/run/debug entry point.
# Cross-platform (macOS arm64/x64, Linux, Windows+make).

SHELL := /bin/bash

# --- binaries & paths --------------------------------------------------------
BINARY   := doscapsule
SRC      := ./cmd/doscapsule
FRONTEND := frontend
DIST     := $(SRC)/dist
GO       ?= go
PNPM     ?= pnpm

# Version stamp (git tag/sha or timestamp).
VERSION ?= $(shell git describe --tags --always 2>/dev/null || date +%Y%m%d-%H%M%S)
LDFLAGS  := -s -w -X main.version=$(VERSION)

# --- targets ----------------------------------------------------------------

.PHONY: help dist build build-arm build-x64 build-all test run debug zip zip-arm zip-x64 zip-all clean

help: ## show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS=":.*?## "}; {printf "  \033[36m%-14s\033[0m %s\n", $$1, $$2}'

dist: ## build the frontend (output to cmd/doscapsule/dist)
	cd $(FRONTEND) && $(PNPM) install && $(PNPM) build

build: ## build for the current platform
	cd $(SRC) && $(GO) build -ldflags "$(LDFLAGS)" -o ../../$(BINARY) .

build-arm: ## build macOS arm64
	cd $(SRC) && GOOS=darwin GOARCH=arm64 $(GO) build -ldflags "$(LDFLAGS)" -o ../../$(BINARY)-arm64 .

build-x64: ## build macOS amd64
	cd $(SRC) && GOOS=darwin GOARCH=amd64 $(GO) build -ldflags "$(LDFLAGS)" -o ../../$(BINARY)-amd64 .

build-all: dist build-arm build-x64 ## frontend + both macOS arches

test: ## run backend tests
	$(GO) test ./...

run: dist build ## build then run (opens browser)
	./$(BINARY)

debug: ## run with local debugging (no embed optimization)
	cd $(SRC) && $(GO) run -gcflags "all=-N -l" . -port=8420

zip: dist build ## package the current-platform binary into a zip (no DOSBox)
	@scripts/zip.sh $(BINARY)

zip-arm: dist build-arm ## package macOS arm64
	@scripts/zip.sh $(BINARY)-arm64 arm64

zip-x64: dist build-x64 ## package macOS amd64
	@scripts/zip.sh $(BINARY)-amd64 x64

zip-all: zip-arm zip-x64 ## package both macOS arches

clean: ## remove build artifacts
	rm -rf $(DIST) $(BINARY) $(BINARY)-arm64 $(BINARY)-amd64 *.zip

# ── package-manager manifests ────────────────────────────────────────────────
# Homebrew and Scoop pin an exact version and checksum, so every release needs
# the manifests bumped or 'brew upgrade' keeps reporting the old version as
# current. These two targets keep that from being something to remember.

LATEST_TAG = $(shell gh release view --repo bytecapsule/doscapsule --json tagName --jq .tagName 2>/dev/null)
MANIFEST_TAG = $(shell grep -oE 'doscapsule-arm64-[0-9]{8}-[0-9]+' Formula/doscapsule.rb 2>/dev/null | head -1 | sed 's/doscapsule-arm64-//')

manifest: ## point Formula/ and bucket/ at the latest GitHub release
	@if [ -z "$(LATEST_TAG)" ]; then echo "error: cannot read the latest release (is gh authenticated?)" >&2; exit 1; fi
	@echo "latest release: $(LATEST_TAG)"
	@bash scripts/update-packaging.sh "$(LATEST_TAG)"
	@echo
	@echo "Now commit and push (note [skip ci]: a manifest bump must never trigger a build,"
	@echo "or an auto-build-on-push setup would bump again and loop forever):"
	@echo "  git add Formula bucket"
	@echo "  git commit -m \"packaging: bump to $(LATEST_TAG) [skip ci]\""
	@echo "  git push"

manifest-check: ## warn when the manifests lag behind the latest release
	@echo "manifests: $(MANIFEST_TAG)"
	@echo "latest:    $(LATEST_TAG)"
	@if [ "$(MANIFEST_TAG)" != "$(LATEST_TAG)" ]; then \
	  echo "STALE: run 'make manifest' and push, or package managers keep serving $(MANIFEST_TAG)"; \
	  exit 1; \
	fi
	@echo "up to date"
