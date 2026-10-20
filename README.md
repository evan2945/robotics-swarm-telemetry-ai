# Autonomous Multi-Agent Mesh Simulation

A decentralized multi-agent simulation built in **Go** and visualized with **Pygame** (Python), demonstrating peer-to-peer mesh networking, custom kinematic state machines, and autonomous agent recovery via a gossip protocol.

## Key Features

* **Gossip Protocol Network**: Decentralized packet propagation with TTL management, deduplication (`SeenPackets`), and heartbeat tracking (`PendingPing`).
* **Autonomous Recovery ("Sparker" Units)**: Emergency dispatch logic that listens for network-wide `SOS` distress signals, navigates to downed units, injects initial jump-start voltages, and hands off charging station coordinates.
* **Kinematic State Engine**: Finite state machines (FSM) governing movement, battery depletion rates, charging cycles, and spatial awareness for multiple agent blueprints (Watchers, Scrappers, Sparkers).
* **Thread-Safe Concurrency**: Modular Go architecture utilizing goroutines, mutexes, and channels (`Inbox`/`Outbox`) for isolated agent lifecycles.
* **Stream & Visualization Pipeline**: Redis stream integration powering a real-time visual UI rendered in Python (Pygame).

---

## Architecture Overview

The system models agents operating within a shared coordinate environment communicating purely through gossip network packets:

* **Patrol & Low Power**: Units autonomously wander or locate charging stations based on proximity caches.
* **Heartbeats & SOS**: Peers send heartbeat requests to visible units. Unresponsive or dead units trigger an `SOS` packet broadcast across the mesh.
* **Emergency Rescue**: Sparker units receive `SOS` broadcasts, pause patrol routes, navigate to the target, and jump-start dead agent batteries.

---

## Configuration & Demonstration Tuning

Agent properties are fully decoupled from code and defined in `configs/cauldron_registry.json`:

* **Kinematic Specs**: Speeds, vision angles/ranges, and gossip communication radiuses.
* **Battery & Transfer Mechanics**: Initial battery levels, base drain rates, low-power drain rates, charging speeds, and Sparker transfer efficiencies (`recharging_drain_rate: 8.0`, `transfer_efficiency: 0.8`).
* **Accelerated Testing**: Default battery drain values in `cauldron_registry.json` are intentionally set to higher rates so units enter low-power and dead states quickly, demonstrating the SPARKER rescue lifecycle in real time without long wait intervals.

---

### Agent Spawning & Scaling

The simulation layout and machine counts are configured directly in `main.go` via the `createMachine` generator function:

* **Default Fleet**: Spawns one `WATCHER`, one `SCRAPPER`, and one `SPARKER` centered at the main charging station coordinate.
* **Scaling Simulation**: To scale up the mesh network and test gossip routing under higher packet volume, update the `numOfMachines` argument in `main.go` to spawn larger swarms.

  ---

## System Requirements

* **Go** (`>= 1.22`)
* **Python** (`3.x`) + `venv`
* **Docker & Docker Compose** (for Redis infrastructure)

---

## Getting Started

### 1. Clone the Repository
```bash
git clone evan2945/robotics-swarm-telemetry-ai
cd swarm-engine
```

### 2. Start Infrastructure (Redis)
The simulation uses Redis to bridge simulation state with the visualizer. Start the containerized Redis instance in the background:
```bash
docker-compose up -d
```

### 3. Set Up Python Virtual Environment
Create and activate a virtual environment for the Pygame visualizer, then install the dependencies:

**macOS / Linux:**
```bash
python3 -m venv venv
source venv/bin/activate
pip install -r requirements.txt
```
**Windows:**
```
python -m venv venv
.\venv\Scripts\activate
pip install -r requirements.txt
```

### 4. Run the Application
**Note on Terminal Output**: When running `main.go`, you will see real-time console output displaying various messages such as packet propagation, heartbeat checks, and state transitions (e.g., `[SPARKER] SOS message received...`). These log lines are left enabled intentionally so you can observe the agents communicating across the gossip network as the simulation runs.

**Step A: Launch the Go Simulation Engine**  
In your primary terminal, download dependencies and start the Go engine:
```
go mod download
go run main.go
```

**Step B: Launch the Pygame Visualizer**  
In a second terminal window (with the Python `venv` activated):
```
python main.py
```
