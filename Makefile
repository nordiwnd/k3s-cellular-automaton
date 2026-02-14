# Variables
REPO_PREFIX ?= ghcr.io/nordiwnd/k3s-cellular-automaton
TAG ?= dev
VERSION ?= $(shell git rev-parse --short HEAD)

# Contexts
DEV_CONTEXT := k3d-gearpit-dev
PROD_CONTEXT := default

.PHONY: all dev release build-amd64 build-arm64

all: dev

# --- Development (AMD64 -> k3d) ---
dev: build-amd64 import-k3d

build-amd64:
	@echo "Building AMD64 images..."
	docker build -t $(REPO_PREFIX)/cells-worker:$(TAG) --platform linux/amd64 ./cells-worker
	docker build -t $(REPO_PREFIX)/grid-controller:$(TAG) --platform linux/amd64 ./grid-controller
	docker build -t $(REPO_PREFIX)/dashboard-ui:$(TAG) --platform linux/amd64 ./dashboard-ui

import-k3d:
	@echo "Importing images to $(DEV_CONTEXT)..."
	k3d image import $(REPO_PREFIX)/cells-worker:$(TAG) -c gearpit-dev
	k3d image import $(REPO_PREFIX)/grid-controller:$(TAG) -c gearpit-dev
	k3d image import $(REPO_PREFIX)/dashboard-ui:$(TAG) -c gearpit-dev

# --- Release (ARM64 -> Registry) ---
release: build-push-arm64

build-push-arm64:
	@echo "Building and Pushing ARM64 images..."
	# Worker (Rust - Optimized)
	docker buildx build --platform linux/arm64 --push -t $(REPO_PREFIX)/cells-worker:latest -t $(REPO_PREFIX)/cells-worker:$(VERSION) ./cells-worker
	# Controller (Go)
	docker buildx build --platform linux/arm64 --push -t $(REPO_PREFIX)/grid-controller:latest -t $(REPO_PREFIX)/grid-controller:$(VERSION) ./grid-controller
	# UI
	docker buildx build --platform linux/arm64 --push -t $(REPO_PREFIX)/dashboard-ui:latest -t $(REPO_PREFIX)/dashboard-ui:$(VERSION) ./dashboard-ui

# --- Utility ---
clean:
	docker system prune -f
