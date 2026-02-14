---
trigger: always_on
---

# Infrastructure & Deployment Constraints

You are operating within a hybrid architecture environment (AMD64 Dev -> ARM64 Prod). All code and configurations MUST adhere to the following constraints.

## 1. Environment Contexts
* **Development (Local):**
    * Host: WSL2 (AMD64).
    * Cluster: `k3d` (Context: `k3d-gearpit-dev`).
* **Production (Edge):**
    * Host: Raspberry Pi 4/5 (ARM64).
    * Cluster: `k3s` (Context: `default`).
    * **Constraint:** 8GB Total RAM shared across system and all pods.

## 2. Kubernetes Architecture (Cellular Automaton)
* **Workload:** StatefulSet for Cells.
* **Networking:** Headless Service for deterministic DNS (`pod-N.svc...`).
* **Storage:** Ephemeral only (unless persistence is strictly required for game state).

## 3. Rust Resource Optimization (STRICT)
To run efficiently on Raspberry Pi, the Rust worker nodes MUST be optimized for **Minimum Footprint** over Maximum Throughput.
* **Cargo Profile (`release`):**
    * `opt-level = "z"` (Optimize for binary size).
    * `lto = true` (Enable Link Time Optimization).
    * `codegen-units = 1` (Maximize optimization, slower builds).
    * `panic = "abort"` (Remove unwinding symbols to save size/memory).
    * `strip = true` (Remove symbols).
* **Runtime:** * Use `tokio` with minimal features enabled.
    * Consider using a single-threaded runtime (`worker_threads = 1`) if the logic allows, to reduce context switching overhead on the Pi.
* **Docker Image:**
    * **Base:** `scratch` (Static binary) or `gcr.io/distroless/static:nonroot`.
    * **Target Size:** Compressed image size should be < 10MB.

## 4. Multi-Arch Build Strategy
* **Tool:** `docker buildx` is mandatory.
* **Development Workflow (Makefile `dev`):**
    * Build for `linux/amd64`.
    * Import directly to k3d (`k3d image import`).
    * Apply to `k3d-gearpit-dev` context.
* **Release Workflow (GitHub Actions / Makefile `release`):**
    * Build for `linux/arm64`.
    * Push to container registry (GHCR recommended).
    * Deployment target: `default` context (k3s).

## 5. Technology Stack
* **Workers:** Rust (Optimized) + gRPC.
* **Controller:** Go + WebSocket (Serve dashboard).
* **Frontend:** TypeScript/React (Vite).