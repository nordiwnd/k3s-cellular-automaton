# Project Concept: K8s-Native Cellular Automaton

## 1. Core Vision
This project implements a Cellular Automaton (e.g., Game of Life) where **Kubernetes Pods themselves are the cells**.
Instead of simulating the grid in memory, we use the Kubernetes cluster orchestration as the simulation engine.

* **The Grid:** A virtual 2D grid mapped to a 1D Kubernetes StatefulSet.
* **The Cells:** Individual Rust Pods.
* **The State:** Whether a Pod is "Running" (Alive) or "Terminated/CrashLoop" (Dead).
* **The Game Loop:** Kubernetes Self-Healing mechanism acts as the game tick.

## 2. Key Mechanics (The "Second Grid")
We define a "Virtual 2D Grid" overlaying the StatefulSet logic.

* **Mapping:**
    * StatefulSet Name: `cell`
    * Pod Names: `cell-0`, `cell-1`, ... `cell-N`
    * Logic: A linear index `i` maps to `(x, y)` coordinates on a `W x H` grid.
    * Topology: Each Pod must calculate its own neighbors' FQDNs (e.g., `cell-{i-1}`, `cell-{i+W}`) to query their state via gRPC.

## 3. User Experience (The Dashboard)
A "God View" web dashboard allows users to interact with the cluster physically.

* **Visualization:** Real-time grid view of Pod statuses.
* **Interaction:**
    * **Chaos Engineering as Gameplay:** Clicking a cell sends a `kubectl delete pod` command to the cluster.
    * **Observation:** Watching Kubernetes struggle to "heal" the deleted pod (respawn), effectively reviving the cell in the game.

## 4. Architectural Goals & Constraints
* **Hyper-Efficiency:** Since we run 100+ Pods on a Raspberry Pi (ARM64, 8GB RAM), each Pod must use minimal resources (<10MB RAM).
* **Deterministic Networking:** Use Headless Services for stable peer-to-peer discovery.
* **Hybrid Build:** Develop on AMD64 (WSL2), Deploy to ARM64 (Pi/k3s).